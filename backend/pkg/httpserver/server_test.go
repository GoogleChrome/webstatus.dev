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

package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/httpmiddlewares"
)

func valuePtr[T any](in T) *T { return &in }

type MockGetFeatureMetadataConfig struct {
	expectedFeatureID string
	result            *backend.FeatureMetadata
	err               error
}

type MockWebFeatureMetadataStorer struct {
	t                         *testing.T
	mockGetFeatureMetadataCfg MockGetFeatureMetadataConfig
}

func (s *MockWebFeatureMetadataStorer) GetFeatureMetadata(
	_ context.Context,
	featureID string,
) (*backend.FeatureMetadata, error) {
	if featureID != s.mockGetFeatureMetadataCfg.expectedFeatureID {
		s.t.Error("unexpected feature id")
	}

	return s.mockGetFeatureMetadataCfg.result, s.mockGetFeatureMetadataCfg.err
}

type MockListMetricsForFeatureIDBrowserAndChannelConfig struct {
	expectedFeatureID string
	expectedBrowser   string
	expectedChannel   string
	expectedMetric    backend.WPTMetricView
	expectedStartAt   time.Time
	expectedEndAt     time.Time
	expectedPageSize  int
	expectedPageToken *string
	data              []backend.WPTRunMetric
	pageToken         *string
	err               error
}

type MockListMetricsOverTimeWithAggregatedTotalsConfig struct {
	expectedFeatureIDs []string
	expectedBrowser    string
	expectedChannel    string
	expectedMetric     backend.WPTMetricView
	expectedStartAt    time.Time
	expectedEndAt      time.Time
	expectedPageSize   int
	expectedPageToken  *string
	data               []backend.WPTRunMetric
	pageToken          *string
	err                error
}

type MockListChromeDailyUsageStatsConfig struct {
	expectedFeatureID string
	expectedStartAt   time.Time
	expectedEndAt     time.Time
	expectedPageSize  int
	expectedPageToken *string
	data              []backend.ChromeUsageStat
	pageToken         *string
	err               error
}

type MockFeaturesSearchConfig struct {
	expectedPageToken     *string
	expectedPageSize      int
	expectedSearchNode    *searchtypes.SearchNode
	expectedSortBy        *backend.ListFeaturesParamsSort
	expectedWPTMetricView backend.WPTMetricView
	expectedBrowsers      []backend.BrowserPathParam
	page                  *backend.FeaturePage
	err                   error
}

type MockGetFeatureByIDConfig struct {
	expectedFeatureID     string
	expectedWPTMetricView backend.WPTMetricView
	expectedBrowsers      []backend.BrowserPathParam
	data                  *backend.Feature
	err                   error
}

type MockGetIDFromFeatureKeyConfig struct {
	expectedFeatureKey string
	result             *string
	err                error
}

type MockListBrowserFeatureCountMetricConfig struct {
	expectedBrowser   string
	expectedStartAt   time.Time
	expectedEndAt     time.Time
	expectedPageSize  int
	expectedPageToken *string
	pageToken         *string
	page              *backend.BrowserReleaseFeatureMetricsPage
	err               error
}

type MockListMissingOneImplCountsConfig struct {
	expectedTargetBrowser string
	expectedOtherBrowsers []string
	expectedStartAt       time.Time
	expectedEndAt         time.Time
	expectedPageSize      int
	expectedPageToken     *string
	pageToken             *string
	page                  *backend.BrowserReleaseFeatureMetricsPage
	err                   error
}

type MockListMissingOneImplFeaturesConfig struct {
	expectedTargetBrowser string
	expectedOtherBrowsers []string
	expectedtargetDate    time.Time
	expectedPageSize      int
	expectedPageToken     *string
	pageToken             *string
	page                  *backend.MissingOneImplFeaturesPage
	err                   error
}

