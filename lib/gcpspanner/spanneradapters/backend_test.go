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
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/google/go-cmp/cmp"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// nolint: gochecknoglobals
var (
	nonNilInputPageToken = new("input-token")
	nonNilNextPageToken  = new("test-token")
	errTest              = errors.New("something is wrong")
	testStart            = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	testEnd              = time.Date(2000, time.January, 31, 0, 0, 0, 0, time.UTC)
)

type mockFeaturesSearchConfig struct {
	expectedPageToken     *string
	expectedPageSize      int
	expectedSortable      gcpspanner.Sortable
	expectedNode          *searchtypes.SearchNode
	expectedWPTMetricView gcpspanner.WPTMetricView
	expectedBrowsers      []string
	result                *gcpspanner.FeatureResultPage
	returnedError         error
}

type mockGetFeatureConfig struct {
	expectedFilterable    gcpspanner.Filterable
	expectedWPTMetricView gcpspanner.WPTMetricView
	expectedBrowsers      []string
	result                *gcpspanner.FeatureResult
	returnedError         error
}

type mockGetIDByFeaturesIDConfig struct {
	expectedFilterable gcpspanner.Filterable
	result             *string
	returnedError      error
}

type mockGetMovedWebFeatureDetailsByOriginalFeatureKeyConfig struct {
	expectedFeatureKey string
	result             *gcpspanner.MovedWebFeature
	returnedError      error
}

type mockGetSplitWebFeatureByOriginalFeatureKeyConfig struct {
	expectedFeatureKey string
	result             *gcpspanner.SplitWebFeature
	returnedError      error
}

type mockGetNotificationChannelConfig struct {
	expectedChannelID string
	expectedUserID    string
	result            *gcpspanner.NotificationChannel
	returnedError     error
}

type mockDeleteNotificationChannelConfig struct {
	expectedChannelID string
	expectedUserID    string
	returnedError     error
}

type mockCreateNotificationChannelConfig struct {
	expectedRequest gcpspanner.CreateNotificationChannelRequest
	result          *string
	returnedError   error
}

type mockUpdateNotificationChannelConfig struct {
	expectedRequest gcpspanner.UpdateNotificationChannelRequest
	returnedError   error
}

type mockListNotificationChannelsConfig struct {
	expectedRequest gcpspanner.ListNotificationChannelsRequest
	result          []gcpspanner.NotificationChannel
	nextPageToken   *string
	returnedError   error
}

type mockListBrowserFeatureCountMetricConfig struct {
	result        *gcpspanner.BrowserFeatureCountResultPage
	returnedError error
}

type mockListBaselineStatusCountsConfig struct {
	result        *gcpspanner.BaselineStatusCountResultPage
	returnedError error
}

type mockListMissingOneImplCountsConfig struct {
	result        *gcpspanner.MissingOneImplCountPage
	returnedError error
}

type mockListMissingOneImplFeaturesConfig struct {
	result        *gcpspanner.MissingOneImplFeatureListPage
	returnedError error
}

type mockCreateNewUserSavedSearchConfig struct {
	expectedNewSearch gcpspanner.CreateUserSavedSearchRequest
	result            *string
	returnedError     error
}

type mockGetUserSavedSearchConfig struct {
	expectedSavedSearchID       string
	expectedAuthenticatedUserID *string
	result                      *gcpspanner.UserSavedSearch
	returnedError               error
}

type mockDeleteUserSavedSearchConfig struct {
	expectedDeleteRequest gcpspanner.DeleteUserSavedSearchRequest
	returnedError         error
}

type mockListUserSavedSearchesConfig struct {
	expectedUserID    string
	expectedPageSize  int
	expectedPageToken *string
	result            *gcpspanner.UserSavedSearchesPage
	returnedError     error
}

type mockUpdateUserSavedSearchConfig struct {
	expectedRequest gcpspanner.UpdateSavedSearchRequest
	returnedError   error
}

type mockAddUserSearchBookmarkConfig struct {
	expectedRequest gcpspanner.UserSavedSearchBookmark
	returnedError   error
}

type mockDeleteUserSearchBookmarkConfig struct {
	expectedRequest gcpspanner.UserSavedSearchBookmark
	returnedError   error
}

type mockCreateSavedSearchSubscriptionConfig struct {
	expectedRequest gcpspanner.CreateSavedSearchSubscriptionRequest
	result          *string
	returnedError   error
}

type mockGetSavedSearchSubscriptionConfig struct {
	expectedSubscriptionID string
	expectedUserID         string
	result                 *gcpspanner.SavedSearchSubscriptionView
	returnedError          error
}

type mockUpdateSavedSearchSubscriptionConfig struct {
	expectedRequest gcpspanner.UpdateSavedSearchSubscriptionRequest
	returnedError   error
}

type mockDeleteSavedSearchSubscriptionConfig struct {
	expectedSubscriptionID string
	expectedUserID         string
	returnedError          error
}

type mockGetSavedSearchSubscriptionPublicConfig struct {
	expectedSubscriptionID string
	result                 *gcpspanner.SavedSearchSubscriptionView
	returnedError          error
}

type mockListSavedSearchNotificationEventsConfig struct {
	expectedSavedSearchID string
	expectedSnapshotType  string
	expectedPageSize      int
	expectedPageToken     *string
	result                []gcpspanner.SavedSearchNotificationEvent
	outputNextPageToken   *string
	returnedError         error
}

type mockListSavedSearchSubscriptionsConfig struct {
	expectedRequest gcpspanner.ListSavedSearchSubscriptionsRequest
	result          []gcpspanner.SavedSearchSubscriptionView
	nextPageToken   *string
	returnedError   error
}

type mockGetSystemGlobalSavedSearchConfig struct {
	results map[string]*gcpspanner.SystemGlobalSavedSearchWithSortOption
	errs    map[string]error
}

type mockGetSavedSearchConfig struct {
	results map[string]*gcpspanner.SavedSearch
	errs    map[string]error
}

type mockGetReferencingSavedSearchIDsConfig struct {
	results map[string][]string
	errs    map[string]error
}

type mockBackendSpannerClient struct {
	t                                        *testing.T
	aggregationData                          []gcpspanner.WPTRunAggregationMetricWithTime
	featureData                              []gcpspanner.WPTRunFeatureMetricWithTime
	chromeDailyUsageData                     []gcpspanner.ChromeDailyUsageStatWithDate
	mockFeaturesSearchCfg                    mockFeaturesSearchConfig
	mockGetFeatureCfg                        mockGetFeatureConfig
	mockGetIDByFeaturesIDCfg                 mockGetIDByFeaturesIDConfig
	mockListBrowserFeatureCountMetricCfg     mockListBrowserFeatureCountMetricConfig
	mockListMissingOneImplCountsCfg          mockListMissingOneImplCountsConfig
	mockListMissingOneImplFeaturesCfg        mockListMissingOneImplFeaturesConfig
	mockListBaselineStatusCountsCfg          mockListBaselineStatusCountsConfig
	mockGetNotificationChannelCfg            *mockGetNotificationChannelConfig
	mockDeleteNotificationChannelCfg         *mockDeleteNotificationChannelConfig
	mockListNotificationChannelsCfg          *mockListNotificationChannelsConfig
	mockCreateNotificationChannelCfg         *mockCreateNotificationChannelConfig
	mockUpdateNotificationChannelCfg         *mockUpdateNotificationChannelConfig
	mockCreateNewUserSavedSearchCfg          *mockCreateNewUserSavedSearchConfig
	mockGetUserSavedSearchCfg                *mockGetUserSavedSearchConfig
	mockDeleteUserSavedSearchCfg             *mockDeleteUserSavedSearchConfig
	mockListUserSavedSearchesCfg             *mockListUserSavedSearchesConfig
	mockUpdateUserSavedSearchCfg             *mockUpdateUserSavedSearchConfig
	mockAddUserSearchBookmarkCfg             *mockAddUserSearchBookmarkConfig
	mockDeleteUserSearchBookmarkCfg          *mockDeleteUserSearchBookmarkConfig
	mockCreateSavedSearchSubscriptionCfg     *mockCreateSavedSearchSubscriptionConfig
	mockGetSavedSearchSubscriptionCfg        *mockGetSavedSearchSubscriptionConfig
	mockGetSavedSearchSubscriptionPublicCfg  *mockGetSavedSearchSubscriptionPublicConfig
	mockListSavedSearchNotificationEventsCfg *mockListSavedSearchNotificationEventsConfig
	mockUpdateSavedSearchSubscriptionCfg     *mockUpdateSavedSearchSubscriptionConfig
	mockDeleteSavedSearchSubscriptionCfg     *mockDeleteSavedSearchSubscriptionConfig
	mockListSavedSearchSubscriptionsCfg      *mockListSavedSearchSubscriptionsConfig
	mockGetSystemGlobalSavedSearchCfg        *mockGetSystemGlobalSavedSearchConfig
	mockGetSavedSearchCfg                    *mockGetSavedSearchConfig
	mockGetReferencingSavedSearchIDsCfg      *mockGetReferencingSavedSearchIDsConfig
	pageToken                                *string
	err                                      error

	mockGetMovedWebFeatureDetailsByOriginalFeatureKeyCfg *mockGetMovedWebFeatureDetailsByOriginalFeatureKeyConfig
	mockGetSplitWebFeatureByOriginalFeatureKeyCfg        *mockGetSplitWebFeatureByOriginalFeatureKeyConfig
	mockSyncUserProfileInfoCfg                           *mockSyncUserProfileInfoConfig
}

type mockSyncUserProfileInfoConfig struct {
	expectedUserProfile gcpspanner.UserProfile
	returnedError       error
}

func (c mockBackendSpannerClient) SyncUserProfileInfo(
	_ context.Context, userProfile gcpspanner.UserProfile) error {
	if !reflect.DeepEqual(userProfile, c.mockSyncUserProfileInfoCfg.expectedUserProfile) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockSyncUserProfileInfoCfg.returnedError
}

func (c mockBackendSpannerClient) GetSystemGlobalSavedSearch(
	_ context.Context,
	id string,
) (*gcpspanner.SystemGlobalSavedSearchWithSortOption, error) {
	if c.mockGetSystemGlobalSavedSearchCfg != nil {
		if err, ok := c.mockGetSystemGlobalSavedSearchCfg.errs[id]; ok {
			return nil, err
		}
		if res, ok := c.mockGetSystemGlobalSavedSearchCfg.results[id]; ok {
			return res, nil
		}
	}

	return nil, gcpspanner.ErrQueryReturnedNoResults
}

func (c mockBackendSpannerClient) ListSystemGlobalSavedSearches(
	_ context.Context,
	_ int,
	_ *string,
) ([]gcpspanner.SystemGlobalSavedSearch, *string, error) {
	return nil, nil, nil
}

func (c mockBackendSpannerClient) GetSavedSearch(
	_ context.Context,
	id string,
) (*gcpspanner.SavedSearch, error) {
	if c.mockGetSavedSearchCfg != nil {
		if err, ok := c.mockGetSavedSearchCfg.errs[id]; ok {
			return nil, err
		}
		if res, ok := c.mockGetSavedSearchCfg.results[id]; ok {
			return res, nil
		}
	}

	return nil, gcpspanner.ErrQueryReturnedNoResults
}

func (c mockBackendSpannerClient) GetReferencingSavedSearchIDs(
	_ context.Context,
	id string,
) ([]string, error) {
	if c.mockGetReferencingSavedSearchIDsCfg != nil {
		if err, ok := c.mockGetReferencingSavedSearchIDsCfg.errs[id]; ok {
			return nil, err
		}
		if res, ok := c.mockGetReferencingSavedSearchIDsCfg.results[id]; ok {
			return res, nil
		}
	}

	return nil, nil
}

// GetMovedWebFeatureDetailsByOriginalFeatureKey implements BackendSpannerClient.
func (c mockBackendSpannerClient) GetMovedWebFeatureDetailsByOriginalFeatureKey(
	_ context.Context, featureKey string) (*gcpspanner.MovedWebFeature, error) {
	if featureKey != c.mockGetMovedWebFeatureDetailsByOriginalFeatureKeyCfg.expectedFeatureKey {
		c.t.Errorf("unexpected input to mock: %s", featureKey)
	}

	return c.mockGetMovedWebFeatureDetailsByOriginalFeatureKeyCfg.result,
		c.mockGetMovedWebFeatureDetailsByOriginalFeatureKeyCfg.returnedError
}

// GetSplitWebFeatureByOriginalFeatureKey implements BackendSpannerClient.
func (c mockBackendSpannerClient) GetSplitWebFeatureByOriginalFeatureKey(
	_ context.Context, featureKey string) (*gcpspanner.SplitWebFeature, error) {
	if featureKey != c.mockGetSplitWebFeatureByOriginalFeatureKeyCfg.expectedFeatureKey {
		c.t.Errorf("unexpected input to mock: %s", featureKey)
	}

	return c.mockGetSplitWebFeatureByOriginalFeatureKeyCfg.result,
		c.mockGetSplitWebFeatureByOriginalFeatureKeyCfg.returnedError
}

// AddUserSearchBookmark implements BackendSpannerClient.
func (c mockBackendSpannerClient) AddUserSearchBookmark(
	_ context.Context, req gcpspanner.UserSavedSearchBookmark) error {
	if !reflect.DeepEqual(req, c.mockAddUserSearchBookmarkCfg.expectedRequest) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockAddUserSearchBookmarkCfg.returnedError
}

// DeleteUserSearchBookmark implements BackendSpannerClient.
func (c mockBackendSpannerClient) DeleteUserSearchBookmark(
	_ context.Context, req gcpspanner.UserSavedSearchBookmark) error {
	if !reflect.DeepEqual(req, c.mockDeleteUserSearchBookmarkCfg.expectedRequest) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockDeleteUserSearchBookmarkCfg.returnedError
}

func (c mockBackendSpannerClient) ListUserSavedSearches(
	_ context.Context, userID string, pageSize int, pageToken *string) (*gcpspanner.UserSavedSearchesPage, error) {
	if userID != c.mockListUserSavedSearchesCfg.expectedUserID ||
		pageSize != c.mockListUserSavedSearchesCfg.expectedPageSize ||
		!reflect.DeepEqual(pageToken, c.mockListUserSavedSearchesCfg.expectedPageToken) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockListUserSavedSearchesCfg.result, c.mockListUserSavedSearchesCfg.returnedError
}

func (c mockBackendSpannerClient) CreateNewUserSavedSearch(
	_ context.Context, newSearch gcpspanner.CreateUserSavedSearchRequest) (*string, error) {
	if !reflect.DeepEqual(newSearch, c.mockCreateNewUserSavedSearchCfg.expectedNewSearch) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockCreateNewUserSavedSearchCfg.result, c.mockCreateNewUserSavedSearchCfg.returnedError
}

func (c mockBackendSpannerClient) GetFeature(
	_ context.Context,
	filter gcpspanner.Filterable,
	view gcpspanner.WPTMetricView,
	browsers []string) (*gcpspanner.FeatureResult, error) {
	if !reflect.DeepEqual(filter, c.mockGetFeatureCfg.expectedFilterable) ||
		view != c.mockGetFeatureCfg.expectedWPTMetricView ||
		!slices.Equal(browsers, c.mockGetFeatureCfg.expectedBrowsers) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockGetFeatureCfg.result, c.mockGetFeatureCfg.returnedError
}

func (c mockBackendSpannerClient) GetIDFromFeatureKey(
	_ context.Context, filter *gcpspanner.FeatureIDFilter) (*string, error) {
	if !reflect.DeepEqual(filter, c.mockGetIDByFeaturesIDCfg.expectedFilterable) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockGetIDByFeaturesIDCfg.result, c.mockGetIDByFeaturesIDCfg.returnedError
}

func (c mockBackendSpannerClient) ListBrowserFeatureCountMetric(
	ctx context.Context,
	targetBrowser string,
	targetMobileBrowser *string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) (*gcpspanner.BrowserFeatureCountResultPage, error) {
	//nolint: goconst
	if ctx != context.Background() ||
		targetBrowser != "mybrowser" ||
		targetMobileBrowser != nil ||
		!startAt.Equal(testStart) ||
		!endAt.Equal(testEnd) ||
		pageSize != 100 ||
		pageToken != nonNilInputPageToken {
		c.t.Error("unexpected input to mock")
	}

	return c.mockListBrowserFeatureCountMetricCfg.result, c.mockListBrowserFeatureCountMetricCfg.returnedError
}

func (c mockBackendSpannerClient) ListMetricsForFeatureIDBrowserAndChannel(
	ctx context.Context,
	featureID string,
	browser string,
	channel string,
	metric gcpspanner.WPTMetricView,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]gcpspanner.WPTRunFeatureMetricWithTime, *string, error) {
	if ctx != context.Background() ||
		featureID != "feature" ||
		browser != "browser" ||
		channel != "channel" ||
		metric != gcpspanner.WPTSubtestView ||
		!startAt.Equal(testStart) ||
		!endAt.Equal(testEnd) ||
		pageSize != 100 ||
		pageToken != nonNilInputPageToken {
		c.t.Error("unexpected input to mock")
	}

	return c.featureData, c.pageToken, c.err
}

func (c mockBackendSpannerClient) ListChromeDailyUsageStatsForFeatureID(
	ctx context.Context,
	featureID string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]gcpspanner.ChromeDailyUsageStatWithDate, *string, error) {
	if ctx != context.Background() ||
		featureID != "feature" ||
		!startAt.Equal(testStart) ||
		!endAt.Equal(testEnd) ||
		pageSize != 100 ||
		pageToken != nonNilInputPageToken {
		c.t.Error("unexpected input to mock")
	}

	return c.chromeDailyUsageData, c.pageToken, c.err
}

func (c mockBackendSpannerClient) ListMetricsOverTimeWithAggregatedTotals(
	ctx context.Context,
	featureIDs []string,
	browser string,
	channel string,
	metric gcpspanner.WPTMetricView,
	startAt, endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]gcpspanner.WPTRunAggregationMetricWithTime, *string, error) {
	if ctx != context.Background() ||
		!slices.Equal[[]string](featureIDs, []string{"feature1", "feature2"}) ||
		browser != "browser" ||
		channel != "channel" ||
		metric != gcpspanner.WPTSubtestView ||
		!startAt.Equal(testStart) ||
		!endAt.Equal(testEnd) ||
		pageSize != 100 ||
		pageToken != nonNilInputPageToken {
		c.t.Error("unexpected input to mock")
	}

	return c.aggregationData, c.pageToken, c.err
}

func (c mockBackendSpannerClient) GetNotificationChannel(
	_ context.Context, channelID string, userID string) (*gcpspanner.NotificationChannel, error) {
	if channelID != c.mockGetNotificationChannelCfg.expectedChannelID ||
		userID != c.mockGetNotificationChannelCfg.expectedUserID {
		c.t.Error("unexpected input to mock")
	}

	return c.mockGetNotificationChannelCfg.result, c.mockGetNotificationChannelCfg.returnedError
}

func (c mockBackendSpannerClient) DeleteNotificationChannel(_ context.Context, channelID string, userID string) error {
	if channelID != c.mockDeleteNotificationChannelCfg.expectedChannelID ||
		userID != c.mockDeleteNotificationChannelCfg.expectedUserID {
		c.t.Error("unexpected input to mock")
	}

	return c.mockDeleteNotificationChannelCfg.returnedError
}

func (c mockBackendSpannerClient) CreateNotificationChannel(
	_ context.Context, req gcpspanner.CreateNotificationChannelRequest) (*string, error) {
	if !reflect.DeepEqual(req, c.mockCreateNotificationChannelCfg.expectedRequest) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockCreateNotificationChannelCfg.result, c.mockCreateNotificationChannelCfg.returnedError
}

func (c mockBackendSpannerClient) UpdateNotificationChannel(
	_ context.Context, req gcpspanner.UpdateNotificationChannelRequest) error {
	if !reflect.DeepEqual(req, c.mockUpdateNotificationChannelCfg.expectedRequest) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockUpdateNotificationChannelCfg.returnedError
}

