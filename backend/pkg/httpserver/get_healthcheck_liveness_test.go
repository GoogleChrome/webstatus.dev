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
	"testing"
)

func TestGetHealthcheckLiveness(t *testing.T) {
	testCases := []struct {
		name             string
		request          *http.Request
		expectedResponse *http.Response
	}{
		{
			name: "success",
			request: httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"/v1/healthchecks/liveness",
				nil,
			),
			expectedResponse: testJSONResponse(http.StatusOK, `{"status":"healthy"}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := setupTestServer(t)
			assertTestServerRequest(t, server, tc.request, tc.expectedResponse)
		})
	}
}
