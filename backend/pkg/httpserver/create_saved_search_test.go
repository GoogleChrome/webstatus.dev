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
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func createStringOfNLength(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += "a"
	}

	return s
}

func TestCreateSavedSearch(t *testing.T) {
	testUser := &auth.User{
		ID: "testID1",
	}
	testCases := []struct {
		name                            string
		mockCreateUserSavedSearchConfig *MockCreateUserSavedSearchConfig
		authMiddlewareOption            testServerOption
		request                         *http.Request
		expectedResponse                *http.Response
	}{
		{
			name:                            "name is 33 characters long, missing query",
			mockCreateUserSavedSearchConfig: nil,
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/saved-searches",
				strings.NewReader(`{"name": "`+createStringOfNLength(33)+`"}`),
			),
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			expectedResponse: testJSONResponse(400,
				`{
					"code":400,
					"errors":{
						"name":"name must be between 1 and 32 characters long",
						"query":"query must be between 1 and 256 characters long"
					},
					"message":"input validation errors"
				}`),
		},
		{
			name:                            "name is empty",
			mockCreateUserSavedSearchConfig: nil,
			authMiddlewareOption:            withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/saved-searches",
				strings.NewReader(`{"name" : "", "query" : "test query"}`),
			),
			expectedResponse: testJSONResponse(400,
				`{
					"code":400,
					"errors":{
						"name":"name must be between 1 and 32 characters long"
					},
					"message":"input validation errors"
				}`),
		},
		{
			name:                            "name is missing",
			mockCreateUserSavedSearchConfig: nil,
			authMiddlewareOption:            withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/saved-searches",
				strings.NewReader(`{"query" : "test query"}`),
			),
			expectedResponse: testJSONResponse(400,
				`{
					"code":400,
					"errors":{
						"name":"name must be between 1 and 32 characters long"
					},
					"message":"input validation errors"
				}`),
		},
		{
			name:                            "query is empty",
			mockCreateUserSavedSearchConfig: nil,
			authMiddlewareOption:            withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/saved-searches",
				strings.NewReader(`{"query" : "", "name" : "test name"}`),
			),
			expectedResponse: testJSONResponse(400,
				`{
					"code":400,
					"errors":{
						"query":"query must be between 1 and 256 characters long"
					},
					"message":"input validation errors"
				}`),
		},
		{
			name:                            "query is 257 characters long",
			mockCreateUserSavedSearchConfig: nil,
			authMiddlewareOption:            withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/saved-searches",
				strings.NewReader(`{"query": "`+createStringOfNLength(257)+`", "name" : "test name"}`),
			),
			expectedResponse: testJSONResponse(400,
				`{
					"code":400,
					"errors":{
						"query":"query must be between 1 and 256 characters long"
					},
					"message":"input validation errors"
				}`),
		},
		{
			name:                            "description is empty",
			mockCreateUserSavedSearchConfig: nil,
			authMiddlewareOption:            withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/saved-searches",
				strings.NewReader(`{"query": "test query", "name" : "test name", "description": ""}`),
			),
			expectedResponse: testJSONResponse(400,
				`{
					"code":400,
					"errors":{
						"description":"description must be between 1 and 1024 characters long"
					},
					"message":"input validation errors"
				}`),
		},
		{
			name:                            "description is 1025 characters long",
			mockCreateUserSavedSearchConfig: nil,
			authMiddlewareOption:            withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/saved-searches",
				strings.NewReader(`{"query": "test query", "name" : "test name", "description": "`+
					createStringOfNLength(1025)+`"}`),
			),
			expectedResponse: testJSONResponse(400,
				`{
					"code":400,
					"errors":{
						"description":"description must be between 1 and 1024 characters long"
					},
					"message":"input validation errors"
				}`),
		},
		{
			name:                            "query has bad syntax",
			mockCreateUserSavedSearchConfig: nil,
			authMiddlewareOption:            withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/saved-searches",
				strings.NewReader(`{"query": "name:", "name" : "test name"}`),
			),
			expectedResponse: testJSONResponse(400,
				`{
					"code":400,
					"errors":{
						"query":"query does not match grammar"
					},
					"message":"input validation errors"
				}`),
		},
		{
			name: "general creation error",
			mockCreateUserSavedSearchConfig: &MockCreateUserSavedSearchConfig{
				expectedSavedSearch: backend.SavedSearch{
					Name:        "test name",
					Query:       `name:"test"`,
					Description: nil,
				},
				expectedUserID: "testID1",
				output:         nil,
				err:            errTest,
			},
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/saved-searches",
				strings.NewReader(`{"query": "name:\"test\"", "name" : "test name"}`),
			),
			expectedResponse: testJSONResponse(500,
				`{
					"code":500,
					"message":"unable to create user saved search"
				}`,
			),
		},
		{
			name: "user limit exceeded error",
			mockCreateUserSavedSearchConfig: &MockCreateUserSavedSearchConfig{
				expectedSavedSearch: backend.SavedSearch{
					Name:        "test name",
					Query:       `name:"test"`,
					Description: nil,
				},
				expectedUserID: "testID1",
				output:         nil,
				err:            errors.Join(backendtypes.ErrUserMaxSavedSearches, errTest),
			},
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/saved-searches",
				strings.NewReader(`{"query": "name:\"test\"", "name" : "test name"}`),
			),
			expectedResponse: testJSONResponse(403,
				`{
					"code":403,
					"message":"user has reached the maximum number of allowed saved searches"
				}`,
			),
		},
		{
			name: "successful with name and query",
			mockCreateUserSavedSearchConfig: &MockCreateUserSavedSearchConfig{
				expectedSavedSearch: backend.SavedSearch{
					Name:        "test name",
					Query:       `name:"test"`,
					Description: nil,
				},
				expectedUserID: "testID1",
				output: &backend.SavedSearchResponse{
					Id:          "searchID1",
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
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/saved-searches",
				strings.NewReader(`{"query": "name:\"test\"", "name" : "test name"}`),
			),
			expectedResponse: testJSONResponse(201,
				`{
					"created_at":"2000-01-01T00:00:00Z",
					"id":"searchID1",
					"name":"test name",
					"query":"name:\"test\"",
					"updated_at":"2000-01-01T00:00:00Z",
					"bookmark_status":{"status":"bookmark_active"},
					"permissions":{"role":"saved_search_owner"}
				}`,
			),
		},
		{
			name: "successful with name, query and description",
			mockCreateUserSavedSearchConfig: &MockCreateUserSavedSearchConfig{
				expectedSavedSearch: backend.SavedSearch{
					Name:        "test name",
					Query:       `name:"test"`,
					Description: valuePtr("test description"),
				},
				expectedUserID: "testID1",
				output: &backend.SavedSearchResponse{
					Id:          "searchID1",
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
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodPost,
				"/v1/saved-searches",
				strings.NewReader(
					`{
						"query": "name:\"test\"",
						"name" : "test name",
						"description": "test description"
					}`,
				),
			),
			expectedResponse: testJSONResponse(201,
				`{
					"created_at":"2000-01-01T00:00:00Z",
					"id":"searchID1",
					"name":"test name",
					"description" : "test description",
					"query":"name:\"test\"",
					"updated_at":"2000-01-01T00:00:00Z",
					"bookmark_status":{"status":"bookmark_active"},
					"permissions":{"role":"saved_search_owner"}
				}`,
			),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint:exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				createUserSavedSearchCfg: tc.mockCreateUserSavedSearchConfig,
				t:                        t,
			}
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil,
				operationResponseCaches: nil}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse,
				[]testServerOption{tc.authMiddlewareOption}...)
		})
	}
}