func (c mockBackendSpannerClient) ListNotificationChannels(
	_ context.Context,
	req gcpspanner.ListNotificationChannelsRequest) ([]gcpspanner.NotificationChannel, *string, error) {
	if !reflect.DeepEqual(req, c.mockListNotificationChannelsCfg.expectedRequest) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockListNotificationChannelsCfg.result,
		c.mockListNotificationChannelsCfg.nextPageToken, c.mockListNotificationChannelsCfg.returnedError
}

func (c mockBackendSpannerClient) FeaturesSearch(
	_ context.Context,
	pageToken *string,
	pageSize int,
	searchNode *searchtypes.SearchNode,
	sortOrder gcpspanner.Sortable,
	wptMetricView gcpspanner.WPTMetricView,
	browsers []string) (*gcpspanner.FeatureResultPage, error) {
	if pageToken != c.mockFeaturesSearchCfg.expectedPageToken ||
		pageSize != c.mockFeaturesSearchCfg.expectedPageSize ||
		!reflect.DeepEqual(searchNode, c.mockFeaturesSearchCfg.expectedNode) ||
		!reflect.DeepEqual(sortOrder, c.mockFeaturesSearchCfg.expectedSortable) ||
		wptMetricView != c.mockFeaturesSearchCfg.expectedWPTMetricView ||
		!slices.Equal(browsers, c.mockFeaturesSearchCfg.expectedBrowsers) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockFeaturesSearchCfg.result,
		c.mockFeaturesSearchCfg.returnedError
}

func (c mockBackendSpannerClient) ListMissingOneImplCounts(
	ctx context.Context,
	targetBrowser string,
	targetMobileBrowser *string,
	otherBrowsers []string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) (*gcpspanner.MissingOneImplCountPage, error) {
	if ctx != context.Background() ||
		targetBrowser != "mybrowser" ||
		targetMobileBrowser != nil ||
		!slices.Equal(otherBrowsers, []string{"browser1", "browser2"}) ||
		!startAt.Equal(testStart) ||
		!endAt.Equal(testEnd) ||
		pageSize != 100 ||
		pageToken != nonNilInputPageToken {
		c.t.Error("unexpected input to mock")
	}

	return c.mockListMissingOneImplCountsCfg.result, c.mockListMissingOneImplCountsCfg.returnedError
}

func (c mockBackendSpannerClient) ListMissingOneImplementationFeatures(
	ctx context.Context,
	targetBrowser string,
	targetMobileBrowser *string,
	otherBrowsers []string,
	targetDate time.Time,
	pageSize int,
	pageToken *string,
) (*gcpspanner.MissingOneImplFeatureListPage, error) {
	if ctx != context.Background() ||
		targetBrowser != "mybrowser" ||
		targetMobileBrowser != nil ||
		!slices.Equal(otherBrowsers, []string{"browser1", "browser2"}) ||
		!targetDate.Equal(testStart) ||
		pageSize != 100 ||
		pageToken != nonNilInputPageToken {
		c.t.Error("unexpected input to mock")
	}

	return c.mockListMissingOneImplFeaturesCfg.result, c.mockListMissingOneImplFeaturesCfg.returnedError
}

// ListBaselineStatusCounts implements BackendSpannerClient.
func (c mockBackendSpannerClient) ListBaselineStatusCounts(
	ctx context.Context, dateType gcpspanner.BaselineDateType, startAt time.Time,
	endAt time.Time, pageSize int, pageToken *string) (*gcpspanner.BaselineStatusCountResultPage, error) {
	if ctx != context.Background() ||
		dateType != gcpspanner.BaselineDateTypeLow ||
		!startAt.Equal(testStart) ||
		!endAt.Equal(testEnd) ||
		pageSize != 100 ||
		pageToken != nonNilInputPageToken {
		c.t.Error("unexpected input to mock")
	}

	return c.mockListBaselineStatusCountsCfg.result, c.mockListBaselineStatusCountsCfg.returnedError
}

func (c mockBackendSpannerClient) GetUserSavedSearch(
	_ context.Context, id string, authenticatedUserID *string) (
	*gcpspanner.UserSavedSearch, error) {
	if id != c.mockGetUserSavedSearchCfg.expectedSavedSearchID ||
		!reflect.DeepEqual(authenticatedUserID, c.mockGetUserSavedSearchCfg.expectedAuthenticatedUserID) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockGetUserSavedSearchCfg.result, c.mockGetUserSavedSearchCfg.returnedError

}

func (c mockBackendSpannerClient) DeleteUserSavedSearch(
	_ context.Context, req gcpspanner.DeleteUserSavedSearchRequest) error {
	if !reflect.DeepEqual(req, c.mockDeleteUserSavedSearchCfg.expectedDeleteRequest) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockDeleteUserSavedSearchCfg.returnedError
}

func (c mockBackendSpannerClient) UpdateUserSavedSearch(
	_ context.Context, req gcpspanner.UpdateSavedSearchRequest) error {
	if !reflect.DeepEqual(req, c.mockUpdateUserSavedSearchCfg.expectedRequest) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockUpdateUserSavedSearchCfg.returnedError
}

// CreateSavedSearchSubscription implements BackendSpannerClient.
func (c mockBackendSpannerClient) CreateSavedSearchSubscription(
	_ context.Context, req gcpspanner.CreateSavedSearchSubscriptionRequest) (*string, error) {
	if !reflect.DeepEqual(req, c.mockCreateSavedSearchSubscriptionCfg.expectedRequest) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockCreateSavedSearchSubscriptionCfg.result, c.mockCreateSavedSearchSubscriptionCfg.returnedError
}

// DeleteSavedSearchSubscription implements BackendSpannerClient.
func (c mockBackendSpannerClient) DeleteSavedSearchSubscription(
	_ context.Context, subscriptionID string, userID string) error {
	if subscriptionID != c.mockDeleteSavedSearchSubscriptionCfg.expectedSubscriptionID ||
		userID != c.mockDeleteSavedSearchSubscriptionCfg.expectedUserID {
		c.t.Error("unexpected input to mock")
	}

	return c.mockDeleteSavedSearchSubscriptionCfg.returnedError
}

// GetSavedSearchSubscription implements BackendSpannerClient.
func (c mockBackendSpannerClient) GetSavedSearchSubscription(
	_ context.Context,
	subscriptionID string,
	userID string) (*gcpspanner.SavedSearchSubscriptionView, error) {
	if subscriptionID != c.mockGetSavedSearchSubscriptionCfg.expectedSubscriptionID ||
		userID != c.mockGetSavedSearchSubscriptionCfg.expectedUserID {
		c.t.Error("unexpected input to mock")
	}

	return c.mockGetSavedSearchSubscriptionCfg.result, c.mockGetSavedSearchSubscriptionCfg.returnedError
}

// GetSavedSearchSubscriptionPublic implements BackendSpannerClient.
func (c mockBackendSpannerClient) GetSavedSearchSubscriptionPublic(
	_ context.Context,
	subscriptionID string) (*gcpspanner.SavedSearchSubscriptionView, error) {
	if subscriptionID != c.mockGetSavedSearchSubscriptionPublicCfg.expectedSubscriptionID {
		c.t.Error("unexpected input to mock")
	}

	return c.mockGetSavedSearchSubscriptionPublicCfg.result, c.mockGetSavedSearchSubscriptionPublicCfg.returnedError
}

// ListSavedSearchNotificationEvents implements BackendSpannerClient.
func (c mockBackendSpannerClient) ListSavedSearchNotificationEvents(
	_ context.Context,
	savedSearchID string,
	snapshotType string,
	pageSize int,
	pageToken *string) ([]gcpspanner.SavedSearchNotificationEvent, *string, error) {
	if savedSearchID != c.mockListSavedSearchNotificationEventsCfg.expectedSavedSearchID ||
		snapshotType != c.mockListSavedSearchNotificationEventsCfg.expectedSnapshotType ||
		pageSize != c.mockListSavedSearchNotificationEventsCfg.expectedPageSize {
		c.t.Error("unexpected input to mock")
	}
	if c.mockListSavedSearchNotificationEventsCfg.expectedPageToken != nil && pageToken != nil {
		if *c.mockListSavedSearchNotificationEventsCfg.expectedPageToken != *pageToken {
			c.t.Error("unexpected page token in mock")
		}
	} else if c.mockListSavedSearchNotificationEventsCfg.expectedPageToken != pageToken {
		c.t.Error("unexpected page token in mock")
	}

	return c.mockListSavedSearchNotificationEventsCfg.result,
		c.mockListSavedSearchNotificationEventsCfg.outputNextPageToken,
		c.mockListSavedSearchNotificationEventsCfg.returnedError
}

// ListSavedSearchSubscriptions implements BackendSpannerClient.
func (c mockBackendSpannerClient) ListSavedSearchSubscriptions(
	_ context.Context,
	req gcpspanner.ListSavedSearchSubscriptionsRequest) ([]gcpspanner.SavedSearchSubscriptionView, *string, error) {
	if !reflect.DeepEqual(req, c.mockListSavedSearchSubscriptionsCfg.expectedRequest) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockListSavedSearchSubscriptionsCfg.result,
		c.mockListSavedSearchSubscriptionsCfg.nextPageToken,
		c.mockListSavedSearchSubscriptionsCfg.returnedError
}

// UpdateSavedSearchSubscription implements BackendSpannerClient.
func (c mockBackendSpannerClient) UpdateSavedSearchSubscription(
	_ context.Context, req gcpspanner.UpdateSavedSearchSubscriptionRequest) error {
	if !reflect.DeepEqual(req, c.mockUpdateSavedSearchSubscriptionCfg.expectedRequest) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockUpdateSavedSearchSubscriptionCfg.returnedError
}

func TestCreateSavedSearchSubscriptionMapsLimitError(t *testing.T) {
	mock := new(mockBackendSpannerClient)
	mock.t = t
	mock.mockCreateSavedSearchSubscriptionCfg = &mockCreateSavedSearchSubscriptionConfig{
		expectedRequest: gcpspanner.CreateSavedSearchSubscriptionRequest{
			UserID:        "user",
			ChannelID:     "channel",
			SavedSearchID: "search",
			Triggers:      []gcpspanner.SubscriptionTrigger{},
			Frequency:     gcpspanner.SavedSearchSnapshotTypeImmediate,
		},
		result:        nil,
		returnedError: gcpspanner.ErrSubscriptionLimitExceeded,
	}

	bk := NewBackend(mock)
	_, err := bk.CreateSavedSearchSubscription(context.Background(), "user", backend.Subscription{
		ChannelId:     "channel",
		SavedSearchId: "search",
		Triggers:      []backend.SubscriptionTriggerWritable{},
		Frequency:     backend.SubscriptionFrequencyImmediate,
	})
	if !errors.Is(err, backendtypes.ErrUserMaxSubscriptions) {
		t.Errorf("expected ErrUserMaxSubscriptions, got %v", err)
	}
}

func TestListMetricsForFeatureIDBrowserAndChannel(t *testing.T) {
	testCases := []struct {
		name              string
		featureData       []gcpspanner.WPTRunFeatureMetricWithTime
		pageToken         *string
		err               error
		expectedOutput    []backend.WPTRunMetric
		expectedPageToken *string
		expectedErr       error
	}{
		{
			name: "success",
			featureData: []gcpspanner.WPTRunFeatureMetricWithTime{
				{
					TimeStart:  time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
					RunID:      10,
					TotalTests: new(int64(20)),
					TestPass:   new(int64(10)),
				},
				{
					TimeStart:  time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
					RunID:      9,
					TotalTests: new(int64(19)),
					TestPass:   new(int64(9)),
				},
				{
					TimeStart:  time.Date(2000, time.January, 8, 0, 0, 0, 0, time.UTC),
					RunID:      8,
					TotalTests: new(int64(18)),
					TestPass:   new(int64(8)),
				},
			},
			pageToken: nonNilNextPageToken,
			err:       nil,
			expectedOutput: []backend.WPTRunMetric{
				{
					RunTimestamp:    time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: new(int64(20)),
					TestPassCount:   new(int64(10)),
				},
				{
					RunTimestamp:    time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: new(int64(19)),
					TestPassCount:   new(int64(9)),
				},
				{
					RunTimestamp:    time.Date(2000, time.January, 8, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: new(int64(18)),
					TestPassCount:   new(int64(8)),
				},
			},
			expectedPageToken: nonNilNextPageToken,
			expectedErr:       nil,
		},
		{
			name:              "failure",
			featureData:       nil,
			pageToken:         nil,
			err:               errTest,
			expectedOutput:    nil,
			expectedPageToken: nil,
			expectedErr:       errTest,
		},
		{
			name:              "invalid cursor",
			featureData:       nil,
			pageToken:         new(""),
			err:               gcpspanner.ErrInvalidCursorFormat,
			expectedOutput:    nil,
			expectedPageToken: nil,
			expectedErr:       backendtypes.ErrInvalidPageToken,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:           t,
				featureData: tc.featureData,
				pageToken:   tc.pageToken,
				err:         tc.err,
			}
			b := NewBackend(mock)
			metrics, pageToken, err := b.ListMetricsForFeatureIDBrowserAndChannel(
				context.Background(), "feature", "browser", "channel", backend.SubtestCounts,
				testStart, testEnd, 100, nonNilInputPageToken)
			if !errors.Is(err, tc.expectedErr) {
				t.Error("unexpected error")
			}

			if pageToken != tc.expectedPageToken {
				t.Error("unexpected page token")
			}

			if !reflect.DeepEqual(metrics, tc.expectedOutput) {
				t.Error("unexpected metrics")
			}
		})
	}
}

func TestListBrowserFeatureCountMetric(t *testing.T) {
	testCases := []struct {
		name         string
		cfg          mockListBrowserFeatureCountMetricConfig
		expectedPage *backend.BrowserReleaseFeatureMetricsPage
		expectedErr  error
	}{
		{
			name: "success",
			cfg: mockListBrowserFeatureCountMetricConfig{
				result: &gcpspanner.BrowserFeatureCountResultPage{
					NextPageToken: nonNilNextPageToken,
					Metrics: []gcpspanner.BrowserFeatureCountMetric{
						{
							ReleaseDate:  time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
							FeatureCount: 10,
						},
						{
							ReleaseDate:  time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
							FeatureCount: 9,
						},
					},
				},
				returnedError: nil,
			},
			expectedPage: &backend.BrowserReleaseFeatureMetricsPage{
				Metadata: &backend.PageMetadata{
					NextPageToken: nonNilNextPageToken,
				},
				Data: []backend.BrowserReleaseFeatureMetric{
					{
						Count:     new(int64(10)),
						Timestamp: time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
					},
					{
						Count:     new(int64(9)),
						Timestamp: time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "failure",
			cfg: mockListBrowserFeatureCountMetricConfig{
				result:        nil,
				returnedError: errTest,
			},
			expectedPage: nil,
			expectedErr:  errTest,
		},
		{
			name: "invalid cursor",
			cfg: mockListBrowserFeatureCountMetricConfig{
				result:        nil,
				returnedError: gcpspanner.ErrInvalidCursorFormat,
			},
			expectedPage: nil,
			expectedErr:  backendtypes.ErrInvalidPageToken,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                                    t,
				mockListBrowserFeatureCountMetricCfg: tc.cfg,
			}
			backend := NewBackend(mock)
			page, err := backend.ListBrowserFeatureCountMetric(
				context.Background(),
				"mybrowser",
				nil,
				testStart,
				testEnd,
				100,
				nonNilInputPageToken)
			if !errors.Is(err, tc.expectedErr) {
				t.Error("unexpected error")
			}

			if !reflect.DeepEqual(page, tc.expectedPage) {
				t.Error("unexpected metrics")
			}
		})
	}
}

func TestListMetricsOverTimeWithAggregatedTotals(t *testing.T) {

	testCases := []struct {
		name              string
		aggregatedData    []gcpspanner.WPTRunAggregationMetricWithTime
		pageToken         *string
		err               error
		expectedOutput    []backend.WPTRunMetric
		expectedPageToken *string
		expectedErr       error
	}{
		{
			name: "success",
			aggregatedData: []gcpspanner.WPTRunAggregationMetricWithTime{
				{
					WPTRunFeatureMetricWithTime: gcpspanner.WPTRunFeatureMetricWithTime{
						TimeStart:  time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
						RunID:      10,
						TotalTests: new(int64(20)),
						TestPass:   new(int64(10)),
					},
				},
				{
					WPTRunFeatureMetricWithTime: gcpspanner.WPTRunFeatureMetricWithTime{
						TimeStart:  time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
						RunID:      9,
						TotalTests: new(int64(19)),
						TestPass:   new(int64(9)),
					},
				},
				{
					WPTRunFeatureMetricWithTime: gcpspanner.WPTRunFeatureMetricWithTime{
						TimeStart:  time.Date(2000, time.January, 8, 0, 0, 0, 0, time.UTC),
						RunID:      8,
						TotalTests: new(int64(18)),
						TestPass:   new(int64(8)),
					},
				},
			},
			pageToken: nonNilNextPageToken,
			err:       nil,
			expectedOutput: []backend.WPTRunMetric{
				{
					RunTimestamp:    time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: new(int64(20)),
					TestPassCount:   new(int64(10)),
				},
				{
					RunTimestamp:    time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: new(int64(19)),
					TestPassCount:   new(int64(9)),
				},
				{
					RunTimestamp:    time.Date(2000, time.January, 8, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: new(int64(18)),
					TestPassCount:   new(int64(8)),
				},
			},
			expectedPageToken: nonNilNextPageToken,
			expectedErr:       nil,
		},
		{
			name:              "failure",
			aggregatedData:    nil,
			pageToken:         nil,
			err:               errTest,
			expectedOutput:    nil,
			expectedPageToken: nil,
			expectedErr:       errTest,
		},
		{
			name:              "invalid cursor",
			aggregatedData:    nil,
			pageToken:         new(""),
			err:               gcpspanner.ErrInvalidCursorFormat,
			expectedOutput:    nil,
			expectedPageToken: nil,
			expectedErr:       backendtypes.ErrInvalidPageToken,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:               t,
				aggregationData: tc.aggregatedData,
				pageToken:       tc.pageToken,
				err:             tc.err,
			}
			b := NewBackend(mock)
			metrics, pageToken, err := b.ListMetricsOverTimeWithAggregatedTotals(
				context.Background(),
				[]string{"feature1", "feature2"},
				"browser",
				"channel",
				backend.SubtestCounts,
				testStart,
				testEnd,
				100,
				nonNilInputPageToken)
			if !errors.Is(err, tc.expectedErr) {
				t.Error("unexpected error")
			}

			if pageToken != tc.expectedPageToken {
				t.Error("unexpected page token")
			}

			if !reflect.DeepEqual(metrics, tc.expectedOutput) {
				t.Error("unexpected metrics")
			}
		})
	}
}

