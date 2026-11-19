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
	"maps"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/codescan"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
)

// decideNow is the fixed current time used by the decision tests.
func decideNow() time.Time {
	return time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
}

// scanAgo is a branch-head scan of "main" with the current rules, started d before decideNow.
func scanAgo(status gcpspanner.ScanStatus, d time.Duration) gcpspanner.RepositoryScanSummary {
	var errClass *gcpspanner.ScanErrorClass
	if status == gcpspanner.ScanStatusFailed {
		errClass = new(gcpspanner.ScanErrorClassTimeout)
	}

	return gcpspanner.RepositoryScanSummary{
		ID:           "scan",
		ScanStatus:   status,
		ScannedAt:    decideNow().Add(-d),
		Branch:       "main",
		CommitSHA:    "",
		ErrorClass:   errClass,
		RulesVersion: new(codescan.RulesVersion),
	}
}

// failuresAgo is k consecutive failed scans, the newest started d before decideNow.
func failuresAgo(k int, d time.Duration) []gcpspanner.RepositoryScanSummary {
	scans := make([]gcpspanner.RepositoryScanSummary, 0, k+1)
	for i := range k {
		scans = append(scans, scanAgo(gcpspanner.ScanStatusFailed, d+time.Duration(i)*time.Hour))
	}

	return append(scans, scanAgo(gcpspanner.ScanStatusSuccess, d+time.Duration(k)*time.Hour))
}

func withScan(
	scan gcpspanner.RepositoryScanSummary,
	edit func(*gcpspanner.RepositoryScanSummary),
) []gcpspanner.RepositoryScanSummary {
	edit(&scan)

	return []gcpspanner.RepositoryScanSummary{scan}
}

func pushedAgo(d time.Duration) repoSnapshot {
	return repoSnapshot{defaultBranch: "main", pushedAt: decideNow().Add(-d)}
}

