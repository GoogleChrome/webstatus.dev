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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
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

// TestClient_DownloadTarball_ArchiveLinkErrorClassification exercises DownloadTarball against a real
// go-github client so that the error shape returned by GetArchiveLink for unexpected HTTP statuses
// (a plain error rather than a *github.ErrorResponse) is covered end-to-end.
func TestClient_DownloadTarball_ArchiveLinkErrorClassification(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		statusCode    int
		headers       map[string]string
		body          string
		wantPermanent bool
		wantRateLimit bool
		wantServer    bool
	}{
		{
			name:       "403 repository access blocked is permanent",
			statusCode: http.StatusForbidden,
			headers:    map[string]string{"X-RateLimit-Remaining": "4999"},
			body: `{"message":"Repository access blocked",` +
				`"block":{"reason":"tos","created_at":"2022-08-12T17:28:06Z"}}`,
			wantPermanent: true,
			wantRateLimit: false,
			wantServer:    false,
		},
		{
			name:          "404 not found is permanent",
			statusCode:    http.StatusNotFound,
			headers:       nil,
			body:          `{"message":"Not Found"}`,
			wantPermanent: true,
			wantRateLimit: false,
			wantServer:    false,
		},
		{
			name:          "451 unavailable for legal reasons is permanent",
			statusCode:    http.StatusUnavailableForLegalReasons,
			headers:       nil,
			body:          `{"message":"Repository access blocked"}`,
			wantPermanent: true,
			wantRateLimit: false,
			wantServer:    false,
		},
		{
			name:          "403 primary rate limit is retryable",
			statusCode:    http.StatusForbidden,
			headers:       map[string]string{"X-RateLimit-Remaining": "0"},
			body:          `{"message":"API rate limit exceeded"}`,
			wantPermanent: false,
			wantRateLimit: true,
			wantServer:    false,
		},
		{
			name:          "403 secondary rate limit is retryable",
			statusCode:    http.StatusForbidden,
			headers:       map[string]string{"Retry-After": "60"},
			body:          `{"message":"You have exceeded a secondary rate limit"}`,
			wantPermanent: false,
			wantRateLimit: true,
			wantServer:    false,
		},
		{
			name:          "429 too many requests is retryable",
			statusCode:    http.StatusTooManyRequests,
			headers:       nil,
			body:          `{"message":"Too Many Requests"}`,
			wantPermanent: false,
			wantRateLimit: true,
			wantServer:    false,
		},
		{
			name:          "500 internal server error is retryable",
			statusCode:    http.StatusInternalServerError,
			headers:       nil,
			body:          `{"message":"Server Error"}`,
			wantPermanent: false,
			wantRateLimit: false,
			wantServer:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/repos/owner/repo/tarball/main" {
					t.Errorf("unexpected request path: %s", r.URL.Path)
				}
				for k, v := range tc.headers {
					w.Header().Set(k, v)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			baseURL, err := url.Parse(srv.URL + "/")
			if err != nil {
				t.Fatalf("failed to parse test server url: %v", err)
			}

			c := NewClientWithHTTPClient(srv.Client(), "test-token", WithBaseURL(baseURL))

			stream, err := c.DownloadTarball(context.Background(), "owner", "repo", "main")
			if err == nil {
				_ = stream.Close()
				t.Fatal("expected error, got nil")
			}

			var ghErr *github.ErrorResponse
			if !errors.As(err, &ghErr) {
				t.Fatalf("expected error to wrap *github.ErrorResponse, got %T: %v", err, err)
			}
			if ghErr.Response.StatusCode != tc.statusCode {
				t.Errorf("ErrorResponse status = %d, want %d", ghErr.Response.StatusCode, tc.statusCode)
			}
			if got := IsPermanentClientError(err); got != tc.wantPermanent {
				t.Errorf("IsPermanentClientError() = %v, want %v (err: %v)", got, tc.wantPermanent, err)
			}
			if got := IsRateLimitError(err); got != tc.wantRateLimit {
				t.Errorf("IsRateLimitError() = %v, want %v (err: %v)", got, tc.wantRateLimit, err)
			}
			if got := IsServerError(err); got != tc.wantServer {
				t.Errorf("IsServerError() = %v, want %v (err: %v)", got, tc.wantServer, err)
			}
		})
	}
}