type MockListBaselineStatusCountsConfig struct {
	expectedStartAt   time.Time
	expectedEndAt     time.Time
	expectedPageSize  int
	expectedPageToken *string
	pageToken         *string
	page              *backend.BaselineStatusMetricsPage
	err               error
}

type MockCreateUserSavedSearchConfig struct {
	expectedSavedSearch backend.SavedSearch
	expectedUserID      string
	output              *backend.SavedSearchResponse
	err                 error
}

type MockDeleteUserSavedSearchConfig struct {
	expectedSavedSearchID string
	expectedUserID        string
	err                   error
}

type MockGetSavedSearchConfig struct {
	expectedSavedSearchID string
	expectedUserID        *string
	output                *backend.SavedSearchResponse
	err                   error
}

type MockListUserSavedSeachesConfig struct {
	expectedUserID    string
	expectedPageSize  int
	expectedPageToken *string
	output            *backend.UserSavedSearchPage
	err               error
}

type MockUpdateUserSavedSearchConfig struct {
	expectedSavedSearchID string
	expectedUserID        string
	expectedUpdateRequest *backend.SavedSearchUpdateRequest
	output                *backend.SavedSearchResponse
	err                   error
}

type MockWPTMetricsStorer struct {
	featureCfg                                        *MockListMetricsForFeatureIDBrowserAndChannelConfig
	aggregateCfg                                      *MockListMetricsOverTimeWithAggregatedTotalsConfig
	featuresSearchCfg                                 *MockFeaturesSearchConfig
	listBrowserFeatureCountMetricCfg                  *MockListBrowserFeatureCountMetricConfig
	listMissingOneImplCountCfg                        *MockListMissingOneImplCountsConfig
	listMissingOneImplFeaturesCfg                     *MockListMissingOneImplFeaturesConfig
	listBaselineStatusCountsCfg                       *MockListBaselineStatusCountsConfig
	listChromeDailyUsageStatsCfg                      *MockListChromeDailyUsageStatsConfig
	getFeatureByIDConfig                              *MockGetFeatureByIDConfig
	getIDFromFeatureKeyConfig                         *MockGetIDFromFeatureKeyConfig
	createUserSavedSearchCfg                          *MockCreateUserSavedSearchConfig
	deleteUserSavedSearchCfg                          *MockDeleteUserSavedSearchConfig
	getSavedSearchCfg                                 *MockGetSavedSearchConfig
	listUserSavedSearchesCfg                          *MockListUserSavedSeachesConfig
	updateUserSavedSearchCfg                          *MockUpdateUserSavedSearchConfig
	t                                                 *testing.T
	callCountListMissingOneImplCounts                 int
	callCountListMissingOneImplFeatures               int
	callCountListBaselineStatusCounts                 int
	callCountListBrowserFeatureCountMetric            int
	callCountFeaturesSearch                           int
	callCountListChromeDailyUsageStats                int
	callCountListMetricsForFeatureIDBrowserAndChannel int
	callCountListMetricsOverTimeWithAggregatedTotals  int
	callCountGetFeature                               int
	callCountCreateUserSavedSearch                    int
	callCountDeleteUserSavedSearch                    int
	callCountGetSavedSearch                           int
	callCountListUserSavedSearches                    int
	callCountUpdateUserSavedSearch                    int
}

func (m *MockWPTMetricsStorer) GetIDFromFeatureKey(
	_ context.Context,
	featureID string,
) (*string, error) {
	if featureID != m.getIDFromFeatureKeyConfig.expectedFeatureKey {
		m.t.Errorf("unexpected feature key %s", featureID)
	}

	return m.getIDFromFeatureKeyConfig.result, m.getIDFromFeatureKeyConfig.err
}

