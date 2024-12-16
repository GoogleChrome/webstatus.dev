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

			return &zeroDate, errors.Join(ErrQueryReturnedNoResults, err)
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
		if errors.Is(err, iterator.Done) {
			return nil, errors.Join(ErrQueryReturnedNoResults, err)
		}
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

// UpsertDailyChromiumHistogramMetric upserts a daily chromium histogram metric.
//
// Errors:
//   - ErrQueryReturnedNoResults: If the histogram key or value ID is not found.
//   - ErrInternalQueryFailure: If any internal query fails during the process.
//   - ErrUsageMetricUpsertNoFeatureIDFound: If no feature ID is found while
//     attempting to upsert the latest daily chromium usage metric.
//   - ErrUsageMetricUpsertNoHistogramFound: If the histogram is not found
//   - ErrUsageMetricUpsertNoHistogramEnumFound: If a particular enum in the histogram is not found.
func (c *Client) UpsertDailyChromiumHistogramMetric(
	ctx context.Context,
	histogramName metricdatatypes.HistogramName,
	bucketID int64,
	metric DailyChromiumHistogramMetric) error {
	// TODO: When we have a generic way to do batch upserts, change this to accept an array of metrics.
	chromiumHistogramEnumID, err := c.GetIDFromChromiumHistogramKey(ctx, string(histogramName))
	if err != nil {
		slog.ErrorContext(ctx, "unable to find histogram key id from histogram name", "name", string(histogramName))

		return errors.Join(err, ErrUsageMetricUpsertNoHistogramFound)
	}
	chromiumHistogramEnumValueID, err := c.GetIDFromChromiumHistogramEnumValueKey(
		ctx, *chromiumHistogramEnumID, bucketID)
	if err != nil {
		if errors.Is(err, ErrQueryReturnedNoResults) {
			slog.WarnContext(ctx, "unable to find histogram value id. likely a draft or obsolete feature. will skip",
				"id", *chromiumHistogramEnumID,
				"bucketID", bucketID)

			return errors.Join(err, ErrUsageMetricUpsertNoHistogramEnumFound)
		}

		slog.ErrorContext(ctx, "unable to find histogram value id",
			"id", *chromiumHistogramEnumID,
			"bucketID", bucketID)

		return err
	}

	_, err = c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var mutations []*spanner.Mutation
		m0, err := spanner.InsertOrUpdateStruct(
			dailyChromiumHistogramMetricsTable,
			spannerDailyChromiumHistogramMetric{
				DailyChromiumHistogramMetric: metric,
				ChromiumHistogramEnumValueID: *chromiumHistogramEnumValueID,
			})
		if err != nil {
			return err
		}
		mutations = append(mutations, m0)

		existingDate, err := getLatestDailyChromiumMetricDate(ctx, txn, *chromiumHistogramEnumValueID)
		if err != nil {
			if !errors.Is(err, ErrQueryReturnedNoResults) { // Handle errors other than "not found"
				return errors.Join(ErrInternalQueryFailure, err)
			}
		}

		if shouldUpsertLatestDailyChromiumUsageMetric(existingDate, metric.Day) {
			featureID, err := getWebFeatureIDByChromiumHistogramEnumValueID(ctx, txn, *chromiumHistogramEnumValueID)
			if err != nil {
				if errors.Is(err, ErrQueryReturnedNoResults) {
					return errors.Join(err, ErrUsageMetricUpsertNoFeatureIDFound)
				}

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
		}
		err = txn.BufferWrite(mutations)
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}

		return nil
	})

	return err
}
