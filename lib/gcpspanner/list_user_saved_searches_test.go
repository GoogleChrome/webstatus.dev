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
	"cmp"
	"context"
	"reflect"
	"slices"
	"testing"
	"time"
)

func loadFakeSavedSearches(t *testing.T) []SavedSearch {
	requests := []CreateUserSavedSearchRequest{
		{
			Name:        "z",
			Query:       "group:foo",
			OwnerUserID: "userID1",
			Description: nil,
		},
		{
			Name:        "a",
			Query:       "group:css",
			OwnerUserID: "userID1",
			Description: nil,
		},
		{
			Name:        "samename",
			Query:       "group:css",
			OwnerUserID: "userID1",
			Description: nil,
		},
		{
			Name:        "samename",
			Query:       "group:javascript",
			OwnerUserID: "userID1",
			Description: nil,
		},
	}
	searches := make([]SavedSearch, len(requests))
	for idx, request := range requests {
		id, err := spannerClient.CreateNewUserSavedSearch(context.Background(), request)
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}
		searches[idx] = SavedSearch{
			ID:          *id,
			Name:        request.Name,
			Query:       request.Query,
			AuthorID:    request.OwnerUserID,
			Description: request.Description,
			Scope:       UserPublicScope,
			// Timestamps don't matter for testing
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	slices.SortFunc(searches, func(a, b SavedSearch) int {
		ret := cmp.Compare(a.Name, b.Name)
		if ret == 0 {
			return cmp.Compare(a.ID, b.ID)
		}

		return ret
	})

	return searches
}

func loadFakeBookmarks(t *testing.T, savedSearchIDs []SavedSearch) {
	// Insert a bookmark for userID2 for the first saved search (name ="a")
	err := spannerClient.AddUserSearchBookmark(context.Background(), UserSavedSearchBookmark{
		UserID:        "userID2",
		SavedSearchID: savedSearchIDs[0].ID,
	})
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}
}

// userID1 owns all the saved searches which means the user also has them bookmarked automatically.
func testUserID1(t *testing.T, searches []SavedSearch) {
	expectedSearchs := make([]UserSavedSearch, len(searches))
	for idx, search := range searches {
		expectedSearchs[idx] = UserSavedSearch{
			SavedSearch: search,
			// userID1 should own and bookmark all of them
			IsBookmarked: valuePtr(true),
			Role:         valuePtr(string(SavedSearchOwner)),
		}
	}
	t.Run("list all the saved searches", func(t *testing.T) {
		expectedPage := &UserSavedSearchesPage{
			NextPageToken: nil,
			Searches:      expectedSearchs,
		}
		page, err := spannerClient.ListUserSavedSearches(context.Background(), "userID1", 100, nil)
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}
		if !userSavedSearchesPageEquality(page, expectedPage) {
			t.Errorf("\nexpected: %v\nreceived: %v.", expectedPage, page)
			t.FailNow()
		}
		// First result should be name = "a"
		assertSavedSearchName(t, "a", page.Searches[0])
		// Second and third result should be name = "samename"
		assertSavedSearchName(t, "samename", page.Searches[1])
		assertSavedSearchName(t, "samename", page.Searches[2])

		// Fourth result should be name = "z"
		assertSavedSearchName(t, "z", page.Searches[3])
	})

	t.Run("paginated", func(t *testing.T) {
		// Only request 2 results at a time
		// First page
		firstPageToken := valuePtr(encodeUserSavedSearchesCursor(searches[1].ID, searches[1].Name))
		expectedPage := &UserSavedSearchesPage{
			NextPageToken: firstPageToken,
			// Only the first 2
			Searches: expectedSearchs[0:2],
		}
		page, err := spannerClient.ListUserSavedSearches(context.Background(), "userID1", 2, nil)
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}
		if len(page.Searches) != 2 {
			t.Errorf("expected 2 results. received %d", len(page.Searches))
		}
		if !userSavedSearchesPageEquality(page, expectedPage) {
			t.Errorf("\nexpected: %v\nreceived: %v.", expectedPage, page)
			t.FailNow()
		}
		assertSavedSearchName(t, "a", page.Searches[0])
		assertSavedSearchName(t, "samename", page.Searches[1])
		// Second page
		secondPageToken := valuePtr(encodeUserSavedSearchesCursor(searches[3].ID, searches[3].Name))
		expectedPage = &UserSavedSearchesPage{
			NextPageToken: secondPageToken,
			// Only the second 2
			Searches: expectedSearchs[2:4],
		}
		page, err = spannerClient.ListUserSavedSearches(context.Background(), "userID1", 2, firstPageToken)
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}
		if len(page.Searches) != 2 {
			t.Errorf("expected 2 results. received %d", len(page.Searches))
		}
		if !userSavedSearchesPageEquality(page, expectedPage) {
			t.Errorf("\nexpected: %v\nreceived: %v.", expectedPage, page)
			t.FailNow()
		}
		assertSavedSearchName(t, "samename", page.Searches[0])
		assertSavedSearchName(t, "z", page.Searches[1])
		// Last Page
		expectedPage = &UserSavedSearchesPage{
			NextPageToken: nil,
			Searches:      nil,
		}
		page, err = spannerClient.ListUserSavedSearches(context.Background(), "userID1", 2, secondPageToken)
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}
		if len(page.Searches) != 0 {
			t.Errorf("expected 0 results. received %d", len(page.Searches))
		}
		if !userSavedSearchesPageEquality(page, expectedPage) {
			t.Errorf("\nexpected: %v\nreceived: %v.", expectedPage, page)
			t.FailNow()
		}
	})
}