func (m *MockWPTMetricsStorer) ListMetricsForFeatureIDBrowserAndChannel(_ context.Context,
	featureID string, browser string, channel string,
	metric backend.WPTMetricView,
	startAt time.Time, endAt time.Time,
	pageSize int, pageToken *string) ([]backend.WPTRunMetric, *string, error) {
	m.callCountListMetricsForFeatureIDBrowserAndChannel++

	if featureID != m.featureCfg.expectedFeatureID ||
		browser != m.featureCfg.expectedBrowser ||
		channel != m.featureCfg.expectedChannel ||
		metric != m.featureCfg.expectedMetric ||
		!startAt.Equal(m.featureCfg.expectedStartAt) ||
		!endAt.Equal(m.featureCfg.expectedEndAt) ||
		pageSize != m.featureCfg.expectedPageSize ||
		!reflect.DeepEqual(pageToken, m.featureCfg.expectedPageToken) {

		m.t.Errorf("Incorrect arguments. Expected: %v, Got: { %s, %s, %s, %s, %s, %s, %d %v }",
			m.featureCfg, featureID, browser, channel, metric, startAt, endAt, pageSize, pageToken)
	}

	return m.featureCfg.data, m.featureCfg.pageToken, m.featureCfg.err
}

func (m *MockWPTMetricsStorer) ListMetricsOverTimeWithAggregatedTotals(
	_ context.Context,
	featureIDs []string,
	browser string,
	channel string,
	metric backend.WPTMetricView,
	startAt, endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]backend.WPTRunMetric, *string, error) {
	m.callCountListMetricsOverTimeWithAggregatedTotals++

	if !slices.Equal(featureIDs, m.aggregateCfg.expectedFeatureIDs) ||
		browser != m.aggregateCfg.expectedBrowser ||
		channel != m.aggregateCfg.expectedChannel ||
		metric != m.aggregateCfg.expectedMetric ||
		!startAt.Equal(m.aggregateCfg.expectedStartAt) ||
		!endAt.Equal(m.aggregateCfg.expectedEndAt) ||
		pageSize != m.aggregateCfg.expectedPageSize ||
		!reflect.DeepEqual(pageToken, m.aggregateCfg.expectedPageToken) {

		m.t.Errorf("Incorrect arguments. Expected: %v, Got: { %v, %s, %s, %s, %s, %s, %d %v }",
			m.aggregateCfg, featureIDs, browser, channel, metric, startAt, endAt, pageSize, pageToken)
	}

	return m.aggregateCfg.data, m.aggregateCfg.pageToken, m.aggregateCfg.err
}

func (m *MockWPTMetricsStorer) ListChromeDailyUsageStats(
	_ context.Context,
	featureID string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]backend.ChromeUsageStat, *string, error) {
	m.callCountListChromeDailyUsageStats++

	if featureID != m.listChromeDailyUsageStatsCfg.expectedFeatureID ||
		!startAt.Equal(m.listChromeDailyUsageStatsCfg.expectedStartAt) ||
		!endAt.Equal(m.listChromeDailyUsageStatsCfg.expectedEndAt) ||
		pageSize != m.listChromeDailyUsageStatsCfg.expectedPageSize ||
		!reflect.DeepEqual(pageToken, m.listChromeDailyUsageStatsCfg.expectedPageToken) {

		m.t.Errorf("Incorrect arguments. Expected: %v, Got: { %s, %s, %s, %d %v }",
			m.listChromeDailyUsageStatsCfg, featureID, startAt, endAt, pageSize, pageToken)
	}

	return m.listChromeDailyUsageStatsCfg.data,
		m.listChromeDailyUsageStatsCfg.pageToken,
		m.listChromeDailyUsageStatsCfg.err
}

