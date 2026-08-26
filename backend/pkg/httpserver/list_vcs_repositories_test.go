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

// Package httpserver tests for VCS repositories handler.
//
//nolint:dupl // Similar structure across listing handlers
package httpserver

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestListVCSRepositories(t *testing.T) {
	githubUserID := "12345"
	testUser := &auth.User{ID: "user-123", GitHubUserID: &githubUserID}
	invalidPageToken := "invalid-token"

	mockRepositories := []backend.VCSRepositorySummary{
		{
			FullName:          "GoogleChrome/webstatus.dev",
			Id:                "repo-123",
			Name:              "webstatus.dev",
			Owner:             "GoogleChrome",
			Private:           false,
			RepositoryId:      "repo-123",
			VcsInstallationId: "12345",
			VcsProvider:       "github",
		},
	}

	testCases := []struct {
		name                 string
		authMiddlewareOption testServerOption
		cfg                  *MockListVCSRepositoriesConfig
		request              *http.Request
		expectedResponse     *http.Response
	}{
		{
			name:                 "Success with Data",
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			cfg: &MockListVCSRepositoriesConfig{
				expectedProvider:  "github",
				expectedPageSize:  100,
				expectedPageToken: nil,
				result: &backend.VCSRepositoryPage{
					Data:     &mockRepositories,
					Metadata: nil,
				},
				err: nil,
			},
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/github/repositories", nil),
			expectedResponse: testJSONResponse(http.StatusOK, `{
				"data": [
					{
						"full_name": "GoogleChrome/webstatus.dev",
						"id": "repo-123",
						"name": "webstatus.dev",
						"owner": "GoogleChrome",
						"private": false,
						"repository_id": "repo-123",
						"vcs_installation_id": "12345",
						"vcs_provider": "github"
					}
				]
			}`),
		},
		{
			name:                 "Invalid Page Token",
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			cfg: &MockListVCSRepositoriesConfig{
				expectedProvider:  "github",
				expectedPageSize:  100,
				expectedPageToken: &invalidPageToken,
				result:            nil,
				err:               backendtypes.ErrInvalidPageToken,
			},
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/github/repositories?page_token=invalid-token", nil),
			expectedResponse: testJSONResponse(http.StatusBadRequest, `{
				"code": 400,
				"message": "invalid page token"
			}`),
		},
		{
			name:                 "Unsupported Provider",
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			cfg: &MockListVCSRepositoriesConfig{
				expectedProvider:  "unknown",
				expectedPageSize:  100,
				expectedPageToken: nil,
				result:            nil,
				err:               backendtypes.ErrUnsupportedVCSProvider,
			},
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/unknown/repositories", nil),
			expectedResponse: testJSONResponse(http.StatusBadRequest, `{
				"code": 400,
				"message": "unsupported VCS provider: unknown"
			}`),
		},
		{
			name:                 "Database Error",
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			cfg: &MockListVCSRepositoriesConfig{
				expectedProvider:  "github",
				expectedPageSize:  100,
				expectedPageToken: nil,
				result:            nil,
				err:               errors.New("db error"),
			},
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/github/repositories", nil),
			expectedResponse: testJSONResponse(http.StatusInternalServerError, `{
				"code": 500,
				"message": "failed to list VCS repositories"
			}`),
		},
		{
			name:                 "Unauthenticated",
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(nil)),
			cfg:                  nil,
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/github/repositories", nil),
			expectedResponse: testJSONResponse(http.StatusInternalServerError, `{
				"code": 500,
				"message": "internal server error"
			}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var opts []TestServerOption
			if tc.cfg != nil {
				//nolint:exhaustruct
				mockStorer := &MockWPTMetricsStorer{
					listVCSRepositoriesCfg: tc.cfg,
					t:                      t,
				}
				opts = append(opts, withCustomStorer(mockStorer))
			}
			server := setupTestServer(t, opts...)
			var authOpts []testServerOption
			if tc.authMiddlewareOption != nil {
				authOpts = append(authOpts, tc.authMiddlewareOption)
			}
			assertTestServerRequest(t, server, tc.request, tc.expectedResponse, authOpts...)
		})
	}
}
