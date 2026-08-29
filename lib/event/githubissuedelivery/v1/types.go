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

package v1

// IssueOccurrence represents a directive occurrence in code.
type IssueOccurrence struct {
	FilePath       string `json:"file_path"`
	LineNumber     int64  `json:"line_number"`
	CommentSnippet string `json:"comment_snippet"`
}

// GitHubIssueDeliveryEvent represents the Pub/Sub job payload for delivering an issue.
type GitHubIssueDeliveryEvent struct {
	DeliveryID         string            `json:"delivery_id"`
	SubscriptionID     string            `json:"subscription_id"`
	VCSProvider        string            `json:"vcs_provider"`
	VCSInstallationID  string            `json:"vcs_installation_id"`
	VCSRepositoryID    string            `json:"vcs_repository_id"`
	RepositoryOwner    string            `json:"repository_owner"`
	RepositoryName     string            `json:"repository_name"`
	RepositoryFullName string            `json:"repository_full_name"`
	FeatureID          string            `json:"feature_id"`
	FeatureName        string            `json:"feature_name"`
	Trigger            string            `json:"trigger"`
	CommitSHA          string            `json:"commit_sha"`
	Occurrences        []IssueOccurrence `json:"occurrences"`
	WebStatusURL       string            `json:"webstatus_url"`
}

func (GitHubIssueDeliveryEvent) Kind() string       { return "GitHubIssueDeliveryEvent" }
func (GitHubIssueDeliveryEvent) APIVersion() string { return "v1" }
