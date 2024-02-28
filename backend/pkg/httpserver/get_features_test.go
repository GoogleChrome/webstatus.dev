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
	"reflect"
	"testing"

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
				expectedPageToken:            nil,
				expectedPageSize:             100,
				expectedAvailableBrowsers:    []string{},
				expectedNotAvailableBrowsers: []string{},
				data: []backend.Feature{
					{
						BaselineStatus: backend.High,
						FeatureId:      "feature1",
						Name:           "feature 1",
						Spec:           nil,
						Usage:          nil,
						Wpt:            nil,
					},
				},
				pageToken: nil,
				err:       nil,
			},
			expectedCallCount: 1,
			expectedResponse: backend.GetV1Features200JSONResponse{
				Data: []backend.Feature{
					{
						BaselineStatus: backend.High,
						FeatureId:      "feature1",
						Name:           "feature 1",
						Spec:           nil,
						Usage:          nil,
						Wpt:            nil,
					},
				},
				Metadata: &backend.PageMetadata{
					NextPageToken: nil,
				},
			},
			request: backend.GetV1FeaturesRequestObject{
				Params: backend.GetV1FeaturesParams{
					PageToken:      nil,
					PageSize:       nil,
					AvailableOn:    nil,
					NotAvailableOn: nil,
				},
			},
			expectedError: nil,
		},
		{
			name: "Success Case - include optional params",
			mockConfig: MockFeaturesSearchConfig{
				expectedPageToken:            inputPageToken,
				expectedPageSize:             50,
				expectedAvailableBrowsers:    []string{"browser1", "browser3"},
				expectedNotAvailableBrowsers: []string{"browser2", "browser4"},
				data: []backend.Feature{
					{
						BaselineStatus: backend.High,
						FeatureId:      "feature1",
						Name:           "feature 1",
						Spec:           nil,
						Usage:          nil,
						Wpt:            nil,
					},
				},
				pageToken: nextPageToken,
				err:       nil,
			},
			expectedCallCount: 1,
			expectedResponse: backend.GetV1Features200JSONResponse{
				Data: []backend.Feature{
					{
						BaselineStatus: backend.High,
						FeatureId:      "feature1",
						Name:           "feature 1",
						Spec:           nil,
						Usage:          nil,
						Wpt:            nil,
					},
				},
				Metadata: &backend.PageMetadata{
					NextPageToken: nextPageToken,
				},
			},
			request: backend.GetV1FeaturesRequestObject{
				Params: backend.GetV1FeaturesParams{
					PageToken:      inputPageToken,
					PageSize:       valuePtr[int](50),
					AvailableOn:    &[]string{"browser1", "browser3"},
					NotAvailableOn: &[]string{"browser2", "browser4"},
				},
			},
			expectedError: nil,
		},
		{
			name: "500 case",
			mockConfig: MockFeaturesSearchConfig{
				expectedPageToken:            nil,
				expectedPageSize:             100,
				expectedAvailableBrowsers:    []string{},
				expectedNotAvailableBrowsers: []string{},
				data: []backend.Feature{
					{
						BaselineStatus: backend.High,
						FeatureId:      "feature1",
						Name:           "feature 1",
						Spec:           nil,
						Usage:          nil,
						Wpt:            nil,
					},
				},
				pageToken: nil,
				err:       errTest,
			},
			expectedCallCount: 1,
			expectedResponse: backend.GetV1Features500JSONResponse{
				Code:    500,
				Message: "unable to get list of features",
			},
			request: backend.GetV1FeaturesRequestObject{
				Params: backend.GetV1FeaturesParams{
					PageToken:      nil,
					PageSize:       nil,
					AvailableOn:    nil,
					NotAvailableOn: nil,
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
