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

func TestListMissingOneImplementationCounts(t *testing.T) {
	testCases := []struct {
		name               string
		mockConfig         *MockListMissingOneImplCountsConfig
		expectedCallCount  int // For the mock method
		expectedCacheCalls []*ExpectedCacheCall
		expectedGetCalls   []*ExpectedGetCall
		request            *http.Request
		expectedResponse   *http.Response
	}{
		{
			name: "Success Case - no optional params - use defaults",
			mockConfig: &MockListMissingOneImplCountsConfig{
				expectedTargetBrowser:       "chrome",
				expectedTargetMobileBrowser: nil,
				expectedOtherBrowsers:       []string{"edge", "firefox", "safari"},
				expectedStartAt:             time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:               time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:            100,
				expectedPageToken:           nil,
				pageToken:                   nil,
				err:                         nil,
				page: &backend.BrowserReleaseFeatureMetricsPage{
					Metadata: &backend.PageMetadata{
						NextPageToken: nil,
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
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `ListMissingOneImplementationCounts-{"browser":"chrome","Params":{"startAt":"2000-01-01",` +
						`"endAt":"2000-01-10","browser":["edge","firefox","safari"]}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: []*ExpectedCacheCall{
				{
					Key: `ListMissingOneImplementationCounts-{"browser":"chrome","Params":{"startAt":"2000-01-01",` +
						`"endAt":"2000-01-10","browser":["edge","firefox","safari"]}}`,
					Value: []byte(
						`{"data":[{"count":10,"timestamp":"2000-01-10T00:00:00Z"},` +
							`{"count":9,"timestamp":"2000-01-09T00:00:00Z"}],"metadata":{}}`,
					),
					CacheCfg: getTestAggregatedCacheConfig(),
				},
			},
			expectedCallCount: 1,
			expectedResponse: testJSONResponse(200, `
{
	"data":[
		{
			"count":10,
			"timestamp":"2000-01-10T00:00:00Z"
		},
		{
			"count":9,
			"timestamp":"2000-01-09T00:00:00Z"
		}
	],
	"metadata":{

	}
}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/features/browsers/chrome/missing_one_implementation_counts?"+
					"browser=edge&browser=firefox&browser=safari&"+
					"startAt=2000-01-01&endAt=2000-01-10", nil),
		},
		{
			name:       "Success Case - no optional params - use defaults - cached",
			mockConfig: nil,
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `ListMissingOneImplementationCounts-{"browser":"chrome","Params":{"startAt":"2000-01-01",` +
						`"endAt":"2000-01-10","browser":["edge","firefox","safari"]}}`,
					Value: []byte(
						`{"data":[{"count":10,"timestamp":"2000-01-10T00:00:00Z"},` +
							`{"count":9,"timestamp":"2000-01-09T00:00:00Z"}],"metadata":{}}`,
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
			"count":10,
			"timestamp":"2000-01-10T00:00:00Z"
		},
		{
			"count":9,
			"timestamp":"2000-01-09T00:00:00Z"
		}
	],
	"metadata":{

	}
}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/features/browsers/chrome/missing_one_implementation_counts?"+
					"browser=edge&browser=firefox&browser=safari&"+
					"startAt=2000-01-01&endAt=2000-01-10", nil),
		},
		{
			name: "Success Case - include optional params",
			mockConfig: &MockListMissingOneImplCountsConfig{
				expectedTargetBrowser:       "chrome",
				expectedTargetMobileBrowser: valuePtr("chrome_android"),
				expectedOtherBrowsers:       []string{"firefox", "firefox_android", "safari", "safari_ios"},
				expectedStartAt:             time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:               time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:            50,
				expectedPageToken:           inputPageToken,
				err:                         nil,
				page: &backend.BrowserReleaseFeatureMetricsPage{
					Metadata: &backend.PageMetadata{
						NextPageToken: nextPageToken,
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
				pageToken: nextPageToken,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `ListMissingOneImplementationCounts-{"browser":"chrome","Params":{"startAt":"2000-01-01",` +
						`"endAt":"2000-01-10","page_token":"input-token",` +
						`"page_size":50,"include_baseline_mobile_browsers":true,"browser":["firefox","safari"]}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: []*ExpectedCacheCall{
				{
					Key: `ListMissingOneImplementationCounts-{"browser":"chrome","Params":{"startAt":"2000-01-01",` +
						`"endAt":"2000-01-10","page_token":"input-token",` +
						`"page_size":50,"include_baseline_mobile_browsers":true,"browser":["firefox","safari"]}}`,
					Value: []byte(
						`{"data":[{"count":10,"timestamp":"2000-01-10T00:00:00Z"},` +
							`{"count":9,"timestamp":"2000-01-09T00:00:00Z"}],` +
							`"metadata":{"next_page_token":"next-page-token"}}`,
					),
					CacheCfg: getTestAggregatedCacheConfig(),
				},
			},
			expectedCallCount: 1,
			expectedResponse: testJSONResponse(200, `
{
	"data":[
		{
			"count":10,
			"timestamp":"2000-01-10T00:00:00Z"
		},
		{
			"count":9,
			"timestamp":"2000-01-09T00:00:00Z"
		}
	],
	"metadata":{
		"next_page_token":"next-page-token"
	}
}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/features/browsers/chrome/missing_one_implementation_counts?"+
					"browser=firefox&browser=safari&include_baseline_mobile_browsers=true&"+
					"startAt=2000-01-01&endAt=2000-01-10&page_size=50&page_token="+*inputPageToken, nil),
		},
		{
			name:       "Success Case - include optional params - cached",
			mockConfig: nil,
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `ListMissingOneImplementationCounts-{"browser":"chrome","Params":{"startAt":"2000-01-01",` +
						`"endAt":"2000-01-10","page_token":"input-token",` +
						`"page_size":50,"browser":["edge","firefox","safari"]}}`,
					Value: []byte(
						`{"data":[{"count":10,"timestamp":"2000-01-10T00:00:00Z"},` +
							`{"count":9,"timestamp":"2000-01-09T00:00:00Z"}],` +
							`"metadata":{"next_page_token":"next-page-token"}}`,
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
			"count":10,
			"timestamp":"2000-01-10T00:00:00Z"
		},
		{
			"count":9,
			"timestamp":"2000-01-09T00:00:00Z"
		}
	],
	"metadata":{
		"next_page_token":"next-page-token"
	}
}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/features/browsers/chrome/missing_one_implementation_counts?"+
					"browser=edge&browser=firefox&browser=safari&"+
					"startAt=2000-01-01&endAt=2000-01-10&page_size=50&page_token="+*inputPageToken, nil),
		},
		{
			name: "500 case",
			mockConfig: &MockListMissingOneImplCountsConfig{
				expectedTargetBrowser:       "chrome",
				expectedTargetMobileBrowser: nil,
				expectedOtherBrowsers:       []string{"edge", "firefox", "safari"},
				expectedStartAt:             time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:               time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:            100,
				expectedPageToken:           nil,
				page:                        nil,
				pageToken:                   nil,
				err:                         errTest,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `ListMissingOneImplementationCounts-{"browser":"chrome","Params":{"startAt":"2000-01-01",` +
						`"endAt":"2000-01-10","browser":["edge","firefox","safari"]}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  1,
			expectedResponse: testJSONResponse(
				500, `{"code":500,"message":"unable to get missing one implementation metrics"}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/features/browsers/chrome/missing_one_implementation_counts?"+
					"browser=edge&browser=firefox&browser=safari&"+
					"startAt=2000-01-01&endAt=2000-01-10", nil),
		},
		{
			name: "400 case - invalid page token",
			mockConfig: &MockListMissingOneImplCountsConfig{
				expectedTargetBrowser:       "chrome",
				expectedTargetMobileBrowser: nil,
				expectedOtherBrowsers:       []string{"edge", "firefox", "safari"},
				expectedStartAt:             time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:               time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:            100,
				expectedPageToken:           badPageToken,
				page:                        nil,
				pageToken:                   nil,
				err:                         backendtypes.ErrInvalidPageToken,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `ListMissingOneImplementationCounts-{"browser":"chrome","Params":{"startAt":"2000-01-01",` +
						`"endAt":"2000-01-10","page_token":"","browser":["edge","firefox","safari"]}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  1,
			expectedResponse:   testJSONResponse(400, `{"code":400,"message":"invalid page token"}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/features/browsers/chrome/missing_one_implementation_counts?"+
					"browser=edge&browser=firefox&browser=safari&"+
					"startAt=2000-01-01&endAt=2000-01-10&page_token"+*badPageToken, nil),
		},
		{
			name:       "400 case - no matching mobile browser",
			mockConfig: nil,
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `ListMissingOneImplementationCounts-{"browser":"edge","Params":{"startAt":"2000-01-01",` +
						`"endAt":"2000-01-10","include_baseline_mobile_browsers":true,` +
						`"browser":["chrome","firefox","safari"]}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  0,
			expectedResponse: testJSONResponse(400,
				`{"code":400,"message":"browser does not have a matching mobile browser"}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/features/browsers/edge/missing_one_implementation_counts?"+
					"browser=chrome&browser=firefox&browser=safari&"+
					"startAt=2000-01-01&endAt=2000-01-10&include_baseline_mobile_browsers=true", nil),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				listMissingOneImplCountCfg: tc.mockConfig,
				t:                          t,
			}
			mockCacher := NewMockRawBytesDataCacher(t, tc.expectedCacheCalls, tc.expectedGetCalls)
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil,
				operationResponseCaches: initOperationResponseCaches(mockCacher, getTestRouteCacheOptions())}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse)
			assertMocksExpectations(t, tc.expectedCallCount, mockStorer.callCountListMissingOneImplCounts,
				"ListMissingOneImplCounts", mockCacher)
		})
	}
}
