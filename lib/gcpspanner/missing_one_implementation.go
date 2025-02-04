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

func init() {
	gcpMissingOneImplTemplate = NewQueryTemplate(gcpMissingOneImplCountRawTemplate)
	localMissingOneImplTemplate = NewQueryTemplate(localMissingOneImplCountRawTemplate)
}

// nolint: gochecknoglobals // WONTFIX. Compile the template once at startup. Startup fails if invalid.
var (
	// gcpMissingOneImplTemplate is the compiled version of gcpMissingOneImplCountRawTemplate.
	gcpMissingOneImplTemplate BaseQueryTemplate
	// localMissingOneImplTemplate is the compiled version of localMissingOneImplCountRawTemplate.
	localMissingOneImplTemplate BaseQueryTemplate
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

const localMissingOneImplCountRawTemplate = `
WITH TargetBrowserUnsupportedFeatures AS (
    SELECT bfse.WebFeatureID, bfse.EventReleaseDate
    FROM BrowserFeatureSupportEvents bfse
    WHERE bfse.TargetBrowserName = @targetBrowserParam
      AND bfse.SupportStatus = 'unsupported'
	  {{ .ExcludedFeatureFilter }}
),
OtherBrowsersSupportedFeatures AS (
    SELECT
        bfse_other.WebFeatureID,
        bfse_other.EventReleaseDate,
        ARRAY_AGG(DISTINCT bfse_other.TargetBrowserName) AS SupportedBrowsers
    FROM
        BrowserFeatureSupportEvents bfse_other
    WHERE
        bfse_other.SupportStatus = 'supported'
		{{ .OtherExcludedFeatureFilter }}
    GROUP BY
        bfse_other.WebFeatureID, bfse_other.EventReleaseDate
)
SELECT releases.EventReleaseDate,
	(
		SELECT COUNT(DISTINCT tbuf.WebFeatureID)
		FROM TargetBrowserUnsupportedFeatures tbuf
		INNER JOIN OtherBrowsersSupportedFeatures obsf
			ON tbuf.WebFeatureID = obsf.WebFeatureID
			AND obsf.EventReleaseDate = tbuf.EventReleaseDate
		WHERE
			tbuf.EventReleaseDate = releases.EventReleaseDate
			{{ range $browserParamName := .OtherBrowsersParamNames }}
				AND
				@{{ $browserParamName }} IN UNNEST(obsf.SupportedBrowsers)
			{{ end }}
	) AS Count
FROM (
    SELECT DISTINCT ReleaseDate AS EventReleaseDate
    FROM BrowserReleases
    WHERE BrowserName IN UNNEST(@allBrowsersParam)
	AND ReleaseDate >= @startAt
	AND ReleaseDate < @endAt
	{{if .ReleaseDateParam }}
	AND ReleaseDate < @{{ .ReleaseDateParam }}
	{{end}}
	AND ReleaseDate < CURRENT_TIMESTAMP()
) releases
ORDER BY releases.EventReleaseDate DESC
LIMIT @limit;
`

const gcpMissingOneImplCountRawTemplate = `
SELECT releases.EventReleaseDate,
	(
		SELECT COUNT(DISTINCT bfse.WebFeatureID)
		FROM BrowserFeatureSupportEvents bfse
		WHERE
			bfse.EventReleaseDate = releases.EventReleaseDate
			AND bfse.TargetBrowserName = @targetBrowserParam
			AND bfse.SupportStatus = 'unsupported'
			{{ .ExcludedFeatureFilter }}
		{{ range $browserParamName := .OtherBrowsersParamNames }}
			AND EXISTS (
				SELECT 1
				FROM BrowserFeatureSupportEvents bfse_other
				WHERE bfse_other.WebFeatureID = bfse.WebFeatureID
				AND bfse_other.TargetBrowserName = @{{ $browserParamName }}
				AND bfse_other.SupportStatus = 'supported'
				AND bfse_other.EventReleaseDate = bfse.EventReleaseDate
				{{ $.OtherExcludedFeatureFilter }}
			)
		{{ end }}
	) AS Count
FROM (
    SELECT DISTINCT ReleaseDate AS EventReleaseDate
    FROM BrowserReleases
    WHERE BrowserName IN UNNEST(@allBrowsersParam)
	AND ReleaseDate >= @startAt
	AND ReleaseDate < @endAt
	{{if .ReleaseDateParam }}
	AND ReleaseDate < @{{ .ReleaseDateParam }}
	{{end}}
	AND ReleaseDate < CURRENT_TIMESTAMP()
) releases
ORDER BY releases.EventReleaseDate DESC
LIMIT @limit;
`

type missingOneImplTemplateData struct {
	ReleaseDateParam           string
	OtherBrowsersParamNames    []string
	ExcludedFeatureFilter      string
	OtherExcludedFeatureFilter string
}

// MissingOneImplementationQuery contains the base query for all missing one implementation
// related queries.
type MissingOneImplementationQuery interface {
	Query(missingOneImplTemplateData) string
}

// GCPMissingOneImplementationQuery provides a base query that is optimal for GCP Spanner to retrieve the information
// described in the MissingOneImplementationQuery interface.
type GCPMissingOneImplementationQuery struct{}

func (q GCPMissingOneImplementationQuery) Query(data missingOneImplTemplateData) string {
	return gcpMissingOneImplTemplate.Execute(data)
}

// LocalMissingOneImplementationQuery is a version of the base query that works well on the local emulator.
// For some reason, the local emulator takes at least 1 minute with the fake data when using the
// GCPMissingOneImplementationQuery.
// Rather than sacrifice performance for the sake of compatibility, we have this LocalMissingOneImplementationQuery
// implementation which is good for the volume of data locally.
// TODO. Consolidate to using either LocalMissingOneImplementationQuery or GCPMissingOneImplementationQuery to reduce
// the maintenance burden.
type LocalMissingOneImplementationQuery struct{}

func (q LocalMissingOneImplementationQuery) Query(data missingOneImplTemplateData) string {
	return localMissingOneImplTemplate.Execute(data)
}

func buildMissingOneImplTemplate(
	cursor *missingOneImplCursor,
	targetBrowser string,
	otherBrowsers []string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	excludedFeatureIDs []string,
	tmpl MissingOneImplementationQuery,
) spanner.Statement {
	params := map[string]interface{}{}
	allBrowsers := make([]string, len(otherBrowsers)+1)
	copy(allBrowsers, otherBrowsers)
	allBrowsers[len(allBrowsers)-1] = targetBrowser
	params["targetBrowserParam"] = targetBrowser
	params["allBrowsersParam"] = allBrowsers
	otherBrowsersParamNames := make([]string, 0, len(otherBrowsers))
	for i := range otherBrowsers {
		paramName := fmt.Sprintf("otherBrowser%d", i)
		params[paramName] = otherBrowsers[i]
		otherBrowsersParamNames = append(otherBrowsersParamNames, paramName)
	}

	params["limit"] = pageSize

	releaseDateParamName := ""
	if cursor != nil {
		releaseDateParamName = "releaseDateCursor"
		params[releaseDateParamName] = cursor.ReleaseDate
	}

	var excludedFeatureFilter, otherExcludedFeatureFilter string
	if len(excludedFeatureIDs) > 0 {
		params["excludedFeatureIDs"] = excludedFeatureIDs
		excludedFeatureFilter = "AND bfse.WebFeatureID NOT IN UNNEST(@excludedFeatureIDs)"
		otherExcludedFeatureFilter = "AND bfse_other.WebFeatureID NOT IN UNNEST(@excludedFeatureIDs)"
	}

	params["startAt"] = startAt
	params["endAt"] = endAt

	tmplData := missingOneImplTemplateData{
		ReleaseDateParam:           releaseDateParamName,
		OtherBrowsersParamNames:    otherBrowsersParamNames,
		ExcludedFeatureFilter:      excludedFeatureFilter,
		OtherExcludedFeatureFilter: otherExcludedFeatureFilter,
	}
	sql := tmpl.Query(tmplData)
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

	// Get excluded feature IDs
	excludedFeatureIDs, err := c.getFeatureIDsForEachExcludedFeatureKey(ctx, txn)
	if err != nil {
		return nil, err
	}

	stmt := buildMissingOneImplTemplate(
		cursor,
		targetBrowser,
		otherBrowsers,
		startAt,
		endAt,
		pageSize,
		excludedFeatureIDs,
		c.missingOneImplQuery,
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
