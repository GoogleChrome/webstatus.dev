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

func TestGetFeatureMetadata(t *testing.T) {
	testCases := []struct {
		name                  string
		mockGetIDConfig       MockGetIDFromFeatureKeyConfig
		mockGetMetadataConfig MockGetFeatureMetadataConfig
		request               backend.GetFeatureMetadataRequestObject
		expectedResponse      backend.GetFeatureMetadataResponseObject
		expectedError         error
	}{
		{
			name: "success",
			mockGetIDConfig: MockGetIDFromFeatureKeyConfig{
				expectedFeatureKey: "key1",
				result:             valuePtr("id1"),
				err:                nil,
			},
			mockGetMetadataConfig: MockGetFeatureMetadataConfig{
				expectedFeatureID: "id1",
				result: &backend.FeatureMetadata{
					CanIUse: &backend.CanIUseInfo{
						Items: &[]backend.CanIUseItem{
							{
								Id: valuePtr("caniuse1"),
							},
						},
					},
					Description: valuePtr("desc"),
				},
				err: nil,
			},
			request: backend.GetFeatureMetadataRequestObject{
				FeatureId: "key1",
			},
			expectedResponse: backend.GetFeatureMetadata200JSONResponse(
				backend.FeatureMetadata{
					CanIUse: &backend.CanIUseInfo{
						Items: &[]backend.CanIUseItem{
							{
								Id: valuePtr("caniuse1"),
							},
						},
					},
					Description: valuePtr("desc"),
				},
			),
			expectedError: nil,
		},
		// TODO(jcscottiii). Add more test cases later.
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				getIDFromFeatureKeyConfig: tc.mockGetIDConfig,
				t:                         t,
			}
			mockMetadataStorer := &MockWebFeatureMetadataStorer{
				mockGetFeatureMetadataCfg: tc.mockGetMetadataConfig,
				t:                         t,
			}
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: mockMetadataStorer}

			// Call the function under test
			resp, err := myServer.GetFeatureMetadata(context.Background(), tc.request)

			// Assertions
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(tc.expectedResponse, resp) {
				t.Errorf("Unexpected response: %v", resp)
			}
		})
	}
}
