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
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestListNotificationChannels(t *testing.T) {
	testUser := &auth.User{
		ID:           "listUserID1",
		GitHubUserID: nil,
	}
	testCases := []struct {
		name                 string
		cfg                  *MockListNotificationChannelsConfig
		expectedCallCount    int
		authMiddlewareOption testServerOption
		request              *http.Request
		expectedResponse     *http.Response
	}{
		{
			name: "success",
			cfg: &MockListNotificationChannelsConfig{
				expectedUserID:    "listUserID1",
				expectedPageSize:  100,
				expectedPageToken: nil,
				output: &backend.NotificationChannelPage{
					Metadata: nil,
					Data: &[]backend.NotificationChannelResponse{
						{
							Id:        "channel1",
							Name:      "My Email",
							Type:      backend.NotificationChannelResponseTypeEmail,
							Value:     "test@example.com",
							Status:    backend.NotificationChannelStatusEnabled,
							CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
							UpdatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						},
					},
				},
				err: nil,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request:              httptest.NewRequest(http.MethodGet, "/v1/users/me/notification-channels", nil),
			expectedResponse: testJSONResponse(200, `
{
	"data":[
		{
			"id": "channel1",
			"name": "My Email",
			"type": "email",
			"value": "test@example.com",
			"status": "enabled",
			"created_at":"2000-01-01T00:00:00Z",
			"updated_at":"2000-01-01T00:00:00Z"
		}
	]
}`),
		},
		{
			name: "500 case",
			cfg: &MockListNotificationChannelsConfig{
				expectedUserID:    "listUserID1",
				expectedPageSize:  100,
				expectedPageToken: nil,
				output:            nil,
				err:               errTest,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request:              httptest.NewRequest(http.MethodGet, "/v1/users/me/notification-channels", nil),
			expectedResponse: testJSONResponse(500, `
			{
				"code":500,
				"message":"Could not list notification channels"
			}`),
		},
		{
			name: "400 case - invalid page token",
			cfg: &MockListNotificationChannelsConfig{
				expectedUserID:    "listUserID1",
				expectedPageSize:  100,
				expectedPageToken: badPageToken,
				output:            nil,
				err:               backendtypes.ErrInvalidPageToken,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodGet, "/v1/users/me/notification-channels?page_token="+*badPageToken, nil),
			expectedResponse: testJSONResponse(400, `{"code":400,"message":"Invalid page token"}`),
		},
		{
			name:                 "unauthenticated",
			cfg:                  nil,
			expectedCallCount:    0,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(nil)),
			request:              httptest.NewRequest(http.MethodGet, "/v1/users/me/notification-channels", nil),
			expectedResponse: testJSONResponse(500, `
			{
				"code": 500,
				"message": "internal server error"
			}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint:exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				listNotificationChannelsCfg: tc.cfg,
				t:                           t,
			}
			myServer := Server{
				wptMetricsStorer:        mockStorer,
				baseURL:                 getTestBaseURL(t),
				metadataStorer:          nil,
				operationResponseCaches: nil,
				userGitHubClientFactory: nil,
			}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse,
				[]testServerOption{tc.authMiddlewareOption}...)
			assertMocksExpectations(t, tc.expectedCallCount, mockStorer.callCountListNotificationChannels,
				"ListNotificationChannels", nil)
		})
	}
}
