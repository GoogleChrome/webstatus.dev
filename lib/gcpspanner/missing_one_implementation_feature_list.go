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
    WHERE bfse.TargetBrowserName = @targetBrowserParam
      AND bfse.EventReleaseDate = @targetDate
      AND bfse.SupportStatus = 'unsupported'
)
AND {{ range $browserParamName := .OtherBrowsersParamNames }}
EXISTS (
    SELECT 1
    FROM BrowserFeatureSupportEvents bfse_other
    WHERE bfse_other.WebFeatureID = wf.ID
      AND bfse_other.TargetBrowserName = @{{ $browserParamName }}
      AND bfse_other.EventReleaseDate = @targetDate
      AND bfse_other.SupportStatus = 'supported'
)
AND
{{ end }}
1=1
`

type missingOneImplFeatureListTemplateData struct {
	OtherBrowsersParamNames []string
}

func buildMissingOneImplFeatureListTemplate(
	targetBrowser string,
	otherBrowsers []string,
	targetDate time.Time,
) spanner.Statement {
	params := map[string]interface{}{}
	allBrowsers := make([]string, len(otherBrowsers)+1)
	copy(allBrowsers, otherBrowsers)
	allBrowsers[len(allBrowsers)-1] = targetBrowser
	params["targetBrowserParam"] = targetBrowser
	otherBrowsersParamNames := make([]string, 0, len(otherBrowsers))
	for i := range otherBrowsers {
		paramName := fmt.Sprintf("otherBrowser%d", i)
		params[paramName] = otherBrowsers[i]
		otherBrowsersParamNames = append(otherBrowsersParamNames, paramName)
	}

	params["targetDate"] = targetDate

	tmplData := missingOneImplFeatureListTemplateData{
		OtherBrowsersParamNames: otherBrowsersParamNames,
	}

	sql := missingOneImplFeatureListTemplate.Execute(tmplData)
	stmt := spanner.NewStatement(sql)
	stmt.Params = params

	return stmt
}

func (c *Client) MissingOneImplFeatureList(
	ctx context.Context,
	targetBrowser string,
	otherBrowsers []string,
	targetDate time.Time,
) (*MissingOneImplFeatureListPage, error) {
	txn := c.ReadOnlyTransaction()
	defer txn.Close()

	stmt := buildMissingOneImplFeatureListTemplate(
		targetBrowser,
		otherBrowsers,
		targetDate,
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

	return &page, nil
}
