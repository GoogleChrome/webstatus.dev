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

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestListFeatureWPTMetrics(t *testing.T) {
	testCases := []struct {
		name               string
		mockConfig         *MockListMetricsForFeatureIDBrowserAndChannelConfig
		expectedCallCount  int // For the mock method
		expectedCacheCalls []*ExpectedCacheCall
		expectedGetCalls   []*ExpectedGetCall
		request            *http.Request
		expectedResponse   *http.Response
	}{
		{
			name: "Success Case - no optional params - use defaults",
			mockConfig: &MockListMetricsForFeatureIDBrowserAndChannelConfig{
				expectedFeatureID: "feature1",
				expectedBrowser:   "chrome",
				expectedChannel:   "experimental",
				expectedMetric:    backend.TestCounts,
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  100,
				expectedPageToken: nil,
				pageToken:         nil,
				err:               nil,

				data: []backend.WPTRunMetric{
					{
						RunTimestamp:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						TestPassCount:   valuePtr[int64](2),
						TotalTestsCount: valuePtr[int64](2),
					},
				},
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `listFeatureWPTMetrics-{"feature_id":"feature1","browser":"chrome",` +
						`"channel":"experimental","metric_view":"test_counts",` +
						`"Params":{"startAt":"2000-01-01","endAt":"2000-01-10"}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: []*ExpectedCacheCall{
				{
					Key: `listFeatureWPTMetrics-{"feature_id":"feature1","browser":"chrome",` +
						`"channel":"experimental","metric_view":"test_counts",` +
						`"Params":{"startAt":"2000-01-01","endAt":"2000-01-10"}}`,
					Value: []byte(
						`{"data":[{"run_timestamp":"2000-01-01T00:00:00Z","test_pass_count":2,` +
							`"total_tests_count":2}],"metadata":{}}`,
					),
					CacheCfg: getDefaultCacheConfig(),
				},
			},
			expectedCallCount: 1,
			expectedResponse: testJSONResponse(200, `
{
	"data":[
		{
			"run_timestamp":"2000-01-01T00:00:00Z",
			"test_pass_count":2,
			"total_tests_count":2
		}
	],
	"metadata":{}
}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/features/feature1/stats/wpt/browsers/chrome/channels/experimental/test_counts"+
					"?startAt=2000-01-01&endAt=2000-01-10", nil),
		},
		{
			name:       "Success Case - no optional params - use defaults - cached",
			mockConfig: nil,
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `listFeatureWPTMetrics-{"feature_id":"feature1","browser":"chrome",` +
						`"channel":"experimental","metric_view":"test_counts",` +
						`"Params":{"startAt":"2000-01-01","endAt":"2000-01-10"}}`,
					Value: []byte(
						`{"data":[{"run_timestamp":"2000-01-01T00:00:00Z","test_pass_count":2,` +
							`"total_tests_count":2}],"metadata":{}}`,
					),
					Err: nil,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  0,
			expectedResponse: testJSONResponse(200, `
{
	"data":[
		{
			"run_timestamp":"2000-01-01T00:00:00Z",
			"test_pass_count":2,
			"total_tests_count":2
		}
	],
	"metadata":{}
}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/features/feature1/stats/wpt/browsers/chrome/channels/experimental/test_counts"+
					"?startAt=2000-01-01&endAt=2000-01-10", nil),
		},
		{
			name: "Success Case - include optional params",
			mockConfig: &MockListMetricsForFeatureIDBrowserAndChannelConfig{
				expectedFeatureID: "feature1",
				expectedBrowser:   "chrome",
				expectedChannel:   "experimental",
				expectedMetric:    backend.TestCounts,
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  50,
				expectedPageToken: inputPageToken,
				err:               nil,

				data: []backend.WPTRunMetric{
					{
						RunTimestamp:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						TestPassCount:   valuePtr[int64](2),
						TotalTestsCount: valuePtr[int64](2),
					},
				},
				pageToken: nextPageToken,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `listFeatureWPTMetrics-{"feature_id":"feature1","browser":"chrome",` +
						`"channel":"experimental","metric_view":"test_counts","Params":{"startAt":"2000-01-01",` +
						`"endAt":"2000-01-10","page_token":"input-token","page_size":50}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: []*ExpectedCacheCall{
				{
					Key: `listFeatureWPTMetrics-{"feature_id":"feature1","browser":"chrome",` +
						`"channel":"experimental","metric_view":"test_counts","Params":{"startAt":"2000-01-01",` +
						`"endAt":"2000-01-10","page_token":"input-token","page_size":50}}`,
					Value: []byte(
						`{"data":[{"run_timestamp":"2000-01-01T00:00:00Z","test_pass_count":2,` +
							`"total_tests_count":2}],"metadata":{"next_page_token":"next-page-token"}}`,
					),
					CacheCfg: getDefaultCacheConfig(),
				},
			},
			expectedCallCount: 1,
			expectedResponse: testJSONResponse(200, `
{
	"data":[
		{
			"run_timestamp":"2000-01-01T00:00:00Z",
			"test_pass_count":2,
			"total_tests_count":2
		}
	],
	"metadata":{
		"next_page_token":"next-page-token"
	}
}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/features/feature1/stats/wpt/browsers/chrome/channels/experimental/test_counts"+
					"?startAt=2000-01-01&endAt=2000-01-10&"+
					"page_size=50&page_token="+*inputPageToken, nil),
		},
		{
			name:       "Success Case - include optional params - cached",
			mockConfig: nil,
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `listFeatureWPTMetrics-{"feature_id":"feature1","browser":"chrome",` +
						`"channel":"experimental","metric_view":"test_counts","Params":{"startAt":"2000-01-01",` +
						`"endAt":"2000-01-10","page_token":"input-token","page_size":50}}`,
					Value: []byte(
						`{"data":[{"run_timestamp":"2000-01-01T00:00:00Z","test_pass_count":2,` +
							`"total_tests_count":2}],"metadata":{"next_page_token":"next-page-token"}}`,
					),
					Err: nil,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  0,
			expectedResponse: testJSONResponse(200, `
{
	"data":[
		{
			"run_timestamp":"2000-01-01T00:00:00Z",
			"test_pass_count":2,
			"total_tests_count":2
		}
	],
	"metadata":{
		"next_page_token":"next-page-token"
	}
}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/features/feature1/stats/wpt/browsers/chrome/channels/experimental/test_counts"+
					"?startAt=2000-01-01&endAt=2000-01-10&"+
					"page_size=50&page_token="+*inputPageToken, nil),
		},
		{
			name: "500 case",
			mockConfig: &MockListMetricsForFeatureIDBrowserAndChannelConfig{
				expectedFeatureID: "feature1",
				expectedBrowser:   "chrome",
				expectedChannel:   "experimental",
				expectedMetric:    backend.TestCounts,
				expectedPageToken: nil,
				data:              nil,
				pageToken:         nil,
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  100,
				err:               errTest,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `listFeatureWPTMetrics-{"feature_id":"feature1","browser":"chrome",` +
						`"channel":"experimental","metric_view":"test_counts",` +
						`"Params":{"startAt":"2000-01-01","endAt":"2000-01-10"}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  1,
			expectedResponse:   testJSONResponse(500, `{"code":500,"message":"unable to get feature metrics"}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/features/feature1/stats/wpt/browsers/chrome/channels/experimental/test_counts"+
					"?startAt=2000-01-01&endAt=2000-01-10", nil),
		},
		{
			name: "400 case - invalid page token",
			mockConfig: &MockListMetricsForFeatureIDBrowserAndChannelConfig{
				expectedFeatureID: "feature1",
				expectedBrowser:   "chrome",
				expectedChannel:   "experimental",
				expectedMetric:    backend.TestCounts,
				expectedPageToken: badPageToken,
				data:              nil,
				pageToken:         nil,
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  100,
				err:               backendtypes.ErrInvalidPageToken,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `listFeatureWPTMetrics-{"feature_id":"feature1","browser":"chrome",` +
						`"channel":"experimental","metric_view":"test_counts",` +
						`"Params":{"startAt":"2000-01-01","endAt":"2000-01-10","page_token":""}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  1,
			expectedResponse:   testJSONResponse(400, `{"code":400,"message":"invalid page token"}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/features/feature1/stats/wpt/browsers/chrome/channels/experimental/test_counts"+
					"?startAt=2000-01-01&endAt=2000-01-10&page_token="+*badPageToken, nil),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				featureCfg: tc.mockConfig,
				t:          t,
			}
			mockCacher := NewMockRawBytesDataCacher(t, tc.expectedCacheCalls, tc.expectedGetCalls)
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil,
				operationResponseCaches: initOperationResponseCaches(mockCacher, getTestRouteCacheOptions()),
				baseURL:                 getTestBaseURL(t)}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse)
			assertMocksExpectations(t, tc.expectedCallCount, mockStorer.callCountListMetricsForFeatureIDBrowserAndChannel,
				"ListMetricsForFeatureIDBrowserAndChannel", mockCacher)
		})
	}
}
