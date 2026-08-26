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

package scanner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/codescan"
	"github.com/GoogleChrome/webstatus.dev/lib/event"
	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/google/go-github/v79/github"
	"github.com/google/uuid"
)

const (
	MaxBlobSizeBytes        int64 = 1024 * 1024 // 1MB per file
	MaxScannedFiles         int   = 10000       // 10k files max
	MaxSubscriptionsPerRepo int   = 500
	DeletedBranchSHA              = "0000000000000000000000000000000000000000"
)

var (
	ErrNonDefaultBranch = errors.New("ignoring non-default branch commit")
	ErrDeletedBranch    = errors.New("ignoring deleted branch event")
	ErrTreeTruncated    = errors.New("repository git tree exceeds GitHub API limit")
)

type GitFetcher interface {
	GetCommitTree(ctx context.Context, owner, repo, sha string) (*github.Tree, error)
	GetBlobContent(ctx context.Context, owner, repo, sha string) ([]byte, error)
}

type SpannerSyncer interface {
	SynchronizeRepositoryCodeSubscriptions(
		ctx context.Context, provider gcpspanner.VCSProvider, repoID string, subs []gcpspanner.CodeSubscriptionInput) error
	InsertCodeSubscriptionScanLog(ctx context.Context, scanLog gcpspanner.CodeSubscriptionScanLog) error
}

type Scanner struct {
	gitFetcher    GitFetcher
	spannerSyncer SpannerSyncer
}

func NewScanner(gitFetcher GitFetcher, spannerSyncer SpannerSyncer) *Scanner {
	return &Scanner{
		gitFetcher:    gitFetcher,
		spannerSyncer: spannerSyncer,
	}
}

func splitOwnerRepo(fullName string) (string, string) {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return "", fullName
}

func (s *Scanner) recordFailedScanLog(
	ctx context.Context,
	task codescantaskv1.CodeScanTaskEvent,
	now time.Time,
	errMsg string,
) {
	errCopy := errMsg
	vcsProvider, err := gcpspanner.ParseVCSProvider(task.VCSProvider)
	if err != nil {
		vcsProvider = gcpspanner.VCSProviderGitHub
	}
	scanLog := gcpspanner.CodeSubscriptionScanLog{
		ID:              uuid.NewString(),
		VCSProvider:     vcsProvider,
		VCSRepositoryID: task.VCSRepositoryID,
		CommitSHA:       task.CommitSHA,
		Branch:          task.Branch,
		ScanStatus:      gcpspanner.ScanStatusFailed,
		FilesScanned:    0,
		DirectivesFound: 0,
		ErrorMessage:    &errCopy,
		ScannedAt:       now,
	}

	if logErr := s.spannerSyncer.InsertCodeSubscriptionScanLog(ctx, scanLog); logErr != nil {
		slog.ErrorContext(ctx, "failed to insert failed scan log", "error", logErr)
	}
}

func (s *Scanner) scanFiles(
	ctx context.Context,
	task codescantaskv1.CodeScanTaskEvent,
	entries []*github.TreeEntry,
) (map[string][]gcpspanner.SubscriptionOccurrence, map[string]map[string]struct{}, int64, error) {
	owner, repo := splitOwnerRepo(task.RepositoryFullName)
	occurrencesByQuery := make(map[string][]gcpspanner.SubscriptionOccurrence)
	triggersByQuery := make(map[string]map[string]struct{})
	var filesScanned int64

	for _, entry := range entries {
		if entry == nil || entry.GetType() != "blob" || !codescan.IsSupportedWebExtension(filepath.Ext(entry.GetPath())) {
			continue
		}
		if int64(entry.GetSize()) > MaxBlobSizeBytes {
			slog.WarnContext(ctx, "skipping giant file exceeding 1MB", "path", entry.GetPath(), "size", entry.GetSize())

			continue
		}

		content, blobErr := s.gitFetcher.GetBlobContent(ctx, owner, repo, entry.GetSHA())
		if blobErr != nil {
			slog.ErrorContext(ctx, "failed to get blob content, aborting scan for retry",
				"path", entry.GetPath(), "error", blobErr)

			return nil, nil, 0, fmt.Errorf(
				"%w: failed to fetch blob for %s: %w",
				event.ErrTransientFailure,
				entry.GetPath(),
				blobErr,
			)
		}
		if int64(len(content)) > MaxBlobSizeBytes {
			slog.WarnContext(ctx, "skipping giant file exceeding 1MB after fetch", "path", entry.GetPath(), "size", len(content))

			continue
		}
		filesScanned++

		directives := codescan.ParseFileDirectives(content, entry.GetPath(), "")
		for _, d := range directives {
			queryKey := strings.ToLower(strings.TrimSpace(d.TargetQuery))
			if _, exists := triggersByQuery[queryKey]; !exists {
				triggersByQuery[queryKey] = make(map[string]struct{})
			}
			triggersByQuery[queryKey][string(d.Trigger)] = struct{}{}
			occurrencesByQuery[queryKey] = append(occurrencesByQuery[queryKey], gcpspanner.SubscriptionOccurrence{
				FilePath:       d.FilePath,
				LineNumber:     int64(d.LineNumber),
				CommentSnippet: d.RawSnippet,
			})
		}
	}

	return occurrencesByQuery, triggersByQuery, filesScanned, nil
}

