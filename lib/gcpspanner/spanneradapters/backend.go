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

package spanneradapters

import (
	"context"
	"log/slog"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

type BackendSpannerClient interface {
	ListMetricsForFeatureIDBrowserAndChannel(
		ctx context.Context,
		featureID string,
		browser string,
		channel string,
		startAt time.Time,
		endAt time.Time,
		pageSize int,
		pageToken *string,
	) ([]gcpspanner.WPTRunFeatureMetricWithTime, *string, error)
	ListMetricsOverTimeWithAggregatedTotals(
		ctx context.Context,
		featureIDs []string,
		browser string,
		channel string,
		startAt, endAt time.Time,
		pageSize int,
		pageToken *string,
	) ([]gcpspanner.WPTRunAggregationMetricWithTime, *string, error)
	FeaturesSearch(
		ctx context.Context,
		pageToken *string,
		pageSize int,
		searchNode *searchtypes.SearchNode,
		sortOrder gcpspanner.Sortable,
	) (*gcpspanner.FeatureResultPage, error)
	GetFeature(
		ctx context.Context,
		filter gcpspanner.Filterable,
	) (*gcpspanner.FeatureResult, error)
	GetIDFromFeatureID(
		ctx context.Context,
		filter *gcpspanner.FeatureIDFilter,
	) (*string, error)
	ListBrowserFeatureCountMetric(
		ctx context.Context,
		browser string,
		startAt time.Time,
		endAt time.Time,
		pageSize int,
		pageToken *string,
	) (*gcpspanner.BrowserFeatureCountResultPage, error)
}

// Backend converts queries to spaner to useable entities for the backend
// service.
type Backend struct {
	client BackendSpannerClient
}

// NewBackend constructs an adapter for the backend service.
func NewBackend(client BackendSpannerClient) *Backend {
	return &Backend{client: client}
}

func (s *Backend) ListBrowserFeatureCountMetric(
	ctx context.Context,
	browser string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) (*backend.BrowserReleaseFeatureMetricsPage, error) {
	page, err := s.client.ListBrowserFeatureCountMetric(
		ctx,
		browser,
		startAt,
		endAt,
		pageSize,
		pageToken,
	)
	if err != nil {
		return nil, err
	}

	results := make([]backend.BrowserReleaseFeatureMetric, 0, len(page.Metrics))
	for idx := range page.Metrics {
		results = append(results, backend.BrowserReleaseFeatureMetric{
			Timestamp: page.Metrics[idx].ReleaseDate,
			Count:     &(page.Metrics[idx].FeatureCount),
		})
	}

	return &backend.BrowserReleaseFeatureMetricsPage{
		Metadata: &backend.PageMetadata{
			NextPageToken: page.NextPageToken,
		},
		Data: results,
	}, nil
}

func (s *Backend) ListMetricsOverTimeWithAggregatedTotals(
	ctx context.Context,
	featureIDs []string,
	browser string,
	channel string,
	startAt, endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]backend.WPTRunMetric, *string, error) {
	metrics, nextPageToken, err := s.client.ListMetricsOverTimeWithAggregatedTotals(
		ctx,
		featureIDs,
		browser,
		channel,
		startAt,
		endAt,
		pageSize,
		pageToken,
	)
	if err != nil {
		return nil, nil, err
	}

	// Convert the aggregate metric type to backend metrics
	backendMetrics := make([]backend.WPTRunMetric, 0, len(metrics))
	for _, metric := range metrics {
		backendMetrics = append(backendMetrics, backend.WPTRunMetric{
			RunTimestamp:    metric.TimeStart,
			TestPassCount:   metric.TestPass,
			TotalTestsCount: metric.TotalTests,
		})
	}

	return backendMetrics, nextPageToken, nil
}

func (s *Backend) ListMetricsForFeatureIDBrowserAndChannel(
	ctx context.Context,
	featureID string,
	browser string,
	channel string,
	startAt, endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]backend.WPTRunMetric, *string, error) {
	metrics, nextPageToken, err := s.client.ListMetricsForFeatureIDBrowserAndChannel(
		ctx,
		featureID,
		browser,
		channel,
		startAt,
		endAt,
		pageSize,
		pageToken,
	)
	if err != nil {
		return nil, nil, err
	}

	// Convert the feature metric type to backend metrics
	backendMetrics := make([]backend.WPTRunMetric, 0, len(metrics))
	for _, metric := range metrics {
		backendMetrics = append(backendMetrics, backend.WPTRunMetric{
			RunTimestamp:    metric.TimeStart,
			TestPassCount:   metric.TestPass,
			TotalTestsCount: metric.TotalTests,
		})
	}

	return backendMetrics, nextPageToken, nil
}

