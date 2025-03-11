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
	"errors"
	"log/slog"
	"math/big"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type BackendSpannerClient interface {
	ListMetricsForFeatureIDBrowserAndChannel(
		ctx context.Context,
		featureID string,
		browser string,
		channel string,
		metric gcpspanner.WPTMetricView,
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
		metric gcpspanner.WPTMetricView,
		startAt, endAt time.Time,
		pageSize int,
		pageToken *string,
	) ([]gcpspanner.WPTRunAggregationMetricWithTime, *string, error)
	ListChromiumDailyUsageStatsForFeatureID(
		ctx context.Context,
		featureID string,
		startAt, endAt time.Time,
		pageSize int,
		pageToken *string,
	) ([]gcpspanner.ChromiumDailyUsageStatWithDate, *string, error)
	FeaturesSearch(
		ctx context.Context,
		pageToken *string,
		pageSize int,
		searchNode *searchtypes.SearchNode,
		sortOrder gcpspanner.Sortable,
		wptMetricView gcpspanner.WPTMetricView,
		browsers []string,
	) (*gcpspanner.FeatureResultPage, error)
	GetFeature(
		ctx context.Context,
		filter gcpspanner.Filterable,
		wptMetricView gcpspanner.WPTMetricView,
		browsers []string,
	) (*gcpspanner.FeatureResult, error)
	GetIDFromFeatureKey(
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
	ListMissingOneImplCounts(
		ctx context.Context,
		targetBrowser string,
		otherBrowsers []string,
		startAt time.Time,
		endAt time.Time,
		pageSize int,
		pageToken *string,
	) (*gcpspanner.MissingOneImplCountPage, error)
	ListBaselineStatusCounts(
		ctx context.Context,
		dateType gcpspanner.BaselineDateType,
		startAt time.Time,
		endAt time.Time,
		pageSize int,
		pageToken *string,
	) (*gcpspanner.BaselineStatusCountResultPage, error)
	CreateNewUserSavedSearch(
		ctx context.Context,
		newSearch gcpspanner.CreateUserSavedSearchRequest) (*string, error)
	GetUserSavedSearch(
		ctx context.Context,
		savedSearchID string,
		authenticatedUserID *string) (*gcpspanner.UserSavedSearch, error)
	DeleteUserSavedSearch(ctx context.Context, req gcpspanner.DeleteUserSavedSearchRequest) error
}

// Backend converts queries to spanner to usable entities for the backend
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
		if errors.Is(err, gcpspanner.ErrInvalidCursorFormat) {
			return nil, errors.Join(err, backendtypes.ErrInvalidPageToken)
		}

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
	metricView backend.MetricViewPathParam,
	startAt, endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]backend.WPTRunMetric, *string, error) {
	metrics, nextPageToken, err := s.client.ListMetricsOverTimeWithAggregatedTotals(
		ctx,
		featureIDs,
		browser,
		channel,
		getSpannerWPTMetricView(metricView),
		startAt,
		endAt,
		pageSize,
		pageToken,
	)
	if err != nil {
		if errors.Is(err, gcpspanner.ErrInvalidCursorFormat) {
			return nil, nil, errors.Join(err, backendtypes.ErrInvalidPageToken)
		}

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
	metricView backend.MetricViewPathParam,
	startAt, endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]backend.WPTRunMetric, *string, error) {
	metrics, nextPageToken, err := s.client.ListMetricsForFeatureIDBrowserAndChannel(
		ctx,
		featureID,
		browser,
		channel,
		getSpannerWPTMetricView(metricView),
		startAt,
		endAt,
		pageSize,
		pageToken,
	)
	if err != nil {
		if errors.Is(err, gcpspanner.ErrInvalidCursorFormat) {
			return nil, nil, errors.Join(err, backendtypes.ErrInvalidPageToken)
		}

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

func (s *Backend) ListChromiumDailyUsageStats(
	ctx context.Context,
	featureID string,
	startAt, endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]backend.ChromiumUsageStat, *string, error) {
	metrics, nextPageToken, err := s.client.ListChromiumDailyUsageStatsForFeatureID(
		ctx,
		featureID,
		startAt,
		endAt,
		pageSize,
		pageToken,
	)
	if err != nil {
		if errors.Is(err, gcpspanner.ErrInvalidCursorFormat) {
			return nil, nil, errors.Join(err, backendtypes.ErrInvalidPageToken)
		}

		return nil, nil, err
	}

	// Convert the feature metric type to backend metrics
	backendStats := make([]backend.ChromiumUsageStat, 0, len(metrics))
	for _, stat := range metrics {
		var usage float64
		if stat.Usage != nil {
			usage, _ = stat.Usage.Float64()
		}
		backendStats = append(backendStats, backend.ChromiumUsageStat{
			Timestamp: stat.Date.In(time.UTC),
			Usage:     &usage,
		})
	}

	return backendStats, nextPageToken, nil
}

func (s *Backend) ListMissingOneImplCounts(
	ctx context.Context,
	targetBrowser string,
	otherBrowsers []string,
	startAt, endAt time.Time,
	pageSize int,
	pageToken *string,
) (*backend.BrowserReleaseFeatureMetricsPage, error) {
	spannerPage, err := s.client.ListMissingOneImplCounts(
		ctx,
		targetBrowser,
		otherBrowsers,
		startAt,
		endAt,
		pageSize,
		pageToken,
	)
	if err != nil {
		if errors.Is(err, gcpspanner.ErrInvalidCursorFormat) {
			return nil, errors.Join(err, backendtypes.ErrInvalidPageToken)
		}

		return nil, err
	}

	// Convert the feature metric type to backend metrics
	backendData := make([]backend.BrowserReleaseFeatureMetric, 0, len(spannerPage.Metrics))
	for _, metric := range spannerPage.Metrics {
		backendData = append(backendData, backend.BrowserReleaseFeatureMetric{
			Timestamp: metric.EventReleaseDate,
			Count:     &metric.Count,
		})
	}

	return &backend.BrowserReleaseFeatureMetricsPage{
		Metadata: &backend.PageMetadata{
			NextPageToken: spannerPage.NextPageToken,
		},
		Data: backendData,
	}, nil
}

func (s *Backend) ListBaselineStatusCounts(
	ctx context.Context,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) (*backend.BaselineStatusMetricsPage, error) {
	spannerPage, err := s.client.ListBaselineStatusCounts(
		ctx,
		// For now, base it on the low date
		gcpspanner.BaselineDateTypeLow,
		startAt,
		endAt,
		pageSize,
		pageToken,
	)
	if err != nil {
		if errors.Is(err, gcpspanner.ErrInvalidCursorFormat) {
			return nil, errors.Join(err, backendtypes.ErrInvalidPageToken)
		}

		return nil, err
	}

	// Convert the metric type to backend metrics
	backendData := make([]backend.BaselineStatusMetric, 0, len(spannerPage.Metrics))
	for _, metric := range spannerPage.Metrics {
		backendData = append(backendData, backend.BaselineStatusMetric{
			Timestamp: metric.Date,
			Count:     &metric.StatusCount,
		})
	}

	return &backend.BaselineStatusMetricsPage{
		Metadata: &backend.PageMetadata{
			NextPageToken: spannerPage.NextPageToken,
		},
		Data: backendData,
	}, nil
}

func (s *Backend) CreateUserSavedSearch(ctx context.Context, userID string,
	savedSearch backend.SavedSearch) (*backend.SavedSearchResponse, error) {
	output, err := s.client.CreateNewUserSavedSearch(ctx, gcpspanner.CreateUserSavedSearchRequest{
		OwnerUserID: userID,
		Query:       savedSearch.Query,
		Name:        savedSearch.Name,
		Description: savedSearch.Description,
	})
	if err != nil {
		if errors.Is(err, gcpspanner.ErrOwnerSavedSearchLimitExceeded) {
			return nil, errors.Join(err, backendtypes.ErrUserMaxSavedSearches)
		}

		return nil, err
	}

	createdSavedSearch, err := s.client.GetUserSavedSearch(ctx, *output, &userID)
	if err != nil {
		return nil, err
	}

	return &backend.SavedSearchResponse{
		Id:          *output,
		CreatedAt:   createdSavedSearch.CreatedAt,
		UpdatedAt:   createdSavedSearch.UpdatedAt,
		Name:        createdSavedSearch.Name,
		Query:       createdSavedSearch.Query,
		Description: createdSavedSearch.Description,
	}, nil
}

func (s *Backend) DeleteUserSavedSearch(ctx context.Context, userID, savedSearchID string) error {
	err := s.client.DeleteUserSavedSearch(ctx, gcpspanner.DeleteUserSavedSearchRequest{
		SavedSearchID:    savedSearchID,
		RequestingUserID: userID,
	})
	if err != nil {
		if errors.Is(err, gcpspanner.ErrMissingRequiredRole) {
			return errors.Join(err, backendtypes.ErrUserNotAuthorizedForAction)
		} else if errors.Is(err, gcpspanner.ErrQueryReturnedNoResults) {
			return errors.Join(err, backendtypes.ErrEntityDoesNotExist)
		}

		return err
	}

	return nil
}

func convertBaselineStatusBackendToSpanner(status backend.BaselineInfoStatus) gcpspanner.BaselineStatus {
	switch status {
	case backend.Widely:
		return gcpspanner.BaselineStatusHigh
	case backend.Newly:
		return gcpspanner.BaselineStatusLow
	case backend.Limited:
		return gcpspanner.BaselineStatusNone
	}

	return ""
}

func valuePtr[T any](in T) *T { return &in }

func convertBaselineSpannerToBackend(strStatus *string,
	lowDate, highDate *time.Time) *backend.BaselineInfo {
	var ret *backend.BaselineInfo

	var status gcpspanner.BaselineStatus
	if strStatus != nil {
		status = gcpspanner.BaselineStatus(*strStatus)
	}
	var backendStatus *backend.BaselineInfoStatus
	switch status {
	case gcpspanner.BaselineStatusHigh:
		backendStatus = valuePtr(backend.Widely)
	case gcpspanner.BaselineStatusLow:
		backendStatus = valuePtr(backend.Newly)
	case gcpspanner.BaselineStatusNone:
		backendStatus = valuePtr(backend.Limited)
	}
	var retLowDate, retHighDate *openapi_types.Date
	if lowDate != nil {
		retLowDate = &openapi_types.Date{Time: *lowDate}
	}

	if highDate != nil {
		retHighDate = &openapi_types.Date{Time: *highDate}
	}

	if backendStatus != nil || retLowDate != nil || retHighDate != nil {
		ret = &backend.BaselineInfo{
			Status:   backendStatus,
			LowDate:  retLowDate,
			HighDate: retHighDate,
		}
	}

	return ret
}

func convertChromiumUsageToBackend(chromiumUsage *big.Rat) *backend.ChromiumUsageInfo {
	ret := &backend.ChromiumUsageInfo{
		Daily: nil,
	}
	if chromiumUsage != nil {
		usage, _ := chromiumUsage.Float64()
		ret.Daily = &usage
	}

	return ret
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

func convertMetrics(
	metrics []*gcpspanner.FeatureResultMetric,
	wpt *backend.FeatureWPTSnapshots,
	experimental bool) *backend.FeatureWPTSnapshots {
	metricsMap := make(map[string]backend.WPTFeatureData)
	for idx, metric := range metrics {
		if metric.PassRate == nil && metric.FeatureRunDetails == nil {
			continue
		}
		data := backend.WPTFeatureData{
			Metadata: nil,
			Score:    nil,
		}
		if metric.PassRate != nil {
			passRate, _ := metric.PassRate.Float64()
			data.Score = &passRate
		}
		if metric.FeatureRunDetails != nil {
			data.Metadata = &metrics[idx].FeatureRunDetails
		}
		metricsMap[metric.BrowserName] = data
	}

	if len(metricsMap) > 0 {
		// The database implementation should only return metrics that have PassRate.
		// The logic below is only proactive in case something changes where we return
		// a BrowserName without a PassRate. This will prevent the code from returning
		// an initialized, but empty map by only overriding the default map when it actually
		// has a value.
		wpt = cmp.Or(wpt, &backend.FeatureWPTSnapshots{
			Experimental: nil,
			Stable:       nil,
		})
		if experimental {
			wpt.Experimental = &metricsMap
		} else {
			wpt.Stable = &metricsMap
		}
	}

	return wpt
}

func (s *Backend) convertFeatureResult(featureResult *gcpspanner.FeatureResult) *backend.Feature {
	// Initialize the returned feature with the default values.
	// The logic below will fill in nullable fields.
	ret := &backend.Feature{
		FeatureId: featureResult.FeatureKey,
		Name:      featureResult.Name,
		Baseline: convertBaselineSpannerToBackend(
			featureResult.Status,
			featureResult.LowDate,
			featureResult.HighDate,
		),
		Wpt:  nil,
		Spec: nil,
		Usage: &backend.BrowserUsage{
			Chromium: convertChromiumUsageToBackend(featureResult.ChromiumUsage),
		},
		BrowserImplementations: nil,
	}

	if len(featureResult.ExperimentalMetrics) > 0 {
		ret.Wpt = convertMetrics(featureResult.ExperimentalMetrics, ret.Wpt, true)
	}

	if len(featureResult.StableMetrics) > 0 {
		ret.Wpt = convertMetrics(featureResult.StableMetrics, ret.Wpt, false)
	}

	if len(featureResult.ImplementationStatuses) > 0 {
		implementationMap := make(map[string]backend.BrowserImplementation, len(featureResult.ImplementationStatuses))
		for _, status := range featureResult.ImplementationStatuses {
			backendStatus := convertImplementationStatusToBackend(status.ImplementationStatus)
			var date *openapi_types.Date
			if status.ImplementationDate != nil {
				date = &openapi_types.Date{Time: *status.ImplementationDate}
			}
			implementationMap[status.BrowserName] = backend.BrowserImplementation{
				Status:  &backendStatus,
				Date:    date,
				Version: status.ImplementationVersion,
			}
		}
		ret.BrowserImplementations = &implementationMap
	}

	if len(featureResult.SpecLinks) > 0 {
		links := make([]backend.SpecLink, 0, len(featureResult.SpecLinks))
		for idx := range featureResult.SpecLinks {
			links = append(links, backend.SpecLink{
				Link: &featureResult.SpecLinks[idx],
			})
		}
		ret.Spec = &backend.FeatureSpecInfo{
			Links: &links,
		}
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

type BrowserList []backend.BrowserPathParam

func (b BrowserList) ToStringList() []string {
	if len(b) == 0 {
		return nil
	}
	ret := make([]string, 0, len(b))
	for _, browser := range b {
		ret = append(ret, string(browser))
	}

	return ret
}

func (s *Backend) FeaturesSearch(
	ctx context.Context,
	pageToken *string,
	pageSize int,
	searchNode *searchtypes.SearchNode,
	sortOrder *backend.ListFeaturesParamsSort,
	wptMetricView backend.WPTMetricView,
	browsers []backend.BrowserPathParam,
) (*backend.FeaturePage, error) {
	spannerSortOrder := getFeatureSearchSortOrder(sortOrder)
	page, err := s.client.FeaturesSearch(ctx, pageToken, pageSize, searchNode,
		spannerSortOrder, getSpannerWPTMetricView(wptMetricView),
		BrowserList(browsers).ToStringList())
	if err != nil {
		if errors.Is(err, gcpspanner.ErrInvalidCursorFormat) {
			return nil, errors.Join(err, backendtypes.ErrInvalidPageToken)
		}

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

// TODO: Pass in context to be used by slog.ErrorContext.
// nolint: gocyclo // WONTFIX. Keep all the cases here so that the exhaustive
// linter can catch a missing case.
func getFeatureSearchSortOrder(
	sortOrder *backend.ListFeaturesParamsSort) gcpspanner.Sortable {
	if sortOrder == nil {
		return gcpspanner.NewBaselineStatusSort(false)
	}
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
	case backend.ChromiumUsageAsc:
		return gcpspanner.NewChromiumUsageSort(true)
	case backend.ChromiumUsageDesc:
		return gcpspanner.NewChromiumUsageSort(false)
	case backend.AvailabilityChromeAsc:
		return gcpspanner.NewBrowserFeatureSupportSort(true, string(backend.Chrome))
	case backend.AvailabilityChromeDesc:
		return gcpspanner.NewBrowserFeatureSupportSort(false, string(backend.Chrome))
	case backend.AvailabilityEdgeAsc:
		return gcpspanner.NewBrowserFeatureSupportSort(true, string(backend.Edge))
	case backend.AvailabilityEdgeDesc:
		return gcpspanner.NewBrowserFeatureSupportSort(false, string(backend.Edge))
	case backend.AvailabilityFirefoxAsc:
		return gcpspanner.NewBrowserFeatureSupportSort(true, string(backend.Firefox))
	case backend.AvailabilityFirefoxDesc:
		return gcpspanner.NewBrowserFeatureSupportSort(false, string(backend.Firefox))
	case backend.AvailabilitySafariAsc:
		return gcpspanner.NewBrowserFeatureSupportSort(true, string(backend.Safari))
	case backend.AvailabilitySafariDesc:
		return gcpspanner.NewBrowserFeatureSupportSort(false, string(backend.Safari))
	}

	// Unknown sort order
	slog.Warn("unsupported sort order", "order", *sortOrder)

	return gcpspanner.NewBaselineStatusSort(false)
}

func (s *Backend) GetFeature(
	ctx context.Context,
	featureID string,
	wptMetricView backend.WPTMetricView,
	browsers []backend.BrowserPathParam,
) (*backend.Feature, error) {
	filter := gcpspanner.NewFeatureKeyFilter(featureID)
	featureResult, err := s.client.GetFeature(ctx, filter, getSpannerWPTMetricView(wptMetricView),
		BrowserList(browsers).ToStringList())
	if err != nil {
		return nil, err
	}

	return s.convertFeatureResult(featureResult), nil
}

func (s *Backend) GetIDFromFeatureKey(
	ctx context.Context,
	featureID string,
) (*string, error) {
	filter := gcpspanner.NewFeatureKeyFilter(featureID)
	id, err := s.client.GetIDFromFeatureKey(ctx, filter)
	if err != nil {
		return nil, err
	}

	return id, nil
}
