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
	"maps"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// FeatureSearchBaseQuery contains the base query for all feature search
// related queries.
type FeatureSearchBaseQuery interface {
	// Prefilter does any necessary queries to generate useful information for
	// the query to help expedite it.
	Prefilter(
		ctx context.Context,
		txn *spanner.ReadOnlyTransaction) (FeatureSearchPrefilterResult, error)
	// Query generates a query to return rows about the features in the system.
	// Each row includes:
	//  1. The Internal ID of the feature
	//  2. The external ID from web features repo
	//  3. The human readable name.
	//  4. The baseline status.
	//  5. The latest metrics from WPT.
	//     It provides these metrics for both "stable" and "experimental" channels.
	//     The metrics retrieved are for each unique BrowserName/Channel/FeatureID.
	Query(prefilter FeatureSearchPrefilterResult) (string, map[string]interface{})
}

type FeatureSearchPrefilterResult struct {
	stableParams       map[string]interface{}
	stableClause       string
	experimentalParams map[string]interface{}
	experimentalClause string
}

// GCPFeatureSearchBaseQuery provides a base query that is optimal for GCP Spanner to retrieve the information
// described in the FeatureBaseQuery interface.
type GCPFeatureSearchBaseQuery struct{}

func (f GCPFeatureSearchBaseQuery) Prefilter(
	ctx context.Context,
	txn *spanner.ReadOnlyTransaction) (FeatureSearchPrefilterResult, error) {
	results, err := f.getLatestRunResultGroupedByChannel(ctx, txn)
	if err != nil {
		return FeatureSearchPrefilterResult{}, err
	}
	stableClause, stableParams := f.buildChannelMetricsFilter("stable", results["stable"])
	experimentalClause, experimentalParams := f.buildChannelMetricsFilter("experimental", results["experimental"])

	return FeatureSearchPrefilterResult{
		stableParams:       stableParams,
		stableClause:       stableClause,
		experimentalParams: experimentalParams,
		experimentalClause: experimentalClause,
	}, nil
}

func (f GCPFeatureSearchBaseQuery) buildChannelMetricsFilter(
	channel string, latestRunResults []LatestRunResult) (string, map[string]interface{}) {
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

// LatestRunResult contains the information for when a given BrowserName & Channel combination last ran.
type LatestRunResult struct {
	Channel     string    `spanner:"Channel"`
	BrowserName string    `spanner:"BrowserName"`
	TimeStart   time.Time `spanner:"TimeStart"`
}

// LatestRunResultsGroupedByChannel is a mapping of channel to list LatestRunResult.
// Useful for building the filter per channel in the Query method of GCPFeatureSearchBaseQuery.
type LatestRunResultsGroupedByChannel map[string][]LatestRunResult

const latestRunsByChannelAndBrowserQuery = `
SELECT
    Channel,
    BrowserName,
    MAX(TimeStart) AS TimeStart
FROM WPTRuns
GROUP BY BrowserName, Channel;
`

// getLatestRunResultGroupedByChannel creates the needed information for the Query filter.
// It queries for the last start time for a given BrowserName & Channel.
func (f GCPFeatureSearchBaseQuery) getLatestRunResultGroupedByChannel(
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
			return nil, err
		}
		var latestRunResult LatestRunResult
		if err := row.ToStruct(&latestRunResult); err != nil {
			return nil, err
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

// Query uses the latest browsername/channel/timestart mapping to build a query from the prefilter query.
// This prevents an extra join to figure out the latest run for a particular.
// The one thing to note about to this implementation: If the latest run ever deprecates a feature,
// it will not be included in the query. However, a feature can only be deprecated by a bigger change in the ecosystem
// and is not a common thing and will have bigger changes outside of this repository than just here.
func (f GCPFeatureSearchBaseQuery) Query(prefilter FeatureSearchPrefilterResult) (string, map[string]interface{}) {
	params := make(map[string]interface{}, len(prefilter.stableParams)+len(prefilter.experimentalParams))
	maps.Copy(params, prefilter.stableParams)
	maps.Copy(params, prefilter.experimentalParams)

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
`, prefilter.stableClause, prefilter.experimentalClause), params
}

// LocalFeatureBaseQuery is a version of the base query that works well on the local emulator.
// For some reason, the local emulator takes forever for the GCPFeatureSearchBaseQuery as the number
// of features and metrics grows. But GCPFeatureSearchBaseQuery works extremely well on GCP.
// At least 1.5-2x faster than LocalFeatureBaseQuery with 2400 metrics as of March 2024.
// Rather han sacrifice performance for the sake of compatibility, we have this LocalFeatureBaseQuery implementation
// which is good for the volume of data locally.
// TODO. Consolidate to using either LocalFeatureBaseQuery to reduce the maintenance burden.
type LocalFeatureBaseQuery struct{}

// Prefilter not used in LocalFeatureBaseQuery.
func (f LocalFeatureBaseQuery) Prefilter(
	_ context.Context,
	_ *spanner.ReadOnlyTransaction) (FeatureSearchPrefilterResult, error) {
	return FeatureSearchPrefilterResult{
		stableParams:       nil,
		stableClause:       "",
		experimentalParams: nil,
		experimentalClause: "",
	}, nil
}

// Query is a version of the base query that works on the local emulator.
// It leverages a common table expression CTE to help query the metrics.
func (f LocalFeatureBaseQuery) Query(_ FeatureSearchPrefilterResult) (string, map[string]interface{}) {
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
`, nil
}
