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
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func init() {
	missingOneImplFeatureListTemplate = NewQueryTemplate(missingOneImplFeatureListRawTemplate)
}

// nolint: gochecknoglobals // WONTFIX. Compile the template once at startup. Startup fails if invalid.
var (
	// missingOneImplFeatureListTemplate is the compiled version of missingOneImplFeatureListRawTemplate.
	missingOneImplFeatureListTemplate BaseQueryTemplate
)

// MissingOneImplFeatureListPage contains the details for the missing one implementation feature list request.
type MissingOneImplFeatureListPage struct {
	NextPageToken *string
	FeatureList   []MissingOneImplFeature
}

// missingOneImplFeatureListCursor: Represents a point for resuming queries based on the
// numerical offset from the start of the result set. Useful for pagination.
type missingOneImplFeatureListCursor struct {
	Offset int `json:"offset"`
}

// decodeMissingOneImplFeatureListCursor provides a wrapper around the generic decodeCursor.
func decodeMissingOneImplFeatureListCursor(cursor string) (*missingOneImplFeatureListCursor, error) {
	return decodeCursor[missingOneImplFeatureListCursor](cursor)
}

// encodeMissingOneImplFeatureListCursor provides a wrapper around the generic encodeCursor.
func encodeMissingOneImplFeatureListCursor(offset int) string {
	return encodeCursor(missingOneImplFeatureListCursor{
		Offset: offset,
	})
}

// MissingOneImplFeature contains information regarding the list of features implemented in all other browsers but not
// in the target browser.
type MissingOneImplFeature struct {
	WebFeatureID string `spanner:"KEY"`
}

const missingOneImplFeatureListRawTemplate = `
WITH UnsupportedFeatures AS (
    -- This CTE identifies WebFeatureIDs that are not supported by the target browser.
    SELECT DISTINCT
        bfse1.WebFeatureID
    FROM
        BrowserFeatureSupportEvents bfse1
	{{ .BrowserSupportedFeaturesFilter }}
),
OtherBrowserSupport AS (
    -- This CTE identifies WebFeatureIDs that are supported by all
    -- 'other' browsers on the specific event release date.
    SELECT
        WebFeatureID
    FROM
        BrowserFeatureSupportEvents
    WHERE
        EventReleaseDate = @targetDate
        AND SupportStatus = 'supported'
        AND TargetBrowserName IN UNNEST(@otherBrowserNames)
    GROUP BY
        WebFeatureID
    HAVING
        -- Ensures all other browsers support the feature.
        COUNT(DISTINCT TargetBrowserName) = @numOtherBrowsers
)
SELECT
    wf.FeatureKey AS KEY
FROM
    -- Start with features that meet the "other browser" support criteria
    OtherBrowserSupport obs
JOIN
    -- Then ensure they also meet the unsupported conditions for target browser.
    UnsupportedFeatures uf ON obs.WebFeatureID = uf.WebFeatureID
JOIN
    -- Finally, get the FeatureKey
    WebFeatures wf ON obs.WebFeatureID = wf.ID
	{{ .ExcludedFeatureFilter }}
ORDER BY
    KEY ASC
LIMIT
	@limit
{{ if .Offset }}
OFFSET {{ .Offset }}
{{ end }}`

type missingOneImplFeatureListTemplateData struct {
	BrowserSupportedFeaturesFilter string
	Offset                         int
	ExcludedFeatureFilter          string
}

func buildMissingOneImplFeatureListTemplate(
	targetBrowser string,
	targetMobileBrowser *string,
	otherBrowsers []string,
	targetDate time.Time,
	cursor *missingOneImplFeatureListCursor,
	pageSize int,
	excludedFeatureIDs []string,
) spanner.Statement {
	params := map[string]interface{}{}
	params["numOtherBrowsers"] = len(otherBrowsers)
	params["otherBrowserNames"] = otherBrowsers
	params["targetBrowserName"] = targetBrowser

	var browserSupportedFeaturesFilter string
	if targetMobileBrowser != nil {
		params["targetMobileBrowserName"] = *targetMobileBrowser
		browserSupportedFeaturesFilter = `
			JOIN
				BrowserFeatureSupportEvents bfse2
			ON bfse1.WebFeatureID = bfse2.WebFeatureID
				AND bfse1.EventReleaseDate = @targetDate
				AND bfse2.EventReleaseDate = @targetDate
			WHERE
				bfse1.TargetBrowserName = @targetBrowserName
				AND bfse2.TargetBrowserName = @targetMobileBrowserName
				AND (
					bfse1.SupportStatus = 'unsupported'
					OR bfse2.SupportStatus = 'unsupported'
				)`
	} else {
		browserSupportedFeaturesFilter = `
			WHERE
				bfse1.TargetBrowserName = @targetBrowserName
				AND bfse1.SupportStatus = 'unsupported'
				AND bfse1.EventReleaseDate = @targetDate`
	}

	var excludedFeatureFilter string
	if len(excludedFeatureIDs) > 0 {
		params["excludedFeatureIDs"] = excludedFeatureIDs
		excludedFeatureFilter = "AND wf.ID NOT IN UNNEST(@excludedFeatureIDs)"
	}

	params["targetDate"] = targetDate
	params["limit"] = pageSize

	tmplData := missingOneImplFeatureListTemplateData{
		BrowserSupportedFeaturesFilter: browserSupportedFeaturesFilter,
		Offset:                         0,
		ExcludedFeatureFilter:          excludedFeatureFilter,
	}

	if cursor != nil {
		tmplData.Offset = cursor.Offset
	}

	sql := missingOneImplFeatureListTemplate.Execute(tmplData)
	stmt := spanner.NewStatement(sql)
	stmt.Params = params

	return stmt
}

func (c *Client) ListMissingOneImplementationFeatures(
	ctx context.Context,
	targetBrowser string,
	targetMobileBrowser *string,
	otherBrowsers []string,
	targetDate time.Time,
	pageSize int,
	pageToken *string,
) (*MissingOneImplFeatureListPage, error) {
	var cursor *missingOneImplFeatureListCursor
	var err error
	if pageToken != nil {
		cursor, err = decodeMissingOneImplFeatureListCursor(*pageToken)
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
	}

	txn := c.ReadOnlyTransaction()
	defer txn.Close()

	// Get ignored feature IDs
	ignoredFeatureIDs, err := c.getIgnoredFeatureIDsForStats(ctx, txn)
	if err != nil {
		return nil, err
	}

	stmt := buildMissingOneImplFeatureListTemplate(
		targetBrowser,
		targetMobileBrowser,
		otherBrowsers,
		targetDate,
		cursor,
		pageSize,
		ignoredFeatureIDs,
	)

	it := txn.Query(ctx, stmt)
	defer it.Stop()

	var results []MissingOneImplFeature
	for {
		row, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var result MissingOneImplFeature
		if err := row.ToStruct(&result); err != nil {
			return nil, err
		}
		results = append(results, MissingOneImplFeature{result.WebFeatureID})
	}

	page := MissingOneImplFeatureListPage{
		FeatureList:   results,
		NextPageToken: nil,
	}

	if len(results) == pageSize {
		previousOffset := 0
		if cursor != nil {
			previousOffset = cursor.Offset
		}
		token := encodeMissingOneImplFeatureListCursor(previousOffset + pageSize)
		page.NextPageToken = &token
	}

	return &page, nil
}
