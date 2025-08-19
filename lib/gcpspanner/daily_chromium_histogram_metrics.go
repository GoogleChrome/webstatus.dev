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

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
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

// StoreDailyChromiumHistogramMetrics stores a slice of daily chromium histogram metrics in a bulk insert.
func (c *Client) StoreDailyChromiumHistogramMetrics(
	ctx context.Context,
	histogramName metricdatatypes.HistogramName,
	metrics map[int64]DailyChromiumHistogramMetric) error {
	chromiumHistogramEnumID, err := c.GetIDFromChromiumHistogramKey(ctx, string(histogramName))
	if err != nil {
		slog.ErrorContext(ctx, "unable to find histogram key id from histogram name", "name", string(histogramName))

		return errors.Join(err, ErrUsageMetricUpsertNoHistogramFound)
	}

	producerFn := func(metricChan chan<- spannerDailyChromiumHistogramMetric) {
		for bucketID, metric := range metrics {
			chromiumHistogramEnumValueID, err := c.GetIDFromChromiumHistogramEnumValueKey(
				ctx, *chromiumHistogramEnumID, bucketID)
			if err != nil {
				slog.WarnContext(ctx, "unable to find histogram value id. likely a draft or obsolete feature. will skip",
					"id", *chromiumHistogramEnumID,
					"bucketID", bucketID)

				continue
			}
			metricChan <- spannerDailyChromiumHistogramMetric{
				DailyChromiumHistogramMetric: metric,
				ChromiumHistogramEnumValueID: *chromiumHistogramEnumValueID,
			}
		}
	}

	toMutationFn := func(m spannerDailyChromiumHistogramMetric) (*spanner.Mutation, error) {
		return spanner.InsertOrUpdateStruct(dailyChromiumHistogramMetricsTable, m)
	}

	return runConcurrentBatch(ctx, c, producerFn, dailyChromiumHistogramMetricsTable, toMutationFn)
}

// Implements the syncableEntityMapper interface for WebFeature and SpannerLatestDailyChromiumHistogramMetric.
type latestDailyChromiumHistogramMetricMapper struct{}

// Key for the latestDailyChromiumHistogramMetricMapper.
type latestDailyChromiumHistogramMetricKey struct {
	WebFeatureID                 string
	ChromiumHistogramEnumValueID string
}

// Table returns the name of the Spanner table.
func (m latestDailyChromiumHistogramMetricMapper) Table() string {
	return LatestDailyChromiumHistogramMetricsTable
}

// SelectAll returns a statement to select all LatestDailyChromiumHistogramMetrics.
func (m latestDailyChromiumHistogramMetricMapper) SelectAll() spanner.Statement {
	return spanner.NewStatement(`
		SELECT
			WebFeatureID, ChromiumHistogramEnumValueID, Day
		FROM LatestDailyChromiumHistogramMetrics`)
}

// GetKeyFromExternal returns the business key from an external struct.
func (m latestDailyChromiumHistogramMetricMapper) GetKeyFromExternal(
	in SpannerLatestDailyChromiumHistogramMetric) latestDailyChromiumHistogramMetricKey {
	return latestDailyChromiumHistogramMetricKey{in.WebFeatureID, in.ChromiumHistogramEnumValueID}
}

// GetKeyFromInternal returns the business key from an internal Spanner struct.
func (m latestDailyChromiumHistogramMetricMapper) GetKeyFromInternal(
	in SpannerLatestDailyChromiumHistogramMetric) latestDailyChromiumHistogramMetricKey {
	return latestDailyChromiumHistogramMetricKey{in.WebFeatureID, in.ChromiumHistogramEnumValueID}
}

// MergeAndCheckChanged will merge the entity and return if the entity has changed.
func (m latestDailyChromiumHistogramMetricMapper) MergeAndCheckChanged(
	in SpannerLatestDailyChromiumHistogramMetric,
	existing SpannerLatestDailyChromiumHistogramMetric,
) (SpannerLatestDailyChromiumHistogramMetric, bool) {
	merged := SpannerLatestDailyChromiumHistogramMetric{
		WebFeatureID:                 existing.WebFeatureID,
		ChromiumHistogramEnumValueID: existing.ChromiumHistogramEnumValueID,
		Day:                          in.Day,
	}

	hasChanged := merged.Day != existing.Day

	return merged, hasChanged
}

// GetChildDeleteKeyMutations is a no-op for this table.
func (m latestDailyChromiumHistogramMetricMapper) GetChildDeleteKeyMutations(
	_ context.Context,
	_ *Client,
	_ []SpannerLatestDailyChromiumHistogramMetric,
) ([]ChildDeleteKeyMutations, error) {
	return nil, nil
}

// DeleteMutation creates a Spanner delete mutation.
func (m latestDailyChromiumHistogramMetricMapper) DeleteMutation(
	in SpannerLatestDailyChromiumHistogramMetric) *spanner.Mutation {
	return spanner.Delete(LatestDailyChromiumHistogramMetricsTable,
		spanner.Key{in.WebFeatureID, in.ChromiumHistogramEnumValueID})
}

// SyncLatestDailyChromiumHistogramMetrics reconciles the LatestDailyChromiumHistogramMetrics table.
func (c *Client) SyncLatestDailyChromiumHistogramMetrics(ctx context.Context) error {
	slog.InfoContext(ctx, "Starting latest daily chromium histogram metrics synchronization")
	synchronizer := newEntitySynchronizer[latestDailyChromiumHistogramMetricMapper,
		SpannerLatestDailyChromiumHistogramMetric,
		SpannerLatestDailyChromiumHistogramMetric,
		latestDailyChromiumHistogramMetricKey](c)

	desiredState, err := c.getDesiredLatestDailyChromiumHistogramMetrics(ctx)
	if err != nil {
		return err
	}

	return synchronizer.Sync(ctx, desiredState)
}

func (c *Client) getDesiredLatestDailyChromiumHistogramMetrics(
	ctx context.Context) ([]SpannerLatestDailyChromiumHistogramMetric, error) {
	txn := c.ReadOnlyTransaction()
	defer txn.Close()

	stmt := spanner.NewStatement(`
		WITH LatestMetrics AS (
			SELECT
				ChromiumHistogramEnumValueID,
				MAX(Day) AS MaxDay
			FROM DailyChromiumHistogramMetrics
			GROUP BY ChromiumHistogramEnumValueID
		)
		SELECT
			w.WebFeatureID,
			w.ChromiumHistogramEnumValueID,
			l.MaxDay AS Day
		FROM WebFeatureChromiumHistogramEnumValues w
		JOIN LatestMetrics l ON w.ChromiumHistogramEnumValueID = l.ChromiumHistogramEnumValueID
	`)

	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	var desiredState []SpannerLatestDailyChromiumHistogramMetric
	err := iter.Do(func(r *spanner.Row) error {
		var metric SpannerLatestDailyChromiumHistogramMetric
		if err := r.ToStruct(&metric); err != nil {
			return err
		}
		desiredState = append(desiredState, metric)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return desiredState, nil
}
