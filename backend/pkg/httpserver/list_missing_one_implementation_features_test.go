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

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestListMissingOneImplementationFeatures(t *testing.T) {
	testCases := []struct {
		name               string
		mockConfig         MockListMissingOneImplFeaturesConfig
		expectedCallCount  int // For the mock method
		expectedCacheCalls []*ExpectedCacheCall
		expectedGetCalls   []*ExpectedGetCall
		request            *http.Request
		expectedResponse   *http.Response
	}{
		{
			name: "Success Case - no optional params - use defaults",
			mockConfig: MockListMissingOneImplFeaturesConfig{
				expectedTargetBrowser:       "chrome",
				expectedTargetMobileBrowser: nil,
				expectedOtherBrowsers:       []string{"edge", "firefox", "safari"},
				expectedtargetDate:          time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedPageSize:            100,
				expectedPageToken:           nil,
				pageToken:                   nil,
				err:                         nil,
				page: &backend.MissingOneImplFeaturesPage{
					Metadata: &backend.PageMetadata{
						NextPageToken: nil,
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
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `ListMissingOneImplementationFeatures-{"browser":["chrome"],"targetDate":"2000-01-01","Params":{` +
						`"browser":["edge","firefox","safari"]}}`,
					Value: nil,
					Err:   nil,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  1,
			expectedResponse: testJSONResponse(200, `
			{
				"data":[{"feature_id":"foo"},{"feature_id":"bar"}],
				"metadata":{

				}
			}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/features/browsers/chrome/missing_one_implementation_counts/2000-01-01/features?"+
					"browser=edge&browser=firefox&browser=safari", nil),
		},
		{
			name: "Success Case - include optional params",
			mockConfig: MockListMissingOneImplFeaturesConfig{
				expectedTargetBrowser:       "chrome",
				expectedTargetMobileBrowser: valuePtr("chrome_android"),
				expectedOtherBrowsers:       []string{"firefox", "firefox_android", "safari", "safari_ios"},
				expectedtargetDate:          time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedPageSize:            50,
				expectedPageToken:           inputPageToken,
				err:                         nil,
				page: &backend.MissingOneImplFeaturesPage{
					Metadata: &backend.PageMetadata{
						NextPageToken: nextPageToken,
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
				pageToken: nextPageToken,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `ListMissingOneImplementationFeatures-{"browser":["chrome","chrome_android"],` +
						`"targetDate":"2000-01-01","Params":{"page_token":"input-token","page_size":50,` +
						`"browser":["firefox","firefox_android","safari","safari_ios"],` +
						`"include_baseline_mobile_browsers":true}}`,
					Value: nil,
					Err:   nil,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  1,
			expectedResponse: testJSONResponse(200, `
			{
				"data":[{"feature_id":"foo"},{"feature_id":"bar"}],
				"metadata":{
					"next_page_token":"next-page-token"
				}
			}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/features/browsers/chrome/missing_one_implementation_counts/2000-01-01/features?"+
					"browser=firefox&browser=safari&include_baseline_mobile_browsers=true"+
					"&page_size=50&page_token="+*inputPageToken, nil),
		},
		{
			name: "500 case",
			mockConfig: MockListMissingOneImplFeaturesConfig{
				expectedTargetBrowser:       "chrome",
				expectedTargetMobileBrowser: nil,
				expectedOtherBrowsers:       []string{"edge", "firefox", "safari"},
				expectedtargetDate:          time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedPageSize:            100,
				expectedPageToken:           nil,
				page:                        nil,
				pageToken:                   nil,
				err:                         errTest,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `ListMissingOneImplementationFeatures-{"browser":["chrome"],"targetDate":"2000-01-01","Params":{` +
						`"browser":["edge","firefox","safari"]}}`,
					Value: nil,
					Err:   nil,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  1,
			expectedResponse: testJSONResponse(
				500, `{"code":500,"message":"unable to get missing one implementation feature list"}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/features/browsers/chrome/missing_one_implementation_counts/2000-01-01/features?"+
					"browser=edge&browser=firefox&browser=safari", nil),
		},
		{
			name: "400 case - invalid page token",
			mockConfig: MockListMissingOneImplFeaturesConfig{
				expectedTargetBrowser:       "chrome",
				expectedTargetMobileBrowser: nil,
				expectedOtherBrowsers:       []string{"edge", "firefox", "safari"},
				expectedtargetDate:          time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedPageSize:            100,
				expectedPageToken:           badPageToken,
				page:                        nil,
				pageToken:                   nil,
				err:                         backendtypes.ErrInvalidPageToken,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `ListMissingOneImplementationFeatures-{"browser":["chrome"],"targetDate":"2000-01-01","Params":{` +
						`"browser":["edge","firefox","safari"]}}`,
					Value: nil,
					Err:   nil,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  1,
			expectedResponse:   testJSONResponse(400, `{"code":400,"message":"invalid page token"}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/features/browsers/chrome/missing_one_implementation_counts/2000-01-01/features?"+
					"browser=edge&browser=firefox&browser=safari&"+
					"page_token"+*badPageToken, nil),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				listMissingOneImplFeaturesCfg: &tc.mockConfig,
				t:                             t,
			}
			mockCacher := NewMockRawBytesDataCacher(t, nil, nil)
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil,
				operationResponseCaches: initOperationResponseCaches(mockCacher, getTestRouteCacheOptions())}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse)
			assertMocksExpectations(t, tc.expectedCallCount, mockStorer.callCountListMissingOneImplFeatures,
				"ListMissingOneImplementationFeatures", mockCacher)
		})
	}
}
