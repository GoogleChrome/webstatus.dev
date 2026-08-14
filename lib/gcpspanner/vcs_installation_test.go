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

func TestVCSInstallationConversion(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Microsecond)
	writeLevel := GitHubPermissionLevelWrite
	readLevel := GitHubPermissionLevelRead
	original := VCSInstallation{
		ID:                  "test-installation-uuid",
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
		CreatedAt: now,
		UpdatedAt: now,
	}

	mapper := vcsInstallationMapper{}
	wrapper, err := mapper.NewEntity("fallback-id", original)
	if err != nil {
		t.Fatalf("unexpected NewEntity error: %v", err)
	}

	converted, err := wrapper.toVCSInstallation()
	if err != nil {
		t.Fatalf("unexpected toVCSInstallation error: %v", err)
	}

	if diff := cmp.Diff(&original, converted); diff != "" {
		t.Errorf("VCSInstallation conversion mismatch (-want +got):\n%s", diff)
	}
}

func TestVCSInstallationConversionNilPermissions(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Microsecond)
	original := VCSInstallation{
		ID:                  "test-installation-nil-perm",
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   "87654321",
		AccountLogin:        "GoogleChrome",
		AccountType:         "Organization",
		RepositorySelection: "all",
		Permissions:         VCSPermissions{GitHub: nil},
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	mapper := vcsInstallationMapper{}
	wrapper, err := mapper.NewEntity("fallback-id", original)
	if err != nil {
		t.Fatalf("unexpected NewEntity error: %v", err)
	}

	converted, err := wrapper.toVCSInstallation()
	if err != nil {
		t.Fatalf("unexpected toVCSInstallation error: %v", err)
	}

	if diff := cmp.Diff(&original, converted); diff != "" {
		t.Errorf("VCSInstallation conversion mismatch (-want +got):\n%s", diff)
	}
}

