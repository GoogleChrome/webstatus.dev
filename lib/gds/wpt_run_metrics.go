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
)

// WPTRunToMetrics contains metrics about a particular WPT run.
type WPTRunToMetrics struct {
	WPTRunMetadata
	*WPTRunMetric
}

// WPTRunToMetrics contains metrics about a particular WPT run.
type WPTRunToMetricsByFeature struct {
	WPTRunMetadata
	*WPTRunMetric
	FeatureID string
}

// WPTRunMetrics contains metrics for multiple WPT runs over time.
type WPTRunMetrics []WPTRunToMetrics

// WPTRunMetric is the basic unit for measuring the tests in a given run.
type WPTRunMetric struct {
	// Datastore does not support unsigned integer currently.
	TotalTests *int `datastore:"total_tests"`
	TestPass   *int `datastore:"test_pass"`
}

// wptRunMetricMerge implements Mergeable for WPTRunMetric.
type wptRunMetricMerge struct{}

func (m wptRunMetricMerge) Merge(existing *WPTRunMetric, new *WPTRunMetric) *WPTRunMetric {
	if existing == nil && new != nil {
		return new
	}
	if existing != nil && new == nil {
		return existing
	}
	if existing == nil && new == nil {
		return nil
	}

	return &WPTRunMetric{
		TotalTests: cmp.Or[*int](new.TotalTests, existing.TotalTests),
		TestPass:   cmp.Or[*int](new.TestPass, existing.TestPass),
	}
}

// StoreWPTRunMetrics stores the metrics for a given run.
func (c *Client) StoreWPTRunMetrics(
	ctx context.Context,
	runID int64,
	metric *WPTRunMetric) error {
	// Try to get the WPT Run first.
	run, err := c.GetWPTRun(ctx, runID)
	if err != nil {
		return err
	}

	run.TestMetric = metric
	entityClient := entityClient[WPTRun]{c}

	return entityClient.upsert(
		ctx,
		wptRunsKey,
		run,
		wptRunMerge{},
		wptRunIDFilter{runID: runID},
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

	runs, _, err := c.ListWPTRunsByBrowser(ctx, browser, channel, startAt, endAt, pageToken)
	if err != nil {
		return nil, nil, err
	}

	ret := make([]WPTRunToMetrics, 0, len(runs))
	for _, run := range runs {
		ret = append(ret, WPTRunToMetrics{
			WPTRunMetadata: *run.WPTRunMetadata,
			WPTRunMetric:   run.TestMetric,
		})
	}

	return ret, nil, nil
}
