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
	"cmp"
	"context"
	"errors"
	"log/slog"
	"math/big"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const WPTRunFeatureMetricTable = "WPTRunFeatureMetrics"

func init() {
	getFeatureMetricBaseTemplate = NewQueryTemplate(getFeatureMetricBaseRawTemplate)
}

// nolint: gochecknoglobals // WONTFIX. Compile the template once at startup. Startup fails if invalid.
var (
	// getFeatureMetricBaseTemplate is the compiled version of getFeatureMetricBaseRawTemplate.
	getFeatureMetricBaseTemplate BaseQueryTemplate
)

const (
	getFeatureMetricBaseRawTemplate = `
	SELECT
		r.ExternalRunID,
		r.TimeStart,
{{ if .IsSingleFeature }}
		{{ .TotalColumn }} AS TotalTests,
		{{ .PassColumn }} AS TestPass
{{ else }}
		SUM(wpfm.{{ .TotalColumn }}) AS TotalTests,
		SUM(wpfm.{{ .PassColumn }}) AS TestPass
{{ end }}
	FROM WPTRuns r
	JOIN WPTRunFeatureMetrics wpfm ON r.ID = wpfm.ID
	LEFT OUTER JOIN WebFeatures wf ON wf.ID = wpfm.WebFeatureID
	LEFT OUTER JOIN ExcludedFeatureKeys efk ON wf.FeatureKey = efk.FeatureKey
	WHERE r.BrowserName = @browserName
		AND wpfm.{{ .TotalColumn }} IS NOT NULL
		AND wpfm.{{ .PassColumn }} IS NOT NULL
{{ if .FeatureKeyFilter }}
		{{ .FeatureKeyFilter }}
{{ end }}
{{ if .ExtraFilter }}
		{{ .ExtraFilter }}
{{ end }}
		AND r.Channel = @channel
		AND r.TimeStart >= @startAt AND r.TimeStart < @endAt
{{ if .PageFilter }}
		{{ .PageFilter }}
{{ end }}
{{ if not .IsSingleFeature }}
	GROUP BY r.ExternalRunID, r.TimeStart
{{ end }}
	ORDER BY r.TimeStart DESC, r.ExternalRunID DESC LIMIT @pageSize`

	commonFeatureMetricPaginationRawTemplate = `
		AND (r.TimeStart < @lastTimestamp OR
			r.TimeStart = @lastTimestamp AND r.ExternalRunID < @lastRunID)`

	singleFeatureMetricSubsetRawTemplate    = `AND wf.FeatureKey = @featureKey`
	multipleFeaturesMetricSubsetRawTemplate = `AND wf.FeatureKey IN UNNEST(@featureKeys)`
)

// FeatureMetricsTemplateData contains the variables for getFeatureMetricBaseRawTemplate.
type FeatureMetricsTemplateData struct {
	TotalColumn      string
	PassColumn       string
	PageFilter       string
	FeatureKeyFilter string
	ExtraFilter      string
	IsSingleFeature  bool
}

// SpannerWPTRunFeatureMetric is a wrapper for the metric data that is actually
// stored in spanner. This is useful because the spanner id is not useful to
// return to the end user since it is used to decouple the primary keys between
// this system and wpt.fyi.
type SpannerWPTRunFeatureMetric struct {
	ID                string           `spanner:"ID"`
	WebFeatureID      string           `spanner:"WebFeatureID"`
	FeatureRunDetails spanner.NullJSON `spanner:"FeatureRunDetails"`
	WPTRunFeatureMetric
	// Calculated pass rate
	TestPassRate    *big.Rat `spanner:"TestPassRate"`
	SubtestPassRate *big.Rat `spanner:"SubtestPassRate"`
	// Denormalized data from wpt runs.
	BrowserName string    `spanner:"BrowserName"`
	Channel     string    `spanner:"Channel"`
	TimeStart   time.Time `spanner:"TimeStart"`
}

// WPTRunFeatureMetric represents the metrics for a particular feature in a run.
type WPTRunFeatureMetric struct {
	TotalTests        *int64                 `spanner:"TotalTests"`
	TestPass          *int64                 `spanner:"TestPass"`
	TotalSubtests     *int64                 `spanner:"TotalSubtests"`
	SubtestPass       *int64                 `spanner:"SubtestPass"`
	FeatureRunDetails map[string]interface{} `spanner:"-"` // Not directly stored in Spanner
}

func getPassRate(testPass, totalTests *int64) *big.Rat {
	if testPass == nil || totalTests == nil || *totalTests == 0 {
		return nil
	}

	return big.NewRat(*testPass, *totalTests)
}

