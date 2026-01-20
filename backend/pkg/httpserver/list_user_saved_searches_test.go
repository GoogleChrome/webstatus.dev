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

func TestListUserSavedSearches(t *testing.T) {
	testUser := &auth.User{
		ID:           "listUserID1",
		GitHubUserID: nil,
	}
	testCases := []struct {
		name                 string
		cfg                  *MockListUserSavedSeachesConfig
		expectedCallCount    int
		authMiddlewareOption testServerOption
		request              *http.Request
		expectedResponse     *http.Response
	}{
		{
			name: "success",
			cfg: &MockListUserSavedSeachesConfig{
				expectedUserID:    "listUserID1",
				expectedPageSize:  100,
				expectedPageToken: nil,
				output: &backend.UserSavedSearchPage{
					Metadata: nil,
					Data: valuePtr([]backend.SavedSearchResponse{
						{
							Id:          "saved-search-id-2",
							CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
							UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
							Name:        "z",
							Description: valuePtr("test description"),
							Query:       "test query",
							Permissions: nil,
							BookmarkStatus: &backend.UserSavedSearchBookmark{
								Status: backend.BookmarkActive,
							},
						},
					}),
				},
				err: nil,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request:              httptest.NewRequest(http.MethodGet, "/v1/users/me/saved-searches", nil),
			expectedResponse: testJSONResponse(200, `
{
	"data":[
		{
			"bookmark_status":{
				"status":"bookmark_active"
			},
			"created_at":"2000-01-01T00:00:00Z",
			"description":"test description",
			"id":"saved-search-id-2",
			"name":"z",
			"query":"test query",
			"updated_at":"2000-01-01T00:00:00Z"
		}
	]
}`),
		},
		{
			name: "success with optional parameters",
			cfg: &MockListUserSavedSeachesConfig{
				expectedUserID:    "listUserID1",
				expectedPageSize:  50,
				expectedPageToken: inputPageToken,
				output: &backend.UserSavedSearchPage{
					Metadata: &backend.PageMetadata{
						NextPageToken: nextPageToken,
					},
					Data: valuePtr([]backend.SavedSearchResponse{
						{
							Id:          "saved-search-id-2",
							CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
							UpdatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
							Name:        "z",
							Description: valuePtr("test description"),
							Query:       "test query",
							Permissions: nil,
							BookmarkStatus: &backend.UserSavedSearchBookmark{
								Status: backend.BookmarkActive,
							},
						},
					}),
				},
				err: nil,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodGet,
				"/v1/users/me/saved-searches?page_size=50&page_token="+*inputPageToken,
				nil,
			),
			expectedResponse: testJSONResponse(200, `
{
	"data":[
		{
			"bookmark_status":{
				"status":"bookmark_active"
			},
			"created_at":"2000-01-01T00:00:00Z",
			"description":"test description",
			"id":"saved-search-id-2",
			"name":"z",
			"query":"test query",
			"updated_at":"2000-01-01T00:00:00Z"
		}
	],
	"metadata":{
		"next_page_token":"next-page-token"
	}
}`),
		},
		{
			name: "500 case",
			cfg: &MockListUserSavedSeachesConfig{
				expectedUserID:    "listUserID1",
				expectedPageSize:  100,
				expectedPageToken: nil,
				output:            nil,
				err:               errTest,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request:              httptest.NewRequest(http.MethodGet, "/v1/users/me/saved-searches", nil),
			expectedResponse: testJSONResponse(500, `
			{
				"code":500,
				"message":"unable to list saved searches for user"
			}`),
		},
		{
			name: "400 case - invalid page token",
			cfg: &MockListUserSavedSeachesConfig{
				expectedUserID:    "listUserID1",
				expectedPageSize:  100,
				expectedPageToken: badPageToken,
				output:            nil,
				err:               backendtypes.ErrInvalidPageToken,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequest(
				http.MethodGet, "/v1/users/me/saved-searches?page_token="+*badPageToken, nil),
			expectedResponse: testJSONResponse(400, `{"code":400,"message":"invalid page token"}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint:exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				listUserSavedSearchesCfg: tc.cfg,
				t:                        t,
			}
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil, userGitHubClientFactory: nil,
				operationResponseCaches: nil, baseURL: getTestBaseURL(t),
				eventPublisher: nil}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse,
				[]testServerOption{tc.authMiddlewareOption}...)
			assertMocksExpectations(t, tc.expectedCallCount, mockStorer.callCountListUserSavedSearches,
				"ListUserSavedSearches", nil)
		})
	}
}
