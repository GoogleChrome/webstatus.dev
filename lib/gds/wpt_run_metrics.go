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
	"context"
	"time"

	"cloud.google.com/go/datastore"
)

const wptRunMetricsKey = "WptRunMetrics"

// WPTRunToMetrics contains metrics about a particular WPT run.
type WPTRunToMetrics struct {
	WPTRun
	metrics *WPTRunMetric
}

// WPTRunMetrics contains metrics for multiple WPT runs over time.
type WPTRunMetrics []WPTRunToMetrics

// WPTRunMetric is the basic unit for measuring the tests in a given run.
type WPTRunMetric struct {
	// Datastore does not support unsigned integer currently.
	TotalTests *int `datastore:"total_tests"`
	TestPass   *int `datastore:"test_pass"`
	// The below fields cannot be overridden during a merge.
	RunID int64 `datastore:"run_id"`
}

// wptRunMetricMerge implements Mergeable for WPTRunMetric.
type wptRunMetricMerge struct{}

func (m wptRunMetricMerge) Merge(existing *WPTRunMetric, new *WPTRunMetric) *WPTRunMetric {
	return &WPTRunMetric{
		TotalTests: cmp.Or[*int](new.TotalTests, existing.TotalTests),
		TestPass:   cmp.Or[*int](new.TestPass, existing.TestPass),
	}
}

// wptMetricByRunIDs implements Filterable to filter by:
// - run_id (if given entity has a run_id that is part of a list)
// Compatible kinds:
// - wptRunsKey.
type wptMetricByRunIDs struct {
	runIDs []int64
}

func (f wptMetricByRunIDs) FilterQuery(query *datastore.Query) *datastore.Query {
	filters := make([]datastore.EntityFilter, 0, len(f.runIDs))
	for i := 0; i < len(f.runIDs); i++ {
		filters = append(filters, datastore.PropertyFilter{
			FieldName: "run_id",
			Operator:  "=",
			Value:     f.runIDs[i],
		})
	}
	return query.FilterEntity(datastore.OrFilter{Filters: filters})
}

// StoreWPTRunMetrics stores the metrics for a given run.
func (c *Client) StoreWPTRunMetrics(
	ctx context.Context,
	metric WPTRunMetric) error {
	// Try to get the WPT Run first.
	_, err := c.GetWPTRun(ctx, metric.RunID)
	if err != nil {
		return err
	}

	entityClient := entityClient[WPTRunMetric]{c}

	return entityClient.upsert(
		ctx,
		wptRunMetricsKey,
		&metric,
		wptRunMetricMerge{},
		wptRunIDFilter{runID: metric.RunID},
	)
}

// ListWPTMetricsByBrowser retrieves a list of metrics for the given
// browser name and channel.
func (c *Client) ListWPTMetricsByBrowser(
	ctx context.Context,
	browser string,
	channel string,
	startAt time.Time,
	endAt time.Time,
	pageToken *string) ([]WPTRunToMetrics, *string, error) {

	// TODO. create nested page token.
	runs, _, err := c.ListWPTRunsByBrowser(ctx, browser, channel, startAt, endAt, pageToken)
	if err != nil {
		return nil, nil, err
	}

	// TODO. If the number of run ids grows too much, will need to batch these.
	runIDs := make([]int64, len(runs))
	m := make(map[int64]WPTRunToMetrics, len(runs))
	for idx := range runs {
		runIDs[idx] = runs[idx].RunID
		m[runs[idx].RunID] = WPTRunToMetrics{WPTRun: *runs[idx], metrics: nil}
	}
	entityClient := entityClient[WPTRunMetric]{c}

	metrics, _, err := entityClient.list(
		ctx,
		wptRunMetricsKey,
		pageToken,
		wptMetricByRunIDs{
			runIDs: runIDs,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	for _, metric := range metrics {
		current := m[metric.RunID]
		current.metrics = metric
		m[metric.RunID] = current
	}
	ret := make([]WPTRunToMetrics, 0, len(m))
	for _, v := range ret {
		ret = append(ret, v)
	}
	return ret, nil, nil
}
