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
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func init() {
	missingOneImplTemplate = NewQueryTemplate(missingOneImplCountRawTemplate)
}

// nolint: gochecknoglobals // WONTFIX. Compile the template once at startup. Startup fails if invalid.
var (
	// missingOneImplTemplate is the compiled version of missingOneImplCountRawTemplate.
	missingOneImplTemplate BaseQueryTemplate
)

// MissingOneImplCountPage contains the details for the missing one implementation count request.
type MissingOneImplCountPage struct {
	NextPageToken *string
	Metrics       []MissingOneImplCount
}

// spannerMissingOneImplCount is a wrapper for the missing one implementation count.
type spannerMissingOneImplCount struct {
	MissingOneImplCount
}

// MissingOneImplCount contains information regarding the count of features implemented in all other browsers but not
// in the target browser.
type MissingOneImplCount struct {
	EventReleaseDate time.Time `spanner:"EventReleaseDate"`
	Count            int64     `spanner:"Count"`
}

// missingOneImplCursor: Represents a point for resuming queries based on the last
// browser release date. Useful for pagination.
type missingOneImplCursor struct {
	ReleaseDate time.Time `json:"release_date"`
}

// decodeMissingOneImplCursor provides a wrapper around the generic decodeCursor.
func decodeMissingOneImplCursor(cursor string) (*missingOneImplCursor, error) {
	return decodeCursor[missingOneImplCursor](cursor)
}

// encodeMissingOneImplCursor provides a wrapper around the generic encodeCursor.
func encodeMissingOneImplCursor(releaseDate time.Time) string {
	return encodeCursor(missingOneImplCursor{
		ReleaseDate: releaseDate,
	})
}

const missingOneImplCountRawTemplate = `
SELECT releases.EventReleaseDate,
	(
		SELECT COUNT(DISTINCT wf.ID)
		FROM WebFeatures wf
		INNER JOIN BrowserFeatureSupportEvents bfse
			ON wf.ID = bfse.WebFeatureID
			AND bfse.EventReleaseDate = releases.EventReleaseDate
		WHERE
			bfse.TargetBrowserName = @targetBrowserParam AND bfse.SupportStatus = 'unsupported'
			AND EXISTS (
				SELECT 1
				FROM BrowserFeatureSupportEvents bfse_other
				WHERE bfse_other.WebFeatureID = wf.ID
					AND bfse_other.SupportStatus = 'supported'
					AND bfse_other.TargetBrowserName IN UNNEST(@{{.OtherBrowsersListParamName}})
					AND bfse_other.EventReleaseDate = releases.EventReleaseDate
					AND bfse_other.EventReleaseDate >= @startAt
					AND bfse_other.EventReleaseDate < @endAt
					{{if .ReleaseDateParam }}
					AND bfse_other.EventReleaseDate < @{{ .ReleaseDateParam }}
					{{end}}
				GROUP BY bfse_other.WebFeatureID
				HAVING COUNT(DISTINCT bfse_other.TargetBrowserName) = @{{ .OtherBrowserListSizeParamName }}
			)
	) AS Count
FROM (
    SELECT DISTINCT EventReleaseDate
    FROM BrowserFeatureSupportEvents
    WHERE TargetBrowserName = @targetBrowserParam
		AND EventBrowserName IN UNNEST(@allBrowsersParam)
) releases
WHERE releases.EventReleaseDate >= @startAt
  AND releases.EventReleaseDate < @endAt
  {{if .ReleaseDateParam }}
  AND releases.EventReleaseDate < @{{ .ReleaseDateParam }}
  {{end}}
ORDER BY releases.EventReleaseDate DESC
LIMIT @limit;
`

type missingOneImplTemplateData struct {
	OtherBrowserListSizeParamName string
	OtherBrowsersListParamName    string
	ReleaseDateParam              string
}

func buildMissingOneImplTemplate(
	cursor *missingOneImplCursor,
	targetBrowser string,
	otherBrowsers []string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
) spanner.Statement {
	params := map[string]interface{}{}
	allBrowsers := make([]string, len(otherBrowsers)+1)
	copy(allBrowsers, otherBrowsers)
	allBrowsers[len(allBrowsers)-1] = targetBrowser
	params["targetBrowserParam"] = targetBrowser
	params["allBrowsersParam"] = allBrowsers

	params["limit"] = pageSize

	releaseDateParamName := ""
	if cursor != nil {
		releaseDateParamName = "releaseDateCursor"
		params[releaseDateParamName] = cursor.ReleaseDate
	}

	params["startAt"] = startAt
	params["endAt"] = endAt

	otherBrowsersListParamName := "otherBrowsersList"
	params[otherBrowsersListParamName] = otherBrowsers

	otherBrowserListSizeParamName := "otherBrowserListSize"
	params[otherBrowserListSizeParamName] = len(otherBrowsers)

	tmplData := missingOneImplTemplateData{
		OtherBrowserListSizeParamName: otherBrowserListSizeParamName,
		OtherBrowsersListParamName:    otherBrowsersListParamName,
		ReleaseDateParam:              releaseDateParamName,
	}
	sql := missingOneImplTemplate.Execute(tmplData)
	stmt := spanner.NewStatement(sql)
	stmt.Params = params

	return stmt
}

func (c *Client) ListMissingOneImplCounts(
	ctx context.Context,
	targetBrowser string,
	otherBrowsers []string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) (*MissingOneImplCountPage, error) {

	var cursor *missingOneImplCursor
	var err error
	if pageToken != nil {
		cursor, err = decodeMissingOneImplCursor(*pageToken)
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
	}

	txn := c.ReadOnlyTransaction()
	defer txn.Close()

	stmt := buildMissingOneImplTemplate(
		cursor,
		targetBrowser,
		otherBrowsers,
		startAt,
		endAt,
		pageSize,
	)

	it := txn.Query(ctx, stmt)
	defer it.Stop()

	var results []MissingOneImplCount
	for {
		row, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var result spannerMissingOneImplCount
		if err := row.ToStruct(&result); err != nil {
			return nil, err
		}
		actualResult := MissingOneImplCount{
			EventReleaseDate: result.EventReleaseDate,
			Count:            result.Count,
		}
		results = append(results, actualResult)
	}

	page := MissingOneImplCountPage{
		Metrics:       results,
		NextPageToken: nil,
	}

	if len(results) == pageSize {
		token := encodeMissingOneImplCursor(results[len(results)-1].EventReleaseDate)
		page.NextPageToken = &token
	}

	return &page, nil
}
