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
	"fmt"
	"log/slog"
	"strings"
	"text/template"
	"time"
)

type WPTMetricView string

const (
	WPTSubtestView WPTMetricView = "subtest"
	WPTTestView    WPTMetricView = "test"
)

// nolint: gochecknoglobals // WONTFIX: thread safe globals.
// Have to make sure not to reassign them.
// These templates are compiled once at startup so that query building is fast per request.
var (
	// gcpFSMetricsSubQueryTemplate is the compiled version of gcpFSMetricsSubQueryRawTemplate.
	gcpFSMetricsSubQueryTemplate BaseQueryTemplate
	// gcpFSCountQueryTemplate is the compiled version of gcpFSCountQueryRawTemplate.
	gcpFSCountQueryTemplate BaseQueryTemplate
	// gcpFSSelectQueryTemplate is the compiled version of gcpFSSelectQueryRawTemplate.
	gcpFSSelectQueryTemplate BaseQueryTemplate
	// gcpFSPassRateForBrowserTemplate is the compiled version of gcpFSPassRateForBrowserRawTemplate.
	gcpFSPassRateForBrowserTemplate BaseQueryTemplate
	// gcpFSBrowserImplementationStatusTemplate is the compiled version of gcpFSBrowserImplementationStatusRawTemplate.
	gcpFSBrowserImplementationStatusTemplate BaseQueryTemplate
	// gcpFSBrowserFeatureSupportTemplate is the compiled version of gcpFSBrowserFeatureSupportRawTemplate.
	gcpFSBrowserFeatureSupportTemplate BaseQueryTemplate

	// localFSMetricsSubQueryTemplate is the compiled version of localFSMetricsSubQueryRawTemplate.
	localFSMetricsSubQueryTemplate BaseQueryTemplate
	// localFSCountQueryTemplate is the compiled version of localFSCountQueryRawTemplate.
	localFSCountQueryTemplate BaseQueryTemplate
	// localFSSelectQueryTemplate is the compiled version of localFSSelectQueryRawTemplate.
	localFSSelectQueryTemplate BaseQueryTemplate
	// localFSPassRateForBrowserTemplate is the compiled version of localFSPassRateForBrowserRawTemplate.
	localFSPassRateForBrowserTemplate BaseQueryTemplate
	// localFSBrowserImplementationStatusTemplate is the compiled version of localFSBrowserImplementationStatusRawTemplate.
	localFSBrowserImplementationStatusTemplate BaseQueryTemplate
	// localFSBrowserFeatureSupportTemplate is the compiled version of localFSBrowserFeatureSupportRawTemplate.
	localFSBrowserFeatureSupportTemplate BaseQueryTemplate
)

func init() {
	gcpFSMetricsSubQueryTemplate = NewQueryTemplate(gcpFSMetricsSubQueryRawTemplate)
	gcpFSCountQueryTemplate = NewQueryTemplate(gcpFSCountQueryRawTemplate)
	gcpFSSelectQueryTemplate = NewQueryTemplate(gcpFSSelectQueryRawTemplate)
	gcpFSPassRateForBrowserTemplate = NewQueryTemplate(gcpFSPassRateForBrowserRawTemplate)
	gcpFSBrowserImplementationStatusTemplate = NewQueryTemplate(gcpFSBrowserImplementationStatusRawTemplate)
	gcpFSBrowserFeatureSupportTemplate = NewQueryTemplate(gcpFSBrowserFeatureSupportRawTemplate)

	localFSMetricsSubQueryTemplate = NewQueryTemplate(localFSMetricsSubQueryRawTemplate)
	localFSCountQueryTemplate = NewQueryTemplate(localFSCountQueryRawTemplate)
	localFSSelectQueryTemplate = NewQueryTemplate(localFSSelectQueryRawTemplate)
	localFSPassRateForBrowserTemplate = NewQueryTemplate(localFSPassRateForBrowserRawTemplate)
	localFSBrowserImplementationStatusTemplate = NewQueryTemplate(localFSBrowserImplementationStatusRawTemplate)
	localFSBrowserFeatureSupportTemplate = NewQueryTemplate(localFSBrowserFeatureSupportRawTemplate)
}

