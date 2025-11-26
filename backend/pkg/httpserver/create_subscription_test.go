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
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestCreateSubscription(t *testing.T) {
	now := time.Now()
	testUser := &auth.User{
		ID:           "test-user",
		GitHubUserID: nil,
	}

	testCases := []struct {
		name                 string
		cfg                  *MockCreateSavedSearchSubscriptionConfig
		expectedCallCount    int
		authMiddlewareOption testServerOption
		request              *http.Request
		expectedResponse     *http.Response
	}{
		{
			name: "success",
			cfg: &MockCreateSavedSearchSubscriptionConfig{
				expectedUserID: "test-user",
				expectedSubscription: backend.Subscription{
					ChannelId:     "channel-id",
					SavedSearchId: "search-id",
					Triggers: []backend.SubscriptionTriggerWritable{
						backend.SubscriptionTriggerFeatureAnyBrowserImplementationComplete},
					Frequency: "daily",
				},
				output: &backend.SubscriptionResponse{
					Id:            "sub-id",
					ChannelId:     "channel-id",
					SavedSearchId: "search-id",
					Triggers: []backend.SubscriptionTriggerResponseItem{
						{
							Value: backendtypes.AttemptToStoreSubscriptionTrigger(
								backend.SubscriptionTriggerFeatureAnyBrowserImplementationComplete),
							RawValue: nil,
						},
					},
					Frequency: "daily",
					CreatedAt: now,
					UpdatedAt: now,
				},
				err: nil,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/users/me/subscriptions",
				strings.NewReader(`{
					"channel_id": "channel-id",
					"saved_search_id": "search-id",
					"triggers": ["feature_any_browser_implementation_complete"],
					"frequency": "daily"
				}`),
			),
			expectedResponse: testJSONResponse(http.StatusCreated, `{
				"id":"sub-id",
				"channel_id":"channel-id",
				"saved_search_id":"search-id",
				"triggers": [{"value":"feature_any_browser_implementation_complete"}],
				"frequency":"daily",
				"created_at":"`+now.Format(time.RFC3339Nano)+`",
				"updated_at":"`+now.Format(time.RFC3339Nano)+`"
			}`),
		},
		{
			name:                 "bad request - missing channel id",
			cfg:                  nil,
			expectedCallCount:    0,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/users/me/subscriptions",
				strings.NewReader(`{
					"saved_search_id": "search-id",
					"triggers": ["feature_any_browser_implementation_complete"],
					"frequency": "daily"
				}`),
			),
			expectedResponse: testJSONResponse(http.StatusBadRequest, `
			{
				"code":400,
				"message":"input validation errors",
				"errors":{
					"channel_id":"channel_id is required"
				}
			}`),
		},
		{
			name: "forbidden - user not authorized",
			cfg: &MockCreateSavedSearchSubscriptionConfig{
				expectedUserID: "test-user",
				expectedSubscription: backend.Subscription{
					ChannelId:     "channel-id",
					SavedSearchId: "search-id",
					Triggers: []backend.SubscriptionTriggerWritable{
						backend.SubscriptionTriggerFeatureAnyBrowserImplementationComplete},
					Frequency: "daily",
				},
				output: nil,
				err:    backendtypes.ErrUserNotAuthorizedForAction,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/users/me/subscriptions",
				strings.NewReader(`{
					"channel_id": "channel-id",
					"saved_search_id": "search-id",
					"triggers": ["feature_any_browser_implementation_complete"],
					"frequency": "daily"
				}`)),
			expectedResponse: testJSONResponse(http.StatusForbidden, `{
				"code":403,
				"message":"user not authorized to create this subscription using the specified channel"
			}`),
		},
		{
			name: "internal server error",
			cfg: &MockCreateSavedSearchSubscriptionConfig{
				expectedUserID: "test-user",
				expectedSubscription: backend.Subscription{
					ChannelId:     "channel-id",
					SavedSearchId: "search-id",
					Triggers: []backend.SubscriptionTriggerWritable{
						backend.SubscriptionTriggerFeatureAnyBrowserImplementationComplete},
					Frequency: "daily",
				},
				output: nil,
				err:    fmt.Errorf("database error"),
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/users/me/subscriptions",
				strings.NewReader(`{
					"channel_id": "channel-id",
					"saved_search_id": "search-id",
					"triggers": ["feature_any_browser_implementation_complete"],
					"frequency": "daily"
				}`)),
			expectedResponse: testJSONResponse(http.StatusInternalServerError, `{
				"code":500,
				"message":"could not create subscription"
			}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint:exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				createSavedSearchSubscriptionCfg: tc.cfg,
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
				mockStorer.callCountCreateSavedSearchSubscription,
				"CreateSavedSearchSubscription",
				nil)
		})
	}
}

func TestValidateSubscriptionCreation(t *testing.T) {
	testCases := []struct {
		name  string
		input *backend.Subscription
		want  *fieldValidationErrors
	}{
		{
			name: "valid subscription",
			input: &backend.Subscription{
				ChannelId:     "channel-id",
				SavedSearchId: "search-id",
				Triggers: []backend.SubscriptionTriggerWritable{
					backend.SubscriptionTriggerFeatureAnyBrowserImplementationComplete},
				Frequency: backend.SubscriptionFrequencyDaily,
			},
			want: nil,
		},
		{
			name: "invalid channel id",
			input: &backend.Subscription{
				ChannelId:     "",
				SavedSearchId: "searchid",
				Triggers: []backend.SubscriptionTriggerWritable{
					backend.SubscriptionTriggerFeatureAnyBrowserImplementationComplete},
				Frequency: backend.SubscriptionFrequencyDaily,
			},
			want: &fieldValidationErrors{
				fieldErrorMap: map[string]string{
					"channel_id": errSubscriptionChannelIDRequired.Error(),
				},
			},
		},
		{
			name: "invalid saved search id",
			input: &backend.Subscription{
				ChannelId:     "channelid",
				SavedSearchId: "",
				Triggers: []backend.SubscriptionTriggerWritable{
					backend.SubscriptionTriggerFeatureAnyBrowserImplementationComplete},
				Frequency: backend.SubscriptionFrequencyDaily,
			},
			want: &fieldValidationErrors{
				fieldErrorMap: map[string]string{
					"saved_search_id": errSubscriptionSavedSearchIDRequired.Error(),
				},
			},
		},
		{
			name: "invalid trigger",
			input: &backend.Subscription{
				ChannelId:     "channelid",
				SavedSearchId: "searchid",
				Triggers: []backend.SubscriptionTriggerWritable{
					"invalid_trigger"},
				Frequency: backend.SubscriptionFrequencyDaily,
			},
			want: &fieldValidationErrors{
				fieldErrorMap: map[string]string{
					"triggers": errSubscriptionInvalidTrigger.Error(),
				},
			},
		},
		{
			name: "invalid frequency",
			input: &backend.Subscription{
				ChannelId:     "channelid",
				SavedSearchId: "searchid",
				Triggers: []backend.SubscriptionTriggerWritable{
					backend.SubscriptionTriggerFeatureAnyBrowserImplementationComplete},
				Frequency: "invalid_frequency",
			},
			want: &fieldValidationErrors{
				fieldErrorMap: map[string]string{
					"frequency": errSubscriptionInvalidFrequency.Error(),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := validateSubscriptionCreation(tc.input)
			if !reflect.DeepEqual(tc.want, out) {
				t.Errorf("validateSubscriptionCreation() = %v, want %v", out, tc.want)
			}
		})
	}
}
