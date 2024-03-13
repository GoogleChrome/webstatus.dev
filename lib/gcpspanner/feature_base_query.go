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
type FeatureBaseQuery struct{}

// Query providess the basic information for a feature.
// It provides:
//  1. The Internal ID of the feature
//  2. The external ID from web features repo
//  3. The human readable name.
//  4. The baseline status.
//  5. The latest metrics from WPT.
//     It provides these metrics for both "stable" and "experimental" channels.
//     The metrics retrieved are for each unique BrowserName/Channel/FeatureID.
//
// Note about the metrics calculations:
// The metrics columns need to be wrapped in TO_JSON. As a result, the metrics
// need to be parsed. More details about it in the TODO below.
// TODO: Fix https://github.com/GoogleChrome/webstatus.dev/issues/77
func (f FeatureBaseQuery) Query() string {
	return `
SELECT
	wf.ID,
	wf.FeatureID,
	wf.Name,
	COALESCE(fbs.Status, 'undefined') AS Status,

    -- StableMetrics Calculation
    (SELECT TO_JSON(ARRAY_AGG(STRUCT(metrics.BrowserName, CAST(PassRate AS FLOAT64) AS PassRate)))
        FROM WPTRunFeatureMetrics metrics
        JOIN (
              SELECT FeatureID, Channel, BrowserName, MAX(TimeStart) AS MostRecentTimeStart
              FROM WPTRunFeatureMetrics
              GROUP BY FeatureID, Channel, BrowserName
        ) latest_runs ON
            metrics.FeatureID = latest_runs.FeatureID
            AND metrics.Channel = latest_runs.Channel
            AND metrics.BrowserName = latest_runs.BrowserName
            AND metrics.TimeStart = latest_runs.MostRecentTimeStart
	WHERE metrics.FeatureID = wf.FeatureID AND metrics.Channel = 'stable') AS StableMetrics,

    -- ExperimentalMetrics Calculation
    (SELECT TO_JSON(ARRAY_AGG(STRUCT(metrics.BrowserName, CAST(PassRate AS FLOAT64) AS PassRate)))
        FROM WPTRunFeatureMetrics metrics
        JOIN (
              SELECT FeatureID, Channel, BrowserName, MAX(TimeStart) AS MostRecentTimeStart
              FROM WPTRunFeatureMetrics
              GROUP BY FeatureID, Channel, BrowserName
        ) latest_runs ON
            metrics.FeatureID = latest_runs.FeatureID
            AND metrics.Channel = latest_runs.Channel
            AND metrics.BrowserName = latest_runs.BrowserName
            AND metrics.TimeStart = latest_runs.MostRecentTimeStart
        WHERE metrics.FeatureID = wf.FeatureID AND metrics.Channel = 'experimental') AS ExperimentalMetrics

FROM WebFeatures wf
LEFT OUTER JOIN FeatureBaselineStatus fbs ON wf.FeatureID = fbs.FeatureID
`
}
