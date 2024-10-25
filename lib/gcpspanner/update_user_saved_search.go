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
	"cmp"
	"context"
	"errors"
	"log/slog"

	"cloud.google.com/go/spanner"
)

// ErrMissingRequiredRole indicates that the user is missing the required role
// for the transaction.
var ErrMissingRequiredRole = errors.New("user is missing required role")

// UpdateSavedSearchRequest is a request to update the saved search.
type UpdateSavedSearchRequest struct {
	ID       string
	AuthorID string
	Query    *string
	Name     *string
}

type updateUserSavedSearchMapper struct {
	unauthenticatedUserSavedSearchMapper
}

func (m updateUserSavedSearchMapper) GetKey(in UpdateSavedSearchRequest) string { return in.ID }

func (m updateUserSavedSearchMapper) Table() string { return savedSearchesTable }

func (m updateUserSavedSearchMapper) Merge(req UpdateSavedSearchRequest, existing SavedSearch) SavedSearch {
	var incomingName, incomingQuery string
	if req.Name != nil {
		incomingName = *req.Name
	}
	if req.Query != nil {
		incomingQuery = *req.Query
	}

	return SavedSearch{
		ID:        existing.ID,
		Name:      cmp.Or(incomingName, existing.Name),
		Query:     cmp.Or(incomingQuery, existing.Query),
		Scope:     existing.Scope,
		AuthorID:  req.AuthorID,
		CreatedAt: existing.CreatedAt,
		UpdatedAt: spanner.CommitTimestamp,
	}
}

func (c *Client) UpdateUserSavedSearch(ctx context.Context, req UpdateSavedSearchRequest) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Check if the user has permission to update (OWNER role)
		err := c.checkForSavedSearchRole(ctx, txn, SavedSearchOwner, req.AuthorID, req.ID)
		if err != nil {
			return err
		}

		// 2. Read and update the existing saved search
		err = newEntityWriter[updateUserSavedSearchMapper](c).updateWithTransaction(ctx, txn, req)
		if err != nil {
			slog.ErrorContext(ctx, "failed to update the saved search", "error", err)

			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
