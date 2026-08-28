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

package gh

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-github/v79/github"
)

type mockCollaboratorPermissionService struct {
	permissionLevel *github.RepositoryPermissionLevel
	response        *github.Response
	err             error
	calledUser      string
}

func (m *mockCollaboratorPermissionService) GetPermissionLevel(
	_ context.Context, _, _, user string,
) (*github.RepositoryPermissionLevel, *github.Response, error) {
	m.calledUser = user

	return m.permissionLevel, m.response, m.err
}

type mockUserLookupService struct {
	user     *github.User
	response *github.Response
	err      error
}

func (m *mockUserLookupService) GetByID(
	_ context.Context, _ int64,
) (*github.User, *github.Response, error) {
	return m.user, m.response, m.err
}

//nolint:exhaustruct
func TestGitHubPermissionChecker_HasRepositoryAdminAccess(t *testing.T) {
	adminStr := "admin"
	writeStr := "write"
	readStr := "read"
	loginOctocat := "octocat"
	testErr := errors.New("github api failure")

	testCases := []struct {
		name          string
		reposClient   CollaboratorPermissionService
		usersClient   UserLookupService
		owner         string
		repo          string
		githubUserID  string
		expectedAdmin bool
		expectedErr   error
		expectedUser  string
	}{
		{
			name: "Numeric GitHub ID resolves login and returns true for admin",
			reposClient: &mockCollaboratorPermissionService{
				permissionLevel: &github.RepositoryPermissionLevel{Permission: &adminStr},
				response:        &github.Response{Response: &http.Response{StatusCode: http.StatusOK}},
				err:             nil,
			},
			usersClient: &mockUserLookupService{
				user:     &github.User{Login: &loginOctocat},
				response: &github.Response{Response: &http.Response{StatusCode: http.StatusOK}},
				err:      nil,
			},
			owner:         "owner",
			repo:          "repo",
			githubUserID:  "12345",
			expectedAdmin: true,
			expectedErr:   nil,
			expectedUser:  "octocat",
		},
		{
			name: "Direct string login returns true for admin without calling usersClient",
			reposClient: &mockCollaboratorPermissionService{
				permissionLevel: &github.RepositoryPermissionLevel{Permission: &adminStr},
				response:        &github.Response{Response: &http.Response{StatusCode: http.StatusOK}},
				err:             nil,
			},
			usersClient:   nil,
			owner:         "owner",
			repo:          "repo",
			githubUserID:  "octocat",
			expectedAdmin: true,
			expectedErr:   nil,
			expectedUser:  "octocat",
		},
		{
			name: "Non-admin write permission returns false",
			reposClient: &mockCollaboratorPermissionService{
				permissionLevel: &github.RepositoryPermissionLevel{Permission: &writeStr},
				response:        &github.Response{Response: &http.Response{StatusCode: http.StatusOK}},
				err:             nil,
			},
			usersClient: &mockUserLookupService{
				user:     &github.User{Login: &loginOctocat},
				response: &github.Response{Response: &http.Response{StatusCode: http.StatusOK}},
				err:      nil,
			},
			owner:         "owner",
			repo:          "repo",
			githubUserID:  "12345",
			expectedAdmin: false,
			expectedErr:   nil,
			expectedUser:  "octocat",
		},
		{
			name: "Non-admin read permission returns false",
			reposClient: &mockCollaboratorPermissionService{
				permissionLevel: &github.RepositoryPermissionLevel{Permission: &readStr},
				response:        &github.Response{Response: &http.Response{StatusCode: http.StatusOK}},
				err:             nil,
			},
			usersClient:   nil,
			owner:         "owner",
			repo:          "repo",
			githubUserID:  "octocat",
			expectedAdmin: false,
			expectedErr:   nil,
			expectedUser:  "octocat",
		},
		{
			name: "404 Not Found from reposClient returns false without error",
			reposClient: &mockCollaboratorPermissionService{
				permissionLevel: nil,
				response:        &github.Response{Response: &http.Response{StatusCode: http.StatusNotFound}},
				err:             testErr,
			},
			usersClient:   nil,
			owner:         "owner",
			repo:          "repo",
			githubUserID:  "octocat",
			expectedAdmin: false,
			expectedErr:   nil,
			expectedUser:  "octocat",
		},
		{
			name: "404 Not Found from usersClient returns false without error",
			reposClient: &mockCollaboratorPermissionService{
				permissionLevel: nil,
				response:        nil,
				err:             nil,
			},
			usersClient: &mockUserLookupService{
				user:     nil,
				response: &github.Response{Response: &http.Response{StatusCode: http.StatusNotFound}},
				err:      testErr,
			},
			owner:         "owner",
			repo:          "repo",
			githubUserID:  "12345",
			expectedAdmin: false,
			expectedErr:   nil,
			expectedUser:  "",
		},
		{
			name:          "Empty owner returns false without error",
			reposClient:   &mockCollaboratorPermissionService{nil, nil, nil, ""},
			usersClient:   nil,
			owner:         "",
			repo:          "repo",
			githubUserID:  "12345",
			expectedAdmin: false,
			expectedErr:   nil,
			expectedUser:  "",
		},
		{
			name:          "Empty repo returns false without error",
			reposClient:   &mockCollaboratorPermissionService{nil, nil, nil, ""},
			usersClient:   nil,
			owner:         "owner",
			repo:          "",
			githubUserID:  "12345",
			expectedAdmin: false,
			expectedErr:   nil,
			expectedUser:  "",
		},
		{
			name:          "Empty githubUserID returns false without error",
			reposClient:   &mockCollaboratorPermissionService{nil, nil, nil, ""},
			usersClient:   nil,
			owner:         "owner",
			repo:          "repo",
			githubUserID:  "",
			expectedAdmin: false,
			expectedErr:   nil,
			expectedUser:  "",
		},
		{
			name:          "Nil reposClient returns ErrClientNotConfigured",
			reposClient:   nil,
			usersClient:   nil,
			owner:         "owner",
			repo:          "repo",
			githubUserID:  "12345",
			expectedAdmin: false,
			expectedErr:   ErrClientNotConfigured,
			expectedUser:  "",
		},
		{
			name: "API error from reposClient propagates error",
			reposClient: &mockCollaboratorPermissionService{
				permissionLevel: nil,
				response:        &github.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}},
				err:             testErr,
			},
			usersClient:   nil,
			owner:         "owner",
			repo:          "repo",
			githubUserID:  "octocat",
			expectedAdmin: false,
			expectedErr:   testErr,
			expectedUser:  "octocat",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checker := NewGitHubPermissionChecker(tc.reposClient, tc.usersClient)
			admin, err := checker.HasRepositoryAdminAccess(t.Context(), tc.owner, tc.repo, tc.githubUserID)

			if tc.expectedErr != nil {
				if err == nil || !errors.Is(err, tc.expectedErr) {
					t.Fatalf("expected error %v, got %v", tc.expectedErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if admin != tc.expectedAdmin {
				t.Errorf("expected admin %v, got %v", tc.expectedAdmin, admin)
			}

			if tc.expectedUser != "" {
				if mockRepo, ok := tc.reposClient.(*mockCollaboratorPermissionService); ok {
					if mockRepo.calledUser != tc.expectedUser {
						t.Errorf("expected called user %q, got %q", tc.expectedUser, mockRepo.calledUser)
					}
				}
			}
		})
	}
}

