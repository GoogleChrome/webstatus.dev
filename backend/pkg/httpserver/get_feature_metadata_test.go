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

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestGetFeatureMetadata(t *testing.T) {
	testCases := []struct {
		name                  string
		mockGetIDConfig       *MockGetIDFromFeatureKeyConfig
		mockGetMetadataConfig MockGetFeatureMetadataConfig
		expectedCacheCalls    []*ExpectedCacheCall
		expectedGetCalls      []*ExpectedGetCall
		request               *http.Request
		expectedResponse      *http.Response
	}{
		{
			name: "success",
			mockGetIDConfig: &MockGetIDFromFeatureKeyConfig{
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
			expectedCacheCalls: nil,
			expectedGetCalls:   nil,
			request:            httptest.NewRequest(http.MethodGet, "/v1/features/key1/feature-metadata", nil),
			expectedResponse: testJSONResponse(200,
				`{"can_i_use":{"items":[{"id":"caniuse1"}]},"description":"desc"}`,
			),
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
			mockCacher := NewMockRawBytesDataCacher(t, tc.expectedCacheCalls, tc.expectedGetCalls)
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: mockMetadataStorer,
				operationResponseCaches: initOperationResponseCaches(mockCacher)}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse)
			// TODO: Start tracking call count and assert call count. Then we can use assertMocksExpectations
			mockCacher.AssertExpectations()
		})
	}
}