func convertBaselineStatusBackendToSpanner(status backend.FeatureBaselineStatus) gcpspanner.BaselineStatus {
	switch status {
	case backend.Widely:
		return gcpspanner.BaselineStatusHigh
	case backend.Newly:
		return gcpspanner.BaselineStatusLow
	case backend.Limited:
		return gcpspanner.BaselineStatusNone
	case backend.Undefined:
		fallthrough
	default:
		return gcpspanner.BaselineStatusUndefined
	}
}

func convertBaselineStatusSpannerToBackend(status gcpspanner.BaselineStatus) backend.FeatureBaselineStatus {
	switch status {
	case gcpspanner.BaselineStatusHigh:
		return backend.Widely
	case gcpspanner.BaselineStatusLow:
		return backend.Newly
	case gcpspanner.BaselineStatusNone:
		return backend.Limited
	case gcpspanner.BaselineStatusUndefined:
		fallthrough
	default:
		return backend.Undefined
	}
}

func (s *Backend) convertFeatureResult(featureResult *gcpspanner.FeatureResult) *backend.Feature {
	experimentalMetricsMap := make(map[string]backend.WPTFeatureData)
	for _, metric := range featureResult.ExperimentalMetrics {
		if metric.PassRate == nil {
			continue
		}
		passRate, _ := metric.PassRate.Float64()
		experimentalMetricsMap[metric.BrowserName] = backend.WPTFeatureData{
			Score: &passRate,
		}
	}
	stableMetricsMap := make(map[string]backend.WPTFeatureData)
	for _, metric := range featureResult.StableMetrics {
		passRate, _ := metric.PassRate.Float64()
		if metric.PassRate == nil {
			continue
		}
		stableMetricsMap[metric.BrowserName] = backend.WPTFeatureData{
			Score: &passRate,
		}
	}

	return &backend.Feature{
		FeatureId:      featureResult.FeatureID,
		Name:           featureResult.Name,
		BaselineStatus: convertBaselineStatusSpannerToBackend(gcpspanner.BaselineStatus(featureResult.Status)),
		Wpt: &backend.FeatureWPTSnapshots{
			Experimental: &experimentalMetricsMap,
			Stable:       &stableMetricsMap,
		},
		Spec:  nil,
		Usage: nil,
		// TODO(https://github.com/GoogleChrome/webstatus.dev/issues/160)
		BrowserImplementations: nil,
	}
}

func (s *Backend) FeaturesSearch(
	ctx context.Context,
	pageToken *string,
	pageSize int,
	searchNode *searchtypes.SearchNode,
	sortOrder *backend.GetV1FeaturesParamsSort,
) (*backend.FeaturePage, error) {
	spannerSortOrder := getFeatureSearchSortOrder(sortOrder)
	page, err := s.client.FeaturesSearch(ctx, pageToken, pageSize, searchNode, spannerSortOrder)
	if err != nil {
		return nil, err
	}

	results := make([]backend.Feature, 0, len(page.Features))
	for idx := range page.Features {

		results = append(results, *s.convertFeatureResult(&page.Features[idx]))
	}

	ret := &backend.FeaturePage{
		Metadata: backend.PageMetadataWithTotal{
			NextPageToken: page.NextPageToken,
			Total:         page.Total,
		},
		Data: results,
	}

	return ret, nil
}

func getFeatureSearchSortOrder(
	sortOrder *backend.GetV1FeaturesParamsSort) gcpspanner.Sortable {
	if sortOrder == nil {
		return gcpspanner.NewFeatureNameSort(true)
	}
	// nolint: exhaustive // Remove once we support all the cases.
	switch *sortOrder {
	case backend.NameAsc:
		return gcpspanner.NewFeatureNameSort(true)
	case backend.NameDesc:
		return gcpspanner.NewFeatureNameSort(false)
	case backend.BaselineStatusAsc:
		return gcpspanner.NewBaselineStatusSort(true)
	case backend.BaselineStatusDesc:
		return gcpspanner.NewBaselineStatusSort(false)
	}

	// Unknown sort order
	slog.Warn("unsupported sort order", "order", *sortOrder)

	return gcpspanner.NewFeatureNameSort(true)
}

func (s *Backend) GetFeature(
	ctx context.Context,
	featureID string,
) (*backend.Feature, error) {
	filter := gcpspanner.NewFeatureIDFilter(featureID)
	featureResult, err := s.client.GetFeature(ctx, filter)
	if err != nil {
		return nil, err
	}

	return s.convertFeatureResult(featureResult), nil
}

func (s *Backend) GetIDFromFeatureID(
	ctx context.Context,
	featureID string,
) (*string, error) {
	filter := gcpspanner.NewFeatureIDFilter(featureID)
	id, err := s.client.GetIDFromFeatureID(ctx, filter)
	if err != nil {
		return nil, err
	}

	return id, nil
}