// newTestGitHubClient returns a Client whose GitHub API base URL points at srv.
func newTestGitHubClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()

	baseURL, err := url.Parse(srv.URL + "/")
	if err != nil {
		t.Fatalf("failed to parse test server url: %v", err)
	}

	return NewClientWithHTTPClient(srv.Client(), "test-token", WithBaseURL(baseURL))
}

// TestClient_DownloadTarball_FollowsRenamedRepositoryRedirect verifies that DownloadTarball
// follows the 301 GitHub returns for the old name of a renamed / transferred repository.
func TestClient_DownloadTarball_FollowsRenamedRepositoryRedirect(t *testing.T) {
	t.Parallel()

	tarData := []byte("renamed-repo-tarball")
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Old name -> canonical repository ID URL (what GitHub returns for a renamed repo).
	mux.HandleFunc("/repos/owner/old-name/tarball/main", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL+"/repositories/42/tarball/main", http.StatusMovedPermanently)
	})
	// Canonical URL -> archive download location.
	mux.HandleFunc("/repositories/42/tarball/main", func(w http.ResponseWriter, r *http.Request) {
		// The followed request must still be authenticated, otherwise private renamed
		// repositories would 404 and be dropped as permanent failures.
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("followed redirect Authorization = %q, want %q", got, "Bearer test-token")
		}
		http.Redirect(w, r, srv.URL+"/download/archive.tar.gz", http.StatusFound)
	})
	mux.HandleFunc("/download/archive.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(tarData)
	})

	stream, err := newTestGitHubClient(t, srv).DownloadTarball(context.Background(), "owner", "old-name", "main")
	if err != nil {
		t.Fatalf("DownloadTarball failed: %v", err)
	}
	defer func() {
		_ = stream.Close()
	}()

	readBytes, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("failed to read from stream: %v", err)
	}
	if !bytes.Equal(readBytes, tarData) {
		t.Fatalf("expected %q, got %q", tarData, readBytes)
	}
}

// TestClient_DownloadTarball_UnfollowedRedirectsArePermanent verifies that redirects DownloadTarball
// cannot follow are classified as permanent (ErrRedirectNotFollowed) instead of being retried forever.
func TestClient_DownloadTarball_UnfollowedRedirectsArePermanent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		// newHandler builds the GitHub API handler. hops counts requests received.
		newHandler func(hops *atomic.Int32) http.Handler
		// wantHops is the expected number of requests, or 0 to skip the check.
		wantHops int32
	}{
		{
			name: "redirect chain longer than the limit",
			newHandler: func(hops *atomic.Int32) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					n := hops.Add(1)
					http.Redirect(w, r, fmt.Sprintf("/hop/%d", n), http.StatusMovedPermanently)
				})
			},
			// The initial request plus maxArchiveLinkRedirects followed redirects.
			wantHops: maxArchiveLinkRedirects + 1,
		},
		{
			name: "307 temporary redirect (not followed by go-github)",
			newHandler: func(hops *atomic.Int32) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					hops.Add(1)
					http.Redirect(w, r, "/elsewhere", http.StatusTemporaryRedirect)
				})
			},
			wantHops: 1,
		},
		{
			name: "301 without a Location header",
			newHandler: func(hops *atomic.Int32) http.Handler {
				mux := http.NewServeMux()
				mux.HandleFunc("/repos/owner/repo/tarball/main", func(w http.ResponseWriter, _ *http.Request) {
					hops.Add(1)
					w.WriteHeader(http.StatusMovedPermanently)
				})
				// go-github follows an empty Location to the base URL, like the GitHub API root.
				mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
					hops.Add(1)
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{}`))
				})

				return mux
			},
			wantHops: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var hops atomic.Int32
			srv := httptest.NewServer(tc.newHandler(&hops))
			defer srv.Close()

			stream, err := newTestGitHubClient(t, srv).DownloadTarball(context.Background(), "owner", "repo", "main")
			if err == nil {
				_ = stream.Close()
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, ErrRedirectNotFollowed) {
				t.Errorf("expected ErrRedirectNotFollowed, got %v", err)
			}
			if !IsPermanentClientError(err) {
				t.Errorf("IsPermanentClientError() = false, want true (err: %v)", err)
			}
			if IsRateLimitError(err) {
				t.Errorf("IsRateLimitError() = true, want false (err: %v)", err)
			}
			if got := hops.Load(); tc.wantHops != 0 && got != tc.wantHops {
				t.Errorf("server saw %d requests, want %d", got, tc.wantHops)
			}
		})
	}
}
