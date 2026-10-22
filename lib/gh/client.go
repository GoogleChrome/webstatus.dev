// Copyright 2023 Google LLC
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
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/google/go-github/v79/github"
)

type ClientOption func(*github.Client)

// maxArchiveLinkRedirects is the number of 301 Moved Permanently responses followed when
// requesting a repository archive link. GitHub returns a 301 for the old name of a renamed
// or transferred repository, which can happen for scan tasks queued before the rename.
const maxArchiveLinkRedirects = 3

// ErrRedirectNotFollowed is returned when GitHub responds with a redirect that was not followed
// (for example, the redirect limit was exhausted). Retrying will not succeed, so callers should
// treat it as permanent (see IsPermanentClientError).
var ErrRedirectNotFollowed = errors.New("github redirect not followed")

func WithBaseURL(baseURL *url.URL) ClientOption {
	return func(c *github.Client) {
		c.BaseURL = baseURL
		c.UploadURL = baseURL
	}
}

type RepoClient interface {
	GetLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error)
	GetArchiveLink(
		ctx context.Context,
		owner, repo string,
		archiveformat github.ArchiveFormat,
		opts *github.RepositoryContentGetOptions,
		maxRedirects int,
	) (*url.URL, *github.Response, error)
}

type GitClient interface {
	GetTree(ctx context.Context, owner, repo, sha string, recursive bool) (*github.Tree, *github.Response, error)
	GetBlobRaw(ctx context.Context, owner, repo, sha string) ([]byte, *github.Response, error)
}

type IssuesClient interface {
	Create(ctx context.Context, owner, repo string, issue *github.IssueRequest) (*github.Issue, *github.Response, error)
}

type AppsClient interface {
	ListRepos(ctx context.Context, opts *github.ListOptions) (*github.ListRepositories, *github.Response, error)
	ListInstallations(ctx context.Context, opts *github.ListOptions) ([]*github.Installation, *github.Response, error)
}

type UsersClient interface {
	ListEmails(ctx context.Context, opts *github.ListOptions) ([]*github.UserEmail, *github.Response, error)
	Get(ctx context.Context, user string) (*github.User, *github.Response, error)
}

type Client struct {
	repoClient   RepoClient
	gitClient    GitClient
	issuesClient IssuesClient
	appsClient   AppsClient
	httpClient   *http.Client
}

// NewClient creates a new Github Client. If the token is not empty, it will
// use it as the auth token to make calls.
func NewClient(token string, opts ...ClientOption) *Client {
	return NewClientWithHTTPClient(nil, token, opts...)
}

// NewClientWithHTTPClient creates a new Github Client with a custom HTTP client.
func NewClientWithHTTPClient(httpClient *http.Client, token string, opts ...ClientOption) *Client {
	ghClient := github.NewClient(httpClient)
	if token != "" {
		ghClient = ghClient.WithAuthToken(token)
	}
	for _, opt := range opts {
		opt(ghClient)
	}

	return &Client{
		repoClient:   ghClient.Repositories,
		gitClient:    ghClient.Git,
		issuesClient: ghClient.Issues,
		appsClient:   ghClient.Apps,
		httpClient:   httpClient,
	}
}

// ListAppInstallations returns all installations of the authenticated GitHub App.
func (c *Client) ListAppInstallations(
	ctx context.Context,
	opts *github.ListOptions,
) ([]*github.Installation, error) {
	if c.appsClient == nil {
		return nil, errors.New("apps client not initialized")
	}
	result, _, err := c.appsClient.ListInstallations(ctx, opts)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ListInstallationRepositories returns all repositories accessible to the installation.
func (c *Client) ListInstallationRepositories(
	ctx context.Context,
	opts *github.ListOptions,
) ([]*github.Repository, error) {
	if c.appsClient == nil {
		return nil, errors.New("apps client not initialized")
	}
	result, _, err := c.appsClient.ListRepos(ctx, opts)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	return result.Repositories, nil
}

// DownloadTarball downloads a tarball archive stream (.tar.gz) for a repository at a given git ref.
// The caller is responsible for closing the returned ReadCloser.
func (c *Client) DownloadTarball(
	ctx context.Context,
	owner, repo, ref string,
) (io.ReadCloser, error) {
	if c.repoClient == nil {
		return nil, errors.New("repo client not initialized")
	}

	opts := &github.RepositoryContentGetOptions{Ref: ref}
	archiveURL, resp, err := c.repoClient.GetArchiveLink(ctx, owner, repo, github.Tarball, opts, maxArchiveLinkRedirects)
	if err != nil {
		if resp != nil && resp.Response != nil && resp.Body != nil && resp.StatusCode == http.StatusOK {
			return resp.Body, nil
		}

		if resp != nil && resp.Response != nil {
			// go-github's GetArchiveLink reports unexpected HTTP statuses as a plain
			// fmt.Errorf("unexpected status code: ...") instead of a *github.ErrorResponse.
			// Without the HTTP response attached, IsClientError / IsPermanentClientError /
			// IsRateLimitError cannot classify the failure, so permanent errors (e.g. a
			// 403 "Repository access blocked" or a 404) would be retried forever.
			//nolint:exhaustruct // Only Response and Message required for error classification
			ghErr := &github.ErrorResponse{
				Response: resp.Response,
				Message:  err.Error(),
			}

			// A redirect that is still returned after following up to maxArchiveLinkRedirects
			// 301s (or a redirect status go-github does not follow) will not resolve on retry.
			if resp.StatusCode >= http.StatusMultipleChoices && resp.StatusCode < http.StatusBadRequest {
				return nil, fmt.Errorf("failed to get tarball archive link for %s/%s@%s: %w: %w",
					owner, repo, ref, ErrRedirectNotFollowed, ghErr)
			}

			err = ghErr
		}

		return nil, fmt.Errorf("failed to get tarball archive link for %s/%s@%s: %w", owner, repo, ref, err)
	}

	if archiveURL == nil {
		return nil, fmt.Errorf("received nil archive URL for %s/%s@%s", owner, repo, ref)
	}

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create tarball request: %w", err)
	}

	httpResp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download tarball stream: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		_ = httpResp.Body.Close()

		//nolint:exhaustruct // Only StatusCode and Message required for error classification
		return nil, fmt.Errorf("failed to download tarball: %w", &github.ErrorResponse{
			Response: httpResp,
			Message:  fmt.Sprintf("unexpected HTTP status %d", httpResp.StatusCode),
		})
	}

	return httpResp.Body, nil
}

// UserGitHubClient is a client that receives a token from a user that has installed our GitHub App.
// It uses that token to make requests on behalf of that user to verify things about them.
// It is different from the regular Client which is used for internal operations.
type UserGitHubClient struct {
	usersClient UsersClient
}

// NewUserGitHubClient creates a new UserGitHubClient with the given token.
// Assumes that the token is not empty.
func NewUserGitHubClient(token string, opts ...ClientOption) *UserGitHubClient {
	c := github.NewClient(nil).WithAuthToken(token)

	for _, opt := range opts {
		opt(c)
	}

	return &UserGitHubClient{
		usersClient: c.Users,
	}
}
