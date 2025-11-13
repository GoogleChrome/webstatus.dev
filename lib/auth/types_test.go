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

import "testing"

func TestUser_HasGitHubUserID(t *testing.T) {
	tests := []struct {
		name           string
		user           User
		inputID        int64
		expectedResult bool
	}{
		{
			name: "Matching ID",
			user: User{
				ID:           "user1",
				GitHubUserID: valuePtr("12345"),
			},
			inputID:        12345,
			expectedResult: true,
		},
		{
			name: "Non-matching ID",
			user: User{
				ID:           "user1",
				GitHubUserID: valuePtr("12345"),
			},
			inputID:        54321,
			expectedResult: false,
		},
		{
			name: "Nil GitHubUserID",
			user: User{
				ID:           "user1",
				GitHubUserID: nil,
			},
			inputID:        12345,
			expectedResult: false,
		},
		{
			name: "Zero ID",
			user: User{
				ID:           "user1",
				GitHubUserID: valuePtr("0"),
			},
			inputID:        0,
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.HasGitHubUserID(tt.inputID); got != tt.expectedResult {
				t.Errorf("User.HasGitHubUserID() = %v, want %v", got, tt.expectedResult)
			}
		})
	}
}

// valuePtr is a helper function to get a pointer to a value.
func valuePtr[T any](v T) *T {
	return &v
}
