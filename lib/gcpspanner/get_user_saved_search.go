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
	"fmt"

	"cloud.google.com/go/spanner"
)

// UserSavedSearch represents a SavedSearch and a user's attributes related to that SavedSearch.
type UserSavedSearch struct {
	SavedSearch
	// Role will be nil if the user is not authenticated.
	Role *string `spanner:"Role"`
	// IsBookmarked will be nil if the user is not authenticated.
	IsBookmarked *bool `spanner:"IsBookmarked"`
}

// unauthenticatedUserSavedSearchMapper contains the entityMapper implementation for an unauthenticated user.
type unauthenticatedUserSavedSearchMapper struct{}

func (m unauthenticatedUserSavedSearchMapper) SelectOne(
	key string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID,
		Name,
		Query,
		Scope,
		AuthorID,
		CreatedAt,
		UpdatedAt
	FROM %s
	WHERE
		ID = @id
		AND Scope = 'USER_PUBLIC'
	LIMIT 1`,
		savedSearchesTable))
	parameters := map[string]interface{}{
		"id": key,
	}
	stmt.Params = parameters

	return stmt
}

// authenticatedUserSavedSearchMapper contains the entityMapper implementation for an authenticated user.
type authenticatedUserSavedSearchMapper struct{}

type authenticatedUserSavedSearchMapperKey struct {
	ID     string
	UserID string
}

func (m authenticatedUserSavedSearchMapper) SelectOne(
	key authenticatedUserSavedSearchMapperKey) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID,
		Name,
		Query,
		Scope,
		AuthorID,
		CreatedAt,
		UpdatedAt,
		r.UserRole AS Role,
		CASE
			WHEN b.UserID IS NOT NULL THEN TRUE
			ELSE FALSE
		END AS IsBookmarked
	FROM %s s
	LEFT JOIN
    	SavedSearchUserRoles r ON s.ID = r.SavedSearchID AND r.UserID = @userID
	LEFT JOIN
    	UserSavedSearchBookmarks b ON s.ID = b.SavedSearchID AND b.UserID = @userID
	WHERE
		s.ID = @id
		AND s.Scope = 'USER_PUBLIC'
	LIMIT 1`,
		savedSearchesTable))
	parameters := map[string]interface{}{
		"id":     key.ID,
		"userID": key.UserID,
	}
	stmt.Params = parameters

	return stmt
}

func (c *Client) GetUserSavedSearch(
	ctx context.Context,
	savedSearchID string,
	authenticatedUserID *string) (*UserSavedSearch, error) {
	if authenticatedUserID == nil {
		// For an unauthenticated user, we only read the SavedSearches row then fill in the rest of
		// UserSavedSearch struct with nil values.
		row, err := newEntityReader[unauthenticatedUserSavedSearchMapper, SavedSearch, string](c).
			readRowByKey(ctx, savedSearchID)
		if err != nil {
			return nil, err
		}

		return &UserSavedSearch{
			SavedSearch:  *row,
			IsBookmarked: nil,
			Role:         nil,
		}, nil
	}

	return newEntityReader[authenticatedUserSavedSearchMapper, UserSavedSearch, authenticatedUserSavedSearchMapperKey](c).
		readRowByKey(ctx, authenticatedUserSavedSearchMapperKey{
			UserID: *authenticatedUserID,
			ID:     savedSearchID,
		})
}
