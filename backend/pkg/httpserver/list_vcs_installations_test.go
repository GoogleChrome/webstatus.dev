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
//nolint:dupl // Similar structure across simple listing handlers
package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
)

func TestListVCSInstallations(t *testing.T) {
	testUser := &auth.User{ID: "user-123", GitHubUserID: new("12345")}

	testCases := []struct {
		name                 string
		authMiddlewareOption testServerOption
		request              *http.Request
		expectedResponse     *http.Response
	}{
		{
			name:                 "Success",
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/github/installations", nil),
			expectedResponse: testJSONResponse(http.StatusOK, `{
				"data": []
			}`),
		},
		{
			name:                 "Unauthenticated",
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(nil)),
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
			server := setupTestServer(t)
			assertTestServerRequest(t, server, tc.request, tc.expectedResponse, tc.authMiddlewareOption)
		})
	}
}
