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
	"cmp"
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
		wptMetricView gcpspanner.WPTMetricView,
	) (*gcpspanner.FeatureResultPage, error)
	GetFeature(
		ctx context.Context,
		filter gcpspanner.Filterable,
		wptMetricView gcpspanner.WPTMetricView,
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

func convertImplementationStatusToBackend(
	status gcpspanner.BrowserImplementationStatus) backend.BrowserImplementationStatus {
	switch status {
	case gcpspanner.Available:
		return backend.Available
	case gcpspanner.Unavailable:
		return backend.Unavailable
	}

	return backend.Unavailable
}

func (s *Backend) convertFeatureResult(featureResult *gcpspanner.FeatureResult) *backend.Feature {
	// Initialize the returned feature with the default values.
	// The logic below will fill in nullable fields.
	ret := &backend.Feature{
		FeatureId:              featureResult.FeatureID,
		Name:                   featureResult.Name,
		BaselineStatus:         convertBaselineStatusSpannerToBackend(gcpspanner.BaselineStatus(featureResult.Status)),
		Wpt:                    nil,
		Spec:                   nil,
		Usage:                  nil,
		BrowserImplementations: nil,
	}

	if len(featureResult.ExperimentalMetrics) > 0 {
		experimentalMetricsMap := make(map[string]backend.WPTFeatureData, len(featureResult.ExperimentalMetrics))
		for _, metric := range featureResult.ExperimentalMetrics {
			if metric.PassRate == nil {
				continue
			}
			passRate, _ := metric.PassRate.Float64()
			experimentalMetricsMap[metric.BrowserName] = backend.WPTFeatureData{
				Score: &passRate,
			}
		}

		// The database implementation should only return metrics that have PassRate.
		// The logic below is only proactive in case something changes where we return
		// a BrowserName without a PassRate. This will prevent the code from returning
		// an initialized, but empty map by only overriding the default map when it actually
		// has a value.
		if len(experimentalMetricsMap) > 0 {
			wpt := cmp.Or(ret.Wpt, &backend.FeatureWPTSnapshots{
				Stable:       nil,
				Experimental: nil,
			})
			wpt.Experimental = &experimentalMetricsMap
			ret.Wpt = wpt
		}
	}

	if len(featureResult.StableMetrics) > 0 {
		stableMetricsMap := make(map[string]backend.WPTFeatureData, len(featureResult.StableMetrics))
		for _, metric := range featureResult.StableMetrics {
			if metric.PassRate == nil {
				continue
			}
			passRate, _ := metric.PassRate.Float64()
			stableMetricsMap[metric.BrowserName] = backend.WPTFeatureData{
				Score: &passRate,
			}
		}

		// The database implementation should only return metrics that have PassRate.
		// The logic below is only proactive in case something changes where we return
		// a BrowserName without a PassRate. This will prevent the code from returning
		// an initialized, but empty map by only overriding the default map when it actually
		// has a value.
		if len(stableMetricsMap) > 0 {
			wpt := cmp.Or(ret.Wpt, &backend.FeatureWPTSnapshots{
				Stable:       nil,
				Experimental: nil,
			})
			wpt.Stable = &stableMetricsMap
			ret.Wpt = wpt
		}
	}

	if len(featureResult.ImplementationStatuses) > 0 {
		implementationMap := make(map[string]backend.BrowserImplementation, len(featureResult.ImplementationStatuses))
		for _, status := range featureResult.ImplementationStatuses {
			backendStatus := convertImplementationStatusToBackend(status.ImplementationStatus)
			implementationMap[status.BrowserName] = backend.BrowserImplementation{
				Status: &backendStatus,
			}
		}
		ret.BrowserImplementations = &implementationMap
	}

	return ret
}

func getSpannerWPTMetricView(wptMetricView backend.WPTMetricView) gcpspanner.WPTMetricView {
	switch wptMetricView {
	case backend.SubtestCounts:
		return gcpspanner.WPTSubtestView
	case backend.TestCounts:
		return gcpspanner.WPTTestView
	}

	// Default to subtest view for unknown
	return gcpspanner.WPTSubtestView
}

func (s *Backend) FeaturesSearch(
	ctx context.Context,
	pageToken *string,
	pageSize int,
	searchNode *searchtypes.SearchNode,
	sortOrder *backend.GetV1FeaturesParamsSort,
	wptMetricView backend.WPTMetricView,
) (*backend.FeaturePage, error) {
	spannerSortOrder := getFeatureSearchSortOrder(sortOrder)
	page, err := s.client.FeaturesSearch(ctx, pageToken, pageSize, searchNode,
		spannerSortOrder, getSpannerWPTMetricView(wptMetricView))
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
	case backend.ExperimentalChromeAsc:
		return gcpspanner.NewBrowserImplSort(true, string(backend.Chrome), false)
	case backend.ExperimentalChromeDesc:
		return gcpspanner.NewBrowserImplSort(false, string(backend.Chrome), false)
	case backend.ExperimentalEdgeAsc:
		return gcpspanner.NewBrowserImplSort(true, string(backend.Edge), false)
	case backend.ExperimentalEdgeDesc:
		return gcpspanner.NewBrowserImplSort(false, string(backend.Edge), false)
	case backend.ExperimentalFirefoxAsc:
		return gcpspanner.NewBrowserImplSort(true, string(backend.Firefox), false)
	case backend.ExperimentalFirefoxDesc:
		return gcpspanner.NewBrowserImplSort(false, string(backend.Firefox), false)
	case backend.ExperimentalSafariAsc:
		return gcpspanner.NewBrowserImplSort(true, string(backend.Safari), false)
	case backend.ExperimentalSafariDesc:
		return gcpspanner.NewBrowserImplSort(false, string(backend.Safari), false)
	case backend.StableChromeAsc:
		return gcpspanner.NewBrowserImplSort(true, string(backend.Chrome), true)
	case backend.StableChromeDesc:
		return gcpspanner.NewBrowserImplSort(false, string(backend.Chrome), true)
	case backend.StableEdgeAsc:
		return gcpspanner.NewBrowserImplSort(true, string(backend.Edge), true)
	case backend.StableEdgeDesc:
		return gcpspanner.NewBrowserImplSort(false, string(backend.Edge), true)
	case backend.StableFirefoxAsc:
		return gcpspanner.NewBrowserImplSort(true, string(backend.Firefox), true)
	case backend.StableFirefoxDesc:
		return gcpspanner.NewBrowserImplSort(false, string(backend.Firefox), true)
	case backend.StableSafariAsc:
		return gcpspanner.NewBrowserImplSort(true, string(backend.Safari), true)
	case backend.StableSafariDesc:
		return gcpspanner.NewBrowserImplSort(false, string(backend.Safari), true)
	}

	// Unknown sort order
	slog.Warn("unsupported sort order", "order", *sortOrder)

	return gcpspanner.NewFeatureNameSort(true)
}

func (s *Backend) GetFeature(
	ctx context.Context,
	featureID string,
	wptMetricView backend.WPTMetricView,
) (*backend.Feature, error) {
	filter := gcpspanner.NewFeatureIDFilter(featureID)
	featureResult, err := s.client.GetFeature(ctx, filter, getSpannerWPTMetricView(wptMetricView))
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
