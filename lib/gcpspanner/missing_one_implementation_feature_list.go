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
SELECT wf.FeatureKey as KEY
FROM WebFeatures wf
WHERE wf.ID IN (
    SELECT bfse.WebFeatureID
    FROM BrowserFeatureSupportEvents bfse
    WHERE {{ .TargetBrowserFilter }}
      AND bfse.EventReleaseDate = @targetDate
      AND bfse.SupportStatus = 'unsupported'
)
AND {{ range $browserFilter := .OtherBrowsersFilters }}
EXISTS (
    SELECT 1
    FROM BrowserFeatureSupportEvents bfse_other
    WHERE bfse_other.WebFeatureID = wf.ID
      AND {{ $browserFilter }}
      AND bfse_other.EventReleaseDate = @targetDate
      AND bfse_other.SupportStatus = 'supported'
)
AND
{{ end }}
1=1
{{ .ExcludedFeatureFilter }}
ORDER BY KEY ASC
LIMIT @limit
{{ if .Offset }}
OFFSET {{ .Offset }}
{{ end }}
`

type missingOneImplFeatureListTemplateData struct {
	OtherBrowsersFilters  []string
	TargetBrowserFilter   string
	Offset                int
	ExcludedFeatureFilter string
}

func buildMissingOneImplFeatureListTemplate(
	targetBrowsers []string,
	otherBrowsers [][]string,
	targetDate time.Time,
	cursor *missingOneImplFeatureListCursor,
	pageSize int,
	excludedFeatureIDs []string,
) spanner.Statement {
	params := map[string]interface{}{}
	var targetBrowserFilter string
	if len(targetBrowsers) > 1 {
		targetBrowserFilter = fmt.Sprintf("(bfse.TargetBrowserName = '%s' OR bfse.TargetBrowserName = '%s')",
			targetBrowsers[0], targetBrowsers[1])
	} else {
		targetBrowserFilter = fmt.Sprintf("bfse.TargetBrowserName = '%s'", targetBrowsers[0])
	}

	var otherBrowsersFilters []string
	for i := range otherBrowsers {
		if len(otherBrowsers[i]) > 1 {
			otherBrowsersFilters = append(otherBrowsersFilters,
				fmt.Sprintf("(bfse_other.TargetBrowserName = '%s' OR bfse_other.TargetBrowserName = '%s')",
					otherBrowsers[i][0], otherBrowsers[i][1]))
		} else {
			otherBrowsersFilters = append(otherBrowsersFilters,
				fmt.Sprintf("bfse_other.TargetBrowserName = '%s'", otherBrowsers[i][0]))
		}
	}

	var excludedFeatureFilter string
	if len(excludedFeatureIDs) > 0 {
		params["excludedFeatureIDs"] = excludedFeatureIDs
		excludedFeatureFilter = "AND wf.ID NOT IN UNNEST(@excludedFeatureIDs)"
	}

	params["targetDate"] = targetDate
	params["limit"] = pageSize

	tmplData := missingOneImplFeatureListTemplateData{
		OtherBrowsersFilters:  otherBrowsersFilters,
		TargetBrowserFilter:   targetBrowserFilter,
		Offset:                0,
		ExcludedFeatureFilter: excludedFeatureFilter,
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
	targetBrowsers []string,
	otherBrowsers [][]string,
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
		targetBrowsers,
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
