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
	"log/slog"
	"maps"
	"strings"
	"text/template"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
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
)

const (
	latestRunsByChannelAndBrowserQuery = `
SELECT
  Channel,
  BrowserName,
  MAX(TimeStart) AS TimeStart
FROM WPTRuns
GROUP BY BrowserName, Channel;
`
)

func init() {
	gcpFSMetricsSubQueryTemplate = NewQueryTemplate(gcpFSMetricsSubQueryRawTemplate)
	gcpFSCountQueryTemplate = NewQueryTemplate(gcpFSCountQueryRawTemplate)
	gcpFSSelectQueryTemplate = NewQueryTemplate(gcpFSSelectQueryRawTemplate)
	gcpFSPassRateForBrowserTemplate = NewQueryTemplate(gcpFSPassRateForBrowserRawTemplate)
	gcpFSBrowserImplementationStatusTemplate = NewQueryTemplate(gcpFSBrowserImplementationStatusRawTemplate)

	localFSMetricsSubQueryTemplate = NewQueryTemplate(localFSMetricsSubQueryRawTemplate)
	localFSCountQueryTemplate = NewQueryTemplate(localFSCountQueryRawTemplate)
	localFSSelectQueryTemplate = NewQueryTemplate(localFSSelectQueryRawTemplate)
	localFSPassRateForBrowserTemplate = NewQueryTemplate(localFSPassRateForBrowserRawTemplate)
	localFSBrowserImplementationStatusTemplate = NewQueryTemplate(localFSBrowserImplementationStatusRawTemplate)
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

func (t *BaseQueryTemplate) Execute(data any) string {
	var buf strings.Builder
	err := t.tmpl.Execute(&buf, data)
	if err != nil {
		slog.Error("unable to execute template", "error", err)

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

// GCPFSMetricsTemplateData contains the template data for gcpFSMetricsSubQueryTemplate.
type GCPFSMetricsTemplateData struct {
	Channel        string
	Clause         string
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

type FeatureSearchQueryArgs struct {
	MetricView              WPTMetricView
	Filters                 []string
	PageFilters             []string
	PageSize                int
	Offset                  int
	Prefilter               FeatureSearchPrefilterResult
	SortClause              string
	SortByStableBrowserImpl *SortByBrowserImplDetails
	SortByExpBrowserImpl    *SortByBrowserImplDetails
}

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
	//     The metrics retrieved are for each unique BrowserName/Channel/WebFeatureID.
	Query(args FeatureSearchQueryArgs) (string, map[string]interface{})

	// CountQuery generates the base query to return only the count of items.
	CountQuery(args FeatureSearchCountArgs) string
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
`
	gcpFSBaseQueryTemplate   = commonFSBaseQueryTemplate
	localFSBaseQueryTemplate = commonFSBaseQueryTemplate

	// commonFSImplementationStatusRawTemplate returns an array of structs that represent the implementation status.
	commonFSImplementationStatusRawTemplate = `
COALESCE(
	(
		SELECT ARRAY_AGG(
			STRUCT(
				bfa.BrowserName AS BrowserName,
				COALESCE(
					(
						SELECT 'available'
						FROM BrowserFeatureAvailabilities bfa1
						WHERE bfa.WebFeatureID = wf.ID
							AND bfa1.BrowserName = bfa.BrowserName
						LIMIT 1
					),
					'unavailable' -- Default if no match
				) AS ImplementationStatus,
				COALESCE(br.ReleaseDate, CAST(NULL AS TIMESTAMP)) AS ImplementationDate
			)
		)
		FROM BrowserFeatureAvailabilities bfa
		LEFT JOIN BrowserReleases br
			ON bfa.BrowserName = br.BrowserName AND bfa.BrowserVersion = br.BrowserVersion
		WHERE bfa.WebFeatureID = wf.ID
	),
	(
		SELECT ARRAY(
	   		SELECT AS STRUCT
				'' BrowserName,
				'unavailable' AS ImplementationStatus,
				CAST(NULL AS TIMESTAMP) AS ImplementationDate
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
			m.{{ .PassRateColumn }}
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
			{{ .Clause }}
		LIMIT 1
) AS SortMetric
`

	// gcpFSMetricsSubQueryRawTemplate generates a nested query that aggregates metrics by browser and
	// channel. It uses COALESCE to handle cases with no matching metrics.
	gcpFSMetricsSubQueryRawTemplate = `
COALESCE(
	(
		SELECT ARRAY_AGG(STRUCT(
				BrowserName AS BrowserName,
				{{ .PassRateColumn }} AS PassRate
			))
		FROM WPTRunFeatureMetrics @{FORCE_INDEX={{ .MetricIndex }}} metrics
		WHERE metrics.WebFeatureID = wf.ID
		AND metrics.Channel = @{{ .ChannelParam }}
    	{{ .Clause }}
	),
	(
		SELECT ARRAY(
			SELECT AS STRUCT
			'' BrowserName,
			CAST(0.0 AS NUMERIC) PassRate
		)
	)
) AS {{ .Channel }}Metrics
`

	// localFSMetricsSubQueryRawTemplate generates a nested query that aggregates metrics by browser and
	// channel. It uses COALESCE to handle cases with no matching metrics.
	localFSMetricsSubQueryRawTemplate = `
COALESCE(
	(
		SELECT ARRAY_AGG(
			STRUCT(
				BrowserName,
				{{ .PassRateColumn }} AS PassRate
			)
		)
		FROM MetricsAggregation WHERE WebFeatureID = wf.ID AND Channel = @{{ .ChannelParam }}
	),
	(
		SELECT ARRAY(
			SELECT AS STRUCT
				'' BrowserName,
				CAST(0.0 AS NUMERIC) PassRate
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

// Query uses the latest browsername/channel/timestart mapping to build a query from the prefilter query.
// This prevents an extra join to figure out the latest run for a particular.
// The one thing to note about to this implementation: If the latest run ever deprecates a feature,
// it will not be included in the query. However, a feature can only be deprecated by a bigger change in the ecosystem
// and is not a common thing and will have bigger changes outside of this repository than just here.
func (f GCPFeatureSearchBaseQuery) Query(args FeatureSearchQueryArgs) (
	string, map[string]interface{}) {
	params := make(map[string]interface{}, len(args.Prefilter.stableParams)+len(args.Prefilter.experimentalParams))
	maps.Copy(params, args.Prefilter.stableParams)
	maps.Copy(params, args.Prefilter.experimentalParams)
	stableParamName := "stableChannelParam"
	params[stableParamName] = "stable"
	experimentalParamName := "experimentalChannelParam"
	params[experimentalParamName] = "experimental"

	stableMetricsData := GCPFSMetricsTemplateData{
		Channel:        "Stable",
		Clause:         args.Prefilter.stableClause,
		PassRateColumn: metricsPassRateColumn(args.MetricView),
		MetricIndex:    metricsPassRateIndex(args.MetricView),
		ChannelParam:   stableParamName,
	}
	stableMetrics := gcpFSMetricsSubQueryTemplate.Execute(stableMetricsData)

	experimentalMetricsData := GCPFSMetricsTemplateData{
		Channel:        "Experimental",
		Clause:         args.Prefilter.experimentalClause,
		PassRateColumn: metricsPassRateColumn(args.MetricView),
		MetricIndex:    metricsPassRateIndex(args.MetricView),
		ChannelParam:   experimentalParamName,
	}
	experimentalMetrics := gcpFSMetricsSubQueryTemplate.Execute(experimentalMetricsData)

	var optionalJoins []JoinData
	if args.SortByStableBrowserImpl != nil {
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