func (m *MockWPTMetricsStorer) FeaturesSearch(
	_ context.Context,
	pageToken *string,
	pageSize int,
	node *searchtypes.SearchNode,
	sortBy *backend.ListFeaturesParamsSort,
	view backend.WPTMetricView,
	browsers []backend.BrowserPathParam,
) (*backend.FeaturePage, error) {
	m.callCountFeaturesSearch++

	if !reflect.DeepEqual(pageToken, m.featuresSearchCfg.expectedPageToken) ||
		pageSize != m.featuresSearchCfg.expectedPageSize ||
		!reflect.DeepEqual(node, m.featuresSearchCfg.expectedSearchNode) ||
		!reflect.DeepEqual(sortBy, m.featuresSearchCfg.expectedSortBy) ||
		view != m.featuresSearchCfg.expectedWPTMetricView ||
		!slices.Equal(browsers, m.featuresSearchCfg.expectedBrowsers) {
		m.t.Errorf("Incorrect arguments. Expected: %v, Got: { %v %d %v %v %v %v }",
			m.featuresSearchCfg, pageSize, pageToken, node, sortBy, view, browsers)
	}

	return m.featuresSearchCfg.page, m.featuresSearchCfg.err
}

func (m *MockWPTMetricsStorer) GetFeature(
	_ context.Context,
	featureID string,
	view backend.WPTMetricView,
	browsers []backend.BrowserPathParam,
) (*backend.Feature, error) {
	m.callCountGetFeature++

	if featureID != m.getFeatureByIDConfig.expectedFeatureID ||
		view != m.getFeatureByIDConfig.expectedWPTMetricView ||
		!slices.Equal(browsers, m.getFeatureByIDConfig.expectedBrowsers) {
		m.t.Errorf("Incorrect arguments. Expected: %v, Got: { %s %v %v }",
			m.getFeatureByIDConfig, featureID, view, browsers)
	}

	return m.getFeatureByIDConfig.data, m.getFeatureByIDConfig.err
}

func (m *MockWPTMetricsStorer) ListBrowserFeatureCountMetric(
	_ context.Context,
	browser string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) (*backend.BrowserReleaseFeatureMetricsPage, error) {
	m.callCountListBrowserFeatureCountMetric++

	if browser != m.listBrowserFeatureCountMetricCfg.expectedBrowser ||
		!startAt.Equal(m.listBrowserFeatureCountMetricCfg.expectedStartAt) ||
		!endAt.Equal(m.listBrowserFeatureCountMetricCfg.expectedEndAt) ||
		pageSize != m.listBrowserFeatureCountMetricCfg.expectedPageSize ||
		!reflect.DeepEqual(pageToken, m.listBrowserFeatureCountMetricCfg.expectedPageToken) {

		m.t.Errorf("Incorrect arguments. Expected: %v, Got: { %v, %s, %s, %d %v }",
			m.listBrowserFeatureCountMetricCfg, browser, startAt, endAt, pageSize, pageToken)
	}

	return m.listBrowserFeatureCountMetricCfg.page, m.listBrowserFeatureCountMetricCfg.err
}

func (m *MockWPTMetricsStorer) ListMissingOneImplCounts(
	_ context.Context,
	targetBrowser string,
	otherBrowsers []string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) (*backend.BrowserReleaseFeatureMetricsPage, error) {
	m.callCountListMissingOneImplCounts++

	if targetBrowser != m.listMissingOneImplCountCfg.expectedTargetBrowser ||
		!slices.Equal(otherBrowsers, m.listMissingOneImplCountCfg.expectedOtherBrowsers) ||
		!startAt.Equal(m.listMissingOneImplCountCfg.expectedStartAt) ||
		!endAt.Equal(m.listMissingOneImplCountCfg.expectedEndAt) ||
		pageSize != m.listMissingOneImplCountCfg.expectedPageSize ||
		!reflect.DeepEqual(pageToken, m.listMissingOneImplCountCfg.expectedPageToken) {

		m.t.Errorf("Incorrect arguments. Expected: %v, Got: { %v, %s, %s, %s, %d %v }",
			m.listMissingOneImplCountCfg, targetBrowser, otherBrowsers, startAt, endAt, pageSize, pageToken)
	}

	return m.listMissingOneImplCountCfg.page, m.listMissingOneImplCountCfg.err
}

