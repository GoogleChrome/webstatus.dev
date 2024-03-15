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
func (f FeatureBaseQuery) Query() string {
	if f.useCTE {
		return f.cteQuery()
	}

	return f.nonCTEQuery()
}

// nonCTEQuery is the optimal query and is used on GCP Spanner.
// For some reason, the local emulator takes forever for this query as the number
// of features and metrics grows.
// This query is about 1.5-2x faster than the CTE version when used in GCP Spanner.
// Rather than sacrifice performance for the sake of compatibility, we will keep
// the nonCTEQuery.
func (f FeatureBaseQuery) nonCTEQuery() string {
	return `
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
		AND metrics.TimeStart IN (
			SELECT MAX(TimeStart)
			FROM WPTRunFeatureMetrics @{FORCE_INDEX=MetricsFeatureChannelBrowserTimePassRate} metrics2
			WHERE metrics2.FeatureID = wf.FeatureID
				AND metrics2.Channel = 'stable'
			GROUP BY BrowserName
		)
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
		AND metrics.TimeStart IN (
			SELECT MAX(TimeStart)
			FROM WPTRunFeatureMetrics @{FORCE_INDEX=MetricsFeatureChannelBrowserTimePassRate} metrics2
			WHERE metrics2.FeatureID = wf.FeatureID
				AND metrics2.Channel = 'experimental'
			GROUP BY BrowserName
		)
		-- GCP Spanner could have ARRAY<STRUCT<string, NUMERIC>>[]) as the default.
		-- but the emulator complains.
		-- Replace the following line in the future when the emulator supports it.
		-- ), ARRAY<STRUCT<string, NUMERIC>>[]) AS StableMetrics,
	), (SELECT ARRAY(SELECT AS STRUCT '' BrowserName, CAST(0.0 AS NUMERIC) PassRate))) AS ExperimentalMetrics
FROM WebFeatures wf
LEFT OUTER JOIN FeatureBaselineStatus fbs ON wf.FeatureID = fbs.FeatureID
`
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