type BaseQueryTemplate struct {
	tmpl *template.Template
}

func NewQueryTemplate(in string) BaseQueryTemplate {
	tmpl, err := template.New("").Parse(in)
	if err != nil {
		panic(err)
	}

	return BaseQueryTemplate{
		tmpl: tmpl,
	}
}

// TODO: Pass in context to be used by slog.ErrorContext.
func (t *BaseQueryTemplate) Execute(data any) string {
	var buf strings.Builder
	err := t.tmpl.Execute(&buf, data)
	if err != nil {
		slog.ErrorContext(context.TODO(), "unable to execute template", "error", err)

		return ""
	}

	return buf.String()
}

type CommonFSSelectTemplateData struct {
	BaseQueryFragment    string
	StableMetrics        string
	ExperimentalMetrics  string
	ImplementationStatus string
	PageFilters          []string
	Filters              []string
	SortClause           string
	Offset               int
	PageSize             int
	OptionalJoins        []JoinData
}

// JoinData contains template data for the optional joins.
type JoinData struct {
	Alias    string
	Template string
}

// GCPFSSelectTemplateData contains the template data for gcpFSSelectQueryTemplate.
type GCPFSSelectTemplateData struct {
	CommonFSSelectTemplateData
}

// LocalFSSelectTemplateData contains the template data for localFSSelectQueryTemplate.
type LocalFSSelectTemplateData struct {
	CommonFSSelectTemplateData
	PassRateColumn string
}

// GCPFSBrowserMetricTemplateData contains the template data for gcpFSPassRateForBrowserTemplate.
type GCPFSBrowserMetricTemplateData struct {
	BrowserNameParam string
	GCPFSMetricsTemplateData
}

// LocalFSBrowserMetricTemplateData contains the template data for localFSPassRateForBrowserTemplate.
type LocalFSBrowserMetricTemplateData struct {
	BrowserNameParam string
	LocalFSMetricsTemplateData
}

// GCPFSBrowserImplStatusTemplateData contains the template data for gcpFSBrowserImplementationStatusTemplate.
type GCPFSBrowserImplStatusTemplateData struct {
	BrowserNameParam string
}

// LocalFSBrowserImplStatusTemplateData contains the template data for localFSBrowserImplementationStatusTemplate.
type LocalFSBrowserImplStatusTemplateData struct {
	BrowserNameParam string
}

// GCPFSBrowserFeatureSupportTemplateData contains the template data for gcpFSBrowserFeatureSupportTemplate.
type GCPFSBrowserFeatureSupportTemplateData struct {
	BrowserNameParam string
}

// LocalFSBrowserFeatureSupportTemplateData contains the template data for localFSBrowserFeatureSupportTemplate.
type LocalFSBrowserFeatureSupportTemplateData struct {
	BrowserNameParam string
}

// GCPFSMetricsTemplateData contains the template data for gcpFSMetricsSubQueryTemplate.
type GCPFSMetricsTemplateData struct {
	Channel        string
	BrowserList    []string
	PassRateColumn string
	ChannelParam   string
	MetricIndex    string
}

// LocalFSMetricsTemplateData contains the template for localFSMetricsSubQueryTemplate.
type LocalFSMetricsTemplateData struct {
	Channel        string
	PassRateColumn string
	ChannelParam   string
}

// CommonFSCountTemplateData contains the template for commonCountQueryTemplate.
type CommonFSCountTemplateData struct {
	BaseQueryFragment string
	Filters           []string
}

// GCPFSCountTemplateData contains the template for gcpFSCountQueryTemplate.
type GCPFSCountTemplateData struct {
	CommonFSCountTemplateData
}

// LocalFSCountTemplateData contains the template for localFSCountQueryTemplate.
type LocalFSCountTemplateData struct {
	CommonFSCountTemplateData
}

// Helper function to determine the correct PassRate column name.
func metricsPassRateColumn(metricView WPTMetricView) string {
	switch metricView {
	case WPTSubtestView:
		return "SubtestPassRate"
	case WPTTestView:
		return "TestPassRate"
	}

	return "SubtestPassRate"
}

