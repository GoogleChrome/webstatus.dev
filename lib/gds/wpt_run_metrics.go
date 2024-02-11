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
	"time"

	"cloud.google.com/go/datastore"
	"golang.org/x/net/context"
)

const wptRunMetricsKey = "WptRunMetrics"

type WPTRunMetrics struct {
	WPTRunMetadata
	WPTTestMetric
}

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

type WPTTestMetric struct {
	TotalTests int64 `datastore:"total_tests"`
	TestPass   int64 `datastore:"test_pass"`
}

type wptRunsMetricsFilter struct {
	runID int64
}

func (f wptRunsMetricsFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.FilterField("run_id", "=", f.runID)
}

func (c *Client) StoreWPTMetrics(
	ctx context.Context,
	runMetadata WPTRunMetadata,
	runData WPTTestMetric) error {
	entityClient := entityClient[WPTRunMetrics]{c}

	return entityClient.upsert(
		ctx,
		wptRunMetricsKey,
		&WPTRunMetrics{
			WPTRunMetadata: runMetadata,
			WPTTestMetric:  runData,
		},
		wptRunsMetricsFilter{runID: runMetadata.RunID},
	)
}

type wptMetricsByBrowserFilter struct {
	startAt time.Time
	endAt   time.Time
	browser string
}

func (f wptMetricsByBrowserFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.FilterField("browser_name", "=", f.browser).
		FilterField("time_start", ">=", f.startAt).
		FilterField("time_start", "<=", f.endAt)
}

type wptMetricsSortByTimeStart struct{}

func (f wptMetricsSortByTimeStart) FilterQuery(query *datastore.Query) *datastore.Query {
	// For now, only sort by descending.
	return query.Order("-time_start")
}

func (c *Client) GetWPTMetricsByBrowser(
	ctx context.Context,
	browser string,
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
		},
		wptMetricsSortByTimeStart{},
	)
}
