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

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gh"
)

func testEmptyJSONObjectBody() io.Reader {
	return strings.NewReader("{}")
}

func TestPingUser(t *testing.T) {
	testCases := []struct {
		name                   string
		authMiddleware         func(http.Handler) http.Handler
		body                   io.Reader
		getCurrentUserCfg      *mockGetCurrentUserConfig
		listEmailsCfg          *mockListEmailsConfig
		syncUserProfileInfoCfg *MockSyncUserProfileInfoConfig
		expectedResponse       *http.Response
	}{
		{
			name:                   "success no github access token in body",
			body:                   testEmptyJSONObjectBody(),
			authMiddleware:         mockAuthMiddleware(createTestID1User()),
			getCurrentUserCfg:      nil,
			listEmailsCfg:          nil,
			syncUserProfileInfoCfg: nil,
			expectedResponse:       createEmptyBodyResponse(http.StatusNoContent),
		},
		{
			name:                   "no user",
			authMiddleware:         mockAuthMiddleware(nil),
			body:                   testEmptyJSONObjectBody(),
			getCurrentUserCfg:      nil,
			listEmailsCfg:          nil,
			syncUserProfileInfoCfg: nil,
			expectedResponse: testJSONResponse(http.StatusInternalServerError, `{
				"code": 500,
				"message": "internal server error"
			}`),
		},
		{
			name: "github user id missing from token",
			body: strings.NewReader(`{"github_token": "foo"}`),
			authMiddleware: mockAuthMiddleware(&auth.User{
				ID:           "hi",
				GitHubUserID: nil,
			}),
			getCurrentUserCfg:      nil,
			listEmailsCfg:          nil,
			syncUserProfileInfoCfg: nil,
			expectedResponse: testJSONResponse(http.StatusInternalServerError, `{
				"code": 500,
				"message": "token is missing github user id"
			}`),
		},
		{
			name: "failed to get github user",
			body: strings.NewReader(`{"github_token": "foo"}`),
			authMiddleware: mockAuthMiddleware(&auth.User{
				ID:           "hi",
				GitHubUserID: valuePtr("123456"),
			}),
			getCurrentUserCfg: &mockGetCurrentUserConfig{
				err:  errTest,
				user: nil,
			},
			listEmailsCfg:          nil,
			syncUserProfileInfoCfg: nil,
			expectedResponse: testJSONResponse(http.StatusInternalServerError, `{
				"code": 500,
				"message": "failed to get GitHub user"
			}`),
		},
		{
			name: "user does not match id from github",
			body: strings.NewReader(`{"github_token": "foo"}`),
			authMiddleware: mockAuthMiddleware(&auth.User{
				ID:           "hi",
				GitHubUserID: valuePtr("123456"),
			}),
			getCurrentUserCfg: &mockGetCurrentUserConfig{
				err: nil,
				user: &gh.GitHubUser{
					ID:       12345,
					Username: "username",
				},
			},
			listEmailsCfg:          nil,
			syncUserProfileInfoCfg: nil,
			expectedResponse: testJSONResponse(http.StatusForbidden, `{
				"code": 403,
				"message": "user does not match specified GitHub User ID"
			}`),
		},
		{
			name: "user does not match id from github",
			body: strings.NewReader(`{"github_token": "foo"}`),
			authMiddleware: mockAuthMiddleware(&auth.User{
				ID:           "hi",
				GitHubUserID: valuePtr("123456"),
			}),
			getCurrentUserCfg: &mockGetCurrentUserConfig{
				err: nil,
				user: &gh.GitHubUser{
					ID:       12345,
					Username: "username",
				},
			},
			listEmailsCfg:          nil,
			syncUserProfileInfoCfg: nil,
			expectedResponse: testJSONResponse(http.StatusForbidden, `{
				"code": 403,
				"message": "user does not match specified GitHub User ID"
			}`),
		},
		{
			name: "failed to list GitHub emails",
			body: strings.NewReader(`{"github_token": "foo"}`),
			authMiddleware: mockAuthMiddleware(&auth.User{
				ID:           "hi",
				GitHubUserID: valuePtr("123456"),
			}),
			getCurrentUserCfg: &mockGetCurrentUserConfig{
				err: nil,
				user: &gh.GitHubUser{
					ID:       123456,
					Username: "username",
				},
			},
			listEmailsCfg: &mockListEmailsConfig{
				err:    errTest,
				emails: nil,
			},
			syncUserProfileInfoCfg: nil,
			expectedResponse: testJSONResponse(http.StatusInternalServerError, `{
				"code": 500,
				"message": "failed to list GitHub emails"
			}`),
		},
		{
			name: "sync profile failed",
			body: strings.NewReader(`{"github_token": "foo"}`),
			authMiddleware: mockAuthMiddleware(&auth.User{
				ID:           "hi",
				GitHubUserID: valuePtr("123456"),
			}),
			getCurrentUserCfg: &mockGetCurrentUserConfig{
				err: nil,
				user: &gh.GitHubUser{
					ID:       123456,
					Username: "username",
				},
			},
			listEmailsCfg: &mockListEmailsConfig{
				err: nil,
				emails: []*gh.UserEmail{
					{
						Email:    "email1",
						Verified: true,
					},
					{
						Email:    "email2",
						Verified: false,
					},
					{
						Email:    "email3",
						Verified: true,
					},
				},
			},
			syncUserProfileInfoCfg: &MockSyncUserProfileInfoConfig{
				expectedUserProfile: backendtypes.UserProfile{
					UserID:       "hi",
					GitHubUserID: 123456,
					Emails: []string{
						"email1",
						"email3",
					},
				},
				err: errTest,
			},
			expectedResponse: testJSONResponse(http.StatusInternalServerError, `{
				"code": 500,
				"message": "failed to sync user profile"
			}`),
		},
		{
			name: "sync profile success",
			body: strings.NewReader(`{"github_token": "foo"}`),
			authMiddleware: mockAuthMiddleware(&auth.User{
				ID:           "hi",
				GitHubUserID: valuePtr("123456"),
			}),
			getCurrentUserCfg: &mockGetCurrentUserConfig{
				err: nil,
				user: &gh.GitHubUser{
					ID:       123456,
					Username: "username",
				},
			},
			listEmailsCfg: &mockListEmailsConfig{
				err: nil,
				emails: []*gh.UserEmail{
					{
						Email:    "email1",
						Verified: true,
					},
					{
						Email:    "email2",
						Verified: false,
					},
					{
						Email:    "email3",
						Verified: true,
					},
				},
			},
			syncUserProfileInfoCfg: &MockSyncUserProfileInfoConfig{
				expectedUserProfile: backendtypes.UserProfile{
					UserID:       "hi",
					GitHubUserID: 123456,
					Emails: []string{
						"email1",
						"email3",
					},
				},
				err: nil,
			},
			expectedResponse: createEmptyBodyResponse(http.StatusNoContent),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			authMiddlewareOption := withAuthMiddleware(tc.authMiddleware)
			myServer := Server{
				// nolint:exhaustruct
				wptMetricsStorer: &MockWPTMetricsStorer{t: t, syncUserProfileInfoCfg: tc.syncUserProfileInfoCfg},
				metadataStorer:   nil,
				userGitHubClientFactory: setupMockGitHubUserClient(
					t,
					"foo",
					tc.getCurrentUserCfg,
					tc.listEmailsCfg,
				),
				operationResponseCaches: nil,
				baseURL:                 getTestBaseURL(t),
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/users/me/ping", tc.body)
			req.Header.Set("Content-Type", "application/json")
			assertTestServerRequest(t, &myServer, req, tc.expectedResponse, authMiddlewareOption)
		})
	}
}
