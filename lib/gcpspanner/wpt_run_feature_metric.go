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
	"cmp"
	"context"
	"errors"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const wptRunFeatureMetricTable = "WPTRunFeatureMetrics"

// SpannerWPTRunFeatureMetric is a wrapper for the metric data that is actually
// stored in spanner. This is useful because the spanner id is not useful to
// return to the end user since it is used to decouple the primary keys between
// this system and wpt.fyi.
type SpannerWPTRunFeatureMetric struct {
	ID string `spanner:"ID"`
	WPTRunFeatureMetric
}

// WPTRunFeatureMetric represents the metrics for a particular feature in a run.
type WPTRunFeatureMetric struct {
	RunID      int64  `spanner:"ExternalRunID"`
	FeatureID  string `spanner:"FeatureID"`
	TotalTests *int64 `spanner:"TotalTests"`
	TestPass   *int64 `spanner:"TestPass"`
}

// UpsertWPTRunFeatureMetric will upsert the given WPT Run metric.
// The RunID must exists in a row in the WPTRuns table.
// If the metric does not exist, it will insert a new metric.
// If the metric exists, it will only update the TotalTests and TestPass columns.
func (c *Client) UpsertWPTRunFeatureMetric(ctx context.Context, in WPTRunFeatureMetric) error {
	id, err := c.GetIDOfWPTRunByRunID(ctx, in.RunID)
	if err != nil {
		return err
	}

	// Create a metric with the retrieved ID
	metric := SpannerWPTRunFeatureMetric{
		ID:                  *id,
		WPTRunFeatureMetric: in,
	}
	_, err = c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.NewStatement(`
			SELECT
				ID, ExternalRunID, FeatureID, TotalTests, TestPass
			FROM WPTRunFeatureMetrics
			WHERE ExternalRunID = @externalRunID
			LIMIT 1`)
		parameters := map[string]interface{}{
			"externalRunID": metric.RunID,
		}
		stmt.Params = parameters

		// Attempt to query for the row.
		it := txn.Query(ctx, stmt)
		defer it.Stop()
		var m *spanner.Mutation
		row, err := it.Next()

		// nolint: nestif // TODO: fix in the future.
		if err != nil {
			if errors.Is(err, iterator.Done) {
				// No rows returned. Act as if this is an insertion.
				var err error
				m, err = spanner.InsertOrUpdateStruct(wptRunFeatureMetricTable, metric)
				if err != nil {
					return errors.Join(ErrInternalQueryFailure, err)
				}
			} else {
				// An unexpected error occurred.

				return errors.Join(ErrInternalQueryFailure, err)
			}
		} else {
			// Read the existing metric and merge the values.
			var existingMetric SpannerWPTRunFeatureMetric
			err = row.ToStruct(&existingMetric)
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
			// Only allow overriding of the test numbers.
			existingMetric.TestPass = cmp.Or[*int64](metric.TestPass, existingMetric.TestPass, nil)
			existingMetric.TotalTests = cmp.Or[*int64](metric.TotalTests, existingMetric.TotalTests, nil)
			m, err = spanner.InsertOrUpdateStruct(wptRunFeatureMetricTable, metric)
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
		}

		// Buffer the mutation to be committed.
		err = txn.BufferWrite([]*spanner.Mutation{m})
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}

		return nil
	})
	if err != nil {
		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}

// GetMetricByRunIDAndFeatureID attempts to get a metric for the given id from
// wpt.fyi and web feature id.
func (c *Client) GetMetricByRunIDAndFeatureID(
	ctx context.Context,
	runID int64,
	featureID string,
) (*WPTRunFeatureMetric, error) {
	txn := c.ReadOnlyTransaction()
	defer txn.Close()
	stmt := spanner.NewStatement(`
		SELECT
			ExternalRunID, FeatureID, TotalTests, TestPass
		FROM WPTRunFeatureMetrics
		WHERE ExternalRunID = @externalRunID AND FeatureID = @featureID
		LIMIT 1`)
	parameters := map[string]interface{}{
		"externalRunID": runID,
		"featureID":     featureID,
	}
	stmt.Params = parameters
	it := txn.Query(ctx, stmt)
	defer it.Stop()

	row, err := it.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, errors.Join(ErrQueryReturnedNoResults, err)
		}

		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	var metric WPTRunFeatureMetric
	if err := row.ToStruct(&metric); err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	return &metric, nil
}

// WPTRunFeatureMetricWithTime contains metrics for a feature at a given time.
type WPTRunFeatureMetricWithTime struct {
	TimeStart  time.Time `spanner:"TimeStart"`
	RunID      int64     `spanner:"ExternalRunID"`
	TotalTests *int64    `spanner:"TotalTests"`
	TestPass   *int64    `spanner:"TestPass"`
}

