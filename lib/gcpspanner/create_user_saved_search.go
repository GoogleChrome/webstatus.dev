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
	"github.com/google/uuid"
)

// CreateUserSavedSearchRequest is the request to create a new user saved search.
type CreateUserSavedSearchRequest struct {
	Name        string
	Query       string
	OwnerUserID string
}

var (
	// ErrOwnerSavedSearchLimitExceeded indicates that the user already has
	// reached the limit of saved searches that a given user can own.
	ErrOwnerSavedSearchLimitExceeded = errors.New("saved search limit reached")
)

// CreateNewUserSavedSearch creates a new user-owned saved search.
// It returns the ID of the newly created saved search if successful.
func (c *Client) CreateNewUserSavedSearch(
	ctx context.Context,
	newSearch CreateUserSavedSearchRequest) (*string, error) {
	id := uuid.NewString()
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Read the current count of owned searches
		var count int64
		stmt := spanner.Statement{
			SQL: fmt.Sprintf(`SELECT COUNT(*)
                  FROM %s
                  WHERE UserID = @OwnerID AND UserRole = @Role`, savedSearchUserRolesTable),
			Params: map[string]interface{}{
				"OwnerID": newSearch.OwnerUserID,
				"Role":    SavedSearchOwner,
			},
		}
		row, err := txn.Query(ctx, stmt).Next()
		if err != nil {
			return err
		}
		if err := row.Columns(&count); err != nil {
			return err
		}

		// 2. Check against the limit
		if count >= int64(c.searchCfg.maxOwnedSearchesPerUser) {
			return ErrOwnerSavedSearchLimitExceeded
		}

		var mutations []*spanner.Mutation
		// TODO: In the future, look into using an entityMapper for SavedSearch.
		// Then, we can use createInsertMutation.
		m1, err := spanner.InsertStruct(savedSearchesTable, SavedSearch{
			ID:        id,
			Name:      newSearch.Name,
			Query:     newSearch.Query,
			Scope:     UserPublicScope,
			AuthorID:  newSearch.OwnerUserID,
			CreatedAt: spanner.CommitTimestamp,
			UpdatedAt: spanner.CommitTimestamp,
		})
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}
		mutations = append(mutations, m1)

		// TODO: In the future, look into using an entityMapper for SavedSearchUserRole.
		// Then, we can use createInsertMutation.
		m2, err := spanner.InsertStruct(savedSearchUserRolesTable, SavedSearchUserRole{
			SavedSearchID: id,
			UserID:        newSearch.OwnerUserID,
			UserRole:      SavedSearchOwner,
		})
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}
		mutations = append(mutations, m2)

		// TODO: In the future, look into using an entityMapper for UserSavedSearchBookmark.
		// Then, we can use createInsertMutation.
		m3, err := spanner.InsertStruct(userSavedSearchBookmarksTable, UserSavedSearchBookmark{
			SavedSearchID: id,
			UserID:        newSearch.OwnerUserID,
		})
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}
		mutations = append(mutations, m3)

		err = txn.BufferWrite(mutations)
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &id, nil
}
