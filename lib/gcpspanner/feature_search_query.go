package gcpspanner

import (
	"cloud.google.com/go/spanner"
)

// Filterable modifies a query with a given filter.
type Filterable interface {
	Params() map[string]interface{}
	Clause() string
}

// AvailabileFilter applies a filter to limit the features based on their availability in a list of browsers.
type AvailabileFilter struct {
	availableBrowsers []string
}

func (f AvailabileFilter) Clause() string {
	return "wf.FeatureID IN (SELECT FeatureID FROM BrowserFeatureAvailabilities WHERE BrowserName IN UNNEST(@availableBrowsers))"
}

func (f AvailabileFilter) Params() map[string]interface{} {
	return map[string]interface{}{
		"availableBrowsers": f.availableBrowsers,
	}
}

// NotAvailabileFilter applies a filter to limit the features based on their unavailability in a list of browsers.
type NotAvailabileFilter struct {
	notAvailableBrowsers []string
}

func (f NotAvailabileFilter) Clause() string {
	return "wf.FeatureID NOT IN (SELECT FeatureID FROM BrowserFeatureAvailabilities WHERE BrowserName IN UNNEST(@notAvailableBrowsers))"
}

func (f NotAvailabileFilter) Params() map[string]interface{} {
	return map[string]interface{}{
		"notAvailableBrowsers": f.notAvailableBrowsers,
	}
}

// FeatureSearchQueryBuilder
type FeatureSearchQueryBuilder struct {
	cursorID *string
	pageSize int
}

func (q FeatureSearchQueryBuilder) Base() string {
	return `
SELECT
    wf.ID,
    wf.FeatureID,
    wf.Name,
    (SELECT ARRAY_AGG(STRUCT(BrowserName, TotalTests, TestPass))
		FROM (
		SELECT BrowserName, TotalTests, TestPass
		FROM WPTRunFeatureMetrics metrics
		JOIN WPTRuns wpr ON metrics.ExternalRunID = wpr.ExternalRunID
		WHERE metrics.FeatureID = wf.FeatureID AND wpr.Channel = 'stable'
		AND metrics.ID = (
				SELECT ARRAY_AGG(wpr2.ID)[OFFSET(0)]
				FROM WPTRunFeatureMetrics wpfm
				JOIN WPTRuns wpr2 ON wpfm.ExternalRunID = wpr2.ExternalRunID
				WHERE wpfm.FeatureID = wf.FeatureID
				AND wpr2.Channel = 'stable'
       	)
	) latest_metric) AS StableMetrics,
    (SELECT ARRAY_AGG(STRUCT(BrowserName, TotalTests, TestPass))
     FROM (
		SELECT BrowserName, TotalTests, TestPass
		FROM WPTRunFeatureMetrics metrics
		JOIN WPTRuns wpr ON metrics.ExternalRunID = wpr.ExternalRunID
		WHERE metrics.FeatureID = wf.FeatureID AND wpr.Channel = 'experimental'
		AND metrics.ID = (
				SELECT ARRAY_AGG(wpr2.ID)[OFFSET(0)]
				FROM WPTRunFeatureMetrics wpfm
				JOIN WPTRuns wpr2 ON wpfm.ExternalRunID = wpr2.ExternalRunID
				WHERE wpfm.FeatureID = wf.FeatureID
				AND wpr2.Channel = 'experimental'
		)
	) latest_metric) AS ExperimentalMetrics
FROM WebFeatures wf
JOIN FeatureBaselineStatus fbs ON wf.FeatureID = fbs.FeatureID
`
}

func (q FeatureSearchQueryBuilder) Order() string {
	// Stable sorting
	return "ORDER BY wf.ID"
}

func (q FeatureSearchQueryBuilder) Build(filters ...Filterable) spanner.Statement {
	filterQuery := ""

	filterParams := make(map[string]interface{})
	if q.cursorID != nil {
		filterParams["cursorId"] = *q.cursorID
	} else {
		filterParams["cursorId"] = nil
	}

	filterParams["pageSize"] = q.pageSize

	for idx, filter := range filters {
		filterQuery += filter.Clause() + " "
		if idx+1 < len(filters) {
			filterQuery += "AND "
		}
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
