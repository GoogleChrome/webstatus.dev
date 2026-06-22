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
const LatestWPTRunFeatureMetricsTable = "LatestWPTRunFeatureMetrics"

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
	TotalTests        *int64         `spanner:"TotalTests"`
	TestPass          *int64         `spanner:"TestPass"`
	TotalSubtests     *int64         `spanner:"TotalSubtests"`
	SubtestPass       *int64         `spanner:"SubtestPass"`
	FeatureRunDetails map[string]any `spanner:"-"` // Not directly stored in Spanner
}

// SpannerLatestWPTRunFeatureMetric represents a pointer to an entry in WPTRunFeatureMetrics.
type SpannerLatestWPTRunFeatureMetric struct {
	RunMetricID  string `spanner:"RunMetricID"`
	WebFeatureID string `spanner:"WebFeatureID"`
	BrowserName  string `spanner:"BrowserName"`
	Channel      string `spanner:"Channel"`
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

// shouldUpsertLatestMetric determines whether the latest metric should be upserted based on timestamp comparison.
func shouldUpsertLatestMetric(existingTimeStart *time.Time, newTimeStart time.Time) bool {
	return existingTimeStart == nil || existingTimeStart.IsZero() || newTimeStart.After(*existingTimeStart)
}

// mergeAndCreateWPTRunFeatureMetricMutation merges the incoming metric with the existing one (if present)
// and returns the Spanner mutation.
func mergeAndCreateWPTRunFeatureMetricMutation(
	metric SpannerWPTRunFeatureMetric,
	existingMetric *SpannerWPTRunFeatureMetric,
) (*spanner.Mutation, error) {
	if existingMetric == nil {
		// Act as if this is an insertion.
		return spanner.InsertOrUpdateStruct(WPTRunFeatureMetricTable, metric)
	}

	// Read the existing metric and merge the values.
	merged := *existingMetric
	// Only allow overriding of the test numbers.
	merged.TestPass = cmp.Or[*int64](metric.TestPass, merged.TestPass, nil)
	merged.TotalTests = cmp.Or[*int64](metric.TotalTests, merged.TotalTests, nil)
	merged.TestPassRate = getPassRate(merged.TestPass, merged.TotalTests)
	// Allow subtest metrics to be reset to nil.
	merged.SubtestPass = metric.SubtestPass
	merged.TotalSubtests = metric.TotalSubtests
	merged.SubtestPassRate = getPassRate(merged.SubtestPass, merged.TotalSubtests)
	// Allow feature run details to be reset
	merged.FeatureRunDetails = metric.FeatureRunDetails

	return spanner.InsertOrUpdateStruct(WPTRunFeatureMetricTable, merged)
}

// buildWPTRunFeatureMetricMutations creates the database mutations for a single WPT run metric,
// merging with existing values and updating the latest metrics if newer.
func buildWPTRunFeatureMetricMutations(
	metric SpannerWPTRunFeatureMetric,
	existingMetric *SpannerWPTRunFeatureMetric,
	existingTimeStart *time.Time,
) ([]*spanner.Mutation, error) {
	var mutations []*spanner.Mutation

	m0, err := mergeAndCreateWPTRunFeatureMetricMutation(metric, existingMetric)
	if err != nil {
		return nil, err
	}
	if m0 != nil {
		mutations = append(mutations, m0)
	}

	// Update LatestWPTRunFeatureMetrics if newer
	if shouldUpsertLatestMetric(existingTimeStart, metric.TimeStart) {
		m1, err := spanner.InsertOrUpdateStruct(
			LatestWPTRunFeatureMetricsTable,
			SpannerLatestWPTRunFeatureMetric{
				RunMetricID:  metric.ID,
				WebFeatureID: metric.WebFeatureID,
				BrowserName:  metric.BrowserName,
				Channel:      metric.Channel,
			},
		)
		if err != nil {
			return nil, err
		}
		mutations = append(mutations, m1)
	}

	return mutations, nil
}

// UpsertWPTRunFeatureMetrics will upsert WPT Run metrics for a given WPT Run ID.
// The RunID must exist in a row in the WPTRuns table.
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
		// Extract browser, channel, and runID (they are all the same for this batch)
		var runID string
		var browserName, channel string
		for i, metric := range spannerMetrics {
			if i == 0 {
				runID = metric.ID
				browserName = metric.BrowserName
				channel = metric.Channel

				break
			}
		}

		// 1. Batch read latest timestamps for this browser/channel (using comparable key)
		latestTimeStartsRows, err := newAllByKeysEntityReader[
			latestWPTRunFeatureMetricTimeStartsMapper,
			latestTimeStartsKey,
			webFeatureLatestTimeStart,
		](c).readAllByKeysWithTransaction(ctx, latestTimeStartsKey{
			BrowserName: browserName,
			Channel:     channel,
		}, txn)
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}
		// Convert to map for fast lookup
		existingTimestamps := make(map[string]time.Time)
		for _, row := range latestTimeStartsRows {
			existingTimestamps[row.WebFeatureID] = row.TimeStart
		}

		// 2. Batch read ALL existing metrics for this run (using comparable string runID key)
		existingMetricsRows, err := newAllByKeysEntityReader[
			existingWPTRunFeatureMetricsMapper,
			string,
			SpannerWPTRunFeatureMetric,
		](c).readAllByKeysWithTransaction(ctx, runID, txn)
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}
		// Convert to map for fast lookup
		existingMetrics := make(map[string]SpannerWPTRunFeatureMetric)
		for _, row := range existingMetricsRows {
			existingMetrics[row.WebFeatureID] = row
		}

		mutations := []*spanner.Mutation{}
		for _, metric := range spannerMetrics {
			// Resolve existing metric from batch read
			var existingMetric *SpannerWPTRunFeatureMetric
			if em, found := existingMetrics[metric.WebFeatureID]; found {
				existingMetric = &em
			}

			// Resolve latest timestamp from batch read
			var existingTimeStart *time.Time
			if t, found := existingTimestamps[metric.WebFeatureID]; found {
				existingTimeStart = &t
			}

			ms, err := buildWPTRunFeatureMetricMutations(metric, existingMetric, existingTimeStart)
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
			mutations = append(mutations, ms...)
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
	params := map[string]any{
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
	params := map[string]any{
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
func noPageTokenFeatureSubset(params map[string]any, featureKeys []string,
	tmplData *FeatureMetricsTemplateData) {
	params["featureKeys"] = featureKeys
	tmplData.FeatureKeyFilter = multipleFeaturesMetricSubsetRawTemplate
}

// withPageTokenAllFeatures adjusts the template data and parameters when a page token is
// provided and the aggregation applies to all features.
func withPageTokenAllFeatures(params map[string]any, cursor WPTRunCursor,
	tmplData *FeatureMetricsTemplateData) {
	tmplData.PageFilter = commonFeatureMetricPaginationRawTemplate
	params["lastTimestamp"] = cursor.LastTimeStart
	params["lastRunID"] = cursor.LastRunID
}

// withPageTokenFeatureSubset adjusts the template data and parameters when a page token is
// provided and the aggregation applies to a particular list of features.
func withPageTokenFeatureSubset(
	params map[string]any,
	featureKeys []string,
	cursor WPTRunCursor,
	tmplData *FeatureMetricsTemplateData) {
	tmplData.PageFilter = commonFeatureMetricPaginationRawTemplate
	tmplData.FeatureKeyFilter = multipleFeaturesMetricSubsetRawTemplate
	params["featureKeys"] = featureKeys
	params["lastTimestamp"] = cursor.LastTimeStart
	params["lastRunID"] = cursor.LastRunID
}

type spannerWPTRunFeatureMetricIDAndWebFeatureID struct {
	ID           string `spanner:"ID"`
	WebFeatureID string `spanner:"WebFeatureID"`
}

func (c *Client) getAllSpannerWPTRunFeatureMetricIDsByWebFeatureID(
	ctx context.Context,
	webFeatureID string) ([]spannerWPTRunFeatureMetricIDAndWebFeatureID, error) {
	metrics, err := c.getAllWPTRunFeatureMetricIDsByWebFeatureID(ctx, webFeatureID)
	if err != nil {
		return nil, err
	}

	pairs := make([]spannerWPTRunFeatureMetricIDAndWebFeatureID, 0, len(metrics))
	for _, metric := range metrics {
		pairs = append(pairs, spannerWPTRunFeatureMetricIDAndWebFeatureID{
			ID:           metric.ID,
			WebFeatureID: metric.WebFeatureID,
		})
	}

	return pairs, nil
}

type wptRunFeatureMetricMapper struct{}

func (m wptRunFeatureMetricMapper) SelectAllByKeys(webFeatureID string) spanner.Statement {
	stmt := spanner.NewStatement(`
		SELECT
			*
		FROM WPTRunFeatureMetrics
		WHERE WebFeatureID = @webFeatureID
		ORDER BY TimeStart DESC`)
	stmt.Params = map[string]any{
		"webFeatureID": webFeatureID,
	}

	return stmt
}

func (c *Client) getAllWPTRunFeatureMetricIDsByWebFeatureID(
	ctx context.Context,
	webFeatureID string) ([]SpannerWPTRunFeatureMetric, error) {
	return newAllByKeysEntityReader[
		wptRunFeatureMetricMapper,
		string,
		SpannerWPTRunFeatureMetric,
	](c).readAllByKeys(ctx, webFeatureID)
}

type latestWptRunsFeatureMetricMapper struct{}

func (m latestWptRunsFeatureMetricMapper) SelectAllByKeys(webFeatureID string) spanner.Statement {
	stmt := spanner.NewStatement(`
		SELECT
			*
		FROM LatestWPTRunFeatureMetrics
		WHERE WebFeatureID = @webFeatureID`)
	stmt.Params = map[string]any{
		"webFeatureID": webFeatureID,
	}

	return stmt
}

func (c *Client) getAllSpannerLatestWPTRunFeatureMetricIDsByWebFeatureID(
	ctx context.Context,
	webFeatureID string) ([]SpannerLatestWPTRunFeatureMetric, error) {
	return newAllByKeysEntityReader[
		latestWptRunsFeatureMetricMapper,
		string,
		SpannerLatestWPTRunFeatureMetric,
	](c).readAllByKeys(ctx, webFeatureID)
}

// --- NEW MAPPERS AND STRUCTS FOR BATCH QUERIES ---

type webFeatureLatestTimeStart struct {
	WebFeatureID string    `spanner:"WebFeatureID"`
	TimeStart    time.Time `spanner:"TimeStart"`
}

type latestTimeStartsKey struct {
	BrowserName string
	Channel     string
}

type latestWPTRunFeatureMetricTimeStartsMapper struct{}

func (m latestWPTRunFeatureMetricTimeStartsMapper) SelectAllByKeys(keys latestTimeStartsKey) spanner.Statement {
	stmt := spanner.NewStatement(`
        SELECT l.WebFeatureID, wpfm.TimeStart
        FROM LatestWPTRunFeatureMetrics l
        JOIN WPTRunFeatureMetrics wpfm ON l.RunMetricID = wpfm.ID AND l.WebFeatureID = wpfm.WebFeatureID
        WHERE l.BrowserName = @browserName
        AND l.Channel = @channel`)
	stmt.Params = map[string]any{
		"browserName": keys.BrowserName,
		"channel":     keys.Channel,
	}

	return stmt
}

type existingWPTRunFeatureMetricsMapper struct{}

func (m existingWPTRunFeatureMetricsMapper) SelectAllByKeys(runID string) spanner.Statement {
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
        WHERE ID = @runID`)
	stmt.Params = map[string]any{
		"runID": runID,
	}

	return stmt
}
