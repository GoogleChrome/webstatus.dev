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

const wptRunMetricsGroupByFeatureKey = "WPTRunMetricsGroupByFeature"

// WPTRunMetricsGroupByFeature contains metrics for a given web feature in a
// WPT run.
type WPTRunMetricsGroupByFeature struct {
	WPTRunMetadata
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
		WPTRunMetadata: *wptRunMetadataMerge{}.Merge(
			&existing.WPTRunMetadata,
			&new.WPTRunMetadata,
		),
		// The below fields cannot be overridden during a merge.
		FeatureID: existing.FeatureID,
	}
}

// StoreWPTRunMetricsForFeatures stores the metrics for a given web feature and run.
func (c *Client) StoreWPTRunMetricsForFeatures(
	ctx context.Context,
	runMetadata WPTRunMetadata,
	dataPerFeature map[string]WPTRunMetric) error {
	entityClient := entityClient[WPTRunMetricsGroupByFeature]{c}
	for featureID, featureData := range dataPerFeature {
		err := entityClient.upsert(
			ctx,
			wptRunMetricsGroupByFeatureKey,
			&WPTRunMetricsGroupByFeature{
				WPTRunMetadata: runMetadata,
				WPTRunMetric:   featureData,
				FeatureID:      featureID,
			},
			wptRunMetricsGroupByFeatureMerge{},
			wptRunIDFilter{
				runID: runMetadata.RunID,
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
	entityClient := entityClient[WPTRunMetricsGroupByFeature]{c}

	return entityClient.list(
		ctx,
		wptRunMetricsGroupByFeatureKey,
		pageToken,
		wptMetricsByBrowserFilter{
			startAt: startAt,
			endAt:   endAt,
			browser: browser,
			channel: channel,
		},
		wptMetricsByFeatureFilter{
			featureID: featureID,
		},
		wptMetricsSortByTimeStart{},
	)
}