func TestListMissingOneImplCounts(t *testing.T) {
	// nolint:dupl // WONTFIX - not exactly the same as ListBaselineStatusCounts
	testCases := []struct {
		name         string
		cfg          mockListMissingOneImplCountsConfig
		expectedPage *backend.BrowserReleaseFeatureMetricsPage
		expectedErr  error
	}{
		{
			name: "success",
			cfg: mockListMissingOneImplCountsConfig{
				result: &gcpspanner.MissingOneImplCountPage{
					NextPageToken: nonNilNextPageToken,
					Metrics: []gcpspanner.MissingOneImplCount{
						{
							Count:            90,
							EventReleaseDate: time.Date(2010, time.March, 10, 0, 0, 0, 0, time.UTC),
						},
						{
							Count:            99,
							EventReleaseDate: time.Date(2010, time.March, 9, 0, 0, 0, 0, time.UTC),
						},
					},
				},
				returnedError: nil,
			},
			expectedPage: &backend.BrowserReleaseFeatureMetricsPage{
				Metadata: &backend.PageMetadata{
					NextPageToken: nonNilNextPageToken,
				},
				Data: []backend.BrowserReleaseFeatureMetric{
					{
						Count:     new(int64(90)),
						Timestamp: time.Date(2010, time.March, 10, 0, 0, 0, 0, time.UTC),
					},
					{
						Count:     new(int64(99)),
						Timestamp: time.Date(2010, time.March, 9, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "failure",
			cfg: mockListMissingOneImplCountsConfig{
				result:        nil,
				returnedError: errTest,
			},
			expectedPage: nil,
			expectedErr:  errTest,
		},
		{
			name: "invalid cursor",
			cfg: mockListMissingOneImplCountsConfig{
				result:        nil,
				returnedError: gcpspanner.ErrInvalidCursorFormat,
			},
			expectedPage: nil,
			expectedErr:  backendtypes.ErrInvalidPageToken,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                               t,
				mockListMissingOneImplCountsCfg: tc.cfg,
			}
			backend := NewBackend(mock)
			page, err := backend.ListMissingOneImplCounts(
				context.Background(),
				"mybrowser",
				nil,
				[]string{"browser1", "browser2"},
				testStart,
				testEnd,
				100,
				nonNilInputPageToken)
			if !errors.Is(err, tc.expectedErr) {
				t.Error("unexpected error")
			}

			if !reflect.DeepEqual(page, tc.expectedPage) {
				t.Error("unexpected metrics")
			}
		})
	}
}

func TestListMissingOneImplementationFeatures(t *testing.T) {
	// nolint:dupl // WONTFIX
	testCases := []struct {
		name         string
		cfg          mockListMissingOneImplFeaturesConfig
		expectedPage *backend.MissingOneImplFeaturesPage
		expectedErr  error
	}{
		{
			name: "success",
			cfg: mockListMissingOneImplFeaturesConfig{
				result: &gcpspanner.MissingOneImplFeatureListPage{
					NextPageToken: nonNilNextPageToken,
					FeatureList: []gcpspanner.MissingOneImplFeature{
						{
							WebFeatureID: "foo",
						},
						{
							WebFeatureID: "bar",
						},
					},
				},
				returnedError: nil,
			},
			expectedPage: &backend.MissingOneImplFeaturesPage{
				Metadata: &backend.PageMetadata{
					NextPageToken: nonNilNextPageToken,
				},
				Data: []backend.MissingOneImplFeature{
					{
						FeatureId: new("foo"),
					},
					{
						FeatureId: new("bar"),
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "failure",
			cfg: mockListMissingOneImplFeaturesConfig{
				result:        nil,
				returnedError: errTest,
			},
			expectedPage: nil,
			expectedErr:  errTest,
		},
		{
			name: "invalid cursor",
			cfg: mockListMissingOneImplFeaturesConfig{
				result:        nil,
				returnedError: gcpspanner.ErrInvalidCursorFormat,
			},
			expectedPage: nil,
			expectedErr:  backendtypes.ErrInvalidPageToken,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                                 t,
				mockListMissingOneImplFeaturesCfg: tc.cfg,
			}
			backend := NewBackend(mock)
			page, err := backend.ListMissingOneImplementationFeatures(
				context.Background(),
				"mybrowser",
				nil,
				[]string{"browser1", "browser2"},
				testStart,
				100,
				nonNilInputPageToken)
			if !errors.Is(err, tc.expectedErr) {
				t.Error("unexpected error")
			}

			if !reflect.DeepEqual(page, tc.expectedPage) {
				t.Error("unexpected metrics")
			}
		})
	}
}

func TestListBaselineStatusCounts(t *testing.T) {
	// nolint:dupl // WONTFIX - not exactly the same as TestListMissingOneImplCounts
	testCases := []struct {
		name         string
		cfg          mockListBaselineStatusCountsConfig
		expectedPage *backend.BaselineStatusMetricsPage
		expectedErr  error
	}{
		{
			name: "success",
			cfg: mockListBaselineStatusCountsConfig{
				result: &gcpspanner.BaselineStatusCountResultPage{
					NextPageToken: nonNilNextPageToken,
					Metrics: []gcpspanner.BaselineStatusCountMetric{
						{
							StatusCount: 89,
							Date:        time.Date(2010, time.January, 10, 0, 0, 0, 0, time.UTC),
						},
						{
							StatusCount: 99,
							Date:        time.Date(2010, time.January, 9, 0, 0, 0, 0, time.UTC),
						},
					},
				},
				returnedError: nil,
			},
			expectedPage: &backend.BaselineStatusMetricsPage{
				Metadata: &backend.PageMetadata{
					NextPageToken: nonNilNextPageToken,
				},
				Data: []backend.BaselineStatusMetric{
					{
						Count:     new(int64(89)),
						Timestamp: time.Date(2010, time.January, 10, 0, 0, 0, 0, time.UTC),
					},
					{
						Count:     new(int64(99)),
						Timestamp: time.Date(2010, time.January, 9, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "failure",
			cfg: mockListBaselineStatusCountsConfig{
				result:        nil,
				returnedError: errTest,
			},
			expectedPage: nil,
			expectedErr:  errTest,
		},
		{
			name: "invalid cursor",
			cfg: mockListBaselineStatusCountsConfig{
				result:        nil,
				returnedError: gcpspanner.ErrInvalidCursorFormat,
			},
			expectedPage: nil,
			expectedErr:  backendtypes.ErrInvalidPageToken,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                               t,
				mockListBaselineStatusCountsCfg: tc.cfg,
			}
			backend := NewBackend(mock)
			page, err := backend.ListBaselineStatusCounts(
				context.Background(),
				testStart,
				testEnd,
				100,
				nonNilInputPageToken)
			if !errors.Is(err, tc.expectedErr) {
				t.Error("unexpected error")
			}

			if !reflect.DeepEqual(page, tc.expectedPage) {
				t.Error("unexpected metrics")
			}
		})
	}
}

func TestConvertBaselineStatusBackendToSpanner(t *testing.T) {
	var backendToSpannerTests = []struct {
		name     string
		input    backend.BaselineInfoStatus
		expected gcpspanner.BaselineStatus
	}{
		{"Widely to High", backend.Widely, gcpspanner.BaselineStatusHigh},
		{"Newly to Low", backend.Newly, gcpspanner.BaselineStatusLow},
		{"Limited to None", backend.Limited, gcpspanner.BaselineStatusNone},
		{"Invalid to Undefined", backend.BaselineInfoStatus("invalid"),
			""}, // Test default case
	}
	for _, tt := range backendToSpannerTests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertBaselineStatusBackendToSpanner(tt.input)
			if result != tt.expected {
				t.Errorf("convertBaselineStatusBackendToSpanner(%v): got %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvertBaselineStatusSpannerToBackend(t *testing.T) {
	var spannerToBackendTests = []struct {
		name          string
		inputStatus   *string
		inputLowDate  *time.Time
		inputHighDate *time.Time
		expected      *backend.BaselineInfo
	}{
		{
			name:          "High Status to Widely",
			inputStatus:   new("high"),
			inputLowDate:  nil,
			inputHighDate: nil,
			expected: &backend.BaselineInfo{
				Status:   new(backend.Widely),
				LowDate:  nil,
				HighDate: nil,
			},
		},
		{
			name:          "Low Status to Newly",
			inputStatus:   new("low"),
			inputLowDate:  nil,
			inputHighDate: nil,
			expected: &backend.BaselineInfo{
				Status:   new(backend.Newly),
				LowDate:  nil,
				HighDate: nil,
			},
		},
		{
			name:          "None Status to Limited",
			inputStatus:   new("none"),
			inputLowDate:  nil,
			inputHighDate: nil,
			expected: &backend.BaselineInfo{
				Status:   new(backend.Limited),
				LowDate:  nil,
				HighDate: nil,
			},
		},
		{
			name:          "Status with Low Date",
			inputStatus:   new("none"),
			inputLowDate:  new(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)),
			inputHighDate: nil,
			expected: &backend.BaselineInfo{
				Status:   new(backend.Limited),
				LowDate:  &openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
				HighDate: nil,
			},
		},
		{
			name:          "Status with Low Date & High Date",
			inputStatus:   new("none"),
			inputLowDate:  new(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)),
			inputHighDate: new(time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)),
			expected: &backend.BaselineInfo{
				Status:   new(backend.Limited),
				LowDate:  &openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
				HighDate: &openapi_types.Date{Time: time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)},
			},
		},
		{
			name:          "Invalid Status to nil",
			inputStatus:   new("invalid"),
			inputLowDate:  nil,
			inputHighDate: nil,
			expected:      nil,
		}, // Test default case
	}
	for _, tt := range spannerToBackendTests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertBaselineSpannerToBackend(tt.inputStatus, tt.inputLowDate, tt.inputHighDate)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("convertBaselineSpannerToBackend(%v %v %v): got %v, want %v", tt.inputStatus, tt.inputLowDate,
					tt.inputHighDate, result, tt.expected)
			}
		})
	}
}

func TestFeaturesSearch(t *testing.T) {
	testCases := []struct {
		name               string
		cfg                mockFeaturesSearchConfig
		inputPageToken     *string
		inputPageSize      int
		inputWPTMetricView backend.WPTMetricView
		inputBrowsers      BrowserList
		searchNode         *searchtypes.SearchNode
		sortOrder          *backend.ListFeaturesParamsSort
		expectedPage       *backend.FeaturePage
	}{
		{
			name: "regular",
			cfg: mockFeaturesSearchConfig{
				expectedPageToken: nonNilInputPageToken,
				expectedPageSize:  100,
				expectedNode: &searchtypes.SearchNode{
					Keyword:  searchtypes.KeywordRoot,
					Term:     nil,
					Children: nil,
				},
				expectedWPTMetricView: gcpspanner.WPTSubtestView,
				expectedSortable:      gcpspanner.NewBaselineStatusSort(false),
				expectedBrowsers: []string{
					"browser1",
					"browser2",
					"browser3",
				},
				result: &gcpspanner.FeatureResultPage{
					Total:         100,
					NextPageToken: nonNilNextPageToken,
					Features: []gcpspanner.FeatureResult{
						{
							Name:       "feature 1",
							FeatureKey: "feature1",
							Status:     new("low"),
							LowDate:    new(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)),
							HighDate:   nil,
							StableMetrics: []*gcpspanner.FeatureResultMetric{
								{
									BrowserName:       "browser3",
									PassRate:          big.NewRat(10, 20),
									FeatureRunDetails: nil,
								},
							},
							ExperimentalMetrics: []*gcpspanner.FeatureResultMetric{
								{
									BrowserName:       "browser3",
									PassRate:          big.NewRat(10, 50),
									FeatureRunDetails: nil,
								},
							},
							ImplementationStatuses: []*gcpspanner.ImplementationStatus{
								{
									BrowserName:          "browser3",
									ImplementationStatus: gcpspanner.Available,
									ImplementationDate: new(
										time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC)),
									ImplementationVersion: new("103"),
								},
							},
							SpecLinks:              nil,
							ChromiumUsage:          big.NewRat(91, 100),
							DeveloperSignalUpvotes: new(int64(9)),
							DeveloperSignalLink:    new("http://example.com"),
							AccordingTo: []string{
								"accordingto1",
								"accordingto2",
							},
							Alternatives: []string{
								"alternative1",
								"alternative2",
							},
							VendorPositions:            spanner.NullJSON{Value: nil, Valid: false},
							SystemManagedSavedSearchID: nil,
						},

						{
							Name:       "feature 2",
							FeatureKey: "feature2",
							Status:     new("high"),
							LowDate:    new(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)),
							HighDate:   new(time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)),
							StableMetrics: []*gcpspanner.FeatureResultMetric{
								{
									BrowserName:       "browser1",
									PassRate:          big.NewRat(10, 20),
									FeatureRunDetails: nil,
								},
								{
									BrowserName:       "browser2",
									PassRate:          big.NewRat(5, 20),
									FeatureRunDetails: nil,
								},
							},
							ExperimentalMetrics: []*gcpspanner.FeatureResultMetric{
								{
									BrowserName:       "browser1",
									PassRate:          big.NewRat(10, 20),
									FeatureRunDetails: nil,
								},
								{
									BrowserName: "browser2",
									PassRate:    big.NewRat(2, 20),
									FeatureRunDetails: map[string]any{
										"test": "browser2-exp",
									},
								},
							},
							ImplementationStatuses: []*gcpspanner.ImplementationStatus{
								{
									BrowserName:          "browser1",
									ImplementationStatus: gcpspanner.Available,
									ImplementationDate: new(
										time.Date(1998, time.January, 1, 0, 0, 0, 0, time.UTC)),
									ImplementationVersion: new("101"),
								},
								{
									BrowserName:          "browser2",
									ImplementationStatus: gcpspanner.Available,
									ImplementationDate: new(
										time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC)),
									ImplementationVersion: new("102"),
								},
							},
							SpecLinks: []string{
								"link1",
								"link2",
							},
							ChromiumUsage:          big.NewRat(10, 100),
							DeveloperSignalUpvotes: nil,
							DeveloperSignalLink:    nil,
							AccordingTo: []string{
								"accordingto3",
								"accordingto4",
							},
							Alternatives:               nil,
							VendorPositions:            spanner.NullJSON{Value: nil, Valid: false},
							SystemManagedSavedSearchID: nil,
						},
					},
				},
				returnedError: nil,
			},
			inputPageToken: nonNilInputPageToken,
			inputPageSize:  100,
			searchNode: &searchtypes.SearchNode{
				Keyword:  searchtypes.KeywordRoot,
				Term:     nil,
				Children: nil,
			},
			sortOrder:          nil,
			inputWPTMetricView: backend.SubtestCounts,
			inputBrowsers: []backend.BrowserPathParam{
				"browser1",
				"browser2",
				"browser3",
			},
			expectedPage: &backend.FeaturePage{
				Metadata: backend.PageMetadataWithTotal{
					NextPageToken: nonNilNextPageToken,
					Total:         100,
				},
				Data: []backend.Feature{
					{
						Baseline: &backend.BaselineInfo{
							Status: new(backend.Newly),
							LowDate: new(
								openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
							),
							HighDate: nil,
						},
						FeatureId: "feature1",
						Name:      "feature 1",
						Spec:      nil,
						Usage: &backend.BrowserUsage{
							Chrome: &backend.ChromeUsageInfo{
								Daily: new(0.91),
							},
						},
						Wpt: &backend.FeatureWPTSnapshots{
							Experimental: &map[string]backend.WPTFeatureData{
								"browser3": {
									Score:    new(0.2),
									Metadata: nil,
								},
							},
							Stable: &map[string]backend.WPTFeatureData{
								"browser3": {
									Score:    new(0.5),
									Metadata: nil,
								},
							},
						},
						BrowserImplementations: &map[string]backend.BrowserImplementation{
							"browser3": {
								Status: new(backend.Available),
								Date: &openapi_types.Date{
									Time: time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC)},
								Version: new("103"),
							},
						},
						DeveloperSignals: &backend.FeatureDeveloperSignals{
							Upvotes: new(int64(9)),
							Link:    new("http://example.com"),
						},
						Discouraged: &backend.FeatureDiscouragedInfo{
							AccordingTo: &[]backend.FeatureDiscouragedAccordingTo{
								{
									Link: "accordingto1",
								},
								{
									Link: "accordingto2",
								},
							},
							Alternatives: &[]backend.FeatureDiscouragedAlternative{
								{
									Id: "alternative1",
								},
								{
									Id: "alternative2",
								},
							},
						},
						VendorPositions:            nil,
						SystemManagedSavedSearchId: nil,
					},
					{
						Baseline: &backend.BaselineInfo{
							Status: new(backend.Widely),
							LowDate: new(
								openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
							),
							HighDate: new(
								openapi_types.Date{Time: time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)},
							),
						},
						FeatureId: "feature2",
						Name:      "feature 2",
						Spec: &backend.FeatureSpecInfo{
							Links: &[]backend.SpecLink{
								{
									Link: new("link1"),
								},
								{
									Link: new("link2"),
								},
							},
						},
						Usage: &backend.BrowserUsage{
							Chrome: &backend.ChromeUsageInfo{
								Daily: new(0.1),
							},
						},
						Wpt: &backend.FeatureWPTSnapshots{
							Experimental: &map[string]backend.WPTFeatureData{
								"browser1": {
									Score:    new(0.5),
									Metadata: nil,
								},
								"browser2": {
									Score: new(0.1),
									Metadata: &map[string]any{
										"test": "browser2-exp",
									},
								},
							},
							Stable: &map[string]backend.WPTFeatureData{
								"browser1": {
									Score:    new(0.5),
									Metadata: nil,
								},
								"browser2": {
									Score:    new(0.25),
									Metadata: nil,
								},
							},
						},
						BrowserImplementations: &map[string]backend.BrowserImplementation{
							"browser1": {
								Status: new(backend.Available),
								Date: &openapi_types.Date{
									Time: time.Date(1998, time.January, 1, 0, 0, 0, 0, time.UTC)},
								Version: new("101"),
							},
							"browser2": {
								Status: new(backend.Available),
								Date: &openapi_types.Date{
									Time: time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC)},
								Version: new("102"),
							},
						},
						DeveloperSignals: nil,
						Discouraged: &backend.FeatureDiscouragedInfo{
							AccordingTo: &[]backend.FeatureDiscouragedAccordingTo{
								{
									Link: "accordingto3",
								},
								{
									Link: "accordingto4",
								},
							},
							Alternatives: nil,
						},
						VendorPositions:            nil,
						SystemManagedSavedSearchId: nil,
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                     t,
				mockFeaturesSearchCfg: tc.cfg,
			}
			bk := NewBackend(mock)
			page, err := bk.FeaturesSearch(
				context.Background(),
				tc.inputPageToken,
				tc.inputPageSize,
				tc.searchNode,
				tc.sortOrder,
				tc.inputWPTMetricView,
				tc.inputBrowsers)
			if !errors.Is(err, tc.cfg.returnedError) {
				t.Error("unexpected error")
			}

			if diff := cmp.Diff(tc.expectedPage, page); diff != "" {
				t.Errorf("page mismatch (-want +got):\n%s", diff)
			}

		})
	}
}

// CompareFeatures checks if two backend.Feature structs are deeply equal.
func CompareFeatures(f1, f2 backend.Feature) bool {
	// 1. Basic Equality Checks
	if f1.FeatureId != f2.FeatureId ||
		f1.Name != f2.Name {
		return false
	}

	// 2. Compare 'spec' (slice of strings)
	if !reflect.DeepEqual(f1.Spec, f2.Spec) {
		return false
	}

	// 3. Compare FeatureWPTSnapshots (nested structs)
	if !compareWPTSnapshots(f1.Wpt, f2.Wpt) {
		return false
	}

	if !compareImplementationStatus(f1.BrowserImplementations, f2.BrowserImplementations) {
		return false
	}

	// 4. Compare Baseline Objects
	if !reflect.DeepEqual(f1.Baseline, f2.Baseline) {
		return false
	}

	if !compareChromeUsage(*f1.Usage.Chrome, *f2.Usage.Chrome) {
		return false
	}

	if !compareDeveloperSignals(f1.DeveloperSignals, f2.DeveloperSignals) {
		return false
	}

	// All fields match
	return true
}

func compareChromeUsage(c1, c2 backend.ChromeUsageInfo) bool {
	return reflect.DeepEqual(c1.Daily, c2.Daily)
}

func compareImplementationStatus(s1, s2 *map[string]backend.BrowserImplementation) bool {
	return reflect.DeepEqual(s1, s2)
}

// compareWPTSnapshots helps compare FeatureWPTSnapshots structs.
func compareWPTSnapshots(w1, w2 *backend.FeatureWPTSnapshots) bool {
	// Handle nil cases
	if (w1 == nil && w2 != nil) || (w1 != nil && w2 == nil) {
		return false
	}

	if w1 == nil && w2 == nil { // Both nil
		return true
	}

	// Compare 'Experimental' maps
	if !compareFeatureDataMap(w1.Experimental, w2.Experimental) {
		return false
	}

	// Compare 'Stable' maps
	if !compareFeatureDataMap(w1.Stable, w2.Stable) {
		return false
	}

	return true
}

func compareDeveloperSignals(s1, s2 *backend.FeatureDeveloperSignals) bool {
	return reflect.DeepEqual(s1, s2)
}

// compareFeatureDataMap helps compare maps of WPTFeatureData.
func compareFeatureDataMap(m1, m2 *map[string]backend.WPTFeatureData) bool {
	// Handle nil cases
	if (m1 == nil && m2 != nil) || (m1 != nil && m2 == nil) {
		return false
	}

	if m1 == nil && m2 == nil { // Both nil
		return true
	}

	// Check if lengths are equal
	if len(*m1) != len(*m2) {
		return false
	}

	// Compare each key-value pair
	for k, v1 := range *m1 {
		v2, ok := (*m2)[k]
		if !ok || *v1.Score != *v2.Score {
			return false
		}
	}

	return true
}

