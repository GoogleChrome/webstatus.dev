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
	"strings"
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
	sub1 := CodeSubscriptionInput{
		VCSProvider:        VCSProviderGitHub,
		VCSInstallationID:  installationID,
		VCSRepositoryID:    repoID,
		RepositoryFullName: repoFullName,
		TargetQuery:        "id:view-transitions",
		Triggers:           []SubscriptionTrigger{SubscriptionTriggerFeatureBaselinePromoteToWidely},
		Occurrences: []SubscriptionOccurrence{
			{FilePath: "src/app.ts", LineNumber: 10, CommentSnippet: "// TODO(baseline/view-transitions): transition"},
		},
	}

	sub2 := CodeSubscriptionInput{
		VCSProvider:        VCSProviderGitHub,
		VCSInstallationID:  installationID,
		VCSRepositoryID:    repoID,
		RepositoryFullName: repoFullName,
		TargetQuery:        "id:subgrid",
		Triggers:           []SubscriptionTrigger{SubscriptionTriggerFeatureBaselinePromoteToWidely},
		Occurrences: []SubscriptionOccurrence{
			{FilePath: "src/grid.css", LineNumber: 5, CommentSnippet: "// TODO(baseline/subgrid): grid"},
		},
	}

	if err := spannerClient.SynchronizeRepositoryCodeSubscriptions(
		ctx, VCSProviderGitHub, repoID, []CodeSubscriptionInput{sub1, sub2}); err != nil {
		t.Fatalf("SynchronizeRepositoryCodeSubscriptions failed: %v", err)
	}

	// 3. Test ListCodeSubscriptionsByRepository with pagination
	listReq := ListCodeSubscriptionsRequest{
		VCSProvider:        VCSProviderGitHub,
		RepositoryFullName: repoFullName,
		PageSize:           10,
		PageToken:          nil,
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
		VCSProvider:        VCSProviderGitHub,
		RepositoryFullName: repoFullName,
		PageSize:           1,
		PageToken:          nil,
	}
	pagedList1, token1, err := spannerClient.ListCodeSubscriptionsByRepository(ctx, pagedReq1)
	if err != nil {
		t.Fatalf("ListCodeSubscriptionsByRepository page 1 failed: %v", err)
	}
	if len(pagedList1) != 1 || token1 == nil {
		t.Fatalf("expected 1 item and valid next token on page 1, got len=%d token=%v", len(pagedList1), token1)
	}

	pagedReq2 := ListCodeSubscriptionsRequest{
		VCSProvider:        VCSProviderGitHub,
		RepositoryFullName: repoFullName,
		PageSize:           1,
		PageToken:          token1,
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

	// 3c. Test case-insensitivity of repository full name lookup (e.g. lowercase)
	lowerListReq := ListCodeSubscriptionsRequest{
		VCSProvider:        VCSProviderGitHub,
		RepositoryFullName: strings.ToLower(repoFullName),
		PageSize:           10,
		PageToken:          nil,
	}
	lowerList, _, err := spannerClient.ListCodeSubscriptionsByRepository(ctx, lowerListReq)
	if err != nil {
		t.Fatalf("ListCodeSubscriptionsByRepository with lowercased name failed: %v", err)
	}
	if len(lowerList) != 2 {
		t.Fatalf("expected 2 active subscriptions when queried with lowercased name, got %d", len(lowerList))
	}

	// 4. Test ListCodeSubscriptionsByTargetQuery
	targetList, err := spannerClient.ListCodeSubscriptionsByTargetQuery(
		ctx, "id:view-transitions", SubscriptionTriggerFeatureBaselinePromoteToWidely)
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

	// 10. Test ListVCSRepositoriesByProvider and ListVCSInstallationsByAccount
	testListVCSRepositoriesAndInstallations(ctx, t, repoID)
}