// Helper function to determine the correct PassRate index.
func metricsPassRateIndex(metricView WPTMetricView) string {
	switch metricView {
	case WPTSubtestView:
		return "MetricsFeatureChannelBrowserTimeSubtestPassRate"
	case WPTTestView:
		return "MetricsFeatureChannelBrowserTimeTestPassRate"
	}

	return "MetricsFeatureChannelBrowserTimeSubtestPassRate"
}

type FeatureSearchCountArgs struct {
	Filters []string
}

// SortByBrowserImplDetails contains parameter data for the Implementation Status templates.
type SortByBrowserImplDetails struct {
	BrowserName string
}

// SortByBrowserFeatureSupportDetails contains parameter data for the Browser Feature Support templates.
type SortByBrowserFeatureSupportDetails struct {
	BrowserName string
}

type FeatureSearchQueryArgs struct {
	MetricView                  WPTMetricView
	Filters                     []string
	PageFilters                 []string
	PageSize                    int
	Offset                      int
	SortClause                  string
	SortByStableBrowserImpl     *SortByBrowserImplDetails
	SortByExpBrowserImpl        *SortByBrowserImplDetails
	SortByBrowserFeatureSupport *SortByBrowserFeatureSupportDetails
	Browsers                    []string
}

// FeatureSearchBaseQuery contains the base query for all feature search
// related queries.
type FeatureSearchBaseQuery interface {
	// Query generates a query to return rows about the features in the system.
	// Each row includes:
	//  1. The Internal ID of the feature
	//  2. The external ID from web features repo
	//  3. The human readable name.
	//  4. The baseline status.
	//  5. The latest metrics from WPT.
	//     It provides these metrics for both "stable" and "experimental" channels.
	//     The metrics retrieved are for each unique BrowserName/Channel/WebFeatureID.
	Query(args FeatureSearchQueryArgs) (string, map[string]interface{})

	// CountQuery generates the base query to return only the count of items.
	CountQuery(args FeatureSearchCountArgs) string
}

// GCPFeatureSearchBaseQuery provides a base query that is optimal for GCP Spanner to retrieve the information
// described in the FeatureBaseQuery interface.
type GCPFeatureSearchBaseQuery struct{}

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
	var filterStr string
	var retParams map[string]interface{}
	if len(filters) > 0 {
		filterStr = strings.Join(filters, " OR ")
		filterStr = " AND (" + filterStr + ")"
		retParams = params
	}

	return filterStr, retParams
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

func (f GCPFeatureSearchBaseQuery) buildBaseQueryFragment() string { return gcpFSBaseQueryTemplate }

func (f GCPFeatureSearchBaseQuery) CountQuery(args FeatureSearchCountArgs) string {
	return gcpFSCountQueryTemplate.Execute(GCPFSCountTemplateData{
		CommonFSCountTemplateData: CommonFSCountTemplateData{
			BaseQueryFragment: f.buildBaseQueryFragment(),
			Filters:           args.Filters,
		},
	})
}

