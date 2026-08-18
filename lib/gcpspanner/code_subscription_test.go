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

	"cloud.google.com/go/spanner"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

func TestCodeSubscriptionConversion(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Microsecond)
	original := CodeSubscription{
		ID:                 "sub-uuid-1",
		VCSProvider:        string(VCSProviderGitHub),
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
	}

	scs := fromCodeSubscription(&original)

	converted, err := scs.toCodeSubscription()
	if err != nil {
		t.Fatalf("unexpected toCodeSubscription error: %v", err)
	}

	if diff := cmp.Diff(&original, converted); diff != "" {
		t.Errorf("CodeSubscription conversion mismatch (-want +got):\n%s", diff)
	}
}

func TestCodeSubscriptionMapper(t *testing.T) {
	t.Parallel()

	mapper := codeSubscriptionMapper{}
	if mapper.Table() != "CodeSubscriptions" {
		t.Errorf("expected table CodeSubscriptions, got %s", mapper.Table())
	}

	stmt := mapper.SelectOne("sub-123")
	if stmt.Params["id"] != "sub-123" {
		t.Errorf("expected param id=sub-123, got %v", stmt.Params["id"])
	}
}

func TestClient_CodeSubscriptionMappers(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	now := time.Now().UTC().Truncate(time.Microsecond)
	instID := uuid.NewString()
	subID := uuid.NewString()

	// 1. Create parent installation
	writeLevel := GitHubPermissionLevelWrite
	inst := VCSInstallation{
		ID:                  instID,
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
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := spannerClient.UpsertVCSInstallation(ctx, inst); err != nil {
		t.Fatalf("UpsertVCSInstallation failed: %v", err)
	}

	// 2. Insert CodeSubscription via mutation
	sub := CodeSubscription{
		ID:                 subID,
		VCSProvider:        string(VCSProviderGitHub),
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

	// 3. Query back using codeSubscriptionMapper
	subMapper := codeSubscriptionMapper{}
	subStmt := subMapper.SelectOne(subID)
	iter := spannerClient.Single().Query(ctx, subStmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		t.Fatalf("Query subscription failed: %v", err)
	}
	var retrievedSpannerSub spannerCodeSubscription
	if err := row.ToStruct(&retrievedSpannerSub); err != nil {
		t.Fatalf("ToStruct failed: %v", err)
	}
	retrievedSub, err := retrievedSpannerSub.toCodeSubscription()
	if err != nil {
		t.Fatalf("toCodeSubscription failed: %v", err)
	}
	if retrievedSub.ID != subID || len(retrievedSub.Occurrences) != 1 {
		t.Fatalf("retrieved subscription mismatch: %+v", retrievedSub)
	}
}
