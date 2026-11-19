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
	"strconv"
	"strings"
	"time"

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
	scanHistory        ScanHistoryReader
	skipPolicy         SkipPolicy
	now                func() time.Time
	// logger writes the logs that alerts and dashboards depend on, so tests can capture them.
	logger *slog.Logger
}

// NewVCSSyncProcessor constructs a new scheduled VCS sync processor.
//
// The processor reads each repository's recent scans to decide whether it needs a new scan. It
// only skips repositories of installations allowed by skipPolicy; for the others it logs what it
// would have skipped and publishes every repository.
func NewVCSSyncProcessor(
	installationStorer InstallationStorer,
	tokenProvider TokenProvider,
	repoLister GitHubRepoLister,
	taskPublisher TaskPublisher,
	scanHistory ScanHistoryReader,
	skipPolicy SkipPolicy,
) *VCSSyncProcessor {
	return &VCSSyncProcessor{
		installationStorer: installationStorer,
		tokenProvider:      tokenProvider,
		repoLister:         repoLister,
		taskPublisher:      taskPublisher,
		scanHistory:        scanHistory,
		skipPolicy:         skipPolicy,
		now:                time.Now,
		logger:             slog.Default(),
	}
}

// Process runs the scheduled reconciliation job across all active installations.
func (p *VCSSyncProcessor) Process(ctx context.Context, _ JobArguments) error {
	slog.InfoContext(ctx, "starting scheduled VCS repository sync", "skip_policy", p.skipPolicy.String())

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

	stats := newSyncStats()
	var errs []error
	for _, inst := range installations {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := p.syncInstallation(ctx, inst, stats); err != nil {
			if gh.IsClientError(err) {
				slog.WarnContext(ctx, "client error syncing installation, skipping",
					"provider", inst.VCSProvider,
					"installation_id", inst.VCSInstallationID,
					"account", inst.AccountLogin,
					"error", err,
				)

				continue
			}

			slog.ErrorContext(ctx, "failed to sync installation",
				"provider", inst.VCSProvider,
				"installation_id", inst.VCSInstallationID,
				"account", inst.AccountLogin,
				"error", err,
			)
			errs = append(errs, err)
		}
	}

	p.logger.InfoContext(ctx, "scan task decisions",
		"skip_policy", p.skipPolicy.String(),
		"published", stats.published,
		"skipped", stats.skipped,
		"would_skip", stats.wouldSkip,
		"reasons", stats.reasonCounts(),
	)

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	slog.InfoContext(ctx, "completed scheduled VCS repository sync successfully")

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
	stats *syncStats,
) error {
	switch inst.VCSProvider {
	case gcpspanner.VCSProviderGitHub:
		return p.syncGitHubInstallation(ctx, inst, stats)
	default:
		slog.WarnContext(ctx, "unsupported VCS provider for sync", "provider", inst.VCSProvider)

		return nil
	}
}

func (p *VCSSyncProcessor) syncGitHubInstallation(
	ctx context.Context,
	inst gcpspanner.VCSInstallation,
	stats *syncStats,
) error {
	token, err := p.tokenProvider.GetInstallationToken(ctx, inst.VCSInstallationID)
	if err != nil {
		return fmt.Errorf("failed to obtain installation token for %s: %w", inst.VCSInstallationID, err)
	}

	skipEnabled := p.skipPolicy.EnabledFor(inst.VCSInstallationID)
	page := 1
	var publishErrs []error
	publishAttempts := 0

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

		history, historyOK := p.readScanHistory(ctx, inst, repos)
		now := p.now()
		for _, repo := range repos {
			if repo == nil {
				continue
			}

			task := newScanTask(inst, repo)
			decision := p.decide(ctx, now, task, repo.GetPushedAt().Time, history, historyOK)
			if !stats.record(decision, skipEnabled) {
				continue
			}

			publishAttempts++
			if err := p.taskPublisher.PublishCodeScanTask(ctx, task); err != nil {
				slog.ErrorContext(ctx, "failed to publish scan task for repository",
					"repo", repo.GetFullName(),
					"error", err,
				)
				publishErrs = append(publishErrs, err)
			}
		}

		if len(repos) < 100 {
			break
		}
		page++
	}

	// Scan failures are no longer retried through Pub/Sub, so vcs-sync is the only retry path.
	// A run that could not publish must fail, so the "no successful vcs-sync" alert can fire,
	// instead of silently reporting success while nothing gets scanned.
	if len(publishErrs) > 0 {
		return fmt.Errorf("%w: %d/%d scan-task publishes failed: %w",
			ErrScanTaskPublishFailed, len(publishErrs), publishAttempts, errors.Join(publishErrs...))
	}

	return nil
}

