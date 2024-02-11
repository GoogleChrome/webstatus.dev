package gds

import (
	"context"
	"time"

	"cloud.google.com/go/datastore"
)

const wptRunMetricsGroupByFeatureKey = "WPTRunMetricsGroupByFeature"

type WPTRunMetricsGroupByFeature struct {
	WPTRunMetadata
	WPTTestMetric
	FeatureID string `datastore:"web_feature_id"`
}

type wptMetricsByFeatureFilter struct {
	featureID string
}

func (f wptMetricsByFeatureFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.FilterField("web_feature_id", "=", f.featureID)
}

func (c *Client) StoreWPTMetricsForFeatures(
	ctx context.Context,
	runMetadata WPTRunMetadata,
	dataPerFeature map[string]WPTTestMetric) error {
	entityClient := entityClient[WPTRunMetricsGroupByFeature]{c}
	for featureID, featureData := range dataPerFeature {
		err := entityClient.upsert(
			ctx,
			wptRunMetricsGroupByFeatureKey,
			&WPTRunMetricsGroupByFeature{
				WPTRunMetadata: runMetadata,
				WPTTestMetric:  featureData,
				FeatureID:      featureID,
			},
			wptRunsMetricsFilter{
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

func (c *Client) GetWPTMetricsByBrowserByFeature(
	ctx context.Context,
	browser string,
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
		},
		wptMetricsByFeatureFilter{
			featureID: featureID,
		},
		wptMetricsSortByTimeStart{},
	)
}
