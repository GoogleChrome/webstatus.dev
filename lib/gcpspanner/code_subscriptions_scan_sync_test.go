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

package gcpspanner

import (
	"context"
	"errors"
	"testing"
	"time"
)

func pathSet(paths ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		set[p] = struct{}{}
	}

	return set
}

func subscriptionWithPaths(target string, paths ...string) CodeSubscription {
	occurrences := make([]SubscriptionOccurrence, 0, len(paths))
	for i, p := range paths {
		occurrences = append(occurrences, SubscriptionOccurrence{
			FilePath:       p,
			LineNumber:     int64(i + 1),
			CommentSnippet: "// TODO(baseline/" + target + ")",
		})
	}

	return CodeSubscription{
		ID:                 "sub-" + target,
		VCSProvider:        VCSProviderGitHub,
		VCSInstallationID:  "inst-1",
		VCSRepositoryID:    "repo-1",
		RepositoryFullName: "owner/repo",
		TargetQuery:        target,
		Triggers:           []SubscriptionTrigger{SubscriptionTriggerFeatureBaselinePromoteToWidely},
		Status:             SubscriptionActive,
		Occurrences:        occurrences,
		CreatedAt:          time.Time{},
		UpdatedAt:          time.Time{},
	}
}

func TestObsolescenceScopeShouldObsolete(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		scope    ObsolescenceScope
		existing CodeSubscription
		want     bool
	}{
		{
			name: "complete scan, directive gone",
			scope: ObsolescenceScope{
				Truncated: false, FoundTargets: nil, ParsedPaths: nil, UnscannedPaths: nil,
			},
			existing: subscriptionWithPaths("popover", "src/a.ts"),
			want:     true,
		},
		{
			name: "complete scan, directive found but not persisted (subscription cap)",
			scope: ObsolescenceScope{
				Truncated: false, FoundTargets: pathSet("popover"), ParsedPaths: nil, UnscannedPaths: nil,
			},
			existing: subscriptionWithPaths("popover", "src/a.ts"),
			want:     false,
		},
		{
			name: "complete scan, file skipped for size",
			scope: ObsolescenceScope{
				Truncated: false, FoundTargets: nil, ParsedPaths: nil, UnscannedPaths: pathSet("src/big.js"),
			},
			existing: subscriptionWithPaths("popover", "src/a.ts", "src/big.js"),
			want:     false,
		},
		{
			name: "truncated scan, every file was parsed",
			scope: ObsolescenceScope{
				Truncated: true, FoundTargets: nil, ParsedPaths: pathSet("src/a.ts", "src/b.ts"), UnscannedPaths: nil,
			},
			existing: subscriptionWithPaths("popover", "src/a.ts", "src/b.ts"),
			want:     true,
		},
		{
			name: "truncated scan, a file is past the cutoff",
			scope: ObsolescenceScope{
				Truncated: true, FoundTargets: nil, ParsedPaths: pathSet("src/a.ts"), UnscannedPaths: nil,
			},
			existing: subscriptionWithPaths("popover", "src/a.ts", "src/z.ts"),
			want:     false,
		},
		{
			name: "truncated scan, no stored occurrences",
			scope: ObsolescenceScope{
				Truncated: true, FoundTargets: nil, ParsedPaths: pathSet("src/a.ts"), UnscannedPaths: nil,
			},
			existing: subscriptionWithPaths("dialog"),
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.scope.shouldObsolete(tc.existing); got != tc.want {
				t.Errorf("shouldObsolete() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestScanStartedAfter(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 11, 17, 17, 20, 0, 0, time.UTC)
	testCases := []struct {
		name   string
		aStart time.Time
		aID    string
		bStart time.Time
		bID    string
		want   bool
	}{
		{name: "later start", aStart: base.Add(time.Second), aID: "a", bStart: base, bID: "b", want: true},
		{name: "earlier start", aStart: base, aID: "z", bStart: base.Add(time.Second), bID: "a", want: false},
		{name: "tie, larger ID wins", aStart: base, aID: "b", bStart: base, bID: "a", want: true},
		{name: "tie, smaller ID loses", aStart: base, aID: "a", bStart: base, bID: "b", want: false},
		{name: "same scan", aStart: base, aID: "a", bStart: base, bID: "a", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := scanStartedAfter(tc.aStart, tc.aID, tc.bStart, tc.bID); got != tc.want {
				t.Errorf("scanStartedAfter() = %v, want %v", got, tc.want)
			}
		})
	}
}

const scanSyncTestRepoID = "repo-scan-sync"

func scanSyncInput(target, path string) CodeSubscriptionInput {
	return CodeSubscriptionInput{
		VCSProvider:        VCSProviderGitHub,
		VCSInstallationID:  "inst-scan-sync",
		VCSRepositoryID:    scanSyncTestRepoID,
		RepositoryFullName: "owner/scan-sync",
		TargetQuery:        target,
		Triggers:           []SubscriptionTrigger{SubscriptionTriggerFeatureBaselinePromoteToWidely},
		Occurrences: []SubscriptionOccurrence{
			{FilePath: path, LineNumber: 1, CommentSnippet: "// TODO(baseline/" + target + ")"},
		},
	}
}

func scanSyncRequest(
	scanID string,
	start time.Time,
	commitSHA string,
	status ScanStatus,
	scope ObsolescenceScope,
	subs ...CodeSubscriptionInput,
) RepositoryScanSync {
	if subs == nil {
		subs = []CodeSubscriptionInput{}
	}

	return RepositoryScanSync{
		VCSProvider:     VCSProviderGitHub,
		VCSRepositoryID: scanSyncTestRepoID,
		Subscriptions:   subs,
		Obsolescence:    scope,
		ScanLog: CodeSubscriptionScanLog{
			ID:              scanID,
			VCSProvider:     VCSProviderGitHub,
			VCSRepositoryID: scanSyncTestRepoID,
			CommitSHA:       commitSHA,
			Branch:          "main",
			ScanStatus:      status,
			FilesScanned:    int64(len(subs)),
			DirectivesFound: int64(len(subs)),
			ErrorMessage:    nil,
			ErrorClass:      nil,
			RulesVersion:    nil,
			ScannedAt:       start,
		},
	}
}

func completeScope() ObsolescenceScope {
	return ObsolescenceScope{Truncated: false, FoundTargets: nil, ParsedPaths: nil, UnscannedPaths: nil}
}

func repositorySubscriptionStatuses(ctx context.Context, t *testing.T) map[string]SubscriptionStatus {
	t.Helper()

	spannerSubs, err := newAllByKeysEntityReader[
		codeSubscriptionAllByRepoMapper, codeSubscriptionAllByRepoKey, spannerCodeSubscription,
	](spannerClient).readAllByKeys(ctx, codeSubscriptionAllByRepoKey{
		VCSProvider: VCSProviderGitHub,
		RepoID:      scanSyncTestRepoID,
	})
	if err != nil {
		t.Fatalf("failed to read repository subscriptions: %v", err)
	}

	statuses := make(map[string]SubscriptionStatus, len(spannerSubs))
	for _, s := range spannerSubs {
		statuses[s.TargetQuery] = SubscriptionStatus(s.Status)
	}

	return statuses
}

func mustSyncRepositoryScan(ctx context.Context, t *testing.T, req RepositoryScanSync, wantApplied bool) {
	t.Helper()

	applied, err := spannerClient.SyncRepositoryScan(ctx, req)
	if err != nil {
		t.Fatalf("SyncRepositoryScan(%s) failed: %v", req.ScanLog.ID, err)
	}
	if applied != wantApplied {
		t.Fatalf("SyncRepositoryScan(%s) applied = %v, want %v", req.ScanLog.ID, applied, wantApplied)
	}
}

func assertStatuses(ctx context.Context, t *testing.T, want map[string]SubscriptionStatus) {
	t.Helper()

	got := repositorySubscriptionStatuses(ctx, t)
	if len(got) != len(want) {
		t.Fatalf("subscription statuses = %v, want %v", got, want)
	}
	for target, status := range want {
		if got[target] != status {
			t.Fatalf("subscription statuses = %v, want %v", got, want)
		}
	}
}

func TestSyncRepositoryScan(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	// Spanner rejects future commit timestamps, so all start times are in the past.
	base := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Microsecond)

	if _, err := spannerClient.SyncRepositoryScan(ctx, RepositoryScanSync{
		VCSProvider:     VCSProviderGitHub,
		VCSRepositoryID: scanSyncTestRepoID,
		Subscriptions:   nil,
		Obsolescence:    completeScope(),
		ScanLog:         scanSyncRequest("", base, "", ScanStatusSuccess, completeScope()).ScanLog,
	}); !errors.Is(err, ErrInvalidScanLog) {
		t.Fatalf("expected ErrInvalidScanLog for a scan log without an ID, got %v", err)
	}

	// Branch-head scan Y applies.
	mustSyncRepositoryScan(ctx, t, scanSyncRequest("scan-y", base.Add(100*time.Second), "", ScanStatusSuccess,
		completeScope(), scanSyncInput("popover", "src/a.ts")), true)
	assertStatuses(ctx, t, map[string]SubscriptionStatus{"popover": SubscriptionActive})

	// A slow webhook scan pinned to an older commit started before Y and finishes after it.
	// It must not overwrite Y's results, and it writes no scan log row.
	mustSyncRepositoryScan(ctx, t, scanSyncRequest("scan-w-a", base.Add(1*time.Second), "commit-a", ScanStatusSuccess,
		completeScope(), scanSyncInput("dialog", "src/b.ts")), false)
	assertStatuses(ctx, t, map[string]SubscriptionStatus{"popover": SubscriptionActive})
	if _, err := spannerClient.GetCodeSubscriptionScanLog(ctx, "scan-w-a"); !errors.Is(err, ErrScanLogNotFound) {
		t.Fatalf("expected no scan log row for a superseded scan, got %v", err)
	}

	// A newer branch-head scan Z applies: popover was removed from src/a.ts, dialog was added.
	mustSyncRepositoryScan(ctx, t, scanSyncRequest("scan-z", base.Add(200*time.Second), "", ScanStatusSuccess,
		completeScope(), scanSyncInput("dialog", "src/b.ts")), true)
	assertStatuses(ctx, t, map[string]SubscriptionStatus{
		"popover": SubscriptionObsolete,
		"dialog":  SubscriptionActive,
	})

	// Same start time as Z: the smaller ID is superseded, the larger ID applies.
	mustSyncRepositoryScan(ctx, t, scanSyncRequest("scan-a-tie", base.Add(200*time.Second), "", ScanStatusSuccess,
		completeScope()), false)
	mustSyncRepositoryScan(ctx, t, scanSyncRequest("scan-zz-tie", base.Add(200*time.Second), "", ScanStatusSuccess,
		completeScope(), scanSyncInput("dialog", "src/b.ts")), true)

	// A truncated scan that never reached src/b.ts must not mark dialog OBSOLETE.
	mustSyncRepositoryScan(ctx, t, scanSyncRequest("scan-truncated", base.Add(300*time.Second), "", ScanStatusTruncated,
		ObsolescenceScope{
			Truncated:      true,
			FoundTargets:   nil,
			ParsedPaths:    pathSet("src/a.ts"),
			UnscannedPaths: nil,
		}), true)
	assertStatuses(ctx, t, map[string]SubscriptionStatus{
		"popover": SubscriptionObsolete,
		"dialog":  SubscriptionActive,
	})

	// The SUCCESS/TRUNCATED rows were written atomically with the sync.
	for _, id := range []string{"scan-y", "scan-z", "scan-zz-tie", "scan-truncated"} {
		if _, err := spannerClient.GetCodeSubscriptionScanLog(ctx, id); err != nil {
			t.Fatalf("expected scan log row %q to exist, got %v", id, err)
		}
	}
}

func TestScanLogErrorClassRoundTrip(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	errMsg := "download timed out"
	class := ScanErrorClassTimeout
	if err := spannerClient.InsertCodeSubscriptionScanLog(ctx, CodeSubscriptionScanLog{
		ID:              "scan-failed-timeout",
		VCSProvider:     VCSProviderGitHub,
		VCSRepositoryID: scanSyncTestRepoID,
		CommitSHA:       "",
		Branch:          "main",
		ScanStatus:      ScanStatusFailed,
		FilesScanned:    0,
		DirectivesFound: 0,
		ErrorMessage:    &errMsg,
		ErrorClass:      &class,
		RulesVersion:    nil,
		ScannedAt:       time.Now().UTC().Add(-1 * time.Minute),
	}); err != nil {
		t.Fatalf("InsertCodeSubscriptionScanLog failed: %v", err)
	}

	got, err := spannerClient.GetCodeSubscriptionScanLog(ctx, "scan-failed-timeout")
	if err != nil {
		t.Fatalf("GetCodeSubscriptionScanLog failed: %v", err)
	}
	if got.ErrorClass == nil || *got.ErrorClass != ScanErrorClassTimeout {
		t.Fatalf("ErrorClass = %v, want %v", got.ErrorClass, ScanErrorClassTimeout)
	}
}

func TestParseScanErrorClass(t *testing.T) {
	t.Parallel()

	if _, err := ParseScanErrorClass("NOT_A_CLASS"); !errors.Is(err, ErrUnknownScanErrorClass) {
		t.Fatalf("expected ErrUnknownScanErrorClass, got %v", err)
	}
	got, err := ParseScanErrorClass(string(ScanErrorClassStall))
	if err != nil || got != ScanErrorClassStall {
		t.Fatalf("ParseScanErrorClass(STALL) = %v, %v", got, err)
	}
	if !ScanErrorClassPermanentClient.IsPermanent() || ScanErrorClassStall.IsPermanent() {
		t.Fatalf("unexpected IsPermanent classification")
	}
}
