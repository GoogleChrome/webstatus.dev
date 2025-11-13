// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package httpserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testEmptyJSONObjectBody() io.Reader {
	return strings.NewReader("{}")
}

func TestPingUser(t *testing.T) {
	testCases := []struct {
		name             string
		authMiddleware   func(http.Handler) http.Handler
		expectedResponse *http.Response
	}{
		{
			name:             "success",
			authMiddleware:   mockAuthMiddleware(createTestID1User()),
			expectedResponse: createEmptyBodyResponse(http.StatusNoContent),
		},
		{
			name:           "no user",
			authMiddleware: mockAuthMiddleware(nil),
			expectedResponse: testJSONResponse(http.StatusInternalServerError, `{
				"code": 500,
				"message": "internal server error"
			}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			authMiddlewareOption := withAuthMiddleware(tc.authMiddleware)
			myServer := Server{
				wptMetricsStorer:        nil,
				metadataStorer:          nil,
				operationResponseCaches: nil,
				baseURL:                 getTestBaseURL(t),
			}
			req := httptest.NewRequest(http.MethodPost, "/v1/users/me/ping", testEmptyJSONObjectBody())
			assertTestServerRequest(t, &myServer, req, tc.expectedResponse, authMiddlewareOption)
		})
	}
}
