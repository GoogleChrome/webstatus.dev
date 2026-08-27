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
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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
			client:      &Client{repoClient: nil, gitClient: nil, issuesClient: nil, appsClient: nil, httpClient: nil},
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
					httpClient:   nil,
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
			client:      &Client{repoClient: nil, gitClient: nil, issuesClient: nil, appsClient: nil, httpClient: nil},
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
					httpClient:   nil,
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

type mockTarballRepoClient struct {
	getArchiveLinkFunc func(
		ctx context.Context,
		owner, repo string,
		archiveformat github.ArchiveFormat,
		opts *github.RepositoryContentGetOptions,
		maxRedirects int,
	) (*url.URL, *github.Response, error)
}

func (m *mockTarballRepoClient) GetLatestRelease(
	_ context.Context, _, _ string,
) (*github.RepositoryRelease, *github.Response, error) {
	return nil, nil, nil
}

func (m *mockTarballRepoClient) GetArchiveLink(
	ctx context.Context,
	owner, repo string,
	archiveformat github.ArchiveFormat,
	opts *github.RepositoryContentGetOptions,
	maxRedirects int,
) (*url.URL, *github.Response, error) {
	if m.getArchiveLinkFunc == nil {
		return nil, nil, nil
	}

	return m.getArchiveLinkFunc(ctx, owner, repo, archiveformat, opts, maxRedirects)
}

func TestClient_DownloadTarball(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("uninitialized repoClient returns error", func(t *testing.T) {
		t.Parallel()

		c := &Client{
			repoClient:   nil,
			gitClient:    nil,
			issuesClient: nil,
			appsClient:   nil,
			httpClient:   nil,
		}
		_, err := c.DownloadTarball(ctx, "owner", "repo", "main")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("GetArchiveLink error returns error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("archive link failed")
		repoMock := &mockTarballRepoClient{
			getArchiveLinkFunc: func(
				_ context.Context, _, _ string, _ github.ArchiveFormat, _ *github.RepositoryContentGetOptions, _ int,
			) (*url.URL, *github.Response, error) {
				return nil, nil, expectedErr
			},
		}

		c := &Client{
			repoClient:   repoMock,
			gitClient:    nil,
			issuesClient: nil,
			appsClient:   nil,
			httpClient:   nil,
		}
		_, err := c.DownloadTarball(ctx, "owner", "repo", "main")
		if err == nil || !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("successful streaming from archive link", func(t *testing.T) {
		t.Parallel()

		tarData := []byte("mock-tarball-stream-bytes")
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(tarData)
		}))
		defer srv.Close()

		srvURL, err := url.Parse(srv.URL)
		if err != nil {
			t.Fatalf("failed to parse test server url: %v", err)
		}

		repoMock := &mockTarballRepoClient{
			getArchiveLinkFunc: func(
				_ context.Context, owner, repo string, _ github.ArchiveFormat,
				opts *github.RepositoryContentGetOptions, _ int,
			) (*url.URL, *github.Response, error) {
				if owner != "GoogleChrome" || repo != "webstatus.dev" || opts.Ref != "main" {
					t.Fatalf("unexpected parameters: owner=%s, repo=%s, ref=%s", owner, repo, opts.Ref)
				}

				return srvURL, nil, nil
			},
		}

		c := &Client{
			repoClient:   repoMock,
			gitClient:    nil,
			issuesClient: nil,
			appsClient:   nil,
			httpClient:   srv.Client(),
		}

		stream, err := c.DownloadTarball(ctx, "GoogleChrome", "webstatus.dev", "main")
		if err != nil {
			t.Fatalf("DownloadTarball failed: %v", err)
		}
		defer func() {
			if closeErr := stream.Close(); closeErr != nil {
				t.Fatalf("failed to close stream: %v", closeErr)
			}
		}()

		readBytes, err := io.ReadAll(stream)
		if err != nil {
			t.Fatalf("failed to read from stream: %v", err)
		}
		if !bytes.Equal(readBytes, tarData) {
			t.Fatalf("expected %s, got %s", string(tarData), string(readBytes))
		}
	})
}