func testDuplicateTargetQuery(
	ctx context.Context,
	t *testing.T,
	repoID string,
	sub1 CodeSubscriptionInput,
) {
	t.Helper()

	dupSub := sub1

	err := spannerClient.SynchronizeRepositoryCodeSubscriptions(
		ctx, VCSProviderGitHub, repoID, []CodeSubscriptionInput{sub1, dupSub})
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
	if !errors.Is(err, ErrDeliveryAlreadyLocked) {
		t.Fatalf("expected ErrDeliveryAlreadyLocked from AcquireDeliveryLock 2, got %v", err)
	}
	if secondLock {
		t.Errorf("expected second lock acquisition to fail")
	}

	// Test lock fencing: worker-wrong cannot record delivery success
	wrongWorkerErr := spannerClient.RecordDeliverySuccess(
		ctx, delID1, "worker-wrong", "issue-123", "https://github.com/owner/repo/issues/123")
	if !errors.Is(wrongWorkerErr, ErrDeliveryLockMismatch) {
		t.Fatalf("expected ErrDeliveryLockMismatch from worker-wrong, got %v", wrongWorkerErr)
	}

	err = spannerClient.RecordDeliverySuccess(
		ctx, delID1, "worker-1", "issue-123", "https://github.com/owner/repo/issues/123")
	if err != nil {
		t.Fatalf("RecordDeliverySuccess failed: %v", err)
	}

	// Verify delivery is marked delivered and cannot be re-locked (returns false, nil)
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

	// Test lock fencing on release: worker-other cannot clear worker-temp's lock
	errReleaseOther := spannerClient.ReleaseDeliveryLock(ctx, delID2, "worker-other")
	if !errors.Is(errReleaseOther, ErrDeliveryLockMismatch) {
		t.Fatalf("expected ErrDeliveryLockMismatch from worker-other, got %v", errReleaseOther)
	}
	// Lock should still be held by worker-temp, so worker-next cannot acquire yet
	blockedAcquire, err := spannerClient.AcquireDeliveryLock(ctx, list[1].ID, delID2, "worker-next", 30*time.Second)
	if !errors.Is(err, ErrDeliveryAlreadyLocked) || blockedAcquire {
		t.Fatalf(
			"expected lock to still be held by worker-temp (ErrDeliveryAlreadyLocked), got acquired=%v, err=%v",
			blockedAcquire,
			err,
		)
	}

	// Now legitimate owner releases lock
	if err := spannerClient.ReleaseDeliveryLock(ctx, delID2, "worker-temp"); err != nil {
		t.Fatalf("ReleaseDeliveryLock failed: %v", err)
	}

	reacquired, err := spannerClient.AcquireDeliveryLock(ctx, list[1].ID, delID2, "worker-next", 30*time.Second)
	if err != nil || !reacquired {
		t.Fatalf("failed to reacquire lock after release: %v", err)
	}

	// Automatic TTL expiry: acquire short lease, sleep past TTL, then test expired release and reacquisition
	delID3 := uuid.NewString()
	shortLock, err := spannerClient.AcquireDeliveryLock(ctx, list[0].ID, delID3, "worker-short", 50*time.Millisecond)
	if err != nil || !shortLock {
		t.Fatalf("failed to acquire short TTL lock: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Trying to release an already-expired lock should fail with ErrDeliveryLockExpired
	errExpiredRelease := spannerClient.ReleaseDeliveryLock(ctx, delID3, "worker-short")
	if !errors.Is(errExpiredRelease, ErrDeliveryLockExpired) {
		t.Fatalf("expected ErrDeliveryLockExpired for expired worker-short, got %v", errExpiredRelease)
	}

	expiredLock, err := spannerClient.AcquireDeliveryLock(ctx, list[0].ID, delID3, "worker-after-expiry", 30*time.Second)
	if err != nil || !expiredLock {
		t.Fatalf("expected lock reacquisition after TTL expiry, got: %v", err)
	}
}

func testDeclarativeObsolescenceAndRevival(
	ctx context.Context,
	t *testing.T,
	repoID string,
	sub1, sub2 CodeSubscriptionInput,
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
		ctx, VCSProviderGitHub, repoID, []CodeSubscriptionInput{sub2}); err != nil {
		t.Fatalf("SynchronizeRepositoryCodeSubscriptions (obsolete sub1) failed: %v", err)
	}
	activeList, _, err := spannerClient.ListCodeSubscriptionsByRepository(ctx, ListCodeSubscriptionsRequest{
		VCSProvider:        VCSProviderGitHub,
		RepositoryFullName: sub2.RepositoryFullName,
		PageSize:           10,
		PageToken:          nil,
	})
	if err != nil {
		t.Fatalf("ListCodeSubscriptionsByRepository after obsolescence failed: %v", err)
	}
	if len(activeList) != 1 || activeList[0].TargetQuery != sub2.TargetQuery {
		t.Fatalf("expected 1 active subscription (sub2), got %+v", activeList)
	}

	if err := spannerClient.SynchronizeRepositoryCodeSubscriptions(
		ctx, VCSProviderGitHub, repoID, []CodeSubscriptionInput{sub1, sub2}); err != nil {
		t.Fatalf("SynchronizeRepositoryCodeSubscriptions (revival) failed: %v", err)
	}
	revivedList, _, err := spannerClient.ListCodeSubscriptionsByRepository(ctx, ListCodeSubscriptionsRequest{
		VCSProvider:        VCSProviderGitHub,
		RepositoryFullName: sub1.RepositoryFullName,
		PageSize:           10,
		PageToken:          nil,
	})
	if err != nil {
		t.Fatalf("ListCodeSubscriptionsByRepository after revival failed: %v", err)
	}
	if len(revivedList) != 2 {
		t.Fatalf("expected 2 active subscriptions after revival, got %d", len(revivedList))
	}

	// Assert original ID and CreatedAt were preserved upon revival
	for _, revived := range revivedList {
		if revived.TargetQuery == sub1.TargetQuery {
			if revived.ID != originalSub1.ID {
				t.Errorf("expected durable ID preserved on revival: want %s, got %s", originalSub1.ID, revived.ID)
			}
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

func testListVCSRepositoriesAndInstallations(
	ctx context.Context,
	t *testing.T,
	repoID string,
) {
	t.Helper()

	repos, err := spannerClient.ListVCSRepositoriesByProvider(ctx, VCSProviderGitHub)
	if err != nil {
		t.Fatalf("ListVCSRepositoriesByProvider failed: %v", err)
	}
	if len(repos) == 0 {
		t.Errorf("expected at least 1 repository, got 0")
	}
	found := false
	for _, r := range repos {
		if r.VCSRepositoryID == repoID {
			found = true
			if r.RepositoryFullName != "GoogleChrome/webstatus.dev" {
				t.Errorf("unexpected repo full name: %s", r.RepositoryFullName)
			}
		}
	}
	if !found {
		t.Errorf("expected repoID %s in ListVCSRepositoriesByProvider results", repoID)
	}

	inst := VCSInstallation{
		ID:                  uuid.NewString(),
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   "inst-acct-test-1",
		AccountLogin:        "TestAccountLogin",
		AccountType:         "Organization",
		RepositorySelection: "all",
		Permissions:         VCSPermissions{GitHub: nil},
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}
	if _, err := spannerClient.UpsertVCSInstallation(ctx, inst); err != nil {
		t.Fatalf("UpsertVCSInstallation failed: %v", err)
	}

	allInstallations, err := spannerClient.ListVCSInstallations(ctx)
	if err != nil {
		t.Fatalf("ListVCSInstallations failed: %v", err)
	}
	if len(allInstallations) < 1 {
		t.Fatalf("expected at least 1 installation, got %d", len(allInstallations))
	}

	installations, err := spannerClient.ListVCSInstallationsByAccount(ctx, VCSProviderGitHub, "TestAccountLogin")
	if err != nil {
		t.Fatalf("ListVCSInstallationsByAccount failed: %v", err)
	}
	if len(installations) != 1 {
		t.Fatalf("expected 1 installation for account, got %d", len(installations))
	}
	if installations[0].AccountLogin != "TestAccountLogin" {
		t.Errorf("unexpected account login: %s", installations[0].AccountLogin)
	}
}
