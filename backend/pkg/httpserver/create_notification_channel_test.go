// Copyright 2026 Google LLC
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

func TestCreateNotificationChannel(t *testing.T) {
	testUser := &auth.User{
		ID:           "testUserID1",
		GitHubUserID: nil,
	}

	testCases := []struct {
		name              string
		requestBody       string
		storerCfg         *MockCreateNotificationChannelConfig
		expectedCallCount int
		expectedResponse  *http.Response
	}{
		{
			name: "success webhook",
			requestBody: `
{
	"name": "My Webhook",
	"config": {
		"type": "webhook",
		"url": "https://hooks.slack.com/services/123"
	}
}`,
			storerCfg: &MockCreateNotificationChannelConfig{
				expectedUserID: testUser.ID,
				expectedRequest: backend.CreateNotificationChannelRequest{
					Name: "My Webhook",
					Config: newTestCreateNotificationChannelConfig(t, backend.WebhookConfig{
						Type: backend.WebhookConfigTypeWebhook,
						Url:  "https://hooks.slack.com/services/123",
					}),
				},
				output: &backend.NotificationChannelResponse{
					Id:   "channel123",
					Name: "My Webhook",
					Type: backend.NotificationChannelResponseTypeWebhook,
					Config: newTestNotificationChannelConfig(t, backend.WebhookConfig{
						Type: backend.WebhookConfigTypeWebhook,
						Url:  "https://hooks.slack.com/services/123",
					}),
					Status:    backend.NotificationChannelStatusEnabled,
					CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				},
				err: nil,
			},
			expectedCallCount: 1,
			expectedResponse: testJSONResponse(201, `
{
	"id": "channel123",
	"name": "My Webhook",
	"type": "webhook",
	"config": {
		"type": "webhook",
		"url": "https://hooks.slack.com/services/123"
	},
	"status": "enabled",
	"created_at": "2000-01-01T00:00:00Z",
	"updated_at": "2000-01-01T00:00:00Z"
}`),
		},
		{
			name: "reject email config",
			// Attempt to create an email channel manually should be rejected.
			// We use a raw JSON body since the generated client won't even allow this.
			requestBody: `
{
	"name": "My Email",
	"config": {
		"type": "email",
		"address": "test@example.com"
	}
}`,
			storerCfg:         nil,
			expectedCallCount: 0,
			expectedResponse: testJSONResponse(400, `
{
	"code": 400,
	"message": "input validation errors",
	"errors": {
		"config": "invalid config: only webhook channels can be created manually"
	}
}`),
		},
		{
			name: "max channels failure",
			requestBody: `
{
	"name": "Another Webhook",
	"config": {
		"type": "webhook",
		"url": "https://hooks.slack.com/services/456"
	}
}`,
			storerCfg: &MockCreateNotificationChannelConfig{
				expectedUserID: testUser.ID,
				expectedRequest: backend.CreateNotificationChannelRequest{
					Name: "Another Webhook",
					Config: newTestCreateNotificationChannelConfig(t, backend.WebhookConfig{
						Type: backend.WebhookConfigTypeWebhook,
						Url:  "https://hooks.slack.com/services/456",
					}),
				},
				output: nil,
				err:    backendtypes.ErrUserMaxNotificationChannels,
			},
			expectedCallCount: 1,
			expectedResponse: testJSONResponse(429, `
{
	"code": 429,
	"message": "user has reached the maximum number of allowed notification channels (25)"
}`),
		},
		{
			name: "generic failure",
			requestBody: `
{
	"name": "Generic Webhook",
	"config": {
		"type": "webhook",
		"url": "https://hooks.slack.com/services/789"
	}
}`,
			storerCfg: &MockCreateNotificationChannelConfig{
				expectedUserID: testUser.ID,
				expectedRequest: backend.CreateNotificationChannelRequest{
					Name: "Generic Webhook",
					Config: newTestCreateNotificationChannelConfig(t, backend.WebhookConfig{
						Type: backend.WebhookConfigTypeWebhook,
						Url:  "https://hooks.slack.com/services/789",
					}),
				},
				output: nil,
				err:    errTest,
			},
			expectedCallCount: 1,
			expectedResponse: testJSONResponse(500, `
{
	"code": 500,
	"message": "unable to create notification channel"
}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/users/me/notification-channels", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			//nolint:exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				createNotificationChannelCfg: tc.storerCfg,
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

			assertTestServerRequest(t, &myServer, req, tc.expectedResponse,
				withAuthMiddleware(mockAuthMiddleware(testUser)))
			assertMocksExpectations(t, tc.expectedCallCount,
				mockStorer.callCountCreateNotificationChannel, "CreateNotificationChannel", nil)
		})
	}
}
