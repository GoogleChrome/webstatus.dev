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
	"github.com/google/uuid"
)

func TestCodeSubscriptionConversion(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Microsecond)

	testCases := []struct {
		name        string
		input       CodeSubscription
		checkOutput func(t *testing.T, converted *CodeSubscription)
	}{
		{
			name: "Populated occurrences roundtrip",
			input: CodeSubscription{
				ID:                 "sub-uuid-1",
				VCSProvider:        VCSProviderGitHub,
				VCSInstallationID:  "123456",
				VCSRepositoryID:    "987654",
				RepositoryFullName: "owner/repo",
				TargetQuery:        "id:css-subgrid AND baseline_status:widely",
				Triggers:           []string{"feature_baseline_to_widely"},
				Status:             SubscriptionActive,
				Occurrences: []SubscriptionOccurrence{
					{
						FilePath:       "src/styles.css",
						LineNumber:     42,
						CommentSnippet: "// TODO(baseline/subgrid): remove flex fallback",
					},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			checkOutput: func(t *testing.T, converted *CodeSubscription) {
				if len(converted.Occurrences) != 1 {
					t.Fatalf("expected 1 occurrence, got %d", len(converted.Occurrences))
				}
				if converted.Occurrences[0].LineNumber != 42 {
					t.Errorf("expected line 42, got %d", converted.Occurrences[0].LineNumber)
				}
			},
		},
		{
			name: "Nil occurrences deserializes as non-nil empty slice",
			input: CodeSubscription{
				ID:                 "sub-uuid-nil-occ",
				VCSProvider:        VCSProviderGitHub,
				VCSInstallationID:  "123456",
				VCSRepositoryID:    "987654",
				RepositoryFullName: "owner/repo",
				TargetQuery:        "id:css-subgrid",
				Triggers:           []string{"feature_baseline_to_widely"},
				Status:             SubscriptionActive,
				Occurrences:        nil,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			checkOutput: func(t *testing.T, converted *CodeSubscription) {
				if converted.Occurrences == nil {
					t.Errorf("expected non-nil empty slice for occurrences, got nil")
				}
				if len(converted.Occurrences) != 0 {
					t.Errorf("expected 0 occurrences, got %d", len(converted.Occurrences))
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scs := fromCodeSubscription(&tc.input)
			converted, err := scs.toCodeSubscription()
			if err != nil {
				t.Fatalf("unexpected toCodeSubscription error: %v", err)
			}
			tc.checkOutput(t, converted)
		})
	}
}

func TestCodeSubscription_InvalidStatus(t *testing.T) {
	t.Parallel()

	invalid := spannerCodeSubscription{
		ID:                 "sub-invalid-status",
		VCSProvider:        "github",
		VCSInstallationID:  "123",
		VCSRepositoryID:    "456",
		RepositoryFullName: "owner/repo",
		TargetQuery:        "query",
		Triggers:           []string{"trigger"},
		Status:             "INVALID_STATUS",
		Occurrences:        spanner.NullJSON{Value: nil, Valid: false},
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	_, err := invalid.toCodeSubscription()
	if err == nil {
		t.Fatalf("expected error for invalid status, got nil")
	}
	if !errors.Is(err, ErrUnknownSubscriptionStatus) {
		t.Fatalf("expected ErrUnknownSubscriptionStatus, got %v", err)
	}
}

func testCodeSubscriptionNotFound(ctx context.Context, t *testing.T) {
	_, err := spannerClient.GetCodeSubscription(ctx, "non-existent-sub-id")
	if !errors.Is(err, ErrCodeSubscriptionNotFound) {
		t.Fatalf("expected ErrCodeSubscriptionNotFound, got: %v", err)
	}
}

func testCodeSubscriptionInsertWithOccurrences(ctx context.Context, t *testing.T, instID string) {
	subID := uuid.NewString()
	now := time.Now().UTC().Truncate(time.Microsecond)
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

	retrievedSub, err := spannerClient.GetCodeSubscription(ctx, subID)
	if err != nil {
		t.Fatalf("GetCodeSubscription failed: %v", err)
	}
	if retrievedSub.ID != subID || len(retrievedSub.Occurrences) != 1 {
		t.Fatalf("retrieved subscription mismatch: %+v", retrievedSub)
	}
	if retrievedSub.VCSProvider != VCSProviderGitHub {
		t.Errorf("expected VCSProvider %s, got %s", VCSProviderGitHub, retrievedSub.VCSProvider)
	}
	if retrievedSub.Status != SubscriptionActive {
		t.Errorf("expected Status %s, got %s", SubscriptionActive, retrievedSub.Status)
	}
	if retrievedSub.Occurrences[0].FilePath != "src/app.ts" || retrievedSub.Occurrences[0].LineNumber != 10 {
		t.Errorf("unexpected occurrence data: %+v", retrievedSub.Occurrences[0])
	}
}

func testCodeSubscriptionEmptyOccurrences(ctx context.Context, t *testing.T, instID string) {
	emptySubID := uuid.NewString()
	now := time.Now().UTC().Truncate(time.Microsecond)
	emptySub := CodeSubscription{
		ID:                 emptySubID,
		VCSProvider:        VCSProviderGitHub,
		VCSInstallationID:  instID,
		VCSRepositoryID:    "repo-empty-occ",
		RepositoryFullName: "GoogleChrome/webstatus.dev",
		TargetQuery:        "id:subgrid",
		Triggers:           []string{"feature_baseline_to_widely"},
		Status:             SubscriptionActive,
		Occurrences:        []SubscriptionOccurrence{},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	spannerEmptySub := fromCodeSubscription(&emptySub)
	emptyMut, err := spanner.InsertStruct(codeSubscriptionTable, spannerEmptySub)
	if err != nil {
		t.Fatalf("InsertStruct failed: %v", err)
	}
	if _, err := spannerClient.Apply(ctx, []*spanner.Mutation{emptyMut}); err != nil {
		t.Fatalf("Apply empty subscription mutation failed: %v", err)
	}

	retrieved, err := spannerClient.GetCodeSubscription(ctx, emptySubID)
	if err != nil {
		t.Fatalf("GetCodeSubscription failed: %v", err)
	}
	if retrieved.Occurrences == nil {
		t.Errorf("expected non-nil empty occurrences slice, got nil")
	}
	if len(retrieved.Occurrences) != 0 {
		t.Errorf("expected 0 occurrences, got %d", len(retrieved.Occurrences))
	}
}

func TestClient_GetCodeSubscription(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	// Create parent installation for foreign key / relationship
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

	t.Run("NotFound returns ErrCodeSubscriptionNotFound", func(t *testing.T) {
		testCodeSubscriptionNotFound(ctx, t)
	})
	t.Run("Insert and retrieve with occurrences", func(t *testing.T) {
		testCodeSubscriptionInsertWithOccurrences(ctx, t, instID)
	})
	t.Run("Roundtrip with empty occurrences", func(t *testing.T) {
		testCodeSubscriptionEmptyOccurrences(ctx, t, instID)
	})
}
