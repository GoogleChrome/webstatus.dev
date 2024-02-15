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
)

// WPTRunMetricsGroupByFeature contains metrics for a given web feature in a
// WPT run.
type WPTRunMetricsGroupByFeature struct {
	WPTRunMetric
	FeatureID string `datastore:"web_feature_id"`
}

// wptRunFeatureTestMetricsMerge implements Mergeable for []WPTRunMetricsGroupByFeature.
type wptRunFeatureTestMetricsMerge struct{}

func (m wptRunFeatureTestMetricsMerge) Merge(
	existing *[]WPTRunMetricsGroupByFeature,
	new *[]WPTRunMetricsGroupByFeature) *[]WPTRunMetricsGroupByFeature {
	if existing == nil && new != nil {
		return new
	}
	if existing != nil && new == nil {
		return existing
	}
	if existing == nil && new == nil {
		return nil
	}
	metricNameMap := make(map[string]int) // Map feature name to index
	for idx, metric := range *existing {
		metricNameMap[metric.FeatureID] = idx
	}
	for newIdx := range *new {
		if idx, exists := metricNameMap[(*new)[newIdx].FeatureID]; exists {
			(*existing)[idx] = WPTRunMetricsGroupByFeature{
				WPTRunMetric: *wptRunMetricMerge{}.Merge(&(*existing)[idx].WPTRunMetric, &(*new)[newIdx].WPTRunMetric),
				// Do not override the feature ID.
				FeatureID: (*existing)[idx].FeatureID,
			}
		} else {
			// New item
			*existing = append(*existing, (*new)[newIdx])
		}
	}

	return existing
}

// StoreWPTRunMetricsForFeatures stores the metrics for a given web feature and run.
// Assumes that all the data belongs to a single run.
func (c *Client) StoreWPTRunMetricsForFeatures(
	ctx context.Context,
	runID int64,
	dataPerFeature map[string]WPTRunMetric) error {
	// Try to get the WPT Run first.
	run, err := c.GetWPTRun(ctx, runID)
	if err != nil {
		return err
	}
	featureTestMetrics := make([]WPTRunMetricsGroupByFeature, 0, len(dataPerFeature))
	for featureID, featureData := range dataPerFeature {
		featureTestMetrics = append(
			featureTestMetrics,
			WPTRunMetricsGroupByFeature{WPTRunMetric: featureData, FeatureID: featureID})
	}

	entityClient := entityClient[WPTRun]{c}

	run.FeatureTestMetrics = featureTestMetrics

	return entityClient.upsert(
		ctx,
		wptRunsKey,
		run,
		wptRunMerge{},
		wptRunIDFilter{runID: run.RunID},
	)
}

// GetWPTMetricsByBrowserByFeature retrieves a list of metrics grouped by a
// web feature the given browser name and channel.
func (c *Client) ListWPTMetricsByBrowserByFeature(
	ctx context.Context,
	browser string,
	channel string,
	startAt time.Time,
	endAt time.Time,
	featureID string,
	pageToken *string) ([]*WPTRunToMetricsByFeature, *string, error) {
	runs, _, err := c.ListWPTRunsByBrowser(ctx, browser, channel, startAt, endAt, pageToken)
	if err != nil {
		return nil, nil, err
	}

	ret := make([]*WPTRunToMetricsByFeature, 0, len(runs))
	for _, run := range runs {
		for i, metric := range run.FeatureTestMetrics {
			if metric.FeatureID == featureID {
				ret = append(ret, &WPTRunToMetricsByFeature{
					WPTRunMetadata: run.WPTRunMetadata,
					WPTRunMetric:   &run.FeatureTestMetrics[i].WPTRunMetric,
					FeatureID:      featureID,
				})
			}
		}

	}

	return ret, nil, nil
}
