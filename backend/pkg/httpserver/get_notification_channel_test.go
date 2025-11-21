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

func TestGetNotificationChannel(t *testing.T) {
	testUser := &auth.User{
		ID:           "listUserID1",
		GitHubUserID: nil,
	}
	testCases := []struct {
		name                 string
		cfg                  *MockGetNotificationChannelConfig
		expectedCallCount    int
		authMiddlewareOption testServerOption
		request              *http.Request
		expectedResponse     *http.Response
	}{
		{
			name: "success",
			cfg: &MockGetNotificationChannelConfig{
				expectedUserID:    "listUserID1",
				expectedChannelID: "channel1",
				output: &backend.NotificationChannelResponse{

					Id:        "channel1",
					Name:      "My Email",
					Type:      backend.NotificationChannelResponseTypeEmail,
					Value:     "test@example.com",
					Status:    backend.NotificationChannelStatusEnabled,
					CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				},
				err: nil,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request:              httptest.NewRequest(http.MethodGet, "/v1/users/me/notification-channels/channel1", nil),
			expectedResponse: testJSONResponse(200, `
{
	"id": "channel1",
	"name": "My Email",
	"type": "email",
	"value": "test@example.com",
	"status": "enabled",
	"created_at":"2000-01-01T00:00:00Z",
	"updated_at":"2000-01-01T00:00:00Z"
}`),
		},
		{
			name: "not found",
			cfg: &MockGetNotificationChannelConfig{
				expectedUserID:    "listUserID1",
				expectedChannelID: "channel1",
				output:            nil,
				err:               backendtypes.ErrEntityDoesNotExist,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request:              httptest.NewRequest(http.MethodGet, "/v1/users/me/notification-channels/channel1", nil),
			expectedResponse: testJSONResponse(404, `
			{
				"code":404,
				"message":"Notification channel not found or not owned by user"
			}`),
		},
		{
			name: "500 error",
			cfg: &MockGetNotificationChannelConfig{
				expectedUserID:    "listUserID1",
				expectedChannelID: "channel1",
				output:            nil,
				err:               errTest,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request:              httptest.NewRequest(http.MethodGet, "/v1/users/me/notification-channels/channel1", nil),
			expectedResponse: testJSONResponse(500, `
			{
				"code":500,
				"message":"Could not retrieve notification channel"
			}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint:exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				getNotificationChannelCfg: tc.cfg,
				t:                         t,
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
			assertMocksExpectations(t, tc.expectedCallCount, mockStorer.callCountGetNotificationChannel,
				"GetNotificationChannel", nil)
		})
	}
}
