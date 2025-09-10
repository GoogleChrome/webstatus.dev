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

	"github.com/google/go-github/v74/github"
)

type RepoClient interface {
	GetLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error)
}

type Client struct {
	repoClient RepoClient
}

// NewClient creates a new Github Client. If the token is not empty, it will
// use it as the auth token to make calls.
func NewClient(token string) *Client {
	ghClient := github.NewClient(nil)
	if token != "" {
		ghClient = ghClient.WithAuthToken(token)
	}
	c := &Client{
		repoClient: ghClient.Repositories,
	}

	return c
}