func (c *Client) CreateSpannerWPTRunFeatureMetric(
	webFeatureID string,
	wptRunData WPTRunDataForMetrics,
	in WPTRunFeatureMetric) SpannerWPTRunFeatureMetric {
	var featureRunDetails spanner.NullJSON
	if in.FeatureRunDetails != nil {
		featureRunDetails = spanner.NullJSON{Value: in.FeatureRunDetails, Valid: true}
	}

	return SpannerWPTRunFeatureMetric{
		ID:                  wptRunData.ID,
		WebFeatureID:        webFeatureID,
		Channel:             wptRunData.Channel,
		BrowserName:         wptRunData.BrowserName,
		TimeStart:           wptRunData.TimeStart,
		WPTRunFeatureMetric: in,
		TestPassRate:        getPassRate(in.TestPass, in.TotalTests),
		SubtestPassRate:     getPassRate(in.SubtestPass, in.TotalSubtests),
		FeatureRunDetails:   featureRunDetails,
	}
}

func (c *Client) convertExternalMetricsToSpannerMetrics(ctx context.Context,
	wptRunData *WPTRunDataForMetrics,
	inputMetrics map[string]WPTRunFeatureMetric) ([]SpannerWPTRunFeatureMetric, error) {
	spannerMetrics := make([]SpannerWPTRunFeatureMetric, 0, len(inputMetrics))
	for externalFeatureID, inputMetric := range inputMetrics {
		featureID, err := c.GetIDFromFeatureKey(ctx, NewFeatureKeyFilter(externalFeatureID))
		if err != nil {
			if errors.Is(err, ErrQueryReturnedNoResults) {
				slog.WarnContext(
					ctx,
					"unable to find internal webfeatureID for key. will skip", "key",
					externalFeatureID)

				continue
			}

			return nil, errors.Join(err, ErrInternalQueryFailure)
		}
		if featureID == nil {
			return nil, ErrInternalQueryFailure
		}

		spannerMetrics = append(spannerMetrics,
			c.CreateSpannerWPTRunFeatureMetric(*featureID, *wptRunData, inputMetric))
	}

	return spannerMetrics, nil
}

// UpsertWPTRunFeatureMetrics will upsert WPT Run metrics for a given WPT Run ID.
// The RunID must exist in a row in the WPTRuns table.
// If a metric does not exist, it will insert a new metric.
// If a metric exists, it will only update the following columns:
//  1. TotalTests
//  2. TestPass
//  3. TestPassRate
//  4. TotalSubtests
//  5. SubtestPass
//  6. SubtestPassRate
func (c *Client) UpsertWPTRunFeatureMetrics(
	ctx context.Context,
	externalRunID int64,
	inputMetrics map[string]WPTRunFeatureMetric) error {
	wptRunData, err := c.GetWPTRunDataByRunIDForMetrics(ctx, externalRunID)
	if err != nil {
		return err
	}

	spannerMetrics, err := c.convertExternalMetricsToSpannerMetrics(ctx, wptRunData, inputMetrics)
	if err != nil {
		return err
	}

	_, err = c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{}
		for _, metric := range spannerMetrics {
			// Create a metric with the retrieved ID
			stmt := spanner.NewStatement(`
			SELECT
				ID,
				WebFeatureID,
				TotalTests,
				TestPass,
				TestPassRate,
				TotalSubtests,
				SubtestPass,
				SubtestPassRate,
				FeatureRunDetails,
				TimeStart,
				Channel,
				BrowserName
			FROM WPTRunFeatureMetrics
			WHERE ID = @id AND WebFeatureID = @webFeatureID
			LIMIT 1`)
			parameters := map[string]interface{}{
				"id":           metric.ID,
				"webFeatureID": metric.WebFeatureID,
			}
			stmt.Params = parameters

			// Attempt to query for the row.
			it := txn.Query(ctx, stmt)
			defer it.Stop()
			var m *spanner.Mutation
			row, err := it.Next()

			// nolint: nestif // TODO: fix in the future.
			if err != nil {
				if errors.Is(err, iterator.Done) {
					// No rows returned. Act as if this is an insertion.
					var err error
					m, err = spanner.InsertOrUpdateStruct(WPTRunFeatureMetricTable, metric)
					if err != nil {
						return errors.Join(ErrInternalQueryFailure, err)
					}
				} else {
					// An unexpected error occurred.

					return errors.Join(ErrInternalQueryFailure, err)
				}
			} else {
				// Read the existing metric and merge the values.
				var existingMetric SpannerWPTRunFeatureMetric
				err = row.ToStruct(&existingMetric)
				if err != nil {
					return errors.Join(ErrInternalQueryFailure, err)
				}
				// Only allow overriding of the test numbers.
				existingMetric.TestPass = cmp.Or[*int64](metric.TestPass, existingMetric.TestPass, nil)
				existingMetric.TotalTests = cmp.Or[*int64](metric.TotalTests, existingMetric.TotalTests, nil)
				existingMetric.TestPassRate = getPassRate(existingMetric.TestPass, existingMetric.TotalTests)
				// Allow subtest metrics to be reset to nil.
				existingMetric.SubtestPass = metric.SubtestPass
				existingMetric.TotalSubtests = metric.TotalSubtests
				existingMetric.SubtestPassRate = getPassRate(existingMetric.SubtestPass, existingMetric.TotalSubtests)
				// Allow feature run details to be reset
				existingMetric.FeatureRunDetails = metric.FeatureRunDetails
				m, err = spanner.InsertOrUpdateStruct(WPTRunFeatureMetricTable, existingMetric)
				if err != nil {
					return errors.Join(ErrInternalQueryFailure, err)
				}
			}
			mutations = append(mutations, m)
		}

		// Buffer the mutation to be committed.
		err = txn.BufferWrite(mutations)
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}

		return nil
	})
	if err != nil {
		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}