const (
	// commonFSBaseQueryTemplate provides the core of a Spanner query, joining
	// the WebFeatures table with FeatureBaselineStatus for status information.
	commonFSBaseQueryTemplate = `
FROM WebFeatures wf
LEFT OUTER JOIN FeatureBaselineStatus fbs ON wf.ID = fbs.WebFeatureID
LEFT OUTER JOIN ExcludedFeatureKeys efk ON wf.FeatureKey = efk.FeatureKey
LEFT OUTER JOIN FeatureSpecs fs ON wf.ID = fs.WebFeatureID
LEFT OUTER JOIN (
	SELECT
		bfa.WebFeatureID,
		ARRAY_AGG(
			STRUCT(
				bfa.BrowserName AS BrowserName,
				bfa.BrowserVersion AS ImplementationVersion,
				IF(br.ReleaseDate IS NULL, 'unavailable', 'available') AS ImplementationStatus,
				br.ReleaseDate AS ImplementationDate
			)
		) AS BrowserInfo
	FROM BrowserFeatureAvailabilities bfa
	LEFT OUTER JOIN BrowserReleases br
		ON bfa.BrowserName = br.BrowserName AND bfa.BrowserVersion = br.BrowserVersion
	GROUP BY bfa.WebFeatureID
) AS browser_info ON wf.ID = browser_info.WebFeatureID
LEFT OUTER JOIN (
    SELECT
        ldchm.WebFeatureID,
        dchm.Rate AS ChromiumUsage
    FROM LatestDailyChromiumHistogramMetrics ldchm
    JOIN DailyChromiumHistogramMetrics dchm
	ON ldchm.ChromiumHistogramEnumValueID = dchm.ChromiumHistogramEnumValueID
	AND ldchm.Day = dchm.Day
) AS chromium_usage_metrics ON wf.ID = chromium_usage_metrics.WebFeatureID
`
	gcpFSBaseQueryTemplate   = commonFSBaseQueryTemplate
	localFSBaseQueryTemplate = commonFSBaseQueryTemplate

	// commonFSImplementationStatusRawTemplate returns an array of structs that represent the implementation status.
	// GCP Spanner doesn't support directly returning a NULL-valued array of structs.
	// If browser_info.BrowserInfo is NULL (no browser data available for the feature),
	// we need to provide a default non-NULL array with a specific structure to avoid errors.
	// https://github.com/GoogleChrome/webstatus.dev/issues/576
	commonFSImplementationStatusRawTemplate = `
COALESCE(
	browser_info.BrowserInfo,
	(
		SELECT ARRAY(
			SELECT AS STRUCT
				'' BrowserName,
				'' AS ImplementationVersion,
				'unavailable' AS ImplementationStatus,
				CAST(NULL AS TIMESTAMP) AS ImplementationDate,
		)
	)
) AS ImplementationStatuses
`
	gcpFSImplementationStatusRawTemplate   = commonFSImplementationStatusRawTemplate
	localFSImplementationStatusRawTemplate = commonFSImplementationStatusRawTemplate

	// commonFSBrowserImplementationStatusRawTemplate returns the implementation status for the feature of a given
	// browser.
	commonFSBrowserImplementationStatusRawTemplate = `
(
	SELECT COALESCE(
		(SELECT 'available'
			FROM BrowserFeatureAvailabilities bfa
			WHERE bfa.WebFeatureID = wf.ID
				AND BrowserName = @{{ .BrowserNameParam }}
			LIMIT 1),
		'unavailable' -- Default if no match
	) AS ImplementationStatus
) AS SortImplStatus
	`
	gcpFSBrowserImplementationStatusRawTemplate   = commonFSBrowserImplementationStatusRawTemplate
	localFSBrowserImplementationStatusRawTemplate = commonFSBrowserImplementationStatusRawTemplate

	// commonFSBrowserFeatureSupportRawTemplate returns the implementation status for the feature of a given
	// browser.
	commonFSBrowserFeatureSupportRawTemplate = `
(
    SELECT COALESCE(
        (SELECT br.ReleaseDate
            FROM BrowserFeatureAvailabilities bfa
            LEFT OUTER JOIN
				BrowserReleases br
			ON
				bfa.BrowserName = br.BrowserName
				AND bfa.BrowserVersion = br.BrowserVersion
            WHERE bfa.WebFeatureID = wf.ID
                AND bfa.BrowserName = @{{ .BrowserNameParam }}
            LIMIT 1),
        CAST('1900-01-01' AS TIMESTAMP) -- Default if no match
    ) AS SortDate
) AS SortDate
	`
	gcpFSBrowserFeatureSupportRawTemplate   = commonFSBrowserFeatureSupportRawTemplate
	localFSBrowserFeatureSupportRawTemplate = commonFSBrowserFeatureSupportRawTemplate

	// commonCountQueryRawTemplate returns the count of items, using the base query fragment
	// for consistency.
	commonCountQueryRawTemplate = `
SELECT COUNT(*)
{{ .BaseQueryFragment }}
WHERE 1=1
{{ range .Filters }}
	AND {{ . }}
{{ end }}
`
	gcpFSCountQueryRawTemplate   = commonCountQueryRawTemplate
	localFSCountQueryRawTemplate = commonCountQueryRawTemplate

	// gcpFSSelectQueryRawTemplate builds the core SELECT query. It retrieves feature
	// information, baseline status, and aggregated metrics.
	gcpFSSelectQueryRawTemplate = `
SELECT
	wf.ID,
	wf.FeatureKey,
	wf.Name,
	fbs.Status,
	fbs.LowDate,
	fbs.HighDate,
	fs.Links AS SpecLinks,
	chromium_usage_metrics.ChromiumUsage,
	{{ .StableMetrics }},
	{{ .ExperimentalMetrics }},
	{{ .ImplementationStatus }}
{{ .BaseQueryFragment }}
{{ if .OptionalJoins }}
	{{ range $index, $join := .OptionalJoins }}
LEFT OUTER JOIN (
    SELECT
        wf.ID,
		{{ $join.Template }}
   FROM WebFeatures wf
) {{ $join.Alias }} ON wf.ID = {{ $join.Alias }}.ID
	{{ end }}
{{ end }}
WHERE 1=1 -- This ensures valid syntax even with no filters
{{ range .PageFilters }}
	AND {{ . }}
{{ end }}
{{ range .Filters }}
	AND {{ . }}
{{ end }}
{{ if .SortClause }}
ORDER BY {{ .SortClause }}
{{ end }}
LIMIT {{ .PageSize }}
{{ if .Offset }}
OFFSET {{ .Offset }}
{{ end }}
`
	localFSSelectQueryRawTemplate = `
WITH
	LatestMetrics AS (
		SELECT
			WebFeatureID,
			Channel,
			BrowserName,
			MAX(TimeStart) AS LatestTimeStart
		FROM WPTRunFeatureMetrics
		GROUP BY WebFeatureID, Channel, BrowserName
	),
	MetricsAggregation AS (
		SELECT
			lm.WebFeatureID,
			lm.Channel,
			lm.BrowserName,
			m.{{ .PassRateColumn }},
			FeatureRunDetails
		FROM LatestMetrics lm
		JOIN WPTRunFeatureMetrics m ON
			lm.WebFeatureID = m.WebFeatureID AND
			lm.Channel = m.Channel AND
			lm.BrowserName = m.BrowserName AND
			lm.LatestTimeStart = m.TimeStart
	)
SELECT
	wf.ID,
	wf.FeatureKey,
	wf.Name,
	fbs.Status,
	fbs.LowDate,
	fbs.HighDate,
	fs.Links AS SpecLinks,
	chromium_usage_metrics.ChromiumUsage,
	{{ .StableMetrics }},
	{{ .ExperimentalMetrics }},
	{{ .ImplementationStatus }}
{{ .BaseQueryFragment }}
{{ if .OptionalJoins }}
	{{ range $index, $join := .OptionalJoins }}
LEFT OUTER JOIN (
    SELECT
        wf.ID,
		{{ $join.Template }}
   FROM WebFeatures wf
) {{ $join.Alias }} ON wf.ID = {{ $join.Alias }}.ID
	{{ end }}
{{ end }}
WHERE 1=1 -- This ensures valid syntax even with no filters
{{ range .PageFilters }}
	AND {{ . }}
{{ end }}
{{ range .Filters }}
	AND {{ . }}
{{ end }}
{{ if .SortClause }}
ORDER BY {{ .SortClause }}
{{ end }}
LIMIT {{ .PageSize }}
{{ if .Offset }}
OFFSET {{ .Offset }}
{{ end }}
`
	// gcpFSPassRateForBrowserRawTemplate generates a nested query that gets the pass rate for a particular
	// browser for the examined feature.
	// nolint: gosec // WONTFIX: false positive.
	gcpFSPassRateForBrowserRawTemplate = `
(
	SELECT {{ .PassRateColumn }} AS PassRate
		FROM WPTRunFeatureMetrics @{FORCE_INDEX={{ .MetricIndex }}} metrics
		WHERE metrics.WebFeatureID = wf.ID
			AND metrics.Channel = @{{ .ChannelParam }}
			AND metrics.BrowserName = @{{ .BrowserNameParam }}
			AND metrics.TimeStart = (
				SELECT MAX(TimeStart)
				FROM WPTRunFeatureMetrics metrics2
				WHERE metrics2.WebFeatureID = wf.ID
					AND metrics2.Channel = @{{ .ChannelParam }}
					AND metrics2.BrowserName = @{{ .BrowserNameParam }}
			)
) AS SortMetric
`

	// gcpFSMetricsSubQueryRawTemplate generates a nested query that aggregates metrics by browser and
	// channel. It uses COALESCE to handle cases with no matching metrics.
	gcpFSMetricsSubQueryRawTemplate = `
COALESCE(
	(
		SELECT ARRAY_AGG(
			STRUCT(
				latest.BrowserName AS BrowserName,
				wpfm.{{ $.PassRateColumn }} AS PassRate,
				wpfm.FeatureRunDetails AS FeatureRunDetails
			)
		)
		FROM LatestWPTRunFeatureMetrics latest
		JOIN WPTRunFeatureMetrics wpfm ON latest.RunMetricID = wpfm.ID
			AND latest.WebFeatureID = wpfm.WebFeatureID
			AND latest.WebFeatureID = wf.ID
		WHERE latest.Channel = @{{ $.ChannelParam }}
			AND latest.BrowserName IN UNNEST(@browserNames)
	),
	(
		SELECT ARRAY(
			SELECT AS STRUCT
			'' BrowserName,
			CAST(0.0 AS NUMERIC) PassRate,
			CAST(NULL AS JSON) FeatureRunDetails
		)
	)
) AS {{ $.Channel }}Metrics
`

	// localFSMetricsSubQueryRawTemplate generates a nested query that aggregates metrics by browser and
	// channel. It uses COALESCE to handle cases with no matching metrics.
	localFSMetricsSubQueryRawTemplate = `
COALESCE(
	(
		SELECT ARRAY_AGG(
			STRUCT(
				BrowserName,
				{{ .PassRateColumn }} AS PassRate,
				FeatureRunDetails
			)
		)
		FROM MetricsAggregation WHERE WebFeatureID = wf.ID AND Channel = @{{ .ChannelParam }}
	),
	(
		SELECT ARRAY(
			SELECT AS STRUCT
				'' BrowserName,
				CAST(0.0 AS NUMERIC) PassRate,
				CAST(NULL AS JSON) FeatureRunDetails
		)
	)
) AS {{ .Channel }}Metrics
`

	// localFSPassRateForBrowserRawTemplate generates a nested query that gets the pass rate for a particular
	// browser for the examined feature.
	// nolint: gosec // WONTFIX: false positive.
	localFSPassRateForBrowserRawTemplate = `
(
	SELECT {{ .PassRateColumn }} AS PassRate
		FROM MetricsAggregation
		WHERE WebFeatureID = wf.ID
			AND Channel = @{{ .ChannelParam }}
			AND BrowserName = @{{ .BrowserNameParam }}
		LIMIT 1
) AS SortMetric
`
)

