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

	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gh"
	"github.com/google/go-github/v79/github"
)

// JobArguments represents CLI / Job runtime configuration.
type JobArguments struct{}

// NewJobArguments creates empty job arguments for scheduled execution.
func NewJobArguments() JobArguments {
	return JobArguments{}
}

// InstallationLister retrieves all active VCS installations from Spanner.
type InstallationLister interface {
	ListVCSInstallations(ctx context.Context) ([]gcpspanner.VCSInstallation, error)
}

// TokenProvider mints installation access tokens for VCS providers.
type TokenProvider interface {
	GetInstallationToken(ctx context.Context, installationID string) (string, error)
}

// GitHubRepoLister abstracts listing installation repositories from GitHub.
type GitHubRepoLister interface {
	ListInstallationRepositories(ctx context.Context, token string, opts *github.ListOptions) ([]*github.Repository, error)
}

// DefaultGitHubRepoLister creates default *gh.Client instances and lists repositories.
type DefaultGitHubRepoLister struct{}

func (l DefaultGitHubRepoLister) ListInstallationRepositories(
	ctx context.Context,
	token string,
	opts *github.ListOptions,
) ([]*github.Repository, error) {
	return gh.NewClient(token).ListInstallationRepositories(ctx, opts)
}

// TaskPublisher publishes scan tasks to Pub/Sub.
type TaskPublisher interface {
	PublishCodeScanTask(ctx context.Context, task codescantaskv1.CodeScanTaskEvent) error
}

// VCSSyncProcessor orchestrates the scheduled discovery and scan triggering of repositories.
type VCSSyncProcessor struct {
	installationLister InstallationLister
	tokenProvider      TokenProvider
	repoLister         GitHubRepoLister
	taskPublisher      TaskPublisher
}

// NewVCSSyncProcessor constructs a new scheduled VCS sync processor.
func NewVCSSyncProcessor(
	installationLister InstallationLister,
	tokenProvider TokenProvider,
	repoLister GitHubRepoLister,
	taskPublisher TaskPublisher,
) *VCSSyncProcessor {
	if repoLister == nil {
		repoLister = DefaultGitHubRepoLister{}
	}

	return &VCSSyncProcessor{
		installationLister: installationLister,
		tokenProvider:      tokenProvider,
		repoLister:         repoLister,
		taskPublisher:      taskPublisher,
	}
}

// Process runs the scheduled reconciliation job across all active installations.
func (p *VCSSyncProcessor) Process(ctx context.Context, _ JobArguments) error {
	slog.InfoContext(ctx, "starting scheduled VCS repository sync")

	installations, err := p.installationLister.ListVCSInstallations(ctx)
	if err != nil {
		return fmt.Errorf("failed to list VCS installations: %w", err)
	}

	slog.InfoContext(ctx, "discovered active VCS installations", "count", len(installations))

	var errs []error
	for _, inst := range installations {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := p.syncInstallation(ctx, inst); err != nil {
			slog.ErrorContext(ctx, "failed to sync installation",
				"provider", inst.VCSProvider,
				"installation_id", inst.VCSInstallationID,
				"account", inst.AccountLogin,
				"error", err,
			)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	slog.InfoContext(ctx, "completed scheduled VCS repository sync successfully")

	return nil
}

func (p *VCSSyncProcessor) syncInstallation(ctx context.Context, inst gcpspanner.VCSInstallation) error {
	switch inst.VCSProvider {
	case gcpspanner.VCSProviderGitHub:
		return p.syncGitHubInstallation(ctx, inst)
	default:
		slog.WarnContext(ctx, "unsupported VCS provider for sync", "provider", inst.VCSProvider)

		return nil
	}
}

func (p *VCSSyncProcessor) syncGitHubInstallation(ctx context.Context, inst gcpspanner.VCSInstallation) error {
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
				VCSProvider:        string(inst.VCSProvider),
				VCSInstallationID:  inst.VCSInstallationID,
				VCSRepositoryID:    strconv.FormatInt(repo.GetID(), 10),
				RepositoryFullName: repo.GetFullName(),
				CommitSHA:          branch,
				Branch:             branch,
				IsDefaultBranch:    true,
				ModifiedFiles:      nil,
			}

			if err := p.taskPublisher.PublishCodeScanTask(ctx, task); err != nil {
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
