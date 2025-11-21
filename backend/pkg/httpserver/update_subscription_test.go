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
	"strings"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestUpdateSubscription(t *testing.T) {
	now := time.Now()
	testUser := &auth.User{
		ID:           "test-user",
		GitHubUserID: nil,
	}

	testCases := []struct {
		name                 string
		cfg                  *MockUpdateSavedSearchSubscriptionConfig
		expectedCallCount    int
		authMiddlewareOption testServerOption
		request              *http.Request
		expectedResponse     *http.Response
	}{
		{
			name: "success",
			cfg: &MockUpdateSavedSearchSubscriptionConfig{
				expectedUserID:         "test-user",
				expectedSubscriptionID: "sub-id",
				expectedUpdateRequest: backend.UpdateSubscriptionRequest{
					Triggers: &[]backend.SubscriptionTrigger{
						backend.SubscriptionTriggerFeatureAnyBrowserImplementationComplete},
					UpdateMask: []backend.UpdateSubscriptionRequestUpdateMask{
						backend.UpdateSubscriptionRequestMaskTriggers},
					Frequency: nil,
				},
				output: &backend.SubscriptionResponse{
					Id:            "sub-id",
					ChannelId:     "channel-id",
					SavedSearchId: "search-id",
					Triggers: []backend.SubscriptionTrigger{
						backend.SubscriptionTriggerFeatureAnyBrowserImplementationComplete},
					Frequency: "daily",
					CreatedAt: now,
					UpdatedAt: now,
				},
				err: nil,
			},
			expectedCallCount: 1,
			request: httptest.NewRequest(
				http.MethodPatch,
				"/v1/users/me/subscriptions/sub-id",
				strings.NewReader(`
					{
						"triggers":
							["feature_any_browser_implementation_complete"],
						"update_mask": ["triggers"]
					}`)),
			expectedResponse: testJSONResponse(http.StatusOK,
				`{
					"id":"sub-id",
					"channel_id":"channel-id",
					"saved_search_id":"search-id",
					"triggers":["feature_any_browser_implementation_complete"],
					"frequency":"daily",
					"created_at":"`+now.Format(time.RFC3339Nano)+`",
					"updated_at":"`+now.Format(time.RFC3339Nano)+`"
				}`),
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
		},
		{
			name: "not found",
			cfg: &MockUpdateSavedSearchSubscriptionConfig{
				expectedUserID:         "test-user",
				expectedSubscriptionID: "sub-id",
				expectedUpdateRequest: backend.UpdateSubscriptionRequest{
					Triggers: &[]backend.SubscriptionTrigger{
						backend.SubscriptionTriggerFeatureAnyBrowserImplementationComplete,
					},
					Frequency: nil,
					UpdateMask: []backend.UpdateSubscriptionRequestUpdateMask{
						backend.UpdateSubscriptionRequestMaskTriggers},
				},
				output: nil,
				err:    backendtypes.ErrEntityDoesNotExist,
			},
			expectedCallCount: 1,
			request: httptest.NewRequest(
				http.MethodPatch,
				"/v1/users/me/subscriptions/sub-id",
				strings.NewReader(`
				{
					"triggers": ["feature_any_browser_implementation_complete"],
					"update_mask": ["triggers"]
				}`)),
			expectedResponse: testJSONResponse(http.StatusNotFound, `
			{
				"code":404,
				"message":"subscription not found"
			}`),
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
		},
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			// nolint:exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				updateSavedSearchSubscriptionCfg: tc.cfg,
				t:                                t,
			}

			myServer := Server{
				wptMetricsStorer:        mockStorer,
				baseURL:                 getTestBaseURL(t),
				metadataStorer:          nil,
				operationResponseCaches: nil,
				userGitHubClientFactory: nil,
			}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse, tc.authMiddlewareOption)
			assertMocksExpectations(t,
				tc.expectedCallCount,
				mockStorer.callCountUpdateSavedSearchSubscription,
				"UpdateSavedSearchSubscription",
				nil)

		})

	}
}