func TestGetFeature(t *testing.T) {
	const (
		defaultFeatureID  = "feature1"
		defaultMetricView = backend.SubtestCounts
	)
	var (
		defaultInputBrowsers = []backend.BrowserPathParam{
			"browser1",
			"browser2",
			"browser3",
		}
	)
	testCases := []struct {
		name            string
		cfg             mockGetFeatureConfig
		movedFeatureCfg *mockGetMovedWebFeatureDetailsByOriginalFeatureKeyConfig
		splitFeatureCfg *mockGetSplitWebFeatureByOriginalFeatureKeyConfig
		visitor         func(t *testing.T) backendtypes.FeatureResultVisitor
		expectedError   error
	}{
		{
			name: "regular",
			cfg: mockGetFeatureConfig{
				expectedFilterable:    gcpspanner.NewFeatureKeyFilter("feature1"),
				expectedWPTMetricView: gcpspanner.WPTSubtestView,
				expectedBrowsers: []string{
					"browser1",
					"browser2",
					"browser3",
				},
				result: &gcpspanner.FeatureResult{
					Name:       "feature 1",
					FeatureKey: "feature1",
					Status:     new("low"),
					LowDate:    new(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)),
					HighDate:   nil,
					StableMetrics: []*gcpspanner.FeatureResultMetric{
						{
							BrowserName: "browser3",
							PassRate:    big.NewRat(10, 20),
							FeatureRunDetails: map[string]any{
								"browser3": "test",
							},
						},
					},
					ExperimentalMetrics: []*gcpspanner.FeatureResultMetric{
						{
							BrowserName:       "browser3",
							PassRate:          big.NewRat(10, 50),
							FeatureRunDetails: nil,
						},
					},
					ImplementationStatuses: []*gcpspanner.ImplementationStatus{
						{
							BrowserName:          "browser3",
							ImplementationStatus: gcpspanner.Available,
						},
					},
					SpecLinks: []string{
						"link1",
						"link2",
					},
					ChromiumUsage:              nil,
					DeveloperSignalUpvotes:     new(int64(4)),
					DeveloperSignalLink:        new("http://example.com"),
					Alternatives:               []string{"alternative1", "alternative2"},
					AccordingTo:                []string{"according1", "according2"},
					VendorPositions:            spanner.NullJSON{Value: nil, Valid: false},
					SystemManagedSavedSearchID: nil,
				},
				returnedError: nil,
			}, splitFeatureCfg: nil,
			movedFeatureCfg: nil,
			visitor: func(t *testing.T) backendtypes.FeatureResultVisitor {
				return &TestRegularFeatureVisitor{
					t: t,
					expected: backendtypes.NewRegularFeatureResult(&backend.Feature{
						Baseline: &backend.BaselineInfo{
							Status: new(backend.Newly),
							LowDate: new(
								openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
							),
							HighDate: nil,
						},
						FeatureId: "feature1",
						Name:      "feature 1",
						Spec: &backend.FeatureSpecInfo{
							Links: &[]backend.SpecLink{
								{
									Link: new("link1"),
								},
								{
									Link: new("link2"),
								},
							},
						},
						Usage: &backend.BrowserUsage{
							Chrome: &backend.ChromeUsageInfo{
								Daily: nil,
							},
						},
						Wpt: &backend.FeatureWPTSnapshots{
							Experimental: &map[string]backend.WPTFeatureData{
								"browser3": {
									Score:    new(0.2),
									Metadata: nil,
								},
							},
							Stable: &map[string]backend.WPTFeatureData{
								"browser3": {
									Score: new(0.5),
									Metadata: &map[string]any{
										"browser3": "test",
									},
								},
							},
						},
						BrowserImplementations: &map[string]backend.BrowserImplementation{
							"browser3": {
								Status:  new(backend.Available),
								Date:    nil,
								Version: nil,
							},
						},
						DeveloperSignals: &backend.FeatureDeveloperSignals{
							Upvotes: new(int64(4)),
							Link:    new("http://example.com"),
						},
						Discouraged: &backend.FeatureDiscouragedInfo{
							Alternatives: &[]backend.FeatureDiscouragedAlternative{
								{
									Id: "alternative1",
								},
								{
									Id: "alternative2",
								},
							},
							AccordingTo: &[]backend.FeatureDiscouragedAccordingTo{
								{
									Link: "according1",
								},
								{
									Link: "according2",
								},
							},
						},
						VendorPositions:            nil,
						SystemManagedSavedSearchId: nil,
					}),
				}
			},
			expectedError: nil,
		},
		{
			name: "moved",
			cfg: mockGetFeatureConfig{
				expectedFilterable:    gcpspanner.NewFeatureKeyFilter("feature1"),
				expectedWPTMetricView: gcpspanner.WPTSubtestView,
				expectedBrowsers: []string{
					"browser1",
					"browser2",
					"browser3",
				},
				result:        nil,
				returnedError: gcpspanner.ErrQueryReturnedNoResults,
			},
			splitFeatureCfg: nil,
			movedFeatureCfg: &mockGetMovedWebFeatureDetailsByOriginalFeatureKeyConfig{
				expectedFeatureKey: "feature1",
				result: &gcpspanner.MovedWebFeature{
					OriginalFeatureKey: "feature1",
					NewFeatureKey:      "feature2",
				},
				returnedError: nil,
			},
			visitor: func(t *testing.T) backendtypes.FeatureResultVisitor {
				return &TestMovedFeatureVisitor{
					t:        t,
					expected: *backendtypes.NewMovedFeatureResult("feature2"),
				}
			},
			expectedError: nil,
		},
		{
			name: "split",
			cfg: mockGetFeatureConfig{
				expectedFilterable:    gcpspanner.NewFeatureKeyFilter("feature1"),
				expectedWPTMetricView: gcpspanner.WPTSubtestView,
				expectedBrowsers: []string{
					"browser1",
					"browser2",
					"browser3",
				},
				result:        nil,
				returnedError: gcpspanner.ErrQueryReturnedNoResults,
			},
			splitFeatureCfg: &mockGetSplitWebFeatureByOriginalFeatureKeyConfig{
				expectedFeatureKey: "feature1",
				result: &gcpspanner.SplitWebFeature{
					OriginalFeatureKey: "feature1",
					TargetFeatureKeys:  []string{"feature2", "feature3"},
				},
				returnedError: nil,
			},
			movedFeatureCfg: &mockGetMovedWebFeatureDetailsByOriginalFeatureKeyConfig{
				expectedFeatureKey: "feature1",
				result:             nil,
				returnedError:      gcpspanner.ErrQueryReturnedNoResults,
			},
			visitor: func(t *testing.T) backendtypes.FeatureResultVisitor {
				return &TestSplitFeatureVisitor{
					t: t,
					expected: *backendtypes.NewSplitFeatureResult(
						backend.FeatureEvolutionSplit{
							Features: []backend.FeatureSplitInfo{
								{Id: "feature2"},
								{Id: "feature3"},
							},
						},
					),
				}
			},
			expectedError: nil,
		},
		{
			name: "feature not found",
			cfg: mockGetFeatureConfig{
				expectedFilterable:    gcpspanner.NewFeatureKeyFilter("feature1"),
				expectedWPTMetricView: gcpspanner.WPTSubtestView,
				expectedBrowsers: []string{
					"browser1",
					"browser2",
					"browser3",
				},
				result:        nil,
				returnedError: gcpspanner.ErrQueryReturnedNoResults,
			},
			splitFeatureCfg: &mockGetSplitWebFeatureByOriginalFeatureKeyConfig{
				expectedFeatureKey: "feature1",
				result: &gcpspanner.SplitWebFeature{
					OriginalFeatureKey: "feature1",
					TargetFeatureKeys:  []string{"feature2", "feature3"},
				},
				returnedError: gcpspanner.ErrQueryReturnedNoResults,
			},
			movedFeatureCfg: &mockGetMovedWebFeatureDetailsByOriginalFeatureKeyConfig{
				expectedFeatureKey: "feature1",
				result:             nil,
				returnedError:      gcpspanner.ErrQueryReturnedNoResults,
			},
			visitor:       nil,
			expectedError: backendtypes.ErrEntityDoesNotExist,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                 t,
				mockGetFeatureCfg: tc.cfg,
				mockGetMovedWebFeatureDetailsByOriginalFeatureKeyCfg: tc.movedFeatureCfg,
				mockGetSplitWebFeatureByOriginalFeatureKeyCfg:        tc.splitFeatureCfg,
			}
			bk := NewBackend(mock)
			feature, err := bk.GetFeature(
				context.Background(),
				defaultFeatureID, defaultMetricView, defaultInputBrowsers)
			if !errors.Is(err, tc.expectedError) {
				t.Error("unexpected error")
			}
			if tc.visitor == nil {
				return
			}
			err = feature.Visit(t.Context(), tc.visitor(t))
			if err != nil {
				t.Error("unexpected error")
			}
		})
	}
}

// TestRegularFeatureVisitor expects a RegularFeatureResult and compares it.
// Other Visit methods will cause an error.
type TestRegularFeatureVisitor struct {
	t        *testing.T
	expected *backendtypes.RegularFeatureResult
}

func (v *TestRegularFeatureVisitor) VisitRegularFeature(_ context.Context,
	actual backendtypes.RegularFeatureResult) error {
	if !CompareFeatures(*actual.Feature(), *v.expected.Feature()) {
		v.t.Error("unexpected feature")
	}

	return nil
}

func (v *TestRegularFeatureVisitor) VisitMovedFeature(_ context.Context, actual backendtypes.MovedFeatureResult) error {
	v.t.Errorf("VisitMovedFeature called unexpectedly for a RegularFeature test. Actual: %+v", actual)

	return nil
}

func (v *TestRegularFeatureVisitor) VisitSplitFeature(_ context.Context, actual backendtypes.SplitFeatureResult) error {
	v.t.Errorf("VisitSplitFeature called unexpectedly for a RegularFeature test. Actual: %+v", actual)

	return nil
}

// TestMovedFeatureVisitor expects a MovedFeatureResult and compares it.
// Other Visit methods will cause an error.
type TestMovedFeatureVisitor struct {
	t        *testing.T
	expected backendtypes.MovedFeatureResult
}

func (v *TestMovedFeatureVisitor) VisitMovedFeature(_ context.Context, actual backendtypes.MovedFeatureResult) error {
	if !reflect.DeepEqual(v.expected, actual) {
		v.t.Errorf("MovedFeature mismatch:\nExpected: %+v\nActual:   %+v", v.expected, actual)
	}

	return nil
}

func (v *TestMovedFeatureVisitor) VisitRegularFeature(_ context.Context,
	actual backendtypes.RegularFeatureResult) error {
	v.t.Errorf("VisitRegularFeature called unexpectedly for a MovedFeature test. Actual: %+v", actual)

	return nil
}

func (v *TestMovedFeatureVisitor) VisitSplitFeature(_ context.Context, actual backendtypes.SplitFeatureResult) error {
	v.t.Errorf("VisitSplitFeature called unexpectedly for a MovedFeature test. Actual: %+v", actual)

	return nil
}

// TestSplitFeatureVisitor expects a SplitFeatureResult and compares it.
// Other Visit methods will cause an error.
type TestSplitFeatureVisitor struct {
	t        *testing.T
	expected backendtypes.SplitFeatureResult
}

func (v *TestSplitFeatureVisitor) VisitSplitFeature(_ context.Context, actual backendtypes.SplitFeatureResult) error {
	if !reflect.DeepEqual(v.expected, actual) {
		v.t.Errorf("SplitFeature mismatch:\nExpected: %+v\nActual:   %+v", v.expected, actual)
	}

	return nil
}

func (v *TestSplitFeatureVisitor) VisitRegularFeature(_ context.Context,
	actual backendtypes.RegularFeatureResult) error {
	v.t.Errorf("VisitRegularFeature called unexpectedly for a SplitFeature test. Actual: %+v", actual)

	return nil
}

func (v *TestSplitFeatureVisitor) VisitMovedFeature(_ context.Context, actual backendtypes.MovedFeatureResult) error {
	v.t.Errorf("VisitMovedFeature called unexpectedly for a SplitFeature test. Actual: %+v", actual)

	return nil
}

