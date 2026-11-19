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

package workflow

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gh"
	"github.com/google/go-github/v79/github"
)

// JobArguments represents CLI / Job runtime configuration.
type JobArguments struct{}

// ErrScanTaskPublishFailed indicates that one or more scan tasks could not be published.
var ErrScanTaskPublishFailed = errors.New("failed to publish scan tasks")

// NewJobArguments creates empty job arguments for scheduled execution.
func NewJobArguments() JobArguments {
	return JobArguments{}
}

// InstallationStorer handles reading and persisting VCS installations in Spanner.
type InstallationStorer interface {
	ListVCSInstallations(ctx context.Context) ([]gcpspanner.VCSInstallation, error)
	UpsertVCSInstallation(ctx context.Context, in gcpspanner.VCSInstallation) (*string, error)
}

// TokenProvider mints installation and app-level access tokens for VCS providers.
type TokenProvider interface {
	GetInstallationToken(ctx context.Context, installationID string) (string, error)
	GetAppToken() (string, error)
}

// GitHubRepoLister abstracts listing installation repositories and app installations from GitHub.
type GitHubRepoLister interface {
	ListInstallationRepositories(ctx context.Context, token string, opts *github.ListOptions) ([]*github.Repository, error)
	ListAppInstallations(ctx context.Context, token string, opts *github.ListOptions) ([]*github.Installation, error)
}

// DefaultGitHubRepoLister creates default *gh.Client instances and lists repositories/installations.
type DefaultGitHubRepoLister struct{}

func (l DefaultGitHubRepoLister) ListInstallationRepositories(
	ctx context.Context,
	token string,
	opts *github.ListOptions,
) ([]*github.Repository, error) {
	return gh.NewClient(token).ListInstallationRepositories(ctx, opts)
}

func (l DefaultGitHubRepoLister) ListAppInstallations(
	ctx context.Context,
	token string,
	opts *github.ListOptions,
) ([]*github.Installation, error) {
	return gh.NewClient(token).ListAppInstallations(ctx, opts)
}

// TaskPublisher publishes scan tasks to Pub/Sub.
type TaskPublisher interface {
	PublishCodeScanTask(ctx context.Context, task codescantaskv1.CodeScanTaskEvent) error
}

// VCSSyncProcessor orchestrates the scheduled discovery and scan triggering of repositories.
type VCSSyncProcessor struct {
	installationStorer InstallationStorer
	tokenProvider      TokenProvider
	repoLister         GitHubRepoLister
	taskPublisher      TaskPublisher
}

// NewVCSSyncProcessor constructs a new scheduled VCS sync processor.
func NewVCSSyncProcessor(
	installationStorer InstallationStorer,
	tokenProvider TokenProvider,
	repoLister GitHubRepoLister,
	taskPublisher TaskPublisher,
) *VCSSyncProcessor {
	return &VCSSyncProcessor{
		installationStorer: installationStorer,
		tokenProvider:      tokenProvider,
		repoLister:         repoLister,
		taskPublisher:      taskPublisher,
	}
}

// Process runs the scheduled reconciliation job across all active installations.
//
// The run fails if any scan task can't be published or any installation can't be synced, except
// installations that were suspended or uninstalled. A failure doesn't stop the remaining
// repositories from being published. The job runs every 5 minutes without retries, so the next
// run retries a one-off failure, and the job's alert only fires when no run succeeds for a while.
func (p *VCSSyncProcessor) Process(ctx context.Context, _ JobArguments) error {
	slog.InfoContext(ctx, "starting scheduled VCS repository sync")

	// Tier 1: Reconcile GitHub App installations (healing any missed installation webhooks)
	if err := p.reconcileGitHubAppInstallations(ctx); err != nil {
		slog.WarnContext(ctx, "failed to reconcile GitHub App installations from API, falling back to database",
			"error", err)
	}

	// Tier 2: Read active installations from database and discover repositories
	installations, err := p.installationStorer.ListVCSInstallations(ctx)
	if err != nil {
		return fmt.Errorf("failed to list VCS installations: %w", err)
	}

	slog.InfoContext(ctx, "discovered active VCS installations", "count", len(installations))

	var counts publishCounts
	var errs []error
	for _, inst := range installations {
		if err := ctx.Err(); err != nil {
			return err
		}

		// Publishes are counted even if the installation fails partway, e.g. while listing a later page.
		err := p.syncInstallation(ctx, inst, &counts)
		if err == nil {
			continue
		}

		attrs := []any{
			"provider", inst.VCSProvider,
			"installation_id", inst.VCSInstallationID,
			"account", inst.AccountLogin,
			"error", err,
		}
		// Skip installations that can't be synced, e.g. because they were suspended or uninstalled.
		// Rate limits and 401s still fail the run. A 401 means GitHub rejected the app's credentials,
		// so every installation fails the same way.
		if gh.IsPermanentClientError(err) && !isUnauthorized(err) {
			slog.WarnContext(ctx, "client error syncing installation, skipping", attrs...)

			continue
		}

		slog.ErrorContext(ctx, "failed to sync installation", attrs...)
		errs = append(errs, err)
	}

	if err := counts.err(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	slog.InfoContext(ctx, "completed scheduled VCS repository sync successfully", "published", counts.published)

	return nil
}

func parseGitHubAppPermissionLevel(level *string) *gcpspanner.GitHubAppPermissionLevel {
	if level == nil || *level == "" {
		return nil
	}

	var perm gcpspanner.GitHubAppPermissionLevel
	switch strings.ToLower(strings.TrimSpace(*level)) {
	case "read":
		perm = gcpspanner.GitHubAppPermissionLevelRead
	case "write":
		perm = gcpspanner.GitHubAppPermissionLevelWrite
	default:
		return nil
	}

	return &perm
}

func toGitHubPermissions(perms *github.InstallationPermissions) *gcpspanner.GitHubPermissions {
	if perms == nil {
		return nil
	}

	return &gcpspanner.GitHubPermissions{
		Issues:       parseGitHubAppPermissionLevel(perms.Issues),
		Contents:     parseGitHubAppPermissionLevel(perms.Contents),
		Metadata:     parseGitHubAppPermissionLevel(perms.Metadata),
		PullRequests: parseGitHubAppPermissionLevel(perms.PullRequests),
		Workflows:    parseGitHubAppPermissionLevel(perms.Workflows),
		Actions:      parseGitHubAppPermissionLevel(perms.Actions),
	}
}

func (p *VCSSyncProcessor) reconcileGitHubAppInstallations(ctx context.Context) error {
	appToken, err := p.tokenProvider.GetAppToken()
	if err != nil {
		return fmt.Errorf("failed to get app JWT token: %w", err)
	}

	page := 1

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		opts := &github.ListOptions{
			Page:    page,
			PerPage: 100,
		}

		installations, err := p.repoLister.ListAppInstallations(ctx, appToken, opts)
		if err != nil {
			return fmt.Errorf("failed to list app installations page %d: %w", page, err)
		}

		if len(installations) == 0 {
			break
		}

		for _, inst := range installations {
			if inst == nil || inst.Account == nil {
				continue
			}

			accountLogin := inst.Account.GetLogin()
			accountType := inst.Account.GetType()
			installationID := strconv.FormatInt(inst.GetID(), 10)

			spannerInst := gcpspanner.VCSInstallation{
				ID:                  "",
				VCSProvider:         gcpspanner.VCSProviderGitHub,
				VCSInstallationID:   installationID,
				AccountLogin:        accountLogin,
				AccountType:         accountType,
				RepositorySelection: inst.GetRepositorySelection(),
				Permissions: gcpspanner.VCSPermissions{
					GitHub: toGitHubPermissions(inst.GetPermissions()),
				},
				CreatedAt: inst.GetCreatedAt().Time,
				UpdatedAt: inst.GetUpdatedAt().Time,
			}

			if _, err := p.installationStorer.UpsertVCSInstallation(ctx, spannerInst); err != nil {
				slog.ErrorContext(ctx, "failed to upsert VCS installation",
					"installation_id", installationID,
					"account", accountLogin,
					"error", err,
				)
			}
		}

		if len(installations) < 100 {
			break
		}
		page++
	}

	return nil
}

