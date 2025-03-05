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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestGetFeature(t *testing.T) {
	testCases := []struct {
		name               string
		mockConfig         *MockGetFeatureByIDConfig
		expectedCallCount  int // For the mock method
		expectedCacheCalls []*ExpectedCacheCall
		expectedGetCalls   []*ExpectedGetCall
		request            *http.Request
		expectedResponse   *http.Response
	}{
		// nolint:dupl // WONTFIX - being explicit for short list of tests.
		{
			name: "Success Case - no optional params - use defaults",
			mockConfig: &MockGetFeatureByIDConfig{
				expectedFeatureID:     "feature1",
				expectedWPTMetricView: backend.TestCounts,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
				},
				data: &backend.Feature{
					Baseline: &backend.BaselineInfo{
						Status: valuePtr(backend.Widely),
						LowDate: valuePtr(
							openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
						),
						HighDate: valuePtr(
							openapi_types.Date{Time: time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)},
						),
					},
					BrowserImplementations: &map[string]backend.BrowserImplementation{
						"chrome": {
							Status:  valuePtr(backend.Available),
							Date:    &openapi_types.Date{Time: time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC)},
							Version: valuePtr("100"),
						},
					},
					FeatureId: "feature1",
					Name:      "feature 1",
					Spec:      nil,
					Usage:     nil,
					Wpt:       nil,
				},
				err: nil,
			},
			expectedCallCount: 1,
			request:           httptest.NewRequest(http.MethodGet, "/v1/features/feature1", nil),
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `getFeature-{"feature_id":"feature1","Params":{}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: []*ExpectedCacheCall{
				{
					Key: `getFeature-{"feature_id":"feature1","Params":{}}`,
					Value: []byte(
						`{"baseline":{"high_date":"2001-01-01","low_date":"2000-01-01","status":"widely"},` +
							`"browser_implementations":{"chrome":{"date":"1999-01-01",` +
							`"status":"available","version":"100"}},"feature_id":"feature1","name":"feature 1"}`,
					),
					CacheCfg: getDefaultCacheConfig(),
				},
			},
			expectedResponse: testJSONResponse(200, `
			{
				"baseline":{
				   "high_date":"2001-01-01",
				   "low_date":"2000-01-01",
				   "status":"widely"
				},
				"browser_implementations":{
				   "chrome":{
					  "date":"1999-01-01",
					  "status":"available",
					  "version":"100"
				   }
				},
				"feature_id":"feature1",
				"name":"feature 1"
			 }`),
		},
		// nolint:dupl // WONTFIX - being explicit for short list of tests.
		{
			name:              "Success Case - no optional params - use defaults - cached",
			mockConfig:        nil,
			expectedCallCount: 0,
			request:           httptest.NewRequest(http.MethodGet, "/v1/features/feature1", nil),
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `getFeature-{"feature_id":"feature1","Params":{}}`,
					Value: []byte(
						`{"baseline":{"high_date":"2001-01-01","low_date":"2000-01-01","status":"widely"},` +
							`"browser_implementations":{"chrome":{"date":"1999-01-01",` +
							`"status":"available","version":"100"}},"feature_id":"feature1","name":"feature 1"}`,
					),
					Err: nil,
				},
			},
			expectedCacheCalls: nil,
			expectedResponse: testJSONResponse(200, `
					{
						"baseline":{
						   "high_date":"2001-01-01",
						   "low_date":"2000-01-01",
						   "status":"widely"
						},
						"browser_implementations":{
						   "chrome":{
							  "date":"1999-01-01",
							  "status":"available",
							  "version":"100"
						   }
						},
						"feature_id":"feature1",
						"name":"feature 1"
					 }`),
		},
		// nolint:dupl // WONTFIX - being explicit for short list of tests.
		{
			name: "Success Case - with optional params",
			mockConfig: &MockGetFeatureByIDConfig{
				expectedFeatureID:     "feature1",
				expectedWPTMetricView: backend.SubtestCounts,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
				},
				data: &backend.Feature{
					Baseline: &backend.BaselineInfo{
						Status: valuePtr(backend.Widely),
						LowDate: valuePtr(
							openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
						),
						HighDate: valuePtr(
							openapi_types.Date{Time: time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)},
						),
					},
					BrowserImplementations: &map[string]backend.BrowserImplementation{
						"chrome": {
							Status:  valuePtr(backend.Available),
							Date:    &openapi_types.Date{Time: time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC)},
							Version: valuePtr("100"),
						},
					},
					FeatureId: "feature1",
					Name:      "feature 1",
					Spec:      nil,
					Usage:     nil,
					Wpt:       nil,
				},
				err: nil,
			},
			expectedCallCount: 1,
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `getFeature-{"feature_id":"feature1","Params":{"wpt_metric_view":"subtest_counts"}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: []*ExpectedCacheCall{
				{
					Key: `getFeature-{"feature_id":"feature1","Params":{"wpt_metric_view":"subtest_counts"}}`,
					Value: []byte(
						`{"baseline":{"high_date":"2001-01-01","low_date":"2000-01-01","status":"widely"},` +
							`"browser_implementations":` +
							`{"chrome":{"date":"1999-01-01","status":"available","version":"100"}},` +
							`"feature_id":"feature1","name":"feature 1"}`,
					),
					CacheCfg: getDefaultCacheConfig(),
				},
			},
			request: httptest.NewRequest(http.MethodGet, "/v1/features/feature1?wpt_metric_view=subtest_counts", nil),
			expectedResponse: testJSONResponse(200, `
{
	"baseline":{
		"high_date":"2001-01-01",
		"low_date":"2000-01-01",
		"status":"widely"
	},
	"browser_implementations":{
		"chrome":{
			"date":"1999-01-01",
			"status":"available",
			"version":"100"
		}
	},
	"feature_id":"feature1",
	"name":"feature 1"
}`,
			),
		},
		// nolint:dupl // WONTFIX - being explicit for short list of tests.
		{
			name:              "Success Case - with optional params - cached",
			mockConfig:        nil,
			expectedCallCount: 0,
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `getFeature-{"feature_id":"feature1","Params":{"wpt_metric_view":"subtest_counts"}}`,
					Value: []byte(
						`{"baseline":{"high_date":"2001-01-01","low_date":"2000-01-01","status":"widely"},` +
							`"browser_implementations":` +
							`{"chrome":{"date":"1999-01-01","status":"available","version":"100"}},` +
							`"feature_id":"feature1","name":"feature 1"}`,
					),
					Err: nil,
				},
			},
			expectedCacheCalls: nil,
			request:            httptest.NewRequest(http.MethodGet, "/v1/features/feature1?wpt_metric_view=subtest_counts", nil),
			expectedResponse: testJSONResponse(200, `
{
	"baseline":{
		"high_date":"2001-01-01",
		"low_date":"2000-01-01",
		"status":"widely"
	},
	"browser_implementations":{
		"chrome":{
			"date":"1999-01-01",
			"status":"available",
			"version":"100"
		}
	},
	"feature_id":"feature1",
	"name":"feature 1"
}`,
			),
		},
		{
			name: "404",
			mockConfig: &MockGetFeatureByIDConfig{
				expectedFeatureID:     "feature1",
				expectedWPTMetricView: backend.TestCounts,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
				},
				data: nil,
				err:  gcpspanner.ErrQueryReturnedNoResults,
			},
			expectedCallCount: 1,
			request:           httptest.NewRequest(http.MethodGet, "/v1/features/feature1", nil),
			expectedResponse:  testJSONResponse(404, `{"code":404,"message":"feature id feature1 is not found"}`),
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `getFeature-{"feature_id":"feature1","Params":{}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: nil,
		},
		{
			name: "500",
			mockConfig: &MockGetFeatureByIDConfig{
				expectedFeatureID:     "feature1",
				expectedWPTMetricView: backend.TestCounts,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
				},
				data: nil,
				err:  errTest,
			},
			expectedCallCount: 1,
			request:           httptest.NewRequest(http.MethodGet, "/v1/features/feature1", nil),
			expectedResponse:  testJSONResponse(500, `{"code":500,"message":"unable to get feature"}`),
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `getFeature-{"feature_id":"feature1","Params":{}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				getFeatureByIDConfig: tc.mockConfig,
				t:                    t,
			}
			mockCacher := NewMockRawBytesDataCacher(t, tc.expectedCacheCalls, tc.expectedGetCalls)
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil,
				operationResponseCaches: initOperationResponseCaches(mockCacher, getTestRouteCacheOptions())}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse)
			assertMocksExpectations(t, tc.expectedCallCount, mockStorer.callCountGetFeature,
				"GetFeature", mockCacher)
		})
	}
}