func (m *MockWPTMetricsStorer) ListMissingOneImplementationFeatures(
	_ context.Context,
	targetBrowser string,
	otherBrowsers []string,
	targetDate time.Time,
	pageSize int,
	pageToken *string,
) (*backend.MissingOneImplFeaturesPage, error) {
	m.callCountListMissingOneImplFeatures++

	if targetBrowser != m.listMissingOneImplFeaturesCfg.expectedTargetBrowser ||
		!slices.Equal(otherBrowsers, m.listMissingOneImplFeaturesCfg.expectedOtherBrowsers) ||
		!targetDate.Equal(m.listMissingOneImplFeaturesCfg.expectedtargetDate) ||
		pageSize != m.listMissingOneImplFeaturesCfg.expectedPageSize ||
		!reflect.DeepEqual(pageToken, m.listMissingOneImplFeaturesCfg.expectedPageToken) {

		m.t.Errorf("Incorrect arguments. Expected: %v, Got: { %v, %s, %s, %d %v }",
			m.listMissingOneImplFeaturesCfg, targetBrowser, otherBrowsers, targetDate, pageSize, pageToken)
	}

	return m.listMissingOneImplFeaturesCfg.page, m.listMissingOneImplFeaturesCfg.err
}

func (m *MockWPTMetricsStorer) ListBaselineStatusCounts(
	_ context.Context,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) (*backend.BaselineStatusMetricsPage, error) {
	m.callCountListBaselineStatusCounts++

	if !startAt.Equal(m.listBaselineStatusCountsCfg.expectedStartAt) ||
		!endAt.Equal(m.listBaselineStatusCountsCfg.expectedEndAt) ||
		pageSize != m.listBaselineStatusCountsCfg.expectedPageSize ||
		!reflect.DeepEqual(pageToken, m.listBaselineStatusCountsCfg.expectedPageToken) {

		m.t.Errorf("Incorrect arguments. Expected: %v, Got: { %s, %s, %d %v }",
			m.listBaselineStatusCountsCfg, startAt, endAt, pageSize, pageToken)
	}

	return m.listBaselineStatusCountsCfg.page, m.listBaselineStatusCountsCfg.err
}

func (m *MockWPTMetricsStorer) CreateUserSavedSearch(
	_ context.Context,
	userID string,
	savedSearch backend.SavedSearch,
) (*backend.SavedSearchResponse, error) {
	m.callCountCreateUserSavedSearch++

	if !reflect.DeepEqual(savedSearch, m.createUserSavedSearchCfg.expectedSavedSearch) ||
		userID != m.createUserSavedSearchCfg.expectedUserID {
		m.t.Errorf("Incorrect arguments. Expected: %v, Got: { %v %s }",
			m.createUserSavedSearchCfg.expectedSavedSearch, savedSearch, userID)
	}

	return m.createUserSavedSearchCfg.output, m.createUserSavedSearchCfg.err
}

func (m *MockWPTMetricsStorer) GetSavedSearch(
	_ context.Context,
	savedSearchID string,
	userID *string) (*backend.SavedSearchResponse, error) {
	m.callCountGetSavedSearch++

	if savedSearchID != m.getSavedSearchCfg.expectedSavedSearchID ||
		!reflect.DeepEqual(userID, m.getSavedSearchCfg.expectedUserID) {
		m.t.Errorf("Incorrect arguments. Expected: { %s %v }, Got: { %s %v }",
			m.getSavedSearchCfg.expectedSavedSearchID, m.getSavedSearchCfg.expectedUserID,
			savedSearchID, userID)
	}

	return m.getSavedSearchCfg.output, m.getSavedSearchCfg.err
}

func (m *MockWPTMetricsStorer) DeleteUserSavedSearch(
	_ context.Context,
	userID string,
	savedSearchID string,
) error {
	m.callCountDeleteUserSavedSearch++

	if userID != m.deleteUserSavedSearchCfg.expectedUserID ||
		savedSearchID != m.deleteUserSavedSearchCfg.expectedSavedSearchID {
		m.t.Errorf("Incorrect arguments. Expected: ( %s %s ), Got: { %s %s }",
			m.deleteUserSavedSearchCfg.expectedUserID, m.deleteUserSavedSearchCfg.expectedSavedSearchID,
			userID, savedSearchID)
	}

	return m.deleteUserSavedSearchCfg.err
}

