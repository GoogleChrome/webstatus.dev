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

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestGetSavedSearch(t *testing.T) {
	// testUser := &auth.User{
	// 	ID: "getSavedSearchUser",
	// }
	testCases := []struct {
		name                 string
		cfg                  *MockGetSavedSearchConfig
		authMiddlewareOption testServerOption
		request              *http.Request
		expectedResponse     *http.Response
	}{
		{
			name: "success unauthenticated",
			cfg: &MockGetSavedSearchConfig{
				expectedSavedSearchID: "saved-search-id",
				expectedUserID:        nil,
				output: &backend.SavedSearchResponse{
					Id:             "saved-search-id",
					CreatedAt:      time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt:      time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
					Name:           "test search",
					Query:          "test query",
					Description:    valuePtr("test description"),
					BookmarkStatus: nil,
					Permissions:    nil,
				},
				err: nil,
			},
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(nil)),
			request: httptest.NewRequest(http.MethodGet, "/v1/saved-searches/saved-search-id",
				nil),
			expectedResponse: testJSONResponse(200, `
			{
				"created_at":"2000-01-01T00:00:00Z",
				"description":"test description",
				"id":"saved-search-id",
				"name":"test search",
				"query":"test query",
				"updated_at":"2000-01-01T00:00:00Z"
			}`),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint:exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				getSavedSearchCfg: tc.cfg,
				t:                 t,
			}
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil,
				operationResponseCaches: nil, baseURL: getTestBaseURL(t)}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse,
				[]testServerOption{tc.authMiddlewareOption}...)
		})
	}
}
