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
	"fmt"
	"strings"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/codescan"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
)

// ScanHistoryReader reads the newest scan log rows of repositories.
type ScanHistoryReader interface {
	GetRecentRepositoryScans(
		ctx context.Context,
		vcsProvider gcpspanner.VCSProvider,
		repoIDs []string,
		perRepoLimit int,
	) (map[string][]gcpspanner.RepositoryScanSummary, error)
}

const (
	// scanHistoryDepth is how many recent scans are read per repository. It caps the consecutive
	// failure count used for backoff and for the repeated-failure warning.
	scanHistoryDepth = 6
	// scanHistoryTimeout bounds the scan history read for one page, so a stuck read fails open
	// instead of holding up the run.
	scanHistoryTimeout = 30 * time.Second
	// pushGracePeriod is how long after a push a scan must start to count as covering it. An
	// archive downloaded right after a push can still contain the previous commit.
	pushGracePeriod = time.Minute
	// noPushedAtRefresh is how often repositories without a pushed_at timestamp are rescanned.
	noPushedAtRefresh = 24 * time.Hour
	// failureBackoffBase is the wait after one failed scan. It doubles with each further
	// consecutive failure, up to failureBackoffMax.
	failureBackoffBase = 5 * time.Minute
	failureBackoffMax  = time.Hour
	// repeatedFailureThreshold is the number of consecutive failed scans that is reported as a
	// repository repeatedly failing to scan.
	repeatedFailureThreshold = 4
)

// scanReason explains why a repository was published or skipped. Reasons are logged as counts.
type scanReason string

const (
	reasonHistoryUnavailable scanReason = "history_unavailable"
	reasonNoScan             scanReason = "no_scan"
	reasonUnknownStatus      scanReason = "unknown_status"
	reasonPushSinceFailure   scanReason = "push_since_failure"
	reasonBackoff            scanReason = "backoff"
	reasonRetryAfterBackoff  scanReason = "retry_after_backoff"
	reasonCommitPinned       scanReason = "latest_commit_pinned"
	reasonBranchChanged      scanReason = "branch_changed"
	reasonRulesChanged       scanReason = "rules_changed"
	reasonNoPushedAtRefresh  scanReason = "no_pushed_at_refresh"
	reasonPushedSinceScan    scanReason = "pushed_since_scan"
	reasonNoPushedAtRecent   scanReason = "no_pushed_at_recent"
	reasonUnchanged          scanReason = "unchanged"
)

// scanDecision is whether a repository needs a new scan, and why.
type scanDecision struct {
	publish bool
	reason  scanReason
}

func publishBecause(reason scanReason) scanDecision {
	return scanDecision{publish: true, reason: reason}
}

func skipBecause(reason scanReason) scanDecision {
	return scanDecision{publish: false, reason: reason}
}

// repoSnapshot is what GitHub reported about a repository in this sync run.
type repoSnapshot struct {
	defaultBranch string
	// pushedAt is zero when GitHub did not report it.
	pushedAt time.Time
}

// decideScan reports whether a repository needs a new scan, given its newest scans (newest first).
//
// A repository is skipped only when its latest scan is a branch-head SUCCESS or TRUNCATED scan of
// the current default branch, made with the current scan rules, that started at least
// pushGracePeriod after the last push. Failed scans of the default branch are retried with
// exponential backoff, or right away after a new push.
func decideScan(now time.Time, repo repoSnapshot, scans []gcpspanner.RepositoryScanSummary) scanDecision {
	if len(scans) == 0 {
		return publishBecause(reasonNoScan)
	}

	latest := scans[0]
	hasPush := !repo.pushedAt.IsZero()
	pushedSince := hasPush && latest.ScannedAt.Before(repo.pushedAt.Add(pushGracePeriod))

	switch latest.ScanStatus {
	case gcpspanner.ScanStatusFailed:
		if pushedSince {
			return publishBecause(reasonPushSinceFailure)
		}
		if normalizeBranch(latest.Branch) != repo.defaultBranch {
			// Failures of a renamed or deleted default branch say nothing about the new one.
			return publishBecause(reasonBranchChanged)
		}
		if now.Before(latest.ScannedAt.Add(failureBackoff(consecutiveFailures(scans)))) {
			return skipBecause(reasonBackoff)
		}

		return publishBecause(reasonRetryAfterBackoff)
	case gcpspanner.ScanStatusSuccess, gcpspanner.ScanStatusTruncated:
		// Truncation is deterministic, so a TRUNCATED scan is as valid as a SUCCESS scan.
	default:
		return publishBecause(reasonUnknownStatus)
	}

	switch {
	case latest.CommitSHA != "":
		// A webhook scan pinned to a commit may not be the branch head.
		return publishBecause(reasonCommitPinned)
	case normalizeBranch(latest.Branch) != repo.defaultBranch:
		return publishBecause(reasonBranchChanged)
	case latest.RulesVersion == nil || *latest.RulesVersion < codescan.RulesVersion:
		return publishBecause(reasonRulesChanged)
	case !hasPush && now.Sub(latest.ScannedAt) >= noPushedAtRefresh:
		return publishBecause(reasonNoPushedAtRefresh)
	case pushedSince:
		return publishBecause(reasonPushedSinceScan)
	case !hasPush:
		return skipBecause(reasonNoPushedAtRecent)
	default:
		return skipBecause(reasonUnchanged)
	}
}

