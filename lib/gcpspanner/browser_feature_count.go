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
	targetMobileBrowser *string,
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
	targetMobileBrowser *string,
	startAt time.Time,
	excludedFeatureIDs []string) (int64, error) {
	// For pagination, we have the existing count. Return early.
	if parsedToken != nil {
		return parsedToken.LastCumulativeCount, nil
	}

	params := map[string]interface{}{
		"targetBrowserName": targetBrowser,
		"startAt":           startAt,
	}

	var excludedFeatureFilter string
	if len(excludedFeatureIDs) > 0 {
		excludedFeatureFilter = `
            AND bfa1.WebFeatureID NOT IN UNNEST(@excludedFeatureIDs)`
		params["excludedFeatureIDs"] = excludedFeatureIDs
	}

	var targetMobileBrowserFilter string
	if targetMobileBrowser != nil {
		targetMobileBrowserFilter = "AND bfa2.BrowserName = @targetMobileBrowserName"
		params["targetMobileBrowserName"] = *targetMobileBrowser
	}

	// On the initial page, we need to get the sum of all the features before the start.
	var initialCount int64
	err := txn.Query(ctx, spanner.Statement{
		SQL: fmt.Sprintf(`
SELECT
	COALESCE(SUM(daily_feature_count), 0)
FROM (
	SELECT
		COUNT(DISTINCT bfa1.WebFeatureID) AS daily_feature_count
	FROM
		BrowserFeatureAvailabilities AS bfa1
	JOIN
		BrowserReleases AS br
		ON bfa1.BrowserName = br.BrowserName
		AND bfa1.BrowserVersion = br.BrowserVersion
		%s
	JOIN
		BrowserFeatureAvailabilities AS bfa2
		ON bfa1.WebFeatureID = bfa2.WebFeatureID
	WHERE
		bfa1.BrowserName = @targetBrowserName
		AND br.ReleaseDate < @startAt
		%s
)`, excludedFeatureFilter, targetMobileBrowserFilter),
		Params: params,
	}).Do(func(r *spanner.Row) error {
		return r.Column(0, &initialCount)
	})

	return initialCount, err
}

func createListBrowserFeatureCountMetricStatement(
	targetBrowser string,
	targetMobileBrowser *string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *BrowserFeatureCountCursor,
	excludedFeatureIDs []string,
) spanner.Statement {

	params := map[string]interface{}{
		"targetBrowserName": targetBrowser,
		"startAt":           startAt,
		"endAt":             endAt,
		"pageSize":          pageSize,
	}
	var pageFilter string
	if pageToken != nil {
		// Add filter for pagination if a page token is provided
		pageFilter = `
		AND br.ReleaseDate > @lastReleaseDate`
		params["lastReleaseDate"] = pageToken.LastReleaseDate
	}

	var excludedFeatureFilter string
	if len(excludedFeatureIDs) > 0 {
		params["excludedFeatureIDs"] = excludedFeatureIDs
		excludedFeatureFilter = "AND cf.WebFeatureID NOT IN UNNEST(@excludedFeatureIDs)"
	}

	var targetMobileBrowserFilter string
	if targetMobileBrowser != nil {
		targetMobileBrowserFilter = "AND bfa2.BrowserName = @targetMobileBrowserName"
		params["targetMobileBrowserName"] = *targetMobileBrowser
	}

	// Construct the query
	// This query selects the 'ReleaseDate' and the feature counts for each release date.
	query := fmt.Sprintf(`
WITH CommonFeatures AS (
    SELECT
        bfa1.BrowserName AS TargetBrowserName,
        bfa1.BrowserVersion AS TargetBrowserVersion,
        bfa1.WebFeatureID
    FROM
        BrowserFeatureAvailabilities AS bfa1
    JOIN
        BrowserFeatureAvailabilities AS bfa2
        ON bfa1.WebFeatureID = bfa2.WebFeatureID
    WHERE
        bfa1.BrowserName = @targetBrowserName
		%s
)
SELECT
    br.ReleaseDate,
    COUNT(DISTINCT CASE WHEN cf.WebFeatureID IS NOT NULL %s THEN cf.WebFeatureID ELSE NULL END) AS FeatureCount
FROM
    BrowserReleases AS br
LEFT JOIN
    CommonFeatures AS cf
    ON br.BrowserName = cf.TargetBrowserName
    AND br.BrowserVersion = cf.TargetBrowserVersion
WHERE
    br.BrowserName = @targetBrowserName
	AND br.ReleaseDate >= @startAt
	AND br.ReleaseDate < @endAt
	%s
GROUP BY
    br.ReleaseDate
ORDER BY
    br.ReleaseDate
LIMIT @pageSize
`, targetMobileBrowserFilter, excludedFeatureFilter, pageFilter)

	stmt := spanner.NewStatement(query)
	stmt.Params = params

	return stmt
}
