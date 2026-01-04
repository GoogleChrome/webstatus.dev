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

func TestUpdateSavedSearch(t *testing.T) {
	testUser := &auth.User{
		ID:           "testID1",
		GitHubUserID: nil,
	}
	// Common Request Bodies and Mock Settings
	updateAllFieldsRequestBody := `{
			"query": "name:\"test\"",
			"name" : "test name",
			"description": "test description",
			"update_mask": ["name", "query", "description"]
	}`
	updateAllFieldsExpectedRequest := &backend.SavedSearchUpdateRequest{
		Name:        valuePtr("test name"),
		Query:       valuePtr(`name:"test"`),
		Description: valuePtr("test description"),
		UpdateMask: []backend.SavedSearchUpdateRequestUpdateMask{
			backend.SavedSearchUpdateRequestMaskName,
			backend.SavedSearchUpdateRequestMaskQuery,
			backend.SavedSearchUpdateRequestMaskDescription,
		},
	}
	updateAllFieldsClearDescriptionExpectedRequest := &backend.SavedSearchUpdateRequest{
		Name:        valuePtr("test name"),
		Query:       valuePtr(`name:"test"`),
		Description: nil,
		UpdateMask: []backend.SavedSearchUpdateRequestUpdateMask{
			backend.SavedSearchUpdateRequestMaskName,
			backend.SavedSearchUpdateRequestMaskQuery,
			backend.SavedSearchUpdateRequestMaskDescription,
		},
	}
	testCases := []struct {
		name                 string
		cfg                  *MockUpdateUserSavedSearchConfig
		publishCfg           *MockPublishSearchConfigurationChangedConfig
		authMiddlewareOption testServerOption
		request              *http.Request
		expectedResponse     *http.Response
	}{
		{
			name:                 "missing body update error",
			cfg:                  nil,
			publishCfg:           nil,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPatch,
				"/v1/saved-searches/saved-search-id",
				strings.NewReader(`{}`),
			),
			expectedResponse: testJSONResponse(400,
				`{"code":400,"errors":{"update_mask":"update_mask must be set"},"message":"input validation errors"}`,
			),
		},
		{
			name:                 "empty update mask error",
			cfg:                  nil,
			publishCfg:           nil,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPatch,
				"/v1/saved-searches/saved-search-id",
				strings.NewReader(`{"update_mask": []}`),
			),
			expectedResponse: testJSONResponse(400,
				`{"code":400,"errors":{"update_mask":"update_mask must be set"},"message":"input validation errors"}`,
			),
		},
		{
			name:                 "update with invalid masks error",
			cfg:                  nil,
			publishCfg:           nil,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPatch,
				"/v1/saved-searches/saved-search-id",
				strings.NewReader(`{"update_mask": ["query", "foo"]}`),
			),
			expectedResponse: testJSONResponse(400,
				`{
					"code":400,
					"errors":{"update_mask":"invalid update_mask values: foo"},"message":"input validation errors"
				}`,
			),
		},
		{
			name:                 "missing fields, all update masks set, update error",
			cfg:                  nil,
			publishCfg:           nil,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPatch,
				"/v1/saved-searches/saved-search-id",
				strings.NewReader(`{
					"update_mask": ["name", "query", "description"]
				}`),
			),
			expectedResponse: testJSONResponse(400,
				`{
					"code":400,
					"errors":{
						"name":"name must be between 1 and 32 characters long",
						"query":"query must be between 1 and 256 characters long"
					},
					"message":"input validation errors"
				}`,
			),
		},
		{
			name: "forbidden error",
			cfg: &MockUpdateUserSavedSearchConfig{
				expectedSavedSearchID: "saved-search-id",
				expectedUserID:        "testID1",
				expectedUpdateRequest: updateAllFieldsExpectedRequest,
				output:                nil,
				err:                   backendtypes.ErrUserNotAuthorizedForAction,
			},
			publishCfg:           nil,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),

			request: httptest.NewRequest(
				http.MethodPatch,
				"/v1/saved-searches/saved-search-id",
				strings.NewReader(updateAllFieldsRequestBody),
			),
			expectedResponse: testJSONResponse(403,
				`{
					"code":403,
					"message":"forbidden"
				}`,
			),
		},
		{
			name: "forbidden error",
			cfg: &MockUpdateUserSavedSearchConfig{
				expectedSavedSearchID: "saved-search-id",
				expectedUserID:        "testID1",
				expectedUpdateRequest: updateAllFieldsExpectedRequest,
				output:                nil,
				err:                   backendtypes.ErrEntityDoesNotExist,
			},
			publishCfg:           nil,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),

			request: httptest.NewRequest(
				http.MethodPatch,
				"/v1/saved-searches/saved-search-id",
				strings.NewReader(updateAllFieldsRequestBody),
			),
			expectedResponse: testJSONResponse(404,
				`{
					"code":404,
					"message":"saved search not found"
				}`,
			),
		},
		{
			name: "general error",
			cfg: &MockUpdateUserSavedSearchConfig{
				expectedSavedSearchID: "saved-search-id",
				expectedUserID:        "testID1",
				expectedUpdateRequest: updateAllFieldsExpectedRequest,
				output:                nil,
				err:                   errTest,
			},
			publishCfg:           nil,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),

			request: httptest.NewRequest(
				http.MethodPatch,
				"/v1/saved-searches/saved-search-id",
				strings.NewReader(updateAllFieldsRequestBody),
			),
			expectedResponse: testJSONResponse(500,
				`{
					"code":500,
					"message":"unable to update user saved search"
				}`,
			),
		},
		{
			name: "success all fields",
			cfg: &MockUpdateUserSavedSearchConfig{
				expectedSavedSearchID: "saved-search-id",
				expectedUserID:        "testID1",
				expectedUpdateRequest: updateAllFieldsExpectedRequest,
				output: &backend.SavedSearchResponse{
					Id:          "saved-search-id",
					Name:        "test name",
					Query:       `name:"test"`,
					Description: valuePtr("test description"),
					Permissions: &backend.UserSavedSearchPermissions{
						Role: valuePtr(backend.SavedSearchOwner),
					},
					BookmarkStatus: &backend.UserSavedSearchBookmark{
						Status: backend.BookmarkActive,
					},
					CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				},
				err: nil,
			},
			publishCfg: &MockPublishSearchConfigurationChangedConfig{
				expectedResp: &backend.SavedSearchResponse{
					Id:          "saved-search-id",
					Name:        "test name",
					Query:       `name:"test"`,
					Description: valuePtr("test description"),
					Permissions: &backend.UserSavedSearchPermissions{
						Role: valuePtr(backend.SavedSearchOwner),
					},
					BookmarkStatus: &backend.UserSavedSearchBookmark{
						Status: backend.BookmarkActive,
					},
					CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				},
				expectedIsCreation: false,
				expectedUserID:     "testID1",
				err:                nil,
			},
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),

			request: httptest.NewRequest(
				http.MethodPatch,
				"/v1/saved-searches/saved-search-id",
				strings.NewReader(updateAllFieldsRequestBody),
			),
			expectedResponse: testJSONResponse(200,
				`{
					"bookmark_status":{
					   "status":"bookmark_active"
					},
					"created_at":"2000-01-01T00:00:00Z",
					"description":"test description",
					"id":"saved-search-id",
					"name":"test name",
					"permissions":{
					   "role":"saved_search_owner"
					},
					"query":"name:\"test\"",
					"updated_at":"2000-01-01T00:00:00Z"
				}`,
			),
		},
		{
			name: "success, all fields, clear description with explicit null, failed publish",
			cfg: &MockUpdateUserSavedSearchConfig{
				expectedSavedSearchID: "saved-search-id",
				expectedUserID:        "testID1",
				expectedUpdateRequest: updateAllFieldsClearDescriptionExpectedRequest,
				output: &backend.SavedSearchResponse{
					Id:          "saved-search-id",
					Name:        "test name",
					Query:       `name:"test"`,
					Description: nil,
					Permissions: &backend.UserSavedSearchPermissions{
						Role: valuePtr(backend.SavedSearchOwner),
					},
					BookmarkStatus: &backend.UserSavedSearchBookmark{
						Status: backend.BookmarkActive,
					},
					CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				},
				err: nil,
			},
			publishCfg: &MockPublishSearchConfigurationChangedConfig{
				expectedResp: &backend.SavedSearchResponse{
					Id:          "saved-search-id",
					Name:        "test name",
					Query:       `name:"test"`,
					Description: nil,
					Permissions: &backend.UserSavedSearchPermissions{
						Role: valuePtr(backend.SavedSearchOwner),
					},
					BookmarkStatus: &backend.UserSavedSearchBookmark{
						Status: backend.BookmarkActive,
					},
					CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				},
				expectedIsCreation: false,
				expectedUserID:     "testID1",
				err:                errTest,
			},
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),

			request: httptest.NewRequest(
				http.MethodPatch,
				"/v1/saved-searches/saved-search-id",
				strings.NewReader(
					`{
						"query": "name:\"test\"",
						"name" : "test name",
						"description": null,
						"update_mask": ["name", "query", "description"]
					}`,
				),
			),
			expectedResponse: testJSONResponse(200,
				`{
					"bookmark_status":{
					   "status":"bookmark_active"
					},
					"created_at":"2000-01-01T00:00:00Z",
					"id":"saved-search-id",
					"name":"test name",
					"permissions":{
					   "role":"saved_search_owner"
					},
					"query":"name:\"test\"",
					"updated_at":"2000-01-01T00:00:00Z"
				}`,
			),
		},
		{
			name: "success, all fields, clear description with implicit null",
			cfg: &MockUpdateUserSavedSearchConfig{
				expectedSavedSearchID: "saved-search-id",
				expectedUserID:        "testID1",
				expectedUpdateRequest: updateAllFieldsClearDescriptionExpectedRequest,
				output: &backend.SavedSearchResponse{
					Id:          "saved-search-id",
					Name:        "test name",
					Query:       `name:"test"`,
					Description: nil,
					Permissions: &backend.UserSavedSearchPermissions{
						Role: valuePtr(backend.SavedSearchOwner),
					},
					BookmarkStatus: &backend.UserSavedSearchBookmark{
						Status: backend.BookmarkActive,
					},
					CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				},
				err: nil,
			},
			publishCfg: &MockPublishSearchConfigurationChangedConfig{
				expectedResp: &backend.SavedSearchResponse{
					Id:          "saved-search-id",
					Name:        "test name",
					Query:       `name:"test"`,
					Description: nil,
					Permissions: &backend.UserSavedSearchPermissions{
						Role: valuePtr(backend.SavedSearchOwner),
					},
					BookmarkStatus: &backend.UserSavedSearchBookmark{
						Status: backend.BookmarkActive,
					},
					CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				},
				expectedUserID:     "testID1",
				expectedIsCreation: false,
				err:                nil,
			},
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),

			request: httptest.NewRequest(
				http.MethodPatch,
				"/v1/saved-searches/saved-search-id",
				strings.NewReader(
					`{
						"query": "name:\"test\"",
						"name" : "test name",
						"update_mask": ["name", "query", "description"]
					}`,
				),
			),
			expectedResponse: testJSONResponse(200,
				`{
					"bookmark_status":{
					   "status":"bookmark_active"
					},
					"created_at":"2000-01-01T00:00:00Z",
					"id":"saved-search-id",
					"name":"test name",
					"permissions":{
					   "role":"saved_search_owner"
					},
					"query":"name:\"test\"",
					"updated_at":"2000-01-01T00:00:00Z"
				}`,
			),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint:exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				updateUserSavedSearchCfg: tc.cfg,
				t:                        t,
			}
			mockPublisher := &MockEventPublisher{
				t: t,
				callCountPublishSearchConfigurationChanged: 0,
				publishSearchConfigurationChangedCfg:       tc.publishCfg,
			}
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil, userGitHubClientFactory: nil,
				operationResponseCaches: nil, baseURL: getTestBaseURL(t), eventPublisher: mockPublisher}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse,
				[]testServerOption{tc.authMiddlewareOption}...)
		})
	}
}