func TestNewGitHubPermissionCheckerWithBaseURL(t *testing.T) {
	baseURL, err := url.Parse("https://custom.github.com/api/v3")
	if err != nil {
		t.Fatalf("failed to parse url: %v", err)
	}

	checker := NewGitHubPermissionCheckerWithBaseURL(baseURL)
	if checker == nil {
		t.Fatal("expected non-nil checker")
	}
	if checker.reposClient == nil {
		t.Error("expected non-nil reposClient")
	}
	if checker.usersClient == nil {
		t.Error("expected non-nil usersClient")
	}
}

func TestNewGitHubPermissionCheckerWithTokenProvider(t *testing.T) {
	baseURL, err := url.Parse("https://custom.github.com/api/v3")
	if err != nil {
		t.Fatalf("failed to parse url: %v", err)
	}

	checker := NewGitHubPermissionCheckerWithTokenProvider(nil, baseURL)
	if checker == nil {
		t.Fatal("expected non-nil checker")
	}
	if checker.reposClient == nil {
		t.Error("expected non-nil reposClient")
	}
	if checker.usersClient == nil {
		t.Error("expected non-nil usersClient")
	}
}

func TestGitHubPermissionChecker_WithTokenProvider_Integration(t *testing.T) {
	t.Parallel()

	_, privPEM := generateTestRSAPEM(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Verify authorization header on all requests
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			t.Errorf("missing Authorization header on path: %s", r.URL.Path)
		}

		switch r.URL.Path {
		case "/repos/admin-org/admin-repo/installation":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 789}`))
		case "/repos/unknown-org/unknown-repo/installation":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "Not Found"}`))
		case "/app/installations/789/access_tokens":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(testTokenResp{
				Token:     mockInstToken1,
				ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
			})
		case "/user/101":
			// Numeric user ID lookup
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"login": "admin-user"}`))
		case "/user/202":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"login": "viewer-user"}`))
		case "/repos/admin-org/admin-repo/collaborators/admin-user/permission":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"permission": "admin"}`))
		case "/repos/admin-org/admin-repo/collaborators/viewer-user/permission":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"permission": "read"}`))
		default:
			t.Errorf("unexpected test request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tp, err := NewTokenProvider("app-99", privPEM, newMockTokenCacher())
	if err != nil {
		t.Fatalf("NewTokenProvider failed: %v", err)
	}
	tp.baseURL = server.URL
	tp.httpClient = server.Client()

	serverURL, _ := url.Parse(server.URL)
	checker := NewGitHubPermissionCheckerWithTokenProvider(tp, serverURL)

	// 1. Admin user with numeric ID -> true
	isAdmin, err := checker.HasRepositoryAdminAccess(context.Background(), "admin-org", "admin-repo", "101")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isAdmin {
		t.Errorf("expected admin=true, got false")
	}

	// 2. Viewer user with numeric ID -> false
	isViewerAdmin, err := checker.HasRepositoryAdminAccess(context.Background(), "admin-org", "admin-repo", "202")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isViewerAdmin {
		t.Errorf("expected admin=false, got true")
	}

	// 3. Unknown repo (not installed) -> false, nil
	isUnknownAdmin, err := checker.HasRepositoryAdminAccess(context.Background(), "unknown-org", "unknown-repo", "101")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isUnknownAdmin {
		t.Errorf("expected admin=false for uninstalled repo, got true")
	}
}
