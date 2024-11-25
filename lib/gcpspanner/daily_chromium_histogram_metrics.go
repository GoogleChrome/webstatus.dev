// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcpspanner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
	"google.golang.org/api/iterator"
)

const dailyChromiumHistogramMetricsTable = "DailyChromiumHistogramMetrics"
const LatestDailyChromiumHistogramMetricsTable = "LatestDailyChromiumHistogramMetrics"

// Implements the entityMapper interface for DailyChromiumHistogramMetric and SpannerDailyChromiumHistogramMetric.
type dailyChromiumHistogramMetricSpannerMapper struct{}

func (m dailyChromiumHistogramMetricSpannerMapper) Table() string {
	return dailyChromiumHistogramMetricsTable
}

type dailyChromiumHistogramMetricKey struct {
	ChromiumHistogramEnumValueID string
	Day                          civil.Date
}

func (m dailyChromiumHistogramMetricSpannerMapper) GetKey(
	in spannerDailyChromiumHistogramMetric) dailyChromiumHistogramMetricKey {
	return dailyChromiumHistogramMetricKey{
		ChromiumHistogramEnumValueID: in.ChromiumHistogramEnumValueID,
		Day:                          in.Day,
	}
}

func (m dailyChromiumHistogramMetricSpannerMapper) Merge(
	in spannerDailyChromiumHistogramMetric,
	existing spannerDailyChromiumHistogramMetric) spannerDailyChromiumHistogramMetric {
	return spannerDailyChromiumHistogramMetric{
		ChromiumHistogramEnumValueID: existing.ChromiumHistogramEnumValueID,
		DailyChromiumHistogramMetric: DailyChromiumHistogramMetric{
			Day:  existing.Day,
			Rate: in.Rate,
		},
	}
}

