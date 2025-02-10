// Copyright 2025 Google LLC
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
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// BaselineDateType is an enum representing the type of date to use for baseline counts.
type BaselineDateType string

const (
	// BaselineDateTypeLow uses the LowDate from FeatureBaselineStatus.
	BaselineDateTypeLow BaselineDateType = "low"
)

// BaselineStatusCountMetric represents a single data point in the baseline status count time series.
type BaselineStatusCountMetric struct {
	Date        time.Time `spanner:"Date"`
	StatusCount int64     `spanner:"StatusCount"`
}

// BaselineStatusCountResultPage is a page of results for the baseline status count query.
type BaselineStatusCountResultPage struct {
	NextPageToken *string
	Metrics       []BaselineStatusCountMetric
}

// baselineStatusCountCursor is used for pagination.
type baselineStatusCountCursor struct {
	LastDate        time.Time `json:"last_date"`
	LastStatusCount int64     `json:"last_status_count"`
}

// decodeBaselineStatusCountCursor decodes a cursor string into a baselineStatusCountCursor.
func decodeBaselineStatusCountCursor(cursor string) (*baselineStatusCountCursor, error) {
	return decodeCursor[baselineStatusCountCursor](cursor)
}

// encodeBaselineStatusCountCursor encodes a baselineStatusCountCursor into a cursor string.
func encodeBaselineStatusCountCursor(lastDate time.Time, lastStatusCount int64) string {
	return encodeCursor(baselineStatusCountCursor{
		LastDate:        lastDate,
		LastStatusCount: lastStatusCount,
	})
}

type fbsColumn string

const fbsColumnLowDate fbsColumn = "fbs.LowDate"

// ListBaselineStatusCounts retrieves a cumulative count of baseline features over time.
func (c *Client) ListBaselineStatusCounts(
	ctx context.Context,
	dateType BaselineDateType,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) (*BaselineStatusCountResultPage, error) {
	var parsedToken *baselineStatusCountCursor
	var err error
	if pageToken != nil {
		parsedToken, err = decodeBaselineStatusCountCursor(*pageToken)
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
	}

	txn := c.ReadOnlyTransaction()
	defer txn.Close()

	// 1. Validate dateType
	switch dateType {
	case BaselineDateTypeLow:
		break
	default:
		return nil, errors.Join(ErrInternalQueryFailure, fmt.Errorf("invalid BaselineDateType: %s", dateType))
	}

	// 2. Get excluded feature IDs
	excludedFeatureIDs, err := c.getFeatureIDsForEachExcludedFeatureKey(ctx, txn)
	if err != nil {
		return nil, err
	}

	// 3. Calculate initial cumulative count
	cumulativeCount, err := c.getInitialBaselineStatusCount(
		ctx, txn, parsedToken, startAt, excludedFeatureIDs, dateType)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	// 4. Process results and update cumulative count
	stmt := createListBaselineStatusCountsStatement(dateType, startAt, endAt, pageSize, parsedToken, excludedFeatureIDs)

	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	var metrics []BaselineStatusCountMetric
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}

		var metric BaselineStatusCountMetric
		if err := row.ToStruct(&metric); err != nil {
			return nil, err
		}

		cumulativeCount += metric.StatusCount
		metric.StatusCount = cumulativeCount
		metrics = append(metrics, metric)
	}

	var newCursor *string
	if len(metrics) == pageSize {
		lastMetric := metrics[len(metrics)-1]
		generatedCursor := encodeBaselineStatusCountCursor(lastMetric.Date, lastMetric.StatusCount)
		newCursor = &generatedCursor
	}

	return &BaselineStatusCountResultPage{
		NextPageToken: newCursor,
		Metrics:       metrics,
	}, nil
}

// getInitialBaselineStatusCount calculates the initial cumulative count for the first page.
func (c *Client) getInitialBaselineStatusCount(
	ctx context.Context,
	txn *spanner.ReadOnlyTransaction,
	parsedToken *baselineStatusCountCursor,
	startAt time.Time,
	excludedFeatureIDs []string,
	dateType BaselineDateType,
) (int64, error) {
	if parsedToken != nil {
		return parsedToken.LastStatusCount, nil
	}

	params := map[string]interface{}{
		"startAt": startAt,
	}

	var excludedFeatureFilter string
	if len(excludedFeatureIDs) > 0 {
		excludedFeatureFilter = `
				AND fbs.WebFeatureID NOT IN UNNEST(@excludedFeatureIDs)`
		params["excludedFeatureIDs"] = excludedFeatureIDs
	}

	// Construct the query based on dateType
	var dateField string
	switch dateType {
	case BaselineDateTypeLow:
		dateField = string(fbsColumnLowDate)
	}

	var initialCount int64
	stmt := spanner.Statement{
		SQL: fmt.Sprintf(`
				SELECT COALESCE(SUM(daily_status_count), 0)
				FROM (
						SELECT COUNT(fbs.WebFeatureID) AS daily_status_count
						FROM FeatureBaselineStatus fbs
						WHERE %s < @startAt %s
						GROUP BY %s
				)`, dateField, excludedFeatureFilter, dateField),
		Params: params,
	}

	err := txn.Query(ctx, stmt).Do(func(r *spanner.Row) error {
		return r.Column(0, &initialCount)
	})

	return initialCount, err
}

// createListBaselineStatusCountsStatement creates the Spanner statement for the main query.
func createListBaselineStatusCountsStatement(
	dateType BaselineDateType,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *baselineStatusCountCursor,
	excludedFeatureIDs []string,
) spanner.Statement {
	params := map[string]interface{}{
		"startAt":  startAt,
		"endAt":    endAt,
		"pageSize": pageSize,
	}

	var pageFilter string
	if pageToken != nil {
		var dateField string
		switch dateType {
		case BaselineDateTypeLow:
			dateField = string(fbsColumnLowDate)
		}
		pageFilter = fmt.Sprintf(`AND %s > @lastDate`, dateField)
		params["lastDate"] = pageToken.LastDate
	}

	var excludedFeatureFilter string
	if len(excludedFeatureIDs) > 0 {
		excludedFeatureFilter = `AND fbs.WebFeatureID NOT IN UNNEST(@excludedFeatureIDs)`
		params["excludedFeatureIDs"] = excludedFeatureIDs
	}

	// Construct the query based on dateType
	var dateField string
	switch dateType {
	case BaselineDateTypeLow:
		dateField = string(fbsColumnLowDate)
	}

	stmt := spanner.Statement{
		SQL: fmt.Sprintf(`
				SELECT %s AS Date, COUNT(fbs.WebFeatureID) AS StatusCount
				FROM FeatureBaselineStatus fbs
				WHERE %s >= @startAt AND %s < @endAt %s %s
				GROUP BY %s
				ORDER BY %s
				LIMIT @pageSize`,
			dateField, dateField, dateField, pageFilter, excludedFeatureFilter, dateField, dateField),
		Params: params,
	}

	return stmt
}
