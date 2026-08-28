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
	"net/http"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	githubissuedeliveryv1 "github.com/GoogleChrome/webstatus.dev/lib/event/githubissuedelivery/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gh"
	"github.com/google/go-github/v79/github"
)

type mockIssueCreator struct {
	hasDeadline      bool
	receivedDeadline time.Time
	createdTitle     string
	createdBody      string
	issueID          int64
	issueURL         string
	createErr        error
}

func (m *mockIssueCreator) CreateIssue(
	ctx context.Context,
	_, _ string,
	req *github.IssueRequest,
) (*github.Issue, error) {
	if deadline, ok := ctx.Deadline(); ok {
		m.hasDeadline = true
		m.receivedDeadline = deadline
	}
	if req.Title != nil {
		m.createdTitle = *req.Title
	}
	if req.Body != nil {
		m.createdBody = *req.Body
	}
	if m.createErr != nil {
		return nil, m.createErr
	}

	//nolint:exhaustruct
	issue := &github.Issue{
		ID:      &m.issueID,
		HTMLURL: &m.issueURL,
	}

	return issue, nil
}

type mockTokenProvider struct {
	token string
	err   error
}

func (m *mockTokenProvider) GetInstallationToken(_ context.Context, _ string) (string, error) {
	return m.token, m.err
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
		Trigger:            "feature.baseline.promote_to_widely",
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
		hasDeadline:      false,
		receivedDeadline: time.Time{},
		createdTitle:     "",
		createdBody:      "",
		issueID:          101,
		issueURL:         "https://github.com/GoogleChrome/webstatus.dev/issues/101",
		createErr:        nil,
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

	tp := &mockTokenProvider{token: "test-token", err: nil}
	deliverer := NewDeliverer(tp, func(_ string) GitHubIssueCreator { return creator }, storer, "worker-1")
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

func TestDelivererAlreadyDelivered(t *testing.T) {
	t.Parallel()

	creator := &mockIssueCreator{
		hasDeadline:      false,
		receivedDeadline: time.Time{},
		createdTitle:     "",
		createdBody:      "",
		issueID:          0,
		issueURL:         "",
		createErr:        nil,
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

	deliverer := NewDeliverer(nil, func(_ string) GitHubIssueCreator { return creator }, storer, "worker-1")
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
		hasDeadline:      false,
		receivedDeadline: time.Time{},
		createdTitle:     "",
		createdBody:      "",
		issueID:          0,
		issueURL:         "",
		createErr:        gh.ErrSecondaryRateLimit,
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

	deliverer := NewDeliverer(nil, func(_ string) GitHubIssueCreator { return creator }, storer, "worker-1")
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
		hasDeadline:      false,
		receivedDeadline: time.Time{},
		createdTitle:     "",
		createdBody:      "",
		issueID:          0,
		issueURL:         "",
		createErr:        errors.New("github api 500"),
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

	deliverer := NewDeliverer(nil, func(_ string) GitHubIssueCreator { return creator }, storer, "worker-1")
	err := deliverer.ProcessJob(context.Background(), sampleJob())
	if err == nil {
		t.Fatalf("expected error from create issue, got nil")
	}
	if storer.recordedSuccess {
		t.Errorf("delivery success should not be recorded on failure")
	}
}

func TestDelivererLockAlreadyHeld(t *testing.T) {
	t.Parallel()

	creator := &mockIssueCreator{
		hasDeadline:      false,
		receivedDeadline: time.Time{},
		createdTitle:     "",
		createdBody:      "",
		issueID:          0,
		issueURL:         "",
		createErr:        nil,
	}

	storer := &mockDeliveryStorer{
		lockAcquired:    false,
		acquireErr:      gcpspanner.ErrDeliveryAlreadyLocked,
		recordedSuccess: false,
		recordedIssueID: "",
		recordedURL:     "",
		recordErr:       nil,
		lockReleased:    false,
		releaseErr:      nil,
	}

	deliverer := NewDeliverer(nil, func(_ string) GitHubIssueCreator { return creator }, storer, "worker-1")
	err := deliverer.ProcessJob(context.Background(), sampleJob())
	if err == nil {
		t.Fatalf("expected error on lock collision, got nil")
	}
	if !errors.Is(err, event.ErrTransientFailure) {
		t.Errorf("expected ErrTransientFailure on lock collision, got %v", err)
	}
}

func TestDelivererTokenProviderError(t *testing.T) {
	t.Parallel()

	tp := &mockTokenProvider{token: "", err: errors.New("auth expired")}
	creator := &mockIssueCreator{
		hasDeadline:      false,
		receivedDeadline: time.Time{},
		createdTitle:     "",
		createdBody:      "",
		issueID:          0,
		issueURL:         "",
		createErr:        nil,
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

	deliverer := NewDeliverer(tp, func(_ string) GitHubIssueCreator { return creator }, storer, "worker-1")
	err := deliverer.ProcessJob(context.Background(), sampleJob())
	if err == nil {
		t.Fatalf("expected error on token failure, got nil")
	}
	if !errors.Is(err, event.ErrTransientFailure) {
		t.Errorf("expected ErrTransientFailure on token error, got %v", err)
	}
	if !storer.lockReleased {
		t.Errorf("expected lock to be released on token failure")
	}
}

func TestDelivererCreateIssueTimeoutEnforced(t *testing.T) {
	t.Parallel()

	creator := &mockIssueCreator{
		hasDeadline:      false,
		receivedDeadline: time.Time{},
		createdTitle:     "",
		createdBody:      "",
		issueID:          12345,
		issueURL:         "https://github.com/owner/repo/issues/1",
		createErr:        nil,
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

	deliverer := NewDeliverer(nil, func(_ string) GitHubIssueCreator { return creator }, storer, "worker-1")
	err := deliverer.ProcessJob(context.Background(), sampleJob())
	if err != nil {
		t.Fatalf("expected successful delivery, got error: %v", err)
	}

	if !creator.hasDeadline {
		t.Fatalf("expected CreateIssue context to have a deadline enforced")
	}

	remaining := time.Until(creator.receivedDeadline)
	if remaining > 25*time.Second || remaining < 10*time.Second {
		t.Errorf("expected deadline ~25s from now (< 30s lock duration), got %v", remaining)
	}
}

func TestDelivererCreateIssueDeadlineExceeded(t *testing.T) {
	t.Parallel()

	creator := &mockIssueCreator{
		hasDeadline:      false,
		receivedDeadline: time.Time{},
		createdTitle:     "",
		createdBody:      "",
		issueID:          0,
		issueURL:         "",
		createErr:        context.DeadlineExceeded,
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

	deliverer := NewDeliverer(nil, func(_ string) GitHubIssueCreator { return creator }, storer, "worker-1")
	err := deliverer.ProcessJob(context.Background(), sampleJob())
	if err == nil {
		t.Fatalf("expected error on deadline exceeded, got nil")
	}
	if !errors.Is(err, event.ErrTransientFailure) {
		t.Errorf("expected ErrTransientFailure on timeout, got %v", err)
	}
	if !storer.lockReleased {
		t.Errorf("expected lock to be released on timeout for retry")
	}
}

//nolint:exhaustruct
func TestDelivererCreateIssueServerError(t *testing.T) {
	t.Parallel()

	serverErr := &github.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusInternalServerError,
		},
		Message: "Internal Server Error",
	}

	creator := &mockIssueCreator{
		hasDeadline:      false,
		receivedDeadline: time.Time{},
		createdTitle:     "",
		createdBody:      "",
		issueID:          0,
		issueURL:         "",
		createErr:        serverErr,
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

	deliverer := NewDeliverer(nil, func(_ string) GitHubIssueCreator { return creator }, storer, "worker-1")
	err := deliverer.ProcessJob(context.Background(), sampleJob())
	if err == nil {
		t.Fatalf("expected error on 500 server error, got nil")
	}
	if !errors.Is(err, event.ErrTransientFailure) {
		t.Errorf("expected ErrTransientFailure on 500 server error, got %v", err)
	}
	if !storer.lockReleased {
		t.Errorf("expected lock to be released on 500 server error")
	}
}

//nolint:exhaustruct
func TestDelivererCreateIssueClientErrorNotRetried(t *testing.T) {
	t.Parallel()

	clientErr := &github.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusNotFound,
		},
		Message: "Not Found",
	}

	creator := &mockIssueCreator{
		hasDeadline:      false,
		receivedDeadline: time.Time{},
		createdTitle:     "",
		createdBody:      "",
		issueID:          0,
		issueURL:         "",
		createErr:        clientErr,
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

	deliverer := NewDeliverer(nil, func(_ string) GitHubIssueCreator { return creator }, storer, "worker-1")
	err := deliverer.ProcessJob(context.Background(), sampleJob())
	if err == nil {
		t.Fatalf("expected error on 404 client error, got nil")
	}
	if errors.Is(err, event.ErrTransientFailure) {
		t.Errorf("expected non-transient error for 404 client error, but got ErrTransientFailure")
	}
	if storer.lockReleased {
		t.Errorf("lock should not be released on permanent client failure")
	}
}
