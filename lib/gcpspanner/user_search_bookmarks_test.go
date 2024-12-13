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
	"testing"

	"cloud.google.com/go/spanner"
)

func TestUserSearchBookmark(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	savedSearchID, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "my little search",
		Query:       "group:css",
		OwnerUserID: "userID1",
		Description: nil,
	})
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}
	if savedSearchID == nil {
		t.Fatal("expected non-nil id.")
	}

	const testUser = "test-user"

	// user initially does not have a bookmark
	expectedSavedSearch := &UserSavedSearch{
		IsBookmarked: valuePtr(false),
		Role:         nil,
		SavedSearch: SavedSearch{
			ID:          *savedSearchID,
			Name:        "my little search",
			Query:       "group:css",
			Scope:       "USER_PUBLIC",
			AuthorID:    "userID1",
			Description: nil,
			// Don't actually compare the last two values.
			CreatedAt: spanner.CommitTimestamp,
			UpdatedAt: spanner.CommitTimestamp,
		},
	}
	actual, err := spannerClient.GetUserSavedSearch(ctx, *savedSearchID, valuePtr(testUser))
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}
	if !userSavedSearchEquality(expectedSavedSearch, actual) {
		t.Errorf("different saved searches\nexpected: %+v\nreceived: %v", expectedSavedSearch, actual)
	}

	// user can successfully have a bookmark added
	expectedSavedSearchAfter := &UserSavedSearch{
		IsBookmarked: valuePtr(true),
		Role:         nil,
		SavedSearch: SavedSearch{
			ID:          *savedSearchID,
			Name:        "my little search",
			Query:       "group:css",
			Scope:       "USER_PUBLIC",
			AuthorID:    "userID1",
			Description: nil,
			// Don't actually compare the last two values.
			CreatedAt: spanner.CommitTimestamp,
			UpdatedAt: spanner.CommitTimestamp,
		},
	}
	err = spannerClient.AddUserSearchBookmark(ctx, UserSavedSearchBookmark{
		UserID:        testUser,
		SavedSearchID: *savedSearchID,
	})
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}
	actual, err = spannerClient.GetUserSavedSearch(ctx, *savedSearchID, valuePtr(testUser))
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}
	if !userSavedSearchEquality(expectedSavedSearchAfter, actual) {
		t.Errorf("different saved searches\nexpected: %+v\nreceived: %v", expectedSavedSearchAfter, actual)
	}

	// user can successfully have a bookmark removed
	err = spannerClient.DeleteUserSearchBookmark(ctx, UserSavedSearchBookmark{
		UserID:        testUser,
		SavedSearchID: *savedSearchID,
	})
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}
	actual, err = spannerClient.GetUserSavedSearch(ctx, *savedSearchID, valuePtr(testUser))
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}
	if !userSavedSearchEquality(expectedSavedSearch, actual) {
		t.Errorf("different saved searches\nexpected: %+v\nreceived: %v", expectedSavedSearch, actual)
	}

}
