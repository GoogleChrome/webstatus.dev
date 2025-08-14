// Copyright 2025 Google LLC
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

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
)

func TestRemoveSavedSearch(t *testing.T) {
	authMiddlewareOption := withAuthMiddleware(mockAuthMiddleware(createTestID1User()))

	testCases := []basicHTTPTestCase[MockDeleteUserSavedSearchConfig]{
		{
			name: "success",
			cfg: &MockDeleteUserSavedSearchConfig{
				expectedSavedSearchID: "saved-search-id",
				expectedUserID:        "testID1",
				err:                   nil,
			},
			request: httptest.NewRequest(
				http.MethodDelete,
				"/v1/saved-searches/saved-search-id",
				nil,
			),
			expectedResponse: createEmptyBodyResponse(http.StatusNoContent),
		},
		{
			name: "not found",
			cfg: &MockDeleteUserSavedSearchConfig{
				expectedSavedSearchID: "saved-search-id",
				expectedUserID:        "testID1",
				err:                   backendtypes.ErrEntityDoesNotExist,
			},
			request: httptest.NewRequest(
				http.MethodDelete,
				"/v1/saved-searches/saved-search-id",
				nil,
			),
			expectedResponse: testJSONResponse(404,
				`{
					"code":404,
					"message":"saved search not found"
				}`,
			),
		},
		{
			name: "not authorized",
			cfg: &MockDeleteUserSavedSearchConfig{
				expectedSavedSearchID: "saved-search-id",
				expectedUserID:        "testID1",
				err:                   backendtypes.ErrUserNotAuthorizedForAction,
			},
			request: httptest.NewRequest(
				http.MethodDelete,
				"/v1/saved-searches/saved-search-id",
				nil,
			),
			expectedResponse: testJSONResponse(403,
				`{
					"code":403,
					"message":"forbidden"
				}`,
			),
		},
		{
			name: "general error",
			cfg: &MockDeleteUserSavedSearchConfig{
				expectedSavedSearchID: "saved-search-id",
				expectedUserID:        "testID1",
				err:                   errTest,
			},
			request: httptest.NewRequest(
				http.MethodDelete,
				"/v1/saved-searches/saved-search-id",
				nil,
			),
			expectedResponse: testJSONResponse(500,
				`{
					"code":500,
					"message":"unable to delete saved search"
				}`,
			),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint:exhaustruct // WONTFIX
			mockStorer := &MockWPTMetricsStorer{
				deleteUserSavedSearchCfg: tc.cfg,
				t:                        t,
			}
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil,
				operationResponseCaches: nil, baseURL: getTestBaseURL(t)}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse,
				[]testServerOption{authMiddlewareOption}...)
			assertMocksExpectations(t, 1, mockStorer.callCountDeleteUserSavedSearch,
				"DeleteUserSavedSearch", nil)
		})
	}
}
