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
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestListAggregatedWPTMetrics(t *testing.T) {
	testCases := []struct {
		name               string
		mockConfig         *MockListMetricsOverTimeWithAggregatedTotalsConfig
		expectedCallCount  int // For the mock method
		expectedCacheCalls []*ExpectedCacheCall
		expectedGetCalls   []*ExpectedGetCall
		request            *http.Request
		expectedResponse   *http.Response
	}{
		{
			name: "Success Case - no optional params - use defaults",
			mockConfig: &MockListMetricsOverTimeWithAggregatedTotalsConfig{
				expectedFeatureIDs: []string{},
				expectedBrowser:    "chrome",
				expectedChannel:    "experimental",
				expectedMetric:     backend.SubtestCounts,
				expectedStartAt:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:      time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:   100,
				expectedPageToken:  nil,
				pageToken:          nil,
				err:                nil,
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
					Key: `listAggregatedWPTMetrics-{"browser":"chrome","channel":"experimental",` +
						`"metric_view":"subtest_counts","Params":{"startAt":"2000-01-01","endAt":"2000-01-10"}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: []*ExpectedCacheCall{
				{
					Key: `listAggregatedWPTMetrics-{"browser":"chrome","channel":"experimental",` +
						`"metric_view":"subtest_counts","Params":{"startAt":"2000-01-01","endAt":"2000-01-10"}}`,
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
	"metadata":{

	}
}
			`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/wpt/browsers/chrome/channels/experimental/subtest_counts"+
					"?startAt=2000-01-01&endAt=2000-01-10", nil),
		},
		{
			name:       "Success Case - no optional params - use defaults - cached",
			mockConfig: nil,
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `listAggregatedWPTMetrics-{"browser":"chrome","channel":"experimental",` +
						`"metric_view":"subtest_counts","Params":{"startAt":"2000-01-01","endAt":"2000-01-10"}}`,
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
	"metadata":{

	}
}
			`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/wpt/browsers/chrome/channels/experimental/subtest_counts"+
					"?startAt=2000-01-01&endAt=2000-01-10", nil),
		},
		{
			name: "Success Case - include optional params",
			mockConfig: &MockListMetricsOverTimeWithAggregatedTotalsConfig{
				expectedFeatureIDs: []string{"feature1", "feature2"},
				expectedBrowser:    "chrome",
				expectedChannel:    "experimental",
				expectedMetric:     backend.SubtestCounts,
				expectedStartAt:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:      time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:   50,
				expectedPageToken:  inputPageToken,
				err:                nil,
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
					Key: `listAggregatedWPTMetrics-{"browser":"chrome","channel":"experimental",` +
						`"metric_view":"subtest_counts","Params":{"startAt":"2000-01-01","endAt":"2000-01-10",` +
						`"page_token":"input-token","page_size":50,"featureId":["feature1","feature2"]}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: []*ExpectedCacheCall{
				{
					Key: `listAggregatedWPTMetrics-{"browser":"chrome","channel":"experimental",` +
						`"metric_view":"subtest_counts","Params":{"startAt":"2000-01-01","endAt":"2000-01-10",` +
						`"page_token":"input-token","page_size":50,"featureId":["feature1","feature2"]}}`,
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
}
			`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/wpt/browsers/chrome/channels/experimental/subtest_counts"+
					"?startAt=2000-01-01&endAt=2000-01-10&page_size=50&"+
					"featureId=feature1&featureId=feature2&"+
					"page_token="+*inputPageToken, nil),
		},
		{
			name:       "Success Case - include optional params - cached",
			mockConfig: nil,
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `listAggregatedWPTMetrics-{"browser":"chrome","channel":"experimental",` +
						`"metric_view":"subtest_counts","Params":{"startAt":"2000-01-01","endAt":"2000-01-10",` +
						`"page_token":"input-token","page_size":50,"featureId":["feature1","feature2"]}}`,
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
}
			`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/wpt/browsers/chrome/channels/experimental/subtest_counts"+
					"?startAt=2000-01-01&endAt=2000-01-10&page_size=50&"+
					"featureId=feature1&featureId=feature2&"+
					"page_token="+*inputPageToken, nil),
		},
		{
			name: "500 case",
			mockConfig: &MockListMetricsOverTimeWithAggregatedTotalsConfig{
				expectedFeatureIDs: []string{},
				expectedBrowser:    "chrome",
				expectedChannel:    "experimental",
				expectedMetric:     backend.SubtestCounts,
				expectedStartAt:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:      time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:   100,
				expectedPageToken:  nil,
				data:               nil,
				pageToken:          nil,
				err:                errTest,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `listAggregatedWPTMetrics-{"browser":"chrome","channel":"experimental",` +
						`"metric_view":"subtest_counts","Params":{"startAt":"2000-01-01","endAt":"2000-01-10"}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  1,
			expectedResponse:   testJSONResponse(500, `{"code":500,"message":"unable to get aggregated metrics"}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/wpt/browsers/chrome/channels/experimental/subtest_counts"+
					"?startAt=2000-01-01&endAt=2000-01-10", nil),
		},
		{
			name: "400 case - invalid page token",
			mockConfig: &MockListMetricsOverTimeWithAggregatedTotalsConfig{
				expectedFeatureIDs: []string{},
				expectedBrowser:    "chrome",
				expectedChannel:    "experimental",
				expectedMetric:     backend.SubtestCounts,
				expectedStartAt:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:      time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:   100,
				expectedPageToken:  badPageToken,
				data:               nil,
				pageToken:          nil,
				err:                backendtypes.ErrInvalidPageToken,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `listAggregatedWPTMetrics-{"browser":"chrome","channel":"experimental",` +
						`"metric_view":"subtest_counts","Params":{"startAt":"2000-01-01","endAt":"2000-01-10",` +
						`"page_token":""}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  1,
			expectedResponse:   testJSONResponse(400, `{"code":400,"message":"invalid page token"}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/wpt/browsers/chrome/channels/experimental/subtest_counts"+
					"?startAt=2000-01-01&endAt=2000-01-10&page_token="+*badPageToken, nil),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				aggregateCfg: tc.mockConfig,
				t:            t,
			}
			mockCacher := NewMockRawBytesDataCacher(t, tc.expectedCacheCalls, tc.expectedGetCalls)
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil,
				operationResponseCaches: initOperationResponseCaches(mockCacher, getTestRouteCacheOptions())}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse)
			assertMocksExpectations(t, tc.expectedCallCount, mockStorer.callCountListMetricsOverTimeWithAggregatedTotals,
				"ListMetricsOverTimeWithAggregatedTotals", mockCacher)
		})
	}
}
