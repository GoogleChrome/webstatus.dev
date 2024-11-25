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
	"math/big"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)
 
const (
	getChromiumDailyUsageBaseRawTemplate = `
	SELECT
		dchm.Day as Date,
		dchm.Rate as Usage
	FROM DailyChromiumHistogramMetrics dchm
	LEFT OUTER JOIN WebFeatureChromiumHistogramEnumValues wfchev
	ON wfchev.ChromiumHistogramEnumValueID = dchm.ChromiumHistogramEnumValueID
	JOIN WebFeatures wf
	ON wfchev.WebFeatureID = wf.ID
	WHERE wf.FeatureKey = @featureKey
	AND TIMESTAMP(dchm.Day) >= @startAt AND TIMESTAMP(dchm.Day) < @endAt
{{ if .PageFilter }}
 	{{ .PageFilter }}
{{ end }}
 	ORDER BY Date DESC LIMIT @pageSize`

	commonChromiumDailyUsagePaginationRawTemplate = `
		AND dchm.Day < @lastDate`
)

func init() {
	getChromiumDailyUsageBaseTemplate = NewQueryTemplate(getChromiumDailyUsageBaseRawTemplate)
}

// nolint: gochecknoglobals // WONTFIX. Compile the template once at startup. Startup fails if invalid.
var (
	// getChromiumDailyUsageBaseTemplate is the compiled version of getChromiumDailyUsageBaseRawTemplate.
	getChromiumDailyUsageBaseTemplate BaseQueryTemplate
)

// ChromiumDailyUsageStatsWithDate contains usage stats for a feature at a given date.
type ChromiumDailyUsageStatWithDate struct {
	Date  civil.Date `spanner:"Date"`
	Usage *big.Rat   `spanner:"Usage"`
}

// ChromiumDailyUsageTemplateData contains the variables for getChromiumDailyUsageBaseRawTemplate.
type ChromiumDailyUsageTemplateData struct {
	PageFilter string
}

// nolint: revive
func (c *Client) ListChromiumDailyUsageStatsForFeatureID(
	ctx context.Context,
	featureKey string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]ChromiumDailyUsageStatWithDate, *string, error) {

	params := map[string]interface{}{
		"featureKey": featureKey,
		"startAt":    startAt,
		"endAt":      endAt,
		"pageSize":   pageSize,
	}

	tmplData := ChromiumDailyUsageTemplateData{
		PageFilter: "",
	}

	if pageToken != nil {
		cursor, err := decodeChromiumDailyUsageCursor(*pageToken)
		if err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		params["lastDate"] = cursor.LastDate
		tmplData.PageFilter = commonChromiumDailyUsagePaginationRawTemplate
	}
	tmpl := getChromiumDailyUsageBaseTemplate.Execute(tmplData)
	stmt := spanner.NewStatement(tmpl)
	stmt.Params = params

	txn := c.Single()
	defer txn.Close()
	it := txn.Query(ctx, stmt)
	defer it.Stop()

	var usageStats []ChromiumDailyUsageStatWithDate
	for {
		row, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var usageStat ChromiumDailyUsageStatWithDate
		if err := row.ToStruct(&usageStat); err != nil {
			return nil, nil, err
		}
		usageStats = append(usageStats, usageStat)
	}

	if len(usageStats) == pageSize {
		lastUsageStat := usageStats[len(usageStats)-1]
		newCursor := encodeChromiumDailyUsageCursor(lastUsageStat.Date)

		return usageStats, &newCursor, nil
	}

	return usageStats, nil, nil
}