// ListMetricsForFeatureIDBrowserAndChannel attempts to return a page of
// metrics based on a web feature id, browser name and channel. A time window
// must be specified to analyze the runs according to the TimeStart of the run.
// If the page size matches the pageSize, a page token is returned. Else,
// no page token is returned.
func (c *Client) ListMetricsForFeatureIDBrowserAndChannel(
	ctx context.Context,
	featureID string,
	browser string,
	channel string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]WPTRunFeatureMetricWithTime, *string, error) {
	var stmt spanner.Statement
	params := map[string]interface{}{
		"featureID":   featureID,
		"browserName": browser,
		"channel":     channel,
		"startAt":     startAt,
		"endAt":       endAt,
		"pageSize":    pageSize,
	}

	if pageToken == nil {
		stmt = spanner.NewStatement(
			`SELECT wpfm.ExternalRunID, r.TimeStart, wpfm.TotalTests, wpfm.TestPass
				FROM WPTRuns r
				JOIN WPTRunFeatureMetrics wpfm ON r.ExternalRunID = wpfm.ExternalRunID
				WHERE wpfm.FeatureID = @featureID
					AND r.BrowserName = @browserName
					AND r.Channel = @channel
		  			AND r.TimeStart >= @startAt AND r.TimeStart < @endAt
				ORDER BY r.TimeStart DESC, r.ExternalRunID DESC LIMIT @pageSize`)
	} else {
		cursor, err := decodeWPTRunCursor(*pageToken)
		if err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		stmt = spanner.NewStatement(
			`SELECT wpfm.ExternalRunID, r.TimeStart, wpfm.TotalTests, wpfm.TestPass
                FROM WPTRuns r
                JOIN WPTRunFeatureMetrics wpfm ON r.ExternalRunID = wpfm.ExternalRunID
                WHERE wpfm.FeatureID = @featureID
					AND r.BrowserName = @browserName
					AND r.Channel = @channel
                   	AND r.TimeStart >= @startAt AND r.TimeStart < @endAt
                   	AND (r.TimeStart < @lastTimestamp OR
                    	r.TimeStart = @lastTimestamp AND r.ExternalRunID < @lastRunID)
                ORDER BY r.TimeStart DESC, r.ExternalRunID DESC LIMIT @pageSize`)
		params["lastTimestamp"] = cursor.LastTimeStart
		params["lastRunID"] = cursor.LastRunID
	}
	stmt.Params = params

	txn := c.Single()
	defer txn.Close()
	it := txn.Query(ctx, stmt)
	defer it.Stop()

	var featureMetrics []WPTRunFeatureMetricWithTime
	for {
		row, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var featureMetric WPTRunFeatureMetricWithTime
		if err := row.ToStruct(&featureMetric); err != nil {
			return nil, nil, err
		}
		featureMetrics = append(featureMetrics, featureMetric)
	}

	if len(featureMetrics) == pageSize {
		lastFeatureMetric := featureMetrics[len(featureMetrics)-1]
		newCursor := encodeWPTRunCursor(lastFeatureMetric.TimeStart, lastFeatureMetric.RunID)

		return featureMetrics, &newCursor, nil
	}

	return featureMetrics, nil, nil
}

// WPTRunAggregationMetricWithTime contains metrics for a particular aggregation
// at a given time. For now, it is the same metrics as
// WPTRunFeatureMetricWithTime.
type WPTRunAggregationMetricWithTime struct {
	WPTRunFeatureMetricWithTime
}

// ListMetricsOverTimeWithAggregatedTotals attempts to return a page of
// metrics based on browser name and channel. Users can provide a list of web
// feature ids. If the list is provided, the aggregation will be scoped to those
// feature ids. If an empty or nil list is provided, the aggregation is applied
// to all features.
// A time window must be specified to analyze the runs according to the
// TimeStart of the run.
// If the page size matches the pageSize, a page token is returned. Else,
// no page token is returned.
func (c *Client) ListMetricsOverTimeWithAggregatedTotals(
	ctx context.Context,
	featureIDs []string,
	browser string,
	channel string,
	startAt, endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]WPTRunAggregationMetricWithTime, *string, error) {
	params := map[string]interface{}{
		"browserName": browser,
		"channel":     channel,
		"startAt":     startAt,
		"endAt":       endAt,
		"pageSize":    pageSize,
	}

	var stmt spanner.Statement
	// nolint: nestif // TODO: fix in the future.
	if pageToken == nil {
		if len(featureIDs) == 0 {
			stmt = noPageTokenAllFeatures(params)
		} else {
			stmt = noPageTokenFeatureSubset(params, featureIDs)
		}
	} else {
		cursor, err := decodeWPTRunCursor(*pageToken)
		if err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		if len(featureIDs) == 0 {
			stmt = withPageTokenAllFeatures(params, *cursor)
		} else {
			stmt = withPageTokenFeatureSubset(params, featureIDs, *cursor)
		}
	}

	txn := c.Single()
	defer txn.Close()
	it := txn.Query(ctx, stmt)
	defer it.Stop()

	var aggregationMetrics []WPTRunAggregationMetricWithTime
	for {
		row, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var aggregationMetric WPTRunAggregationMetricWithTime
		if err := row.ToStruct(&aggregationMetric); err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		aggregationMetrics = append(aggregationMetrics, aggregationMetric)
	}

	if len(aggregationMetrics) == pageSize {
		lastFeatureMetric := aggregationMetrics[len(aggregationMetrics)-1]
		newCursor := encodeWPTRunCursor(lastFeatureMetric.TimeStart, lastFeatureMetric.RunID)

		return aggregationMetrics, &newCursor, nil
	}

	return aggregationMetrics, nil, nil
}

