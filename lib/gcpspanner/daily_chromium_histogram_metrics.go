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
			return &civil.Date{}, nil
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
		slog.ErrorContext(ctx, "error querying for latest run time", "error", err)

		return nil, err
	}
	var featureID string
	if err := row.Columns(&featureID); err != nil {
		slog.ErrorContext(ctx, "error extracting date", "error", err)

		return nil, err
	}

	return &featureID, nil
}

// updateDailyChromiumHistogramMetric handles the insertion or update logic for the DailyChromiumHistogramMetrics table.
// If a metric does not exist, it will insert a new metrics.
// If a metric exists, it will update the Rate column.
func updateDailyChromiumHistogramMetric(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	metric spannerDailyChromiumHistogramMetric) (*spanner.Mutation, error) {
	stmt := spanner.NewStatement(`
				SELECT
					ChromiumHistogramEnumValueID
					Day,
					Rate
				FROM DailyChromiumHistogramMetrics
				WHERE ChromiumHistogramEnumValueID = @chromiumHistogramEnumValueID
				AND Day = @day`)
	parameters := map[string]interface{}{
		"chromiumHistogramEnumValueID": metric.ChromiumHistogramEnumValueID,
		"day":                          metric.Day,
	}
	stmt.Params = parameters

	// Attempt to query for the row.
	it := txn.Query(ctx, stmt)
	defer it.Stop()
	var m *spanner.Mutation
	row, err := it.Next()

	if err != nil && errors.Is(err, iterator.Done) {
		// No rows returned. Act as if this is an insertion.
		var err error
		m, err = spanner.InsertOrUpdateStruct(dailyChromiumHistogramMetricsTable, metric)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		// Read the existing metric and merge the values.
		var existingMetric spannerDailyChromiumHistogramMetric
		err = row.ToStruct(&existingMetric)
		if err != nil {
			return nil, err
		}
		existingMetric.Rate = metric.Rate
		m, err = spanner.InsertOrUpdateStruct(WPTRunFeatureMetricTable, existingMetric)
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
	}

	return m, nil
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

	// err = newEntityWriter[dailyChromiumHistogramMetricSpannerMapper](c).upsert(ctx, spannerDailyChromiumHistogramMetric{
	// 	DailyChromiumHistogramMetric: metric,
	// 	ChromiumHistogramEnumValueID: *chromiumHistogramEnumValueID,
	// })
	// if err != nil {
	// 	return err
	// }

	_, err = c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{}
		existingDate, err := getLatestDailyChromiumMetricDate(ctx, txn, *chromiumHistogramEnumValueID)
		if err != nil {
			if !errors.Is(err, iterator.Done) { // Handle errors other than "not found"
				return errors.Join(ErrInternalQueryFailure, err)
			}
		}
		m0, err := updateDailyChromiumHistogramMetric(ctx, txn, spannerDailyChromiumHistogramMetric{
			DailyChromiumHistogramMetric: metric,
			ChromiumHistogramEnumValueID: *chromiumHistogramEnumValueID,
		})
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}
		if m0 != nil {
			mutations = append(mutations, m0)
		}

		if shouldUpsertLatestDailyChromiumUsageMetric(existingDate, metric.Day) {
			featureID, err := getWebFeatureIDByChromiumHistogramEnumValueID(ctx, txn, *chromiumHistogramEnumValueID)
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
			m1, err := spanner.InsertOrUpdateStruct(
				LatestDailyChromiumHistogramMetricsTable,
				SpannerLatestDailyChromiumHistogramMetric{
					WebFeatureID:                 *featureID,
					ChromiumHistogramEnumValueID: *chromiumHistogramEnumValueID,
					Day:                          metric.Day,
				})
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
			mutations = append(mutations, m1)

			err = txn.BufferWrite(mutations)
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}

			return nil
		}

		return nil
	})

	return err
}
