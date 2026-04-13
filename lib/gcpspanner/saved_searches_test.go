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
	"slices"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

func TestSavedSearchScope(t *testing.T) {
	if UserPublicScope != "USER_PUBLIC" {
		t.Errorf("UserPublicScope should be USER_PUBLIC, but got %s", UserPublicScope)
	}
	if SystemManagedScope != "SYSTEM_MANAGED" {
		t.Errorf("SystemManagedScope should be SYSTEM_MANAGED, but got %s", SystemManagedScope)
	}
}

func TestGetSavedSearch(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	id := uuid.New().String()
	desc := systemSearchDesc
	expectedSavedSearch := &SavedSearch{
		ID:          id,
		Name:        "my system search",
		Query:       "feature:is(\"foo\")",
		Scope:       SystemManagedScope,
		AuthorID:    "system",
		Description: &desc,
		CreatedAt:   spanner.CommitTimestamp,
		UpdatedAt:   spanner.CommitTimestamp,
	}

	_, err := spannerClient.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		m, err := spanner.InsertStruct(savedSearchesTable, expectedSavedSearch)
		if err != nil {
			return err
		}

		return txn.BufferWrite([]*spanner.Mutation{m})
	})

	if err != nil {
		t.Fatalf("unexpected error during insert: %s", err)
	}

	actual, err := spannerClient.GetSavedSearch(ctx, id)
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}
	if !savedSearchEquality(*expectedSavedSearch, *actual) {
		t.Errorf("different saved searches\nexpected: %+v\nreceived: %v", expectedSavedSearch, actual)
	}
}

func TestGetUserSavedSearch(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	savedSearchID, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "my little search",
		Query:       "group:css",
		OwnerUserID: "userID1",
		Description: new("desc"),
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
				ID:          *savedSearchID,
				Name:        "my little search",
				Query:       "group:css",
				Scope:       "USER_PUBLIC",
				AuthorID:    "userID1",
				Description: new("desc"),
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
			IsBookmarked: new(true),
			Role:         new(string(SavedSearchOwner)),
			SavedSearch: SavedSearch{
				ID:          *savedSearchID,
				Name:        "my little search",
				Query:       "group:css",
				Scope:       "USER_PUBLIC",
				AuthorID:    "userID1",
				Description: new("desc"),
				// Don't actually compare the last two values.
				CreatedAt: spanner.CommitTimestamp,
				UpdatedAt: spanner.CommitTimestamp,
			},
		}
		actual, err := spannerClient.GetUserSavedSearch(ctx, *savedSearchID, new("userID1"))
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}
		if !userSavedSearchEquality(expectedSavedSearch, actual) {
			t.Errorf("different saved searches\nexpected: %+v\nreceived: %v", expectedSavedSearch, actual)
		}
	})

	t.Run("other user can access public search. But unassigned roles and false bookmark", func(t *testing.T) {
		expectedSavedSearch := &UserSavedSearch{
			IsBookmarked: new(false),
			Role:         nil,
			SavedSearch: SavedSearch{
				ID:          *savedSearchID,
				Name:        "my little search",
				Query:       "group:css",
				Scope:       "USER_PUBLIC",
				AuthorID:    "userID1",
				Description: new("desc"),
				// Don't actually compare the last two values.
				CreatedAt: spanner.CommitTimestamp,
				UpdatedAt: spanner.CommitTimestamp,
			},
		}
		actual, err := spannerClient.GetUserSavedSearch(ctx, *savedSearchID, new("otherUser"))
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
		reflect.DeepEqual(left.Description, right.Description) &&
		// Just make sure the times are non zero.
		!left.CreatedAt.IsZero() && !right.CreatedAt.IsZero() &&
		!left.UpdatedAt.IsZero() && !right.UpdatedAt.IsZero()
}

func TestGetReferencingSavedSearchIDs(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	targetID := uuid.New().String()

	// Note: We don't need to insert a saved search with targetID first because
	// GetReferencingSavedSearchIDs only performs string matching on the Query field
	// and does not enforce referential integrity or check for the actual existence
	// of the referenced search.

	// Insert searches that reference the targetID
	ref1 := &SavedSearch{
		ID:          uuid.New().String(),
		Name:        "ref 1",
		Query:       "saved:" + targetID,
		Scope:       UserPublicScope,
		AuthorID:    "user1",
		Description: nil,
		CreatedAt:   spanner.CommitTimestamp,
		UpdatedAt:   spanner.CommitTimestamp,
	}
	ref2 := &SavedSearch{
		ID:          uuid.New().String(),
		Name:        "ref 2",
		Query:       "hotlist:" + targetID,
		Scope:       UserPublicScope,
		AuthorID:    "user2",
		Description: nil,
		CreatedAt:   spanner.CommitTimestamp,
		UpdatedAt:   spanner.CommitTimestamp,
	}
	// Insert a search that does NOT reference the targetID
	noref := &SavedSearch{
		ID:          uuid.New().String(),
		Name:        "no ref",
		Query:       "saved:other-id",
		Scope:       UserPublicScope,
		AuthorID:    "user3",
		Description: nil,
		CreatedAt:   spanner.CommitTimestamp,
		UpdatedAt:   spanner.CommitTimestamp,
	}

	_, err := spannerClient.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		m1, _ := spanner.InsertStruct(savedSearchesTable, ref1)
		m2, _ := spanner.InsertStruct(savedSearchesTable, ref2)
		m3, _ := spanner.InsertStruct(savedSearchesTable, noref)

		return txn.BufferWrite([]*spanner.Mutation{m1, m2, m3})
	})
	if err != nil {
		t.Fatalf("unexpected error during insert: %s", err)
	}

	results, err := spannerClient.GetReferencingSavedSearchIDs(ctx, targetID)
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}

	expectedIDs := []string{ref1.ID, ref2.ID}
	if len(results) != len(expectedIDs) {
		t.Errorf("expected %d results, got %d", len(expectedIDs), len(results))
	}

	// Check that expected IDs are in results (order might not be guaranteed)
	for _, id := range expectedIDs {
		found := slices.Contains(results, id)
		if !found {
			t.Errorf("expected result to contain %s", id)
		}
	}
}
