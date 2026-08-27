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
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/codescan"
	"github.com/GoogleChrome/webstatus.dev/lib/event"
	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gh"
	"github.com/google/uuid"
)

const (
	MaxBlobSizeBytes        int64 = 1024 * 1024 // 1MB per file
	MaxSubscriptionsPerRepo int   = 500
	DeletedBranchSHA              = "0000000000000000000000000000000000000000"
)

type TarballFetcher interface {
	DownloadTarball(ctx context.Context, owner, repo, ref string) (io.ReadCloser, error)
}

type TokenProvider interface {
	GetInstallationToken(ctx context.Context, installationID string) (string, error)
}

type ClientFactory func(token string) TarballFetcher

type SpannerSyncer interface {
	SynchronizeRepositoryCodeSubscriptions(
		ctx context.Context,
		provider gcpspanner.VCSProvider,
		repoID string,
		subs []gcpspanner.CodeSubscriptionInput,
	) error
	InsertCodeSubscriptionScanLog(ctx context.Context, scanLog gcpspanner.CodeSubscriptionScanLog) error
}

type Scanner struct {
	tokenProvider TokenProvider
	clientFactory ClientFactory
	spannerSyncer SpannerSyncer
}

func NewScanner(
	tokenProvider TokenProvider,
	clientFactory ClientFactory,
	spannerSyncer SpannerSyncer,
) *Scanner {
	if clientFactory == nil {
		clientFactory = func(token string) TarballFetcher {
			return gh.NewClient(token)
		}
	}

	return &Scanner{
		tokenProvider: tokenProvider,
		clientFactory: clientFactory,
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

func stripTarRootPrefix(path string) string {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return ""
	}

	return parts[1]
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

func (s *Scanner) scanTarballStream(
	ctx context.Context,
	archiveStream io.Reader,
) (
	map[string][]gcpspanner.SubscriptionOccurrence,
	map[string]map[gcpspanner.SubscriptionTrigger]struct{},
	int64,
	error,
) {
	gzr, err := gzip.NewReader(archiveStream)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to initialize gzip reader: %w", err)
	}
	defer func() {
		_ = gzr.Close()
	}()

	tr := tar.NewReader(gzr)
	occurrencesByQuery := make(map[string][]gcpspanner.SubscriptionOccurrence)
	triggersByQuery := make(map[string]map[gcpspanner.SubscriptionTrigger]struct{})
	var filesScanned int64

	for {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, nil, 0, ctxErr
		}

		header, tarErr := tr.Next()
		if errors.Is(tarErr, io.EOF) {
			break
		}
		if tarErr != nil {
			return nil, nil, 0, fmt.Errorf("failed to read tar entry: %w", tarErr)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		cleanPath := stripTarRootPrefix(header.Name)
		if cleanPath == "" {
			continue
		}

		if !codescan.IsSupportedWebExtension(filepath.Ext(cleanPath)) {
			continue
		}

		if header.Size > MaxBlobSizeBytes {
			slog.WarnContext(ctx, "skipping large file in archive", "path", cleanPath, "size", header.Size)

			continue
		}

		limitReader := io.LimitReader(tr, MaxBlobSizeBytes+1)
		directives, parseErr := codescan.ParseReader(
			limitReader,
			cleanPath,
			codescan.SubscriptionTriggerFeatureBaselinePromoteToWidely,
		)
		if parseErr != nil {
			slog.WarnContext(ctx, "failed to parse directives from file", "path", cleanPath, "error", parseErr)

			continue
		}

		filesScanned++
		for _, d := range directives {
			queryKey := strings.ToLower(strings.TrimSpace(d.TargetQuery))
			if _, exists := triggersByQuery[queryKey]; !exists {
				triggersByQuery[queryKey] = make(map[gcpspanner.SubscriptionTrigger]struct{})
			}
			spannerTrigger := gcpspanner.SubscriptionTrigger(d.Trigger)
			triggersByQuery[queryKey][spannerTrigger] = struct{}{}

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
	vcsProvider gcpspanner.VCSProvider,
	occurrencesByQuery map[string][]gcpspanner.SubscriptionOccurrence,
	triggersByQuery map[string]map[gcpspanner.SubscriptionTrigger]struct{},
) ([]gcpspanner.CodeSubscriptionInput, gcpspanner.ScanStatus) {
	scanStatus := gcpspanner.ScanStatusSuccess
	subscriptions := make([]gcpspanner.CodeSubscriptionInput, 0, len(occurrencesByQuery))

	for queryKey, occurrences := range occurrencesByQuery {
		if len(subscriptions) >= MaxSubscriptionsPerRepo {
			scanStatus = gcpspanner.ScanStatusTruncated

			break
		}

		triggersMap := triggersByQuery[queryKey]
		triggers := make([]gcpspanner.SubscriptionTrigger, 0, len(triggersMap))
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

// ProcessTask handles streaming tarball scanning and Spanner subscription synchronization.
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

	var token string
	if s.tokenProvider != nil && task.VCSInstallationID != "" {
		t, err := s.tokenProvider.GetInstallationToken(ctx, task.VCSInstallationID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to get installation token", "error", err, "installation_id", task.VCSInstallationID)
			s.recordFailedScanLog(ctx, task, now, fmt.Sprintf("failed to get installation token: %v", err))

			return fmt.Errorf("%w: failed to get installation token: %w", event.ErrTransientFailure, err)
		}
		token = t
	}

	owner, repo := splitOwnerRepo(task.RepositoryFullName)
	fetcher := s.clientFactory(token)
	archiveStream, err := fetcher.DownloadTarball(ctx, owner, repo, task.CommitSHA)
	if err != nil {
		slog.ErrorContext(ctx, "failed to download tarball", "error", err, "repo", task.RepositoryFullName)
		s.recordFailedScanLog(ctx, task, now, fmt.Sprintf("failed to download tarball: %v", err))

		return fmt.Errorf("%w: failed to download tarball: %w", event.ErrTransientFailure, err)
	}
	defer func() {
		_ = archiveStream.Close()
	}()

	occurrencesByQuery, triggersByQuery, filesScanned, scanErr := s.scanTarballStream(ctx, archiveStream)
	if scanErr != nil {
		slog.ErrorContext(ctx, "failed to scan repository tarball", "error", scanErr, "repo", task.RepositoryFullName)
		s.recordFailedScanLog(ctx, task, now, fmt.Sprintf("failed to scan tarball: %v", scanErr))

		return scanErr
	}

	subscriptions, scanStatus := s.buildSubscriptions(task, vcsProvider, occurrencesByQuery, triggersByQuery)

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
