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

package gcpspanner

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/spanner"
)

func TestDeleteUserSavedSearch(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	savedSearchID, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "my little search",
		Query:       "group:css",
		OwnerUserID: "userID1",
		Description: valuePtr("desc"),
	})
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}
	if savedSearchID == nil {
		t.Error("expected non-nil id.")
	}

	t.Run("non owner cannot delete search", func(t *testing.T) {
		err := spannerClient.DeleteUserSavedSearch(ctx, DeleteUserSavedSearchRequest{
			SavedSearchID:    *savedSearchID,
			RequestingUserID: "userID2",
		})
		if !errors.Is(err, ErrMissingRequiredRole) {
			t.Errorf("expected ErrMissingRequiredRole. received %s", err)
		}

		expectedSavedSearch := &UserSavedSearch{
			IsBookmarked: nil,
			Role:         nil,
			SavedSearch: SavedSearch{
				ID:          *savedSearchID,
				Name:        "my little search",
				Query:       "group:css",
				Scope:       "USER_PUBLIC",
				AuthorID:    "userID1",
				Description: valuePtr("desc"),
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

	t.Run("owner can delete search", func(t *testing.T) {
		err := spannerClient.DeleteUserSavedSearch(ctx, DeleteUserSavedSearchRequest{
			SavedSearchID:    *savedSearchID,
			RequestingUserID: "userID1",
		})
		if !errors.Is(err, nil) {
			t.Errorf("expected nil error. received %s", err)
		}
		_, err = spannerClient.GetUserSavedSearch(ctx, *savedSearchID, nil)
		if !errors.Is(err, ErrQueryReturnedNoResults) {
			t.Errorf("expected ErrQueryReturnedNoResults. received %s", err)
		}
	})

	t.Run("non existent search returns ErrQueryReturnedNoResults", func(t *testing.T) {
		err := spannerClient.DeleteUserSavedSearch(ctx, DeleteUserSavedSearchRequest{
			RequestingUserID: "userID1",
			SavedSearchID:    "bad-id",
		})
		if !errors.Is(err, ErrQueryReturnedNoResults) {
			t.Errorf("expected ErrQueryReturnedNoResults. received %s", err)
		}
	})

	t.Run("SYSTEM_MANAGED search cannot be deleted", func(t *testing.T) {
		// 1. Create a system managed search directly in the DB.
		systemSearchID := "system-search"
		_, err := spannerClient.Apply(ctx, []*spanner.Mutation{
			spanner.Insert("SavedSearches",
				[]string{"ID", "Name", "Query", "Scope", "AuthorID", "CreatedAt", "UpdatedAt"},
				[]interface{}{
					systemSearchID, "System Search", "id:foo", "SYSTEM_MANAGED",
					"system", spanner.CommitTimestamp, spanner.CommitTimestamp}),
		})
		if err != nil {
			t.Fatalf("failed to create system search: %v", err)
		}

		// 2. Attempt to delete it via the user API.
		err = spannerClient.DeleteUserSavedSearch(ctx, DeleteUserSavedSearchRequest{
			RequestingUserID: "userID1",
			SavedSearchID:    systemSearchID,
		})

		// 3. Verify it fails with 'not found' because the mapper filters it out.
		if !errors.Is(err, ErrQueryReturnedNoResults) {
			t.Errorf("expected ErrQueryReturnedNoResults, got %v", err)
		}
	})
}
