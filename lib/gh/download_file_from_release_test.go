// Copyright 2024 Google LLC
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
	"io"
	"net/http"
	"testing"

	"github.com/google/go-github/v60/github"
)

func checkIfFileIsReadable(t *testing.T, file io.Reader) {
	bytes, err := io.ReadAll(file)
	if err != nil {
		t.Errorf("unable to read file. %s", err.Error())
	}
	if len(bytes) == 0 {
		t.Error("file is empty")
	}
}

func TestDownloadFileFromReleaseWebFeatures(t *testing.T) {
	t.Skip("Used for debugging purposes.")
	client := NewClient("")
	ctx := context.Background()
	httpClient := http.DefaultClient
	file, err := client.DownloadFileFromRelease(ctx, "web-platform-dx", "web-features", httpClient, "data.json")
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	} else {
		// Close to be safe at the end.
		defer file.Close()
		checkIfFileIsReadable(t, file)
	}
}

func TestDownloadFileFromReleaseBrowserCompatData(t *testing.T) {
	t.Skip("Used for debugging purposes.")
	t.Skip("Cannot remove until https://github.com/mdn/browser-compat-data/issues/22675 is fixed")
	client := NewClient("")
	ctx := context.Background()
	httpClient := http.DefaultClient
	file, err := client.DownloadFileFromRelease(ctx, "mdn", "browser-compat-data", httpClient, "data.json")
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	} else {
		// Close to be safe at the end.
		defer file.Close()
		checkIfFileIsReadable(t, file)
	}
}

type mockGetLatestReleaseConfig struct {
	expectedOwner string
	expectedRepo  string
	release       *github.RepositoryRelease
	err           error
}

type mockRepoClient struct {
	t                       *testing.T
	mockGetLatestReleaseCfg mockGetLatestReleaseConfig
}

func (c mockRepoClient) GetLatestRelease(
	_ context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error) {
	if c.mockGetLatestReleaseCfg.expectedOwner != owner || c.mockGetLatestReleaseCfg.expectedRepo != repo {
		c.t.Error("unexpected input to GetLatestRelease")
	}

	return c.mockGetLatestReleaseCfg.release, nil, c.mockGetLatestReleaseCfg.err
}

type mockRoundTripperConfig struct {
	expectedURL string
	resp        *http.Response
	err         error
}

type mockRoundTripper struct {
	t   *testing.T
	cfg *mockRoundTripperConfig
}

func (rt *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.String() != rt.cfg.expectedURL {
		rt.t.Error("unexpected url when downloading")
	}

	return rt.cfg.resp, rt.cfg.err
}

func valuePtr[T any](in T) *T { return &in }

func TestMockDownloadFileFromRelease(t *testing.T) {
	testCases := []struct {
		name          string
		cfg           mockGetLatestReleaseConfig
		roundTripCfg  *mockRoundTripperConfig
		expectedError error
	}{
		{
			name: "successful download",
			cfg: mockGetLatestReleaseConfig{
				expectedOwner: "owner",
				expectedRepo:  "repo",
				//nolint: exhaustruct
				release: &github.RepositoryRelease{
					Assets: []*github.ReleaseAsset{
						{
							Name:               valuePtr("file.txt"),
							BrowserDownloadURL: valuePtr("http://example.com/file.txt"),
						},
					},
				},
				err: nil,
			},
			roundTripCfg: &mockRoundTripperConfig{
				expectedURL: "http://example.com/file.txt",
				//nolint: exhaustruct
				resp: &http.Response{
					StatusCode: http.StatusOK,
				},
				err: nil,
			},
			expectedError: nil,
		},
		{
			name: "rate limit with github api",
			cfg: mockGetLatestReleaseConfig{
				expectedOwner: "owner",
				expectedRepo:  "repo",
				release:       nil,
				//nolint: exhaustruct
				err: &github.RateLimitError{},
			},
			roundTripCfg:  nil,
			expectedError: ErrRateLimit,
		},
		{
			name: "unknown error with github api",
			cfg: mockGetLatestReleaseConfig{
				expectedOwner: "owner",
				expectedRepo:  "repo",
				release:       nil,
				err:           errors.New("something went wrong"),
			},
			roundTripCfg:  nil,
			expectedError: ErrFatalError,
		},
		{
			name: "missing asset",
			cfg: mockGetLatestReleaseConfig{
				expectedOwner: "owner",
				expectedRepo:  "repo",
				//nolint: exhaustruct
				release: &github.RepositoryRelease{},
				err:     nil,
			},
			roundTripCfg:  nil,
			expectedError: ErrAssetNotFound,
		},
		{
			name: "request.DO() fails",
			cfg: mockGetLatestReleaseConfig{
				expectedOwner: "owner",
				expectedRepo:  "repo",
				//nolint: exhaustruct
				release: &github.RepositoryRelease{
					Assets: []*github.ReleaseAsset{
						{
							Name:               valuePtr("file.txt"),
							BrowserDownloadURL: valuePtr("http://example.com/file.txt"),
						},
					},
				},
				err: nil,
			},
			roundTripCfg: &mockRoundTripperConfig{
				expectedURL: "http://example.com/file.txt",
				//nolint: exhaustruct
				resp: nil,
				err:  errors.New("something went wrong"),
			},
			expectedError: ErrUnableToDownloadAsset,
		},
		{
			name: "failed to download",
			cfg: mockGetLatestReleaseConfig{
				expectedOwner: "owner",
				expectedRepo:  "repo",
				//nolint: exhaustruct
				release: &github.RepositoryRelease{
					Assets: []*github.ReleaseAsset{
						{
							Name:               valuePtr("file.txt"),
							BrowserDownloadURL: valuePtr("http://example.com/file.txt"),
						},
					},
				},
				err: nil,
			},
			roundTripCfg: &mockRoundTripperConfig{
				expectedURL: "http://example.com/file.txt",
				//nolint: exhaustruct
				resp: &http.Response{
					StatusCode: http.StatusInternalServerError,
				},
				err: nil,
			},
			expectedError: ErrUnableToDownloadAsset,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := Client{
				repoClient: mockRepoClient{
					t:                       t,
					mockGetLatestReleaseCfg: tc.cfg,
				},
			}
			rt := mockRoundTripper{
				t:   t,
				cfg: tc.roundTripCfg,
			}

			httpClient := http.DefaultClient
			httpClient.Transport = &rt
			_, err := client.DownloadFileFromRelease(
				context.Background(),
				"owner",
				"repo",
				httpClient,
				"file.txt",
			)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error expected: %v received: %v", tc.expectedError, err)
			}
		})
	}
}
