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
	"net/http"
	"testing"

	"github.com/google/go-github/v79/github"
)

type MockIssuesClient struct {
	CreateFunc func(
		ctx context.Context,
		owner, repo string,
		issue *github.IssueRequest,
	) (*github.Issue, *github.Response, error)
}

func (m *MockIssuesClient) Create(
	ctx context.Context,
	owner, repo string,
	issue *github.IssueRequest,
) (*github.Issue, *github.Response, error) {
	if m.CreateFunc == nil {
		panic("CreateFunc not set")
	}

	return m.CreateFunc(ctx, owner, repo, issue)
}

func mockHTTPResponse(statusCode int) *github.Response {
	// nolint:exhaustruct // Test helper
	return &github.Response{
		Response: &http.Response{
			StatusCode: statusCode,
			Header:     http.Header{},
			Body:       http.NoBody,
		},
	}
}

func TestCreateIssue(t *testing.T) {
	expectedIssue := &github.Issue{
		ID:                nil,
		Number:            new(42),
		State:             nil,
		StateReason:       nil,
		Locked:            nil,
		Title:             nil,
		Body:              nil,
		AuthorAssociation: nil,
		User:              nil,
		Labels:            nil,
		Assignee:          nil,
		Comments:          nil,
		ClosedAt:          nil,
		CreatedAt:         nil,
		UpdatedAt:         nil,
		ClosedBy:          nil,
		URL:               nil,
		HTMLURL:           new("https://github.com/owner/repo/issues/42"),
		CommentsURL:       nil,
		EventsURL:         nil,
		LabelsURL:         nil,
		RepositoryURL:     nil,
		Milestone:         nil,
		PullRequestLinks:  nil,
		Repository:        nil,
		Reactions:         nil,
		Assignees:         nil,
		NodeID:            nil,
		Draft:             nil,
		Type:              nil,
		TextMatches:       nil,
		ActiveLockReason:  nil,
	}

	testCases := []struct {
		name     string
		req      *github.IssueRequest
		mockFunc func(ctx context.Context, owner, repo string,
			issue *github.IssueRequest) (*github.Issue, *github.Response, error)
		expectErr bool
		errIs     error
	}{
		{
			name: "Success",
			req: &github.IssueRequest{
				Title:       new("Baseline Promotion Notice"),
				Body:        new("Feature view-transitions reached Baseline Newly Available"),
				Labels:      nil,
				Assignee:    nil,
				State:       nil,
				StateReason: nil,
				Milestone:   nil,
				Assignees:   nil,
				Type:        nil,
			},
			mockFunc: func(_ context.Context, _, _ string, _ *github.IssueRequest) (*github.Issue, *github.Response, error) {
				return expectedIssue, mockHTTPResponse(http.StatusCreated), nil
			},
			expectErr: false,
			errIs:     nil,
		},
		{
			name:      "Nil Request",
			req:       nil,
			mockFunc:  nil,
			expectErr: true,
			errIs:     ErrNilIssueRequest,
		},
		{
			name: "Secondary Rate Limit 403",
			req: &github.IssueRequest{
				Title:       new("Rate Limited Issue"),
				Body:        nil,
				Labels:      nil,
				Assignee:    nil,
				State:       nil,
				StateReason: nil,
				Milestone:   nil,
				Assignees:   nil,
				Type:        nil,
			},
			mockFunc: func(_ context.Context, _, _ string, _ *github.IssueRequest) (*github.Issue, *github.Response, error) {
				return nil, mockHTTPResponse(http.StatusForbidden), errors.New("rate limit exceeded")
			},
			expectErr: true,
			errIs:     ErrSecondaryRateLimit,
		},
		{
			name: "Secondary Rate Limit 429",
			req: &github.IssueRequest{
				Title:       new("Rate Limited Issue 429"),
				Body:        nil,
				Labels:      nil,
				Assignee:    nil,
				State:       nil,
				StateReason: nil,
				Milestone:   nil,
				Assignees:   nil,
				Type:        nil,
			},
			mockFunc: func(_ context.Context, _, _ string, _ *github.IssueRequest) (*github.Issue, *github.Response, error) {
				return nil, mockHTTPResponse(http.StatusTooManyRequests), errors.New("too many requests")
			},
			expectErr: true,
			errIs:     ErrSecondaryRateLimit,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &Client{
				repoClient:   nil,
				gitClient:    nil,
				issuesClient: &MockIssuesClient{CreateFunc: tc.mockFunc},
				appsClient:   nil,
			}

			issue, err := client.CreateIssue(context.Background(), "owner", "repo", tc.req)
			if tc.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.errIs != nil && !errors.Is(err, tc.errIs) {
				t.Errorf("expected error %v, got %v", tc.errIs, err)
			}
			if !tc.expectErr && issue.GetNumber() != 42 {
				t.Errorf("expected issue #42, got #%d", issue.GetNumber())
			}
		})
	}
}
