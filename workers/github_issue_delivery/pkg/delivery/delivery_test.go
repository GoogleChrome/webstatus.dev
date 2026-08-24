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

package delivery

import (
	"context"
	"errors"
	"testing"
	"time"

	githubissuedeliveryv1 "github.com/GoogleChrome/webstatus.dev/lib/event/githubissuedelivery/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gh"
	"github.com/google/go-github/v79/github"
)

type mockIssueCreator struct {
	createdTitle string
	createdBody  string
	issueID      int64
	issueURL     string
	createErr    error
}

func (m *mockIssueCreator) CreateIssue(
	_ context.Context,
	_, _ string,
	req *github.IssueRequest,
) (*github.Issue, error) {
	if req.Title != nil {
		m.createdTitle = *req.Title
	}
	if req.Body != nil {
		m.createdBody = *req.Body
	}
	if m.createErr != nil {
		return nil, m.createErr
	}

	issue := &github.Issue{
		ID:                &m.issueID,
		Number:            nil,
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
		HTMLURL:           &m.issueURL,
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

	return issue, nil
}

type mockDeliveryStorer struct {
	lockAcquired    bool
	acquireErr      error
	recordedSuccess bool
	recordedIssueID string
	recordedURL     string
	recordErr       error
	lockReleased    bool
	releaseErr      error
}

func (m *mockDeliveryStorer) AcquireDeliveryLock(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ time.Duration,
) (bool, error) {
	return m.lockAcquired, m.acquireErr
}

func (m *mockDeliveryStorer) RecordDeliverySuccess(
	_ context.Context,
	_ string,
	issueID string,
	issueURL string,
) error {
	m.recordedSuccess = true
	m.recordedIssueID = issueID
	m.recordedURL = issueURL

	return m.recordErr
}

func (m *mockDeliveryStorer) ReleaseDeliveryLock(
	_ context.Context,
	_ string,
) error {
	m.lockReleased = true

	return m.releaseErr
}

func sampleJob() githubissuedeliveryv1.GitHubIssueDeliveryEvent {
	return githubissuedeliveryv1.GitHubIssueDeliveryEvent{
		DeliveryID:         "del-123",
		SubscriptionID:     "sub-456",
		VCSProvider:        "github",
		VCSInstallationID:  "inst-789",
		VCSRepositoryID:    "repo-999",
		RepositoryOwner:    "GoogleChrome",
		RepositoryName:     "webstatus.dev",
		RepositoryFullName: "GoogleChrome/webstatus.dev",
		FeatureID:          "css-subgrid",
		FeatureName:        "CSS Subgrid",
		Trigger:            "feature_baseline_to_widely",
		CommitSHA:          "abcdef1234567890abcdef1234567890abcdef12",
		Occurrences: []githubissuedeliveryv1.IssueOccurrence{
			{
				FilePath:       "src/grid.css",
				LineNumber:     42,
				CommentSnippet: "/* TODO: remove */",
			},
		},
		WebStatusURL: "https://webstatus.dev/features/css-subgrid",
	}
}

func TestDelivererSuccess(t *testing.T) {
	t.Parallel()

	creator := &mockIssueCreator{
		createdTitle: "",
		createdBody:  "",
		issueID:      101,
		issueURL:     "https://github.com/GoogleChrome/webstatus.dev/issues/101",
		createErr:    nil,
	}

	storer := &mockDeliveryStorer{
		lockAcquired:    true,
		acquireErr:      nil,
		recordedSuccess: false,
		recordedIssueID: "",
		recordedURL:     "",
		recordErr:       nil,
		lockReleased:    false,
		releaseErr:      nil,
	}

	deliverer := NewDeliverer(creator, storer, "worker-1")
	err := deliverer.ProcessJob(context.Background(), sampleJob())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !storer.recordedSuccess {
		t.Errorf("expected delivery success to be recorded")
	}
	if storer.recordedIssueID != "101" {
		t.Errorf("expected issue ID 101, got %s", storer.recordedIssueID)
	}
}

func TestDelivererLockAlreadyHeld(t *testing.T) {
	t.Parallel()

	creator := &mockIssueCreator{
		createdTitle: "",
		createdBody:  "",
		issueID:      0,
		issueURL:     "",
		createErr:    nil,
	}

	storer := &mockDeliveryStorer{
		lockAcquired:    false,
		acquireErr:      nil,
		recordedSuccess: false,
		recordedIssueID: "",
		recordedURL:     "",
		recordErr:       nil,
		lockReleased:    false,
		releaseErr:      nil,
	}

	deliverer := NewDeliverer(creator, storer, "worker-1")
	err := deliverer.ProcessJob(context.Background(), sampleJob())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if creator.createdTitle != "" {
		t.Errorf("expected zero issue creation when lock not acquired")
	}
}

func TestDelivererSecondaryRateLimit(t *testing.T) {
	t.Parallel()

	creator := &mockIssueCreator{
		createdTitle: "",
		createdBody:  "",
		issueID:      0,
		issueURL:     "",
		createErr:    gh.ErrSecondaryRateLimit,
	}

	storer := &mockDeliveryStorer{
		lockAcquired:    true,
		acquireErr:      nil,
		recordedSuccess: false,
		recordedIssueID: "",
		recordedURL:     "",
		recordErr:       nil,
		lockReleased:    false,
		releaseErr:      nil,
	}

	deliverer := NewDeliverer(creator, storer, "worker-1")
	err := deliverer.ProcessJob(context.Background(), sampleJob())
	if err == nil {
		t.Fatalf("expected rate limit error, got nil")
	}

	if !storer.lockReleased {
		t.Errorf("expected lock to be released on rate limit")
	}
	if storer.recordedSuccess {
		t.Errorf("delivery success should not be recorded on failure")
	}
}

func TestDelivererCreateIssueError(t *testing.T) {
	t.Parallel()

	creator := &mockIssueCreator{
		createdTitle: "",
		createdBody:  "",
		issueID:      0,
		issueURL:     "",
		createErr:    errors.New("github api 500"),
	}

	storer := &mockDeliveryStorer{
		lockAcquired:    true,
		acquireErr:      nil,
		recordedSuccess: false,
		recordedIssueID: "",
		recordedURL:     "",
		recordErr:       nil,
		lockReleased:    false,
		releaseErr:      nil,
	}

	deliverer := NewDeliverer(creator, storer, "worker-1")
	err := deliverer.ProcessJob(context.Background(), sampleJob())
	if err == nil {
		t.Fatalf("expected error from create issue, got nil")
	}
	if storer.recordedSuccess {
		t.Errorf("delivery success should not be recorded on failure")
	}
}
