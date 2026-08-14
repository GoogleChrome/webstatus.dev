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

func TestCodeSubscriptionScanLogConversion(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Microsecond)
	errMsg := "rate limited by GitHub API"

	testCases := []struct {
		name     string
		original CodeSubscriptionScanLog
	}{
		{
			name: "Populated error message roundtrip",
			original: CodeSubscriptionScanLog{
				ID:              "scan-uuid-1",
				VCSProvider:     VCSProviderGitHub,
				VCSRepositoryID: "repo-999",
				CommitSHA:       "abcdef123456",
				Branch:          "main",
				ScanStatus:      ScanStatusFailed,
				FilesScanned:    42,
				DirectivesFound: 3,
				ErrorMessage:    &errMsg,
				ScannedAt:       now,
			},
		},
		{
			name: "Nil error message roundtrip",
			original: CodeSubscriptionScanLog{
				ID:              "scan-uuid-nil-err",
				VCSProvider:     VCSProviderGitHub,
				VCSRepositoryID: "repo-999",
				CommitSHA:       "abcdef123456",
				Branch:          "main",
				ScanStatus:      ScanStatusSuccess,
				FilesScanned:    100,
				DirectivesFound: 5,
				ErrorMessage:    nil,
				ScannedAt:       now,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scsl := fromCodeSubscriptionScanLog(&tc.original)
			converted, err := scsl.toCodeSubscriptionScanLog()
			if err != nil {
				t.Fatalf("unexpected toCodeSubscriptionScanLog error: %v", err)
			}

			if diff := cmp.Diff(&tc.original, converted); diff != "" {
				t.Errorf("CodeSubscriptionScanLog conversion mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCodeSubscriptionScanLog_InvalidEnums(t *testing.T) {
	t.Parallel()

	invalidProvider := spannerCodeSubscriptionScanLog{
		ID:              "scan-1",
		VCSProvider:     "invalid_provider",
		VCSRepositoryID: "repo-1",
		CommitSHA:       "sha-1",
		Branch:          "main",
		ScanStatus:      "SUCCESS",
		FilesScanned:    10,
		DirectivesFound: 1,
		ErrorMessage:    spanner.NullString{StringVal: "", Valid: false},
		ScannedAt:       time.Now(),
	}
	_, err := invalidProvider.toCodeSubscriptionScanLog()
	if err == nil {
		t.Fatalf("expected error for invalid vcs provider, got nil")
	}
	if !errors.Is(err, ErrUnknownVCSProvider) {
		t.Fatalf("expected ErrUnknownVCSProvider, got %v", err)
	}

	invalidStatus := spannerCodeSubscriptionScanLog{
		ID:              "scan-2",
		VCSProvider:     "github",
		VCSRepositoryID: "repo-1",
		CommitSHA:       "sha-1",
		Branch:          "main",
		ScanStatus:      "INVALID_SCAN_STATUS",
		FilesScanned:    10,
		DirectivesFound: 1,
		ErrorMessage:    spanner.NullString{StringVal: "", Valid: false},
		ScannedAt:       time.Now(),
	}
	_, err = invalidStatus.toCodeSubscriptionScanLog()
	if err == nil {
		t.Fatalf("expected error for invalid scan status, got nil")
	}
	if !errors.Is(err, ErrUnknownScanStatus) {
		t.Fatalf("expected ErrUnknownScanStatus, got %v", err)
	}
}

func testScanLogAndWebhookDeliveryNotFound(ctx context.Context, t *testing.T) {
	_, err := spannerClient.GetCodeSubscriptionScanLog(ctx, "non-existent-scan-id")
	if !errors.Is(err, ErrScanLogNotFound) {
		t.Fatalf("expected ErrScanLogNotFound, got: %v", err)
	}

	_, err = spannerClient.GetVCSWebhookDelivery(ctx, VCSProviderGitHub, "non-existent-guid")
	if !errors.Is(err, ErrVCSWebhookDeliveryNotFound) {
		t.Fatalf("expected ErrVCSWebhookDeliveryNotFound, got: %v", err)
	}
}

func testScanLogInsertAndGet(ctx context.Context, t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	scanID := uuid.NewString()
	errMsg := "rate limit backoff"

	log := CodeSubscriptionScanLog{
		ID:              scanID,
		VCSProvider:     VCSProviderGitHub,
		VCSRepositoryID: "repo-123",
		CommitSHA:       "sha-abc",
		Branch:          "main",
		ScanStatus:      ScanStatusFailed,
		FilesScanned:    15,
		DirectivesFound: 2,
		ErrorMessage:    &errMsg,
		ScannedAt:       now,
	}

	spannerLog := fromCodeSubscriptionScanLog(&log)
	logMut, err := spanner.InsertStruct(codeSubscriptionScanLogTable, spannerLog)
	if err != nil {
		t.Fatalf("InsertStruct scan log failed: %v", err)
	}
	if _, err := spannerClient.Apply(ctx, []*spanner.Mutation{logMut}); err != nil {
		t.Fatalf("Apply scan log mutation failed: %v", err)
	}

	retrievedLog, err := spannerClient.GetCodeSubscriptionScanLog(ctx, scanID)
	if err != nil {
		t.Fatalf("GetCodeSubscriptionScanLog failed: %v", err)
	}
	if retrievedLog.ID != scanID || *retrievedLog.ErrorMessage != errMsg {
		t.Fatalf("retrieved scan log mismatch: %+v", retrievedLog)
	}
	if retrievedLog.ScanStatus != ScanStatusFailed {
		t.Errorf("expected ScanStatus %s, got %s", ScanStatusFailed, retrievedLog.ScanStatus)
	}
}

func testWebhookDeliveryInsertAndGet(ctx context.Context, t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	guid := uuid.NewString()

	delivery := VCSWebhookDelivery{
		VCSProvider:     VCSProviderGitHub,
		DeliveryGUID:    guid,
		EventType:       "push",
		VCSRepositoryID: "repo-123",
		ReceivedAt:      now,
	}

	spannerDelivery := fromVCSWebhookDelivery(&delivery)
	deliveryMut, err := spanner.InsertStruct(vcsWebhookDeliveryTable, spannerDelivery)
	if err != nil {
		t.Fatalf("InsertStruct webhook delivery failed: %v", err)
	}
	if _, err := spannerClient.Apply(ctx, []*spanner.Mutation{deliveryMut}); err != nil {
		t.Fatalf("Apply webhook delivery mutation failed: %v", err)
	}

	retrievedDelivery, err := spannerClient.GetVCSWebhookDelivery(ctx, VCSProviderGitHub, guid)
	if err != nil {
		t.Fatalf("GetVCSWebhookDelivery failed: %v", err)
	}
	if retrievedDelivery.DeliveryGUID != guid || retrievedDelivery.EventType != "push" {
		t.Fatalf("retrieved webhook delivery mismatch: %+v", retrievedDelivery)
	}
}

func TestClient_GetScanLogAndWebhookDelivery(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	t.Run("NotFound returns expected sentinel errors", func(t *testing.T) {
		testScanLogAndWebhookDeliveryNotFound(ctx, t)
	})
	t.Run("Insert and retrieve scan log", func(t *testing.T) {
		testScanLogInsertAndGet(ctx, t)
	})
	t.Run("Insert and retrieve webhook delivery idempotency record", func(t *testing.T) {
		testWebhookDeliveryInsertAndGet(ctx, t)
	})
}
