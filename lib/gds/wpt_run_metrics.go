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

package gds

import (
	"cmp"
	"time"

	"cloud.google.com/go/datastore"
	"golang.org/x/net/context"
)

const wptRunMetricsKey = "WptRunMetrics"

// WPTRunMetrics contains metrics about a particular WPT run.
type WPTRunMetrics struct {
	WPTRunMetadata
	WPTRunMetric
}

// WPTRunMetadata contains common metadata for a run.
type WPTRunMetadata struct {
	RunID          int64     `datastore:"run_id"`
	TimeStart      time.Time `datastore:"time_start"`
	TimeEnd        time.Time `datastore:"time_end"`
	BrowserName    string    `datastore:"browser_name"`
	BrowserVersion string    `datastore:"browser_version"`
	Channel        string    `datastore:"channel"`
	OSName         string    `datastore:"os_name"`
	OSVersion      string    `datastore:"os_version"`
}

// WPTRunMetric is the basic unit for measuring the tests in a given run.
type WPTRunMetric struct {
	// Datastore does not support unsigned integer currently.
	TotalTests *int `datastore:"total_tests"`
	TestPass   *int `datastore:"test_pass"`
}

// wptRunIDFilter implements Filterable to filter by run_id.
// Compatible kinds:
// - wptRunMetricsKey.
// - wptRunMetricsGroupByFeatureKey.
type wptRunIDFilter struct {
	runID int64
}

func (f wptRunIDFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.FilterField("run_id", "=", f.runID)
}

// wptRunMetadataMerge implements Mergeable for WPTRunMetadata.
type wptRunMetadataMerge struct{}

func (m wptRunMetadataMerge) Merge(existing *WPTRunMetadata, _ *WPTRunMetadata) *WPTRunMetadata {
	// The below fields cannot be overridden during a merge.
	return &WPTRunMetadata{
		RunID:          existing.RunID,
		TimeStart:      existing.TimeStart,
		TimeEnd:        existing.TimeEnd,
		BrowserName:    existing.BrowserName,
		BrowserVersion: existing.BrowserVersion,
		Channel:        existing.Channel,
		OSName:         existing.OSName,
		OSVersion:      existing.OSVersion,
	}
}

// wptRunMetricMerge implements Mergeable for WPTRunMetric.
type wptRunMetricMerge struct{}

func (m wptRunMetricMerge) Merge(existing *WPTRunMetric, new *WPTRunMetric) *WPTRunMetric {
	return &WPTRunMetric{
		TotalTests: cmp.Or[*int](new.TotalTests, existing.TotalTests),
		TestPass:   cmp.Or[*int](new.TestPass, existing.TestPass),
	}
}

// wptRunMetricsMerge implements Mergeable for WPTRunMetric.
type wptRunMetricsMerge struct{}

func (m wptRunMetricsMerge) Merge(existing *WPTRunMetrics, new *WPTRunMetrics) *WPTRunMetrics {
	return &WPTRunMetrics{
		WPTRunMetric: *wptRunMetricMerge{}.Merge(
			&existing.WPTRunMetric,
			&new.WPTRunMetric,
		),
		WPTRunMetadata: *wptRunMetadataMerge{}.Merge(
			&existing.WPTRunMetadata,
			&new.WPTRunMetadata,
		),
	}
}

// StoreWPTRunMetrics stores the metrics for a given run.
func (c *Client) StoreWPTRunMetrics(
	ctx context.Context,
	runMetadata WPTRunMetadata,
	runData WPTRunMetric) error {
	entityClient := entityClient[WPTRunMetrics]{c}

	return entityClient.upsert(
		ctx,
		wptRunMetricsKey,
		&WPTRunMetrics{
			WPTRunMetadata: runMetadata,
			WPTRunMetric:   runData,
		},
		wptRunMetricsMerge{},
		wptRunIDFilter{runID: runMetadata.RunID},
	)
}

// nolint: lll
// wptMetricsByBrowserFilter implements Filterable to filter by:
// - browser_name (equality)
// - channel (equality)
// - time_start (startAt >= x < endAt)
// https://github.com/web-platform-tests/wpt.fyi/blob/fb5bae7c6d04563864ef1c28a263a0a8d6637c4e/shared/test_run_query.go#L183-L186
//
// Compatible kinds:
// - wptRunMetricsKey.
// - wptRunMetricsGroupByFeatureKey.
type wptMetricsByBrowserFilter struct {
	startAt time.Time
	endAt   time.Time
	browser string
	channel string
}

func (f wptMetricsByBrowserFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.FilterField("browser_name", "=", f.browser).
		FilterField("channel", "=", f.channel).
		FilterField("time_start", ">=", f.startAt).
		FilterField("time_start", "<", f.endAt)
}

// wptMetricsByBrowserFilter implements Filterable to sort the results by
// time_start in descending order.
// Compatible kinds:
// - wptRunMetricsKey.
// - wptRunMetricsGroupByFeatureKey.
type wptMetricsSortByTimeStart struct{}

func (f wptMetricsSortByTimeStart) FilterQuery(query *datastore.Query) *datastore.Query {
	// For now, only sort by descending.
	return query.Order("-time_start")
}

// GetWPTMetricsByBrowser retrieves a list of metrics for the given
// browser name and channel.
func (c *Client) GetWPTMetricsByBrowser(
	ctx context.Context,
	browser string,
	channel string,
	startAt time.Time,
	endAt time.Time,
	pageToken *string) ([]*WPTRunMetrics, *string, error) {
	entityClient := entityClient[WPTRunMetrics]{c}

	return entityClient.list(
		ctx,
		wptRunMetricsKey,
		pageToken,
		wptMetricsByBrowserFilter{
			startAt: startAt,
			endAt:   endAt,
			browser: browser,
			channel: channel,
		},
		wptMetricsSortByTimeStart{},
	)
}
