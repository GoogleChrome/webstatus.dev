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

type UsersClient interface {
	ListEmails(ctx context.Context, opts *github.ListOptions) ([]*github.UserEmail, *github.Response, error)
	Get(ctx context.Context, user string) (*github.User, *github.Response, error)
}

type Client struct {
	repoClient RepoClient
}

// NewClient creates a new Github Client. If the token is not empty, it will
// use it as the auth token to make calls.
func NewClient(token string, opts ...ClientOption) *Client {
	ghClient := github.NewClient(nil)
	if token != "" {
		ghClient = ghClient.WithAuthToken(token)
	}
	for _, opt := range opts {
		opt(ghClient)
	}
	c := &Client{
		repoClient: ghClient.Repositories,
	}

	return c
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