// consecutiveFailures counts the FAILED scans at the start of scans (newest first).
func consecutiveFailures(scans []gcpspanner.RepositoryScanSummary) int {
	count := 0
	for _, scan := range scans {
		if scan.ScanStatus != gcpspanner.ScanStatusFailed {
			break
		}
		count++
	}

	return count
}

// failureBackoff is how long to wait before retrying a repository after the given number of
// consecutive failed scans: 5m, 10m, 20m, 40m, then 1h.
func failureBackoff(consecutive int) time.Duration {
	backoff := failureBackoffBase
	for i := 1; i < consecutive && backoff < failureBackoffMax; i++ {
		backoff *= 2
	}

	return min(backoff, failureBackoffMax)
}

// repeatedFailure reports whether the newest scans are at least repeatedFailureThreshold
// consecutive failures whose latest error is not permanent, and how many there are. Permanent
// errors (for example a blocked or deleted repository) are expected to persist until a push.
func repeatedFailure(scans []gcpspanner.RepositoryScanSummary) (int, bool) {
	count := consecutiveFailures(scans)
	if count < repeatedFailureThreshold {
		return count, false
	}
	if class := scans[0].ErrorClass; class != nil && class.IsPermanent() {
		return count, false
	}

	return count, true
}

// normalizeBranch strips the "refs/heads/" prefix that webhook scans record.
func normalizeBranch(branch string) string {
	return strings.TrimPrefix(branch, "refs/heads/")
}

// SkipPolicy controls which installations skip repositories that do not need a new scan.
// Installations that are not allowed publish every repository, as before.
type SkipPolicy struct {
	allowAll      bool
	installations map[string]struct{}
}

// ParseSkipPolicy parses a comma-separated list of installation IDs, or "*" for every
// installation. An empty value disables skipping.
func ParseSkipPolicy(raw string) SkipPolicy {
	policy := SkipPolicy{allowAll: false, installations: make(map[string]struct{})}
	for part := range strings.SplitSeq(raw, ",") {
		id := strings.TrimSpace(part)
		switch id {
		case "":
		case "*":
			policy.allowAll = true
		default:
			policy.installations[id] = struct{}{}
		}
	}

	return policy
}

// EnabledFor reports whether the installation skips repositories that do not need a new scan.
func (p SkipPolicy) EnabledFor(installationID string) bool {
	if p.allowAll {
		return true
	}
	_, ok := p.installations[installationID]

	return ok
}

// String describes the policy for logs.
func (p SkipPolicy) String() string {
	switch {
	case p.allowAll:
		return "enabled for all installations"
	case len(p.installations) == 0:
		return "disabled"
	default:
		return fmt.Sprintf("enabled for %d installations", len(p.installations))
	}
}

// syncStats counts scan task decisions across one vcs-sync run.
type syncStats struct {
	published int
	skipped   int
	// wouldSkip counts repositories that were published only because skipping is not enabled for
	// their installation.
	wouldSkip int
	reasons   map[scanReason]int
}

func newSyncStats() *syncStats {
	return &syncStats{published: 0, skipped: 0, wouldSkip: 0, reasons: make(map[scanReason]int)}
}

// record counts a decision and reports whether the repository should be published.
func (s *syncStats) record(decision scanDecision, skipEnabled bool) bool {
	s.reasons[decision.reason]++
	if !decision.publish && skipEnabled {
		s.skipped++

		return false
	}
	if !decision.publish {
		s.wouldSkip++
	}
	s.published++

	return true
}

// reasonCounts returns the reason counts with string keys, for logging.
func (s *syncStats) reasonCounts() map[string]int {
	counts := make(map[string]int, len(s.reasons))
	for reason, count := range s.reasons {
		counts[string(reason)] = count
	}

	return counts
}
