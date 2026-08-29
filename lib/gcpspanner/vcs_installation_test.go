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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
)

func TestVCSInstallation_PermissionsMapping(t *testing.T) {
	t.Parallel()

	writeLevel := GitHubPermissionLevelWrite
	readLevel := GitHubPermissionLevelRead

	testCases := []struct {
		name        string
		input       VCSInstallation
		checkOutput func(t *testing.T, converted *VCSInstallation)
	}{
		{
			name: "Populated GitHub permissions roundtrip",
			input: VCSInstallation{
				ID:                  "test-inst-1",
				VCSProvider:         VCSProviderGitHub,
				VCSInstallationID:   "12345678",
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
				CreatedAt: time.Unix(100, 0).UTC(),
				UpdatedAt: time.Unix(100, 0).UTC(),
			},
			checkOutput: func(t *testing.T, converted *VCSInstallation) {
				if converted.Permissions.GitHub == nil ||
					converted.Permissions.GitHub.Issues == nil ||
					*converted.Permissions.GitHub.Issues != writeLevel {
					t.Errorf("unexpected issues permission: %v", converted.Permissions.GitHub)
				}
			},
		},
		{
			name: "Nil GitHub permissions handled cleanly",
			input: VCSInstallation{
				ID:                  "test-inst-nil-perm",
				VCSProvider:         VCSProviderGitHub,
				VCSInstallationID:   "87654321",
				AccountLogin:        "GoogleChrome",
				AccountType:         "Organization",
				RepositorySelection: "all",
				Permissions:         VCSPermissions{GitHub: nil},
				CreatedAt:           time.Unix(100, 0).UTC(),
				UpdatedAt:           time.Unix(100, 0).UTC(),
			},
			checkOutput: func(t *testing.T, converted *VCSInstallation) {
				if converted.Permissions.GitHub != nil {
					t.Errorf("expected nil GitHub permissions, got %+v", converted.Permissions.GitHub)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mapper := vcsInstallationMapper{}
			wrapper, _, err := mapper.NewEntityWithID(tc.input)
			if err != nil {
				t.Fatalf("unexpected NewEntityWithID error: %v", err)
			}
			converted, err := wrapper.toVCSInstallation()
			if err != nil {
				t.Fatalf("unexpected toVCSInstallation error: %v", err)
			}
			tc.checkOutput(t, converted)
		})
	}
}

func TestVCSInstallation_InvalidProvider(t *testing.T) {
	t.Parallel()

	invalidWrapper := spannerVCSInstallation{
		ID:                  "test-id",
		VCSProvider:         "unsupported-provider",
		VCSInstallationID:   "123",
		AccountLogin:        "user",
		AccountType:         "User",
		RepositorySelection: "all",
		Permissions:         spanner.NullJSON{Value: nil, Valid: false},
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	_, err := invalidWrapper.toVCSInstallation()
	if err == nil {
		t.Fatalf("expected error for invalid provider, got nil")
	}
	if !errors.Is(err, ErrUnknownVCSProvider) {
		t.Fatalf("expected ErrUnknownVCSProvider, got %v", err)
	}
}

func testVCSInstallationNotFound(ctx context.Context, t *testing.T) {
	_, err := spannerClient.GetVCSInstallation(ctx, "non-existent-id")
	if !errors.Is(err, ErrVCSInstallationNotFound) {
		t.Fatalf("expected ErrVCSInstallationNotFound, got %v", err)
	}

	_, err = spannerClient.GetVCSInstallationByProviderID(ctx, VCSProviderGitHub, "non-existent-external-id")
	if !errors.Is(err, ErrVCSInstallationNotFound) {
		t.Fatalf("expected ErrVCSInstallationNotFound, got %v", err)
	}
}

func testVCSInstallationInsertAndGet(ctx context.Context, t *testing.T) {
	instID := uuid.NewString()
	now := time.Now().UTC().Truncate(time.Microsecond)
	writeLevel := GitHubPermissionLevelWrite
	readLevel := GitHubPermissionLevelRead
	inst := VCSInstallation{
		ID:                  instID,
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   "inst-insert-1",
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
		CreatedAt: now,
		UpdatedAt: now,
	}

	insertedID, err := spannerClient.UpsertVCSInstallation(ctx, inst)
	if err != nil {
		t.Fatalf("UpsertVCSInstallation failed: %v", err)
	}
	if insertedID == nil || *insertedID != instID {
		t.Fatalf("expected inserted ID %s, got %v", instID, insertedID)
	}

	retrieved, err := spannerClient.GetVCSInstallation(ctx, instID)
	if err != nil {
		t.Fatalf("GetVCSInstallation failed: %v", err)
	}

	//nolint:exhaustruct // Ignore CreatedAt and UpdatedAt
	ignoreFields := cmpopts.IgnoreFields(VCSInstallation{}, "CreatedAt", "UpdatedAt")
	if diff := cmp.Diff(&inst, retrieved, ignoreFields); diff != "" {
		t.Errorf("GetVCSInstallation mismatch (-want +got):\n%s", diff)
	}
	if retrieved.CreatedAt.IsZero() || retrieved.UpdatedAt.IsZero() {
		t.Errorf("expected non-zero commit timestamps")
	}
}

func testVCSInstallationGetByProviderID(ctx context.Context, t *testing.T) {
	extID := "ext-inst-query-2"
	inst := VCSInstallation{
		ID:                  "",
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   extID,
		AccountLogin:        "GoogleChrome-Ext",
		AccountType:         "User",
		RepositorySelection: "all",
		Permissions:         VCSPermissions{GitHub: nil},
		CreatedAt:           time.Time{},
		UpdatedAt:           time.Time{},
	}

	idPtr, err := spannerClient.UpsertVCSInstallation(ctx, inst)
	if err != nil {
		t.Fatalf("UpsertVCSInstallation failed: %v", err)
	}

	byProvider, err := spannerClient.GetVCSInstallationByProviderID(ctx, VCSProviderGitHub, extID)
	if err != nil {
		t.Fatalf("GetVCSInstallationByProviderID failed: %v", err)
	}
	if byProvider.ID != *idPtr {
		t.Errorf("expected ID %s, got %s", *idPtr, byProvider.ID)
	}
	if byProvider.Permissions.GitHub != nil {
		t.Errorf("expected nil GitHub permissions, got %+v", byProvider.Permissions.GitHub)
	}
}

func testVCSInstallationPartialPermissions(ctx context.Context, t *testing.T) {
	writeLevel := GitHubPermissionLevelWrite
	extID := "ext-inst-partial-3"
	inst := VCSInstallation{
		ID:                  "",
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   extID,
		AccountLogin:        "GoogleChrome-Partial",
		AccountType:         "Organization",
		RepositorySelection: "selected",
		Permissions: VCSPermissions{
			GitHub: &GitHubPermissions{
				Issues:       nil,
				Contents:     nil,
				Metadata:     nil,
				PullRequests: &writeLevel,
				Workflows:    nil,
				Actions:      nil,
			},
		},
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}

	idPtr, err := spannerClient.UpsertVCSInstallation(ctx, inst)
	if err != nil {
		t.Fatalf("UpsertVCSInstallation (partial permissions) failed: %v", err)
	}

	retrieved, err := spannerClient.GetVCSInstallation(ctx, *idPtr)
	if err != nil {
		t.Fatalf("GetVCSInstallation failed: %v", err)
	}
	if retrieved.Permissions.GitHub == nil {
		t.Fatalf("expected non-nil GitHub permissions")
	}
	if retrieved.Permissions.GitHub.PullRequests == nil ||
		*retrieved.Permissions.GitHub.PullRequests != writeLevel {
		t.Errorf("expected PullRequests write, got %+v", retrieved.Permissions.GitHub.PullRequests)
	}
	if retrieved.Permissions.GitHub.Issues != nil ||
		retrieved.Permissions.GitHub.Contents != nil ||
		retrieved.Permissions.GitHub.Metadata != nil ||
		retrieved.Permissions.GitHub.Workflows != nil ||
		retrieved.Permissions.GitHub.Actions != nil {
		t.Errorf("expected all other permission fields to remain nil, got %+v", retrieved.Permissions.GitHub)
	}
}

func testVCSInstallationMergeSemantics(ctx context.Context, t *testing.T) {
	writeLevel := GitHubPermissionLevelWrite
	readLevel := GitHubPermissionLevelRead
	extID := "ext-inst-merge-4"
	now := time.Now().UTC().Add(-10 * time.Minute).Truncate(time.Microsecond)
	initial := VCSInstallation{
		ID:                  "",
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   extID,
		AccountLogin:        "GoogleChrome-Initial",
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
		CreatedAt: now,
		UpdatedAt: now,
	}

	idPtr, err := spannerClient.UpsertVCSInstallation(ctx, initial)
	if err != nil {
		t.Fatalf("UpsertVCSInstallation (initial) failed: %v", err)
	}
	retrievedInitial, err := spannerClient.GetVCSInstallation(ctx, *idPtr)
	if err != nil {
		t.Fatalf("GetVCSInstallation (initial) failed: %v", err)
	}
	initialCreatedAt := retrievedInitial.CreatedAt

	updateTime := now.Add(5 * time.Minute)
	spannerClient.setTimeNowForTesting(func() time.Time { return updateTime })
	defer spannerClient.setTimeNowForTesting(time.Now)

	updated := VCSInstallation{
		ID:                  "",
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   extID,
		AccountLogin:        "GoogleChrome-Updated",
		AccountType:         "Organization",
		RepositorySelection: "all",
		Permissions: VCSPermissions{
			GitHub: &GitHubPermissions{
				Issues:       &writeLevel,
				Contents:     nil,
				Metadata:     nil,
				PullRequests: &writeLevel,
				Workflows:    nil,
				Actions:      nil,
			},
		},
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}

	updatedID, err := spannerClient.UpsertVCSInstallation(ctx, updated)
	if err != nil {
		t.Fatalf("UpsertVCSInstallation (update) failed: %v", err)
	}
	if updatedID == nil || *updatedID != *idPtr {
		t.Fatalf("expected preserved ID %s, got %v", *idPtr, updatedID)
	}

	retrievedUpdated, err := spannerClient.GetVCSInstallation(ctx, *idPtr)
	if err != nil {
		t.Fatalf("GetVCSInstallation after update failed: %v", err)
	}
	if retrievedUpdated.AccountLogin != "GoogleChrome-Updated" {
		t.Errorf("expected AccountLogin GoogleChrome-Updated, got %s", retrievedUpdated.AccountLogin)
	}
	if retrievedUpdated.RepositorySelection != "all" {
		t.Errorf("expected RepositorySelection all, got %s", retrievedUpdated.RepositorySelection)
	}
	if retrievedUpdated.Permissions.GitHub == nil ||
		retrievedUpdated.Permissions.GitHub.PullRequests == nil ||
		*retrievedUpdated.Permissions.GitHub.PullRequests != writeLevel {
		t.Errorf("unexpected permissions after update: %+v", retrievedUpdated.Permissions.GitHub)
	}
	if !retrievedUpdated.CreatedAt.Equal(initialCreatedAt) {
		t.Errorf("CreatedAt was not preserved: got %v, want %v", retrievedUpdated.CreatedAt, initialCreatedAt)
	}
}

func testVCSInstallationUniqueIndex(ctx context.Context, t *testing.T) {
	instA := VCSInstallation{
		ID:                  "",
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   "ext-inst-distinct-A",
		AccountLogin:        "AccountA",
		AccountType:         "Organization",
		RepositorySelection: "all",
		Permissions:         VCSPermissions{GitHub: nil},
		CreatedAt:           time.Time{},
		UpdatedAt:           time.Time{},
	}
	instB := VCSInstallation{
		ID:                  "",
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   "ext-inst-distinct-B",
		AccountLogin:        "AccountB",
		AccountType:         "Organization",
		RepositorySelection: "all",
		Permissions:         VCSPermissions{GitHub: nil},
		CreatedAt:           time.Time{},
		UpdatedAt:           time.Time{},
	}

	idAPtr, err := spannerClient.UpsertVCSInstallation(ctx, instA)
	if err != nil {
		t.Fatalf("UpsertVCSInstallation instA failed: %v", err)
	}
	idBPtr, err := spannerClient.UpsertVCSInstallation(ctx, instB)
	if err != nil {
		t.Fatalf("UpsertVCSInstallation instB failed: %v", err)
	}
	if *idAPtr == *idBPtr {
		t.Fatalf("expected distinct IDs for distinct installations, got identical: %s", *idAPtr)
	}

	retrievedA, err := spannerClient.GetVCSInstallation(ctx, *idAPtr)
	if err != nil {
		t.Fatalf("GetVCSInstallation A failed: %v", err)
	}
	retrievedB, err := spannerClient.GetVCSInstallation(ctx, *idBPtr)
	if err != nil {
		t.Fatalf("GetVCSInstallation B failed: %v", err)
	}
	if retrievedA.AccountLogin != "AccountA" || retrievedB.AccountLogin != "AccountB" {
		t.Errorf("account logins mismatch: A=%s B=%s", retrievedA.AccountLogin, retrievedB.AccountLogin)
	}
}

func testVCSInstallationList(ctx context.Context, t *testing.T) {
	inst1 := VCSInstallation{
		ID:                  uuid.NewString(),
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   "list-test-inst-1",
		AccountLogin:        "ListAccountA",
		AccountType:         "Organization",
		RepositorySelection: "all",
		Permissions:         VCSPermissions{GitHub: nil},
		CreatedAt:           time.Time{},
		UpdatedAt:           time.Time{},
	}
	inst2 := VCSInstallation{
		ID:                  uuid.NewString(),
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   "list-test-inst-2",
		AccountLogin:        "ListAccountB",
		AccountType:         "User",
		RepositorySelection: "selected",
		Permissions:         VCSPermissions{GitHub: nil},
		CreatedAt:           time.Time{},
		UpdatedAt:           time.Time{},
	}

	if _, err := spannerClient.UpsertVCSInstallation(ctx, inst1); err != nil {
		t.Fatalf("UpsertVCSInstallation inst1 failed: %v", err)
	}
	if _, err := spannerClient.UpsertVCSInstallation(ctx, inst2); err != nil {
		t.Fatalf("UpsertVCSInstallation inst2 failed: %v", err)
	}

	all, err := spannerClient.ListVCSInstallations(ctx)
	if err != nil {
		t.Fatalf("ListVCSInstallations failed: %v", err)
	}
	if len(all) < 2 {
		t.Fatalf("expected at least 2 installations from ListVCSInstallations, got %d", len(all))
	}

	byAccount, err := spannerClient.ListVCSInstallationsByAccount(ctx, VCSProviderGitHub, "ListAccountA")
	if err != nil {
		t.Fatalf("ListVCSInstallationsByAccount failed: %v", err)
	}
	if len(byAccount) != 1 {
		t.Fatalf("expected exactly 1 installation for ListAccountA, got %d", len(byAccount))
	}
	if byAccount[0].AccountLogin != "ListAccountA" {
		t.Errorf("unexpected account login: %s", byAccount[0].AccountLogin)
	}
}

func TestVCSInstallationMappers(t *testing.T) {
	t.Parallel()

	t.Run("GetAllMapper produces expected SQL", func(t *testing.T) {
		t.Parallel()
		mapper := vcsInstallationGetAllMapper{}
		stmt := mapper.SelectAll()
		if stmt.SQL == "" {
			t.Errorf("expected non-empty SQL statement")
		}
	})

	t.Run("ByAccountMapper produces expected SQL and parameters", func(t *testing.T) {
		t.Parallel()
		mapper := vcsInstallationByAccountMapper{}
		key := vcsInstallationByAccountKey{
			Provider:     string(VCSProviderGitHub),
			AccountLogin: "TestAccount",
		}
		stmt := mapper.SelectAllByKeys(key)
		if stmt.SQL == "" {
			t.Errorf("expected non-empty SQL statement")
		}
		if stmt.Params["vcsProvider"] != string(VCSProviderGitHub) {
			t.Errorf("expected param vcsProvider = %s, got %v", VCSProviderGitHub, stmt.Params["vcsProvider"])
		}
		if stmt.Params["accountLogin"] != "TestAccount" {
			t.Errorf("expected param accountLogin = TestAccount, got %v", stmt.Params["accountLogin"])
		}
	})
}

func TestClient_VCSInstallation(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	t.Run("NotFound returns ErrVCSInstallationNotFound", func(t *testing.T) {
		testVCSInstallationNotFound(ctx, t)
	})
	t.Run("Insert and retrieve by ID", func(t *testing.T) {
		testVCSInstallationInsertAndGet(ctx, t)
	})
	t.Run("Retrieve by provider and external installation ID", func(t *testing.T) {
		testVCSInstallationGetByProviderID(ctx, t)
	})
	t.Run("Partial permissions payload (only pull_requests)", func(t *testing.T) {
		testVCSInstallationPartialPermissions(ctx, t)
	})
	t.Run("Merge semantics update mutable fields and preserve CreatedAt", func(t *testing.T) {
		testVCSInstallationMergeSemantics(ctx, t)
	})
	t.Run("Unique index allows distinct installations per provider", func(t *testing.T) {
		testVCSInstallationUniqueIndex(ctx, t)
	})
	t.Run("List all and list by account", func(t *testing.T) {
		testVCSInstallationList(ctx, t)
	})
}
