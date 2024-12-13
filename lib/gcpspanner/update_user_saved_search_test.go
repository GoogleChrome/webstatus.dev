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

func TestUpdateUserSavedSearch(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	savedSearchID, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "my little search",
		Query:       "group:css",
		OwnerUserID: "userID1",
		Description: valuePtr("initial description"),
	})
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}
	if savedSearchID == nil {
		t.Error("expected non-nil id.")
	}

	t.Run("non-owner cannot edit", func(t *testing.T) {
		expectedSavedSearch := &UserSavedSearch{
			IsBookmarked: valuePtr(true),
			Role:         valuePtr(string(SavedSearchOwner)),
			SavedSearch: SavedSearch{
				ID:          *savedSearchID,
				Name:        "my little search",
				Query:       "group:css",
				Scope:       "USER_PUBLIC",
				AuthorID:    "userID1",
				Description: valuePtr("initial description"),
				// Don't actually compare the last two values.
				CreatedAt: spanner.CommitTimestamp,
				UpdatedAt: spanner.CommitTimestamp,
			},
		}
		err := spannerClient.UpdateUserSavedSearch(ctx, UpdateSavedSearchRequest{
			ID:       *savedSearchID,
			AuthorID: "non-owner",
			Query: OptionallySet[string]{
				IsSet: true,
				Value: "junkquery",
			},
			Name: OptionallySet[string]{
				IsSet: true,
				Value: "junkName",
			},
			Description: OptionallySet[*string]{
				IsSet: true,
				Value: valuePtr("junkdesc"),
			},
		})
		if !errors.Is(err, ErrMissingRequiredRole) {
			t.Errorf("expected error trying to update %s", err)
		}
		actual, err := spannerClient.GetUserSavedSearch(ctx, *savedSearchID, valuePtr("userID1"))
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}
		if !userSavedSearchEquality(expectedSavedSearch, actual) {
			t.Errorf("different saved searches\nexpected: %+v\nreceived: %v", expectedSavedSearch, actual)
		}
	})
	t.Run("owner can edit but leave optional values unchanged", func(t *testing.T) {
		expectedSavedSearch := &UserSavedSearch{
			IsBookmarked: valuePtr(true),
			Role:         valuePtr(string(SavedSearchOwner)),
			SavedSearch: SavedSearch{
				ID:          *savedSearchID,
				Name:        "my little search",
				Query:       "group:css",
				Scope:       "USER_PUBLIC",
				AuthorID:    "userID1",
				Description: valuePtr("initial description"),
				// Don't actually compare the last two values.
				CreatedAt: spanner.CommitTimestamp,
				UpdatedAt: spanner.CommitTimestamp,
			},
		}
		err := spannerClient.UpdateUserSavedSearch(ctx, UpdateSavedSearchRequest{
			ID:       *savedSearchID,
			AuthorID: "userID1",
			Query: OptionallySet[string]{
				IsSet: false,
				Value: "",
			},
			Name: OptionallySet[string]{
				IsSet: false,
				Value: "",
			},
			Description: OptionallySet[*string]{
				IsSet: false,
				Value: nil,
			},
		})
		if !errors.Is(err, nil) {
			t.Errorf("expected nil error trying to update %s", err)
		}
		actual, err := spannerClient.GetUserSavedSearch(ctx, *savedSearchID, valuePtr("userID1"))
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}
		if !userSavedSearchEquality(expectedSavedSearch, actual) {
			t.Errorf("different saved searches\nexpected: %+v\nreceived: %v", expectedSavedSearch, actual)
		}
	})
	t.Run("owner can edit", func(t *testing.T) {
		expectedSavedSearch := &UserSavedSearch{
			IsBookmarked: valuePtr(true),
			Role:         valuePtr(string(SavedSearchOwner)),
			SavedSearch: SavedSearch{
				ID:          *savedSearchID,
				Name:        "my new search",
				Query:       "group:grid",
				Scope:       "USER_PUBLIC",
				AuthorID:    "userID1",
				Description: nil,
				// Don't actually compare the last two values.
				CreatedAt: spanner.CommitTimestamp,
				UpdatedAt: spanner.CommitTimestamp,
			},
		}
		err := spannerClient.UpdateUserSavedSearch(ctx, UpdateSavedSearchRequest{
			ID:       *savedSearchID,
			AuthorID: "userID1",
			Query: OptionallySet[string]{
				IsSet: true,
				Value: "group:grid",
			},
			Name: OptionallySet[string]{
				IsSet: true,
				Value: "my new search",
			},
			Description: OptionallySet[*string]{
				IsSet: true,
				Value: nil,
			},
		})
		if !errors.Is(err, nil) {
			t.Errorf("expected nil error trying to update %s", err)
		}
		actual, err := spannerClient.GetUserSavedSearch(ctx, *savedSearchID, valuePtr("userID1"))
		if err != nil {
			t.Errorf("expected nil error. received %s", err)
		}
		if !userSavedSearchEquality(expectedSavedSearch, actual) {
			t.Errorf("different saved searches\nexpected: %+v\nreceived: %v", expectedSavedSearch, actual)
		}
	})
}
