package gcpspanner

import (
	"cloud.google.com/go/spanner"
)

type Filterable interface {
	Params() map[string]interface{}
	Clause() string
}

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

type OverviewQueryBuilder struct{}

func (q OverviewQueryBuilder) Base() string {
	return `
	SELECT
    wf.ID,
    wf.FeatureID,
    wf.Name,
    fbs.Status,
    wpfm.TotalTests,
    wpfm.TestPass
FROM WebFeatures wf
JOIN FeatureBaselineStatus fbs ON wf.FeatureID = fbs.FeatureID
LEFT JOIN (
    SELECT
        FeatureID,
        TotalTests,
        TestPass,
        MAX(TimeStart) AS LatestTimeStart
    FROM WPTRuns r
    JOIN WPTRunFeatureMetrics wpfm ON r.ExternalRunID = wpfm.ExternalRunID
    WHERE r.Channel = 'stable'
    GROUP BY FeatureID, TotalTests, TestPass
) wpfm ON wf.FeatureID = wpfm.FeatureID
`
}

func (q OverviewQueryBuilder) Order() string {
	return "ORDER BY wf.Name ASC"
}

func (q OverviewQueryBuilder) Build(filters ...Filterable) spanner.Statement {
	filterQuery := ""

	filterParams := make(map[string]interface{})
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
	stmt := spanner.NewStatement(q.Base() + " " + filterQuery + " " + q.Order())

	stmt.Params = filterParams

	return stmt
}
