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
		Description,
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
		Description,
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

// readSavedSearchMapper provides a way to read any SavedSearch by its ID, regardless of scope.
type readSavedSearchMapper struct{}

func (m readSavedSearchMapper) SelectOne(id string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID, Name, Description, Query, Scope, AuthorID, CreatedAt, UpdatedAt
	FROM %s
	WHERE ID = @id
	LIMIT 1`,
		savedSearchesTable))
	stmt.Params["id"] = id

	return stmt
}

// GetUserSavedSearch returns a single user saved search by its id.
// If the user is authenticated, it will also return their role and bookmark status.
func (c *Client) GetUserSavedSearch(
	ctx context.Context,
	savedSearchID string,
	authenticatedUserID *string) (*UserSavedSearch, error) {

	// Use a single read-only transaction for all operations.
	txn := c.ReadOnlyTransaction()
	defer txn.Close()

	// 1. Fetch the base SavedSearch using the new generic mapper.
	savedSearch, err := newEntityReader[readSavedSearchMapper, SavedSearch, string](c).
		readRowByKeyWithTransaction(ctx, savedSearchID, txn)
	if err != nil {
		return nil, err
	}

	// 2. If the user is unauthenticated, they can only see public or system searches.
	if authenticatedUserID == nil {
		if savedSearch.Scope != UserPublicScope && savedSearch.Scope != SystemManagedScope {
			return nil, ErrQueryReturnedNoResults
		}

		return &UserSavedSearch{SavedSearch: *savedSearch, Role: nil, IsBookmarked: nil}, nil
	}

	// 3. For an authenticated user, fetch their specific role and bookmark status
	// using the original restricted mapper within the SAME transaction.
	userSpecifics, err := newEntityReader[
		authenticatedUserSavedSearchMapper,
		UserSavedSearch,
		authenticatedUserSavedSearchMapperKey,
	](c).readRowByKeyWithTransaction(ctx, authenticatedUserSavedSearchMapperKey{
		UserID: *authenticatedUserID,
		ID:     savedSearchID,
	}, txn)

	// If there are no user-specific details (e.g., for a SYSTEM_MANAGED search),
	// that's okay. We just return the base search info.
	if err != nil {
		if errors.Is(err, ErrQueryReturnedNoResults) {
			return &UserSavedSearch{SavedSearch: *savedSearch, Role: nil, IsBookmarked: nil}, nil
		}

		return nil, err
	}

	return userSpecifics, nil
}
