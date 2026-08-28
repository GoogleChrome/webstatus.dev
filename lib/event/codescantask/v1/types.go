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

// VCSProvider represents the supported VCS provider types for code scan tasks.
type VCSProvider string

const (
	VCSProviderGitHub VCSProvider = "github"
)

// CodeScanTaskEvent defines the Pub/Sub task payload for scanning a repository.
type CodeScanTaskEvent struct {
	VCSProvider        VCSProvider `json:"vcs_provider"`
	VCSInstallationID  string      `json:"vcs_installation_id"`
	VCSRepositoryID    string      `json:"vcs_repository_id"`
	RepositoryFullName string      `json:"repository_full_name"`
	CommitSHA          string      `json:"commit_sha,omitempty"`
	Branch             string      `json:"branch"`
	IsDefaultBranch    bool        `json:"is_default_branch"`
	ModifiedFiles      []string    `json:"modified_files,omitempty"`
}

func (CodeScanTaskEvent) Kind() string       { return "CodeScanTaskEvent" }
func (CodeScanTaskEvent) APIVersion() string { return "v1" }