func TestGetNotificationChannel(t *testing.T) {
	const (
		userID    = "user123"
		channelID = "channel456"
	)
	now := time.Now()

	testCases := []struct {
		name          string
		cfg           *mockGetNotificationChannelConfig
		expected      *backend.NotificationChannelResponse
		expectedError error
	}{
		{
			name: "success",
			cfg: &mockGetNotificationChannelConfig{
				expectedChannelID: channelID,
				expectedUserID:    userID,
				result: &gcpspanner.NotificationChannel{
					ID:     channelID,
					UserID: userID,
					Name:   "My Email",
					Type:   "email",
					EmailConfig: &gcpspanner.EmailConfig{
						Address:           "test@example.com",
						IsVerified:        false,
						VerificationToken: nil,
					},
					WebhookConfig: nil,
					CreatedAt:     now,
					UpdatedAt:     now,
				},
				returnedError: nil,
			},
			expected: &backend.NotificationChannelResponse{
				Id:   channelID,
				Name: "My Email",
				Type: backend.NotificationChannelResponseTypeEmail,
				Config: newTestNotificationChannelConfig(t, backend.EmailConfig{
					Type:    backend.EmailConfigTypeEmail,
					Address: "test@example.com",
				}),
				Status:    backend.NotificationChannelStatusEnabled,
				CreatedAt: now,
				UpdatedAt: now,
			},

			expectedError: nil,
		},
		{
			name: "not found",
			cfg: &mockGetNotificationChannelConfig{
				expectedChannelID: channelID,
				expectedUserID:    userID,
				result:            nil,
				returnedError:     gcpspanner.ErrQueryReturnedNoResults,
			},
			expected:      nil,
			expectedError: backendtypes.ErrEntityDoesNotExist,
		},
		{
			name: "not authorized",
			cfg: &mockGetNotificationChannelConfig{
				expectedChannelID: channelID,
				expectedUserID:    userID,
				result:            nil,
				returnedError:     gcpspanner.ErrMissingRequiredRole,
			},
			expected:      nil,
			expectedError: backendtypes.ErrUserNotAuthorizedForAction,
		},
		{
			name: "other error",
			cfg: &mockGetNotificationChannelConfig{
				expectedChannelID: channelID,
				expectedUserID:    userID,
				result:            nil,
				returnedError:     errTest,
			},
			expected:      nil,
			expectedError: errTest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                             t,
				mockGetNotificationChannelCfg: tc.cfg,
			}
			b := NewBackend(mock)
			resp, err := b.GetNotificationChannel(context.Background(), userID, channelID)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error. got %v, want %v", err, tc.expectedError)
			}
			if diff := cmp.Diff(tc.expected, resp, getNotificationChannelCmpOption()); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateNotificationChannel(t *testing.T) {
	const (
		userID    = "user123"
		channelID = "channel456"
	)
	now := time.Now()

	testCases := []struct {
		name              string
		req               backend.CreateNotificationChannelRequest
		cfg               *mockCreateNotificationChannelConfig
		expected          *backend.NotificationChannelResponse
		expectedGetResult *gcpspanner.NotificationChannel
		expectedError     error
	}{
		{
			name: "success webhook",
			req: backend.CreateNotificationChannelRequest{
				Name: "My Webhook",
				Config: newTestCreateNotificationChannelConfig(t, backend.WebhookConfig{
					Type: backend.Webhook,
					Url:  "https://hooks.slack.com/services/123",
				}),
			},

			cfg: &mockCreateNotificationChannelConfig{
				expectedRequest: gcpspanner.CreateNotificationChannelRequest{
					UserID:      userID,
					Name:        "My Webhook",
					Type:        gcpspanner.NotificationChannelTypeWebhook,
					EmailConfig: nil,
					WebhookConfig: &gcpspanner.WebhookConfig{
						URL: "https://hooks.slack.com/services/123",
					},
				},

				result:        new(channelID),
				returnedError: nil,
			},
			expectedGetResult: &gcpspanner.NotificationChannel{
				ID:            channelID,
				UserID:        userID,
				Name:          "My Webhook",
				Type:          gcpspanner.NotificationChannelTypeWebhook,
				EmailConfig:   nil,
				WebhookConfig: &gcpspanner.WebhookConfig{URL: "https://hooks.slack.com/services/123"},
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			expected: &backend.NotificationChannelResponse{
				Id:   channelID,
				Name: "My Webhook",
				Type: backend.NotificationChannelResponseTypeWebhook,
				Config: newTestNotificationChannelConfig(t, backend.WebhookConfig{
					Type: backend.Webhook,
					Url:  "https://hooks.slack.com/services/123",
				}),
				Status:    backend.NotificationChannelStatusEnabled,
				CreatedAt: now,
				UpdatedAt: now,
			},

			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mock := &mockBackendSpannerClient{
				t:                                t,
				mockCreateNotificationChannelCfg: tc.cfg,
				// Mock the Get call that happens after Create
				mockGetNotificationChannelCfg: &mockGetNotificationChannelConfig{
					expectedUserID:    userID,
					expectedChannelID: channelID,
					result:            tc.expectedGetResult,
					returnedError:     nil,
				},
			}
			b := NewBackend(mock)
			resp, err := b.CreateNotificationChannel(context.Background(), userID, tc.req)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error. got %v, want %v", err, tc.expectedError)
			}
			if diff := cmp.Diff(tc.expected, resp, getNotificationChannelCmpOption()); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUpdateNotificationChannel(t *testing.T) {
	const (
		userID    = "user123"
		channelID = "channel456"
	)
	now := time.Now()

	testCases := []struct {
		name              string
		req               backend.UpdateNotificationChannelRequest
		cfg               *mockUpdateNotificationChannelConfig
		expected          *backend.NotificationChannelResponse
		expectedGetResult *gcpspanner.NotificationChannel
		expectedError     error
	}{
		{
			name: "success update name",
			req: backend.UpdateNotificationChannelRequest{
				UpdateMask: []backend.UpdateNotificationChannelRequestUpdateMask{
					backend.UpdateNotificationChannelRequestMaskName,
				},
				Name:   new("New Name"),
				Config: nil,
			},
			cfg: &mockUpdateNotificationChannelConfig{
				expectedRequest: gcpspanner.UpdateNotificationChannelRequest{
					ID:     channelID,
					UserID: userID,
					Name:   gcpspanner.OptionallySet[string]{Value: "New Name", IsSet: true},
					Type: gcpspanner.OptionallySet[gcpspanner.NotificationChannelType]{
						Value: "",
						IsSet: false,
					},
					EmailConfig:   gcpspanner.OptionallySet[*gcpspanner.EmailConfig]{Value: nil, IsSet: false},
					WebhookConfig: gcpspanner.OptionallySet[*gcpspanner.WebhookConfig]{Value: nil, IsSet: false},
				},
				returnedError: nil,
			},
			expectedGetResult: &gcpspanner.NotificationChannel{
				ID:            channelID,
				UserID:        userID,
				Name:          "New Name",
				Type:          gcpspanner.NotificationChannelTypeWebhook,
				EmailConfig:   nil,
				WebhookConfig: &gcpspanner.WebhookConfig{URL: "https://hooks.slack.com/services/123"},
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			expected: &backend.NotificationChannelResponse{
				Id:   channelID,
				Name: "New Name",
				Type: backend.NotificationChannelResponseTypeWebhook,
				Config: newTestNotificationChannelConfig(t, backend.WebhookConfig{
					Type: backend.Webhook,
					Url:  "https://hooks.slack.com/services/123",
				}),
				Status:    backend.NotificationChannelStatusEnabled,
				CreatedAt: now,
				UpdatedAt: now,
			},

			expectedError: nil,
		},
		{
			name: "error update email channel",
			req: backend.UpdateNotificationChannelRequest{
				UpdateMask: []backend.UpdateNotificationChannelRequestUpdateMask{
					backend.UpdateNotificationChannelRequestMaskName,
				},
				Name:   new("New Name"),
				Config: nil,
			},
			cfg: nil, // Should not be called
			expectedGetResult: &gcpspanner.NotificationChannel{
				ID:     channelID,
				UserID: userID,
				Name:   "Old Name",
				Type:   gcpspanner.NotificationChannelTypeEmail,
				EmailConfig: &gcpspanner.EmailConfig{
					Address:           "test@example.com",
					IsVerified:        false,
					VerificationToken: nil,
				},
				WebhookConfig: nil,
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			expected:      nil,
			expectedError: backendtypes.ErrUserNotAuthorizedForAction,
		},
		{
			name: "success update config to webhook",
			req: backend.UpdateNotificationChannelRequest{
				UpdateMask: []backend.UpdateNotificationChannelRequestUpdateMask{
					backend.UpdateNotificationChannelRequestMaskConfig,
				},
				Name: nil,
				Config: newTestUpdateNotificationChannelRequestConfig(t, backend.WebhookConfig{
					Type: backend.Webhook,
					Url:  "https://hooks.slack.com/services/456",
				}),
			},
			cfg: &mockUpdateNotificationChannelConfig{
				expectedRequest: gcpspanner.UpdateNotificationChannelRequest{
					ID:     channelID,
					UserID: userID,
					Name:   gcpspanner.OptionallySet[string]{Value: "", IsSet: false},
					Type: gcpspanner.OptionallySet[gcpspanner.NotificationChannelType]{
						Value: gcpspanner.NotificationChannelTypeWebhook,
						IsSet: true,
					},
					EmailConfig: gcpspanner.OptionallySet[*gcpspanner.EmailConfig]{Value: nil, IsSet: false},
					WebhookConfig: gcpspanner.OptionallySet[*gcpspanner.WebhookConfig]{
						Value: &gcpspanner.WebhookConfig{URL: "https://hooks.slack.com/services/456"},
						IsSet: true,
					},
				},
				returnedError: nil,
			},
			expectedGetResult: &gcpspanner.NotificationChannel{
				ID:            channelID,
				UserID:        userID,
				Name:          "My Email",
				Type:          gcpspanner.NotificationChannelTypeWebhook,
				EmailConfig:   nil,
				WebhookConfig: &gcpspanner.WebhookConfig{URL: "https://hooks.slack.com/services/456"},
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			expected: &backend.NotificationChannelResponse{
				Id:   channelID,
				Name: "My Email", // Name didn't change
				Type: backend.NotificationChannelResponseTypeWebhook,
				Config: newTestNotificationChannelConfig(t, backend.WebhookConfig{
					Type: backend.Webhook,
					Url:  "https://hooks.slack.com/services/456",
				}),
				Status:    backend.NotificationChannelStatusEnabled,
				CreatedAt: now,
				UpdatedAt: now,
			},

			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mock := &mockBackendSpannerClient{
				t:                                t,
				mockUpdateNotificationChannelCfg: tc.cfg,
				// Mock the Get call that happens after Update
				mockGetNotificationChannelCfg: &mockGetNotificationChannelConfig{
					expectedUserID:    userID,
					expectedChannelID: channelID,
					result:            tc.expectedGetResult,
					returnedError:     nil,
				},
			}

			b := NewBackend(mock)
			resp, err := b.UpdateNotificationChannel(context.Background(), userID, channelID, tc.req)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error. got %v, want %v", err, tc.expectedError)
			}
			if diff := cmp.Diff(tc.expected, resp, getNotificationChannelCmpOption()); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeleteNotificationChannel(t *testing.T) {
	const (
		userID    = "user123"
		channelID = "channel456"
	)

	testCases := []struct {
		name          string
		cfg           *mockDeleteNotificationChannelConfig
		expectedError error
	}{
		{
			name: "success",
			cfg: &mockDeleteNotificationChannelConfig{
				expectedChannelID: channelID,
				expectedUserID:    userID,
				returnedError:     nil,
			},
			expectedError: nil,
		},
		{
			name: "not found",
			cfg: &mockDeleteNotificationChannelConfig{
				expectedChannelID: channelID,
				expectedUserID:    userID,
				returnedError:     gcpspanner.ErrQueryReturnedNoResults,
			},
			expectedError: backendtypes.ErrEntityDoesNotExist,
		},
		{
			name: "not authorized",
			cfg: &mockDeleteNotificationChannelConfig{
				expectedChannelID: channelID,
				expectedUserID:    userID,
				returnedError:     gcpspanner.ErrMissingRequiredRole,
			},
			expectedError: backendtypes.ErrUserNotAuthorizedForAction,
		},
		{
			name: "other error",
			cfg: &mockDeleteNotificationChannelConfig{
				expectedChannelID: channelID,
				expectedUserID:    userID,
				returnedError:     errTest,
			},
			expectedError: errTest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                                t,
				mockDeleteNotificationChannelCfg: tc.cfg,
			}
			b := NewBackend(mock)
			err := b.DeleteNotificationChannel(context.Background(), userID, channelID)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error. got %v, want %v", err, tc.expectedError)
			}
		})
	}
}

func TestListNotificationChannels(t *testing.T) {
	const (
		userID = "user123"
	)
	now := time.Now()

	testCases := []struct {
		name          string
		pageSize      int
		pageToken     *string
		cfg           *mockListNotificationChannelsConfig
		expected      *backend.NotificationChannelPage
		expectedError error
	}{
		{
			name:      "success",
			pageSize:  10,
			pageToken: nil,
			cfg: &mockListNotificationChannelsConfig{
				expectedRequest: gcpspanner.ListNotificationChannelsRequest{
					UserID:    userID,
					PageSize:  10,
					PageToken: nil,
				},
				result: []gcpspanner.NotificationChannel{
					{
						ID:     "id1",
						Name:   "channel1",
						UserID: userID,
						Type:   "email",
						EmailConfig: &gcpspanner.EmailConfig{
							Address:           "b@test.com",
							IsVerified:        false,
							VerificationToken: nil,
						},
						WebhookConfig: nil,
						CreatedAt:     now,
						UpdatedAt:     now,
					},
					{
						ID:     "id2",
						Name:   "channel2",
						UserID: userID,
						Type:   "email",
						EmailConfig: &gcpspanner.EmailConfig{
							Address:           "a@test.com",
							IsVerified:        false,
							VerificationToken: nil,
						},
						WebhookConfig: nil,
						CreatedAt:     now,
						UpdatedAt:     now,
					},
					{
						ID:     "id3",
						Name:   "channel3",
						UserID: userID,
						Type:   "email",
						EmailConfig: &gcpspanner.EmailConfig{
							Address:           "c@test.com",
							IsVerified:        false,
							VerificationToken: nil,
						},
						WebhookConfig: nil,
						CreatedAt:     now.Add(-1 * time.Hour),
						UpdatedAt:     now.Add(-1 * time.Hour),
					},
				},
				nextPageToken: nil,
				returnedError: nil,
			},
			expected: &backend.NotificationChannelPage{
				Data: &[]backend.NotificationChannelResponse{
					{
						Id:   "id2",
						Name: "channel2",
						Type: backend.NotificationChannelResponseTypeEmail,
						Config: newTestNotificationChannelConfig(t, backend.EmailConfig{
							Type:    backend.EmailConfigTypeEmail,
							Address: "a@test.com",
						}),
						Status:    backend.NotificationChannelStatusEnabled,
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						Id:   "id1",
						Name: "channel1",
						Type: backend.NotificationChannelResponseTypeEmail,
						Config: newTestNotificationChannelConfig(t, backend.EmailConfig{
							Type:    backend.EmailConfigTypeEmail,
							Address: "b@test.com",
						}),
						Status:    backend.NotificationChannelStatusEnabled,
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						Id:   "id3",
						Name: "channel3",
						Type: backend.NotificationChannelResponseTypeEmail,
						Config: newTestNotificationChannelConfig(t, backend.EmailConfig{
							Type:    backend.EmailConfigTypeEmail,
							Address: "c@test.com",
						}),
						Status:    backend.NotificationChannelStatusEnabled,
						CreatedAt: now.Add(-1 * time.Hour),
						UpdatedAt: now.Add(-1 * time.Hour),
					},
				},
				Metadata: &backend.PageMetadata{NextPageToken: nil},
			},

			expectedError: nil,
		},
		{
			name:      "db error",
			pageSize:  10,
			pageToken: nil,
			cfg: &mockListNotificationChannelsConfig{
				expectedRequest: gcpspanner.ListNotificationChannelsRequest{
					UserID:    userID,
					PageSize:  10,
					PageToken: nil,
				},
				result:        nil,
				nextPageToken: nil,
				returnedError: errTest,
			},
			expected:      nil,
			expectedError: errTest,
		},
		{
			name:      "invalid cursor",
			pageSize:  10,
			pageToken: nonNilInputPageToken,
			cfg: &mockListNotificationChannelsConfig{
				expectedRequest: gcpspanner.ListNotificationChannelsRequest{
					UserID:    userID,
					PageSize:  10,
					PageToken: nonNilInputPageToken,
				},
				result:        nil,
				nextPageToken: nil,
				returnedError: gcpspanner.ErrInvalidCursorFormat,
			},
			expected:      nil,
			expectedError: backendtypes.ErrInvalidPageToken,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                               t,
				mockListNotificationChannelsCfg: tc.cfg,
			}
			b := NewBackend(mock)
			resp, err := b.ListNotificationChannels(context.Background(), userID, tc.pageSize, tc.pageToken)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error. got %v, want %v", err, tc.expectedError)
			}
			if diff := cmp.Diff(tc.expected, resp, getNotificationChannelCmpOption()); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateUserSavedSearch(t *testing.T) {
	testError := errors.New("test error")
	testCases := []struct {
		name           string
		createCfg      *mockCreateNewUserSavedSearchConfig
		getCfg         *mockGetUserSavedSearchConfig
		inputUserID    string
		savedSearch    backend.SavedSearch
		expectedOutput *backend.SavedSearchResponse
		expectedError  error
	}{
		{
			name:        "success",
			inputUserID: "user1",
			savedSearch: backend.SavedSearch{
				Name:        "test search",
				Description: new("test description"),
				Query:       "test query",
			},
			createCfg: &mockCreateNewUserSavedSearchConfig{
				expectedNewSearch: gcpspanner.CreateUserSavedSearchRequest{
					OwnerUserID: "user1",
					Query:       "test query",
					Name:        "test search",
					Description: new("test description"),
				},
				result:        new("saved-search-id"),
				returnedError: nil,
			},
			getCfg: &mockGetUserSavedSearchConfig{
				expectedAuthenticatedUserID: new("user1"),
				expectedSavedSearchID:       "saved-search-id",
				result: &gcpspanner.UserSavedSearch{
					SavedSearch: gcpspanner.SavedSearch{
						Name:        "test search",
						Description: new("test description"),
						Query:       "test query",
						Scope:       gcpspanner.UserPublicScope,
						AuthorID:    "user1",
						CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						ID:          "saved-search-id",
					},
					Role:         new(string(gcpspanner.SavedSearchOwner)),
					IsBookmarked: new(true),
				},
				returnedError: nil,
			},
			expectedOutput: &backend.SavedSearchResponse{
				Id:          "saved-search-id",
				CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				Name:        "test search",
				Description: new("test description"),
				Query:       "test query",
				Permissions: &backend.UserSavedSearchPermissions{
					Role: new(backend.SavedSearchOwner),
				},
				BookmarkStatus: &backend.UserSavedSearchBookmark{
					Status: backend.BookmarkActive,
				},
			},
			expectedError: nil,
		},
		{
			name:        "limit failure",
			inputUserID: "user1",
			savedSearch: backend.SavedSearch{
				Name:        "test search",
				Description: new("test description"),
				Query:       "test query",
			},
			createCfg: &mockCreateNewUserSavedSearchConfig{
				expectedNewSearch: gcpspanner.CreateUserSavedSearchRequest{
					OwnerUserID: "user1",
					Query:       "test query",
					Name:        "test search",
					Description: new("test description"),
				},
				result:        nil,
				returnedError: gcpspanner.ErrOwnerSavedSearchLimitExceeded,
			},
			getCfg:         nil,
			expectedOutput: nil,
			expectedError:  backendtypes.ErrUserMaxSavedSearches,
		},
		{
			name:        "general create failure",
			inputUserID: "user1",
			savedSearch: backend.SavedSearch{
				Name:        "test search",
				Description: new("test description"),
				Query:       "test query",
			},
			createCfg: &mockCreateNewUserSavedSearchConfig{
				expectedNewSearch: gcpspanner.CreateUserSavedSearchRequest{
					OwnerUserID: "user1",
					Query:       "test query",
					Name:        "test search",
					Description: new("test description"),
				},
				result:        nil,
				returnedError: testError,
			},
			getCfg:         nil,
			expectedOutput: nil,
			expectedError:  testError,
		},
		{
			name:        "general get failure",
			inputUserID: "user1",
			savedSearch: backend.SavedSearch{
				Name:        "test search",
				Description: new("test description"),
				Query:       "test query",
			},
			createCfg: &mockCreateNewUserSavedSearchConfig{
				expectedNewSearch: gcpspanner.CreateUserSavedSearchRequest{
					OwnerUserID: "user1",
					Query:       "test query",
					Name:        "test search",
					Description: new("test description"),
				},
				result:        new("saved-search-id"),
				returnedError: nil,
			},
			getCfg: &mockGetUserSavedSearchConfig{
				expectedAuthenticatedUserID: new("user1"),
				expectedSavedSearchID:       "saved-search-id",
				result:                      nil,
				returnedError:               testError,
			},
			expectedOutput: nil,
			expectedError:  testError,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                               t,
				mockCreateNewUserSavedSearchCfg: tc.createCfg,
				mockGetUserSavedSearchCfg:       tc.getCfg,
			}
			bk := NewBackend(mock)
			output, err := bk.CreateUserSavedSearch(context.Background(), tc.inputUserID, tc.savedSearch)
			if !errors.Is(err, tc.expectedError) {
				t.Error("unexpected error")
			}

			if !reflect.DeepEqual(output, tc.expectedOutput) {
				t.Error("unexpected output")
			}
		})
	}
}

func TestDeleteUserSavedSearch(t *testing.T) {
	testCases := []struct {
		name          string
		cfg           *mockDeleteUserSavedSearchConfig
		userID        string
		savedSearchID string
		expectedErr   error
	}{
		{
			name: "success",
			cfg: &mockDeleteUserSavedSearchConfig{
				expectedDeleteRequest: gcpspanner.DeleteUserSavedSearchRequest{
					RequestingUserID: "user1",
					SavedSearchID:    "saved-search-id",
				},
				returnedError: nil,
			},
			userID:        "user1",
			savedSearchID: "saved-search-id",
			expectedErr:   nil,
		},
		{
			name: "general failure",
			cfg: &mockDeleteUserSavedSearchConfig{
				expectedDeleteRequest: gcpspanner.DeleteUserSavedSearchRequest{
					RequestingUserID: "user1",
					SavedSearchID:    "saved-search-id",
				},
				returnedError: errTest,
			},
			userID:        "user1",
			savedSearchID: "saved-search-id",
			expectedErr:   errTest,
		},
		{
			name: "missing required role error",
			cfg: &mockDeleteUserSavedSearchConfig{
				expectedDeleteRequest: gcpspanner.DeleteUserSavedSearchRequest{
					RequestingUserID: "user1",
					SavedSearchID:    "saved-search-id",
				},
				returnedError: gcpspanner.ErrMissingRequiredRole,
			},
			userID:        "user1",
			savedSearchID: "saved-search-id",
			expectedErr:   backendtypes.ErrUserNotAuthorizedForAction,
		},
		{
			name: "entity does not exist error",
			cfg: &mockDeleteUserSavedSearchConfig{
				expectedDeleteRequest: gcpspanner.DeleteUserSavedSearchRequest{
					RequestingUserID: "user1",
					SavedSearchID:    "saved-search-id",
				},
				returnedError: gcpspanner.ErrQueryReturnedNoResults,
			},
			userID:        "user1",
			savedSearchID: "saved-search-id",
			expectedErr:   backendtypes.ErrEntityDoesNotExist,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                            t,
				mockDeleteUserSavedSearchCfg: tc.cfg,
			}
			bk := NewBackend(mock)
			err := bk.DeleteUserSavedSearch(context.Background(), tc.userID, tc.savedSearchID)

			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("unexpected error %s", err)
			}
		})
	}
}

func TestGetSavedSearch(t *testing.T) {
	testCases := []struct {
		name           string
		cfg            *mockGetUserSavedSearchConfig
		userID         *string
		savedSearchID  string
		expectedOutput *backend.SavedSearchResponse
		expectedError  error
	}{
		{
			name:          "success authenticated user",
			userID:        new("user1"),
			savedSearchID: "saved-search-id",
			cfg: &mockGetUserSavedSearchConfig{
				expectedAuthenticatedUserID: new("user1"),
				expectedSavedSearchID:       "saved-search-id",
				result: &gcpspanner.UserSavedSearch{
					SavedSearch: gcpspanner.SavedSearch{
						Name:        "test search",
						Description: new("test description"),
						Query:       "test query",
						Scope:       gcpspanner.UserPublicScope,
						AuthorID:    "user1",
						CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						ID:          "saved-search-id",
					},
					Role:         new(string(gcpspanner.SavedSearchOwner)),
					IsBookmarked: new(true),
				},
				returnedError: nil,
			},
			expectedOutput: &backend.SavedSearchResponse{
				Id:          "saved-search-id",
				CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				Name:        "test search",
				Query:       "test query",
				Description: new("test description"),
				BookmarkStatus: &backend.UserSavedSearchBookmark{
					Status: backend.BookmarkActive,
				},
				Permissions: &backend.UserSavedSearchPermissions{
					Role: new(backend.SavedSearchOwner),
				},
			},
			expectedError: nil,
		},
		{
			name:          "success unauthenticated user",
			userID:        nil,
			savedSearchID: "saved-search-id",
			cfg: &mockGetUserSavedSearchConfig{
				expectedAuthenticatedUserID: nil,
				expectedSavedSearchID:       "saved-search-id",
				result: &gcpspanner.UserSavedSearch{
					SavedSearch: gcpspanner.SavedSearch{
						Name:        "test search",
						Description: new("test description"),
						Query:       "test query",
						Scope:       gcpspanner.UserPublicScope,
						AuthorID:    "user1",
						CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						ID:          "saved-search-id",
					},
					Role:         nil,
					IsBookmarked: nil,
				},
				returnedError: nil,
			},
			expectedOutput: &backend.SavedSearchResponse{
				Id:             "saved-search-id",
				CreatedAt:      time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:      time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				Name:           "test search",
				Query:          "test query",
				Description:    new("test description"),
				BookmarkStatus: nil,
				Permissions:    nil,
			},
			expectedError: nil,
		},
		{
			name:          "search not found error",
			userID:        nil,
			savedSearchID: "saved-search-id",
			cfg: &mockGetUserSavedSearchConfig{
				expectedAuthenticatedUserID: nil,
				expectedSavedSearchID:       "saved-search-id",
				result:                      nil,
				returnedError:               gcpspanner.ErrQueryReturnedNoResults,
			},
			expectedOutput: nil,
			expectedError:  backendtypes.ErrEntityDoesNotExist,
		},
		{
			name:          "general error",
			userID:        nil,
			savedSearchID: "saved-search-id",
			cfg: &mockGetUserSavedSearchConfig{
				expectedAuthenticatedUserID: nil,
				expectedSavedSearchID:       "saved-search-id",
				result:                      nil,
				returnedError:               errTest,
			},
			expectedOutput: nil,
			expectedError:  errTest,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                         t,
				mockGetUserSavedSearchCfg: tc.cfg,
			}
			bk := NewBackend(mock)
			output, err := bk.GetSavedSearch(context.Background(), tc.savedSearchID, tc.userID)

			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error %s", err)
			}

			if !reflect.DeepEqual(output, tc.expectedOutput) {
				t.Errorf("unexpected output %v", output)
			}
		})
	}
}

