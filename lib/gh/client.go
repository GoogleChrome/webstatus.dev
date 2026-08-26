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
	"net/http"
	"net/url"

	"github.com/google/go-github/v79/github"
)

type ClientOption func(*github.Client)

func WithBaseURL(baseURL *url.URL) ClientOption {
	return func(c *github.Client) {
		c.BaseURL = baseURL
		c.UploadURL = baseURL
	}
}

type RepoClient interface {
	GetLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error)
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
