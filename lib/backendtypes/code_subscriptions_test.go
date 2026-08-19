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

package backendtypes

import (
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestVCSInstallationToSummary(t *testing.T) {
	t.Parallel()

	// 1. Nil input
	nilSummary := VCSInstallationToSummary(nil)
	if nilSummary.ID != "" || nilSummary.VCSProvider != "" || nilSummary.Permissions.GitHub != nil {
		t.Errorf("expected zero struct for nil installation, got %+v", nilSummary)
	}

	// 2. Populated input
	now := time.Now().UTC()
	writePerm := gcpspanner.GitHubPermissionLevelWrite
	inst := &gcpspanner.VCSInstallation{
		ID:                  "inst-uuid-1",
		VCSProvider:         gcpspanner.VCSProviderGitHub,
		VCSInstallationID:   "12345",
		AccountLogin:        "GoogleChrome",
		AccountType:         "Organization",
		RepositorySelection: "selected",
		Permissions: gcpspanner.VCSPermissions{
			GitHub: &gcpspanner.GitHubPermissions{
				Issues:       &writePerm,
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

	summary := VCSInstallationToSummary(inst)
	if summary.ID != "inst-uuid-1" {
		t.Errorf("summary.ID = %s, want inst-uuid-1", summary.ID)
	}
	ghPerms := summary.Permissions.GitHub
	if ghPerms == nil || ghPerms.Issues == nil || *ghPerms.Issues != "write" {
		t.Errorf("summary.Permissions.GitHub.Issues = %v, want write", ghPerms)
	}
}

func TestCodeSubscriptionsToResponse(t *testing.T) {
	t.Parallel()

	// 1. Empty input
	emptyResp := CodeSubscriptionsToResponse(nil)
	if len(emptyResp.Data) != 0 {
		t.Errorf("expected 0 data items for nil slice, got %d", len(emptyResp.Data))
	}

	// 2. Populated input
	now := time.Now().UTC()
	subs := []gcpspanner.CodeSubscription{
		{
			ID:                 "sub-uuid-1",
			VCSProvider:        "github",
			VCSInstallationID:  "12345",
			VCSRepositoryID:    "67890",
			RepositoryFullName: "GoogleChrome/webstatus.dev",
			TargetQuery:        "id:subgrid",
			Triggers:           []string{"feature_baseline_to_widely"},
			Status:             gcpspanner.SubscriptionActive,
			Occurrences: []gcpspanner.SubscriptionOccurrence{
				{
					FilePath:       "src/index.ts",
					LineNumber:     42,
					CommentSnippet: "// TODO(baseline/subgrid): grid",
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	resp := CodeSubscriptionsToResponse(subs)
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 response item, got %d", len(resp.Data))
	}

	item := resp.Data[0]
	if item.Id != "sub-uuid-1" {
		t.Errorf("item.Id = %s, want sub-uuid-1", item.Id)
	}
	if item.Status != backend.CodeSubscriptionResponseStatus(gcpspanner.SubscriptionActive) {
		t.Errorf("item.Status = %v, want ACTIVE", item.Status)
	}
	if item.OccurrenceCount != 1 {
		t.Errorf("item.OccurrenceCount = %d, want 1", item.OccurrenceCount)
	}
}
