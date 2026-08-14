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

	// 3. Test ListCodeSubscriptionsByRepository
	list, err := spannerClient.ListCodeSubscriptionsByRepository(ctx, VCSProviderGitHub, repoID)
	if err != nil {
		t.Fatalf("ListCodeSubscriptionsByRepository failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 active subscriptions, got %d", len(list))
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
	delID1 := uuid.NewString()
	lockAcquired, err := spannerClient.AcquireDeliveryLock(ctx, list[0].ID, delID1, "worker-1", 30*time.Second)
	if err != nil {
		t.Fatalf("AcquireDeliveryLock failed: %v", err)
	}
	if !lockAcquired {
		t.Errorf("expected lock acquisition to succeed")
	}

	// Duplicate lock acquisition while active should fail
	secondLock, err := spannerClient.AcquireDeliveryLock(ctx, list[0].ID, delID1, "worker-2", 30*time.Second)
	if err != nil {
		t.Fatalf("AcquireDeliveryLock 2 failed: %v", err)
	}
	if secondLock {
		t.Errorf("expected second lock acquisition to fail")
	}

	// Record success
	err = spannerClient.RecordDeliverySuccess(
		ctx, delID1, "issue-123", "https://github.com/owner/repo/issues/123")
	if err != nil {
		t.Fatalf("RecordDeliverySuccess failed: %v", err)
	}

	// Test ReleaseDeliveryLock on list[1] and reacquire
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

	// 7. Test DeleteCodeSubscription (Soft Delete) and subsequent revival on resync
	if err := spannerClient.DeleteCodeSubscription(ctx, list[0].ID); err != nil {
		t.Fatalf("DeleteCodeSubscription failed: %v", err)
	}

	// Resync with the same directives must revive the soft-deleted subscription without unique index error
	if err := spannerClient.SynchronizeRepositoryCodeSubscriptions(
		ctx, VCSProviderGitHub, repoID, []CodeSubscription{sub1, sub2}); err != nil {
		t.Fatalf("SynchronizeRepositoryCodeSubscriptions (revival) failed: %v", err)
	}
	revivedList, err := spannerClient.ListCodeSubscriptionsByRepository(ctx, VCSProviderGitHub, repoID)
	if err != nil {
		t.Fatalf("ListCodeSubscriptionsByRepository after revival failed: %v", err)
	}
	if len(revivedList) != 2 {
		t.Fatalf("expected 2 active subscriptions after revival, got %d", len(revivedList))
	}

	// 8. Test Webhook Replay Protection
	delivery := VCSWebhookDelivery{
		VCSProvider:     VCSProviderGitHub,
		DeliveryGUID:    "guid-unique-12345",
		EventType:       "push",
		VCSRepositoryID: repoID,
		ReceivedAt:      now,
	}
	isNew, err := spannerClient.RecordVCSWebhookDelivery(ctx, delivery)
	if err != nil {
		t.Fatalf("RecordVCSWebhookDelivery failed: %v", err)
	}
	if !isNew {
		t.Errorf("expected isNew=true for first delivery")
	}

	isDuplicate, err := spannerClient.RecordVCSWebhookDelivery(ctx, delivery)
	if err != nil {
		t.Fatalf("RecordVCSWebhookDelivery second attempt failed: %v", err)
	}
	if isDuplicate {
		t.Errorf("expected isDuplicate=false for replay delivery")
	}

	// 9. Test InsertCodeSubscriptionScanLog
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