// noPageTokenAllFeatures builds a spanner statement when a page token
// is not provided and the aggregation applies to all features.
func noPageTokenAllFeatures(params map[string]interface{}) spanner.Statement {
	stmt := spanner.NewStatement(`
		SELECT
			r.ExternalRunID,
			r.TimeStart,
			SUM(wpfm.TotalTests) AS TotalTests,
			SUM(wpfm.TestPass) AS TestPass
		FROM WPTRuns r
		JOIN WPTRunFeatureMetrics wpfm ON r.ExternalRunID = wpfm.ExternalRunID
		WHERE r.BrowserName = @browserName
		AND r.Channel = @channel
		AND r.TimeStart >= @startAt AND r.TimeStart < @endAt
		GROUP BY r.ExternalRunID, r.TimeStart
		ORDER BY r.TimeStart DESC, r.ExternalRunID DESC LIMIT @pageSize`)
	stmt.Params = params

	return stmt
}

// noPageTokenFeatureSubset builds a spanner statement when a page token is
// not provided and the aggregation applies to a particular list of features.
func noPageTokenFeatureSubset(params map[string]interface{}, featureIDs []string) spanner.Statement {
	stmt := spanner.NewStatement(`
	SELECT
		r.ExternalRunID,
		r.TimeStart,
		SUM(wpfm.TotalTests) AS TotalTests,
		SUM(wpfm.TestPass) AS TestPass
	FROM WPTRuns r
	JOIN WPTRunFeatureMetrics wpfm ON r.ExternalRunID = wpfm.ExternalRunID
	WHERE wpfm.FeatureID IN UNNEST(@featureIDs)
	AND r.BrowserName = @browserName
	AND r.Channel = @channel
	AND r.TimeStart >= @startAt AND r.TimeStart < @endAt
	GROUP BY r.ExternalRunID, r.TimeStart
	ORDER BY r.TimeStart DESC, r.ExternalRunID DESC LIMIT @pageSize`)
	params["featureIDs"] = featureIDs
	stmt.Params = params

	return stmt
}

// withPageTokenAllFeatures builds a spanner statement when a page token is
// provided and the aggregation applies to all features.
func withPageTokenAllFeatures(params map[string]interface{}, cursor WPTRunCursor) spanner.Statement {
	stmt := spanner.NewStatement(`
		SELECT
			r.ExternalRunID,
			r.TimeStart,
			SUM(wpfm.TotalTests) AS TotalTests,
			SUM(wpfm.TestPass) AS TestPass
		FROM WPTRuns r
		JOIN WPTRunFeatureMetrics wpfm ON r.ExternalRunID = wpfm.ExternalRunID
		WHERE r.BrowserName = @browserName
		AND r.Channel = @channel
		AND r.TimeStart >= @startAt AND r.TimeStart < @endAt
		AND (r.TimeStart < @lastTimestamp OR
			 r.TimeStart = @lastTimestamp AND r.ExternalRunID < @lastRunID)
		GROUP BY r.ExternalRunID, r.TimeStart
		ORDER BY r.TimeStart DESC, r.ExternalRunID DESC LIMIT @pageSize`)
	params["lastTimestamp"] = cursor.LastTimeStart
	params["lastRunID"] = cursor.LastRunID
	stmt.Params = params

	return stmt
}

// withPageTokenFeatureSubset builds a spanner statement when a page token is
// provided and the aggregation applies to a particular list of features.
func withPageTokenFeatureSubset(
	params map[string]interface{},
	featureIDs []string,
	cursor WPTRunCursor) spanner.Statement {
	stmt := spanner.NewStatement(`
		SELECT
			r.ExternalRunID,
			r.TimeStart,
			SUM(wpfm.TotalTests) AS TotalTests,
			SUM(wpfm.TestPass) AS TestPass
		FROM WPTRuns r
		JOIN WPTRunFeatureMetrics wpfm ON r.ExternalRunID = wpfm.ExternalRunID
		WHERE wpfm.FeatureID IN UNNEST(@featureIDs)
		AND r.BrowserName = @browserName
		AND r.Channel = @channel
		AND r.TimeStart >= @startAt AND r.TimeStart < @endAt
		AND (r.TimeStart < @lastTimestamp OR
			 r.TimeStart = @lastTimestamp AND r.ExternalRunID < @lastRunID)
		GROUP BY r.ExternalRunID, r.TimeStart
		ORDER BY r.TimeStart DESC, r.ExternalRunID DESC LIMIT @pageSize`)
	params["featureIDs"] = featureIDs
	params["lastTimestamp"] = cursor.LastTimeStart
	params["lastRunID"] = cursor.LastRunID
	stmt.Params = params

	return stmt
}
