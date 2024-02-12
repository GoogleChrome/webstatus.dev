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
	"errors"
	"time"

	"cloud.google.com/go/datastore"
)

const wptRunMetricsGroupByFeatureKey = "WPTRunMetricsGroupByFeature"

// WPTRunMetricsGroupByFeature contains metrics for a given web feature in a
// WPT run.
type WPTRunMetricsGroupByFeature struct {
	WPTRunMetric
	FeatureID string `datastore:"web_feature_id"`
}

// wptMetricsByFeatureFilter implements Filterable to filter by web_feature_id.
// Compatible kinds:
// - wptRunMetricsGroupByFeatureKey.
type wptMetricsByFeatureFilter struct {
	featureID string
}

func (f wptMetricsByFeatureFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.FilterField("web_feature_id", "=", f.featureID)
}

// wptRunMetricsMerge implements Mergeable for WPTRunMetricsGroupByFeature.
type wptRunMetricsGroupByFeatureMerge struct{}

func (m wptRunMetricsGroupByFeatureMerge) Merge(
	existing *WPTRunMetricsGroupByFeature,
	new *WPTRunMetricsGroupByFeature) *WPTRunMetricsGroupByFeature {
	return &WPTRunMetricsGroupByFeature{
		WPTRunMetric: *wptRunMetricMerge{}.Merge(
			&existing.WPTRunMetric,
			&new.WPTRunMetric,
		),
		// The below fields cannot be overridden during a merge.
		FeatureID: existing.FeatureID,
	}
}

// StoreWPTRunMetricsForFeatures stores the metrics for a given web feature and run.
// Assumes that all the data belongs to a single run.
func (c *Client) StoreWPTRunMetricsForFeatures(
	ctx context.Context,
	run WPTRun,
	dataPerFeature map[string]WPTRunMetric) error {
	// Try to get the WPT Run first.
	_, err := c.GetWPTRun(ctx, run.RunID)
	if err != nil {
		return err
	}
	entityClient := entityClient[WPTRunMetricsGroupByFeature]{c}
	for featureID, featureData := range dataPerFeature {
		if featureData.RunID != run.RunID {
			return errors.New("feature data does not match run data")
		}
		err := entityClient.upsert(
			ctx,
			wptRunMetricsGroupByFeatureKey,
			&WPTRunMetricsGroupByFeature{
				WPTRunMetric: featureData,
				FeatureID:    featureID,
			},
			wptRunMetricsGroupByFeatureMerge{},
			wptRunIDFilter{
				runID: featureData.RunID,
			},
			wptMetricsByFeatureFilter{
				featureID: featureID,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetWPTMetricsByBrowserByFeature retrieves a list of metrics grouped by a
// web feature the given browser name and channel.
func (c *Client) GetWPTMetricsByBrowserByFeature(
	ctx context.Context,
	browser string,
	channel string,
	startAt time.Time,
	endAt time.Time,
	featureID string,
	pageToken *string) ([]*WPTRunMetricsGroupByFeature, *string, error) {
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

	entityClient := entityClient[WPTRunMetricsGroupByFeature]{c}

	return entityClient.list(
		ctx,
		wptRunMetricsGroupByFeatureKey,
		pageToken,
		wptMetricByRunIDs{
			runIDs: runIDs,
		},
		wptMetricsByFeatureFilter{
			featureID: featureID,
		},
	)
}