// userID2 owns no saved searches but has one bookmarked.
func testUserID2(t *testing.T, searches []SavedSearch) {
	// Only keep the first saved search because it was the only one bookmarked
	expectedSearchs := make([]UserSavedSearch, 1)
	expectedSearchs[0] = UserSavedSearch{
		SavedSearch: searches[0],
		// userID2 should only have bookmark status for this search
		IsBookmarked: valuePtr(true),
		// userID2 should have no role for this search
		Role: nil,
	}
	t.Run("list all the saved searches should the bookmarked one", func(t *testing.T) {
		// userID2 should only have one bookmarked search
		expectedPage := &UserSavedSearchesPage{
			NextPageToken: nil,
			Searches:      expectedSearchs,
		}
		page, err := spannerClient.ListUserSavedSearches(context.Background(), "userID2", 100, nil)
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}
		if !userSavedSearchesPageEquality(page, expectedPage) {
			t.Errorf("\nexpected: %v\nreceived: %v.", expectedPage, page)
			t.FailNow()
		}
		if len(page.Searches) != 1 {
			t.Errorf("expected 1 results. received %d", len(page.Searches))
		}
		// First result should be name = "a"
		assertSavedSearchName(t, "a", page.Searches[0])
	})
}

// unknownUserID has no saved searches bookmarked.
func testUnknownUserID(t *testing.T) {
	t.Run("list all the saved searches should return none", func(t *testing.T) {
		// userID2 should only have one bookmarked search
		expectedPage := &UserSavedSearchesPage{
			NextPageToken: nil,
			Searches:      nil,
		}
		page, err := spannerClient.ListUserSavedSearches(context.Background(), "unknownUserID", 100, nil)
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}
		if !userSavedSearchesPageEquality(page, expectedPage) {
			t.Errorf("\nexpected: %v\nreceived: %v.", expectedPage, page)
			t.FailNow()
		}
		if len(page.Searches) != 0 {
			t.Errorf("expected 0 results. received %d", len(page.Searches))
		}
	})
}

func TestListUserSavedSearches(t *testing.T) {
	restartDatabaseContainer(t)
	searches := loadFakeSavedSearches(t)
	loadFakeBookmarks(t, searches)

	t.Run("userID1", func(t *testing.T) {
		testUserID1(t, searches)
	})

	t.Run("userID2", func(t *testing.T) {
		testUserID2(t, searches)
	})

	t.Run("unknownUserID", func(t *testing.T) {
		testUnknownUserID(t)
	})
}
func assertSavedSearchName(t *testing.T, name string, savedSearch UserSavedSearch) {
	if savedSearch.Name != name {
		t.Errorf("expected: %v\nreceived: %v.", name, savedSearch.Name)
	}
}

func userSavedSearchesPageEquality(left, right *UserSavedSearchesPage) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	if !reflect.DeepEqual(left.NextPageToken, right.NextPageToken) {
		return false
	}

	return slices.EqualFunc(left.Searches, right.Searches, func(a, b UserSavedSearch) bool {
		return userSavedSearchEquality(&a, &b)
	})
}
