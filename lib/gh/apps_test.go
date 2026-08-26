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
	"errors"
	"testing"

	"github.com/google/go-github/v79/github"
)

type MockAppsClient struct {
	ListReposFunc func(
		ctx context.Context,
		opts *github.ListOptions,
	) (*github.ListRepositories, *github.Response, error)
	ListInstallationsFunc func(
		ctx context.Context,
		opts *github.ListOptions,
	) ([]*github.Installation, *github.Response, error)
}

func (m *MockAppsClient) ListRepos(
	ctx context.Context,
	opts *github.ListOptions,
) (*github.ListRepositories, *github.Response, error) {
	if m.ListReposFunc == nil {
		panic("ListReposFunc not set")
	}

	return m.ListReposFunc(ctx, opts)
}

func (m *MockAppsClient) ListInstallations(
	ctx context.Context,
	opts *github.ListOptions,
) ([]*github.Installation, *github.Response, error) {
	if m.ListInstallationsFunc == nil {
		panic("ListInstallationsFunc not set")
	}

	return m.ListInstallationsFunc(ctx, opts)
}

func TestClient_ListInstallationRepositories(t *testing.T) {
	t.Parallel()

	testErr := errors.New("github api error")
	repoName1 := "repo1"
	repoName2 := "repo2"
	repos := []*github.Repository{
		{Name: &repoName1},
		{Name: &repoName2},
	}

	testCases := []struct {
		name        string
		client      *Client
		mockFunc    func(ctx context.Context, opts *github.ListOptions) (*github.ListRepositories, *github.Response, error)
		expectedLen int
		wantErr     bool
	}{
		{
			name:        "uninitialized apps client returns error",
			client:      &Client{repoClient: nil, gitClient: nil, issuesClient: nil, appsClient: nil},
			mockFunc:    nil,
			expectedLen: 0,
			wantErr:     true,
		},
		{
			name:   "success returns repositories",
			client: nil, // set in test loop
			mockFunc: func(_ context.Context, _ *github.ListOptions) (*github.ListRepositories, *github.Response, error) {
				return &github.ListRepositories{
					TotalCount:   new(2),
					Repositories: repos,
				}, mockHTTPResponse(200), nil
			},
			expectedLen: 2,
			wantErr:     false,
		},
		{
			name:   "api error returns error",
			client: nil,
			mockFunc: func(_ context.Context, _ *github.ListOptions) (*github.ListRepositories, *github.Response, error) {
				return nil, mockHTTPResponse(500), testErr
			},
			expectedLen: 0,
			wantErr:     true,
		},
		{
			name:   "nil result returns nil",
			client: nil,
			mockFunc: func(_ context.Context, _ *github.ListOptions) (*github.ListRepositories, *github.Response, error) {
				return nil, mockHTTPResponse(200), nil
			},
			expectedLen: 0,
			wantErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := tc.client
			if client == nil {
				client = &Client{
					repoClient:   nil,
					gitClient:    nil,
					issuesClient: nil,
					appsClient:   &MockAppsClient{ListReposFunc: tc.mockFunc, ListInstallationsFunc: nil},
				}
			}

			result, err := client.ListInstallationRepositories(
				context.Background(),
				&github.ListOptions{Page: 1, PerPage: 100},
			)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ListInstallationRepositories() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && len(result) != tc.expectedLen {
				t.Errorf("ListInstallationRepositories() len = %d, want %d", len(result), tc.expectedLen)
			}
		})
	}
}

func TestClient_ListAppInstallations(t *testing.T) {
	t.Parallel()

	testErr := errors.New("github api error")
	instID1 := int64(1)
	instID2 := int64(2)
	installations := []*github.Installation{
		{ID: &instID1},
		{ID: &instID2},
	}

	testCases := []struct {
		name        string
		client      *Client
		mockFunc    func(ctx context.Context, opts *github.ListOptions) ([]*github.Installation, *github.Response, error)
		expectedLen int
		wantErr     bool
	}{
		{
			name:        "uninitialized apps client returns error",
			client:      &Client{repoClient: nil, gitClient: nil, issuesClient: nil, appsClient: nil},
			mockFunc:    nil,
			expectedLen: 0,
			wantErr:     true,
		},
		{
			name:   "success returns installations",
			client: nil,
			mockFunc: func(_ context.Context, _ *github.ListOptions) ([]*github.Installation, *github.Response, error) {
				return installations, mockHTTPResponse(200), nil
			},
			expectedLen: 2,
			wantErr:     false,
		},
		{
			name:   "api error returns error",
			client: nil,
			mockFunc: func(_ context.Context, _ *github.ListOptions) ([]*github.Installation, *github.Response, error) {
				return nil, mockHTTPResponse(500), testErr
			},
			expectedLen: 0,
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := tc.client
			if client == nil {
				client = &Client{
					repoClient:   nil,
					gitClient:    nil,
					issuesClient: nil,
					appsClient:   &MockAppsClient{ListReposFunc: nil, ListInstallationsFunc: tc.mockFunc},
				}
			}

			result, err := client.ListAppInstallations(
				context.Background(),
				&github.ListOptions{Page: 1, PerPage: 100},
			)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ListAppInstallations() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && len(result) != tc.expectedLen {
				t.Errorf("ListAppInstallations() len = %d, want %d", len(result), tc.expectedLen)
			}
		})
	}
}
