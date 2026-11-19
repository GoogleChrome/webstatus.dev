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
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func insertScanHistoryRow(
	ctx context.Context,
	t *testing.T,
	repoID string,
	id string,
	scannedAt time.Time,
	status ScanStatus,
	errClass *ScanErrorClass,
	rulesVersion *int64,
) {
	t.Helper()

	if err := spannerClient.InsertCodeSubscriptionScanLog(ctx, CodeSubscriptionScanLog{
		ID:              id,
		VCSProvider:     VCSProviderGitHub,
		VCSRepositoryID: repoID,
		CommitSHA:       "",
		Branch:          "main",
		ScanStatus:      status,
		FilesScanned:    1,
		DirectivesFound: 0,
		ErrorMessage:    nil,
		ErrorClass:      errClass,
		RulesVersion:    rulesVersion,
		ScannedAt:       scannedAt,
	}); err != nil {
		t.Fatalf("InsertCodeSubscriptionScanLog(%s) failed: %v", id, err)
	}
}

func TestGetRecentRepositoryScans(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	// Spanner rejects future commit timestamps, so all scan times are in the past.
	base := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Microsecond)
	timeout := ScanErrorClassTimeout

	// repo-a has 8 scans, one per minute. The newest 6 must be returned, newest first.
	for i := range 8 {
		status := ScanStatusSuccess
		var errClass *ScanErrorClass
		if i%2 == 1 {
			status = ScanStatusFailed
			errClass = &timeout
		}
		insertScanHistoryRow(ctx, t, "repo-a", fmt.Sprintf("a-%d", i), base.Add(time.Duration(i)*time.Minute),
			status, errClass, new(int64(1)))
	}
	// repo-b has a single scan written before rules versions were recorded.
	insertScanHistoryRow(ctx, t, "repo-b", "b-0", base, ScanStatusTruncated, nil, nil)
	// Two scans of repo-tie started in the same microsecond: the larger ID comes first.
	insertScanHistoryRow(ctx, t, "repo-tie", "tie-1", base, ScanStatusSuccess, nil, nil)
	insertScanHistoryRow(ctx, t, "repo-tie", "tie-2", base, ScanStatusSuccess, nil, nil)
	// repo-odd has a status and error class this version doesn't know. They are returned as stored.
	insertScanHistoryRow(ctx, t, "repo-odd", "odd-0", base, ScanStatus("SOMETHING_NEW"),
		new(ScanErrorClass("NEW_CLASS")), nil)
	// The newest repo-a row belongs to another provider and must not be returned.
	otherProvider := CodeSubscriptionScanLog{
		ID:              "other-provider",
		VCSProvider:     VCSProvider("other"),
		VCSRepositoryID: "repo-a",
		CommitSHA:       "",
		Branch:          "main",
		ScanStatus:      ScanStatusSuccess,
		FilesScanned:    1,
		DirectivesFound: 0,
		ErrorMessage:    nil,
		ErrorClass:      nil,
		RulesVersion:    nil,
		ScannedAt:       base.Add(30 * time.Minute),
	}
	if err := spannerClient.InsertCodeSubscriptionScanLog(ctx, otherProvider); err != nil {
		t.Fatalf("InsertCodeSubscriptionScanLog(other provider) failed: %v", err)
	}

	got, err := spannerClient.GetRecentRepositoryScans(ctx, VCSProviderGitHub,
		[]string{"repo-a", "repo-b", "repo-tie", "repo-odd", "repo-never-scanned"}, 6)
	if err != nil {
		t.Fatalf("GetRecentRepositoryScans failed: %v", err)
	}

	gotIDs := make(map[string][]string, len(got))
	for repoID, scans := range got {
		for _, scan := range scans {
			gotIDs[repoID] = append(gotIDs[repoID], scan.ID)
		}
	}
	wantIDs := map[string][]string{
		"repo-a":   {"a-7", "a-6", "a-5", "a-4", "a-3", "a-2"},
		"repo-b":   {"b-0"},
		"repo-tie": {"tie-2", "tie-1"},
		"repo-odd": {"odd-0"},
	}
	if diff := cmp.Diff(wantIDs, gotIDs); diff != "" {
		t.Fatalf("scan IDs mismatch (-want +got):\n%s", diff)
	}
	if odd := got["repo-odd"][0]; odd.ScanStatus != "SOMETHING_NEW" || odd.ErrorClass == nil ||
		*odd.ErrorClass != "NEW_CLASS" {
		t.Errorf("repo-odd scan = %+v, want the unknown status and class as stored", odd)
	}

	wantNewestA := RepositoryScanSummary{
		ID:           "a-7",
		ScanStatus:   ScanStatusFailed,
		ScannedAt:    base.Add(7 * time.Minute),
		Branch:       "main",
		CommitSHA:    "",
		ErrorClass:   &timeout,
		RulesVersion: new(int64(1)),
	}
	if diff := cmp.Diff(wantNewestA, got["repo-a"][0]); diff != "" {
		t.Errorf("newest repo-a scan mismatch (-want +got):\n%s", diff)
	}
	if b := got["repo-b"][0]; b.RulesVersion != nil || b.ErrorClass != nil || b.ScanStatus != ScanStatusTruncated {
		t.Errorf("repo-b scan = %+v, want TRUNCATED with no rules version and no error class", b)
	}

	empty, err := spannerClient.GetRecentRepositoryScans(ctx, VCSProviderGitHub, nil, 6)
	if err != nil || len(empty) != 0 {
		t.Errorf("GetRecentRepositoryScans(no repos) = %v, %v; want an empty map", empty, err)
	}

	if _, err := spannerClient.GetRecentRepositoryScans(ctx, VCSProviderGitHub, []string{"repo-a"}, 0); !errors.Is(
		err, ErrInvalidScanHistoryLimit) {
		t.Errorf("GetRecentRepositoryScans(limit 0) error = %v, want ErrInvalidScanHistoryLimit", err)
	}
}
