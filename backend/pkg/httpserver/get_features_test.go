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

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestGetV1Features(t *testing.T) {
	testCases := []struct {
		name              string
		mockConfig        MockFeaturesSearchConfig
		expectedCallCount int // For the mock method
		request           backend.GetV1FeaturesRequestObject
		expectedResponse  backend.GetV1FeaturesResponseObject
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
				page: &backend.FeaturePage{
					Metadata: backend.PageMetadataWithTotal{
						NextPageToken: nil,
						Total:         100,
					},
					Data: []backend.Feature{
						{
							BaselineStatus: backend.Widely,
							FeatureId:      "feature1",
							Name:           "feature 1",
							Spec:           nil,
							Usage:          nil,
							Wpt:            nil,
							// TODO(https://github.com/GoogleChrome/webstatus.dev/issues/160)
							BrowserImplementations: nil,
						},
					},
				},
				err: nil,
			},
			expectedCallCount: 1,
			expectedResponse: backend.GetV1Features200JSONResponse{
				Data: []backend.Feature{
					{
						BaselineStatus: backend.Widely,
						FeatureId:      "feature1",
						Name:           "feature 1",
						Spec:           nil,
						Usage:          nil,
						Wpt:            nil,
						// TODO(https://github.com/GoogleChrome/webstatus.dev/issues/160)
						BrowserImplementations: nil,
					},
				},
				Metadata: backend.PageMetadataWithTotal{
					NextPageToken: nil,
					Total:         100,
				},
			},
			request: backend.GetV1FeaturesRequestObject{
				Params: backend.GetV1FeaturesParams{
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
				expectedSearchNode: &searchtypes.SearchNode{
					Operator: searchtypes.OperatorRoot,
					Term:     nil,
					Children: []*searchtypes.SearchNode{
						{
							Operator: searchtypes.OperatorAND,
							Term:     nil,
							Children: []*searchtypes.SearchNode{
								{
									Children: nil,
									Term: &searchtypes.SearchTerm{
										Identifier: searchtypes.IdentifierAvailableOn,
										Value:      "chrome",
									},
									Operator: searchtypes.OperatorNone,
								},
								{
									Children: nil,
									Term: &searchtypes.SearchTerm{
										Identifier: searchtypes.IdentifierName,
										Value:      "grid",
									},
									Operator: searchtypes.OperatorNone,
								},
							},
						},
					},
				},
				expectedSortBy: valuePtr[backend.GetV1FeaturesParamsSort](backend.NameDesc),
				page: &backend.FeaturePage{
					Metadata: backend.PageMetadataWithTotal{
						NextPageToken: nextPageToken,
						Total:         100,
					},
					Data: []backend.Feature{
						{
							BaselineStatus: backend.Widely,
							FeatureId:      "feature1",
							Name:           "feature 1",
							Spec:           nil,
							Usage:          nil,
							Wpt:            nil,
							// TODO(https://github.com/GoogleChrome/webstatus.dev/issues/160)
							BrowserImplementations: nil,
						},
					},
				},
				err: nil,
			},
			expectedCallCount: 1,
			expectedResponse: backend.GetV1Features200JSONResponse{
				Data: []backend.Feature{
					{
						BaselineStatus: backend.Widely,
						FeatureId:      "feature1",
						Name:           "feature 1",
						Spec:           nil,
						Usage:          nil,
						Wpt:            nil,
						// TODO(https://github.com/GoogleChrome/webstatus.dev/issues/160)
						BrowserImplementations: nil,
					},
				},
				Metadata: backend.PageMetadataWithTotal{
					NextPageToken: nextPageToken,
					Total:         100,
				},
			},
			request: backend.GetV1FeaturesRequestObject{
				Params: backend.GetV1FeaturesParams{
					PageToken:     inputPageToken,
					PageSize:      valuePtr[int](50),
					Q:             valuePtr(url.QueryEscape("available_on:chrome AND name:grid")),
					Sort:          valuePtr[backend.GetV1FeaturesParamsSort](backend.NameDesc),
					WptMetricView: valuePtr(backend.TestCounts),
				},
			},
			expectedError: nil,
		},
		{
			name: "500 case",
			mockConfig: MockFeaturesSearchConfig{
				expectedPageToken:     nil,
				expectedPageSize:      100,
				expectedSearchNode:    nil,
				expectedSortBy:        nil,
				expectedWPTMetricView: backend.SubtestCounts,
				page:                  nil,
				err:                   errTest,
			},
			expectedCallCount: 1,
			expectedResponse: backend.GetV1Features500JSONResponse{
				Code:    500,
				Message: "unable to get list of features",
			},
			request: backend.GetV1FeaturesRequestObject{
				Params: backend.GetV1FeaturesParams{
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
				page:                  nil,
				err:                   errTest,
			},
			expectedCallCount: 0,
			expectedResponse: backend.GetV1Features400JSONResponse{
				Code:    400,
				Message: "query string does not match expected grammar",
			},
			request: backend.GetV1FeaturesRequestObject{
				Params: backend.GetV1FeaturesParams{
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
				page:                  nil,
				err:                   errTest,
			},
			expectedCallCount: 0,
			expectedResponse: backend.GetV1Features400JSONResponse{
				Code:    400,
				Message: "query string cannot be decoded",
			},
			request: backend.GetV1FeaturesRequestObject{
				Params: backend.GetV1FeaturesParams{
					PageToken:     nil,
					PageSize:      nil,
					Q:             valuePtr[string]("%"),
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
			resp, err := myServer.GetV1Features(context.Background(), tc.request)

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