// newScanTask builds the branch-head scan task for a repository.
func newScanTask(inst gcpspanner.VCSInstallation, repo *github.Repository) codescantaskv1.CodeScanTaskEvent {
	branch := repo.GetDefaultBranch()
	if branch == "" {
		branch = "main"
	}

	return codescantaskv1.CodeScanTaskEvent{
		VCSProvider:        codescantaskv1.VCSProviderGitHub,
		VCSInstallationID:  inst.VCSInstallationID,
		VCSRepositoryID:    strconv.FormatInt(repo.GetID(), 10),
		RepositoryFullName: repo.GetFullName(),
		CommitSHA:          "",
		Branch:             branch,
		IsDefaultBranch:    true,
		ModifiedFiles:      nil,
	}
}

// readScanHistory reads the newest scans of a page of repositories. If the read fails it returns
// ok=false, and every repository on the page is published, which is what vcs-sync did before it
// could skip repositories.
func (p *VCSSyncProcessor) readScanHistory(
	ctx context.Context,
	inst gcpspanner.VCSInstallation,
	repos []*github.Repository,
) (map[string][]gcpspanner.RepositoryScanSummary, bool) {
	repoIDs := make([]string, 0, len(repos))
	for _, repo := range repos {
		if repo != nil {
			repoIDs = append(repoIDs, strconv.FormatInt(repo.GetID(), 10))
		}
	}

	readCtx, cancel := context.WithTimeout(ctx, scanHistoryTimeout)
	defer cancel()
	history, err := p.scanHistory.GetRecentRepositoryScans(readCtx, gcpspanner.VCSProviderGitHub, repoIDs,
		scanHistoryDepth)
	if err != nil {
		slog.ErrorContext(ctx, "failed to read scan history, publishing every repository on this page",
			"installation_id", inst.VCSInstallationID,
			"account", inst.AccountLogin,
			"error", err,
		)

		return nil, false
	}

	return history, true
}

// decide evaluates whether a repository needs a new scan, and warns about repositories whose
// scans keep failing.
func (p *VCSSyncProcessor) decide(
	ctx context.Context,
	now time.Time,
	task codescantaskv1.CodeScanTaskEvent,
	pushedAt time.Time,
	history map[string][]gcpspanner.RepositoryScanSummary,
	historyOK bool,
) scanDecision {
	if !historyOK {
		return publishBecause(reasonHistoryUnavailable)
	}

	scans := history[task.VCSRepositoryID]
	decision := decideScan(now, repoSnapshot{defaultBranch: task.Branch, pushedAt: pushedAt}, scans)

	if failures, failing := repeatedFailure(scans); failing {
		errorClass := ""
		if scans[0].ErrorClass != nil {
			errorClass = string(*scans[0].ErrorClass)
		}
		// The "Repository Repeatedly Failing to Scan" alert (infra/pubsub/alerts.tf) matches this
		// message and reads the repo field.
		p.logger.WarnContext(ctx, "repository repeatedly failing to scan",
			"repo", task.RepositoryFullName,
			"repository_id", task.VCSRepositoryID,
			"installation_id", task.VCSInstallationID,
			"consecutive_failures", failures,
			"error_class", errorClass,
			"last_scanned_at", scans[0].ScannedAt,
		)
	}

	p.logger.DebugContext(ctx, "scan task decision",
		"repo", task.RepositoryFullName,
		"publish", decision.publish,
		"reason", string(decision.reason),
	)

	return decision
}