// Helper function to determine the correct TestPass column name.
func metricsTestPassColumn(metricView WPTMetricView) string {
	switch metricView {
	case WPTSubtestView:
		return "SubtestPass"
	case WPTTestView:
		return "TestPass"
	}

	return "SubtestPass"
}

// Helper function to determine the correct PassRate column name.
func metricsTotalTestColumn(metricView WPTMetricView) string {
	switch metricView {
	case WPTSubtestView:
		return "TotalSubtests"
	case WPTTestView:
		return "TotalTests"
	}

	return "TotalSubtests"
}

// WPTRunFeatureMetricWithTime contains metrics for a feature at a given time.
type WPTRunFeatureMetricWithTime struct {
	TimeStart  time.Time `spanner:"TimeStart"`
	RunID      int64     `spanner:"ExternalRunID"`
	TotalTests *int64    `spanner:"TotalTests"`
	TestPass   *int64    `spanner:"TestPass"`
}

// ListMetricsForFeatureIDBrowserAndChannel attempts to return a page of
// metrics based on a web feature key, browser name and channel. A time window
// must be specified to analyze the runs according to the TimeStart of the run.
// If the page size matches the pageSize, a page token is returned. Else,
// no page token is returned.
func (c *Client) ListMetricsForFeatureIDBrowserAndChannel(
	ctx context.Context,
	featureKey string,
	browser string,
	channel string,
	metric WPTMetricView,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]WPTRunFeatureMetricWithTime, *string, error) {
	params := map[string]interface{}{
		"featureKey":  featureKey,
		"browserName": browser,
		"channel":     channel,
		"startAt":     startAt,
		"endAt":       endAt,
		"pageSize":    pageSize,
	}

	tmplData := FeatureMetricsTemplateData{
		TotalColumn:      metricsTotalTestColumn(metric),
		PassColumn:       metricsTestPassColumn(metric),
		PageFilter:       "",
		FeatureKeyFilter: singleFeatureMetricSubsetRawTemplate,
		ExtraFilter:      removeExcludedKeyFilterAND,
		IsSingleFeature:  true,
	}

	if pageToken != nil {
		cursor, err := decodeWPTRunCursor(*pageToken)
		if err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		params["lastTimestamp"] = cursor.LastTimeStart
		params["lastRunID"] = cursor.LastRunID
		tmplData.PageFilter = commonFeatureMetricPaginationRawTemplate
	}
	tmpl := getFeatureMetricBaseTemplate.Execute(tmplData)
	stmt := spanner.NewStatement(tmpl)
	stmt.Params = params

	txn := c.Single()
	defer txn.Close()
	it := txn.Query(ctx, stmt)
	defer it.Stop()

	var featureMetrics []WPTRunFeatureMetricWithTime
	for {
		row, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var featureMetric WPTRunFeatureMetricWithTime
		if err := row.ToStruct(&featureMetric); err != nil {
			return nil, nil, err
		}
		featureMetrics = append(featureMetrics, featureMetric)
	}

	if len(featureMetrics) == pageSize {
		lastFeatureMetric := featureMetrics[len(featureMetrics)-1]
		newCursor := encodeWPTRunCursor(lastFeatureMetric.TimeStart, lastFeatureMetric.RunID)

		return featureMetrics, &newCursor, nil
	}

	return featureMetrics, nil, nil
}

