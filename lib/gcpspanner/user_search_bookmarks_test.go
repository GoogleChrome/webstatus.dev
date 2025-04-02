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
	"errors"
	"testing"

	"cloud.google.com/go/spanner"
)

func assertUserSearchSearchWithBookmarkStatus(
	ctx context.Context, t *testing.T, expectedSavedSearch *UserSavedSearch, savedSearchID *string, userID string) {
	actual, err := spannerClient.GetUserSavedSearch(ctx, *savedSearchID, valuePtr(userID))
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}
	if !userSavedSearchEquality(expectedSavedSearch, actual) {
		t.Errorf("different saved searches\nexpected: %+v\nreceived: %v", expectedSavedSearch, actual)
	}
}

func TestUserSearchBookmark(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	// Reset the max bookmarks to 1.
	spannerClient.searchCfg.maxBookmarksPerUser = 1

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

	savedSearchID2, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "my big search",
		Query:       "group:html",
		OwnerUserID: "userID1",
		Description: nil,
	})
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}
	if savedSearchID2 == nil {
		t.Fatal("expected non-nil id.")
	}

	var testUserSavedSearchID *string
	const testUser = "test-user"
	// user initially does not have a bookmark
	expectedSavedSearchBeforeBookmark := &UserSavedSearch{
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
	t.Run("the test user can see they don't have bookmark status", func(t *testing.T) {
		assertUserSearchSearchWithBookmarkStatus(ctx, t, expectedSavedSearchBeforeBookmark, savedSearchID, testUser)
	})

	t.Run("the test user can bookmark it", func(t *testing.T) {
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

		assertUserSearchSearchWithBookmarkStatus(ctx, t, expectedSavedSearchAfter, savedSearchID, testUser)
	})

	t.Run("the test user gets a limit error once it tries to bookmark too many", func(t *testing.T) {
		err = spannerClient.AddUserSearchBookmark(ctx, UserSavedSearchBookmark{
			UserID:        testUser,
			SavedSearchID: *savedSearchID2,
		})
		if !errors.Is(err, ErrUserSearchBookmarkLimitExceeded) {
			t.Errorf("expected ErrUserSearchBookmarkLimitExceeded error. received %s", err)
		}
	})

	// Assuming they have not hit the saved search limit. (That is tested in create_user_saved_search_test.go)
	t.Run("the test user can still make saved searches even after hitting the bookmark limit", func(t *testing.T) {
		var err error
		testUserSavedSearchID, err = spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
			Name:        "my really big search",
			Query:       "group:html OR group:css",
			OwnerUserID: testUser,
			Description: nil,
		})
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}
		if testUserSavedSearchID == nil {
			t.Fatal("expected non-nil id.")
		}
	})

	t.Run("the test user can remove a bookmark for a saved search they don't own", func(t *testing.T) {
		err = spannerClient.DeleteUserSearchBookmark(ctx, UserSavedSearchBookmark{
			UserID:        testUser,
			SavedSearchID: *savedSearchID,
		})
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}

		assertUserSearchSearchWithBookmarkStatus(ctx, t, expectedSavedSearchBeforeBookmark, savedSearchID, testUser)
	})

	t.Run("the test user gets ErrQueryReturnedNoResults when trying to remove bookmark that does not exist (anymore)",
		func(t *testing.T) {
			err = spannerClient.DeleteUserSearchBookmark(ctx, UserSavedSearchBookmark{
				UserID:        testUser,
				SavedSearchID: *savedSearchID,
			})
			if !errors.Is(err, ErrQueryReturnedNoResults) {
				t.Errorf("expected ErrQueryReturnedNoResults error. received %s", err)
			}
		})

	t.Run("the test user gets ErrQueryReturnedNoResults when trying to add a bookmark for search that does not exist",
		func(t *testing.T) {
			err = spannerClient.AddUserSearchBookmark(ctx, UserSavedSearchBookmark{
				UserID:        testUser,
				SavedSearchID: "fake-uuid",
			})
			if !errors.Is(err, ErrQueryReturnedNoResults) {
				t.Errorf("expected ErrQueryReturnedNoResults error. received %s", err)
			}
		})

	t.Run("the test user cannot remove a bookmark for a saved search they own", func(t *testing.T) {
		err = spannerClient.DeleteUserSearchBookmark(ctx, UserSavedSearchBookmark{
			UserID:        testUser,
			SavedSearchID: *testUserSavedSearchID,
		})
		if !errors.Is(err, ErrOwnerCannotDeleteBookmark) {
			t.Errorf("expected ErrOwnerCannotDeleteBookmark error. received %s", err)
		}
	})
}
