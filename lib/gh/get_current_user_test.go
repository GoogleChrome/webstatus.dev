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

package gh

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/google/go-github/v75/github"
)

// MockUsersClient is a mock implementation of the UsersClient interface.
type MockUsersClient struct {
	GetFunc        func(ctx context.Context, user string) (*github.User, *github.Response, error)
	ListEmailsFunc func(ctx context.Context, opts *github.ListOptions) ([]*github.UserEmail, *github.Response, error)
}

func (m *MockUsersClient) Get(ctx context.Context, user string) (*github.User, *github.Response, error) {
	if m.GetFunc == nil {
		panic("GetFunc not set")
	}

	return m.GetFunc(ctx, user)
}

func (m *MockUsersClient) ListEmails(ctx context.Context, opts *github.ListOptions) (
	[]*github.UserEmail, *github.Response, error) {
	if m.ListEmailsFunc == nil {
		panic("ListEmailsFunc not set")
	}

	return m.ListEmailsFunc(ctx, opts)
}

var errTestAPI = errors.New("api error")

func createTestGithubResponse(nextPage int) *github.Response {
	// nolint:exhaustruct
	return &github.Response{
		// nolint:exhaustruct
		Response: &http.Response{StatusCode: http.StatusOK},
		NextPage: nextPage,
	}
}

func createTestGitHubUser(id int64, login string) *github.User {
	// nolint:exhaustruct
	return &github.User{
		ID:    valuePtr(id),
		Login: valuePtr(login),
	}
}

func TestUserGitHubClient_GetCurrentUser(t *testing.T) {
	type mockGetCurrentUserConfig struct {
		user *github.User
		err  error
	}
	tests := []struct {
		name          string
		cfg           *mockGetCurrentUserConfig
		expectedUser  *GitHubUser
		expectedError error
	}{
		{
			name: "Success",
			cfg: &mockGetCurrentUserConfig{
				user: createTestGitHubUser(12345, "testuser"),
				err:  nil,
			},
			expectedUser: &GitHubUser{
				ID:       12345,
				Username: "testuser",
			},
			expectedError: nil,
		},
		{
			name: "API Error",
			cfg: &mockGetCurrentUserConfig{
				user: nil,
				err:  errTestAPI,
			},
			expectedUser:  nil,
			expectedError: errTestAPI,
		},
		{
			name: "Nil ID",
			cfg: &mockGetCurrentUserConfig{
				// nolint:exhaustruct
				user: &github.User{ // Keep this inline to test nil ID specifically
					ID:    nil,
					Login: valuePtr("testuser"),
				},
				err: nil,
			},
			expectedUser:  nil,
			expectedError: ErrFieldUnexpectedlyNil,
		},
		{
			name: "Nil Login",
			cfg: &mockGetCurrentUserConfig{
				// nolint:exhaustruct
				user: &github.User{ // Keep this inline to test nil Login specifically
					ID:    valuePtr(int64(12345)),
					Login: nil,
				},
				err: nil,
			},
			expectedUser:  nil,
			expectedError: ErrFieldUnexpectedlyNil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockUsersClient{
				GetFunc: func(_ context.Context, _ string) (*github.User, *github.Response, error) {
					return tt.cfg.user, createTestGithubResponse(0), tt.cfg.err
				}, ListEmailsFunc: nil,
			}
			appClient := &UserGitHubClient{usersClient: mockClient}

			user, err := appClient.GetCurrentUser(context.Background())

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("GetCurrentUser() error = %v, wantErr %v", err, tt.expectedError)
			}
			if !reflect.DeepEqual(user, tt.expectedUser) {
				t.Errorf("GetCurrentUser() = %v, want %v", user, tt.expectedUser)
			}
		})
	}
}
