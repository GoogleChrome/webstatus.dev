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
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestListSubscriptions(t *testing.T) {
	now := time.Now()
	testUser := &auth.User{
		ID:           "test-user",
		GitHubUserID: nil,
	}
	testCases := []struct {
		name                 string
		cfg                  *MockListSavedSearchSubscriptionsConfig
		expectedCallCount    int
		authMiddlewareOption testServerOption
		request              *http.Request
		expectedResponse     *http.Response
	}{
		{
			name: "success",
			cfg: &MockListSavedSearchSubscriptionsConfig{
				expectedUserID:    "test-user",
				expectedPageSize:  100,
				expectedPageToken: nil,
				output: &backend.SubscriptionPage{
					Data: &[]backend.SubscriptionResponse{
						{
							Id:            "sub-id",
							ChannelId:     "channel-id",
							SavedSearchId: "search-id",
							Triggers:      []backend.SubscriptionTrigger{"trigger"},
							Frequency:     "daily",
							CreatedAt:     now,
							UpdatedAt:     now,
						},
					},
					Metadata: &backend.PageMetadata{
						NextPageToken: nil,
					},
				},
				err: nil,
			},
			expectedCallCount: 1,
			request: httptest.NewRequest(
				http.MethodGet,
				"/v1/users/me/subscriptions/saved-searches",
				nil,
			),
			expectedResponse: testJSONResponse(http.StatusOK,
				`{
					"data":[
						{
							"id":"sub-id",
							"channel_id":"channel-id",
							"saved_search_id":"search-id",
							"triggers":["trigger"],
							"frequency":"daily",
							"created_at":"`+now.Format(time.RFC3339Nano)+`",
							"updated_at":"`+now.Format(time.RFC3339Nano)+`"
						}
					],
					"metadata":{}}`),
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint:exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				listSavedSearchSubscriptionsCfg: tc.cfg,
				t:                               t,
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
				mockStorer.callCountListSavedSearchSubscriptions,
				"ListSavedSearchSubscriptions",
				nil)
		})
	}

}