// WPTRunAggregationMetricWithTime contains metrics for a particular aggregation
// at a given time. For now, it is the same metrics as
// WPTRunFeatureMetricWithTime.
type WPTRunAggregationMetricWithTime struct {
	WPTRunFeatureMetricWithTime
}

// ListMetricsOverTimeWithAggregatedTotals attempts to return a page of
// metrics based on browser name and channel. Users can provide a list of web
// feature keys. If the list is provided, the aggregation will be scoped to those
// feature keys. If an empty or nil list is provided, the aggregation is applied
// to all features.
// A time window must be specified to analyze the runs according to the
// TimeStart of the run.
// If the page size matches the pageSize, a page token is returned. Else,
// no page token is returned.
func (c *Client) ListMetricsOverTimeWithAggregatedTotals(
	ctx context.Context,
	featureKeys []string,
	browser string,
	channel string,
	metric WPTMetricView,
	startAt, endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]WPTRunAggregationMetricWithTime, *string, error) {
	params := map[string]interface{}{
		"browserName": browser,
		"channel":     channel,
		"startAt":     startAt,
		"endAt":       endAt,
		"pageSize":    pageSize,
	}

	tmplData := FeatureMetricsTemplateData{
		TotalColumn:      metricsTotalTestColumn(metric),
		PassColumn:       metricsTestPassColumn(metric),
		PageFilter:       "",
		FeatureKeyFilter: "",
		ExtraFilter:      removeExcludedKeyFilterAND,
		IsSingleFeature:  false,
	}

	// nolint: nestif // TODO: fix in the future.
	if pageToken == nil {
		if len(featureKeys) > 0 {
			noPageTokenFeatureSubset(params, featureKeys, &tmplData)
		}
	} else {
		cursor, err := decodeWPTRunCursor(*pageToken)
		if err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		if len(featureKeys) == 0 {
			withPageTokenAllFeatures(params, *cursor, &tmplData)
		} else {
			withPageTokenFeatureSubset(params, featureKeys, *cursor, &tmplData)
		}
	}

	tmpl := getFeatureMetricBaseTemplate.Execute(tmplData)
	stmt := spanner.NewStatement(tmpl)
	stmt.Params = params

	txn := c.Single()
	defer txn.Close()
	it := txn.Query(ctx, stmt)
	defer it.Stop()

	var aggregationMetrics []WPTRunAggregationMetricWithTime
	for {
		row, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var aggregationMetric WPTRunAggregationMetricWithTime
		if err := row.ToStruct(&aggregationMetric); err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		aggregationMetrics = append(aggregationMetrics, aggregationMetric)
	}

	if len(aggregationMetrics) == pageSize {
		lastFeatureMetric := aggregationMetrics[len(aggregationMetrics)-1]
		newCursor := encodeWPTRunCursor(lastFeatureMetric.TimeStart, lastFeatureMetric.RunID)

		return aggregationMetrics, &newCursor, nil
	}

	return aggregationMetrics, nil, nil
}

// noPageTokenFeatureSubset adjusts the template data and parameters when a page token is
// not provided and the aggregation applies to a particular list of features.
func noPageTokenFeatureSubset(params map[string]interface{}, featureKeys []string,
	tmplData *FeatureMetricsTemplateData) {
	params["featureKeys"] = featureKeys
	tmplData.FeatureKeyFilter = multipleFeaturesMetricSubsetRawTemplate
}

// withPageTokenAllFeatures adjusts the template data and parameters when a page token is
// provided and the aggregation applies to all features.
func withPageTokenAllFeatures(params map[string]interface{}, cursor WPTRunCursor,
	tmplData *FeatureMetricsTemplateData) {
	tmplData.PageFilter = commonFeatureMetricPaginationRawTemplate
	params["lastTimestamp"] = cursor.LastTimeStart
	params["lastRunID"] = cursor.LastRunID
}

// withPageTokenFeatureSubset adjusts the template data and parameters when a page token is
// provided and the aggregation applies to a particular list of features.
func withPageTokenFeatureSubset(
	params map[string]interface{},
	featureKeys []string,
	cursor WPTRunCursor,
	tmplData *FeatureMetricsTemplateData) {
	tmplData.PageFilter = commonFeatureMetricPaginationRawTemplate
	tmplData.FeatureKeyFilter = multipleFeaturesMetricSubsetRawTemplate
	params["featureKeys"] = featureKeys
	params["lastTimestamp"] = cursor.LastTimeStart
	params["lastRunID"] = cursor.LastRunID
}
