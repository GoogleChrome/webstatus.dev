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
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// SubscriptionOccurrence represents an AST location where a directive was parsed.
type SubscriptionOccurrence struct {
	FilePath       string `json:"file_path"`
	LineNumber     int64  `json:"line_number"`
	CommentSnippet string `json:"comment_snippet"`
}

// CodeScanTaskMessage defines the Pub/Sub task payload for scanning a repository.
type CodeScanTaskMessage struct {
	VCSProvider        string   `json:"vcs_provider"`
	VCSInstallationID  string   `json:"vcs_installation_id"`
	VCSRepositoryID    string   `json:"vcs_repository_id"`
	RepositoryFullName string   `json:"repository_full_name"`
	CommitSHA          string   `json:"commit_sha"`
	Branch             string   `json:"branch"`
	IsDefaultBranch    bool     `json:"is_default_branch"`
	ModifiedFiles      []string `json:"modified_files,omitempty"`
}

// IssueDeliveryTaskMessage defines the Pub/Sub task payload for delivering an issue.
type IssueDeliveryTaskMessage struct {
	DeliveryID         string                   `json:"delivery_id"`
	SubscriptionID     string                   `json:"subscription_id"`
	VCSProvider        string                   `json:"vcs_provider"`
	VCSInstallationID  string                   `json:"vcs_installation_id"`
	VCSRepositoryID    string                   `json:"vcs_repository_id"`
	RepositoryFullName string                   `json:"repository_full_name"`
	TargetQuery        string                   `json:"target_query"`
	Trigger            string                   `json:"trigger"`
	Occurrences        []SubscriptionOccurrence `json:"occurrences"`
}

// VCSInstallationSummary represents the summary of an installation for API responses.
type VCSInstallationSummary struct {
	ID                  string                    `json:"id"`
	VCSProvider         string                    `json:"vcs_provider"`
	VCSInstallationID   string                    `json:"vcs_installation_id"`
	AccountLogin        string                    `json:"account_login"`
	AccountType         string                    `json:"account_type"`
	RepositorySelection string                    `json:"repository_selection"`
	Permissions         gcpspanner.VCSPermissions `json:"permissions"`
	CreatedAt           time.Time                 `json:"created_at"`
	UpdatedAt           time.Time                 `json:"updated_at"`
}

// VCSInstallationToSummary converts a Spanner VCSInstallation to summary format.
func VCSInstallationToSummary(in *gcpspanner.VCSInstallation) VCSInstallationSummary {
	if in == nil {
		return VCSInstallationSummary{
			ID:                  "",
			VCSProvider:         "",
			VCSInstallationID:   "",
			AccountLogin:        "",
			AccountType:         "",
			RepositorySelection: "",
			Permissions:         gcpspanner.VCSPermissions{GitHub: nil},
			CreatedAt:           time.Time{},
			UpdatedAt:           time.Time{},
		}
	}

	return VCSInstallationSummary{
		ID:                  in.ID,
		VCSProvider:         string(in.VCSProvider),
		VCSInstallationID:   in.VCSInstallationID,
		AccountLogin:        in.AccountLogin,
		AccountType:         in.AccountType,
		RepositorySelection: in.RepositorySelection,
		Permissions:         in.Permissions,
		CreatedAt:           in.CreatedAt,
		UpdatedAt:           in.UpdatedAt,
	}
}

// CodeSubscriptionsToResponse converts a slice of Spanner CodeSubscription to backend.CodeSubscriptionListResponse.
func CodeSubscriptionsToResponse(subs []gcpspanner.CodeSubscription) backend.CodeSubscriptionListResponse {
	results := make([]backend.CodeSubscriptionResponse, 0, len(subs))
	for _, sub := range subs {
		occurrences := make([]backend.SubscriptionOccurrence, 0, len(sub.Occurrences))
		for _, occ := range sub.Occurrences {
			occurrences = append(occurrences, backend.SubscriptionOccurrence{
				CommentSnippet: occ.CommentSnippet,
				FilePath:       occ.FilePath,
				LineNumber:     occ.LineNumber,
			})
		}

		triggers := make([]backend.SubscriptionTriggerWritable, 0, len(sub.Triggers))
		for _, tr := range sub.Triggers {
			triggers = append(triggers, backend.SubscriptionTriggerWritable(tr))
		}

		instID := sub.VCSInstallationID

		results = append(results, backend.CodeSubscriptionResponse{
			CreatedAt:          sub.CreatedAt,
			FeatureId:          nil,
			Id:                 sub.ID,
			OccurrenceCount:    int64(len(occurrences)),
			Occurrences:        occurrences,
			RawDirective:       nil,
			RepositoryFullName: sub.RepositoryFullName,
			RepositoryName:     "",
			RepositoryOwner:    "",
			Status:             backend.CodeSubscriptionResponseStatus(sub.Status),
			TargetQuery:        sub.TargetQuery,
			Triggers:           triggers,
			UpdatedAt:          sub.UpdatedAt,
			VcsInstallationId:  &instID,
			VcsProvider:        sub.VCSProvider,
			VcsRepositoryId:    sub.VCSRepositoryID,
		})
	}

	return backend.CodeSubscriptionListResponse{
		Data: results,
	}
}