func TestDecideScan(t *testing.T) {
	t.Parallel()

	noPushedAt := repoSnapshot{defaultBranch: "main", pushedAt: time.Time{}}
	success := gcpspanner.ScanStatusSuccess

	testCases := []struct {
		name  string
		repo  repoSnapshot
		scans []gcpspanner.RepositoryScanSummary
		want  scanDecision
	}{
		{
			name:  "never scanned",
			repo:  pushedAgo(time.Hour),
			scans: nil,
			want:  publishBecause(reasonNoScan),
		},
		{
			name:  "unchanged since a successful scan",
			repo:  pushedAgo(2 * time.Hour),
			scans: []gcpspanner.RepositoryScanSummary{scanAgo(success, time.Hour)},
			want:  skipBecause(reasonUnchanged),
		},
		{
			name:  "unchanged since a truncated scan",
			repo:  pushedAgo(2 * time.Hour),
			scans: []gcpspanner.RepositoryScanSummary{scanAgo(gcpspanner.ScanStatusTruncated, time.Hour)},
			want:  skipBecause(reasonUnchanged),
		},
		{
			name:  "pushed after the scan",
			repo:  pushedAgo(10 * time.Minute),
			scans: []gcpspanner.RepositoryScanSummary{scanAgo(success, time.Hour)},
			want:  publishBecause(reasonPushedSinceScan),
		},
		{
			name:  "scan started less than a minute after the push",
			repo:  pushedAgo(time.Hour + 30*time.Second),
			scans: []gcpspanner.RepositoryScanSummary{scanAgo(success, time.Hour)},
			want:  publishBecause(reasonPushedSinceScan),
		},
		{
			name:  "scan started more than a minute after the push",
			repo:  pushedAgo(time.Hour + 2*time.Minute),
			scans: []gcpspanner.RepositoryScanSummary{scanAgo(success, time.Hour)},
			want:  skipBecause(reasonUnchanged),
		},
		{
			name: "latest scan pinned to a webhook commit",
			repo: pushedAgo(2 * time.Hour),
			scans: withScan(scanAgo(success, time.Hour), func(s *gcpspanner.RepositoryScanSummary) {
				s.CommitSHA = "abc123"
				s.Branch = "refs/heads/main"
			}),
			want: publishBecause(reasonCommitPinned),
		},
		{
			name: "webhook ref format is normalized",
			repo: pushedAgo(2 * time.Hour),
			scans: withScan(scanAgo(success, time.Hour), func(s *gcpspanner.RepositoryScanSummary) {
				s.Branch = "refs/heads/main"
			}),
			want: skipBecause(reasonUnchanged),
		},
		{
			name: "default branch changed",
			repo: pushedAgo(2 * time.Hour),
			scans: withScan(scanAgo(success, time.Hour), func(s *gcpspanner.RepositoryScanSummary) {
				s.Branch = "master"
			}),
			want: publishBecause(reasonBranchChanged),
		},
		{
			name: "scan written before rules versions were recorded",
			repo: pushedAgo(2 * time.Hour),
			scans: withScan(scanAgo(success, time.Hour), func(s *gcpspanner.RepositoryScanSummary) {
				s.RulesVersion = nil
			}),
			want: publishBecause(reasonRulesChanged),
		},
		{
			name: "scan made with older rules",
			repo: pushedAgo(2 * time.Hour),
			scans: withScan(scanAgo(success, time.Hour), func(s *gcpspanner.RepositoryScanSummary) {
				s.RulesVersion = new(codescan.RulesVersion - 1)
			}),
			want: publishBecause(reasonRulesChanged),
		},
		{
			name: "scan made with newer rules during a rollout",
			repo: pushedAgo(2 * time.Hour),
			scans: withScan(scanAgo(success, time.Hour), func(s *gcpspanner.RepositoryScanSummary) {
				s.RulesVersion = new(codescan.RulesVersion + 1)
			}),
			want: skipBecause(reasonUnchanged),
		},
		{
			name:  "no pushed_at and a recent scan",
			repo:  noPushedAt,
			scans: []gcpspanner.RepositoryScanSummary{scanAgo(success, time.Hour)},
			want:  skipBecause(reasonNoPushedAtRecent),
		},
		{
			name:  "no pushed_at and a scan older than a day",
			repo:  noPushedAt,
			scans: []gcpspanner.RepositoryScanSummary{scanAgo(success, 25*time.Hour)},
			want:  publishBecause(reasonNoPushedAtRefresh),
		},
		{
			name:  "unknown latest status",
			repo:  pushedAgo(2 * time.Hour),
			scans: []gcpspanner.RepositoryScanSummary{scanAgo("SOMETHING_NEW", time.Hour)},
			want:  publishBecause(reasonUnknownStatus),
		},
		{
			name:  "one failure, inside the 5 minute backoff",
			repo:  pushedAgo(3 * time.Hour),
			scans: failuresAgo(1, 3*time.Minute),
			want:  skipBecause(reasonBackoff),
		},
		{
			name:  "one failure, after the 5 minute backoff",
			repo:  pushedAgo(3 * time.Hour),
			scans: failuresAgo(1, 6*time.Minute),
			want:  publishBecause(reasonRetryAfterBackoff),
		},
		{
			name:  "three failures, inside the 20 minute backoff",
			repo:  pushedAgo(5 * time.Hour),
			scans: failuresAgo(3, 15*time.Minute),
			want:  skipBecause(reasonBackoff),
		},
		{
			name:  "three failures, after the 20 minute backoff",
			repo:  pushedAgo(5 * time.Hour),
			scans: failuresAgo(3, 25*time.Minute),
			want:  publishBecause(reasonRetryAfterBackoff),
		},
		{
			name:  "six failures, backoff capped at an hour",
			repo:  pushedAgo(8 * time.Hour),
			scans: failuresAgo(6, 61*time.Minute),
			want:  publishBecause(reasonRetryAfterBackoff),
		},
		{
			name:  "no pushed_at and failures still back off",
			repo:  noPushedAt,
			scans: failuresAgo(3, 15*time.Minute),
			want:  skipBecause(reasonBackoff),
		},
		{
			name:  "a push resets the backoff",
			repo:  pushedAgo(time.Minute),
			scans: failuresAgo(3, 2*time.Minute),
			want:  publishBecause(reasonPushSinceFailure),
		},
		{
			name: "failures of the previous default branch do not delay the new one",
			repo: pushedAgo(5 * time.Hour),
			scans: withScan(failuresAgo(3, 2*time.Minute)[0], func(s *gcpspanner.RepositoryScanSummary) {
				s.Branch = "master"
			}),
			want: publishBecause(reasonBranchChanged),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := decideScan(decideNow(), tc.repo, tc.scans); got != tc.want {
				t.Errorf("decideScan() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestFailureBackoff(t *testing.T) {
	t.Parallel()

	want := map[int]time.Duration{
		0: 5 * time.Minute,
		1: 5 * time.Minute,
		2: 10 * time.Minute,
		3: 20 * time.Minute,
		4: 40 * time.Minute,
		5: time.Hour,
		6: time.Hour,
	}
	for consecutive, wantBackoff := range want {
		if got := failureBackoff(consecutive); got != wantBackoff {
			t.Errorf("failureBackoff(%d) = %v, want %v", consecutive, got, wantBackoff)
		}
	}
}

func TestRepeatedFailure(t *testing.T) {
	t.Parallel()

	permanent := failuresAgo(5, time.Minute)
	permanent[0].ErrorClass = new(gcpspanner.ScanErrorClassPermanentClient)
	legacy := failuresAgo(4, time.Minute)
	legacy[0].ErrorClass = nil

	testCases := []struct {
		name      string
		scans     []gcpspanner.RepositoryScanSummary
		wantCount int
		wantFlag  bool
	}{
		{name: "no scans", scans: nil, wantCount: 0, wantFlag: false},
		{name: "three failures", scans: failuresAgo(3, time.Minute), wantCount: 3, wantFlag: false},
		{name: "four transient failures", scans: failuresAgo(4, time.Minute), wantCount: 4, wantFlag: true},
		{name: "permanent failures", scans: permanent, wantCount: 5, wantFlag: false},
		{name: "failures without an error class", scans: legacy, wantCount: 4, wantFlag: true},
		{
			name: "latest scan succeeded",
			scans: append([]gcpspanner.RepositoryScanSummary{scanAgo(gcpspanner.ScanStatusSuccess, 0)},
				failuresAgo(4, time.Minute)...),
			wantCount: 0,
			wantFlag:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			count, flag := repeatedFailure(tc.scans)
			if count != tc.wantCount || flag != tc.wantFlag {
				t.Errorf("repeatedFailure() = (%d, %v), want (%d, %v)", count, flag, tc.wantCount, tc.wantFlag)
			}
		})
	}
}

func TestParseSkipPolicy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		raw         string
		enabled     []string
		disabled    []string
		description string
	}{
		{raw: "", enabled: nil, disabled: []string{"1", "*"}, description: "disabled"},
		{raw: " , ", enabled: nil, disabled: []string{"1"}, description: "disabled"},
		{raw: "*", enabled: []string{"1", "2"}, disabled: nil, description: "enabled for all installations"},
		{
			raw:         " 1, ,2 ",
			enabled:     []string{"1", "2"},
			disabled:    []string{"3", "12"},
			description: "enabled for 2 installations",
		},
		{raw: "1,*", enabled: []string{"1", "3"}, disabled: nil, description: "enabled for all installations"},
	}

	for _, tc := range testCases {
		t.Run(tc.raw, func(t *testing.T) {
			t.Parallel()

			policy := ParseSkipPolicy(tc.raw)
			for _, id := range tc.enabled {
				if !policy.EnabledFor(id) {
					t.Errorf("ParseSkipPolicy(%q).EnabledFor(%q) = false, want true", tc.raw, id)
				}
			}
			for _, id := range tc.disabled {
				if policy.EnabledFor(id) {
					t.Errorf("ParseSkipPolicy(%q).EnabledFor(%q) = true, want false", tc.raw, id)
				}
			}
			if got := policy.String(); got != tc.description {
				t.Errorf("ParseSkipPolicy(%q).String() = %q, want %q", tc.raw, got, tc.description)
			}
		})
	}
}

func TestSyncStatsRecord(t *testing.T) {
	t.Parallel()

	stats := newSyncStats()
	if !stats.record(publishBecause(reasonNoScan), true) {
		t.Error("a publish decision must publish")
	}
	if stats.record(skipBecause(reasonUnchanged), true) {
		t.Error("a skip decision must skip when skipping is enabled")
	}
	if !stats.record(skipBecause(reasonUnchanged), false) {
		t.Error("a skip decision must publish when skipping is not enabled")
	}

	if stats.published != 2 || stats.skipped != 1 || stats.wouldSkip != 1 {
		t.Errorf("stats = published %d, skipped %d, would skip %d; want 2, 1, 1",
			stats.published, stats.skipped, stats.wouldSkip)
	}
	wantReasons := map[string]int{"no_scan": 1, "unchanged": 2}
	if got := stats.reasonCounts(); !maps.Equal(got, wantReasons) {
		t.Errorf("reasonCounts() = %v, want %v", got, wantReasons)
	}
}