func (p *VCSSyncProcessor) syncInstallation(
	ctx context.Context,
	inst gcpspanner.VCSInstallation,
	counts *publishCounts,
) error {
	switch inst.VCSProvider {
	case gcpspanner.VCSProviderGitHub:
		return p.syncGitHubInstallation(ctx, inst, counts)
	default:
		slog.WarnContext(ctx, "unsupported VCS provider for sync", "provider", inst.VCSProvider)

		return nil
	}
}

func (p *VCSSyncProcessor) syncGitHubInstallation(
	ctx context.Context,
	inst gcpspanner.VCSInstallation,
	counts *publishCounts,
) error {
	token, err := p.tokenProvider.GetInstallationToken(ctx, inst.VCSInstallationID)
	if err != nil {
		return fmt.Errorf("failed to obtain installation token for %s: %w", inst.VCSInstallationID, err)
	}

	page := 1

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		opts := &github.ListOptions{
			Page:    page,
			PerPage: 100,
		}

		repos, err := p.repoLister.ListInstallationRepositories(ctx, token, opts)
		if err != nil {
			return fmt.Errorf("failed to list installation repositories page %d: %w", page, err)
		}

		if len(repos) == 0 {
			break
		}

		for _, repo := range repos {
			if repo == nil {
				continue
			}

			branch := repo.GetDefaultBranch()
			if branch == "" {
				branch = "main"
			}

			task := codescantaskv1.CodeScanTaskEvent{
				VCSProvider:        codescantaskv1.VCSProviderGitHub,
				VCSInstallationID:  inst.VCSInstallationID,
				VCSRepositoryID:    strconv.FormatInt(repo.GetID(), 10),
				RepositoryFullName: repo.GetFullName(),
				CommitSHA:          "",
				Branch:             branch,
				IsDefaultBranch:    true,
				ModifiedFiles:      nil,
			}

			err := p.taskPublisher.PublishCodeScanTask(ctx, task)
			counts.record(err)
			if err != nil {
				slog.ErrorContext(ctx, "failed to publish scan task for repository",
					"repo", repo.GetFullName(),
					"error", err,
				)
			}
		}

		if len(repos) < 100 {
			break
		}
		page++
	}

	return nil
}

// publishCounts tallies the scan tasks a run tried to publish.
type publishCounts struct {
	published int
	failed    int
	// firstErr is the first publish error, reported in the run's error.
	firstErr error
}

// record counts the outcome of publishing one scan task.
func (c *publishCounts) record(err error) {
	if err == nil {
		c.published++

		return
	}

	c.failed++
	if c.firstErr == nil {
		c.firstErr = err
	}
}

// err returns an error wrapping ErrScanTaskPublishFailed if any scan task failed to publish.
func (c *publishCounts) err() error {
	if c.failed == 0 {
		return nil
	}

	return fmt.Errorf("%w: %d of %d, first error: %w",
		ErrScanTaskPublishFailed, c.failed, c.failed+c.published, c.firstErr)
}

// isUnauthorized reports whether GitHub rejected the app's credentials (HTTP 401), e.g. because the
// app's private key was revoked. Unlike other client errors, it isn't specific to one installation.
func isUnauthorized(err error) bool {
	var ghErr *github.ErrorResponse

	return errors.As(err, &ghErr) && ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusUnauthorized
}