func (m *MockWPTMetricsStorer) UpdateUserSavedSearch(
	_ context.Context,
	savedSearchID string,
	userID string,
	req *backend.SavedSearchUpdateRequest,
) (*backend.SavedSearchResponse, error) {
	m.callCountUpdateUserSavedSearch++

	if savedSearchID != m.updateUserSavedSearchCfg.expectedSavedSearchID ||
		userID != m.updateUserSavedSearchCfg.expectedUserID ||
		!reflect.DeepEqual(req, m.updateUserSavedSearchCfg.expectedUpdateRequest) {
		m.t.Errorf("Incorrect arguments. Expected: ( %s %s %v ), Got: { %s %s %v}",
			m.updateUserSavedSearchCfg.expectedSavedSearchID,
			m.updateUserSavedSearchCfg.expectedUserID,
			m.updateUserSavedSearchCfg.expectedUpdateRequest,
			savedSearchID,
			userID,
			req)
	}

	return m.updateUserSavedSearchCfg.output, m.updateUserSavedSearchCfg.err
}

func (m *MockWPTMetricsStorer) ListUserSavedSearches(
	_ context.Context,
	userID string,
	pageSize int,
	pageToken *string) (*backend.UserSavedSearchPage, error) {
	m.callCountListUserSavedSearches++

	if userID != m.listUserSavedSearchesCfg.expectedUserID ||
		pageSize != m.listUserSavedSearchesCfg.expectedPageSize ||
		!reflect.DeepEqual(pageToken, m.listUserSavedSearchesCfg.expectedPageToken) {
		m.t.Errorf("Incorrect arguments. Expected: ( %s %d %v ), Got: { %s %d %v }",
			m.listUserSavedSearchesCfg.expectedUserID,
			m.listUserSavedSearchesCfg.expectedPageSize,
			m.listUserSavedSearchesCfg.expectedPageToken,
			userID,
			pageSize,
			pageToken,
		)
	}

	return m.listUserSavedSearchesCfg.output, m.listUserSavedSearchesCfg.err
}

