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
	"context"
	"log/slog"

	"cloud.google.com/go/spanner"
)

// removeUserSavedSearchMapper implements removableEntityMapper.
type removeUserSavedSearchMapper struct{}

func (m removeUserSavedSearchMapper) Table() string { return savedSearchesTable }

func (m removeUserSavedSearchMapper) GetKey(in DeleteUserSavedSearchRequest) removeUserSavedSearchMapperKey {
	return removeUserSavedSearchMapperKey{
		ID:     in.SavedSearchID,
		UserID: in.RequestingUserID,
	}
}

type removeUserSavedSearchMapperKey struct {
	ID     string
	UserID string
}

func (m removeUserSavedSearchMapper) SelectOne(key removeUserSavedSearchMapperKey) spanner.Statement {
	return authenticatedUserSavedSearchMapper{}.SelectOne(
		authenticatedUserSavedSearchMapperKey(key))
}

func (m removeUserSavedSearchMapper) DeleteKey(key removeUserSavedSearchMapperKey) spanner.Key {
	return spanner.Key{key.ID}
}

// DeleteUserSavedSearchRequest contains the request parameters for DeleteUserSavedSearch.
type DeleteUserSavedSearchRequest struct {
	RequestingUserID string
	SavedSearchID    string
}

// DeleteUserSavedSearch deletes a user's saved search.
func (c *Client) DeleteUserSavedSearch(ctx context.Context, req DeleteUserSavedSearchRequest) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Check if the user has permission to delete (OWNER role)
		err := c.checkForSavedSearchRole(ctx, txn, SavedSearchOwner, req.RequestingUserID, req.SavedSearchID)
		if err != nil {
			return err
		}

		// 2. Read and update the existing saved search
		err = newEntityRemover[removeUserSavedSearchMapper, UserSavedSearch](c).removeWithTransaction(ctx, txn, req)
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
