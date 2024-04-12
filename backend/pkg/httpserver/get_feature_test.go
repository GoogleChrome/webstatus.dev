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

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestGetV1FeaturesFeatureId(t *testing.T) {
	testCases := []struct {
		name              string
		mockConfig        MockGetFeatureByIDConfig
		expectedCallCount int // For the mock method
		request           backend.GetV1FeaturesFeatureIdRequestObject
		expectedResponse  backend.GetV1FeaturesFeatureIdResponseObject
		expectedError     error
	}{
		{
			name: "Success Case - no optional params - use defaults",
			mockConfig: MockGetFeatureByIDConfig{
				expectedFeatureID: "feature1",
				data: &backend.Feature{
					BaselineStatus: backend.Widely,
					BrowserImplementations: &map[string]backend.BrowserImplementation{
						"chrome": {
							Status: valuePtr(backend.FullyImplemented),
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
			expectedResponse: backend.GetV1FeaturesFeatureId200JSONResponse{
				BaselineStatus: backend.Widely,
				BrowserImplementations: &map[string]backend.BrowserImplementation{
					"chrome": {
						Status: valuePtr(backend.FullyImplemented),
					},
				},
				FeatureId: "feature1",
				Name:      "feature 1",
				Spec:      nil,
				Usage:     nil,
				Wpt:       nil,
			},
			request: backend.GetV1FeaturesFeatureIdRequestObject{
				FeatureId: "feature1",
			},
			expectedError: nil,
		},
		{
			name: "404",
			mockConfig: MockGetFeatureByIDConfig{
				expectedFeatureID: "feature1",
				data:              nil,
				err:               gcpspanner.ErrQueryReturnedNoResults,
			},
			expectedCallCount: 1,
			expectedResponse: backend.GetV1FeaturesFeatureId404JSONResponse{
				Code:    404,
				Message: "feature id feature1 is not found",
			},
			request: backend.GetV1FeaturesFeatureIdRequestObject{
				FeatureId: "feature1",
			},
			expectedError: nil,
		},
		{
			name: "500",
			mockConfig: MockGetFeatureByIDConfig{
				expectedFeatureID: "feature1",
				data:              nil,
				err:               errTest,
			},
			expectedCallCount: 1,
			expectedResponse: backend.GetV1FeaturesFeatureId500JSONResponse{
				Code:    500,
				Message: "unable to get feature",
			},
			request: backend.GetV1FeaturesFeatureIdRequestObject{
				FeatureId: "feature1",
			},
			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				getFeatureByIDConfig: tc.mockConfig,
				t:                    t,
			}
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil}

			// Call the function under test
			resp, err := myServer.GetV1FeaturesFeatureId(context.Background(), tc.request)

			// Assertions
			if mockStorer.callCountGetFeature != tc.expectedCallCount {
				t.Errorf("Incorrect call count: expected %d, got %d",
					tc.expectedCallCount,
					mockStorer.callCountGetFeature)
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