func TestGetPageSizeOrDefault(t *testing.T) {
	testCases := []struct {
		name          string
		inputPageSize *int
		expected      int
	}{
		{"Nil input", nil, 100},
		{"Input below min", valuePtr[int](0), 100},
		{"Valid input (below max)", valuePtr[int](25), 25},
		{"Input above max", valuePtr[int](100), 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getPageSizeOrDefault(tc.inputPageSize)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}

// nolint: gochecknoglobals
var (
	inputPageToken = valuePtr[string]("input-token")
	nextPageToken  = valuePtr[string]("next-page-token")
	badPageToken   = valuePtr[string]("")
	errTest        = errors.New("test error")
)

func testJSONResponse(statusCode int, body string) *http.Response {
	// nolint:exhaustruct // WONTFIX - only for test purposes
	return &http.Response{
		StatusCode: statusCode,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}

func assertStatusCode(t *testing.T, actual, expected int) {
	if actual != expected {
		t.Errorf("expected status code %d. received %d", expected, actual)
	}
}

func assertHeaders(t *testing.T, actual, expected http.Header) {
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("expected headers %+v. received %+v", expected, actual)
	}
}

func assertResponseBody(t *testing.T, actual, expected io.Reader) {
	if actual == nil && expected == nil {
		// Both nil, no need to compare
		return
	}

	if actual == nil && expected != nil {
		expectedBody, _ := io.ReadAll(expected)
		t.Errorf("expected a body. received no response body (%s)", string(expectedBody))

		return
	}

	if actual != nil && expected == nil {
		actualBody, _ := io.ReadAll(actual)
		t.Errorf("expected no body. received response body (%s)", string(actualBody))

		return
	}

	actualBody, err := io.ReadAll(actual)
	if err != nil {
		t.Fatal("failed to read actual body")
	}

	expectedBody, err := io.ReadAll(expected)
	if err != nil {
		t.Fatal("failed to read expected body")
	}

	if string(actualBody) == "" && string(expectedBody) == "" {
		// Both empty, no need to compare
		return
	}

	compareJSONBodies(t, actualBody, expectedBody)
}

func compareJSONBodies(t *testing.T, actualBody, expectedBody []byte) {
	var actualObj, expectedObj interface{}
	err := json.Unmarshal(actualBody, &actualObj)
	if err != nil {
		t.Fatal("failed to parse json from actual response")
	}
	err = json.Unmarshal(expectedBody, &expectedObj)
	if err != nil {
		t.Fatal("failed to parse json from expected response")
	}

	if !reflect.DeepEqual(actualObj, expectedObj) {
		t.Errorf("expected body %+v. received %+v", string(expectedBody), string(actualBody))
	}
}

func assertMocksExpectations(t *testing.T, expectedCallCount, actualCallCount int, methodName string,
	mockCacher *MockRawBytesDataCacher) {
	if expectedCallCount != actualCallCount {
		t.Errorf("expected %s to be called %d times. it was called %d times",
			methodName, expectedCallCount, actualCallCount)
	}
	if mockCacher != nil {
		mockCacher.AssertExpectations()
	}
}

type testServerConfig struct {
	authMiddleware func(http.Handler) http.Handler
}

type testServerOption func(*testServerConfig)

func withAuthMiddleware(middleware func(http.Handler) http.Handler) testServerOption {
	return func(c *testServerConfig) {
		c.authMiddleware = middleware
	}
}

func noopMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		next.ServeHTTP(w, req)
	})
}

func assertTestServerRequest(t *testing.T, testServer *Server, req *http.Request, expectedResponse *http.Response,
	options ...testServerOption) {
	testServerConfig := &testServerConfig{
		authMiddleware: noopMiddleware,
	}

	for _, option := range options {
		option(testServerConfig)
	}

	srv := createOpenAPIServerServer("", testServer, []func(http.Handler) http.Handler{
		recoveryMiddleware}, testServerConfig.authMiddleware)

	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)

	resp := w.Result()

	assertStatusCode(t, resp.StatusCode, expectedResponse.StatusCode)
	assertHeaders(t, resp.Header, expectedResponse.Header)
	assertResponseBody(t, resp.Body, expectedResponse.Body)
}

type mockServerInterface struct {
	t                 *testing.T
	expectedUserInCtx *auth.User
	callCount         int
}

