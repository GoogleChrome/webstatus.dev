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
	"context"
	"time"

	"cloud.google.com/go/datastore"
)

const wptRunsKey = "WptRuns"

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

// WPTRun contains all information about a WPT run.
type WPTRun struct {
	WPTRunMetadata
	TestMetric         *WPTRunMetric                 `datastore:"test_metric"`
	FeatureTestMetrics []WPTRunMetricsGroupByFeature `datastore:"feature_test_metrics"`
}

// wptRunIDFilter implements Filterable to filter by run_id.
// Compatible kinds:
// - wptRunsKey.
type wptRunIDFilter struct {
	runID int64
}

func (f wptRunIDFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.FilterField("run_id", "=", f.runID)
}

// wptRunMerge implements Mergeable for WPTRun.
type wptRunMerge struct{}

func (m wptRunMerge) Merge(existing *WPTRun, new *WPTRun) *WPTRun {
	return &WPTRun{
		WPTRunMetadata: *wptRunMetadataMerge{}.Merge(
			&existing.WPTRunMetadata, &new.WPTRunMetadata),
		TestMetric:         wptRunMetricMerge{}.Merge(existing.TestMetric, new.TestMetric),
		FeatureTestMetrics: *wptRunFeatureTestMetricsMerge{}.Merge(&existing.FeatureTestMetrics, &new.FeatureTestMetrics),
	}
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

// StoreWPTRun stores the metadata for a given run.
func (c *Client) StoreWPTRunMetadata(
	ctx context.Context,
	metadata WPTRunMetadata) error {
	entityClient := entityClient[WPTRun]{c}

	return entityClient.upsert(
		ctx,
		wptRunsKey,
		&WPTRun{
			WPTRunMetadata:     metadata,
			TestMetric:         nil,
			FeatureTestMetrics: nil,
		},
		wptRunMerge{},
		wptRunIDFilter{runID: metadata.RunID},
	)
}

// GetWPTRun gets the metadata for a given run.
func (c *Client) GetWPTRun(
	ctx context.Context,
	runID int64) (*WPTRun, error) {
	entityClient := entityClient[WPTRun]{c}

	return entityClient.get(
		ctx,
		wptRunsKey,
		wptRunIDFilter{runID: runID},
	)
}

// nolint: lll
// wptRunsByBrowserFilter implements Filterable to filter by:
// - browser_name (equality)
// - channel (equality)
// - time_start (startAt >= x < endAt)
// - sort by time_start
// https://github.com/web-platform-tests/wpt.fyi/blob/fb5bae7c6d04563864ef1c28a263a0a8d6637c4e/shared/test_run_query.go#L183-L186
//
// Compatible kinds:
// - wptRunsKey.
type wptRunsByBrowserFilter struct {
	startAt time.Time
	endAt   time.Time
	browser string
	channel string
}

func (f wptRunsByBrowserFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.FilterField("browser_name", "=", f.browser).
		FilterField("channel", "=", f.channel).
		FilterField("time_start", ">=", f.startAt).
		FilterField("time_start", "<", f.endAt).
		Order("-time_start")
}

// ListWPTRunsByBrowser returns a list of runs
// This is a helper method for other list methods.
func (c *Client) ListWPTRunsByBrowser(
	ctx context.Context,
	browser string,
	channel string,
	startAt time.Time,
	endAt time.Time,
	pageToken *string) ([]*WPTRun, *string, error) {
	entityClient := entityClient[WPTRun]{c}

	return entityClient.list(
		ctx,
		wptRunsKey,
		pageToken,
		wptRunsByBrowserFilter{
			startAt: startAt,
			endAt:   endAt,
			browser: browser,
			channel: channel,
		},
	)
}
