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
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestListMissingOneImplementationFeatures(t *testing.T) {
	foo := "foo"
	bar := "bar"
	testCases := []struct {
		name              string
		mockConfig        MockListMissingOneImplFeaturesConfig
		expectedCallCount int // For the mock method
		request           backend.ListMissingOneImplementationFeaturesRequestObject
		expectedResponse  backend.ListMissingOneImplementationFeaturesResponseObject
		expectedError     error
	}{
		{
			name: "Success Case - no optional params - use defaults",
			mockConfig: MockListMissingOneImplFeaturesConfig{
				expectedTargetBrowser: "chrome",
				expectedOtherBrowsers: []string{"edge", "firefox", "safari"},
				expectedtargetDate:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedPageSize:      100,
				expectedPageToken:     nil,
				pageToken:             nil,
				err:                   nil,
				page: &backend.MissingOneImplFeaturesPage{
					Metadata: &backend.PageMetadata{
						NextPageToken: nil,
					},
					Data: []backend.MissingOneImplFeature{
						{
							FeatureId: &foo,
						},
						{
							FeatureId: &bar,
						},
					},
				},
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListMissingOneImplementationFeatures200JSONResponse{
				Data: []backend.MissingOneImplFeature{
					{
						FeatureId: &foo,
					},
					{
						FeatureId: &bar,
					},
				},
				Metadata: &backend.PageMetadata{
					NextPageToken: nil,
				},
			},
			request: backend.ListMissingOneImplementationFeaturesRequestObject{
				Params: backend.ListMissingOneImplementationFeaturesParams{
					PageToken: nil,
					PageSize:  nil,
					Browser: []backend.SupportedBrowsers{
						backend.Edge, backend.Firefox, backend.Safari,
					},
				},
				Browser: backend.Chrome,
				Date:    openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
			},
			expectedError: nil,
		},
		{
			name: "Success Case - include optional params",
			mockConfig: MockListMissingOneImplFeaturesConfig{
				expectedTargetBrowser: "chrome",
				expectedOtherBrowsers: []string{"edge", "firefox", "safari"},
				expectedtargetDate:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedPageSize:      50,
				expectedPageToken:     inputPageToken,
				err:                   nil,
				page: &backend.MissingOneImplFeaturesPage{
					Metadata: &backend.PageMetadata{
						NextPageToken: nextPageToken,
					},
					Data: []backend.MissingOneImplFeature{
						{
							FeatureId: &foo,
						},
						{
							FeatureId: &bar,
						},
					},
				},
				pageToken: nextPageToken,
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListMissingOneImplementationFeatures200JSONResponse{
				Metadata: &backend.PageMetadata{
					NextPageToken: nextPageToken,
				},
				Data: []backend.MissingOneImplFeature{
					{
						FeatureId: &foo,
					},
					{
						FeatureId: &bar,
					},
				},
			},
			request: backend.ListMissingOneImplementationFeaturesRequestObject{
				Params: backend.ListMissingOneImplementationFeaturesParams{
					PageToken: inputPageToken,
					PageSize:  valuePtr[int](50),
					Browser: []backend.SupportedBrowsers{
						backend.Edge, backend.Firefox, backend.Safari,
					},
				},
				Browser: backend.Chrome,
				Date:    openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
			},
			expectedError: nil,
		},
		{
			name: "500 case",
			mockConfig: MockListMissingOneImplFeaturesConfig{
				expectedTargetBrowser: "chrome",
				expectedOtherBrowsers: []string{"edge", "firefox", "safari"},
				expectedtargetDate:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedPageSize:      100,
				expectedPageToken:     nil,
				page:                  nil,
				pageToken:             nil,
				err:                   errTest,
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListMissingOneImplementationFeatures500JSONResponse{
				Code:    500,
				Message: "unable to get missing one implementation feature list",
			},
			request: backend.ListMissingOneImplementationFeaturesRequestObject{
				Params: backend.ListMissingOneImplementationFeaturesParams{
					PageToken: nil,
					PageSize:  nil,
					Browser: []backend.SupportedBrowsers{
						backend.Edge, backend.Firefox, backend.Safari,
					},
				},
				Browser: backend.Chrome,
				Date:    openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				listMissingOneImplFeaturesCfg: &tc.mockConfig,
				t:                             t,
			}
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil}

			// Call the function under test
			resp, err := myServer.ListMissingOneImplementationFeatures(context.Background(), tc.request)

			// Assertions
			if mockStorer.callCountListMissingOneImplFeatures != tc.expectedCallCount {
				t.Errorf("Incorrect call count: expected %d, got %d",
					tc.expectedCallCount,
					mockStorer.callCountListMissingOneImplFeatures)
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
