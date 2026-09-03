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

	"github.com/google/go-github/v79/github"
)

var (
	// ErrSecondaryRateLimit is returned when GitHub signals a secondary rate limit (403 or 429).
	ErrSecondaryRateLimit = errors.New("github secondary rate limit exceeded")
	// ErrNilIssueRequest indicates a nil IssueRequest was provided.
	ErrNilIssueRequest = errors.New("nil issue request provided")
)

// CreateIssue creates a new issue in the target repository.
func (c *Client) CreateIssue(
	ctx context.Context,
	owner, repo string,
	req *github.IssueRequest,
) (*github.Issue, error) {
	if req == nil {
		return nil, ErrNilIssueRequest
	}

	issue, resp, err := c.issuesClient.Create(ctx, owner, repo, req)
	if err != nil {
		if resp != nil && (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests) {
			return nil, errors.Join(err, ErrSecondaryRateLimit)
		}

		return nil, fmt.Errorf("failed to create issue in %s/%s: %w", owner, repo, err)
	}

	return issue, nil
}
