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

func TestUpdateNotificationChannel_Restrictions(t *testing.T) {
	testUser := &auth.User{
		ID:           "testUserID1",
		GitHubUserID: nil,
	}

	testCases := []struct {
		name                 string
		requestBody          string
		expectedStatus       int
		expectedResponseBody string
		expectFetch          bool
		expectedGetOutput    *backend.NotificationChannelResponse
		updateStorerCfg      *MockUpdateNotificationChannelConfig
		expectedUpdateCount  int
	}{
		{
			name: "success webhook update",
			requestBody: `
{
	"name": "Updated Webhook",
	"update_mask": ["name"]
}`,
			expectedStatus: 200,
			expectedResponseBody: `
{
	"id": "channel123",
	"name": "Updated Webhook",
	"type": "webhook",
	"config": {
		"type": "webhook",
		"url": "https://hooks.slack.com/services/old"
	},
	"status": "enabled",
	"created_at": "2000-01-01T00:00:00Z",
	"updated_at": "2000-01-01T00:00:00Z"
}`,
			expectFetch: true,
			expectedGetOutput: &backend.NotificationChannelResponse{
				Id:   "channel123",
				Name: "Old Webhook",
				Type: backend.NotificationChannelResponseTypeWebhook,
				Config: newTestNotificationChannelConfig(t, backend.WebhookConfig{
					Type: backend.WebhookConfigTypeWebhook,
					Url:  "https://hooks.slack.com/services/old",
				}),
				Status:    backend.NotificationChannelStatusEnabled,
				CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			},
			updateStorerCfg: &MockUpdateNotificationChannelConfig{
				expectedUserID:    testUser.ID,
				expectedChannelID: "channel123",
				expectedRequest: backend.UpdateNotificationChannelRequest{
					Config:     nil,
					Name:       valuePtr("Updated Webhook"),
					UpdateMask: []backend.UpdateNotificationChannelRequestUpdateMask{backend.UpdateNotificationChannelRequestMaskName},
				},
				output: &backend.NotificationChannelResponse{
					Id:   "channel123",
					Name: "Updated Webhook",
					Type: backend.NotificationChannelResponseTypeWebhook,
					Config: newTestNotificationChannelConfig(t, backend.WebhookConfig{
						Type: backend.WebhookConfigTypeWebhook,
						Url:  "https://hooks.slack.com/services/old",
					}),
					Status:    backend.NotificationChannelStatusEnabled,
					CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				},
				err: nil,
			},
			expectedUpdateCount: 1,
		},
		{
			name: "reject update to existing email channel (rename)",
			requestBody: `
{
	"name": "New Name",
	"update_mask": ["name"]
}`,
			expectedStatus: 403,
			expectedResponseBody: `
{
	"code": 403,
	"message": "email notification channels cannot be updated manually"
}`,
			expectFetch: true,
			expectedGetOutput: &backend.NotificationChannelResponse{
				Id:   "channel123",
				Name: "Old Name",
				Type: backend.NotificationChannelResponseTypeEmail,
				Config: newTestNotificationChannelConfig(t, backend.EmailConfig{
					Type:    backend.EmailConfigTypeEmail,
					Address: "old@example.com",
				}),
				Status:    backend.NotificationChannelStatusEnabled,
				CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			},
			updateStorerCfg: &MockUpdateNotificationChannelConfig{
				expectedUserID:    testUser.ID,
				expectedChannelID: "channel123",
				expectedRequest: backend.UpdateNotificationChannelRequest{
					Config:     nil,
					Name:       valuePtr("New Name"),
					UpdateMask: []backend.UpdateNotificationChannelRequestUpdateMask{backend.UpdateNotificationChannelRequestMaskName},
				},
				output: nil,
				err:    backendtypes.ErrUserNotAuthorizedForAction,
			},
			expectedUpdateCount: 1,
		},
		{
			name: "reject update to email config (request validation)",
			requestBody: `
{
	"config": {
		"type": "email",
		"address": "new@example.com"
	},
	"update_mask": ["config"]
}`,
			expectedStatus: 400,
			expectedResponseBody: `
{
	"code": 400,
	"message": "input validation errors",
	"errors": {
		"config": "invalid config: only webhook updates are supported"
	}
}`,
			expectFetch:         false,
			expectedGetOutput:   nil,
			updateStorerCfg:     nil,
			expectedUpdateCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			channelID := "channel123"
			req := httptest.NewRequest(http.MethodPatch,
				"/v1/users/me/notification-channels/"+channelID,
				strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			expectedResponse := testJSONResponse(tc.expectedStatus, tc.expectedResponseBody)

			// Setup Mock
			getCfg := &MockGetNotificationChannelConfig{
				expectedUserID:    testUser.ID,
				expectedChannelID: channelID,
				output:            tc.expectedGetOutput,
				err:               nil,
			}

			updateCfg := tc.updateStorerCfg
			if updateCfg == nil {
				updateCfg = &MockUpdateNotificationChannelConfig{
					expectedUserID:    testUser.ID,
					expectedChannelID: channelID,
					expectedRequest: backend.UpdateNotificationChannelRequest{
						Config:     nil,
						Name:       nil,
						UpdateMask: nil,
					},
					output: nil,
					err:    nil,
				}
			}

			//nolint:exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				getNotificationChannelCfg:    getCfg,
				updateNotificationChannelCfg: updateCfg,
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

			assertTestServerRequest(t, &myServer, req, expectedResponse, withAuthMiddleware(mockAuthMiddleware(testUser)))

			fetchCount := 0
			if tc.expectedStatus != 400 {
				fetchCount = 1
			}
			assertMocksExpectations(t, fetchCount, mockStorer.callCountGetNotificationChannel, "GetNotificationChannel", nil)
			assertMocksExpectations(t, tc.expectedUpdateCount,
				mockStorer.callCountUpdateNotificationChannel, "UpdateNotificationChannel", nil)
		})
	}
}
