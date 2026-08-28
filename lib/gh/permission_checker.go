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
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-github/v79/github"
)

var (
	// ErrClientNotConfigured is returned when the collaborator client is nil.
	ErrClientNotConfigured = errors.New("github collaborator client not configured")
)

// CollaboratorPermissionService abstracts the GitHub API operation for getting collaborator permissions.
type CollaboratorPermissionService interface {
	GetPermissionLevel(ctx context.Context, owner, repo, user string) (
		*github.RepositoryPermissionLevel, *github.Response, error)
}

// UserLookupService abstracts looking up user details from GitHub.
type UserLookupService interface {
	GetByID(ctx context.Context, id int64) (*github.User, *github.Response, error)
}

// GitHubPermissionChecker verifies user permissions against GitHub repositories.
//
// FUTURE: Consider caching (e.g., in Valkey) the numeric-to-login resolution
// and the repository admin permission result to reduce GitHub API quota usage
// and optimize response latency under high traffic.
type GitHubPermissionChecker struct {
	reposClient   CollaboratorPermissionService
	usersClient   UserLookupService
	tokenProvider *TokenProvider
	baseURL       *url.URL
	httpClient    *http.Client
}

// NewGitHubPermissionChecker creates a new GitHubPermissionChecker with the provided services.
func NewGitHubPermissionChecker(
	reposClient CollaboratorPermissionService,
	usersClient UserLookupService,
) *GitHubPermissionChecker {
	return &GitHubPermissionChecker{
		reposClient:   reposClient,
		usersClient:   usersClient,
		tokenProvider: nil,
		baseURL:       nil,
		httpClient:    nil,
	}
}

// NewGitHubPermissionCheckerWithBaseURL creates a GitHubPermissionChecker with an optional custom base URL.
func NewGitHubPermissionCheckerWithBaseURL(baseURL *url.URL) *GitHubPermissionChecker {
	return NewGitHubPermissionCheckerWithTokenProvider(nil, baseURL)
}

func normalizeBaseURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	copyURL := *u
	if !strings.HasSuffix(copyURL.Path, "/") {
		copyURL.Path += "/"
	}

	return &copyURL
}

// NewGitHubPermissionCheckerWithTokenProvider creates a GitHubPermissionChecker with an optional
// TokenProvider and custom base URL.
func NewGitHubPermissionCheckerWithTokenProvider(
	tokenProvider *TokenProvider,
	baseURL *url.URL,
) *GitHubPermissionChecker {
	var httpClient *http.Client
	if tokenProvider != nil {
		httpClient = tokenProvider.httpClient
	}

	normalizedBaseURL := normalizeBaseURL(baseURL)
	ghClient := github.NewClient(httpClient)
	if normalizedBaseURL != nil {
		ghClient.BaseURL = normalizedBaseURL
		ghClient.UploadURL = normalizedBaseURL
	}

	return &GitHubPermissionChecker{
		reposClient:   ghClient.Repositories,
		usersClient:   ghClient.Users,
		tokenProvider: tokenProvider,
		baseURL:       normalizedBaseURL,
		httpClient:    httpClient,
	}
}

// HasRepositoryAdminAccess checks if the given user (identified by GitHub username or numeric user ID string)
// has admin access to owner/repo on GitHub.
func (c *GitHubPermissionChecker) HasRepositoryAdminAccess(
	ctx context.Context,
	owner, repo, githubUserID string,
) (bool, error) {
	if owner == "" || repo == "" || githubUserID == "" {
		return false, nil
	}

	reposClient := c.reposClient
	usersClient := c.usersClient

	if c.tokenProvider != nil {
		token, err := c.tokenProvider.GetInstallationTokenForRepo(ctx, owner, repo)
		if err != nil {
			if errors.Is(err, ErrRepositoryInstallationNotFound) {
				return false, nil
			}

			return false, fmt.Errorf("failed to get repository installation token: %w", err)
		}

		ghClient := github.NewClient(c.httpClient).WithAuthToken(token)
		if c.baseURL != nil {
			ghClient.BaseURL = c.baseURL
			ghClient.UploadURL = c.baseURL
		}

		reposClient = ghClient.Repositories
		usersClient = ghClient.Users
	}

	if reposClient == nil {
		return false, ErrClientNotConfigured
	}

	login := githubUserID
	if id, err := strconv.ParseInt(githubUserID, 10, 64); err == nil && usersClient != nil {
		user, resp, userErr := usersClient.GetByID(ctx, id)
		if userErr != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return false, nil
			}

			return false, userErr
		}
		if user == nil || user.Login == nil || *user.Login == "" {
			return false, nil
		}
		login = *user.Login
	}

	perm, resp, err := reposClient.GetPermissionLevel(ctx, owner, repo, login)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, err
	}

	if perm == nil {
		return false, nil
	}

	return perm.GetPermission() == "admin", nil
}
