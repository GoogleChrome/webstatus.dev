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

	"cloud.google.com/go/spanner"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

func TestCodeSubscriptionDeliveryConversion(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Microsecond)
	lockExp := now.Add(30 * time.Second)
	lockID := "worker-1"
	issueID := "issue-123"
	issueURL := "https://github.com/owner/repo/issues/123"
	errMsg := "rate limited"

	testCases := []struct {
		name     string
		original CodeSubscriptionDelivery
	}{
		{
			name: "Populated nullable fields roundtrip",
			original: CodeSubscriptionDelivery{
				ID:               "del-uuid-1",
				SubscriptionID:   "sub-uuid-1",
				DeliveryStatus:   DeliveryStatusDelivered,
				DeliveryChannel:  DeliveryChannelGitHubIssue,
				LockExpiresAt:    &lockExp,
				WorkerLockID:     &lockID,
				DeliveredAt:      &now,
				ExternalIssueID:  &issueID,
				ExternalIssueURL: &issueURL,
				ErrorMessage:     &errMsg,
				CreatedAt:        now,
				UpdatedAt:        now,
			},
		},
		{
			name: "Nil nullable fields roundtrip",
			original: CodeSubscriptionDelivery{
				ID:               "del-uuid-nil-fields",
				SubscriptionID:   "sub-uuid-1",
				DeliveryStatus:   DeliveryStatusPending,
				DeliveryChannel:  DeliveryChannelGitHubIssue,
				LockExpiresAt:    nil,
				WorkerLockID:     nil,
				DeliveredAt:      nil,
				ExternalIssueID:  nil,
				ExternalIssueURL: nil,
				ErrorMessage:     nil,
				CreatedAt:        now,
				UpdatedAt:        now,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scsd := fromCodeSubscriptionDelivery(&tc.original)
			converted, err := scsd.toCodeSubscriptionDelivery()
			if err != nil {
				t.Fatalf("unexpected toCodeSubscriptionDelivery error: %v", err)
			}

			if diff := cmp.Diff(&tc.original, converted); diff != "" {
				t.Errorf("CodeSubscriptionDelivery conversion mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCodeSubscriptionDelivery_InvalidEnums(t *testing.T) {
	t.Parallel()

	invalidStatus := spannerCodeSubscriptionDelivery{
		ID:               "del-1",
		SubscriptionID:   "sub-1",
		DeliveryStatus:   "INVALID_STATUS",
		DeliveryChannel:  "github_issue",
		LockExpiresAt:    spanner.NullTime{Time: time.Time{}, Valid: false},
		WorkerLockID:     spanner.NullString{StringVal: "", Valid: false},
		DeliveredAt:      spanner.NullTime{Time: time.Time{}, Valid: false},
		ExternalIssueID:  spanner.NullString{StringVal: "", Valid: false},
		ExternalIssueURL: spanner.NullString{StringVal: "", Valid: false},
		ErrorMessage:     spanner.NullString{StringVal: "", Valid: false},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	_, err := invalidStatus.toCodeSubscriptionDelivery()
	if err == nil {
		t.Fatalf("expected error for invalid delivery status, got nil")
	}
	if !errors.Is(err, ErrUnknownDeliveryStatus) {
		t.Fatalf("expected ErrUnknownDeliveryStatus, got %v", err)
	}

	invalidChannel := spannerCodeSubscriptionDelivery{
		ID:               "del-2",
		SubscriptionID:   "sub-1",
		DeliveryStatus:   "PENDING",
		DeliveryChannel:  "unsupported_channel",
		LockExpiresAt:    spanner.NullTime{Time: time.Time{}, Valid: false},
		WorkerLockID:     spanner.NullString{StringVal: "", Valid: false},
		DeliveredAt:      spanner.NullTime{Time: time.Time{}, Valid: false},
		ExternalIssueID:  spanner.NullString{StringVal: "", Valid: false},
		ExternalIssueURL: spanner.NullString{StringVal: "", Valid: false},
		ErrorMessage:     spanner.NullString{StringVal: "", Valid: false},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	_, err = invalidChannel.toCodeSubscriptionDelivery()
	if err == nil {
		t.Fatalf("expected error for invalid delivery channel, got nil")
	}
	if !errors.Is(err, ErrUnknownDeliveryChannel) {
		t.Fatalf("expected ErrUnknownDeliveryChannel, got %v", err)
	}
}

func testCodeSubscriptionDeliveryNotFound(ctx context.Context, t *testing.T) {
	_, err := spannerClient.GetCodeSubscriptionDelivery(ctx, "non-existent-del-id")
	if !errors.Is(err, ErrCodeSubscriptionDeliveryNotFound) {
		t.Fatalf("expected ErrCodeSubscriptionDeliveryNotFound, got %v", err)
	}
}

func testCodeSubscriptionDeliveryInsertAndGet(ctx context.Context, t *testing.T, subID string) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	delID := uuid.NewString()
	del := CodeSubscriptionDelivery{
		ID:               delID,
		SubscriptionID:   subID,
		DeliveryStatus:   DeliveryStatusPending,
		DeliveryChannel:  DeliveryChannelGitHubIssue,
		LockExpiresAt:    nil,
		WorkerLockID:     nil,
		DeliveredAt:      nil,
		ExternalIssueID:  nil,
		ExternalIssueURL: nil,
		ErrorMessage:     nil,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	spannerDel := fromCodeSubscriptionDelivery(&del)
	delMut, err := spanner.InsertStruct(codeSubscriptionDeliveryTable, spannerDel)
	if err != nil {
		t.Fatalf("InsertStruct delivery failed: %v", err)
	}
	if _, err := spannerClient.Apply(ctx, []*spanner.Mutation{delMut}); err != nil {
		t.Fatalf("Apply delivery mutation failed: %v", err)
	}

	retrieved, err := spannerClient.GetCodeSubscriptionDelivery(ctx, delID)
	if err != nil {
		t.Fatalf("GetCodeSubscriptionDelivery failed: %v", err)
	}
	if retrieved.ID != delID || retrieved.SubscriptionID != subID {
		t.Fatalf("retrieved delivery mismatch: %+v", retrieved)
	}
	if retrieved.DeliveryStatus != DeliveryStatusPending {
		t.Errorf("expected status %s, got %s", DeliveryStatusPending, retrieved.DeliveryStatus)
	}
}

func testCodeSubscriptionDeliveryNullableFields(ctx context.Context, t *testing.T, subID string) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	delID := uuid.NewString()
	lockExp := now.Add(30 * time.Second)
	workerID := "worker-lock-42"
	issueID := "gh-issue-99"
	issueURL := "https://github.com/GoogleChrome/webstatus.dev/issues/99"
	errMsg := "rate limited by github"

	del := CodeSubscriptionDelivery{
		ID:               delID,
		SubscriptionID:   subID,
		DeliveryStatus:   DeliveryStatusFailed,
		DeliveryChannel:  DeliveryChannelGitHubIssue,
		LockExpiresAt:    &lockExp,
		WorkerLockID:     &workerID,
		DeliveredAt:      &now,
		ExternalIssueID:  &issueID,
		ExternalIssueURL: &issueURL,
		ErrorMessage:     &errMsg,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	spannerDel := fromCodeSubscriptionDelivery(&del)
	delMut, err := spanner.InsertStruct(codeSubscriptionDeliveryTable, spannerDel)
	if err != nil {
		t.Fatalf("InsertStruct delivery failed: %v", err)
	}
	if _, err := spannerClient.Apply(ctx, []*spanner.Mutation{delMut}); err != nil {
		t.Fatalf("Apply delivery mutation failed: %v", err)
	}

	retrieved, err := spannerClient.GetCodeSubscriptionDelivery(ctx, delID)
	if err != nil {
		t.Fatalf("GetCodeSubscriptionDelivery failed: %v", err)
	}
	if retrieved.WorkerLockID == nil || *retrieved.WorkerLockID != workerID {
		t.Errorf("expected WorkerLockID %s, got %v", workerID, retrieved.WorkerLockID)
	}
	if retrieved.ExternalIssueID == nil || *retrieved.ExternalIssueID != issueID {
		t.Errorf("expected ExternalIssueID %s, got %v", issueID, retrieved.ExternalIssueID)
	}
	if retrieved.ErrorMessage == nil || *retrieved.ErrorMessage != errMsg {
		t.Errorf("expected ErrorMessage %s, got %v", errMsg, retrieved.ErrorMessage)
	}
}

func TestClient_GetCodeSubscriptionDelivery(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	now := time.Now().UTC().Truncate(time.Microsecond)
	subID := uuid.NewString()

	// Create parent installation
	writeLevel := GitHubPermissionLevelWrite
	inst := VCSInstallation{
		ID:                  "",
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   "12345678",
		AccountLogin:        "GoogleChrome",
		AccountType:         "Organization",
		RepositorySelection: "selected",
		Permissions: VCSPermissions{
			GitHub: &GitHubPermissions{
				Issues:       &writeLevel,
				Contents:     nil,
				Metadata:     nil,
				PullRequests: nil,
				Workflows:    nil,
				Actions:      nil,
			},
		},
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}
	instIDPtr, err := spannerClient.UpsertVCSInstallation(ctx, inst)
	if err != nil {
		t.Fatalf("UpsertVCSInstallation failed: %v", err)
	}
	instID := *instIDPtr

	// Insert CodeSubscription via mutation
	sub := CodeSubscription{
		ID:                 subID,
		VCSProvider:        VCSProviderGitHub,
		VCSInstallationID:  instID,
		VCSRepositoryID:    "repo-123",
		RepositoryFullName: "GoogleChrome/webstatus.dev",
		TargetQuery:        "id:view-transitions",
		Triggers:           []string{"feature_baseline_to_widely"},
		Status:             SubscriptionActive,
		Occurrences: []SubscriptionOccurrence{
			{
				FilePath:       "src/app.ts",
				LineNumber:     10,
				CommentSnippet: "// TODO(baseline/view-transitions)",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	spannerSub := fromCodeSubscription(&sub)
	subMut, err := spanner.InsertStruct(codeSubscriptionTable, spannerSub)
	if err != nil {
		t.Fatalf("InsertStruct failed: %v", err)
	}
	if _, err := spannerClient.Apply(ctx, []*spanner.Mutation{subMut}); err != nil {
		t.Fatalf("Apply subscription mutation failed: %v", err)
	}

	t.Run("NotFound returns ErrCodeSubscriptionDeliveryNotFound", func(t *testing.T) {
		testCodeSubscriptionDeliveryNotFound(ctx, t)
	})
	t.Run("Insert and retrieve delivery record", func(t *testing.T) {
		testCodeSubscriptionDeliveryInsertAndGet(ctx, t, subID)
	})
	t.Run("Handles populated nullable fields", func(t *testing.T) {
		testCodeSubscriptionDeliveryNullableFields(ctx, t, subID)
	})
}
