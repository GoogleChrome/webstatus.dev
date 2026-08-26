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

// Package httpserver tests for VCS installations handler.
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

func TestListVCSInstallations(t *testing.T) {
	githubUserID := "12345"
	testUser := &auth.User{ID: "user-123", GitHubUserID: &githubUserID}
	invalidPageToken := "invalid-token"

	mockInstallations := []backend.VCSInstallationSummary{
		{
			AccountLogin:      "GoogleChrome",
			AccountType:       "Organization",
			Id:                "inst-123",
			VcsInstallationId: "12345",
			VcsProvider:       "github",
		},
	}

	testCases := []struct {
		name                 string
		authMiddlewareOption testServerOption
		cfg                  *MockListVCSInstallationsConfig
		request              *http.Request
		expectedResponse     *http.Response
	}{
		{
			name:                 "Success with Data",
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			cfg: &MockListVCSInstallationsConfig{
				expectedProvider:  "github",
				expectedPageSize:  100,
				expectedPageToken: nil,
				result: &backend.VCSInstallationPage{
					Data:     &mockInstallations,
					Metadata: nil,
				},
				err: nil,
			},
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/github/installations", nil),
			expectedResponse: testJSONResponse(http.StatusOK, `{
				"data": [
					{
						"account_login": "GoogleChrome",
						"account_type": "Organization",
						"id": "inst-123",
						"vcs_installation_id": "12345",
						"vcs_provider": "github"
					}
				]
			}`),
		},
		{
			name:                 "Invalid Page Token",
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			cfg: &MockListVCSInstallationsConfig{
				expectedProvider:  "github",
				expectedPageSize:  100,
				expectedPageToken: &invalidPageToken,
				result:            nil,
				err:               backendtypes.ErrInvalidPageToken,
			},
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/github/installations?page_token=invalid-token", nil),
			expectedResponse: testJSONResponse(http.StatusBadRequest, `{
				"code": 400,
				"message": "invalid page token"
			}`),
		},
		{
			name:                 "Unsupported Provider",
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			cfg: &MockListVCSInstallationsConfig{
				expectedProvider:  "unknown",
				expectedPageSize:  100,
				expectedPageToken: nil,
				result:            nil,
				err:               backendtypes.ErrUnsupportedVCSProvider,
			},
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/unknown/installations", nil),
			expectedResponse: testJSONResponse(http.StatusBadRequest, `{
				"code": 400,
				"message": "unsupported VCS provider: unknown"
			}`),
		},
		{
			name:                 "Database Error",
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			cfg: &MockListVCSInstallationsConfig{
				expectedProvider:  "github",
				expectedPageSize:  100,
				expectedPageToken: nil,
				result:            nil,
				err:               errors.New("db error"),
			},
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/github/installations", nil),
			expectedResponse: testJSONResponse(http.StatusInternalServerError, `{
				"code": 500,
				"message": "failed to list VCS installations"
			}`),
		},
		{
			name:                 "Unauthenticated",
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(nil)),
			cfg:                  nil,
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/github/installations", nil),
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
					listVCSInstallationsCfg: tc.cfg,
					t:                       t,
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
