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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestListCodeSubscriptions(t *testing.T) {
	testUser := &auth.User{ID: "user-123", GitHubUserID: new("12345")}
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	sampleSub := backend.CodeSubscriptionResponse{
		Id:                 "sub-1",
		VcsProvider:        "github",
		VcsInstallationId:  new("inst-1"),
		VcsRepositoryId:    "repo-1",
		RepositoryOwner:    "GoogleChrome",
		RepositoryName:     "webstatus.dev",
		RepositoryFullName: "GoogleChrome/webstatus.dev",
		FeatureId:          new("view-transitions"),
		TargetQuery:        "id:view-transitions",
		Triggers:           []backend.SubscriptionTriggerWritable{backend.SubscriptionTriggerFeatureBaselineToWidely},
		RawDirective:       new("// @webstatus: id:view-transitions"),
		Occurrences: []backend.SubscriptionOccurrence{
			{FilePath: "src/app.ts", LineNumber: 10, CommentSnippet: "// @webstatus: id:view-transitions"},
		},
		OccurrenceCount: 1,
		Status:          backend.ACTIVE,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	samplePage := &backend.CodeSubscriptionPage{
		Data: &[]backend.CodeSubscriptionResponse{sampleSub},
		Metadata: &backend.PageMetadata{
			NextPageToken: new("next-page-tok"),
		},
	}

	testCases := []struct {
		name                 string
		cfg                  *MockListCodeSubscriptionsConfig
		expectedCallCount    int
		authMiddlewareOption testServerOption
		request              *http.Request
		expectedResponse     *http.Response
	}{
		{
			name: "Success with Pagination",
			cfg: &MockListCodeSubscriptionsConfig{
				expectedProvider:     "github",
				expectedRepositoryID: "repo-1",
				expectedPageSize:     10,
				expectedPageToken:    new("cur-tok"),
				output:               samplePage,
				err:                  nil,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/github/repositories/repo-1/code-subscriptions?page_size=10&page_token=cur-tok", nil),
			expectedResponse: testJSONResponse(http.StatusOK, `{
				"data": [
					{
						"id": "sub-1",
						"vcs_provider": "github",
						"vcs_installation_id": "inst-1",
						"vcs_repository_id": "repo-1",
						"repository_owner": "GoogleChrome",
						"repository_name": "webstatus.dev",
						"repository_full_name": "GoogleChrome/webstatus.dev",
						"feature_id": "view-transitions",
						"target_query": "id:view-transitions",
						"triggers": ["feature_baseline_to_widely"],
						"raw_directive": "// @webstatus: id:view-transitions",
						"occurrences": [
							{
								"file_path": "src/app.ts",
								"line_number": 10,
								"comment_snippet": "// @webstatus: id:view-transitions"
							}
						],
						"occurrence_count": 1,
						"status": "ACTIVE",
						"created_at": "2026-01-01T00:00:00Z",
						"updated_at": "2026-01-01T00:00:00Z"
					}
				],
				"metadata": {
					"next_page_token": "next-page-tok"
				}
			}`),
		},
		{
			name: "Invalid Page Token (400 Bad Request)",
			cfg: &MockListCodeSubscriptionsConfig{
				expectedProvider:     "github",
				expectedRepositoryID: "repo-1",
				expectedPageSize:     100,
				expectedPageToken:    new("invalid-tok"),
				output:               nil,
				err:                  backendtypes.ErrInvalidPageToken,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/github/repositories/repo-1/code-subscriptions?page_token=invalid-tok", nil),
			expectedResponse: testJSONResponse(http.StatusBadRequest, `{
				"code": 400,
				"message": "invalid page token"
			}`),
		},
		{
			name: "Unsupported VCS Provider (400 Bad Request)",
			cfg: &MockListCodeSubscriptionsConfig{
				expectedProvider:     "gitlab",
				expectedRepositoryID: "repo-1",
				expectedPageSize:     100,
				expectedPageToken:    nil,
				output:               nil,
				err:                  backendtypes.ErrUnsupportedVCSProvider,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/gitlab/repositories/repo-1/code-subscriptions", nil),
			expectedResponse: testJSONResponse(http.StatusBadRequest, `{
				"code": 400,
				"message": "unsupported VCS provider: gitlab"
			}`),
		},
		{
			name: "Repository Not Found (404 Not Found)",
			cfg: &MockListCodeSubscriptionsConfig{
				expectedProvider:     "github",
				expectedRepositoryID: "repo-999",
				expectedPageSize:     100,
				expectedPageToken:    nil,
				output:               nil,
				err:                  backendtypes.ErrEntityDoesNotExist,
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/github/repositories/repo-999/code-subscriptions", nil),
			expectedResponse: testJSONResponse(http.StatusNotFound, `{
				"code": 404,
				"message": "repository not found"
			}`),
		},
		{
			name: "Database Error (500)",
			cfg: &MockListCodeSubscriptionsConfig{
				expectedProvider:     "github",
				expectedRepositoryID: "repo-1",
				expectedPageSize:     100,
				expectedPageToken:    nil,
				output:               nil,
				err:                  errors.New("spanner failure"),
			},
			expectedCallCount:    1,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(testUser)),
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/github/repositories/repo-1/code-subscriptions", nil),
			expectedResponse: testJSONResponse(http.StatusInternalServerError, `{
				"code": 500,
				"message": "failed to list code subscriptions"
			}`),
		},
		{
			name:                 "Missing User in Context",
			cfg:                  nil,
			expectedCallCount:    0,
			authMiddlewareOption: withAuthMiddleware(mockAuthMiddleware(nil)),
			request: httptest.NewRequestWithContext(t.Context(),
				http.MethodGet, "/v1/vcs/github/repositories/repo-1/code-subscriptions", nil),
			expectedResponse: testJSONResponse(http.StatusInternalServerError, `{
				"code": 500,
				"message": "internal server error"
			}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint:exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				listCodeSubscriptionsCfg: tc.cfg,
				t:                        t,
			}
			server := setupTestServer(t, withCustomStorer(mockStorer))
			assertTestServerRequest(t, server, tc.request, tc.expectedResponse, tc.authMiddlewareOption)
			assertMocksExpectations(t, tc.expectedCallCount, mockStorer.callCountListCodeSubscriptions,
				"ListCodeSubscriptions", nil)
		})
	}
}
