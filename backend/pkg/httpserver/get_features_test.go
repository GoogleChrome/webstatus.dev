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
	"errors"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestListFeatures(t *testing.T) {
	testCases := []struct {
		name              string
		mockConfig        MockFeaturesSearchConfig
		expectedCallCount int // For the mock method
		request           backend.ListFeaturesRequestObject
		expectedResponse  backend.ListFeaturesResponseObject
		expectedError     error
	}{
		{
			name: "Success Case - no optional params - use defaults",
			mockConfig: MockFeaturesSearchConfig{
				expectedPageToken:     nil,
				expectedPageSize:      100,
				expectedSearchNode:    nil,
				expectedSortBy:        nil,
				expectedWPTMetricView: backend.SubtestCounts,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
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
						},
					},
				},
				err: nil,
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListFeatures200JSONResponse{
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
					},
				},
				Metadata: backend.PageMetadataWithTotal{
					NextPageToken: nil,
					Total:         100,
				},
			},
			request: backend.ListFeaturesRequestObject{
				Params: backend.ListFeaturesParams{
					PageToken:     nil,
					PageSize:      nil,
					Q:             nil,
					Sort:          nil,
					WptMetricView: nil,
				},
			},
			expectedError: nil,
		},
		{
			name: "Success Case - include optional params",
			mockConfig: MockFeaturesSearchConfig{
				expectedPageToken:     inputPageToken,
				expectedPageSize:      50,
				expectedWPTMetricView: backend.TestCounts,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
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
										Operator:   searchtypes.OperatorEq,
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
						},
					},
				},
				err: nil,
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListFeatures200JSONResponse{
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
					},
				},
				Metadata: backend.PageMetadataWithTotal{
					NextPageToken: nextPageToken,
					Total:         100,
				},
			},
			request: backend.ListFeaturesRequestObject{
				Params: backend.ListFeaturesParams{
					PageToken:     inputPageToken,
					PageSize:      valuePtr[int](50),
					Q:             valuePtr(url.QueryEscape("available_on:chrome AND name:grid")),
					Sort:          valuePtr[backend.ListFeaturesParamsSort](backend.NameDesc),
					WptMetricView: valuePtr(backend.TestCounts),
				},
			},
			expectedError: nil,
		},
		{
			name: "500 case",
			mockConfig: MockFeaturesSearchConfig{
				expectedPageToken:  nil,
				expectedPageSize:   100,
				expectedSearchNode: nil,
				expectedSortBy:     nil,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
				},
				expectedWPTMetricView: backend.SubtestCounts,
				page:                  nil,
				err:                   errTest,
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListFeatures500JSONResponse{
				Code:    500,
				Message: "unable to get list of features",
			},
			request: backend.ListFeaturesRequestObject{
				Params: backend.ListFeaturesParams{
					PageToken:     nil,
					PageSize:      nil,
					Q:             nil,
					Sort:          nil,
					WptMetricView: nil,
				},
			},
			expectedError: nil,
		},
		{
			name: "400 case - query string does not match grammar",
			mockConfig: MockFeaturesSearchConfig{
				expectedPageToken:     nil,
				expectedPageSize:      100,
				expectedSearchNode:    nil,
				expectedSortBy:        nil,
				expectedWPTMetricView: backend.SubtestCounts,
				expectedBrowsers:      nil,
				page:                  nil,
				err:                   errTest,
			},
			expectedCallCount: 0,
			expectedResponse: backend.ListFeatures400JSONResponse{
				Code:    400,
				Message: "query string does not match expected grammar",
			},
			request: backend.ListFeaturesRequestObject{
				Params: backend.ListFeaturesParams{
					PageToken:     nil,
					PageSize:      nil,
					Sort:          nil,
					Q:             valuePtr[string]("badterm:foo"),
					WptMetricView: nil,
				},
			},
			expectedError: nil,
		},
		{
			name: "400 case - query string not safe",
			mockConfig: MockFeaturesSearchConfig{
				expectedPageToken:     nil,
				expectedPageSize:      100,
				expectedSearchNode:    nil,
				expectedSortBy:        nil,
				expectedWPTMetricView: backend.SubtestCounts,
				expectedBrowsers:      nil,
				page:                  nil,
				err:                   errTest,
			},
			expectedCallCount: 0,
			expectedResponse: backend.ListFeatures400JSONResponse{
				Code:    400,
				Message: "query string cannot be decoded",
			},
			request: backend.ListFeaturesRequestObject{
				Params: backend.ListFeaturesParams{
					PageToken:     nil,
					PageSize:      nil,
					Q:             valuePtr[string]("%"),
					Sort:          nil,
					WptMetricView: nil,
				},
			},
			expectedError: nil,
		},
		{
			name: "400 case - invalid page token",
			mockConfig: MockFeaturesSearchConfig{
				expectedPageToken:  badPageToken,
				expectedPageSize:   100,
				expectedSearchNode: nil,
				expectedSortBy:     nil,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
				},
				expectedWPTMetricView: backend.SubtestCounts,
				page:                  nil,
				err:                   backendtypes.ErrInvalidPageToken,
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListFeatures400JSONResponse{
				Code:    400,
				Message: "invalid page token",
			},
			request: backend.ListFeaturesRequestObject{
				Params: backend.ListFeaturesParams{
					PageToken:     badPageToken,
					PageSize:      nil,
					Q:             nil,
					Sort:          nil,
					WptMetricView: nil,
				},
			},
			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				featuresSearchCfg: tc.mockConfig,
				t:                 t,
			}
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil}

			// Call the function under test
			resp, err := myServer.ListFeatures(context.Background(), tc.request)

			// Assertions
			if mockStorer.callCountFeaturesSearch != tc.expectedCallCount {
				t.Errorf("Incorrect call count: expected %d, got %d",
					tc.expectedCallCount,
					mockStorer.callCountFeaturesSearch)
			}

			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(tc.expectedResponse, resp) {
				t.Errorf("Unexpected response: %v", resp)
			}
		})
	}
}