func TestVCSInstallationConversionDefaultJSONBranch(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Microsecond)
	wrapper := spannerVCSInstallation{
		ID:                  "test-inst-default-json",
		VCSProvider:         string(VCSProviderGitHub),
		VCSInstallationID:   "11223344",
		AccountLogin:        "GoogleChrome",
		AccountType:         "Organization",
		RepositorySelection: "selected",
		Permissions: spanner.NullJSON{
			Value: map[string]any{
				"issues":   "write",
				"contents": "read",
			},
			Valid: true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	converted, err := wrapper.toVCSInstallation()
	if err != nil {
		t.Fatalf("unexpected toVCSInstallation error: %v", err)
	}

	if converted.Permissions.GitHub == nil {
		t.Fatalf("expected non-nil GitHub permissions")
	}
	if converted.Permissions.GitHub.Issues == nil ||
		*converted.Permissions.GitHub.Issues != GitHubPermissionLevelWrite {
		t.Errorf("unexpected issues permission: %v", converted.Permissions.GitHub.Issues)
	}
	if converted.Permissions.GitHub.Contents == nil ||
		*converted.Permissions.GitHub.Contents != GitHubPermissionLevelRead {
		t.Errorf("unexpected contents permission: %v", converted.Permissions.GitHub.Contents)
	}
}

func TestVCSInstallationMapper(t *testing.T) {
	t.Parallel()

	mapper := vcsInstallationMapper{}
	if mapper.Table() != "VCSInstallations" {
		t.Errorf("expected table VCSInstallations, got %s", mapper.Table())
	}

	stmt := mapper.SelectOne("test-id")
	if stmt.Params["id"] != "test-id" {
		t.Errorf("expected param id=test-id, got %v", stmt.Params["id"])
	}

	writeLevel := GitHubPermissionLevelWrite
	inst := VCSInstallation{
		ID:                  "test-id",
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   "12345",
		AccountLogin:        "google",
		AccountType:         "Organization",
		RepositorySelection: "all",
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
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if key := mapper.GetKeyFromExternal(inst); key != "test-id" {
		t.Errorf("expected key test-id, got %s", key)
	}

	existing := spannerVCSInstallation{
		ID:                  "test-id",
		VCSProvider:         string(VCSProviderGitHub),
		VCSInstallationID:   "12345",
		AccountLogin:        "google-old",
		AccountType:         "Organization",
		RepositorySelection: "selected",
		Permissions: spanner.NullJSON{
			Value: nil,
			Valid: false,
		},
		CreatedAt: time.Unix(100, 0),
		UpdatedAt: time.Unix(100, 0),
	}

	merged := mapper.Merge(inst, existing)
	if merged.ID != "test-id" {
		t.Errorf("expected merged ID test-id, got %s", merged.ID)
	}
	if merged.AccountLogin != "google" {
		t.Errorf("expected updated AccountLogin google, got %s", merged.AccountLogin)
	}
	if merged.CreatedAt != time.Unix(100, 0) {
		t.Errorf("expected preserved CreatedAt, got %v", merged.CreatedAt)
	}
}

func TestClient_VCSInstallation(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	now := time.Now().UTC().Add(-10 * time.Minute).Truncate(time.Microsecond)
	instID := uuid.NewString()

	// 1. Verify Get returns ErrVCSInstallationNotFound when record is missing
	_, err := spannerClient.GetVCSInstallation(ctx, "non-existent-id")
	if !errors.Is(err, ErrVCSInstallationNotFound) {
		t.Fatalf("expected ErrVCSInstallationNotFound, got %v", err)
	}

	writeLevel := GitHubPermissionLevelWrite
	readLevel := GitHubPermissionLevelRead

	// 2. Insert new installation via UpsertVCSInstallation
	original := VCSInstallation{
		ID:                  instID,
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
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := spannerClient.UpsertVCSInstallation(ctx, original); err != nil {
		t.Fatalf("UpsertVCSInstallation (insert) failed: %v", err)
	}

	retrieved, err := spannerClient.GetVCSInstallation(ctx, instID)
	if err != nil {
		t.Fatalf("GetVCSInstallation failed: %v", err)
	}

	// nolint:exhaustruct // Ignore CreatedAt and UpdatedAt
	ignoreFields := cmpopts.IgnoreFields(VCSInstallation{}, "CreatedAt", "UpdatedAt")
	if diff := cmp.Diff(&original, retrieved, ignoreFields); diff != "" {
		t.Errorf("GetVCSInstallation mismatch (-want +got):\n%s", diff)
	}
	if !retrieved.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", retrieved.CreatedAt, now)
	}

	// 3. Update existing installation (Merge semantics: update permissions & account, preserve CreatedAt)
	updateTime := now.Add(5 * time.Minute)
	spannerClient.setTimeNowForTesting(func() time.Time { return updateTime })
	defer spannerClient.setTimeNowForTesting(time.Now)

	updated := VCSInstallation{
		ID:                  instID,
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   "12345678",
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

	if err := spannerClient.UpsertVCSInstallation(ctx, updated); err != nil {
		t.Fatalf("UpsertVCSInstallation (update) failed: %v", err)
	}

	retrievedUpdated, err := spannerClient.GetVCSInstallation(ctx, instID)
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
		*retrievedUpdated.Permissions.GitHub.PullRequests != GitHubPermissionLevelWrite {
		t.Errorf("unexpected permissions after update: %+v", retrievedUpdated.Permissions.GitHub)
	}
	if !retrievedUpdated.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt was not preserved during update: got %v, want %v", retrievedUpdated.CreatedAt, now)
	}
	if !retrievedUpdated.UpdatedAt.Equal(updateTime) {
		t.Errorf("UpdatedAt was not updated: got %v, want %v", retrievedUpdated.UpdatedAt, updateTime)
	}

	// 4. Test Nil Permissions Round-Trip and GetVCSInstallationByProviderID
	nilPermInst := VCSInstallation{
		ID:                  uuid.NewString(),
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   "87654321",
		AccountLogin:        "GoogleChrome-NoPerm",
		AccountType:         "User",
		RepositorySelection: "all",
		Permissions:         VCSPermissions{GitHub: nil},
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := spannerClient.UpsertVCSInstallation(ctx, nilPermInst); err != nil {
		t.Fatalf("UpsertVCSInstallation (nil permissions) failed: %v", err)
	}

	retrievedNilPerm, err := spannerClient.GetVCSInstallation(ctx, nilPermInst.ID)
	if err != nil {
		t.Fatalf("GetVCSInstallation (nil permissions) failed: %v", err)
	}
	if retrievedNilPerm.Permissions.GitHub != nil {
		t.Errorf("expected nil GitHub permissions, got: %+v", retrievedNilPerm.Permissions.GitHub)
	}

	byProvider, err := spannerClient.GetVCSInstallationByProviderID(
		ctx,
		VCSProviderGitHub,
		"87654321",
	)
	if err != nil {
		t.Fatalf("GetVCSInstallationByProviderID failed: %v", err)
	}
	if byProvider.ID != nilPermInst.ID {
		t.Errorf("expected ID %s, got %s", nilPermInst.ID, byProvider.ID)
	}

	// 5. Upsert with empty ID should update existing installation by provider + installation ID
	upsertEmptyID := VCSInstallation{
		ID:                  "",
		VCSProvider:         VCSProviderGitHub,
		VCSInstallationID:   "87654321",
		AccountLogin:        "GoogleChrome-NoPerm-Renamed",
		AccountType:         "User",
		RepositorySelection: "selected",
		Permissions:         VCSPermissions{GitHub: nil},
		CreatedAt:           time.Time{},
		UpdatedAt:           time.Time{},
	}
	if err := spannerClient.UpsertVCSInstallation(ctx, upsertEmptyID); err != nil {
		t.Fatalf("UpsertVCSInstallation (empty ID update) failed: %v", err)
	}

	retrievedByProviderUpdated, err := spannerClient.GetVCSInstallationByProviderID(
		ctx,
		VCSProviderGitHub,
		"87654321",
	)
	if err != nil {
		t.Fatalf("GetVCSInstallationByProviderID after update failed: %v", err)
	}
	if retrievedByProviderUpdated.ID != nilPermInst.ID {
		t.Errorf("expected preserved ID %s, got %s", nilPermInst.ID, retrievedByProviderUpdated.ID)
	}
	if retrievedByProviderUpdated.AccountLogin != "GoogleChrome-NoPerm-Renamed" {
		t.Errorf("expected updated AccountLogin, got %s", retrievedByProviderUpdated.AccountLogin)
	}
}
