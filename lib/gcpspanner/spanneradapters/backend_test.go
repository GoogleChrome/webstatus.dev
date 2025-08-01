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
	"errors"
	"math/big"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// nolint: gochecknoglobals
var (
	nonNilInputPageToken = valuePtr[string]("input-token")
	nonNilNextPageToken  = valuePtr[string]("test-token")
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

type mockBackendSpannerClient struct {
	t                                    *testing.T
	aggregationData                      []gcpspanner.WPTRunAggregationMetricWithTime
	featureData                          []gcpspanner.WPTRunFeatureMetricWithTime
	chromeDailyUsageData                 []gcpspanner.ChromeDailyUsageStatWithDate
	mockFeaturesSearchCfg                mockFeaturesSearchConfig
	mockGetFeatureCfg                    mockGetFeatureConfig
	mockGetIDByFeaturesIDCfg             mockGetIDByFeaturesIDConfig
	mockListBrowserFeatureCountMetricCfg mockListBrowserFeatureCountMetricConfig
	mockListMissingOneImplCountsCfg      mockListMissingOneImplCountsConfig
	mockListMissingOneImplFeaturesCfg    mockListMissingOneImplFeaturesConfig
	mockListBaselineStatusCountsCfg      mockListBaselineStatusCountsConfig
	mockCreateNewUserSavedSearchCfg      *mockCreateNewUserSavedSearchConfig
	mockGetUserSavedSearchCfg            *mockGetUserSavedSearchConfig
	mockDeleteUserSavedSearchCfg         *mockDeleteUserSavedSearchConfig
	mockListUserSavedSearchesCfg         *mockListUserSavedSearchesConfig
	mockUpdateUserSavedSearchCfg         *mockUpdateUserSavedSearchConfig
	mockAddUserSearchBookmarkCfg         *mockAddUserSearchBookmarkConfig
	mockDeleteUserSearchBookmarkCfg      *mockDeleteUserSearchBookmarkConfig
	pageToken                            *string
	err                                  error
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

	return c.mockGetFeatureCfg.result, c.mockFeaturesSearchCfg.returnedError
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
					TotalTests: valuePtr[int64](20),
					TestPass:   valuePtr[int64](10),
				},
				{
					TimeStart:  time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
					RunID:      9,
					TotalTests: valuePtr[int64](19),
					TestPass:   valuePtr[int64](9),
				},
				{
					TimeStart:  time.Date(2000, time.January, 8, 0, 0, 0, 0, time.UTC),
					RunID:      8,
					TotalTests: valuePtr[int64](18),
					TestPass:   valuePtr[int64](8),
				},
			},
			pageToken: nonNilNextPageToken,
			err:       nil,
			expectedOutput: []backend.WPTRunMetric{
				{
					RunTimestamp:    time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: valuePtr[int64](20),
					TestPassCount:   valuePtr[int64](10),
				},
				{
					RunTimestamp:    time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: valuePtr[int64](19),
					TestPassCount:   valuePtr[int64](9),
				},
				{
					RunTimestamp:    time.Date(2000, time.January, 8, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: valuePtr[int64](18),
					TestPassCount:   valuePtr[int64](8),
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
			pageToken:         valuePtr(""),
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
						Count:     valuePtr[int64](10),
						Timestamp: time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
					},
					{
						Count:     valuePtr[int64](9),
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
						TotalTests: valuePtr[int64](20),
						TestPass:   valuePtr[int64](10),
					},
				},
				{
					WPTRunFeatureMetricWithTime: gcpspanner.WPTRunFeatureMetricWithTime{
						TimeStart:  time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
						RunID:      9,
						TotalTests: valuePtr[int64](19),
						TestPass:   valuePtr[int64](9),
					},
				},
				{
					WPTRunFeatureMetricWithTime: gcpspanner.WPTRunFeatureMetricWithTime{
						TimeStart:  time.Date(2000, time.January, 8, 0, 0, 0, 0, time.UTC),
						RunID:      8,
						TotalTests: valuePtr[int64](18),
						TestPass:   valuePtr[int64](8),
					},
				},
			},
			pageToken: nonNilNextPageToken,
			err:       nil,
			expectedOutput: []backend.WPTRunMetric{
				{
					RunTimestamp:    time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: valuePtr[int64](20),
					TestPassCount:   valuePtr[int64](10),
				},
				{
					RunTimestamp:    time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: valuePtr[int64](19),
					TestPassCount:   valuePtr[int64](9),
				},
				{
					RunTimestamp:    time.Date(2000, time.January, 8, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: valuePtr[int64](18),
					TestPassCount:   valuePtr[int64](8),
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
			pageToken:         valuePtr(""),
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
						Count:     valuePtr[int64](90),
						Timestamp: time.Date(2010, time.March, 10, 0, 0, 0, 0, time.UTC),
					},
					{
						Count:     valuePtr[int64](99),
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
						FeatureId: valuePtr("foo"),
					},
					{
						FeatureId: valuePtr("bar"),
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
						Count:     valuePtr[int64](89),
						Timestamp: time.Date(2010, time.January, 10, 0, 0, 0, 0, time.UTC),
					},
					{
						Count:     valuePtr[int64](99),
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
			inputStatus:   valuePtr("high"),
			inputLowDate:  nil,
			inputHighDate: nil,
			expected: &backend.BaselineInfo{
				Status:   valuePtr(backend.Widely),
				LowDate:  nil,
				HighDate: nil,
			},
		},
		{
			name:          "Low Status to Newly",
			inputStatus:   valuePtr("low"),
			inputLowDate:  nil,
			inputHighDate: nil,
			expected: &backend.BaselineInfo{
				Status:   valuePtr(backend.Newly),
				LowDate:  nil,
				HighDate: nil,
			},
		},
		{
			name:          "None Status to Limited",
			inputStatus:   valuePtr("none"),
			inputLowDate:  nil,
			inputHighDate: nil,
			expected: &backend.BaselineInfo{
				Status:   valuePtr(backend.Limited),
				LowDate:  nil,
				HighDate: nil,
			},
		},
		{
			name:          "Status with Low Date",
			inputStatus:   valuePtr("none"),
			inputLowDate:  valuePtr(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)),
			inputHighDate: nil,
			expected: &backend.BaselineInfo{
				Status:   valuePtr(backend.Limited),
				LowDate:  &openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
				HighDate: nil,
			},
		},
		{
			name:          "Status with Low Date & High Date",
			inputStatus:   valuePtr("none"),
			inputLowDate:  valuePtr(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)),
			inputHighDate: valuePtr(time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)),
			expected: &backend.BaselineInfo{
				Status:   valuePtr(backend.Limited),
				LowDate:  &openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
				HighDate: &openapi_types.Date{Time: time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)},
			},
		},
		{
			name:          "Invalid Status to nil",
			inputStatus:   valuePtr("invalid"),
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
							Status:     valuePtr("low"),
							LowDate:    valuePtr(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)),
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
									ImplementationDate: valuePtr(
										time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC)),
									ImplementationVersion: valuePtr("103"),
								},
							},
							SpecLinks:     nil,
							ChromiumUsage: big.NewRat(91, 100),
						},
						{
							Name:       "feature 2",
							FeatureKey: "feature2",
							Status:     valuePtr("high"),
							LowDate:    valuePtr(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)),
							HighDate:   valuePtr(time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)),
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
									FeatureRunDetails: map[string]interface{}{
										"test": "browser2-exp",
									},
								},
							},
							ImplementationStatuses: []*gcpspanner.ImplementationStatus{
								{
									BrowserName:          "browser1",
									ImplementationStatus: gcpspanner.Available,
									ImplementationDate: valuePtr(
										time.Date(1998, time.January, 1, 0, 0, 0, 0, time.UTC)),
									ImplementationVersion: valuePtr("101"),
								},
								{
									BrowserName:          "browser2",
									ImplementationStatus: gcpspanner.Available,
									ImplementationDate: valuePtr(
										time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC)),
									ImplementationVersion: valuePtr("102"),
								},
							},
							SpecLinks: []string{
								"link1",
								"link2",
							},
							ChromiumUsage: big.NewRat(10, 100),
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
							Status: valuePtr(backend.Newly),
							LowDate: valuePtr(
								openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
							),
							HighDate: nil,
						},
						FeatureId: "feature1",
						Name:      "feature 1",
						Spec:      nil,
						Usage: &backend.BrowserUsage{
							Chrome: &backend.ChromeUsageInfo{
								Daily: valuePtr[float64](0.91),
							},
						},
						Wpt: &backend.FeatureWPTSnapshots{
							Experimental: &map[string]backend.WPTFeatureData{
								"browser3": {
									Score:    valuePtr[float64](0.2),
									Metadata: nil,
								},
							},
							Stable: &map[string]backend.WPTFeatureData{
								"browser3": {
									Score:    valuePtr[float64](0.5),
									Metadata: nil,
								},
							},
						},
						BrowserImplementations: &map[string]backend.BrowserImplementation{
							"browser3": {
								Status: valuePtr(backend.Available),
								Date: &openapi_types.Date{
									Time: time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC)},
								Version: valuePtr("103"),
							},
						},
						// TODO https://github.com/GoogleChrome/webstatus.dev/issues/1675
						DeveloperSignals: nil,
						// TODO https://github.com/GoogleChrome/webstatus.dev/issues/1671
						Evolution: nil,
					},
					{
						Baseline: &backend.BaselineInfo{
							Status: valuePtr(backend.Widely),
							LowDate: valuePtr(
								openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
							),
							HighDate: valuePtr(
								openapi_types.Date{Time: time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)},
							),
						},
						FeatureId: "feature2",
						Name:      "feature 2",
						Spec: &backend.FeatureSpecInfo{
							Links: &[]backend.SpecLink{
								{
									Link: valuePtr("link1"),
								},
								{
									Link: valuePtr("link2"),
								},
							},
						},
						Usage: &backend.BrowserUsage{
							Chrome: &backend.ChromeUsageInfo{
								Daily: valuePtr[float64](0.1),
							},
						},
						Wpt: &backend.FeatureWPTSnapshots{
							Experimental: &map[string]backend.WPTFeatureData{
								"browser1": {
									Score:    valuePtr[float64](0.5),
									Metadata: nil,
								},
								"browser2": {
									Score: valuePtr[float64](0.1),
									Metadata: &map[string]interface{}{
										"test": "browser2-exp",
									},
								},
							},
							Stable: &map[string]backend.WPTFeatureData{
								"browser1": {
									Score:    valuePtr[float64](0.5),
									Metadata: nil,
								},
								"browser2": {
									Score:    valuePtr[float64](0.25),
									Metadata: nil,
								},
							},
						},
						BrowserImplementations: &map[string]backend.BrowserImplementation{
							"browser1": {
								Status: valuePtr(backend.Available),
								Date: &openapi_types.Date{
									Time: time.Date(1998, time.January, 1, 0, 0, 0, 0, time.UTC)},
								Version: valuePtr("101"),
							},
							"browser2": {
								Status: valuePtr(backend.Available),
								Date: &openapi_types.Date{
									Time: time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC)},
								Version: valuePtr("102"),
							},
						},
						// TODO https://github.com/GoogleChrome/webstatus.dev/issues/1675
						DeveloperSignals: nil,
						// TODO https://github.com/GoogleChrome/webstatus.dev/issues/1671
						Evolution: nil,
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

			if !reflect.DeepEqual(page, tc.expectedPage) {
				t.Error("unexpected page")
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
	testCases := []struct {
		name               string
		cfg                mockGetFeatureConfig
		inputFeatureID     string
		inputWPTMetricView backend.WPTMetricView
		inputBrowsers      BrowserList
		expectedFeature    *backend.Feature
	}{
		{
			name:               "regular",
			inputFeatureID:     "feature1",
			inputWPTMetricView: backend.SubtestCounts,
			inputBrowsers: []backend.BrowserPathParam{
				"browser1",
				"browser2",
				"browser3",
			},
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
					Status:     valuePtr("low"),
					LowDate:    valuePtr(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)),
					HighDate:   nil,
					StableMetrics: []*gcpspanner.FeatureResultMetric{
						{
							BrowserName: "browser3",
							PassRate:    big.NewRat(10, 20),
							FeatureRunDetails: map[string]interface{}{
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
					ChromiumUsage: nil,
				},
				returnedError: nil,
			},
			expectedFeature: &backend.Feature{
				Baseline: &backend.BaselineInfo{
					Status: valuePtr(backend.Newly),
					LowDate: valuePtr(
						openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
					),
					HighDate: nil,
				},
				FeatureId: "feature1",
				Name:      "feature 1",
				Spec: &backend.FeatureSpecInfo{
					Links: &[]backend.SpecLink{
						{
							Link: valuePtr("link1"),
						},
						{
							Link: valuePtr("link2"),
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
							Score:    valuePtr[float64](0.2),
							Metadata: nil,
						},
					},
					Stable: &map[string]backend.WPTFeatureData{
						"browser3": {
							Score: valuePtr[float64](0.5),
							Metadata: &map[string]interface{}{
								"browser3": "test",
							},
						},
					},
				},
				BrowserImplementations: &map[string]backend.BrowserImplementation{
					"browser3": {
						Status:  valuePtr(backend.Available),
						Date:    nil,
						Version: nil,
					},
				},
				// TODO https://github.com/GoogleChrome/webstatus.dev/issues/1675
				DeveloperSignals: nil,
				// TODO https://github.com/GoogleChrome/webstatus.dev/issues/1671
				Evolution: nil,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                 t,
				mockGetFeatureCfg: tc.cfg,
			}
			bk := NewBackend(mock)
			feature, err := bk.GetFeature(
				context.Background(),
				tc.inputFeatureID, tc.inputWPTMetricView, tc.inputBrowsers)
			if !errors.Is(err, tc.cfg.returnedError) {
				t.Error("unexpected error")
			}

			if !CompareFeatures(*feature, *tc.expectedFeature) {
				t.Error("unexpected feature")
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
				Description: valuePtr("test description"),
				Query:       "test query",
			},
			createCfg: &mockCreateNewUserSavedSearchConfig{
				expectedNewSearch: gcpspanner.CreateUserSavedSearchRequest{
					OwnerUserID: "user1",
					Query:       "test query",
					Name:        "test search",
					Description: valuePtr("test description"),
				},
				result:        valuePtr("saved-search-id"),
				returnedError: nil,
			},
			getCfg: &mockGetUserSavedSearchConfig{
				expectedAuthenticatedUserID: valuePtr("user1"),
				expectedSavedSearchID:       "saved-search-id",
				result: &gcpspanner.UserSavedSearch{
					SavedSearch: gcpspanner.SavedSearch{
						Name:        "test search",
						Description: valuePtr("test description"),
						Query:       "test query",
						Scope:       gcpspanner.UserPublicScope,
						AuthorID:    "user1",
						CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						ID:          "saved-search-id",
					},
					Role:         valuePtr(string(gcpspanner.SavedSearchOwner)),
					IsBookmarked: valuePtr(true),
				},
				returnedError: nil,
			},
			expectedOutput: &backend.SavedSearchResponse{
				Id:          "saved-search-id",
				CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				Name:        "test search",
				Description: valuePtr("test description"),
				Query:       "test query",
				Permissions: &backend.UserSavedSearchPermissions{
					Role: valuePtr(backend.SavedSearchOwner),
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
				Description: valuePtr("test description"),
				Query:       "test query",
			},
			createCfg: &mockCreateNewUserSavedSearchConfig{
				expectedNewSearch: gcpspanner.CreateUserSavedSearchRequest{
					OwnerUserID: "user1",
					Query:       "test query",
					Name:        "test search",
					Description: valuePtr("test description"),
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
				Description: valuePtr("test description"),
				Query:       "test query",
			},
			createCfg: &mockCreateNewUserSavedSearchConfig{
				expectedNewSearch: gcpspanner.CreateUserSavedSearchRequest{
					OwnerUserID: "user1",
					Query:       "test query",
					Name:        "test search",
					Description: valuePtr("test description"),
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
				Description: valuePtr("test description"),
				Query:       "test query",
			},
			createCfg: &mockCreateNewUserSavedSearchConfig{
				expectedNewSearch: gcpspanner.CreateUserSavedSearchRequest{
					OwnerUserID: "user1",
					Query:       "test query",
					Name:        "test search",
					Description: valuePtr("test description"),
				},
				result:        valuePtr("saved-search-id"),
				returnedError: nil,
			},
			getCfg: &mockGetUserSavedSearchConfig{
				expectedAuthenticatedUserID: valuePtr("user1"),
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
			userID:        valuePtr("user1"),
			savedSearchID: "saved-search-id",
			cfg: &mockGetUserSavedSearchConfig{
				expectedAuthenticatedUserID: valuePtr("user1"),
				expectedSavedSearchID:       "saved-search-id",
				result: &gcpspanner.UserSavedSearch{
					SavedSearch: gcpspanner.SavedSearch{
						Name:        "test search",
						Description: valuePtr("test description"),
						Query:       "test query",
						Scope:       gcpspanner.UserPublicScope,
						AuthorID:    "user1",
						CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						ID:          "saved-search-id",
					},
					Role:         valuePtr(string(gcpspanner.SavedSearchOwner)),
					IsBookmarked: valuePtr(true),
				},
				returnedError: nil,
			},
			expectedOutput: &backend.SavedSearchResponse{
				Id:          "saved-search-id",
				CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				Name:        "test search",
				Query:       "test query",
				Description: valuePtr("test description"),
				BookmarkStatus: &backend.UserSavedSearchBookmark{
					Status: backend.BookmarkActive,
				},
				Permissions: &backend.UserSavedSearchPermissions{
					Role: valuePtr(backend.SavedSearchOwner),
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
						Description: valuePtr("test description"),
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
				Description:    valuePtr("test description"),
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
								Description: valuePtr("test description"),
								Query:       "test query",
								Scope:       gcpspanner.UserPublicScope,
								AuthorID:    "user1",
								CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
								UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
								ID:          "saved-search-id-2",
							},
							IsBookmarked: valuePtr(true),
							Role:         nil,
						},
					},
				},
				returnedError: nil,
			},
			expectedPage: &backend.UserSavedSearchPage{
				Metadata: nil,
				Data: valuePtr([]backend.SavedSearchResponse{
					{
						Id:          "saved-search-id-2",
						CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						Name:        "z",
						Description: valuePtr("test description"),
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
			pageToken: valuePtr("inputToken"),
			cfg: &mockListUserSavedSearchesConfig{
				expectedUserID:    "user1",
				expectedPageSize:  10,
				expectedPageToken: valuePtr("inputToken"),
				result: &gcpspanner.UserSavedSearchesPage{
					NextPageToken: valuePtr("nextToken"),
					Searches: []gcpspanner.UserSavedSearch{
						{
							SavedSearch: gcpspanner.SavedSearch{
								Name:        "test search",
								Description: valuePtr("test description"),
								Query:       "test query",
								Scope:       gcpspanner.UserPublicScope,
								AuthorID:    "user1",
								CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
								UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
								ID:          "saved-search-id",
							},
							Role:         valuePtr(string(gcpspanner.SavedSearchOwner)),
							IsBookmarked: valuePtr(true),
						},
						{
							SavedSearch: gcpspanner.SavedSearch{
								Name:        "z",
								Description: valuePtr("test description"),
								Query:       "test query",
								Scope:       gcpspanner.UserPublicScope,
								AuthorID:    "user1",
								CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
								UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
								ID:          "saved-search-id-2",
							},
							IsBookmarked: valuePtr(true),
							Role:         nil,
						},
					},
				},
				returnedError: nil,
			},
			expectedPage: &backend.UserSavedSearchPage{
				Metadata: &backend.PageMetadata{
					NextPageToken: valuePtr("nextToken"),
				},
				Data: valuePtr([]backend.SavedSearchResponse{
					{
						Id:          "saved-search-id",
						CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						Name:        "test search",
						Description: valuePtr("test description"),
						Query:       "test query",
						Permissions: &backend.UserSavedSearchPermissions{
							Role: valuePtr(backend.SavedSearchOwner),
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
						Description: valuePtr("test description"),
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
				Name:        valuePtr("test search name"),
				Description: valuePtr("test desc"),
				Query:       valuePtr("test query"),
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
						Value: valuePtr("test desc"),
					},
					Query: gcpspanner.OptionallySet[string]{
						IsSet: true,
						Value: "test query",
					},
				},
				returnedError: nil,
			},
			mockGetCfg: &mockGetUserSavedSearchConfig{
				expectedAuthenticatedUserID: valuePtr("test-user"),
				expectedSavedSearchID:       "test-id",
				result: &gcpspanner.UserSavedSearch{
					SavedSearch: gcpspanner.SavedSearch{
						ID:          "test-id",
						Name:        "test search name",
						Description: valuePtr("test desc"),
						Query:       "test query",
						Scope:       gcpspanner.UserPublicScope,
						AuthorID:    "test-user",
						CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
					},
					Role:         valuePtr(string(gcpspanner.SavedSearchOwner)),
					IsBookmarked: valuePtr(true),
				},
				returnedError: nil,
			},
			expectedResp: &backend.SavedSearchResponse{
				Id:          "test-id",
				CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				Name:        "test search name",
				Description: valuePtr("test desc"),
				Query:       "test query",
				Permissions: &backend.UserSavedSearchPermissions{
					Role: valuePtr(backend.SavedSearchOwner),
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
				Name:        valuePtr("test search name"),
				Description: valuePtr("test desc"),
				Query:       valuePtr("test query"),
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
						Value: valuePtr("test desc"),
					},
					Query: gcpspanner.OptionallySet[string]{
						IsSet: true,
						Value: "test query",
					},
				},
				returnedError: nil,
			},
			mockGetCfg: &mockGetUserSavedSearchConfig{
				expectedAuthenticatedUserID: valuePtr("test-user"),
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
				Name:        valuePtr("test search name"),
				Description: valuePtr("test desc"),
				Query:       valuePtr("test query"),
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
						Value: valuePtr("test desc"),
					},
					Query: gcpspanner.OptionallySet[string]{
						IsSet: true,
						Value: "test query",
					},
				},
				returnedError: nil,
			},
			mockGetCfg: &mockGetUserSavedSearchConfig{
				expectedAuthenticatedUserID: valuePtr("test-user"),
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
				Name:        valuePtr("test search name"),
				Description: valuePtr("test desc"),
				Query:       valuePtr("test query"),
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
						Value: valuePtr("test desc"),
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
				Name:        valuePtr("test search name"),
				Description: valuePtr("test desc"),
				Query:       valuePtr("test query"),
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
						Value: valuePtr("test desc"),
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
				Name:        valuePtr("test search name"),
				Description: valuePtr("test desc"),
				Query:       valuePtr("test query"),
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
						Value: valuePtr("test desc"),
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
			resp, err := backend.UpdateUserSavedSearch(context.Background(), testSavedSearchID, testUserID, tc.updateRequest)
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
				Name:        valuePtr("test name"),
				Description: valuePtr("test description"),
				Query:       valuePtr("test query"),
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
				Name:        valuePtr("test name"),
				Description: valuePtr("test description"),
				Query:       valuePtr("test query"),
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
					Value: valuePtr("test description"),
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
			input: valuePtr[backend.ListFeaturesParamsSort](backend.NameAsc),
			want:  gcpspanner.NewFeatureNameSort(true),
		},
		{
			input: valuePtr[backend.ListFeaturesParamsSort](backend.NameDesc),
			want:  gcpspanner.NewFeatureNameSort(false),
		},
		{
			input: valuePtr[backend.ListFeaturesParamsSort](backend.BaselineStatusAsc),
			want:  gcpspanner.NewBaselineStatusSort(true),
		},
		{
			input: valuePtr[backend.ListFeaturesParamsSort](backend.BaselineStatusDesc),
			want:  gcpspanner.NewBaselineStatusSort(false),
		},
		{
			input: valuePtr(backend.ExperimentalChromeAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "chrome", false),
		},
		{
			input: valuePtr(backend.ExperimentalChromeDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "chrome", false),
		},
		{
			input: valuePtr(backend.ExperimentalEdgeAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "edge", false),
		},
		{
			input: valuePtr(backend.ExperimentalEdgeDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "edge", false),
		},
		{
			input: valuePtr(backend.ExperimentalFirefoxAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "firefox", false),
		},
		{
			input: valuePtr(backend.ExperimentalFirefoxDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "firefox", false),
		},
		{
			input: valuePtr(backend.ExperimentalSafariAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "safari", false),
		},
		{
			input: valuePtr(backend.ExperimentalSafariDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "safari", false),
		},
		{
			input: valuePtr(backend.StableChromeAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "chrome", true),
		},
		{
			input: valuePtr(backend.StableChromeDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "chrome", true),
		},
		{
			input: valuePtr(backend.StableEdgeAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "edge", true),
		},
		{
			input: valuePtr(backend.StableEdgeDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "edge", true),
		},
		{
			input: valuePtr(backend.StableFirefoxAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "firefox", true),
		},
		{
			input: valuePtr(backend.StableFirefoxDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "firefox", true),
		},
		{
			input: valuePtr(backend.StableSafariAsc),
			want:  gcpspanner.NewBrowserImplSort(true, "safari", true),
		},
		{
			input: valuePtr(backend.StableSafariDesc),
			want:  gcpspanner.NewBrowserImplSort(false, "safari", true),
		},
		{
			input: valuePtr(backend.AvailabilityChromeAsc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(true, "chrome"),
		},
		{
			input: valuePtr(backend.AvailabilityChromeDesc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(false, "chrome"),
		},
		{
			input: valuePtr(backend.AvailabilityEdgeAsc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(true, "edge"),
		},
		{
			input: valuePtr(backend.AvailabilityEdgeDesc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(false, "edge"),
		},
		{
			input: valuePtr(backend.AvailabilityFirefoxAsc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(true, "firefox"),
		},
		{
			input: valuePtr(backend.AvailabilityFirefoxDesc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(false, "firefox"),
		},
		{
			input: valuePtr(backend.AvailabilitySafariAsc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(true, "safari"),
		},
		{
			input: valuePtr(backend.AvailabilitySafariDesc),
			want:  gcpspanner.NewBrowserFeatureSupportSort(false, "safari"),
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
				Name:       "feature 1",
				FeatureKey: "feature1",
				Status:     valuePtr("low"),
				LowDate:    valuePtr(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)),
				HighDate:   nil,
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
			},

			expectedFeature: &backend.Feature{
				Baseline: &backend.BaselineInfo{
					Status: valuePtr(backend.Newly),
					LowDate: valuePtr(
						openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
					),
					HighDate: nil,
				},
				FeatureId: "feature1",
				Name:      "feature 1",
				Spec:      nil,
				Usage: &backend.BrowserUsage{
					Chrome: &backend.ChromeUsageInfo{
						Daily: valuePtr[float64](0.08),
					},
				},
				Wpt:                    nil,
				BrowserImplementations: nil,
				// TODO https://github.com/GoogleChrome/webstatus.dev/issues/1675
				DeveloperSignals: nil,
				// TODO https://github.com/GoogleChrome/webstatus.dev/issues/1671
				Evolution: nil,
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
