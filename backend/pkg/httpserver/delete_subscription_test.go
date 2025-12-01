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

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
)

func TestDeleteSubscription(t *testing.T) {
	testUser := &auth.User{
		ID:           "test-user",
		GitHubUserID: nil,
	}

	testCases := []struct {
		name                 string
		cfg                  *MockDeleteSavedSearchSubscriptionConfig
		expectedCallCount    int
		authMiddlewareOption testServerOption
		request              *http.Request
		expectedResponse     *http.Response
	}{
		{
			name: "success",
			cfg: &MockDeleteSavedSearchSubscriptionConfig{
				expectedUserID:         "test-user",
				expectedSubscriptionID: "sub-id",
				err:                    nil,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodDelete,
				"/v1/users/me/subscriptions/sub-id",
				nil,
			),
			expectedResponse: createEmptyBodyResponse(http.StatusNoContent),
		},
		{
			name: "not found",
			cfg: &MockDeleteSavedSearchSubscriptionConfig{
				expectedUserID:         "test-user",
				expectedSubscriptionID: "sub-id",
				err:                    backendtypes.ErrEntityDoesNotExist,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodDelete,
				"/v1/users/me/subscriptions/sub-id",
				nil,
			),
			expectedResponse: testJSONResponse(http.StatusNotFound, `{"code":404,"message":"subscription not found"}`),
		},
		{
			name: "internal error",
			cfg: &MockDeleteSavedSearchSubscriptionConfig{
				expectedUserID:         "test-user",
				expectedSubscriptionID: "sub-id",
				err:                    errTest,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodDelete,
				"/v1/users/me/subscriptions/sub-id",
				nil,
			),
			expectedResponse: testJSONResponse(http.StatusInternalServerError,
				`{"code":500,"message":"could not delete subscription"}`),
		},
		{
			name: "forbidden - user cannot access subscription",
			cfg: &MockDeleteSavedSearchSubscriptionConfig{
				expectedUserID:         "test-user",
				expectedSubscriptionID: "sub-id",
				err:                    backendtypes.ErrUserNotAuthorizedForAction,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodDelete,
				"/v1/users/me/subscriptions/sub-id",
				nil,
			),
			expectedResponse: testJSONResponse(http.StatusForbidden,
				`{"code":403,"message":"user not authorized to delete this subscription"}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint:exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				deleteSavedSearchSubscriptionCfg: tc.cfg,
				t:                                t,
			}
			myServer := Server{
				wptMetricsStorer:        mockStorer,
				metadataStorer:          nil,
				userGitHubClientFactory: nil,
				operationResponseCaches: nil,
				baseURL:                 getTestBaseURL(t),
			}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse, tc.authMiddlewareOption)
			assertMocksExpectations(t,
				tc.expectedCallCount,
				mockStorer.callCountDeleteSavedSearchSubscription,
				"DeleteSavedSearchSubscription",
				nil)
		})
	}
}