// Query uses the latest browsername/channel/timestart mapping to build a query.
func (f GCPFeatureSearchBaseQuery) Query(args FeatureSearchQueryArgs) (
	string, map[string]interface{}) {
	params := make(map[string]interface{})
	stableParamName := "stableChannelParam"
	params[stableParamName] = "stable"
	experimentalParamName := "experimentalChannelParam"
	params[experimentalParamName] = "experimental"

	params["browserNames"] = args.Browsers

	stableMetricsData := GCPFSMetricsTemplateData{
		Channel:        "Stable",
		BrowserList:    args.Browsers,
		PassRateColumn: metricsPassRateColumn(args.MetricView),
		MetricIndex:    metricsPassRateIndex(args.MetricView),
		ChannelParam:   stableParamName,
	}
	stableMetrics := gcpFSMetricsSubQueryTemplate.Execute(stableMetricsData)

	experimentalMetricsData := GCPFSMetricsTemplateData{
		Channel:        "Experimental",
		BrowserList:    args.Browsers,
		PassRateColumn: metricsPassRateColumn(args.MetricView),
		MetricIndex:    metricsPassRateIndex(args.MetricView),
		ChannelParam:   experimentalParamName,
	}
	experimentalMetrics := gcpFSMetricsSubQueryTemplate.Execute(experimentalMetricsData)

	var optionalJoins []JoinData
	if args.SortByBrowserFeatureSupport != nil {
		browserNameParamName := "sortBrowserFeatureSupportParam"
		params[browserNameParamName] = args.SortByBrowserFeatureSupport.BrowserName
		optionalJoins = append(optionalJoins, JoinData{
			Template: gcpFSBrowserFeatureSupportTemplate.Execute(
				GCPFSBrowserFeatureSupportTemplateData{
					BrowserNameParam: browserNameParamName,
				}),
			Alias: derivedTableSortBrowserFeatureSupport,
		})
	} else if args.SortByStableBrowserImpl != nil {
		browserNameParamName := "sortStableBrowserNameMetricParam"
		params[browserNameParamName] = args.SortByStableBrowserImpl.BrowserName
		optionalJoins = append(optionalJoins, JoinData{
			Template: gcpFSPassRateForBrowserTemplate.Execute(
				GCPFSBrowserMetricTemplateData{
					BrowserNameParam:         browserNameParamName,
					GCPFSMetricsTemplateData: stableMetricsData,
				}),
			Alias: derviedTableSortMetrics,
		})
		optionalJoins = append(optionalJoins, JoinData{
			Template: gcpFSBrowserImplementationStatusTemplate.Execute(
				GCPFSBrowserImplStatusTemplateData{
					BrowserNameParam: browserNameParamName,
				}),
			Alias: derviedTableSortImpl,
		})
	} else if args.SortByExpBrowserImpl != nil {
		browserNameParamName := "sortExpBrowserNameMetricParam"
		params[browserNameParamName] = args.SortByExpBrowserImpl.BrowserName
		optionalJoins = append(optionalJoins, JoinData{
			Template: gcpFSPassRateForBrowserTemplate.Execute(
				GCPFSBrowserMetricTemplateData{
					BrowserNameParam:         browserNameParamName,
					GCPFSMetricsTemplateData: experimentalMetricsData,
				}),
			Alias: derviedTableSortMetrics,
		})
		optionalJoins = append(optionalJoins, JoinData{
			Template: gcpFSBrowserImplementationStatusTemplate.Execute(
				GCPFSBrowserImplStatusTemplateData{
					BrowserNameParam: browserNameParamName,
				}),
			Alias: derviedTableSortImpl,
		})
	}

	return gcpFSSelectQueryTemplate.Execute(GCPFSSelectTemplateData{
		CommonFSSelectTemplateData: CommonFSSelectTemplateData{
			BaseQueryFragment:    f.buildBaseQueryFragment(),
			StableMetrics:        stableMetrics,
			ExperimentalMetrics:  experimentalMetrics,
			ImplementationStatus: gcpFSImplementationStatusRawTemplate,
			Filters:              args.Filters,
			PageFilters:          args.PageFilters,
			Offset:               args.Offset,
			SortClause:           args.SortClause,
			PageSize:             args.PageSize,
			OptionalJoins:        optionalJoins,
		},
	}), params
}

