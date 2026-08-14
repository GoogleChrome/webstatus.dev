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
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gh"
	"github.com/google/go-github/v79/github"
)

// IssueDeliveryJob represents the Pub/Sub job payload for creating a notification issue.
type IssueDeliveryJob struct {
	DeliveryID         string                              `json:"delivery_id"`
	SubscriptionID     string                              `json:"subscription_id"`
	VCSProvider        string                              `json:"vcs_provider"`
	VCSInstallationID  string                              `json:"vcs_installation_id"`
	VCSRepositoryID    string                              `json:"vcs_repository_id"`
	RepositoryOwner    string                              `json:"repository_owner"`
	RepositoryName     string                              `json:"repository_name"`
	RepositoryFullName string                              `json:"repository_full_name"`
	FeatureID          string                              `json:"feature_id"`
	FeatureName        string                              `json:"feature_name"`
	Trigger            gcpspanner.SubscriptionTrigger      `json:"trigger"`
	CommitSHA          string                              `json:"commit_sha"`
	Occurrences        []gcpspanner.SubscriptionOccurrence `json:"occurrences"`
	WebStatusURL       string                              `json:"webstatus_url"`
}

// GitHubIssueCreator defines the GitHub issue creation operations.
type GitHubIssueCreator interface {
	CreateIssue(ctx context.Context, owner, repo string, req *github.IssueRequest) (*github.Issue, error)
}

// LockStorer defines the Spanner operations for lock management and delivery recording.
type LockStorer interface {
	AcquireDeliveryLock(
		ctx context.Context,
		subscriptionID, deliveryID, workerLockID string,
		ttl time.Duration,
	) (bool, error)
	RecordDeliverySuccess(ctx context.Context, deliveryID string, issueID string, issueURL string) error
	ReleaseDeliveryLock(ctx context.Context, deliveryID string) error
}

// Deliverer coordinates lock acquisition, issue rendering, and issue creation.
type Deliverer struct {
	issueCreator GitHubIssueCreator
	storer       LockStorer
	workerLockID string
}

// NewDeliverer creates a new Deliverer instance.
func NewDeliverer(creator GitHubIssueCreator, storer LockStorer, workerLockID string) *Deliverer {
	return &Deliverer{
		issueCreator: creator,
		storer:       storer,
		workerLockID: workerLockID,
	}
}

// ProcessJob executes the issue delivery workflow with distributed lock protection.
func (d *Deliverer) ProcessJob(ctx context.Context, job IssueDeliveryJob) error {
	// 1. Acquire 30s atomic delivery lock
	lockAcquired, err := d.storer.AcquireDeliveryLock(
		ctx,
		job.SubscriptionID,
		job.DeliveryID,
		d.workerLockID,
		30*time.Second,
	)
	if err != nil {
		return fmt.Errorf("failed to acquire delivery lock: %w", err)
	}
	if !lockAcquired {
		slog.InfoContext(ctx, "delivery lock already held or completed, skipping", "deliveryID", job.DeliveryID)

		return nil
	}

	// 2. Render issue title and markdown body
	title := gh.RenderIssueTitle(job.FeatureName, job.Trigger)
	body := gh.RenderIssueBody(gh.IssueRenderParams{
		FeatureID:          job.FeatureID,
		FeatureName:        job.FeatureName,
		Trigger:            job.Trigger,
		RepositoryFullName: job.RepositoryFullName,
		CommitSHA:          job.CommitSHA,
		Occurrences:        job.Occurrences,
		WebStatusURL:       job.WebStatusURL,
	})

	// 3. Create issue on GitHub
	req := &github.IssueRequest{
		Title:       &title,
		Body:        &body,
		Labels:      nil,
		Assignee:    nil,
		State:       nil,
		StateReason: nil,
		Milestone:   nil,
		Assignees:   nil,
		Type:        nil,
	}

	issue, createErr := d.issueCreator.CreateIssue(ctx, job.RepositoryOwner, job.RepositoryName, req)
	if createErr != nil {
		// If secondary rate limit or transient error, reset lock so subsequent workers can retry
		if errors.Is(createErr, gh.ErrSecondaryRateLimit) {
			slog.WarnContext(ctx, "hit secondary rate limit, releasing lock for retry backoff",
				"repo", job.RepositoryFullName)
			if relErr := d.storer.ReleaseDeliveryLock(ctx, job.DeliveryID); relErr != nil {
				slog.ErrorContext(ctx, "failed to release lock after rate limit", "error", relErr)
			}

			return fmt.Errorf("rate limited on issue creation: %w", createErr)
		}

		return fmt.Errorf("failed to create issue: %w", createErr)
	}

	issueID := strconv.FormatInt(issue.GetID(), 10)
	issueURL := issue.GetHTMLURL()

	// 4. Record successful delivery
	if recErr := d.storer.RecordDeliverySuccess(ctx, job.DeliveryID, issueID, issueURL); recErr != nil {
		slog.ErrorContext(ctx, "failed to record delivery success", "error", recErr, "issueURL", issueURL)

		return fmt.Errorf("failed to record delivery success: %w", recErr)
	}

	slog.InfoContext(ctx, "successfully delivered notification issue",
		"repo", job.RepositoryFullName,
		"issueURL", issueURL,
		"feature", job.FeatureID)

	return nil
}
