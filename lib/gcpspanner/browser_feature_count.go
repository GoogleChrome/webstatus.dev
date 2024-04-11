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
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// BrowserFeatureCountMetric contains a row of data returned by the feature count query.
type BrowserFeatureCountMetric struct {
	ReleaseDate  time.Time `spanner:"ReleaseDate"`
	FeatureCount int64     `spanner:"FeatureCount"`
}

type BrowserFeatureCountResultPage struct {
	NextPageToken *string
	Metrics       []BrowserFeatureCountMetric
}

func (c *Client) ListBrowserFeatureCountMetric(
	ctx context.Context,
	browser string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) (*BrowserFeatureCountResultPage, error) {
	var parsedToken *BrowserFeatureCountCursor
	var err error
	if pageToken != nil {
		parsedToken, err = decodeBrowserFeatureCountCursor(*pageToken)
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
	}

	txn := c.ReadOnlyTransaction()
	defer txn.Close()
	// 1. Calculate initial cumulative count
	cumulativeCount, err := c.getInitialBrowserFeatureCount(ctx, txn, parsedToken, browser, startAt)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	// 2. Process results and update cumulative count
	stmt := createListBrowserFeatureCountMetricStatement(
		browser,
		startAt,
		endAt,
		pageSize,
		parsedToken,
	)
	it := txn.Query(ctx, stmt)
	defer it.Stop()

	var metrics []BrowserFeatureCountMetric
	for {
		row, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var metric BrowserFeatureCountMetric
		if err := row.ToStruct(&metric); err != nil {
			return nil, err
		}
		// Update cumulative count
		cumulativeCount += metric.FeatureCount
		metric.FeatureCount = cumulativeCount

		metrics = append(metrics, metric)
	}

	var newCursor *string
	if len(metrics) == pageSize {
		lastMetric := metrics[len(metrics)-1]
		generatedCursor := encodeBrowserFeatureCountCursor(lastMetric.ReleaseDate, lastMetric.FeatureCount)
		newCursor = &generatedCursor
	}

	return &BrowserFeatureCountResultPage{
		NextPageToken: newCursor,
		Metrics:       metrics,
	}, nil
}

func (c *Client) getInitialBrowserFeatureCount(
	ctx context.Context,
	txn *spanner.ReadOnlyTransaction,
	parsedToken *BrowserFeatureCountCursor,
	browser string,
	startAt time.Time) (int64, error) {
	// For pagination, we have the existing count. Return early.
	if parsedToken != nil {
		return parsedToken.LastCumulativeCount, nil
	}
	// On the initial page, we need to get the sum of all the features before the start.
	var initialCount int64
	err := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT COALESCE(SUM(daily_feature_count), 0)
					FROM (
						SELECT COUNT(DISTINCT FeatureID) AS daily_feature_count
						FROM BrowserFeatureAvailabilities bfa
						JOIN BrowserReleases
						ON bfa.BrowserName = BrowserReleases.BrowserName
						AND bfa.BrowserVersion = BrowserReleases.BrowserVersion
						WHERE bfa.BrowserName = @browserName AND ReleaseDate < @startAt
						GROUP BY ReleaseDate
					)`,
		Params: map[string]interface{}{
			"browserName": browser,
			"startAt":     startAt,
		},
	}).Do(func(r *spanner.Row) error {
		return r.Column(0, &initialCount)
	})

	return initialCount, err
}

func createListBrowserFeatureCountMetricStatement(
	browser string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *BrowserFeatureCountCursor,
) spanner.Statement {
	params := map[string]interface{}{
		"browserName": browser,
		"startAt":     startAt,
		"endAt":       endAt,
		"pageSize":    pageSize,
	}
	var pageFilter string
	if pageToken != nil {
		// Add filter for pagination if a page token is provided
		pageFilter = `
		AND BrowserReleases.ReleaseDate > @lastReleaseDate`
		params["lastReleaseDate"] = pageToken.LastReleaseDate
	}

	// Construct the query
	// This query selects the 'ReleaseDate' and the feature counts for each release date.
	query := fmt.Sprintf(`
SELECT
    BrowserReleases.ReleaseDate AS ReleaseDate,
    COUNT(DISTINCT FeatureID) AS FeatureCount
FROM BrowserFeatureAvailabilities bfa
JOIN BrowserReleases
ON bfa.BrowserName = BrowserReleases.BrowserName
AND bfa.BrowserVersion = BrowserReleases.BrowserVersion
WHERE
    bfa.BrowserName = @browserName
    AND BrowserReleases.ReleaseDate >= @startAt
    AND BrowserReleases.ReleaseDate < @endAt
	%s
GROUP BY ReleaseDate
ORDER BY ReleaseDate ASC
LIMIT @pageSize
`, pageFilter)

	stmt := spanner.NewStatement(query)
	stmt.Params = params

	return stmt
}
