// Copyright 2024 Google LLC
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
	"reflect"
	"testing"

	"cloud.google.com/go/spanner"
)

func TestGetUserSavedSearch(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	savedSearchID, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "my little search",
		Query:       "group:css",
		OwnerUserID: "userID1",
	})
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}
	if savedSearchID == nil {
		t.Error("expected non-nil id.")
	}

	t.Run("unauthenticated user can access public search", func(t *testing.T) {
		expectedSavedSearch := &UserSavedSearch{
			IsBookmarked: nil,
			Role:         nil,
			SavedSearch: SavedSearch{
				ID:       *savedSearchID,
				Name:     "my little search",
				Query:    "group:css",
				Scope:    "USER_PUBLIC",
				AuthorID: "userID1",
				// Don't actually compare the last two values.
				CreatedAt: spanner.CommitTimestamp,
				UpdatedAt: spanner.CommitTimestamp,
			},
		}
		actual, err := spannerClient.GetUserSavedSearch(ctx, *savedSearchID, nil)
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}
		if !userSavedSearchEquality(expectedSavedSearch, actual) {
			t.Errorf("different saved searches\nexpected: %+v\nreceived: %v", expectedSavedSearch, actual)
		}
	})
	t.Run("owner can access public search with roles and bookmark", func(t *testing.T) {
		expectedSavedSearch := &UserSavedSearch{
			IsBookmarked: valuePtr(true),
			Role:         valuePtr(string(SavedSearchOwner)),
			SavedSearch: SavedSearch{
				ID:       *savedSearchID,
				Name:     "my little search",
				Query:    "group:css",
				Scope:    "USER_PUBLIC",
				AuthorID: "userID1",
				// Don't actually compare the last two values.
				CreatedAt: spanner.CommitTimestamp,
				UpdatedAt: spanner.CommitTimestamp,
			},
		}
		actual, err := spannerClient.GetUserSavedSearch(ctx, *savedSearchID, valuePtr("userID1"))
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}
		if !userSavedSearchEquality(expectedSavedSearch, actual) {
			t.Errorf("different saved searches\nexpected: %+v\nreceived: %v", expectedSavedSearch, actual)
		}
	})

	t.Run("other user can access public search. But unassigned roles and false bookmark", func(t *testing.T) {
		expectedSavedSearch := &UserSavedSearch{
			IsBookmarked: valuePtr(false),
			Role:         nil,
			SavedSearch: SavedSearch{
				ID:       *savedSearchID,
				Name:     "my little search",
				Query:    "group:css",
				Scope:    "USER_PUBLIC",
				AuthorID: "userID1",
				// Don't actually compare the last two values.
				CreatedAt: spanner.CommitTimestamp,
				UpdatedAt: spanner.CommitTimestamp,
			},
		}
		actual, err := spannerClient.GetUserSavedSearch(ctx, *savedSearchID, valuePtr("otherUser"))
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}
		if !userSavedSearchEquality(expectedSavedSearch, actual) {
			t.Errorf("different saved searches\nexpected: %+v\nreceived: %v", expectedSavedSearch, actual)
		}
	})
}

func userSavedSearchEquality(left, right *UserSavedSearch) bool {
	return (left == nil && right == nil) ||
		(left != nil && right != nil &&
			reflect.DeepEqual(left.IsBookmarked, right.IsBookmarked) &&
			reflect.DeepEqual(left.Role, right.Role) &&
			savedSearchEquality(left.SavedSearch, right.SavedSearch))

}

func savedSearchEquality(left, right SavedSearch) bool {
	return left.ID == right.ID &&
		left.Name == right.Name &&
		left.Query == right.Query &&
		left.Scope == right.Scope &&
		left.AuthorID == right.AuthorID &&
		// Just make sure the times are non zero.
		!left.CreatedAt.IsZero() && !right.CreatedAt.IsZero() &&
		!left.UpdatedAt.IsZero() && !right.UpdatedAt.IsZero()
}