// ListAggregatedBaselineStatusCounts implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) ListAggregatedBaselineStatusCounts(
	ctx context.Context, _ backend.ListAggregatedBaselineStatusCountsRequestObject) (
	backend.ListAggregatedBaselineStatusCountsResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// CreateSavedSearch implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) CreateSavedSearch(ctx context.Context, _ backend.CreateSavedSearchRequestObject) (
	backend.CreateSavedSearchResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// GetFeature implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) GetFeature(ctx context.Context, _ backend.GetFeatureRequestObject) (
	backend.GetFeatureResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// GetFeatureMetadata implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) GetFeatureMetadata(ctx context.Context, _ backend.GetFeatureMetadataRequestObject) (
	backend.GetFeatureMetadataResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// GetSavedSearch implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) GetSavedSearch(ctx context.Context, _ backend.GetSavedSearchRequestObject) (
	backend.GetSavedSearchResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// ListAggregatedFeatureSupport implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) ListAggregatedFeatureSupport(ctx context.Context,
	_ backend.ListAggregatedFeatureSupportRequestObject) (
	backend.ListAggregatedFeatureSupportResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// ListAggregatedWPTMetrics implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) ListAggregatedWPTMetrics(ctx context.Context,
	_ backend.ListAggregatedWPTMetricsRequestObject) (backend.ListAggregatedWPTMetricsResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// ListChromeDailyUsageStats implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) ListChromeDailyUsageStats(ctx context.Context,
	_ backend.ListChromeDailyUsageStatsRequestObject) (
	backend.ListChromeDailyUsageStatsResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// ListFeatureWPTMetrics implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) ListFeatureWPTMetrics(ctx context.Context,
	_ backend.ListFeatureWPTMetricsRequestObject) (backend.ListFeatureWPTMetricsResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// ListFeatures implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) ListFeatures(ctx context.Context,
	_ backend.ListFeaturesRequestObject) (backend.ListFeaturesResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// ListMissingOneImplementationCounts implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) ListMissingOneImplementationCounts(ctx context.Context,
	_ backend.ListMissingOneImplementationCountsRequestObject) (
	backend.ListMissingOneImplementationCountsResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// ListUserSavedSearches implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) ListUserSavedSearches(ctx context.Context,
	_ backend.ListUserSavedSearchesRequestObject) (backend.ListUserSavedSearchesResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// PutUserSavedSearchBookmark implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) PutUserSavedSearchBookmark(ctx context.Context,
	_ backend.PutUserSavedSearchBookmarkRequestObject) (backend.PutUserSavedSearchBookmarkResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// RemoveSavedSearch implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) RemoveSavedSearch(ctx context.Context,
	_ backend.RemoveSavedSearchRequestObject) (backend.RemoveSavedSearchResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// RemoveUserSavedSearchBookmark implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) RemoveUserSavedSearchBookmark(ctx context.Context,
	_ backend.RemoveUserSavedSearchBookmarkRequestObject) (
	backend.RemoveUserSavedSearchBookmarkResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// UpdateSavedSearch implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) UpdateSavedSearch(ctx context.Context,
	_ backend.UpdateSavedSearchRequestObject) (backend.UpdateSavedSearchResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

// ListMissingOneImplementationFeatures implements backend.StrictServerInterface.
// nolint: ireturn // WONTFIX - generated method signature
func (m *mockServerInterface) ListMissingOneImplementationFeatures(ctx context.Context,
	_ backend.ListMissingOneImplementationFeaturesRequestObject) (
	backend.ListMissingOneImplementationFeaturesResponseObject, error) {
	assertUserInCtx(ctx, m.t, m.expectedUserInCtx)
	m.callCount++
	panic("unimplemented")
}

func (m *mockServerInterface) assertCallCount(expectedCallCount int) {
	if m.callCount != expectedCallCount {
		m.t.Errorf("expected mock server to be used %d times. only used %d times", expectedCallCount, m.callCount)
	}
}

func assertUserInCtx(ctx context.Context, t *testing.T, expectedUser *auth.User) {
	actualUser, _ := httpmiddlewares.AuthenticatedUserFromContext(ctx)
	if !reflect.DeepEqual(expectedUser, actualUser) {
		t.Errorf("expected user %+v in context. received %+v", expectedUser, actualUser)
	}
}

func submitRequest(t *testing.T, url string, method string) {
	req, err := http.NewRequestWithContext(context.Background(), method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenericErrorFn(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		err          error
		expectedBody string
	}{
		{
			name:         "With error",
			statusCode:   http.StatusInternalServerError,
			err:          errors.New("internal error"),
			expectedBody: `{"code":500,"message":"internal error"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			GenericErrorFn(context.Background(), tc.statusCode, rr, tc.err)

			// Check the status code
			if rr.Code != tc.statusCode {
				t.Errorf("Expected status code %d, got %d", tc.statusCode, rr.Code)
			}

			// Check the response body
			actualBody := strings.TrimSpace(rr.Body.String())
			if actualBody != tc.expectedBody {
				t.Errorf("Expected body '%s', got '%s'", tc.expectedBody, actualBody)
			}
		})
	}
}
