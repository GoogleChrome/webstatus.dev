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

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	githubissuedeliveryv1 "github.com/GoogleChrome/webstatus.dev/lib/event/githubissuedelivery/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gh"
	"github.com/google/go-github/v79/github"
)

func toGHOccurrences(occs []githubissuedeliveryv1.IssueOccurrence) []gh.IssueOccurrence {
	if occs == nil {
		return nil
	}

	res := make([]gh.IssueOccurrence, len(occs))
	for i, occ := range occs {
		res[i] = gh.IssueOccurrence{
			FilePath:       occ.FilePath,
			LineNumber:     occ.LineNumber,
			CommentSnippet: occ.CommentSnippet,
		}
	}

	return res
}

// GitHubIssueCreator defines the GitHub issue creation operations.
type GitHubIssueCreator interface {
	CreateIssue(ctx context.Context, owner, repo string, req *github.IssueRequest) (*github.Issue, error)
}

// TokenProvider defines the token retrieval interface for GitHub App installations.
type TokenProvider interface {
	GetInstallationToken(ctx context.Context, installationID string) (string, error)
}

// ClientFactory creates a GitHubIssueCreator for an auth token.
type ClientFactory func(token string) GitHubIssueCreator

// LockStorer defines the Spanner operations for lock management and delivery recording.
type LockStorer interface {
	AcquireDeliveryLock(
		ctx context.Context,
		subscriptionID, deliveryID, workerLockID string,
		ttl time.Duration,
	) (bool, error)
	RecordDeliverySuccess(
		ctx context.Context, deliveryID string, workerLockID string, issueID string, issueURL string) error
	ReleaseDeliveryLock(ctx context.Context, deliveryID string, workerLockID string) error
}

// Deliverer coordinates lock acquisition, issue rendering, and issue creation.
type Deliverer struct {
	tokenProvider TokenProvider
	clientFactory ClientFactory
	storer        LockStorer
	workerLockID  string
}

// NewDeliverer creates a new Deliverer instance.
func NewDeliverer(
	tokenProvider TokenProvider,
	clientFactory ClientFactory,
	storer LockStorer,
	workerLockID string,
) *Deliverer {
	if clientFactory == nil {
		clientFactory = func(token string) GitHubIssueCreator {
			return gh.NewClient(token)
		}
	}

	return &Deliverer{
		tokenProvider: tokenProvider,
		clientFactory: clientFactory,
		storer:        storer,
		workerLockID:  workerLockID,
	}
}

// ProcessJob executes the issue delivery workflow with distributed lock protection.
func (d *Deliverer) ProcessJob(ctx context.Context, job githubissuedeliveryv1.GitHubIssueDeliveryEvent) error {
	// 1. Acquire 30s atomic delivery lock
	lockAcquired, err := d.storer.AcquireDeliveryLock(
		ctx,
		job.SubscriptionID,
		job.DeliveryID,
		d.workerLockID,
		30*time.Second,
	)
	if errors.Is(err, gcpspanner.ErrDeliveryAlreadyLocked) {
		slog.InfoContext(ctx, "delivery lock held by another worker, backing off", "deliveryID", job.DeliveryID)

		return fmt.Errorf("%w: delivery lock held by another worker: %w", event.ErrTransientFailure, err)
	}
	if err != nil {
		return fmt.Errorf("%w: failed to acquire delivery lock: %w", event.ErrTransientFailure, err)
	}
	if !lockAcquired {
		slog.InfoContext(ctx, "delivery lock already delivered, skipping", "deliveryID", job.DeliveryID)

		return nil
	}

	// 2. Render issue title and markdown body
	ghOccs := toGHOccurrences(job.Occurrences)

	title := gh.RenderIssueTitle(job.FeatureName, job.Trigger)
	body := gh.RenderIssueBody(gh.IssueRenderParams{
		FeatureID:          job.FeatureID,
		FeatureName:        job.FeatureName,
		Trigger:            job.Trigger,
		RepositoryFullName: job.RepositoryFullName,
		CommitSHA:          job.CommitSHA,
		Occurrences:        ghOccs,
		WebStatusURL:       job.WebStatusURL,
	})

	// 3. Obtain auth token and initialize issue creator
	var token string
	if d.tokenProvider != nil && job.VCSInstallationID != "" {
		t, tokErr := d.tokenProvider.GetInstallationToken(ctx, job.VCSInstallationID)
		if tokErr != nil {
			slog.ErrorContext(ctx, "failed to get installation token for delivery", "error", tokErr)
			if relErr := d.storer.ReleaseDeliveryLock(ctx, job.DeliveryID, d.workerLockID); relErr != nil {
				slog.ErrorContext(ctx, "failed to release lock after token error", "error", relErr)
			}

			return fmt.Errorf("%w: failed to get installation token: %w", event.ErrTransientFailure, tokErr)
		}
		token = t
	}

	creator := d.clientFactory(token)

	// 4. Create issue on GitHub with 25s sub-timeout before 30s lock expiry
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

	createCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	issue, createErr := creator.CreateIssue(createCtx, job.RepositoryOwner, job.RepositoryName, req)
	if createErr != nil {
		return d.handleCreateIssueError(ctx, job, createErr)
	}

	issueID := strconv.FormatInt(issue.GetID(), 10)
	issueURL := issue.GetHTMLURL()

	// 5. Record successful delivery
	if recErr := d.storer.RecordDeliverySuccess(ctx, job.DeliveryID, d.workerLockID, issueID, issueURL); recErr != nil {
		slog.ErrorContext(ctx, "failed to record delivery success", "error", recErr, "issueURL", issueURL)

		return fmt.Errorf("failed to record delivery success: %w", recErr)
	}

	slog.InfoContext(ctx, "successfully delivered notification issue",
		"repo", job.RepositoryFullName,
		"issueURL", issueURL,
		"feature", job.FeatureID)

	return nil
}

func (d *Deliverer) handleCreateIssueError(
	ctx context.Context,
	job githubissuedeliveryv1.GitHubIssueDeliveryEvent,
	createErr error,
) error {
	if errors.Is(createErr, gh.ErrSecondaryRateLimit) {
		slog.WarnContext(ctx, "hit secondary rate limit, releasing lock for retry backoff",
			"repo", job.RepositoryFullName)
		if relErr := d.storer.ReleaseDeliveryLock(ctx, job.DeliveryID, d.workerLockID); relErr != nil {
			slog.ErrorContext(ctx, "failed to release lock after rate limit", "error", relErr)
		}

		return fmt.Errorf("%w: rate limited on issue creation: %w", event.ErrTransientFailure, createErr)
	}

	if errors.Is(createErr, context.DeadlineExceeded) || gh.IsServerError(createErr) {
		slog.WarnContext(ctx, "transient error creating issue, releasing lock for retry",
			"repo", job.RepositoryFullName, "error", createErr)
		if relErr := d.storer.ReleaseDeliveryLock(ctx, job.DeliveryID, d.workerLockID); relErr != nil {
			slog.ErrorContext(ctx, "failed to release lock after transient error", "error", relErr)
		}

		return fmt.Errorf("%w: transient error creating issue: %w", event.ErrTransientFailure, createErr)
	}

	if gh.IsClientError(createErr) {
		slog.WarnContext(ctx, "client error creating issue, skipping retry",
			"repo", job.RepositoryFullName, "error", createErr)

		return fmt.Errorf("client error creating issue: %w", createErr)
	}

	return fmt.Errorf("failed to create issue: %w", createErr)
}