func (s *Scanner) buildSubscriptions(
	task codescantaskv1.CodeScanTaskEvent,
	occurrencesByQuery map[string][]gcpspanner.SubscriptionOccurrence,
	triggersByQuery map[string]map[string]struct{},
) ([]gcpspanner.CodeSubscriptionInput, gcpspanner.ScanStatus) {
	scanStatus := gcpspanner.ScanStatusSuccess
	subscriptions := make([]gcpspanner.CodeSubscriptionInput, 0, len(occurrencesByQuery))

	for queryKey, occurrences := range occurrencesByQuery {
		if len(subscriptions) >= MaxSubscriptionsPerRepo {
			scanStatus = gcpspanner.ScanStatusTruncated

			break
		}

		vcsProvider, err := gcpspanner.ParseVCSProvider(task.VCSProvider)
		if err != nil {
			vcsProvider = gcpspanner.VCSProviderGitHub
		}

		triggersMap := triggersByQuery[queryKey]
		triggers := make([]string, 0, len(triggersMap))
		for t := range triggersMap {
			triggers = append(triggers, t)
		}
		slices.Sort(triggers)

		subscriptions = append(subscriptions, gcpspanner.CodeSubscriptionInput{
			VCSProvider:        vcsProvider,
			VCSInstallationID:  task.VCSInstallationID,
			VCSRepositoryID:    task.VCSRepositoryID,
			RepositoryFullName: task.RepositoryFullName,
			TargetQuery:        queryKey,
			Triggers:           triggers,
			Occurrences:        occurrences,
		})
	}

	return subscriptions, scanStatus
}

// ProcessTask handles full tree scanning and Spanner subscription synchronization.
func (s *Scanner) ProcessTask(ctx context.Context, task codescantaskv1.CodeScanTaskEvent) error {
	now := time.Now().UTC()

	if task.CommitSHA == DeletedBranchSHA {
		slog.InfoContext(ctx, "skipping deleted branch commit", "repo", task.RepositoryFullName)

		return nil
	}
	if !task.IsDefaultBranch {
		slog.InfoContext(ctx, "skipping non-default branch commit", "repo", task.RepositoryFullName, "branch", task.Branch)

		return nil
	}

	vcsProvider, provErr := gcpspanner.ParseVCSProvider(task.VCSProvider)
	if provErr != nil {
		s.recordFailedScanLog(ctx, task, now, fmt.Sprintf("invalid vcs provider: %v", provErr))

		return fmt.Errorf("invalid vcs provider: %w", provErr)
	}

	owner, repo := splitOwnerRepo(task.RepositoryFullName)
	tree, err := s.gitFetcher.GetCommitTree(ctx, owner, repo, task.CommitSHA)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch git tree", "error", err, "repo", task.RepositoryFullName)
		s.recordFailedScanLog(ctx, task, now, fmt.Sprintf("failed to fetch git tree: %v", err))

		return fmt.Errorf("%w: failed to fetch git tree: %w", event.ErrTransientFailure, err)
	}

	occurrencesByQuery, triggersByQuery, filesScanned, scanErr := s.scanFiles(ctx, task, tree.Entries)
	if scanErr != nil {
		slog.ErrorContext(ctx, "failed to scan repository files", "error", scanErr, "repo", task.RepositoryFullName)
		s.recordFailedScanLog(ctx, task, now, fmt.Sprintf("failed to scan files: %v", scanErr))

		return scanErr
	}
	subscriptions, scanStatus := s.buildSubscriptions(task, occurrencesByQuery, triggersByQuery)

	if tree.GetTruncated() {
		scanStatus = gcpspanner.ScanStatusTruncated
	}

	var directivesFound int64
	for _, sub := range subscriptions {
		directivesFound += int64(len(sub.Occurrences))
	}

	if syncErr := s.spannerSyncer.SynchronizeRepositoryCodeSubscriptions(
		ctx, vcsProvider, task.VCSRepositoryID, subscriptions); syncErr != nil {
		slog.ErrorContext(ctx, "failed to synchronize code subscriptions", "error", syncErr, "repo", task.RepositoryFullName)
		s.recordFailedScanLog(ctx, task, now, fmt.Sprintf("spanner sync error: %v", syncErr))

		return fmt.Errorf("%w: failed to sync code subscriptions: %w", event.ErrTransientFailure, syncErr)
	}

	scanLog := gcpspanner.CodeSubscriptionScanLog{
		ID:              uuid.NewString(),
		VCSProvider:     vcsProvider,
		VCSRepositoryID: task.VCSRepositoryID,
		CommitSHA:       task.CommitSHA,
		Branch:          task.Branch,
		ScanStatus:      scanStatus,
		FilesScanned:    filesScanned,
		DirectivesFound: directivesFound,
		ErrorMessage:    nil,
		ScannedAt:       now,
	}

	if logErr := s.spannerSyncer.InsertCodeSubscriptionScanLog(ctx, scanLog); logErr != nil {
		slog.ErrorContext(ctx, "failed to insert scan log", "error", logErr, "repo", task.RepositoryFullName)
	}

	slog.InfoContext(ctx, "successfully scanned repository",
		"repo", task.RepositoryFullName,
		"files_scanned", filesScanned,
		"subscriptions_synced", len(subscriptions),
		"directives_found", directivesFound,
		"status", scanStatus)

	return nil
}
