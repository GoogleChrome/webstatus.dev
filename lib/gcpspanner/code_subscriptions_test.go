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

	"github.com/google/uuid"
)

func TestCodeSubscriptions(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	now := time.Now().UTC().Truncate(time.Microsecond)
	installationID := "inst-" + uuid.NewString()
	repoID := "repo-" + uuid.NewString()
	repoFullName := "GoogleChrome/webstatus.dev"

	// 1. Create installation record
	writeLevel := GitHubPermissionLevelWrite
	readLevel := GitHubPermissionLevelRead
	inst := VCSInstallation{
		ID:                  "",
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   installationID,
		AccountLogin:        "GoogleChrome",
		AccountType:         "Organization",
		RepositorySelection: "selected",
		Permissions: VCSPermissions{
			GitHub: &GitHubPermissions{
				Issues:       &writeLevel,
				Contents:     &readLevel,
				Metadata:     nil,
				PullRequests: nil,
				Workflows:    nil,
				Actions:      nil,
			},
		},
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}
	if _, err := spannerClient.UpsertVCSInstallation(ctx, inst); err != nil {
		t.Fatalf("UpsertVCSInstallation failed: %v", err)
	}

	// 2. Test SynchronizeRepositoryCodeSubscriptions (Initial Ingestion)
	sub1 := CodeSubscription{
		ID:                 uuid.NewString(),
		VCSProvider:        VCSProviderGitHub,
		VCSInstallationID:  installationID,
		VCSRepositoryID:    repoID,
		RepositoryFullName: repoFullName,
		TargetQuery:        "id:view-transitions",
		Triggers:           []string{"feature_baseline_to_widely"},
		Status:             SubscriptionActive,
		Occurrences: []SubscriptionOccurrence{
			{FilePath: "src/app.ts", LineNumber: 10, CommentSnippet: "// TODO(baseline/view-transitions): transition"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	sub2 := CodeSubscription{
		ID:                 uuid.NewString(),
		VCSProvider:        VCSProviderGitHub,
		VCSInstallationID:  installationID,
		VCSRepositoryID:    repoID,
		RepositoryFullName: repoFullName,
		TargetQuery:        "id:subgrid",
		Triggers:           []string{"feature_baseline_to_widely"},
		Status:             SubscriptionActive,
		Occurrences: []SubscriptionOccurrence{
			{FilePath: "src/grid.css", LineNumber: 5, CommentSnippet: "// TODO(baseline/subgrid): grid"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := spannerClient.SynchronizeRepositoryCodeSubscriptions(
		ctx, VCSProviderGitHub, repoID, []CodeSubscription{sub1, sub2}); err != nil {
		t.Fatalf("SynchronizeRepositoryCodeSubscriptions failed: %v", err)
	}

	// 3. Test ListCodeSubscriptionsByRepository with pagination
	listReq := ListCodeSubscriptionsRequest{
		VCSProvider: VCSProviderGitHub,
		RepoID:      repoID,
		PageSize:    10,
		PageToken:   nil,
	}
	list, nextPageToken, err := spannerClient.ListCodeSubscriptionsByRepository(ctx, listReq)
	if err != nil {
		t.Fatalf("ListCodeSubscriptionsByRepository failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 active subscriptions, got %d", len(list))
	}
	if nextPageToken != nil {
		t.Fatalf("expected nil nextPageToken for 2 items with pageSize 10, got %s", *nextPageToken)
	}

	// 3b. Test pagination page by page (pageSize = 1)
	pagedReq1 := ListCodeSubscriptionsRequest{
		VCSProvider: VCSProviderGitHub,
		RepoID:      repoID,
		PageSize:    1,
		PageToken:   nil,
	}
	pagedList1, token1, err := spannerClient.ListCodeSubscriptionsByRepository(ctx, pagedReq1)
	if err != nil {
		t.Fatalf("ListCodeSubscriptionsByRepository page 1 failed: %v", err)
	}
	if len(pagedList1) != 1 || token1 == nil {
		t.Fatalf("expected 1 item and valid next token on page 1, got len=%d token=%v", len(pagedList1), token1)
	}

	pagedReq2 := ListCodeSubscriptionsRequest{
		VCSProvider: VCSProviderGitHub,
		RepoID:      repoID,
		PageSize:    1,
		PageToken:   token1,
	}
	pagedList2, _, err := spannerClient.ListCodeSubscriptionsByRepository(ctx, pagedReq2)
	if err != nil {
		t.Fatalf("ListCodeSubscriptionsByRepository page 2 failed: %v", err)
	}
	if len(pagedList2) != 1 {
		t.Fatalf("expected 1 item on page 2, got %d", len(pagedList2))
	}
	if pagedList1[0].ID == pagedList2[0].ID {
		t.Fatalf("expected different items on pages 1 and 2, got same ID: %s", pagedList1[0].ID)
	}

	// 4. Test ListCodeSubscriptionsByTargetQuery
	targetList, err := spannerClient.ListCodeSubscriptionsByTargetQuery(
		ctx, "id:view-transitions", "feature_baseline_to_widely")
	if err != nil {
		t.Fatalf("ListCodeSubscriptionsByTargetQuery failed: %v", err)
	}
	if len(targetList) != 1 {
		t.Fatalf("expected 1 subscription for targetQuery, got %d", len(targetList))
	}

	// 5. Test GetCodeSubscription
	retrieved, err := spannerClient.GetCodeSubscription(ctx, list[0].ID)
	if err != nil {
		t.Fatalf("GetCodeSubscription failed: %v", err)
	}
	if retrieved.ID != list[0].ID {
		t.Errorf("retrieved.ID = %s, want %s", retrieved.ID, list[0].ID)
	}

	// 6. Test AcquireDeliveryLock, RecordDeliverySuccess, ReleaseDeliveryLock
	testDeliveryLocking(ctx, t, list)

	// 7. Test duplicate TargetQuery validation
	testDuplicateTargetQuery(ctx, t, repoID, sub1)

	// 8. Test declarative obsolescence (omitting sub1 transitions it to OBSOLETE) and revival
	testDeclarativeObsolescenceAndRevival(ctx, t, repoID, sub1, sub2, list)

	// 9. Test Webhook Replay Protection and Scan Log insertion
	testWebhookDeduplicationAndScanLog(ctx, t, repoID, now)
}

func testDuplicateTargetQuery(
	ctx context.Context,
	t *testing.T,
	repoID string,
	sub1 CodeSubscription,
) {
	t.Helper()

	dupSub := sub1
	dupSub.ID = uuid.NewString()

	err := spannerClient.SynchronizeRepositoryCodeSubscriptions(
		ctx, VCSProviderGitHub, repoID, []CodeSubscription{sub1, dupSub})
	if !errors.Is(err, ErrDuplicateTargetQuery) {
		t.Fatalf("expected ErrDuplicateTargetQuery, got: %v", err)
	}
}

func testDeliveryLocking(ctx context.Context, t *testing.T, list []CodeSubscription) {
	t.Helper()

	delID1 := uuid.NewString()
	lockAcquired, err := spannerClient.AcquireDeliveryLock(ctx, list[0].ID, delID1, "worker-1", 30*time.Second)
	if err != nil {
		t.Fatalf("AcquireDeliveryLock failed: %v", err)
	}
	if !lockAcquired {
		t.Errorf("expected lock acquisition to succeed")
	}

	secondLock, err := spannerClient.AcquireDeliveryLock(ctx, list[0].ID, delID1, "worker-2", 30*time.Second)
	if err != nil {
		t.Fatalf("AcquireDeliveryLock 2 failed: %v", err)
	}
	if secondLock {
		t.Errorf("expected second lock acquisition to fail")
	}

	err = spannerClient.RecordDeliverySuccess(
		ctx, delID1, "issue-123", "https://github.com/owner/repo/issues/123")
	if err != nil {
		t.Fatalf("RecordDeliverySuccess failed: %v", err)
	}

	// Verify delivery is marked delivered and cannot be re-locked
	lockedAgain, err := spannerClient.AcquireDeliveryLock(ctx, list[0].ID, delID1, "worker-3", 30*time.Second)
	if err != nil {
		t.Fatalf("AcquireDeliveryLock after delivery failed: %v", err)
	}
	if lockedAgain {
		t.Errorf("expected lock acquisition on delivered task to return false")
	}

	delID2 := uuid.NewString()
	acquired2, err := spannerClient.AcquireDeliveryLock(ctx, list[1].ID, delID2, "worker-temp", 30*time.Second)
	if err != nil || !acquired2 {
		t.Fatalf("failed to acquire initial lock for delivery 2: %v", err)
	}

	if err := spannerClient.ReleaseDeliveryLock(ctx, delID2); err != nil {
		t.Fatalf("ReleaseDeliveryLock failed: %v", err)
	}

	reacquired, err := spannerClient.AcquireDeliveryLock(ctx, list[1].ID, delID2, "worker-next", 30*time.Second)
	if err != nil || !reacquired {
		t.Fatalf("failed to reacquire lock after release: %v", err)
	}

	// Automatic TTL expiry: acquire short lease, sleep past TTL, then acquire with another worker
	delID3 := uuid.NewString()
	shortLock, err := spannerClient.AcquireDeliveryLock(ctx, list[0].ID, delID3, "worker-short", 50*time.Millisecond)
	if err != nil || !shortLock {
		t.Fatalf("failed to acquire short TTL lock: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	expiredLock, err := spannerClient.AcquireDeliveryLock(ctx, list[0].ID, delID3, "worker-after-expiry", 30*time.Second)
	if err != nil || !expiredLock {
		t.Fatalf("expected lock reacquisition after TTL expiry, got: %v", err)
	}
}

func testDeclarativeObsolescenceAndRevival(
	ctx context.Context,
	t *testing.T,
	repoID string,
	sub1, sub2 CodeSubscription,
	initialList []CodeSubscription,
) {
	t.Helper()

	var originalSub1 CodeSubscription
	for _, s := range initialList {
		if s.TargetQuery == sub1.TargetQuery {
			originalSub1 = s

			break
		}
	}

	if err := spannerClient.SynchronizeRepositoryCodeSubscriptions(
		ctx, VCSProviderGitHub, repoID, []CodeSubscription{sub2}); err != nil {
		t.Fatalf("SynchronizeRepositoryCodeSubscriptions (obsolete sub1) failed: %v", err)
	}
	activeList, _, err := spannerClient.ListCodeSubscriptionsByRepository(ctx, ListCodeSubscriptionsRequest{
		VCSProvider: VCSProviderGitHub,
		RepoID:      repoID,
		PageSize:    10,
		PageToken:   nil,
	})
	if err != nil {
		t.Fatalf("ListCodeSubscriptionsByRepository after obsolescence failed: %v", err)
	}
	if len(activeList) != 1 || activeList[0].TargetQuery != sub2.TargetQuery {
		t.Fatalf("expected 1 active subscription (sub2), got %+v", activeList)
	}

	if err := spannerClient.SynchronizeRepositoryCodeSubscriptions(
		ctx, VCSProviderGitHub, repoID, []CodeSubscription{sub1, sub2}); err != nil {
		t.Fatalf("SynchronizeRepositoryCodeSubscriptions (revival) failed: %v", err)
	}
	revivedList, _, err := spannerClient.ListCodeSubscriptionsByRepository(ctx, ListCodeSubscriptionsRequest{
		VCSProvider: VCSProviderGitHub,
		RepoID:      repoID,
		PageSize:    10,
		PageToken:   nil,
	})
	if err != nil {
		t.Fatalf("ListCodeSubscriptionsByRepository after revival failed: %v", err)
	}
	if len(revivedList) != 2 {
		t.Fatalf("expected 2 active subscriptions after revival, got %d", len(revivedList))
	}

	// Assert original CreatedAt was preserved upon revival
	for _, revived := range revivedList {
		if revived.TargetQuery == sub1.TargetQuery {
			if !revived.CreatedAt.Equal(originalSub1.CreatedAt) {
				t.Errorf("expected CreatedAt preserved on revival: want %v, got %v", originalSub1.CreatedAt, revived.CreatedAt)
			}
		}
	}
}

func testWebhookDeduplicationAndScanLog(
	ctx context.Context,
	t *testing.T,
	repoID string,
	now time.Time,
) {
	t.Helper()

	delivery := VCSWebhookDelivery{
		VCSProvider:     VCSProviderGitHub,
		DeliveryGUID:    "guid-unique-12345",
		EventType:       "push",
		VCSRepositoryID: repoID,
		ReceivedAt:      now,
	}
	inserted1, err := spannerClient.RecordVCSWebhookDelivery(ctx, delivery)
	if err != nil {
		t.Fatalf("RecordVCSWebhookDelivery failed: %v", err)
	}
	if !inserted1 {
		t.Errorf("expected inserted1=true for first delivery")
	}

	inserted2, err := spannerClient.RecordVCSWebhookDelivery(ctx, delivery)
	if err != nil {
		t.Fatalf("RecordVCSWebhookDelivery second attempt failed: %v", err)
	}
	if inserted2 {
		t.Errorf("expected inserted2=false for replay delivery")
	}

	scanLog := CodeSubscriptionScanLog{
		ID:              uuid.NewString(),
		VCSProvider:     VCSProviderGitHub,
		VCSRepositoryID: repoID,
		CommitSHA:       "abcdef123456",
		Branch:          "main",
		ScanStatus:      ScanStatusSuccess,
		FilesScanned:    12,
		DirectivesFound: 2,
		ErrorMessage:    nil,
		ScannedAt:       now,
	}
	if err := spannerClient.InsertCodeSubscriptionScanLog(ctx, scanLog); err != nil {
		t.Fatalf("InsertCodeSubscriptionScanLog failed: %v", err)
	}
}