func TestListUserSavedSearches(t *testing.T) {
	testCases := []struct {
		name          string
		userID        string
		pageSize      int
		pageToken     *string
		cfg           *mockListUserSavedSearchesConfig
		expectedPage  *backend.UserSavedSearchPage
		expectedError error
	}{
		{
			name:      "success",
			userID:    "user1",
			pageSize:  10,
			pageToken: nil,
			cfg: &mockListUserSavedSearchesConfig{
				expectedUserID:    "user1",
				expectedPageSize:  10,
				expectedPageToken: nil,
				result: &gcpspanner.UserSavedSearchesPage{
					NextPageToken: nil,
					Searches: []gcpspanner.UserSavedSearch{
						{
							SavedSearch: gcpspanner.SavedSearch{
								Name:        "z",
								Description: new("test description"),
								Query:       "test query",
								Scope:       gcpspanner.UserPublicScope,
								AuthorID:    "user1",
								CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
								UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
								ID:          "saved-search-id-2",
							},
							IsBookmarked: new(true),
							Role:         nil,
						},
					},
				},
				returnedError: nil,
			},
			expectedPage: &backend.UserSavedSearchPage{
				Metadata: nil,
				Data: new([]backend.SavedSearchResponse{
					{
						Id:          "saved-search-id-2",
						CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						Name:        "z",
						Description: new("test description"),
						Query:       "test query",
						Permissions: nil,
						BookmarkStatus: &backend.UserSavedSearchBookmark{
							Status: backend.BookmarkActive,
						},
					},
				}),
			},
			expectedError: nil,
		},
		{
			name:      "success w/ page token",
			userID:    "user1",
			pageSize:  10,
			pageToken: new("inputToken"),
			cfg: &mockListUserSavedSearchesConfig{
				expectedUserID:    "user1",
				expectedPageSize:  10,
				expectedPageToken: new("inputToken"),
				result: &gcpspanner.UserSavedSearchesPage{
					NextPageToken: new("nextToken"),
					Searches: []gcpspanner.UserSavedSearch{
						{
							SavedSearch: gcpspanner.SavedSearch{
								Name:        "test search",
								Description: new("test description"),
								Query:       "test query",
								Scope:       gcpspanner.UserPublicScope,
								AuthorID:    "user1",
								CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
								UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
								ID:          "saved-search-id",
							},
							Role:         new(string(gcpspanner.SavedSearchOwner)),
							IsBookmarked: new(true),
						},
						{
							SavedSearch: gcpspanner.SavedSearch{
								Name:        "z",
								Description: new("test description"),
								Query:       "test query",
								Scope:       gcpspanner.UserPublicScope,
								AuthorID:    "user1",
								CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
								UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
								ID:          "saved-search-id-2",
							},
							IsBookmarked: new(true),
							Role:         nil,
						},
					},
				},
				returnedError: nil,
			},
			expectedPage: &backend.UserSavedSearchPage{
				Metadata: &backend.PageMetadata{
					NextPageToken: new("nextToken"),
				},
				Data: new([]backend.SavedSearchResponse{
					{
						Id:          "saved-search-id",
						CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						Name:        "test search",
						Description: new("test description"),
						Query:       "test query",
						Permissions: &backend.UserSavedSearchPermissions{
							Role: new(backend.SavedSearchOwner),
						},
						BookmarkStatus: &backend.UserSavedSearchBookmark{
							Status: backend.BookmarkActive,
						},
					},
					{
						Id:          "saved-search-id-2",
						CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						Name:        "z",
						Description: new("test description"),
						Query:       "test query",
						Permissions: nil,
						BookmarkStatus: &backend.UserSavedSearchBookmark{
							Status: backend.BookmarkActive,
						},
					},
				}),
			},
			expectedError: nil,
		},
		{
			name:      "general error",
			userID:    "user1",
			pageSize:  10,
			pageToken: nil,
			cfg: &mockListUserSavedSearchesConfig{
				expectedUserID:    "user1",
				expectedPageSize:  10,
				expectedPageToken: nil,
				result:            nil,
				returnedError:     errTest,
			},
			expectedPage:  nil,
			expectedError: errTest,
		},
		{
			name:      "invalid cursor",
			userID:    "user1",
			pageSize:  10,
			pageToken: nil,
			cfg: &mockListUserSavedSearchesConfig{
				expectedUserID:    "user1",
				expectedPageSize:  10,
				expectedPageToken: nil,
				result:            nil,
				returnedError:     gcpspanner.ErrInvalidCursorFormat,
			},
			expectedPage:  nil,
			expectedError: backendtypes.ErrInvalidPageToken,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                            t,
				mockListUserSavedSearchesCfg: tc.cfg,
			}
			backend := NewBackend(mock)
			page, err := backend.ListUserSavedSearches(context.Background(), tc.userID, tc.pageSize, tc.pageToken)
			if !errors.Is(err, tc.expectedError) {
				t.Error("unexpected error")
			}

			if !reflect.DeepEqual(page, tc.expectedPage) {
				t.Error("unexpected page")
			}
		})
	}
}

func TestUpdateUserSavedSearch(t *testing.T) {
	testSavedSearchID := "test-id"
	testUserID := "test-user"

	testCases := []struct {
		name          string
		updateRequest *backend.SavedSearchUpdateRequest
		mockUpdateCfg *mockUpdateUserSavedSearchConfig
		mockGetCfg    *mockGetUserSavedSearchConfig
		expectedResp  *backend.SavedSearchResponse
		expectedError error
	}{
		{
			name: "success",
			updateRequest: &backend.SavedSearchUpdateRequest{
				Name:        new("test search name"),
				Description: new("test desc"),
				Query:       new("test query"),
				UpdateMask: []backend.SavedSearchUpdateRequestUpdateMask{
					backend.SavedSearchUpdateRequestMaskName,
					backend.SavedSearchUpdateRequestMaskDescription,
					backend.SavedSearchUpdateRequestMaskQuery,
				},
			},
			mockUpdateCfg: &mockUpdateUserSavedSearchConfig{
				expectedRequest: gcpspanner.UpdateSavedSearchRequest{
					ID:       "test-id",
					AuthorID: "test-user",
					Name: gcpspanner.OptionallySet[string]{
						IsSet: true,
						Value: "test search name",
					},
					Description: gcpspanner.OptionallySet[*string]{
						IsSet: true,
						Value: new("test desc"),
					},
					Query: gcpspanner.OptionallySet[string]{
						IsSet: true,
						Value: "test query",
					},
				},
				returnedError: nil,
			},
			mockGetCfg: &mockGetUserSavedSearchConfig{
				expectedAuthenticatedUserID: new("test-user"),
				expectedSavedSearchID:       "test-id",
				result: &gcpspanner.UserSavedSearch{
					SavedSearch: gcpspanner.SavedSearch{
						ID:          "test-id",
						Name:        "test search name",
						Description: new("test desc"),
						Query:       "test query",
						Scope:       gcpspanner.UserPublicScope,
						AuthorID:    "test-user",
						CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
					},
					Role:         new(string(gcpspanner.SavedSearchOwner)),
					IsBookmarked: new(true),
				},
				returnedError: nil,
			},
			expectedResp: &backend.SavedSearchResponse{
				Id:          "test-id",
				CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				Name:        "test search name",
				Description: new("test desc"),
				Query:       "test query",
				Permissions: &backend.UserSavedSearchPermissions{
					Role: new(backend.SavedSearchOwner),
				},
				BookmarkStatus: &backend.UserSavedSearchBookmark{
					Status: backend.BookmarkActive,
				},
			},
			expectedError: nil,
		},
		{
			name: "get user saved search return no results",
			updateRequest: &backend.SavedSearchUpdateRequest{
				Name:        new("test search name"),
				Description: new("test desc"),
				Query:       new("test query"),
				UpdateMask: []backend.SavedSearchUpdateRequestUpdateMask{
					backend.SavedSearchUpdateRequestMaskName,
					backend.SavedSearchUpdateRequestMaskDescription,
					backend.SavedSearchUpdateRequestMaskQuery,
				},
			},
			mockUpdateCfg: &mockUpdateUserSavedSearchConfig{
				expectedRequest: gcpspanner.UpdateSavedSearchRequest{
					ID:       "test-id",
					AuthorID: "test-user",
					Name: gcpspanner.OptionallySet[string]{
						IsSet: true,
						Value: "test search name",
					},
					Description: gcpspanner.OptionallySet[*string]{
						IsSet: true,
						Value: new("test desc"),
					},
					Query: gcpspanner.OptionallySet[string]{
						IsSet: true,
						Value: "test query",
					},
				},
				returnedError: nil,
			},
			mockGetCfg: &mockGetUserSavedSearchConfig{
				expectedAuthenticatedUserID: new("test-user"),
				expectedSavedSearchID:       "test-id",
				result:                      nil,
				returnedError:               gcpspanner.ErrQueryReturnedNoResults,
			},
			expectedResp:  nil,
			expectedError: backendtypes.ErrEntityDoesNotExist,
		},
		{
			name: "get user saved search returns other error",
			updateRequest: &backend.SavedSearchUpdateRequest{
				Name:        new("test search name"),
				Description: new("test desc"),
				Query:       new("test query"),
				UpdateMask: []backend.SavedSearchUpdateRequestUpdateMask{
					backend.SavedSearchUpdateRequestMaskName,
					backend.SavedSearchUpdateRequestMaskDescription,
					backend.SavedSearchUpdateRequestMaskQuery,
				},
			},
			mockUpdateCfg: &mockUpdateUserSavedSearchConfig{
				expectedRequest: gcpspanner.UpdateSavedSearchRequest{
					ID:       "test-id",
					AuthorID: "test-user",
					Name: gcpspanner.OptionallySet[string]{
						IsSet: true,
						Value: "test search name",
					},
					Description: gcpspanner.OptionallySet[*string]{
						IsSet: true,
						Value: new("test desc"),
					},
					Query: gcpspanner.OptionallySet[string]{
						IsSet: true,
						Value: "test query",
					},
				},
				returnedError: nil,
			},
			mockGetCfg: &mockGetUserSavedSearchConfig{
				expectedAuthenticatedUserID: new("test-user"),
				expectedSavedSearchID:       "test-id",
				result:                      nil,
				returnedError:               errTest,
			},
			expectedResp:  nil,
			expectedError: errTest,
		},
		{
			name: "update user saved search return no results",
			updateRequest: &backend.SavedSearchUpdateRequest{
				Name:        new("test search name"),
				Description: new("test desc"),
				Query:       new("test query"),
				UpdateMask: []backend.SavedSearchUpdateRequestUpdateMask{
					backend.SavedSearchUpdateRequestMaskName,
					backend.SavedSearchUpdateRequestMaskDescription,
					backend.SavedSearchUpdateRequestMaskQuery,
				},
			},
			mockUpdateCfg: &mockUpdateUserSavedSearchConfig{
				expectedRequest: gcpspanner.UpdateSavedSearchRequest{
					ID:       "test-id",
					AuthorID: "test-user",
					Name: gcpspanner.OptionallySet[string]{
						IsSet: true,
						Value: "test search name",
					},
					Description: gcpspanner.OptionallySet[*string]{
						IsSet: true,
						Value: new("test desc"),
					},
					Query: gcpspanner.OptionallySet[string]{
						IsSet: true,
						Value: "test query",
					},
				},
				returnedError: gcpspanner.ErrQueryReturnedNoResults,
			},
			mockGetCfg:    nil,
			expectedResp:  nil,
			expectedError: backendtypes.ErrEntityDoesNotExist,
		},
		{
			name: "update user saved search return no required role error",
			updateRequest: &backend.SavedSearchUpdateRequest{
				Name:        new("test search name"),
				Description: new("test desc"),
				Query:       new("test query"),
				UpdateMask: []backend.SavedSearchUpdateRequestUpdateMask{
					backend.SavedSearchUpdateRequestMaskName,
					backend.SavedSearchUpdateRequestMaskDescription,
					backend.SavedSearchUpdateRequestMaskQuery,
				},
			},
			mockUpdateCfg: &mockUpdateUserSavedSearchConfig{
				expectedRequest: gcpspanner.UpdateSavedSearchRequest{
					ID:       "test-id",
					AuthorID: "test-user",
					Name: gcpspanner.OptionallySet[string]{
						IsSet: true,
						Value: "test search name",
					},
					Description: gcpspanner.OptionallySet[*string]{
						IsSet: true,
						Value: new("test desc"),
					},
					Query: gcpspanner.OptionallySet[string]{
						IsSet: true,
						Value: "test query",
					},
				},
				returnedError: gcpspanner.ErrMissingRequiredRole,
			},
			mockGetCfg:    nil,
			expectedResp:  nil,
			expectedError: backendtypes.ErrUserNotAuthorizedForAction,
		},
		{
			name: "update user saved search return other error",
			updateRequest: &backend.SavedSearchUpdateRequest{
				Name:        new("test search name"),
				Description: new("test desc"),
				Query:       new("test query"),
				UpdateMask: []backend.SavedSearchUpdateRequestUpdateMask{
					backend.SavedSearchUpdateRequestMaskName,
					backend.SavedSearchUpdateRequestMaskDescription,
					backend.SavedSearchUpdateRequestMaskQuery,
				},
			},
			mockUpdateCfg: &mockUpdateUserSavedSearchConfig{
				expectedRequest: gcpspanner.UpdateSavedSearchRequest{
					ID:       "test-id",
					AuthorID: "test-user",
					Name: gcpspanner.OptionallySet[string]{
						IsSet: true,
						Value: "test search name",
					},
					Description: gcpspanner.OptionallySet[*string]{
						IsSet: true,
						Value: new("test desc"),
					},
					Query: gcpspanner.OptionallySet[string]{
						IsSet: true,
						Value: "test query",
					},
				},
				returnedError: errTest,
			},
			mockGetCfg:    nil,
			expectedResp:  nil,
			expectedError: errTest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                            t,
				mockGetUserSavedSearchCfg:    tc.mockGetCfg,
				mockUpdateUserSavedSearchCfg: tc.mockUpdateCfg,
			}

			backend := NewBackend(mock)
			resp, err := backend.UpdateUserSavedSearch(
				context.Background(),
				testSavedSearchID,
				testUserID,
				tc.updateRequest,
			)
			if !errors.Is(err, tc.expectedError) {
				t.Error("unexpected error")
			}

			if !reflect.DeepEqual(resp, tc.expectedResp) {
				t.Error("unexpected response")
			}
		})
	}
}

// nolint: dupl // WONTFIX
func TestAddUserSavedSearchBookmark(t *testing.T) {
	userID := "test-add-user"
	savedSearchID := "test-add-id"
	bookmark := gcpspanner.UserSavedSearchBookmark{
		UserID:        "test-add-user",
		SavedSearchID: "test-add-id",
	}
	testCases := []struct {
		name        string
		cfg         *mockAddUserSearchBookmarkConfig
		expectedErr error
	}{
		{
			name: "success",
			cfg: &mockAddUserSearchBookmarkConfig{
				expectedRequest: bookmark,
				returnedError:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "limit exceeded error",
			cfg: &mockAddUserSearchBookmarkConfig{
				expectedRequest: bookmark,
				returnedError:   gcpspanner.ErrUserSearchBookmarkLimitExceeded,
			},
			expectedErr: backendtypes.ErrUserMaxBookmarks,
		},
		{
			name: "not found error",
			cfg: &mockAddUserSearchBookmarkConfig{
				expectedRequest: bookmark,
				returnedError:   gcpspanner.ErrQueryReturnedNoResults,
			},
			expectedErr: backendtypes.ErrEntityDoesNotExist,
		},
		{
			name: "unknown error",
			cfg: &mockAddUserSearchBookmarkConfig{
				expectedRequest: bookmark,
				returnedError:   errTest,
			},
			expectedErr: errTest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                            t,
				mockAddUserSearchBookmarkCfg: tc.cfg,
			}
			bk := NewBackend(mock)
			err := bk.PutUserSavedSearchBookmark(context.Background(), userID, savedSearchID)

			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("unexpected error %s", err)
			}
		})
	}
}

// nolint: dupl // WONTFIX
func TestRemoveUserSavedSearchBookmark(t *testing.T) {
	userID := "test-remove-user"
	savedSearchID := "test-remove-id"
	bookmark := gcpspanner.UserSavedSearchBookmark{
		UserID:        "test-remove-user",
		SavedSearchID: "test-remove-id",
	}
	testCases := []struct {
		name        string
		cfg         *mockDeleteUserSearchBookmarkConfig
		expectedErr error
	}{
		{
			name: "success",
			cfg: &mockDeleteUserSearchBookmarkConfig{
				expectedRequest: bookmark,
				returnedError:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "owner cannot delete bookmark error",
			cfg: &mockDeleteUserSearchBookmarkConfig{
				expectedRequest: bookmark,
				returnedError:   gcpspanner.ErrOwnerCannotDeleteBookmark,
			},
			expectedErr: backendtypes.ErrUserNotAuthorizedForAction,
		},
		{
			name: "not found error",
			cfg: &mockDeleteUserSearchBookmarkConfig{
				expectedRequest: bookmark,
				returnedError:   gcpspanner.ErrQueryReturnedNoResults,
			},
			expectedErr: backendtypes.ErrEntityDoesNotExist,
		},
		{
			name: "unknown error",
			cfg: &mockDeleteUserSearchBookmarkConfig{
				expectedRequest: bookmark,
				returnedError:   errTest,
			},
			expectedErr: errTest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                               t,
				mockDeleteUserSearchBookmarkCfg: tc.cfg,
			}
			bk := NewBackend(mock)
			err := bk.RemoveUserSavedSearchBookmark(context.Background(), userID, savedSearchID)

			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("unexpected error %s", err)
			}
		})
	}
}

