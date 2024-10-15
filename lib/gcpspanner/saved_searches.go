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
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

// NewSavedSearchRequest is the request to create a new user saved search.
type NewSavedSearchRequest struct {
	Name        string
	Query       string
	OwnerUserID string
}

// SearchConfig holds the application configuation for the saved search feature.
type SearchConfig struct {
	MaxOwnedSearchesPerUser uint32
}

// SavedSearchScope represents the scope of a saved search.
type SavedSearchScope string

const savedSearchesTable = "SavedSearches"

const (
	// UserPublicScope indicates that this is user created saved search meant to be publicly accessible.
	UserPublicScope SavedSearchScope = "USER_PUBLIC"
)

// SavedSearchRole is the enum for the saved searches role.
type SavedSearchRole string

const (
	// SavedSearchOwner indicates the user owns the saved search query.
	SavedSearchOwner = "OWNER"
)

var (
	// ErrOwnerSavedSearchLimitExceeded indicates that the user already has
	// reached the limit of saved searches that a given user can own.
	ErrOwnerSavedSearchLimitExceeded = errors.New("saved search limit reached")
)

// SavedSearch represents a saved search retrieved from the database.
type SavedSearch struct {
	ID        string           `spanner:"ID"`
	Name      string           `spanner:"Name"`
	Query     string           `spanner:"Query"`
	Scope     SavedSearchScope `spanner:"Scope"`
	AuthorID  string           `spanner:"AuthorID"`
	CreatedAt time.Time        `spanner:"CreatedAt"`
	UpdatedAt time.Time        `spanner:"UpdatedAt"`
}

const savedSearchUserRolesTable = "SavedSearchUserRoles"

// SavedSearchUserRole represents a user's role in relation to a saved search.
type SavedSearchUserRole struct {
	SavedSearchID string          `spanner:"SavedSearchID"`
	UserID        string          `spanner:"UserID"`
	UserRole      SavedSearchRole `spanner:"UserRole"`
}

// CreateNewUserSavedSearch creates a new user-owned saved search.
// It returns the ID of the newly created saved search if successful.
func (c *Client) CreateNewUserSavedSearch(ctx context.Context, cfg SearchConfig, newSearch NewSavedSearchRequest) (*string, error) {
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
		if count >= int64(cfg.MaxOwnedSearchesPerUser) {
			return ErrOwnerSavedSearchLimitExceeded
		}

		var mutations []*spanner.Mutation
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

		m2, err := spanner.InsertStruct(savedSearchUserRolesTable, SavedSearchUserRole{
			SavedSearchID: id,
			UserID:        newSearch.OwnerUserID,
			UserRole:      SavedSearchOwner,
		})
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}
		mutations = append(mutations, m2)

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