func (m dailyChromiumHistogramMetricSpannerMapper) SelectOne(key dailyChromiumHistogramMetricKey) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ChromiumHistogramEnumValueID, Day, Rate
	FROM %s
	WHERE ChromiumHistogramEnumValueID = @chromiumHistogramEnumValueID AND Day = @day
	LIMIT 1`,
		m.Table()))
	parameters := map[string]interface{}{
		"chromiumHistogramEnumValueID": key.ChromiumHistogramEnumValueID,
		"day":                          key.Day,
	}
	stmt.Params = parameters

	return stmt
}

type DailyChromiumHistogramMetric struct {
	Day  civil.Date `spanner:"Day"`
	Rate big.Rat    `spanner:"Rate"`
}

type SpannerLatestDailyChromiumHistogramMetric struct {
	WebFeatureID                 string     `spanner:"WebFeatureID"`
	ChromiumHistogramEnumValueID string     `spanner:"ChromiumHistogramEnumValueID"`
	Day                          civil.Date `spanner:"Day"`
}

type spannerDailyChromiumHistogramMetric struct {
	DailyChromiumHistogramMetric
	ChromiumHistogramEnumValueID string `spanner:"ChromiumHistogramEnumValueID"`
}

// getLatestDailyChromiumMetricDate retrieves the Date of the latest usage stats for the given feature.
func getLatestDailyChromiumMetricDate(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	chromiumHistogramEnumValueID string) (*civil.Date, error) {
	stmt := spanner.NewStatement(`
        SELECT
			dchm.Day as Date
        FROM LatestDailyChromiumHistogramMetrics l
        JOIN DailyChromiumHistogramMetrics dchm
		ON l.ChromiumHistogramEnumValueID = dchm.ChromiumHistogramEnumValueID
        WHERE l.ChromiumHistogramEnumValueID = @chromiumHistogramEnumValueID`)

	stmt.Params = map[string]interface{}{
		"chromiumHistogramEnumValueID": chromiumHistogramEnumValueID,
	}

	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			// No row found, return zero time
			zeroTime := time.Time{}
			zeroDate := civil.DateOf(zeroTime)

			return &zeroDate, nil
		}
		slog.ErrorContext(ctx, "error querying for latest run time", "error", err)

		return nil, err
	}

	var date civil.Date
	if err := row.Columns(&date); err != nil {
		slog.ErrorContext(ctx, "error extracting date", "error", err)

		return nil, err
	}

	return &date, nil
}

func getWebFeatureIDByChromiumHistogramEnumValueID(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	chromiumHistogramEnumValueID string) (*string, error) {
	stmt := spanner.NewStatement(`
		SELECT
			WebFeatureID
		FROM WebFeatureChromiumHistogramEnumValues
		WHERE chromiumHistogramEnumValueID = @chromiumHistogramEnumValueID`)

	stmt.Params = map[string]interface{}{
		"chromiumHistogramEnumValueID": chromiumHistogramEnumValueID,
	}

	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		slog.ErrorContext(ctx, "error querying for web feature ID", "error", err)

		return nil, err
	}
	var featureID string
	if err := row.Columns(&featureID); err != nil {
		slog.ErrorContext(ctx, "error extracting date", "error", err)

		return nil, err
	}

	return &featureID, nil
}

// shouldUpsertLatestDailyChromiumUsageMetric determines whether the latest metric should be upserted based on
// date comparison.
func shouldUpsertLatestDailyChromiumUsageMetric(existingDate *civil.Date, newDate civil.Date) bool {
	return existingDate == nil || existingDate.IsZero() || newDate.After(*existingDate)
}

func (c *Client) UpsertDailyChromiumHistogramMetric(
	ctx context.Context,
	histogramName metricdatatypes.HistogramName,
	bucketID int64,
	metric DailyChromiumHistogramMetric) error {
	// TODO: When we have a generic way to do batch upserts, change this to accept an array of metrics.
	chromiumHistogramEnumID, err := c.GetIDFromChromiumHistogramKey(ctx, string(histogramName))
	if err != nil {
		slog.ErrorContext(ctx, "unable to find histogram key id from histogram name", "name", string(histogramName))

		return err
	}
	chromiumHistogramEnumValueID, err := c.GetIDFromChromiumHistogramEnumValueKey(
		ctx, *chromiumHistogramEnumID, bucketID)
	if err != nil {
		if errors.Is(err, ErrQueryReturnedNoResults) {
			slog.WarnContext(ctx, "unable to find histogram value id. likely a draft feature. will skip",
				"id", *chromiumHistogramEnumID,
				"bucketID", bucketID)

			// TODO. Create a specific error for ErrQueryReturnedNoResults from GetIDFromChromiumHistogramEnumValueKey
			// and return that. Then have the adapter check for it. For now, we can treat this as a warning and ignore
			// the error.
			return nil
		}

		slog.ErrorContext(ctx, "unable to find histogram value id",
			"id", *chromiumHistogramEnumID,
			"bucketID", bucketID)

		return err
	}

	_, err = c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		err = newEntityWriter[dailyChromiumHistogramMetricSpannerMapper](c).upsert(ctx, spannerDailyChromiumHistogramMetric{
			DailyChromiumHistogramMetric: metric,
			ChromiumHistogramEnumValueID: *chromiumHistogramEnumValueID,
		})
		if err != nil {
			return err
		}

		existingDate, err := getLatestDailyChromiumMetricDate(ctx, txn, *chromiumHistogramEnumValueID)
		if err != nil {
			if !errors.Is(err, iterator.Done) { // Handle errors other than "not found"
				return errors.Join(ErrInternalQueryFailure, err)
			}
		}

		if shouldUpsertLatestDailyChromiumUsageMetric(existingDate, metric.Day) {
			featureID, err := getWebFeatureIDByChromiumHistogramEnumValueID(ctx, txn, *chromiumHistogramEnumValueID)
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
			m, err := spanner.InsertOrUpdateStruct(
				LatestDailyChromiumHistogramMetricsTable,
				SpannerLatestDailyChromiumHistogramMetric{
					WebFeatureID:                 *featureID,
					ChromiumHistogramEnumValueID: *chromiumHistogramEnumValueID,
					Day:                          metric.Day,
				})
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
			err = txn.BufferWrite([]*spanner.Mutation{m})
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}

			return nil
		}

		return nil
	})

	return err
}
