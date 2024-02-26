package gcpspanner

import "text/template"

type OverviewQuery struct{}

func (q OverviewQuery) Template() {
	template.New("").Parse(`

`)
}

var overviewResultsQueryTmpl = template.Must(template.New("overviewResultsSQL").Parse(`
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
WHERE
{{if .HasAvailabilityFilter}}
wf.FeatureID IN (SELECT FeatureID FROM BrowserFeatureAvailabilities WHERE BrowserName IN UNNEST(@availableBrowsers)) AND
{{end}}
ORDER BY wf.Name ASC -- Default sorting
`))
