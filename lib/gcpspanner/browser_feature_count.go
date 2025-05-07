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

// BrowserFeatureCountTemplateData contains the variables for the template query.
type BrowserFeatureCountTemplateData struct {
	BrowserFilter string
}

func (c *Client) ListBrowserFeatureCountMetric(
	ctx context.Context,
	targetBrowser string,
	targetMobileBrowser string,
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
	// 1. Get ignored feature IDs
	ignoredFeatureIDs, err := c.getIgnoredFeatureIDsForStats(ctx, txn)
	if err != nil {
		return nil, err
	}
	// 2. Calculate initial cumulative count
	cumulativeCount, err := c.getInitialBrowserFeatureCount(
		ctx, txn, parsedToken, targetBrowser, targetMobileBrowser, startAt, ignoredFeatureIDs)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	// 3. Process results and update cumulative count
	stmt := createListBrowserFeatureCountMetricStatement(
		targetBrowser,
		targetMobileBrowser,
		startAt,
		endAt,
		pageSize,
		parsedToken,
		ignoredFeatureIDs,
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
	targetBrowser string,
	targetMobileBrowser string,
	startAt time.Time,
	excludedFeatureIDs []string) (int64, error) {
	// For pagination, we have the existing count. Return early.
	if parsedToken != nil {
		return parsedToken.LastCumulativeCount, nil
	}

	var browserFilter string
	if targetMobileBrowser != "" {
		browserFilter = `(bfa.BrowserName = @targetBrowserName OR bfa.BrowserName = @targetMobileBrowserName)`
	} else {
		browserFilter = `bfa.BrowserName = @targetBrowserName`
		targetMobileBrowser = targetBrowser
	}

	params := map[string]interface{}{
		"targetBrowserName":       targetBrowser,
		"targetMobileBrowserName": targetMobileBrowser,
		"startAt":                 startAt,
	}

	var excludedFeatureFilter string
	if len(excludedFeatureIDs) > 0 {
		excludedFeatureFilter = `
            AND WebFeatureID NOT IN UNNEST(@excludedFeatureIDs)`
		params["excludedFeatureIDs"] = excludedFeatureIDs
	}

	// On the initial page, we need to get the sum of all the features before the start.
	var initialCount int64
	err := txn.Query(ctx, spanner.Statement{
		SQL: fmt.Sprintf(`SELECT COALESCE(SUM(daily_feature_count), 0)
					FROM (
						SELECT COUNT(DISTINCT WebFeatureID) AS daily_feature_count
						FROM BrowserReleases br
						LEFT JOIN (
							SELECT
								bfa.WebFeatureID
							FROM BrowserFeatureAvailabilities bfa
							WHERE bfa.BrowserName = @targetBrowserName
							INNER JOIN (
								SELECT
									bfa2. WebFeatureID,
									bfa2.BrowserName
								FROM BrowserFeatureAvailabilities bfa2
								WHERE bfa2.BrowserName = @targetMobileBrowserName
							ON bfa.WebFeatureID = bfa2.WebFeatureID
							AND bfa.BrowserVersion = br.BrowserVersion
						)
						ON bfa.BrowserName = br.BrowserName
						AND bfa.BrowserVersion = br.BrowserVersion
						%s
						WHERE %s
						AND ReleaseDate < @startAt
						GROUP BY ReleaseDate
					)`,
			excludedFeatureFilter, browserFilter),
		Params: params,
	}).Do(func(r *spanner.Row) error {
		return r.Column(0, &initialCount)
	})

	return initialCount, err
}

func createListBrowserFeatureCountMetricStatement(
	targetBrowser string,
	targetMobileBrowser string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *BrowserFeatureCountCursor,
	excludedFeatureIDs []string,
) spanner.Statement {

	params := map[string]interface{}{
		"targetBrowserName":       targetBrowser,
		"targetMobileBrowserName": targetMobileBrowser,
		"startAt":                 startAt,
		"endAt":                   endAt,
		"pageSize":                pageSize,
	}
	var pageFilter string
	if pageToken != nil {
		// Add filter for pagination if a page token is provided
		pageFilter = `
		AND BrowserReleases.ReleaseDate > @lastReleaseDate`
		params["lastReleaseDate"] = pageToken.LastReleaseDate
	}

	var excludedFeatureFilter string
	if len(excludedFeatureIDs) > 0 {
		params["excludedFeatureIDs"] = excludedFeatureIDs
		excludedFeatureFilter = "AND bfa.WebFeatureID NOT IN UNNEST(@excludedFeatureIDs)"
	}

	var browserFilter string
	if targetMobileBrowser != "" {
		browserFilter = `(BrowserReleases.BrowserName = @targetBrowserName ` +
			`OR BrowserReleases.BrowserName = @targetMobileBrowserName)`
	} else {
		browserFilter = `bfa.BrowserName = @targetBrowserName`
	}
	// Construct the query
	// This query selects the 'ReleaseDate' and the feature counts for each release date.
	query := fmt.Sprintf(`
SELECT
    BrowserReleases.ReleaseDate AS ReleaseDate,
    COUNT(DISTINCT CASE WHEN bfa.WebFeatureID IS NOT NULL %s THEN bfa.WebFeatureID ELSE NULL END) AS FeatureCount
FROM BrowserFeatureAvailabilities bfa
RIGHT JOIN BrowserReleases
ON bfa.BrowserName = BrowserReleases.BrowserName
AND bfa.BrowserVersion = BrowserReleases.BrowserVersion
WHERE
	%s
    AND BrowserReleases.ReleaseDate >= @startAt
    AND BrowserReleases.ReleaseDate < @endAt
	%s
GROUP BY ReleaseDate
ORDER BY ReleaseDate ASC
LIMIT @pageSize
`, excludedFeatureFilter, browserFilter, pageFilter)

	stmt := spanner.NewStatement(query)
	stmt.Params = params

	return stmt
}
