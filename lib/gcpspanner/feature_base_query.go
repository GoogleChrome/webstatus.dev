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
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// FeatureBaseQuery contains the base query for all feature
// related queries.
type FeatureBaseQuery struct {
	useCTE bool
}

// Query providess the basic information for a feature.
// It provides:
//  1. The Internal ID of the feature
//  2. The external ID from web features repo
//  3. The human readable name.
//  4. The baseline status.
//  5. The latest metrics from WPT.
//     It provides these metrics for both "stable" and "experimental" channels.
//     The metrics retrieved are for each unique BrowserName/Channel/FeatureID.
func (f FeatureBaseQuery) Query(stableFilters, experimentalFilters string) string {
	if f.useCTE {
		return f.cteQuery()
	}

	return f.nonCTEQuery(stableFilters, experimentalFilters)
}

func buildChannelMetricsFilter(channel string, latestRunResults []LatestRunResult) (string, map[string]interface{}) {
	count := 0
	filters := []string{}
	params := make(map[string]interface{}, len(latestRunResults))
	for _, result := range latestRunResults {
		paramBrowserName := fmt.Sprintf("%sbrowser%d", channel, count)
		paramTimeName := fmt.Sprintf("%stime%d", channel, count)
		params[paramTimeName] = result.TimeStart
		params[paramBrowserName] = result.BrowserName
		count++
		filter := fmt.Sprintf(
			"(metrics.BrowserName = @%s AND metrics.TimeStart = @%s)",
			paramBrowserName,
			paramTimeName,
		)
		filters = append(filters, filter)
	}
	filterStr := strings.Join(filters, " OR ")

	filterStr = " AND (" + filterStr + ")"
	return filterStr, params
}

// nonCTEQuery is the optimal query and is used on GCP Spanner.
// For some reason, the local emulator takes forever for this query as the number
// of features and metrics grows.
// This query is about 1.5-2x faster than the CTE version when used in GCP Spanner.
// Rather than sacrifice performance for the sake of compatibility, we will keep
// the nonCTEQuery.
func (f FeatureBaseQuery) nonCTEQuery(stableFilters, experimentalFilters string) string {
	return fmt.Sprintf(`
SELECT
	wf.ID,
	wf.FeatureID,
	wf.Name,
	COALESCE(fbs.Status, 'undefined') AS Status,
	COALESCE((
		SELECT ARRAY_AGG(STRUCT(
				BrowserName AS BrowserName,
				PassRate AS PassRate
			))
		FROM WPTRunFeatureMetrics @{FORCE_INDEX=MetricsFeatureChannelBrowserTimePassRate} metrics
		WHERE metrics.FeatureID = wf.FeatureID
		AND metrics.Channel = 'stable'
		%s
		-- GCP Spanner could have ARRAY<STRUCT<string, NUMERIC>>[]) as the default.
		-- but the emulator complains.
		-- Replace the following line in the future when the emulator supports it.
		-- ), ARRAY<STRUCT<string, NUMERIC>>[]) AS StableMetrics,
	), (SELECT ARRAY(SELECT AS STRUCT '' BrowserName, CAST(0.0 AS NUMERIC) PassRate))) AS StableMetrics,
	COALESCE((
		SELECT ARRAY_AGG(STRUCT(
				BrowserName AS BrowserName,
				PassRate AS PassRate
			))
		FROM WPTRunFeatureMetrics @{FORCE_INDEX=MetricsFeatureChannelBrowserTimePassRate} metrics
		WHERE metrics.FeatureID = wf.FeatureID
		AND metrics.Channel = 'experimental'
		%s
		-- GCP Spanner could have ARRAY<STRUCT<string, NUMERIC>>[]) as the default.
		-- but the emulator complains.
		-- Replace the following line in the future when the emulator supports it.
		-- ), ARRAY<STRUCT<string, NUMERIC>>[]) AS StableMetrics,
	), (SELECT ARRAY(SELECT AS STRUCT '' BrowserName, CAST(0.0 AS NUMERIC) PassRate))) AS ExperimentalMetrics
FROM WebFeatures wf
LEFT OUTER JOIN FeatureBaselineStatus fbs ON wf.FeatureID = fbs.FeatureID
`, stableFilters, experimentalFilters)
}

// cteQuery is a version of the base query that works on the local emulator.
// Refer to the comment on nonCTEQuery for more details.
// TODO. Consolidate these cteQuery and nonCTEQuery.
func (f FeatureBaseQuery) cteQuery() string {
	// nolint: lll // For now, keep it.
	return `
WITH
	LatestMetrics AS (
		SELECT
			FeatureID,
			Channel,
			BrowserName,
			MAX(TimeStart) AS LatestTimeStart
		FROM WPTRunFeatureMetrics  @{FORCE_INDEX=MetricsFeatureChannelBrowserTimePassRate}
		GROUP BY FeatureID, Channel, BrowserName
	),
	MetricsAggregation AS (
		SELECT
			lm.FeatureID,
			lm.Channel,
			lm.BrowserName,
			m.PassRate
		FROM LatestMetrics lm
		JOIN WPTRunFeatureMetrics m ON
			lm.FeatureID = m.FeatureID AND
			lm.Channel = m.Channel AND
			lm.BrowserName = m.BrowserName AND
			lm.LatestTimeStart = m.TimeStart
	)
SELECT
	wf.ID,
	wf.FeatureID,
	wf.Name,
	COALESCE(fbs.Status, 'undefined') AS Status,
	COALESCE((SELECT ARRAY_AGG(STRUCT(BrowserName, PassRate)) FROM MetricsAggregation WHERE FeatureID = wf.FeatureID AND Channel = 'stable'), (SELECT ARRAY(SELECT AS STRUCT '' BrowserName, CAST(0.0 AS NUMERIC) PassRate))) AS StableMetrics,
	COALESCE((SELECT ARRAY_AGG(STRUCT(BrowserName, PassRate)) FROM MetricsAggregation WHERE FeatureID = wf.FeatureID AND Channel = 'experimental'), (SELECT ARRAY(SELECT AS STRUCT '' BrowserName, CAST(0.0 AS NUMERIC) PassRate))) AS ExperimentalMetrics
FROM WebFeatures wf
LEFT OUTER JOIN FeatureBaselineStatus fbs ON wf.FeatureID = fbs.FeatureID
`
}

const latestRunsByChannelAndBrowserQuery = `
SELECT
    Channel,
    BrowserName,
    MAX(TimeStart) AS TimeStart
FROM WPTRuns
GROUP BY BrowserName, Channel;
`

type LatestRunResult struct {
	Channel     string    `spanner:"Channel"`
	BrowserName string    `spanner:"BrowserName"`
	TimeStart   time.Time `spanner:"TimeStart"`
}

type LatestRunResultsGroupedByChannel map[string][]LatestRunResult

func (c *Client) GetLatestRunResultGroupedByChannel(
	ctx context.Context,
	txn *spanner.ReadOnlyTransaction,
) (LatestRunResultsGroupedByChannel, error) {
	stmt := spanner.NewStatement(latestRunsByChannelAndBrowserQuery)
	it := txn.Query(ctx, stmt)
	defer it.Stop()

	ret := make(LatestRunResultsGroupedByChannel)
	for {
		row, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			// Catch-all for other errors.
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var latestRunResult LatestRunResult
		if err := row.ToStruct(&latestRunResult); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}

		var value []LatestRunResult
		var found bool
		if value, found = ret[latestRunResult.Channel]; !found {
			value = []LatestRunResult{}
		}
		value = append(value, latestRunResult)
		ret[latestRunResult.Channel] = value
	}

	return ret, nil
}