// LocalFeatureBaseQuery is a version of the base query that works well on the local emulator.
// For some reason, the local emulator takes forever for the GCPFeatureSearchBaseQuery as the number
// of features and metrics grows. But GCPFeatureSearchBaseQuery works extremely well on GCP.
// At least 1.5-2x faster than LocalFeatureBaseQuery with 2400 metrics as of March 2024.
// Rather han sacrifice performance for the sake of compatibility, we have this LocalFeatureBaseQuery implementation
// which is good for the volume of data locally.
// TODO. Consolidate to using either LocalFeatureBaseQuery to reduce the maintenance burden.
type LocalFeatureBaseQuery struct{}

func (f LocalFeatureBaseQuery) buildBaseQueryFragment() string { return localFSBaseQueryTemplate }

func (f LocalFeatureBaseQuery) CountQuery(args FeatureSearchCountArgs) string {
	return localFSCountQueryTemplate.Execute(LocalFSCountTemplateData{
		CommonFSCountTemplateData: CommonFSCountTemplateData{
			BaseQueryFragment: f.buildBaseQueryFragment(),
			Filters:           args.Filters,
		},
	})
}

// Query is a version of the base query that works on the local emulator.
// It leverages a common table expression CTE to help query the metrics.
func (f LocalFeatureBaseQuery) Query(args FeatureSearchQueryArgs) (
	string, map[string]interface{}) {
	stableParamName := "stableChannelParam"
	experimentalParamName := "experimentalChannelParam"

	params := map[string]interface{}{
		stableParamName:       "stable",
		experimentalParamName: "experimental",
	}

	stableMetricsData := LocalFSMetricsTemplateData{
		Channel:        "Stable",
		PassRateColumn: metricsPassRateColumn(args.MetricView),
		ChannelParam:   stableParamName,
	}
	stableMetrics := localFSMetricsSubQueryTemplate.Execute(stableMetricsData)

	experimentalMetricsData := LocalFSMetricsTemplateData{
		Channel:        "Experimental",
		PassRateColumn: metricsPassRateColumn(args.MetricView),
		ChannelParam:   experimentalParamName,
	}
	experimentalMetrics := localFSMetricsSubQueryTemplate.Execute(experimentalMetricsData)

	var optionalJoins []JoinData
	if args.SortByStableBrowserImpl != nil {
		browserNameParamName := "sortStableBrowserNameMetricParam"
		params[browserNameParamName] = args.SortByStableBrowserImpl.BrowserName
		optionalJoins = append(optionalJoins, JoinData{
			Template: localFSPassRateForBrowserTemplate.Execute(
				LocalFSBrowserMetricTemplateData{
					BrowserNameParam:           browserNameParamName,
					LocalFSMetricsTemplateData: stableMetricsData,
				}),
			Alias: derviedTableSortMetrics,
		})
		optionalJoins = append(optionalJoins, JoinData{
			Template: localFSBrowserImplementationStatusTemplate.Execute(
				LocalFSBrowserImplStatusTemplateData{
					BrowserNameParam: browserNameParamName,
				}),
			Alias: derviedTableSortImpl,
		})
	} else if args.SortByExpBrowserImpl != nil {
		browserNameParamName := "sortExpBrowserNameMetricParam"
		params[browserNameParamName] = args.SortByExpBrowserImpl.BrowserName
		optionalJoins = append(optionalJoins, JoinData{
			Template: localFSPassRateForBrowserTemplate.Execute(
				LocalFSBrowserMetricTemplateData{
					BrowserNameParam:           browserNameParamName,
					LocalFSMetricsTemplateData: experimentalMetricsData,
				}),
			Alias: derviedTableSortMetrics,
		})
		optionalJoins = append(optionalJoins, JoinData{
			Template: localFSBrowserImplementationStatusTemplate.Execute(
				LocalFSBrowserImplStatusTemplateData{
					BrowserNameParam: browserNameParamName,
				}),
			Alias: derviedTableSortImpl,
		})
	} else if args.SortByBrowserFeatureSupport != nil {
		browserNameParamName := "sortBrowserFeatureSupportParam"
		params[browserNameParamName] = args.SortByBrowserFeatureSupport.BrowserName
		optionalJoins = append(optionalJoins, JoinData{
			Template: localFSBrowserFeatureSupportTemplate.Execute(
				LocalFSBrowserFeatureSupportTemplateData{
					BrowserNameParam: browserNameParamName,
				}),
			Alias: derivedTableSortBrowserFeatureSupport,
		})
	}

	return localFSSelectQueryTemplate.Execute(LocalFSSelectTemplateData{
		PassRateColumn: metricsPassRateColumn(args.MetricView),
		CommonFSSelectTemplateData: CommonFSSelectTemplateData{
			BaseQueryFragment:    f.buildBaseQueryFragment(),
			StableMetrics:        stableMetrics,
			ExperimentalMetrics:  experimentalMetrics,
			ImplementationStatus: localFSImplementationStatusRawTemplate,
			PageFilters:          args.PageFilters,
			Filters:              args.Filters,
			SortClause:           args.SortClause,
			Offset:               args.Offset,
			PageSize:             args.PageSize,
			OptionalJoins:        optionalJoins,
		},
	}), params
}
