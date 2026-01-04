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
					Triggers: &[]backend.SubscriptionTriggerWritable{
						backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete},
					UpdateMask: []backend.UpdateSubscriptionRequestUpdateMask{
						backend.UpdateSubscriptionRequestMaskTriggers},
					Frequency: nil,
				},
				output: &backend.SubscriptionResponse{
					Id:            "sub-id",
					ChannelId:     "channel-id",
					SavedSearchId: "search-id",
					Triggers: []backend.SubscriptionTriggerResponseItem{
						{
							Value: backendtypes.AttemptToStoreSubscriptionTrigger(
								backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete),
							RawValue: nil,
						},
					},
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
							["feature_browser_implementation_any_complete"],
						"update_mask": ["triggers"]
					}`)),
			expectedResponse: testJSONResponse(http.StatusOK,
				`{
					"id":"sub-id",
					"channel_id":"channel-id",
					"saved_search_id":"search-id",
					"triggers": [{"value":"feature_browser_implementation_any_complete"}],
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
					Triggers: &[]backend.SubscriptionTriggerWritable{
						backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete,
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
					"triggers": ["feature_browser_implementation_any_complete"],
					"update_mask": ["triggers"]
				}`)),
			expectedResponse: testJSONResponse(http.StatusNotFound, `
			{
				"code":404,
				"message":"subscription not found"
			}`),
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
		},
		{
			name: "bad request - invalid update mask",
			cfg:  nil,
			request: httptest.NewRequest(
				http.MethodPatch,
				"/v1/users/me/subscriptions/sub-id",
				strings.NewReader(`
				{
					"triggers": ["feature_browser_implementation_any_complete"],
					"update_mask": ["invalid_field"]
				}`)),
			expectedResponse: testJSONResponse(http.StatusBadRequest, `
			{
				"code":400,
				"message":"input validation errors",
				"errors":{
					"update_mask":"update_mask must be one of the following: [frequency triggers]"
				}
			}`),
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			expectedCallCount:    0,
		},
		{
			name: "forbidden - user not authorized",
			cfg: &MockUpdateSavedSearchSubscriptionConfig{
				expectedUserID:         "test-user",
				expectedSubscriptionID: "sub-id",
				expectedUpdateRequest: backend.UpdateSubscriptionRequest{
					Triggers: &[]backend.SubscriptionTriggerWritable{
						backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete},
					UpdateMask: []backend.UpdateSubscriptionRequestUpdateMask{
						backend.UpdateSubscriptionRequestMaskTriggers},
					Frequency: nil,
				},
				output: nil,
				err:    backendtypes.ErrUserNotAuthorizedForAction,
			},
			expectedCallCount: 1,
			request: httptest.NewRequest(
				http.MethodPatch,
				"/v1/users/me/subscriptions/sub-id",
				strings.NewReader(`
				{
					"triggers": ["feature_browser_implementation_any_complete"],
					"update_mask": ["triggers"]
				}`)),
			expectedResponse: testJSONResponse(http.StatusForbidden, `
			{
				"code":403,
				"message":"user not authorized to update this subscription"
			}`),
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
		},
		{
			name: "internal server error",
			cfg: &MockUpdateSavedSearchSubscriptionConfig{
				expectedUserID:         "test-user",
				expectedSubscriptionID: "sub-id",
				expectedUpdateRequest: backend.UpdateSubscriptionRequest{
					Triggers: &[]backend.SubscriptionTriggerWritable{
						backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete},
					UpdateMask: []backend.UpdateSubscriptionRequestUpdateMask{
						backend.UpdateSubscriptionRequestMaskTriggers},
					Frequency: nil,
				},
				output: nil,
				err:    fmt.Errorf("database error"),
			},
			expectedCallCount: 1,
			request: httptest.NewRequest(
				http.MethodPatch,
				"/v1/users/me/subscriptions/sub-id",
				strings.NewReader(`
				{
					"triggers": ["feature_browser_implementation_any_complete"],
					"update_mask": ["triggers"]
				}`)),
			expectedResponse: testJSONResponse(http.StatusInternalServerError, `
			{
				"code":500,
				"message":"could not update subscription"
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
				eventPublisher:          nil,
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

func TestValidateSubscriptionUpdate(t *testing.T) {
	testCases := []struct {
		name  string
		input *backend.UpdateSubscriptionRequest
		want  *fieldValidationErrors
	}{
		{
			name: "valid update",
			input: &backend.UpdateSubscriptionRequest{
				Triggers: &[]backend.SubscriptionTriggerWritable{
					backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete},
				UpdateMask: []backend.UpdateSubscriptionRequestUpdateMask{
					backend.UpdateSubscriptionRequestMaskTriggers},
				Frequency: nil,
			},
			want: nil,
		},
		{
			name: "invalid update mask",
			input: &backend.UpdateSubscriptionRequest{
				Triggers: &[]backend.SubscriptionTriggerWritable{
					backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete},
				UpdateMask: []backend.UpdateSubscriptionRequestUpdateMask{
					"invalid_field"},
				Frequency: nil,
			},
			want: &fieldValidationErrors{
				fieldErrorMap: map[string]string{
					"update_mask": errSubscriptionInvalidUpdateMask.Error(),
				},
			},
		},
		{
			name: "missing required triggers",
			input: &backend.UpdateSubscriptionRequest{
				Triggers: nil,
				UpdateMask: []backend.UpdateSubscriptionRequestUpdateMask{
					backend.UpdateSubscriptionRequestMaskTriggers},
				Frequency: nil,
			},
			want: &fieldValidationErrors{
				fieldErrorMap: map[string]string{
					"triggers": errSubscriptionInvalidTrigger.Error(),
				},
			},
		},
		{
			name: "nil update mask",
			input: &backend.UpdateSubscriptionRequest{
				Triggers: &[]backend.SubscriptionTriggerWritable{
					backend.SubscriptionTriggerFeatureBrowserImplementationAnyComplete},
				UpdateMask: nil,
				Frequency:  nil,
			},
			want: &fieldValidationErrors{
				fieldErrorMap: map[string]string{
					"update_mask": errSubscriptionUpdateMaskRequired.Error(),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := validateSubscriptionUpdate(tc.input)
			if !reflect.DeepEqual(tc.want, out) {
				t.Errorf("validateSubscriptionCreation() = %v, want %v", out, tc.want)
			}
		})
	}
}