func TestBuildUpdateSavedSearchRequestForGCP(t *testing.T) {
	testSavedSearchID := "test-id"
	testUserID := "test-user"
	testCases := []struct {
		name string
		req  *backend.SavedSearchUpdateRequest
		want gcpspanner.UpdateSavedSearchRequest
	}{
		{
			name: "empty mask gives no update",
			req: &backend.SavedSearchUpdateRequest{
				Name:        new("test name"),
				Description: new("test description"),
				Query:       new("test query"),
				UpdateMask:  []backend.SavedSearchUpdateRequestUpdateMask{},
			},
			want: gcpspanner.UpdateSavedSearchRequest{
				ID:       "test-id",
				AuthorID: "test-user",
				Name: gcpspanner.OptionallySet[string]{
					IsSet: false,
					Value: "",
				},
				Description: gcpspanner.OptionallySet[*string]{
					IsSet: false,
					Value: nil,
				},
				Query: gcpspanner.OptionallySet[string]{
					IsSet: false,
					Value: "",
				},
			},
		},
		{
			name: "update mask contains all fields updates all fields",
			req: &backend.SavedSearchUpdateRequest{
				Name:        new("test name"),
				Description: new("test description"),
				Query:       new("test query"),
				UpdateMask: []backend.SavedSearchUpdateRequestUpdateMask{
					backend.SavedSearchUpdateRequestMaskName,
					backend.SavedSearchUpdateRequestMaskDescription,
					backend.SavedSearchUpdateRequestMaskQuery,
				},
			},
			want: gcpspanner.UpdateSavedSearchRequest{
				ID:       "test-id",
				AuthorID: "test-user",
				Name: gcpspanner.OptionallySet[string]{
					IsSet: true,
					Value: "test name",
				},
				Description: gcpspanner.OptionallySet[*string]{
					IsSet: true,
					Value: new("test description"),
				},
				Query: gcpspanner.OptionallySet[string]{
					IsSet: true,
					Value: "test query",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := buildUpdateSavedSearchRequestForGCP(testSavedSearchID, testUserID, tc.req)
			if !reflect.DeepEqual(req, tc.want) {
				t.Errorf("unexpected request %v", req)
			}
		})
	}
}

func TestGetFeatureSearchSortOrder(t *testing.T) {
	sortOrderTests := []struct {
		input *backend.ListFeaturesParamsSort
		want  gcpspanner.Sortable
	}{
		{input: nil, want: gcpspanner.NewBaselineStatusSort(false)},
		{
			input: new(backend.NameAsc),
			want:  gcpspanner.NewFeatureNameSort(true),
		},
		{
			input: new(backend.NameDesc),
			want:  gcpspanner.NewFeatureNameSort(false),
		},
		{
			input: new(backend.BaselineStatusAsc),
			want:  gcpspanner.NewBaselineStatusSort(true),
		},
		{
			input: new(backend.BaselineStatusDesc),
			want:  gcpspanner.NewBaselineStatusSort(false),
		},
		{
			input: new(backend.ExperimentalChromeAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "chrome", false),
		},
		{
			input: new(backend.ExperimentalChromeDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "chrome", false),
		},
		{
			input: new(backend.ExperimentalEdgeAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "edge", false),
		},
		{
			input: new(backend.ExperimentalEdgeDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "edge", false),
		},
		{
			input: new(backend.ExperimentalFirefoxAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "firefox", false),
		},
		{
			input: new(backend.ExperimentalFirefoxDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "firefox", false),
		},
		{
			input: new(backend.ExperimentalSafariAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "safari", false),
		},
		{
			input: new(backend.ExperimentalSafariDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "safari", false),
		},
		{
			input: new(backend.StableChromeAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "chrome", true),
		},
		{
			input: new(backend.StableChromeDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "chrome", true),
		},
		{
			input: new(backend.StableEdgeAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "edge", true),
		},
		{
			input: new(backend.StableEdgeDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "edge", true),
		},
		{
			input: new(backend.StableFirefoxAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "firefox", true),
		},
		{
			input: new(backend.StableFirefoxDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "firefox", true),
		},
		{
			input: new(backend.StableSafariAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "safari", true),
		},
		{
			input: new(backend.StableSafariDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "safari", true),
		},
		{
			input: new(backend.AvailabilityChromeAsc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(true, "chrome"),
		},
		{
			input: new(backend.AvailabilityChromeDesc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(false, "chrome"),
		},
		{
			input: new(backend.AvailabilityEdgeAsc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(true, "edge"),
		},
		{
			input: new(backend.AvailabilityEdgeDesc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(false, "edge"),
		},
		{
			input: new(backend.AvailabilityFirefoxAsc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(true, "firefox"),
		},
		{
			input: new(backend.AvailabilityFirefoxDesc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(false, "firefox"),
		},
		{
			input: new(backend.AvailabilitySafariAsc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(true, "safari"),
		},
		{
			input: new(backend.AvailabilitySafariDesc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(false, "safari"),
		},
		{
			input: new(backend.DeveloperSignalUpvotesAsc),
			want:  gcpspanner.NewDeveloperSignalUpvotesSort(true),
		},
		{
			input: new(backend.DeveloperSignalUpvotesDesc),
			want:  gcpspanner.NewDeveloperSignalUpvotesSort(false),
		},
	}

	for _, tt := range sortOrderTests {
		got := getFeatureSearchSortOrder(tt.input)

		// Compare 'got' and 'tt.want' (Consider using a deep equality check library)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("got: %v, want: %v", got, tt.want)
		}
	}
}

func TestConvertFeatureResult(t *testing.T) {
	testCases := []struct {
		name            string
		featureResult   *gcpspanner.FeatureResult
		expectedFeature *backend.Feature
	}{
		{
			name: "nil PassRate edge case",
			featureResult: &gcpspanner.FeatureResult{
				Name:                       "feature 1",
				FeatureKey:                 "feature1",
				Status:                     new("low"),
				LowDate:                    new(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)),
				HighDate:                   nil,
				SystemManagedSavedSearchID: nil,
				StableMetrics: []*gcpspanner.FeatureResultMetric{
					{
						BrowserName: "browser3",
						PassRate:    nil,
					},
				},
				ExperimentalMetrics: []*gcpspanner.FeatureResultMetric{
					{
						BrowserName: "browser3",
						PassRate:    nil,
					},
				},
				ImplementationStatuses: nil,
				SpecLinks:              nil,
				ChromiumUsage:          big.NewRat(8, 100),
				DeveloperSignalUpvotes: nil,
				DeveloperSignalLink:    nil,
				AccordingTo:            nil,
				Alternatives:           nil,
				VendorPositions:        spanner.NullJSON{Valid: false, Value: nil},
			},

			expectedFeature: &backend.Feature{
				Baseline: &backend.BaselineInfo{
					Status: new(backend.Newly),
					LowDate: new(
						openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
					),
					HighDate: nil,
				},
				FeatureId: "feature1",
				Name:      "feature 1",
				Spec:      nil,
				Usage: &backend.BrowserUsage{
					Chrome: &backend.ChromeUsageInfo{
						Daily: new(0.08),
					},
				},
				Wpt:                        nil,
				BrowserImplementations:     nil,
				DeveloperSignals:           nil,
				Discouraged:                nil,
				VendorPositions:            nil,
				SystemManagedSavedSearchId: nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := Backend{client: nil}
			feature := b.convertFeatureResult(tc.featureResult)
			if !CompareFeatures(*tc.expectedFeature, *feature) {
				t.Errorf("unexpected feature %v", *feature)
			}
		})
	}
}

func TestCreateSavedSearchSubscription(t *testing.T) {
	const (
		userID        = "user123"
		channelID     = "channel-id"
		savedSearchID = "saved-search-id"
		subID         = "sub-id"
	)
	now := time.Now()
	testCases := []struct {
		name          string
		input         backend.Subscription
		createCfg     *mockCreateSavedSearchSubscriptionConfig
		getCfg        *mockGetSavedSearchSubscriptionConfig
		expected      *backend.SubscriptionResponse
		expectedError error
	}{
		{
			name: "success",
			input: backend.Subscription{
				ChannelId:     channelID,
				SavedSearchId: savedSearchID,
				Triggers: []backend.SubscriptionTriggerWritable{
					backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete},
				Frequency: backend.SubscriptionFrequencyImmediate,
			},
			createCfg: &mockCreateSavedSearchSubscriptionConfig{
				expectedRequest: gcpspanner.CreateSavedSearchSubscriptionRequest{
					UserID:        userID,
					ChannelID:     channelID,
					SavedSearchID: savedSearchID,
					Triggers: []gcpspanner.SubscriptionTrigger{
						gcpspanner.SubscriptionTriggerBrowserImplementationAnyComplete},
					Frequency: gcpspanner.SavedSearchSnapshotTypeImmediate,
				},
				result:        new(subID),
				returnedError: nil,
			},
			getCfg: &mockGetSavedSearchSubscriptionConfig{
				expectedSubscriptionID: subID,
				expectedUserID:         userID,
				result: &gcpspanner.SavedSearchSubscriptionView{
					SavedSearchSubscription: gcpspanner.SavedSearchSubscription{
						ID:            subID,
						ChannelID:     channelID,
						SavedSearchID: savedSearchID,
						Triggers: []gcpspanner.SubscriptionTrigger{
							gcpspanner.SubscriptionTriggerBrowserImplementationAnyComplete,
						},
						Frequency: gcpspanner.SavedSearchSnapshotTypeImmediate,
						CreatedAt: now,
						UpdatedAt: now,
					},
					SavedSearchName: "Feature name",
				},
				returnedError: nil,
			},
			expected: &backend.SubscriptionResponse{
				Id:           subID,
				ChannelId:    channelID,
				Subscribable: backend.SavedSearchInfo{Id: savedSearchID, Name: "Feature name"},
				Triggers: []backend.SubscriptionTriggerResponseItem{
					{
						Value: backendtypes.AttemptToStoreSubscriptionTrigger(
							backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete),
						RawValue: nil,
					},
				},
				Frequency: backend.SubscriptionFrequencyImmediate,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectedError: nil,
		},
		{
			name: "create unauthorized",
			input: backend.Subscription{
				ChannelId:     channelID,
				SavedSearchId: savedSearchID,
				Triggers: []backend.SubscriptionTriggerWritable{
					backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete},
				Frequency: backend.SubscriptionFrequencyImmediate,
			},
			createCfg: &mockCreateSavedSearchSubscriptionConfig{
				expectedRequest: gcpspanner.CreateSavedSearchSubscriptionRequest{
					UserID:        userID,
					ChannelID:     channelID,
					SavedSearchID: savedSearchID,
					Triggers: []gcpspanner.SubscriptionTrigger{
						gcpspanner.SubscriptionTriggerBrowserImplementationAnyComplete,
					},
					Frequency: gcpspanner.SavedSearchSnapshotTypeImmediate,
				},
				result:        nil,
				returnedError: gcpspanner.ErrMissingRequiredRole,
			},
			getCfg:        nil,
			expected:      nil,
			expectedError: backendtypes.ErrUserNotAuthorizedForAction,
		},
		{
			name: "create error",
			input: backend.Subscription{
				ChannelId:     channelID,
				SavedSearchId: savedSearchID,
				Triggers: []backend.SubscriptionTriggerWritable{
					backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete},
				Frequency: backend.SubscriptionFrequencyImmediate,
			},
			createCfg: &mockCreateSavedSearchSubscriptionConfig{
				expectedRequest: gcpspanner.CreateSavedSearchSubscriptionRequest{
					UserID:        userID,
					ChannelID:     channelID,
					SavedSearchID: savedSearchID,
					Triggers: []gcpspanner.SubscriptionTrigger{
						gcpspanner.SubscriptionTriggerBrowserImplementationAnyComplete,
					},
					Frequency: gcpspanner.SavedSearchSnapshotTypeImmediate,
				},
				result:        nil,
				returnedError: errTest,
			},
			getCfg:        nil,
			expected:      nil,
			expectedError: errTest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                                    t,
				mockCreateSavedSearchSubscriptionCfg: tc.createCfg,
				mockGetSavedSearchSubscriptionCfg:    tc.getCfg,
			}
			b := NewBackend(mock)
			resp, err := b.CreateSavedSearchSubscription(context.Background(), userID, tc.input)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error. got %v, want %v", err, tc.expectedError)
			}
			if diff := cmp.Diff(tc.expected, resp, getTriggerCmpOption()); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestListSavedSearchSubscriptions(t *testing.T) {
	const (
		userID = "user123"
	)
	now := time.Now()

	testCases := []struct {
		name          string
		pageSize      int
		pageToken     *string
		cfg           *mockListSavedSearchSubscriptionsConfig
		expected      *backend.SubscriptionPage
		expectedError error
	}{
		{
			name:      "success",
			pageSize:  10,
			pageToken: nil,
			cfg: &mockListSavedSearchSubscriptionsConfig{
				expectedRequest: gcpspanner.ListSavedSearchSubscriptionsRequest{
					UserID:    userID,
					PageSize:  10,
					PageToken: nil,
				},
				result: []gcpspanner.SavedSearchSubscriptionView{
					{
						SavedSearchSubscription: gcpspanner.SavedSearchSubscription{
							ID:            "sub1",
							ChannelID:     "chan1",
							SavedSearchID: "search1",
							Triggers: []gcpspanner.SubscriptionTrigger{
								gcpspanner.SubscriptionTriggerBrowserImplementationAnyComplete,
							},
							Frequency: gcpspanner.SavedSearchSnapshotTypeImmediate,
							CreatedAt: now,
							UpdatedAt: now,
						},
						SavedSearchName: "Feature name",
					},
				},
				nextPageToken: nonNilNextPageToken,
				returnedError: nil,
			},
			expected: &backend.SubscriptionPage{
				Data: &[]backend.SubscriptionResponse{
					{
						Id:           "sub1",
						ChannelId:    "chan1",
						Subscribable: backend.SavedSearchInfo{Id: "search1", Name: "Feature name"},
						Triggers: []backend.SubscriptionTriggerResponseItem{
							{
								Value: backendtypes.AttemptToStoreSubscriptionTrigger(
									backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete),
								RawValue: nil,
							},
						},
						Frequency: backend.SubscriptionFrequencyImmediate,
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
				Metadata: &backend.PageMetadata{
					NextPageToken: nonNilNextPageToken,
				},
			},
			expectedError: nil,
		},
		{
			name:      "db error",
			pageSize:  10,
			pageToken: nil,
			cfg: &mockListSavedSearchSubscriptionsConfig{
				expectedRequest: gcpspanner.ListSavedSearchSubscriptionsRequest{
					UserID:    userID,
					PageSize:  10,
					PageToken: nil,
				},
				nextPageToken: nil,
				result:        nil,
				returnedError: errTest,
			},
			expected:      nil,
			expectedError: errTest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                                   t,
				mockListSavedSearchSubscriptionsCfg: tc.cfg,
			}
			b := NewBackend(mock)
			resp, err := b.ListSavedSearchSubscriptions(context.Background(), userID, tc.pageSize, tc.pageToken)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error. got %v, want %v", err, tc.expectedError)
			}
			if diff := cmp.Diff(tc.expected, resp, getTriggerCmpOption()); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetSavedSearchSubscription(t *testing.T) {
	const (
		userID = "user123"
		subID  = "sub456"
	)
	now := time.Now()

	testCases := []struct {
		name          string
		cfg           *mockGetSavedSearchSubscriptionConfig
		expected      *backend.SubscriptionResponse
		expectedError error
	}{
		{
			name: "success",
			cfg: &mockGetSavedSearchSubscriptionConfig{
				expectedSubscriptionID: subID,
				expectedUserID:         userID,
				result: &gcpspanner.SavedSearchSubscriptionView{
					SavedSearchSubscription: gcpspanner.SavedSearchSubscription{
						ID:            subID,
						ChannelID:     "chan1",
						SavedSearchID: "search1",
						Triggers: []gcpspanner.SubscriptionTrigger{
							gcpspanner.SubscriptionTriggerBrowserImplementationAnyComplete,
						},
						Frequency: gcpspanner.SavedSearchSnapshotTypeImmediate,
						CreatedAt: now,
						UpdatedAt: now,
					},
					SavedSearchName: "Feature name",
				},
				returnedError: nil,
			},
			expected: &backend.SubscriptionResponse{
				Id:           subID,
				ChannelId:    "chan1",
				Subscribable: backend.SavedSearchInfo{Id: "search1", Name: "Feature name"},
				Triggers: []backend.SubscriptionTriggerResponseItem{
					{
						Value: backendtypes.AttemptToStoreSubscriptionTrigger(
							backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete),
						RawValue: nil,
					},
				},
				Frequency: backend.SubscriptionFrequencyImmediate,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectedError: nil,
		},
		{
			name: "not found",
			cfg: &mockGetSavedSearchSubscriptionConfig{
				expectedSubscriptionID: subID,
				expectedUserID:         userID,
				result:                 nil,
				returnedError:          gcpspanner.ErrQueryReturnedNoResults,
			},
			expected:      nil,
			expectedError: backendtypes.ErrEntityDoesNotExist,
		},
		{
			name: "not authorized",
			cfg: &mockGetSavedSearchSubscriptionConfig{
				expectedSubscriptionID: subID,
				expectedUserID:         userID,
				result:                 nil,
				returnedError:          gcpspanner.ErrMissingRequiredRole,
			},
			expected:      nil,
			expectedError: backendtypes.ErrUserNotAuthorizedForAction,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                                 t,
				mockGetSavedSearchSubscriptionCfg: tc.cfg,
			}
			b := NewBackend(mock)
			resp, err := b.GetSavedSearchSubscription(context.Background(), userID, subID)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error. got %v, want %v", err, tc.expectedError)
			}
			if diff := cmp.Diff(tc.expected, resp, getTriggerCmpOption()); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUpdateSavedSearchSubscription(t *testing.T) {
	const (
		userID = "user123"
		subID  = "sub456"
	)
	now := time.Now()
	updatedTriggers := []backend.SubscriptionTriggerWritable{
		backend.SubscriptionTriggerFeatureBaselineToNewly,
		backend.SubscriptionTriggerFeatureBaselineRegressionToLimited,
	}
	updatedFrequency := backend.SubscriptionFrequencyImmediate
	updatedSpannerFrequency := gcpspanner.SavedSearchSnapshotTypeImmediate

	testCases := []struct {
		name          string
		input         backend.UpdateSubscriptionRequest
		updateCfg     *mockUpdateSavedSearchSubscriptionConfig
		getCfg        *mockGetSavedSearchSubscriptionConfig
		expected      *backend.SubscriptionResponse
		expectedError error
	}{
		{
			name: "success update triggers",
			input: backend.UpdateSubscriptionRequest{
				UpdateMask: []backend.UpdateSubscriptionRequestUpdateMask{
					backend.UpdateSubscriptionRequestMaskTriggers},
				Triggers:  &updatedTriggers,
				Frequency: nil,
			},
			updateCfg: &mockUpdateSavedSearchSubscriptionConfig{
				expectedRequest: gcpspanner.UpdateSavedSearchSubscriptionRequest{
					ID:     subID,
					UserID: userID,
					Triggers: gcpspanner.OptionallySet[[]gcpspanner.SubscriptionTrigger]{
						Value: []gcpspanner.SubscriptionTrigger{
							gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToNewly,
							gcpspanner.SubscriptionTriggerFeatureBaselineRegressionToLimited,
						}, IsSet: true,
					},
					Frequency: gcpspanner.OptionallySet[gcpspanner.SavedSearchSnapshotType]{IsSet: false, Value: ""},
				},
				returnedError: nil,
			},
			getCfg: &mockGetSavedSearchSubscriptionConfig{
				expectedSubscriptionID: subID,
				expectedUserID:         userID,
				result: &gcpspanner.SavedSearchSubscriptionView{
					SavedSearchSubscription: gcpspanner.SavedSearchSubscription{
						ID: subID,
						Triggers: []gcpspanner.SubscriptionTrigger{
							gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToNewly},
						ChannelID:     "channel",
						SavedSearchID: "savedsearch",
						Frequency:     gcpspanner.SavedSearchSnapshotTypeImmediate,
						CreatedAt:     now,
						UpdatedAt:     now,
					},
					SavedSearchName: "Feature name",
				},
				returnedError: nil,
			},
			expected: &backend.SubscriptionResponse{
				Id: subID,
				Triggers: []backend.SubscriptionTriggerResponseItem{
					{
						Value: backendtypes.AttemptToStoreSubscriptionTrigger(
							backend.SubscriptionTriggerFeatureBaselineToNewly),
						RawValue: nil,
					},
				},
				ChannelId: "channel",
				Subscribable: backend.SavedSearchInfo{
					Id:   "savedsearch",
					Name: "Feature name",
				},
				Frequency: backend.SubscriptionFrequencyImmediate,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectedError: nil,
		},
		{
			name: "success update frequency",
			input: backend.UpdateSubscriptionRequest{
				UpdateMask: []backend.UpdateSubscriptionRequestUpdateMask{
					backend.UpdateSubscriptionRequestMaskFrequency},
				Frequency: &updatedFrequency,
				Triggers:  nil,
			},
			updateCfg: &mockUpdateSavedSearchSubscriptionConfig{
				expectedRequest: gcpspanner.UpdateSavedSearchSubscriptionRequest{
					ID:       subID,
					UserID:   userID,
					Triggers: gcpspanner.OptionallySet[[]gcpspanner.SubscriptionTrigger]{IsSet: false, Value: nil},
					Frequency: gcpspanner.OptionallySet[gcpspanner.SavedSearchSnapshotType]{
						Value: gcpspanner.SavedSearchSnapshotTypeImmediate, IsSet: true,
					},
				},
				returnedError: nil,
			},
			getCfg: &mockGetSavedSearchSubscriptionConfig{
				expectedSubscriptionID: subID,
				expectedUserID:         userID,
				result: &gcpspanner.SavedSearchSubscriptionView{
					SavedSearchSubscription: gcpspanner.SavedSearchSubscription{
						ID:            subID,
						ChannelID:     "channel",
						SavedSearchID: "savedsearchid",
						Triggers: []gcpspanner.SubscriptionTrigger{
							gcpspanner.SubscriptionTriggerBrowserImplementationAnyComplete},
						Frequency: updatedSpannerFrequency,
						CreatedAt: now,
						UpdatedAt: now,
					},
					SavedSearchName: "Feature name",
				},
				returnedError: nil,
			},
			expected: &backend.SubscriptionResponse{
				Id:        subID,
				ChannelId: "channel",
				Subscribable: backend.SavedSearchInfo{
					Id:   "savedsearchid",
					Name: "Feature name",
				},
				Triggers: []backend.SubscriptionTriggerResponseItem{
					{
						Value: backendtypes.AttemptToStoreSubscriptionTrigger(
							backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete),
						RawValue: nil,
					},
				},
				Frequency: backend.SubscriptionFrequency(updatedFrequency),
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectedError: nil,
		},
		{
			name: "not found",
			input: backend.UpdateSubscriptionRequest{
				UpdateMask: []backend.UpdateSubscriptionRequestUpdateMask{
					backend.UpdateSubscriptionRequestMaskTriggers},
				Triggers:  &updatedTriggers,
				Frequency: nil,
			},
			updateCfg: &mockUpdateSavedSearchSubscriptionConfig{
				expectedRequest: gcpspanner.UpdateSavedSearchSubscriptionRequest{
					ID:     subID,
					UserID: userID,
					Triggers: gcpspanner.OptionallySet[[]gcpspanner.SubscriptionTrigger]{
						Value: []gcpspanner.SubscriptionTrigger{
							gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToNewly,
							gcpspanner.SubscriptionTriggerFeatureBaselineRegressionToLimited,
						}, IsSet: true,
					},
					Frequency: gcpspanner.OptionallySet[gcpspanner.SavedSearchSnapshotType]{
						Value: "",
						IsSet: false,
					},
				},
				returnedError: gcpspanner.ErrQueryReturnedNoResults,
			},
			getCfg:        nil,
			expected:      nil,
			expectedError: backendtypes.ErrEntityDoesNotExist,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                                    t,
				mockUpdateSavedSearchSubscriptionCfg: tc.updateCfg,
				mockGetSavedSearchSubscriptionCfg:    tc.getCfg,
			}
			b := NewBackend(mock)
			resp, err := b.UpdateSavedSearchSubscription(context.Background(), userID, subID, tc.input)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error. got %v, want %v", err, tc.expectedError)
			}
			if diff := cmp.Diff(tc.expected, resp, getTriggerCmpOption()); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeleteSavedSearchSubscription(t *testing.T) {
	const (
		userID = "user123"
		subID  = "sub456"
	)

	testCases := []struct {
		name          string
		cfg           *mockDeleteSavedSearchSubscriptionConfig
		expectedError error
	}{
		{
			name: "success",
			cfg: &mockDeleteSavedSearchSubscriptionConfig{
				expectedSubscriptionID: subID,
				expectedUserID:         userID,
				returnedError:          nil,
			},
			expectedError: nil,
		},
		{
			name: "not found",
			cfg: &mockDeleteSavedSearchSubscriptionConfig{
				expectedSubscriptionID: subID,
				expectedUserID:         userID,
				returnedError:          gcpspanner.ErrQueryReturnedNoResults,
			},
			expectedError: backendtypes.ErrEntityDoesNotExist,
		},
		{
			name: "not authorized",
			cfg: &mockDeleteSavedSearchSubscriptionConfig{
				expectedSubscriptionID: subID,
				expectedUserID:         userID,
				returnedError:          gcpspanner.ErrMissingRequiredRole,
			},
			expectedError: backendtypes.ErrUserNotAuthorizedForAction,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                                    t,
				mockDeleteSavedSearchSubscriptionCfg: tc.cfg,
			}
			b := NewBackend(mock)
			err := b.DeleteSavedSearchSubscription(context.Background(), userID, subID)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error. got %v, want %v", err, tc.expectedError)
			}
		})
	}
}

func assertKnownTrigger(t *testing.T, itemIndex int,
	actual backend.SubscriptionTriggerResponseItem, expectedValue string) {
	t.Helper() // Marks this as a helper function for better test failure reporting.

	val, err := actual.Value.AsSubscriptionTriggerWritable()
	if err != nil {
		t.Errorf("item %d: expected SubscriptionTriggerWritable, but it was not. err: %s", itemIndex, err)
	}
	if string(val) != expectedValue {
		t.Errorf("item %d: unexpected value: got %q, want %q", itemIndex, val, expectedValue)
	}
	if actual.RawValue != nil {
		t.Errorf("item %d: RawValue should be nil for known trigger, got %q", itemIndex, *actual.RawValue)
	}
}

func assertUnknownTrigger(t *testing.T, itemIndex int,
	actual backend.SubscriptionTriggerResponseItem, expectedValue string, expectedRawValue *string) {
	t.Helper()

	val, err := actual.Value.AsEnumUnknown()
	if err != nil {
		t.Errorf("item %d: expected EnumUnknown, but it was not. err : %s", itemIndex, err)
	}
	if string(val) != expectedValue {
		t.Errorf("item %d: unexpected unknown value: got %q, want %q", itemIndex, val, expectedValue)
	}
	if actual.RawValue == nil || expectedRawValue == nil || *actual.RawValue != *expectedRawValue {
		t.Errorf("item %d: incorrect RawValue for unknown trigger: got %v, want %v",
			itemIndex, actual.RawValue, expectedRawValue)
	}
}

func TestSpannerTriggersToBackendTriggers(t *testing.T) {
	testCases := []struct {
		name          string
		inputTriggers []gcpspanner.SubscriptionTrigger
		expectedItems []struct {
			IsUnknown bool
			Value     string
			RawValue  *string
		}
	}{
		{
			name: "All Valid Triggers",
			inputTriggers: []gcpspanner.SubscriptionTrigger{
				gcpspanner.SubscriptionTriggerBrowserImplementationAnyComplete,
				gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToNewly,
			},
			expectedItems: []struct {
				IsUnknown bool
				Value     string
				RawValue  *string
			}{
				{IsUnknown: false,
					Value:    string(backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete),
					RawValue: nil},
				{IsUnknown: false,
					Value:    string(backend.SubscriptionTriggerFeatureBaselineToNewly),
					RawValue: nil},
			},
		},
		{
			name: "Mixed Valid and Unknown Triggers",
			inputTriggers: []gcpspanner.SubscriptionTrigger{
				gcpspanner.SubscriptionTriggerBrowserImplementationAnyComplete,
				"deprecated_trigger",
				gcpspanner.SubscriptionTriggerFeatureBaselineRegressionToLimited,
				"another_unknown",
			},
			expectedItems: []struct {
				IsUnknown bool
				Value     string
				RawValue  *string
			}{
				{IsUnknown: false,
					Value:    string(backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete),
					RawValue: nil},
				{IsUnknown: true,
					Value:    string(backend.EnumUnknownValue),
					RawValue: new("deprecated_trigger")},
				{IsUnknown: false,
					Value:    string(backend.SubscriptionTriggerFeatureBaselineRegressionToLimited),
					RawValue: nil},
				{IsUnknown: true,
					Value:    string(backend.EnumUnknownValue),
					RawValue: new("another_unknown")},
			},
		},
		{
			name:          "All Unknown Triggers",
			inputTriggers: []gcpspanner.SubscriptionTrigger{"unknown1", "unknown2"},
			expectedItems: []struct {
				IsUnknown bool
				Value     string
				RawValue  *string
			}{
				{IsUnknown: true, Value: string(backend.EnumUnknownValue), RawValue: new("unknown1")},
				{IsUnknown: true, Value: string(backend.EnumUnknownValue), RawValue: new("unknown2")},
			},
		},
		{
			name:          "Empty Triggers",
			inputTriggers: []gcpspanner.SubscriptionTrigger{},
			expectedItems: []struct {
				IsUnknown bool
				Value     string
				RawValue  *string
			}{},
		},
		{
			name:          "Nil Triggers",
			inputTriggers: nil,
			expectedItems: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualItems := spannerTriggersToBackendTriggers(tc.inputTriggers)
			if tc.name == "Nil Triggers" {
				if len(actualItems) == 0 && tc.expectedItems == nil {
					actualItems = nil
				} else if actualItems != nil && tc.expectedItems == nil {
					t.Fatalf("unexpected non-nil slice for nil input: got %v", actualItems)
				}
			}

			if len(actualItems) != len(tc.expectedItems) {
				t.Fatalf("length mismatch: got %d, want %d", len(actualItems), len(tc.expectedItems))
			}

			for i, actual := range actualItems {
				expected := tc.expectedItems[i]
				if expected.IsUnknown {
					assertUnknownTrigger(t, i, actual, expected.Value, expected.RawValue)
				} else {
					assertKnownTrigger(t, i, actual, expected.Value)
				}
			}
		})
	}
}

// nolint:ireturn // Required to return the cmp.Option interface for use with go-cmp.
func getNotificationChannelCmpOption() cmp.Option {
	return cmp.AllowUnexported(backend.NotificationChannelResponse_Config{})
}

// nolint:ireturn // Required to return the cmp.Option interface for use with go-cmp.
func getTriggerCmpOption() cmp.Option {
	return cmp.Transformer("triggerValue", func(in backend.SubscriptionTriggerResponseValue) string {
		// AsSubscriptionTriggerWritable returns the value and a boolean indicating if it was that type.
		v1, err1 := in.AsSubscriptionTriggerWritable()
		if err1 == nil {
			return string(v1)
		}
		v2, err2 := in.AsEnumUnknown()
		if err2 == nil {
			return string(v2)
		}
		// Should not happen
		panic(fmt.Sprintf("received the following errors trying to conver trigger value. err1: %s err2: %s",
			err1, err2))
	})
}

func newTestNotificationChannelConfig(t *testing.T, config any) backend.NotificationChannelResponse_Config {
	t.Helper()

	var c backend.NotificationChannelResponse_Config

	bytes, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal test config: %v", err)
	}

	if err := c.UnmarshalJSON(bytes); err != nil {
		t.Fatalf("failed to unmarshal test config into wrapper: %v", err)
	}

	return c
}

func newTestCreateNotificationChannelConfig(t *testing.T, config any) backend.CreateNotificationChannelRequest_Config {
	t.Helper()

	var c backend.CreateNotificationChannelRequest_Config

	bytes, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal test config: %v", err)
	}

	if err := c.UnmarshalJSON(bytes); err != nil {
		t.Fatalf("failed to unmarshal test config into wrapper: %v", err)
	}

	return c
}

func newTestUpdateNotificationChannelRequestConfig(
	t *testing.T, config any) *backend.UpdateNotificationChannelRequest_Config {
	t.Helper()

	var c backend.UpdateNotificationChannelRequest_Config

	bytes, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal test config: %v", err)
	}

	if err := c.UnmarshalJSON(bytes); err != nil {
		t.Fatalf("failed to unmarshal test config into wrapper: %v", err)
	}

	return &c
}

func TestValidateQueryReferences(t *testing.T) {
	testCases := []struct {
		name                string
		query               string
		updateID            *string
		systemSearches      map[string]*gcpspanner.SystemGlobalSavedSearchWithSortOption
		userSearches        map[string]*gcpspanner.SavedSearch
		referencingSearches map[string][]string
		expectedError       error
	}{
		{
			name:                "valid simple query",
			query:               "name:\"flexbox\"",
			updateID:            nil,
			systemSearches:      nil,
			userSearches:        nil,
			referencingSearches: nil,
			expectedError:       nil,
		},
		{
			name:           "valid saved search reference",
			query:          "(saved:search1) name:\"flexbox\"",
			updateID:       nil,
			systemSearches: nil,
			userSearches: map[string]*gcpspanner.SavedSearch{
				"search1": {Query: "name:\"flexbox\""},
			},
			referencingSearches: nil,
			expectedError:       nil,
		},
		{
			name:     "valid hotlist reference",
			query:    "(hotlist:search1) name:\"flexbox\"",
			updateID: nil,
			systemSearches: map[string]*gcpspanner.SystemGlobalSavedSearchWithSortOption{
				"search1": {SystemGlobalSavedSearch: gcpspanner.SystemGlobalSavedSearch{
					ID:           "",
					Name:         "",
					Description:  nil,
					Scope:        "",
					AuthorID:     "",
					CreatedAt:    time.Time{},
					UpdatedAt:    time.Time{},
					DisplayOrder: 0,
					Status:       "",
					Query:        "name:\"flexbox\"",
				}},
			},
			userSearches:        nil,
			referencingSearches: nil,
			expectedError:       nil,
		},
		{
			name:                "invalid saved search - missing",
			query:               "(saved:missing) name:\"flexbox\"",
			updateID:            nil,
			systemSearches:      nil,
			userSearches:        nil,
			referencingSearches: nil,
			expectedError:       backendtypes.ErrSavedSearchNotFound,
		},
		{
			name:                "invalid hotlist - missing",
			query:               "(hotlist:missing) name:\"flexbox\"",
			updateID:            nil,
			systemSearches:      nil,
			userSearches:        nil,
			referencingSearches: nil,
			expectedError:       backendtypes.ErrHotlistNotFound,
		},
		{
			name:           "invalid depth - 3 levels",
			query:          "(saved:search1) name:\"flexbox\"",
			updateID:       nil,
			systemSearches: nil,
			userSearches: map[string]*gcpspanner.SavedSearch{
				"search1": {Query: "saved:search2"},
				"search2": {Query: "saved:search3"},
				"search3": {Query: "name:\"flexbox\""},
			},
			referencingSearches: nil,
			expectedError:       backendtypes.ErrSavedSearchMaxDepthExceeded,
		},
		{
			name:           "valid depth - 2 levels",
			query:          "(saved:search1) name:\"flexbox\"",
			updateID:       nil,
			systemSearches: nil,
			userSearches: map[string]*gcpspanner.SavedSearch{
				"search1": {Query: "saved:search2"},
				"search2": {Query: "name:\"flexbox\""},
			},
			referencingSearches: nil,
			expectedError:       nil,
		},
		{
			name:           "cycle detection - direct",
			query:          "(saved:search1) name:\"flexbox\"",
			updateID:       nil,
			systemSearches: nil,
			userSearches: map[string]*gcpspanner.SavedSearch{
				"search1": {Query: "saved:search1"},
			},
			referencingSearches: nil,
			expectedError:       backendtypes.ErrSavedSearchCycleDetected,
		},
		{
			name:           "cycle detection - indirect",
			query:          "(saved:search1) name:\"flexbox\"",
			updateID:       nil,
			systemSearches: nil,
			userSearches: map[string]*gcpspanner.SavedSearch{
				"search1": {Query: "saved:search2"},
				"search2": {Query: "saved:search1"},
			},
			referencingSearches: nil,
			expectedError:       backendtypes.ErrSavedSearchCycleDetected,
		},
		{
			name:           "transitive depth violation during update",
			query:          "(saved:child) name:\"flexbox\"",
			updateID:       new("parent"),
			systemSearches: nil,
			userSearches: map[string]*gcpspanner.SavedSearch{
				"child":      {Query: "saved:grandchild"},
				"grandchild": {Query: "name:\"flexbox\""},
			},
			referencingSearches: map[string][]string{
				"parent": {"ancestor"},
			},
			expectedError: backendtypes.ErrSavedSearchMaxDepthExceeded,
		},
		{
			name:                "entirely single saved search",
			query:               "saved:search1",
			updateID:            nil,
			systemSearches:      nil,
			userSearches:        nil,
			referencingSearches: nil,
			expectedError:       backendtypes.ErrQueryConsistsEntirelyOfSavedSearch,
		},
		{
			name:                "entirely single hotlist",
			query:               "hotlist:search1",
			updateID:            nil,
			systemSearches:      nil,
			userSearches:        nil,
			referencingSearches: nil,
			expectedError:       backendtypes.ErrQueryConsistsEntirelyOfSavedSearch,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t: t,
				mockGetSystemGlobalSavedSearchCfg: &mockGetSystemGlobalSavedSearchConfig{
					results: tc.systemSearches,
				},
				mockGetSavedSearchCfg: &mockGetSavedSearchConfig{
					results: tc.userSearches,
				},
				mockGetReferencingSavedSearchIDsCfg: &mockGetReferencingSavedSearchIDsConfig{
					results: tc.referencingSearches,
				},
			}
			backend := NewBackend(mock)
			err := backend.ValidateQueryReferences(context.Background(), tc.query, tc.updateID)
			if (tc.expectedError != nil || err != nil) && !errors.Is(err, tc.expectedError) {
				t.Errorf("expected error %v, got %v", tc.expectedError, err)
			}
		})
	}
}

func newMockSystemGlobalSavedSearchWithSortOption(
	id string, query string, hasCustomSortOrder bool) *gcpspanner.SystemGlobalSavedSearchWithSortOption {
	return &gcpspanner.SystemGlobalSavedSearchWithSortOption{
		SystemGlobalSavedSearch: gcpspanner.SystemGlobalSavedSearch{
			ID:           id,
			Name:         "Mock Name",
			Description:  nil,
			Query:        query,
			Scope:        gcpspanner.SystemGlobalScope,
			AuthorID:     "system",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			DisplayOrder: 0,
			Status:       "LISTED",
		},
		HasCustomSortOrder: hasCustomSortOrder,
	}
}

func TestExpandSavedSearches_Success(t *testing.T) {
	customID := "h-custom"
	testCases := []struct {
		name               string
		id                 string
		globalSearches     map[string]*gcpspanner.SystemGlobalSavedSearchWithSortOption
		expectedNode       *searchtypes.SearchNode
		expectedSortTarget *string
	}{
		// This test case verifies that 'hotlist:all' expands to an empty query AST.
		// This AST is used directly in TestFeaturesSearch_HotlistAll in lib/gcpspanner/hotlist_subquery_test.go.
		// If this behavior changes, update both tests!
		{
			name: "empty query returns special node",
			id:   "all",
			globalSearches: map[string]*gcpspanner.SystemGlobalSavedSearchWithSortOption{
				"all": newMockSystemGlobalSavedSearchWithSortOption("all", "", false),
			},
			expectedNode:       searchtypes.EmptySearchNode(),
			expectedSortTarget: nil,
		},
		{
			name: "hotlist with custom sort order and empty query",
			id:   "h-custom",
			globalSearches: map[string]*gcpspanner.SystemGlobalSavedSearchWithSortOption{
				"h-custom": newMockSystemGlobalSavedSearchWithSortOption("h-custom", "", true),
			},
			expectedNode:       searchtypes.EmptySearchNode(),
			expectedSortTarget: &customID,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t: t,
				mockGetSystemGlobalSavedSearchCfg: &mockGetSystemGlobalSavedSearchConfig{
					results: tc.globalSearches,
				},
			}
			backend := NewBackend(mock)
			node := &searchtypes.SearchNode{
				Keyword: searchtypes.KeywordNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierHotlist,
					Operator:   searchtypes.OperatorNone,
					Value:      tc.id,
				},
				Children: nil,
			}

			expanded, sortTgt, err := backend.expandSavedSearches(context.Background(), node, 0, map[string]struct{}{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(expanded, tc.expectedNode) {
				t.Errorf("expected node %+v, got %+v", tc.expectedNode, expanded)
			}
			if !reflect.DeepEqual(sortTgt, tc.expectedSortTarget) {
				t.Errorf("expected sort target %+v, got %+v", tc.expectedSortTarget, sortTgt)
			}
		})
	}
}

func TestExpandSavedSearches_Error(t *testing.T) {
	testCases := []struct {
		name           string
		id             string
		globalSearches map[string]*gcpspanner.SystemGlobalSavedSearchWithSortOption
		expectedError  error
	}{
		{
			name: "max depth exceeded",
			id:   "h1",
			globalSearches: map[string]*gcpspanner.SystemGlobalSavedSearchWithSortOption{
				"h1": newMockSystemGlobalSavedSearchWithSortOption("h1", "hotlist:h2", false),
				"h2": newMockSystemGlobalSavedSearchWithSortOption("h2", "hotlist:h3", false),
				"h3": newMockSystemGlobalSavedSearchWithSortOption("h3", "feat1", false),
			},
			expectedError: backendtypes.ErrSavedSearchMaxDepthExceeded,
		},
		{
			name:           "hotlist not found",
			id:             "nonexistent",
			globalSearches: map[string]*gcpspanner.SystemGlobalSavedSearchWithSortOption{},
			expectedError:  backendtypes.ErrHotlistNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t: t,
				mockGetSystemGlobalSavedSearchCfg: &mockGetSystemGlobalSavedSearchConfig{
					results: tc.globalSearches,
				},
			}
			backend := NewBackend(mock)
			node := &searchtypes.SearchNode{
				Keyword: searchtypes.KeywordNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierHotlist,
					Operator:   searchtypes.OperatorNone,
					Value:      tc.id,
				},
				Children: nil,
			}

			_, _, err := backend.expandSavedSearches(context.Background(), node, 0, map[string]struct{}{})
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("expected error %v, got %v", tc.expectedError, err)
			}
		})
	}
}
