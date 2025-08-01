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
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestListFeatures(t *testing.T) {
	testCases := []struct {
		name               string
		mockConfig         *MockFeaturesSearchConfig
		expectedCallCount  int // For the mock method
		expectedCacheCalls []*ExpectedCacheCall
		expectedGetCalls   []*ExpectedGetCall
		request            *http.Request
		expectedResponse   *http.Response
	}{
		{
			name: "Success Case - no optional params - use defaults",
			mockConfig: &MockFeaturesSearchConfig{
				expectedPageToken:     nil,
				expectedPageSize:      100,
				expectedSearchNode:    nil,
				expectedSortBy:        nil,
				expectedWPTMetricView: backend.TestCounts,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
					backend.ChromeAndroid,
					backend.FirefoxAndroid,
					backend.SafariIos,
				},
				page: &backend.FeaturePage{
					Metadata: backend.PageMetadataWithTotal{
						NextPageToken: nil,
						Total:         100,
					},
					Data: []backend.Feature{
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
							FeatureId: "feature1",
							Name:      "feature 1",
							Spec:      nil,
							Usage:     nil,
							Wpt:       nil,
							BrowserImplementations: &map[string]backend.BrowserImplementation{
								"browser1": {
									Status:  valuePtr(backend.Available),
									Date:    nil,
									Version: valuePtr("101"),
								},
							},
							DeveloperSignals: nil,
							Evolution:        nil,
						},
					},
				},
				err: nil,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `listFeatures-{"Params":{}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: []*ExpectedCacheCall{
				{
					Key: `listFeatures-{"Params":{}}`,
					Value: []byte(
						`{"data":[{"baseline":{"high_date":"2001-01-01","low_date":"2000-01-01","status":"widely"},` +
							`"browser_implementations":{"browser1":{"status":"available","version":"101"}},` +
							`"feature_id":"feature1","name":"feature 1"}],"metadata":{"total":100}}`,
					),
					CacheCfg: getDefaultCacheConfig(),
				},
			},
			expectedCallCount: 1,
			expectedResponse: testJSONResponse(200, `
{
	"data":[
		{
			"baseline":{
				"high_date":"2001-01-01",
				"low_date":"2000-01-01",
				"status":"widely"
			},
			"browser_implementations":{
				"browser1":{
				"status":"available",
				"version":"101"
				}
			},
			"feature_id":"feature1",
			"name":"feature 1"
		}
	],
	"metadata":{
		"total":100
	}
}`,
			),
			request: httptest.NewRequest(http.MethodGet, "/v1/features", nil),
		},
		{
			name:       "Success Case - no optional params - use defaults - cached",
			mockConfig: nil,
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `listFeatures-{"Params":{}}`,
					Value: []byte(
						`{"data":[{"baseline":{"high_date":"2001-01-01","low_date":"2000-01-01","status":"widely"},` +
							`"browser_implementations":{"browser1":{"status":"available","version":"101"}},` +
							`"feature_id":"feature1","name":"feature 1"}],"metadata":{"total":100}}`,
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
			"baseline":{
				"high_date":"2001-01-01",
				"low_date":"2000-01-01",
				"status":"widely"
			},
			"browser_implementations":{
				"browser1":{
				"status":"available",
				"version":"101"
				}
			},
			"feature_id":"feature1",
			"name":"feature 1"
		}
	],
	"metadata":{
		"total":100
	}
}`,
			),
			request: httptest.NewRequest(http.MethodGet, "/v1/features", nil),
		},
		{
			name: "Success Case - include optional params",
			mockConfig: &MockFeaturesSearchConfig{
				expectedPageToken:     inputPageToken,
				expectedPageSize:      50,
				expectedWPTMetricView: backend.SubtestCounts,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
					backend.ChromeAndroid,
					backend.FirefoxAndroid,
					backend.SafariIos,
				},
				expectedSearchNode: &searchtypes.SearchNode{
					Keyword: searchtypes.KeywordRoot,
					Term:    nil,
					Children: []*searchtypes.SearchNode{
						{
							Keyword: searchtypes.KeywordAND,
							Term:    nil,
							Children: []*searchtypes.SearchNode{
								{
									Children: nil,
									Term: &searchtypes.SearchTerm{
										Identifier: searchtypes.IdentifierAvailableOn,
										Value:      "chrome",
										Operator:   searchtypes.OperatorEq,
									},
									Keyword: searchtypes.KeywordNone,
								},
								{
									Children: nil,
									Term: &searchtypes.SearchTerm{
										Identifier: searchtypes.IdentifierName,
										Value:      "grid",
										Operator:   searchtypes.OperatorLike,
									},
									Keyword: searchtypes.KeywordNone,
								},
							},
						},
					},
				},
				expectedSortBy: valuePtr[backend.ListFeaturesParamsSort](backend.NameDesc),
				page: &backend.FeaturePage{
					Metadata: backend.PageMetadataWithTotal{
						NextPageToken: nextPageToken,
						Total:         100,
					},
					Data: []backend.Feature{
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
							FeatureId: "feature1",
							Name:      "feature 1",
							Spec:      nil,
							Usage:     nil,
							Wpt:       nil,
							BrowserImplementations: &map[string]backend.BrowserImplementation{
								"chrome": {
									Status: valuePtr(backend.Available),
									Date: &openapi_types.Date{
										Time: time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC)},
									Version: valuePtr("101"),
								},
							},
							DeveloperSignals: &backend.FeatureDeveloperSignals{
								PositiveCount: 24,
							},
							Evolution: &backend.FeatureEvolutionInfo{
								SplitOffInfo: &backend.FeatureEvolutionSplit{
									Features: []backend.FeatureSplitInfo{
										{
											Id: "test-other",
										},
									},
								},
							},
						},
					},
				},
				err: nil,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `listFeatures-{"Params":{"page_token":"input-token","page_size":50,` +
						`"wpt_metric_view":"subtest_counts","q":"available_on:chrome AND name:grid","sort":"name_desc"}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: []*ExpectedCacheCall{
				{
					Key: `listFeatures-{"Params":{"page_token":"input-token","page_size":50,` +
						`"wpt_metric_view":"subtest_counts","q":"available_on:chrome AND name:grid","sort":"name_desc"}}`,
					Value: []byte(
						`{"data":[{"baseline":{"high_date":"2001-01-01","low_date":"2000-01-01","status":"widely"},` +
							`"browser_implementations":` +
							`{"chrome":{"date":"1999-01-01","status":"available","version":"101"}},` +
							`"developer_signals":{"positive_count":24},` +
							`"evolution":{"split_off_info":{"features":[{"id":"test-other"}]}},` +
							`"feature_id":"feature1","name":"feature 1"}],` +
							`"metadata":{"next_page_token":"next-page-token","total":100}}`,
					),
					CacheCfg: getDefaultCacheConfig(),
				},
			},
			expectedCallCount: 1,
			expectedResponse: testJSONResponse(200, `
{
	"data":[
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
				"version":"101"
				}
			},
			"developer_signals":{
				"positive_count":24
			},
			"evolution":{
				"split_off_info":{
					"features":[
						{"id":"test-other"}
					]
				}
			},
			"feature_id":"feature1",
			"name":"feature 1"
		}
	],
	"metadata":{
		"next_page_token":"next-page-token",
		"total":100
	}
}`,
			),
			request: httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("/v1/features?page_token=%s&page_size=50&q=%s&sort=name_desc&wpt_metric_view=subtest_counts",
					*inputPageToken,
					url.QueryEscape("available_on:chrome AND name:grid"),
				),
				nil),
		},
		{
			name:       "Success Case - include optional params - cached",
			mockConfig: nil,
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `listFeatures-{"Params":{"page_token":"input-token","page_size":50,` +
						`"wpt_metric_view":"subtest_counts","q":"available_on:chrome AND name:grid","sort":"name_desc"}}`,
					Value: []byte(
						`{"data":[{"baseline":{"high_date":"2001-01-01","low_date":"2000-01-01","status":"widely"},` +
							`"browser_implementations":` +
							`{"chrome":{"date":"1999-01-01","status":"available","version":"101"}},` +
							`"developer_signals":{"positive_count":24},` +
							`"evolution":{"split_off_info":{"features":[{"id":"test-other"}]}},` +
							`"feature_id":"feature1","name":"feature 1"}],` +
							`"metadata":{"next_page_token":"next-page-token","total":100}}`,
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
			"baseline":{
				"high_date":"2001-01-01",
				"low_date":"2000-01-01",
				"status":"widely"
			},
			"browser_implementations":{
				"chrome":{
				"date":"1999-01-01",
				"status":"available",
				"version":"101"
				}
			},
			"developer_signals":{
				"positive_count":24
			},
			"evolution":{
				"split_off_info":{
					"features":[
						{"id":"test-other"}
					]
				}
			},
			"feature_id":"feature1",
			"name":"feature 1"
		}
	],
	"metadata":{
		"next_page_token":"next-page-token",
		"total":100
	}
}`,
			),
			request: httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("/v1/features?page_token=%s&page_size=50&q=%s&sort=name_desc&wpt_metric_view=subtest_counts",
					*inputPageToken,
					url.QueryEscape("available_on:chrome AND name:grid"),
				),
				nil),
		},
		{
			name: "500 case",
			mockConfig: &MockFeaturesSearchConfig{
				expectedPageToken:  nil,
				expectedPageSize:   100,
				expectedSearchNode: nil,
				expectedSortBy:     nil,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
					backend.ChromeAndroid,
					backend.FirefoxAndroid,
					backend.SafariIos,
				},
				expectedWPTMetricView: backend.TestCounts,
				page:                  nil,
				err:                   errTest,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `listFeatures-{"Params":{}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  1,
			expectedResponse: testJSONResponse(500,
				`{"code":500,"message":"unable to get list of features"}`,
			),
			request: httptest.NewRequest(http.MethodGet, "/v1/features", nil),
		},
		{
			name: "400 case - query string does not match grammar",
			mockConfig: &MockFeaturesSearchConfig{
				expectedPageToken:     nil,
				expectedPageSize:      100,
				expectedSearchNode:    nil,
				expectedSortBy:        nil,
				expectedWPTMetricView: backend.TestCounts,
				expectedBrowsers:      nil,
				page:                  nil,
				err:                   errTest,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `listFeatures-{"Params":{"q":"badterm:foo"}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  0,
			expectedResponse: testJSONResponse(400,
				`{"code":400,"message":"query string does not match expected grammar"}`,
			),
			request: httptest.NewRequest(http.MethodGet, "/v1/features?q=badterm:foo", nil),
		},
		{
			name: "400 case - query string not safe",
			mockConfig: &MockFeaturesSearchConfig{
				expectedPageToken:     nil,
				expectedPageSize:      100,
				expectedSearchNode:    nil,
				expectedSortBy:        nil,
				expectedWPTMetricView: backend.TestCounts,
				expectedBrowsers:      nil,
				page:                  nil,
				err:                   errTest,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `listFeatures-{"Params":{"q":"%"}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  0,
			expectedResponse: testJSONResponse(400,
				`{"code":400,"message":"query string cannot be decoded"}`,
			),
			request: httptest.NewRequest(http.MethodGet, "/v1/features?q="+url.QueryEscape("%"), nil),
		},
		{
			name: "400 case - invalid page token",
			mockConfig: &MockFeaturesSearchConfig{
				expectedPageToken:  badPageToken,
				expectedPageSize:   100,
				expectedSearchNode: nil,
				expectedSortBy:     nil,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
					backend.ChromeAndroid,
					backend.FirefoxAndroid,
					backend.SafariIos,
				},
				expectedWPTMetricView: backend.TestCounts,
				page:                  nil,
				err:                   backendtypes.ErrInvalidPageToken,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `listFeatures-{"Params":{"page_token":""}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  1,
			expectedResponse: testJSONResponse(400,
				`{"code":400,"message":"invalid page token"}`,
			),
			request: httptest.NewRequest(http.MethodGet, "/v1/features?page_token="+*badPageToken, nil),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				featuresSearchCfg: tc.mockConfig,
				t:                 t,
			}
			mockCacher := NewMockRawBytesDataCacher(t, tc.expectedCacheCalls, tc.expectedGetCalls)
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil,
				operationResponseCaches: initOperationResponseCaches(mockCacher, getTestRouteCacheOptions())}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse)
			assertMocksExpectations(t, tc.expectedCallCount, mockStorer.callCountFeaturesSearch,
				"FeaturesSearch", mockCacher)
		})
	}
}
