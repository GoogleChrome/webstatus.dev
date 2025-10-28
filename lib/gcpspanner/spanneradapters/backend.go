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
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"slices"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
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
	ListChromeDailyUsageStatsForFeatureID(
		ctx context.Context,
		featureID string,
		startAt, endAt time.Time,
		pageSize int,
		pageToken *string,
	) ([]gcpspanner.ChromeDailyUsageStatWithDate, *string, error)
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
	GetMovedWebFeatureDetailsByOriginalFeatureKey(
		ctx context.Context,
		featureKey string,
	) (*gcpspanner.MovedWebFeature, error)
	GetSplitWebFeatureByOriginalFeatureKey(
		ctx context.Context,
		featureKey string,
	) (*gcpspanner.SplitWebFeature, error)
	GetIDFromFeatureKey(
		ctx context.Context,
		filter *gcpspanner.FeatureIDFilter,
	) (*string, error)
	ListBrowserFeatureCountMetric(
		ctx context.Context,
		targetBrowser string,
		targetMobileBrowser *string,
		startAt time.Time,
		endAt time.Time,
		pageSize int,
		pageToken *string,
	) (*gcpspanner.BrowserFeatureCountResultPage, error)
	ListMissingOneImplCounts(
		ctx context.Context,
		targetBrowser string,
		targetMobileBrowser *string,
		otherBrowsers []string,
		startAt time.Time,
		endAt time.Time,
		pageSize int,
		pageToken *string,
	) (*gcpspanner.MissingOneImplCountPage, error)
	ListMissingOneImplementationFeatures(
		ctx context.Context,
		targetBrowser string,
		targetMobileBrowser *string,
		otherBrowsers []string,
		targetDate time.Time,
		pageSize int,
		pageToken *string,
	) (*gcpspanner.MissingOneImplFeatureListPage, error)
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
	ListUserSavedSearches(
		ctx context.Context,
		userID string,
		pageSize int,
		pageToken *string) (*gcpspanner.UserSavedSearchesPage, error)
	UpdateUserSavedSearch(ctx context.Context, req gcpspanner.UpdateSavedSearchRequest) error
	AddUserSearchBookmark(ctx context.Context, req gcpspanner.UserSavedSearchBookmark) error
	DeleteUserSearchBookmark(ctx context.Context, req gcpspanner.UserSavedSearchBookmark) error
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
	targetBrowser string,
	targetMobileBrowser *string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) (*backend.BrowserReleaseFeatureMetricsPage, error) {
	page, err := s.client.ListBrowserFeatureCountMetric(
		ctx,
		targetBrowser,
		targetMobileBrowser,
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

func (s *Backend) ListChromeDailyUsageStats(
	ctx context.Context,
	featureID string,
	startAt, endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]backend.ChromeUsageStat, *string, error) {
	metrics, nextPageToken, err := s.client.ListChromeDailyUsageStatsForFeatureID(
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
	backendStats := make([]backend.ChromeUsageStat, 0, len(metrics))
	for _, stat := range metrics {
		var usage float64
		if stat.Usage != nil {
			usage, _ = stat.Usage.Float64()
		}
		backendStats = append(backendStats, backend.ChromeUsageStat{
			Timestamp: stat.Date.In(time.UTC),
			Usage:     &usage,
		})
	}

	return backendStats, nextPageToken, nil
}

func (s *Backend) ListMissingOneImplCounts(
	ctx context.Context,
	targetBrowser string,
	targetMobileBrowser *string,
	otherBrowsers []string,
	startAt, endAt time.Time,
	pageSize int,
	pageToken *string,
) (*backend.BrowserReleaseFeatureMetricsPage, error) {
	spannerPage, err := s.client.ListMissingOneImplCounts(
		ctx,
		targetBrowser,
		targetMobileBrowser,
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

func (s *Backend) ListMissingOneImplementationFeatures(
	ctx context.Context,
	targetBrowser string,
	targetMobileBrowser *string,
	otherBrowsers []string,
	targetDate time.Time,
	pageSize int,
	pageToken *string,
) (*backend.MissingOneImplFeaturesPage, error) {
	spannerPage, err := s.client.ListMissingOneImplementationFeatures(
		ctx,
		targetBrowser,
		targetMobileBrowser,
		otherBrowsers,
		targetDate,
		pageSize,
		pageToken,
	)
	if err != nil {
		if errors.Is(err, gcpspanner.ErrInvalidCursorFormat) {
			return nil, errors.Join(err, backendtypes.ErrInvalidPageToken)
		}

		return nil, err
	}

	// Convert it to backend []MissingOneImplFeature
	backendData := make([]backend.MissingOneImplFeature, 0, len(spannerPage.FeatureList))
	for _, featureID := range spannerPage.FeatureList {
		backendData = append(backendData, backend.MissingOneImplFeature{
			FeatureId: &featureID.WebFeatureID,
		})
	}

	return &backend.MissingOneImplFeaturesPage{
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

	return convertUserSavedSearchToSavedSearchResponse(createdSavedSearch), nil
}

func (s *Backend) ListUserSavedSearches(
	ctx context.Context,
	userID string,
	pageSize int,
	pageToken *string,
) (*backend.UserSavedSearchPage, error) {
	page, err := s.client.ListUserSavedSearches(ctx, userID, pageSize, pageToken)
	if err != nil {
		if errors.Is(err, gcpspanner.ErrInvalidCursorFormat) {
			return nil, errors.Join(err, backendtypes.ErrInvalidPageToken)
		}

		return nil, err
	}
	var metadata *backend.PageMetadata
	if page.NextPageToken != nil {
		metadata = &backend.PageMetadata{
			NextPageToken: page.NextPageToken,
		}
	}
	var results *[]backend.SavedSearchResponse
	if len(page.Searches) > 0 {
		data := make([]backend.SavedSearchResponse, 0, len(page.Searches))
		for _, savedSearch := range page.Searches {
			resp := convertUserSavedSearchToSavedSearchResponse(&savedSearch)
			data = append(data, *resp)
		}
		results = &data
	}

	return &backend.UserSavedSearchPage{
		Metadata: metadata,
		Data:     results,
	}, nil
}

func convertSavedSearchIsBookmarkedFromGCP(isBookmarked *bool) *backend.UserSavedSearchBookmark {
	if isBookmarked == nil {
		return nil
	}

	status := backend.BookmarkNone
	if *isBookmarked {
		status = backend.BookmarkActive
	}

	return &backend.UserSavedSearchBookmark{
		Status: status,
	}
}

// Roles can be found in lib/gcpspanner/saved_search_user_roles.go.
func convertSavedSearchRoleFromGCP(role *string) *backend.UserSavedSearchPermissions {
	if role == nil {
		return nil
	}

	switch gcpspanner.SavedSearchRole(*role) {
	case gcpspanner.SavedSearchOwner:
		return &backend.UserSavedSearchPermissions{
			Role: valuePtr(backend.SavedSearchOwner),
		}
	}

	return nil
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

func (s *Backend) GetSavedSearch(ctx context.Context, savedSearchID string, userID *string) (
	*backend.SavedSearchResponse, error) {
	savedSearch, err := s.client.GetUserSavedSearch(ctx, savedSearchID, userID)
	if err != nil {
		if errors.Is(err, gcpspanner.ErrQueryReturnedNoResults) {
			return nil, errors.Join(err, backendtypes.ErrEntityDoesNotExist)
		}

		return nil, err
	}

	return convertUserSavedSearchToSavedSearchResponse(savedSearch), nil
}

func buildUpdateSavedSearchRequestForGCP(savedSearchID string,
	userID string,
	updateRequest *backend.SavedSearchUpdateRequest) gcpspanner.UpdateSavedSearchRequest {
	req := gcpspanner.UpdateSavedSearchRequest{
		ID:       savedSearchID,
		AuthorID: userID,
		Query: gcpspanner.OptionallySet[string]{
			IsSet: false,
			Value: "",
		},
		Name: gcpspanner.OptionallySet[string]{
			IsSet: false,
			Value: "",
		},
		Description: gcpspanner.OptionallySet[*string]{
			IsSet: false,
			Value: nil,
		},
	}
	if slices.Contains(updateRequest.UpdateMask, backend.SavedSearchUpdateRequestMaskName) {
		req.Name.IsSet = true
		req.Name.Value = *updateRequest.Name
	}
	if slices.Contains(updateRequest.UpdateMask, backend.SavedSearchUpdateRequestMaskQuery) {
		req.Query.IsSet = true
		req.Query.Value = *updateRequest.Query
	}

	if slices.Contains(updateRequest.UpdateMask, backend.SavedSearchUpdateRequestMaskDescription) {
		req.Description.IsSet = true
		req.Description.Value = updateRequest.Description
	}

	return req

}
func (s *Backend) UpdateUserSavedSearch(
	ctx context.Context,
	savedSearchID string,
	userID string,
	updateRequest *backend.SavedSearchUpdateRequest,
) (*backend.SavedSearchResponse, error) {
	req := buildUpdateSavedSearchRequestForGCP(savedSearchID, userID, updateRequest)

	err := s.client.UpdateUserSavedSearch(ctx, req)
	if err != nil {
		if errors.Is(err, gcpspanner.ErrMissingRequiredRole) {
			return nil, errors.Join(err, backendtypes.ErrUserNotAuthorizedForAction)
		} else if errors.Is(err, gcpspanner.ErrQueryReturnedNoResults) {
			return nil, errors.Join(err, backendtypes.ErrEntityDoesNotExist)
		}

		return nil, err
	}

	savedSearch, err := s.client.GetUserSavedSearch(ctx, savedSearchID, &userID)
	if err != nil {
		if errors.Is(err, gcpspanner.ErrQueryReturnedNoResults) {
			// Highly unlikely that another user would delete it in this small time frame but rather be thorough
			// with the possible errors from GetUserSavedSearch.
			return nil, errors.Join(err, backendtypes.ErrEntityDoesNotExist)
		}

		return nil, err
	}

	return convertUserSavedSearchToSavedSearchResponse(savedSearch), nil
}

func (s *Backend) PutUserSavedSearchBookmark(
	ctx context.Context,
	userID string,
	savedSearchID string,
) error {
	err := s.client.AddUserSearchBookmark(ctx, gcpspanner.UserSavedSearchBookmark{
		UserID:        userID,
		SavedSearchID: savedSearchID,
	})
	if err != nil {
		if errors.Is(err, gcpspanner.ErrUserSearchBookmarkLimitExceeded) {
			return errors.Join(err, backendtypes.ErrUserMaxBookmarks)
		} else if errors.Is(err, gcpspanner.ErrQueryReturnedNoResults) {
			return errors.Join(err, backendtypes.ErrEntityDoesNotExist)
		}

		return err
	}

	return nil
}

func (s *Backend) RemoveUserSavedSearchBookmark(
	ctx context.Context,
	userID string,
	savedSearchID string,
) error {
	err := s.client.DeleteUserSearchBookmark(ctx, gcpspanner.UserSavedSearchBookmark{
		UserID:        userID,
		SavedSearchID: savedSearchID,
	})
	if err != nil {
		if errors.Is(err, gcpspanner.ErrOwnerCannotDeleteBookmark) {
			return errors.Join(err, backendtypes.ErrUserNotAuthorizedForAction)
		} else if errors.Is(err, gcpspanner.ErrQueryReturnedNoResults) {
			return errors.Join(err, backendtypes.ErrEntityDoesNotExist)
		}

		return err
	}

	return nil
}

func convertUserSavedSearchToSavedSearchResponse(savedSearch *gcpspanner.UserSavedSearch) *backend.SavedSearchResponse {
	return &backend.SavedSearchResponse{
		Id:             savedSearch.ID,
		CreatedAt:      savedSearch.CreatedAt,
		UpdatedAt:      savedSearch.UpdatedAt,
		Name:           savedSearch.Name,
		Query:          savedSearch.Query,
		Description:    savedSearch.Description,
		BookmarkStatus: convertSavedSearchIsBookmarkedFromGCP(savedSearch.IsBookmarked),
		Permissions:    convertSavedSearchRoleFromGCP(savedSearch.Role),
	}
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

func convertChromeUsageToBackend(chromeUsage *big.Rat) *backend.ChromeUsageInfo {
	ret := &backend.ChromeUsageInfo{
		Daily: nil,
	}
	if chromeUsage != nil {
		usage, _ := chromeUsage.Float64()
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

type spannerDeveloperSignals struct {
	Upvotes *int64
	Link    *string
}

func convertFeatureDeveloperSignals(signals spannerDeveloperSignals) *backend.FeatureDeveloperSignals {
	if signals.Upvotes == nil && signals.Link == nil {
		return nil
	}

	return &backend.FeatureDeveloperSignals{
		Upvotes: signals.Upvotes,
		Link:    signals.Link,
	}
}

// NullJSONToTypedSlice safely converts a spanner.NullJSON object, which the Spanner
// client decodes into a generic `interface{}`, into a strongly-typed slice of type `T`.
//
// A direct type assertion is not feasible because the underlying type is typically
// `[]interface{}`, where each element is a `map[string]interface{}`. The most robust
// and idiomatic way to perform this conversion is to re-marshal the generic structure
// back into a JSON byte slice and then unmarshal it into the desired target struct slice.
// This process also implicitly validates that the data from the database conforms to the
// API's data contract. It returns `backendtypes.ErrEmptyJSONValue` if the input is valid but results in an empty slice.
func NullJSONToTypedSlice[T any](jsonVal spanner.NullJSON) (*[]T, error) {
	if !jsonVal.Valid {
		return nil, backendtypes.ErrEmptyJSONValue
	}

	// Re-marshal the value from spanner.NullJSON to get a JSON byte slice.
	bytes, err := json.Marshal(jsonVal.Value)
	if err != nil {
		return nil, errors.Join(err, backendtypes.ErrJSONMarshal)
	}

	// Unmarshal the byte slice into the target backend type.
	var typedSlice []T
	if err := json.Unmarshal(bytes, &typedSlice); err != nil {
		return nil, errors.Join(err, backendtypes.ErrJSONUnmarshal)
	}

	if len(typedSlice) == 0 {
		return nil, backendtypes.ErrEmptyJSONValue
	}

	return &typedSlice, nil
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
			Chrome: convertChromeUsageToBackend(featureResult.ChromiumUsage),
		},
		BrowserImplementations: nil,
		DeveloperSignals: convertFeatureDeveloperSignals(spannerDeveloperSignals{
			Upvotes: featureResult.DeveloperSignalUpvotes,
			Link:    featureResult.DeveloperSignalLink,
		}),
		Discouraged: convertDiscouragedDetails(
			featureResult.Alternatives,
			featureResult.AccordingTo,
		),
		VendorPositions: nil,
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

	vendorPositions, err := NullJSONToTypedSlice[backend.VendorPosition](featureResult.VendorPositions)
	if err != nil && !errors.Is(err, backendtypes.ErrEmptyJSONValue) {
		// Log the error but don't fail the whole request.
		slog.ErrorContext(context.Background(), "unable to convert vendor positions", "error", err)
	}
	ret.VendorPositions = vendorPositions

	return ret
}

func convertDiscouragedDetails(alternatives, accordingTo []string) *backend.FeatureDiscouragedInfo {
	if len(alternatives) == 0 && len(accordingTo) == 0 {
		return nil
	}

	ret := &backend.FeatureDiscouragedInfo{
		Alternatives: nil,
		AccordingTo:  nil,
	}

	if len(alternatives) > 0 {
		alternativeInfo := make([]backend.FeatureDiscouragedAlternative, 0, len(alternatives))
		for idx := range alternatives {
			alternativeInfo = append(alternativeInfo, backend.FeatureDiscouragedAlternative{
				Id: alternatives[idx],
			})
		}
		ret.Alternatives = &alternativeInfo
	}

	if len(accordingTo) > 0 {
		accordingToInfo := make([]backend.FeatureDiscouragedAccordingTo, 0, len(accordingTo))
		for idx := range accordingTo {
			accordingToInfo = append(accordingToInfo, backend.FeatureDiscouragedAccordingTo{
				Link: accordingTo[idx],
			})
		}
		ret.AccordingTo = &accordingToInfo
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
	case backend.ExperimentalChromeAndroidAsc:
		return gcpspanner.NewBrowserImplSort(true, string(backend.ChromeAndroid), false)
	case backend.ExperimentalChromeAndroidDesc:
		return gcpspanner.NewBrowserImplSort(false, string(backend.ChromeAndroid), false)
	case backend.ExperimentalFirefoxAndroidAsc:
		return gcpspanner.NewBrowserImplSort(true, string(backend.FirefoxAndroid), false)
	case backend.ExperimentalFirefoxAndroidDesc:
		return gcpspanner.NewBrowserImplSort(false, string(backend.FirefoxAndroid), false)
	case backend.ExperimentalSafariIosAsc:
		return gcpspanner.NewBrowserImplSort(true, string(backend.SafariIos), false)
	case backend.ExperimentalSafariIosDesc:
		return gcpspanner.NewBrowserImplSort(false, string(backend.SafariIos), false)
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
	case backend.StableChromeAndroidAsc:
		return gcpspanner.NewBrowserImplSort(true, string(backend.ChromeAndroid), true)
	case backend.StableChromeAndroidDesc:
		return gcpspanner.NewBrowserImplSort(false, string(backend.ChromeAndroid), true)
	case backend.StableFirefoxAndroidAsc:
		return gcpspanner.NewBrowserImplSort(true, string(backend.FirefoxAndroid), true)
	case backend.StableFirefoxAndroidDesc:
		return gcpspanner.NewBrowserImplSort(false, string(backend.FirefoxAndroid), true)
	case backend.StableSafariIosAsc:
		return gcpspanner.NewBrowserImplSort(true, string(backend.SafariIos), true)
	case backend.StableSafariIosDesc:
		return gcpspanner.NewBrowserImplSort(false, string(backend.SafariIos), true)
	case backend.ChromeUsageAsc:
		// TODO: If we change the table in GCP from DailyChromiumHistogramMetrics, we should change the sort name

		return gcpspanner.NewChromiumUsageSort(true)
	case backend.ChromeUsageDesc:
		// TODO: If we change the table in GCP from DailyChromiumHistogramMetrics, we should change the sort name

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
	case backend.AvailabilityChromeAndroidAsc:
		return gcpspanner.NewBrowserFeatureSupportSort(true, string(backend.ChromeAndroid))
	case backend.AvailabilityChromeAndroidDesc:
		return gcpspanner.NewBrowserFeatureSupportSort(false, string(backend.ChromeAndroid))
	case backend.AvailabilityFirefoxAndroidAsc:
		return gcpspanner.NewBrowserFeatureSupportSort(true, string(backend.FirefoxAndroid))
	case backend.AvailabilityFirefoxAndroidDesc:
		return gcpspanner.NewBrowserFeatureSupportSort(false, string(backend.FirefoxAndroid))
	case backend.AvailabilitySafariIosAsc:
		return gcpspanner.NewBrowserFeatureSupportSort(true, string(backend.SafariIos))
	case backend.AvailabilitySafariIosDesc:
		return gcpspanner.NewBrowserFeatureSupportSort(false, string(backend.SafariIos))
	case backend.DeveloperSignalUpvotesAsc:
		return gcpspanner.NewDeveloperSignalUpvotesSort(true)
	case backend.DeveloperSignalUpvotesDesc:
		return gcpspanner.NewDeveloperSignalUpvotesSort(false)
	}

	// Unknown sort order
	slog.WarnContext(context.TODO(), "unsupported sort order", "order", *sortOrder)

	return gcpspanner.NewBaselineStatusSort(false)
}

func (s *Backend) GetFeature(
	ctx context.Context,
	featureID string,
	wptMetricView backend.WPTMetricView,
	browsers []backend.BrowserPathParam,
) (*backendtypes.GetFeatureResult, error) {
	filter := gcpspanner.NewFeatureKeyFilter(featureID)
	featureResult, err := s.client.GetFeature(
		ctx, filter, getSpannerWPTMetricView(wptMetricView), BrowserList(browsers).ToStringList())
	if err == nil {
		return backendtypes.NewGetFeatureResult(
			backendtypes.NewRegularFeatureResult(s.convertFeatureResult(featureResult))), nil
	}

	if !errors.Is(err, gcpspanner.ErrQueryReturnedNoResults) {
		return nil, err
	}

	// If the feature is not found, check if it has been moved.
	movedFeatureResult, err := s.client.GetMovedWebFeatureDetailsByOriginalFeatureKey(ctx, featureID)
	if err == nil {
		return backendtypes.NewGetFeatureResult(
			backendtypes.NewMovedFeatureResult(movedFeatureResult.NewFeatureKey)), nil
	}
	if !errors.Is(err, gcpspanner.ErrQueryReturnedNoResults) {
		return nil, err
	}

	// If the feature is not found and not moved, check if it has been split.
	splitFeatureResult, err := s.client.GetSplitWebFeatureByOriginalFeatureKey(ctx, featureID)
	if err == nil {
		features := make([]backend.FeatureSplitInfo, 0, len(splitFeatureResult.TargetFeatureKeys))
		for _, feature := range splitFeatureResult.TargetFeatureKeys {
			features = append(features, backend.FeatureSplitInfo{
				Id: feature,
			})
		}

		return backendtypes.NewGetFeatureResult(
			backendtypes.NewSplitFeatureResult(backend.FeatureEvolutionSplit{
				Features: features,
			}),
		), nil
	}
	if !errors.Is(err, gcpspanner.ErrQueryReturnedNoResults) {
		return nil, err
	}

	// If the feature is not found, not moved, and not split, then it does not exist in the database.
	return nil, errors.Join(err, backendtypes.ErrEntityDoesNotExist)
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
