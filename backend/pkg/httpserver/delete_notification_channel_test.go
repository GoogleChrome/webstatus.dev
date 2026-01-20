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

func TestDeleteNotificationChannel(t *testing.T) {
	testUser := &auth.User{
		ID:           "listUserID1",
		GitHubUserID: nil,
	}
	testCases := []struct {
		name                 string
		cfg                  *MockDeleteNotificationChannelConfig
		expectedCallCount    int
		authMiddlewareOption testServerOption
		request              *http.Request
		expectedResponse     *http.Response
	}{
		{
			name: "success",
			cfg: &MockDeleteNotificationChannelConfig{
				expectedUserID:    "listUserID1",
				expectedChannelID: "channel1",
				err:               nil,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request:              httptest.NewRequest(http.MethodDelete, "/v1/users/me/notification-channels/channel1", nil),
			expectedResponse:     createEmptyBodyResponse(http.StatusNoContent),
		},
		{
			name: "not found",
			cfg: &MockDeleteNotificationChannelConfig{
				expectedUserID:    "listUserID1",
				expectedChannelID: "channel1",
				err:               backendtypes.ErrEntityDoesNotExist,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request:              httptest.NewRequest(http.MethodDelete, "/v1/users/me/notification-channels/channel1", nil),
			expectedResponse: testJSONResponse(404, `
			{
				"code":404,
				"message":"Notification channel not found or not owned by user"
			}`),
		},
		{
			name: "500 error",
			cfg: &MockDeleteNotificationChannelConfig{
				expectedUserID:    "listUserID1",
				expectedChannelID: "channel1",
				err:               errTest,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request:              httptest.NewRequest(http.MethodDelete, "/v1/users/me/notification-channels/channel1", nil),
			expectedResponse: testJSONResponse(500, `
			{
				"code":500,
				"message":"Could not delete notification channel"
			}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint:exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				deleteNotificationChannelCfg: tc.cfg,
				t:                            t,
			}
			myServer := Server{
				wptMetricsStorer:        mockStorer,
				baseURL:                 getTestBaseURL(t),
				metadataStorer:          nil,
				operationResponseCaches: nil,
				userGitHubClientFactory: nil,
				eventPublisher:          nil,
			}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse,
				[]testServerOption{tc.authMiddlewareOption}...)
			assertMocksExpectations(t, tc.expectedCallCount, mockStorer.callCountDeleteNotificationChannel,
				"DeleteNotificationChannel", nil)
		})
	}
}
