// Copyright 2025 Google LLC
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

package auth

import "strconv"

// User contains the details of an authenticated user.
type User struct {
	ID string
	// GitHubUserID is the string representation of the GitHub user's integer ID,
	// as returned by Firebase Auth.
	//
	// It is a pointer because it may be nil if the user is authenticated but not
	// linked to GitHub, or if the ID hasn't been verified against the GitHub API yet.
	// Verification is deferred until needed to keep most authenticated calls fast.
	GitHubUserID *string
}

// HasGitHubID checks if the authenticated user matches a specific GitHub integer ID.
// It handles the necessary string-to-int64 conversion safely.
func (u User) HasGitHubUserID(in int64) bool {
	if u.GitHubUserID == nil {
		return false
	}

	return *u.GitHubUserID == strconv.FormatInt(in, 10)
}
