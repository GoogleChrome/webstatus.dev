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
	"cloud.google.com/go/spanner"
)

// Filterable modifies a query with a given filter.
type Filterable interface {
	Params() map[string]interface{}
	Clause() string
}

func NewAvailabileFilter(availableBrowsers []string) *AvailabileFilter {
	return &AvailabileFilter{availableBrowsers: availableBrowsers}
}

// AvailabileFilter applies a filter to limit the features based on their availability in a list of browsers.
type AvailabileFilter struct {
	availableBrowsers []string
}

func (f AvailabileFilter) Clause() string {
	return `wf.FeatureID IN (SELECT FeatureID FROM BrowserFeatureAvailabilities
		WHERE BrowserName IN UNNEST(@availableBrowsers))`
}

func (f AvailabileFilter) Params() map[string]interface{} {
	return map[string]interface{}{
		"availableBrowsers": f.availableBrowsers,
	}
}

func NewNotAvailabileFilter(notAvailableBrowsers []string) *NotAvailabileFilter {
	return &NotAvailabileFilter{notAvailableBrowsers: notAvailableBrowsers}
}

// NotAvailabileFilter applies a filter to limit the features based on their unavailability in a list of browsers.
type NotAvailabileFilter struct {
	notAvailableBrowsers []string
}

func (f NotAvailabileFilter) Clause() string {
	return `wf.FeatureID NOT IN (SELECT FeatureID FROM BrowserFeatureAvailabilities
		WHERE BrowserName IN UNNEST(@notAvailableBrowsers))`
}

func (f NotAvailabileFilter) Params() map[string]interface{} {
	return map[string]interface{}{
		"notAvailableBrowsers": f.notAvailableBrowsers,
	}
}

// FeatureSearchQueryBuilder builds a query to search for features.
type FeatureSearchQueryBuilder struct {
	cursor   *FeatureResultCursor
	pageSize int
}

// Base provides the minimum query to get data for the features search.
// This query retrieves the latest metrics for each unique BrowserName/FeatureID
// combination associated with a given feature.
//
// It provides these metrics for both "stable" and "experimental" channels.
// The results are designed to be used for the feature search and filtering.
func (q FeatureSearchQueryBuilder) Base() string {
	return `
SELECT
	wf.ID,
	wf.FeatureID,
	wf.Name,
	COALESCE(fbs.Status, 'undefined') AS Status,

	-- StableMetrics Calculation
	(SELECT ARRAY_AGG(STRUCT(BrowserName, TotalTests, TestPass))
		FROM (
		SELECT browser_feature_list.BrowserName, TotalTests, TestPass
		FROM (
			-- Subquery to get distinct BrowserName, FeatureID combinations and their
			-- associated maximum TimeStart for the specified FeatureID
			SELECT DISTINCT BrowserName, FeatureID, MAX(wpr.TimeStart) AS MaxTimeStart
			FROM WPTRunFeatureMetrics metrics
			JOIN WPTRuns wpr ON metrics.ExternalRunID = wpr.ExternalRunID
			WHERE metrics.FeatureID = wf.FeatureID
			GROUP BY BrowserName, FeatureID
		) browser_feature_list
		-- Join to retrieve metrics, ensuring we get the latest run for each combination
		JOIN WPTRunFeatureMetrics metrics ON browser_feature_list.FeatureID = metrics.FeatureID
		JOIN WPTRuns wpr ON metrics.ExternalRunID = wpr.ExternalRunID AND browser_feature_list.BrowserName = wpr.BrowserName
		WHERE wpr.Channel = 'stable'
		AND wpr.TimeStart = browser_feature_list.MaxTimeStart
	) latest_metric) AS StableMetrics,

	-- ExperimentalMetrics Calculation
	(SELECT ARRAY_AGG(STRUCT(BrowserName, TotalTests, TestPass))
		FROM (
		SELECT browser_feature_list.BrowserName, TotalTests, TestPass
		FROM (
			-- Subquery to get distinct BrowserName, FeatureID combinations and their
			-- associated maximum TimeStart for the specified FeatureID
			SELECT DISTINCT BrowserName, FeatureID, MAX(wpr.TimeStart) AS MaxTimeStart
			FROM WPTRunFeatureMetrics metrics
			JOIN WPTRuns wpr ON metrics.ExternalRunID = wpr.ExternalRunID
			WHERE metrics.FeatureID = wf.FeatureID
			GROUP BY BrowserName, FeatureID
		) browser_feature_list
		-- Join to retrieve metrics, ensuring we get the latest run for each combination
		JOIN WPTRunFeatureMetrics metrics ON browser_feature_list.FeatureID = metrics.FeatureID
		JOIN WPTRuns wpr ON metrics.ExternalRunID = wpr.ExternalRunID AND browser_feature_list.BrowserName = wpr.BrowserName
		WHERE wpr.Channel = 'experimental'
		AND wpr.TimeStart = browser_feature_list.MaxTimeStart
	) latest_metric) AS ExperimentalMetrics,

FROM WebFeatures wf
LEFT OUTER JOIN FeatureBaselineStatus fbs ON wf.FeatureID = fbs.FeatureID
`
}

func (q FeatureSearchQueryBuilder) Order() string {
	// Stable sorting
	return "ORDER BY wf.FeatureID"
}

func (q FeatureSearchQueryBuilder) Build(filters ...Filterable) spanner.Statement {
	filterQuery := ""

	filterParams := make(map[string]interface{})
	if q.cursor != nil {
		filterParams["cursorId"] = q.cursor.LastFeatureID
		filterQuery += " wf.FeatureID > @cursorId"
	}

	filterParams["pageSize"] = q.pageSize

	for _, filter := range filters {
		if len(filterQuery) > 0 {
			filterQuery += "AND "
		}
		filterQuery += filter.Clause() + " "
		for key, value := range filter.Params() {
			filterParams[key] = value
		}
	}
	if len(filterQuery) > 0 {
		filterQuery = "WHERE " + filterQuery
	}
	stmt := spanner.NewStatement(q.Base() + " " + filterQuery + " " + q.Order() + " LIMIT @pageSize")

	stmt.Params = filterParams

	return stmt
}
