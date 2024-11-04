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
	"log/slog"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SavedSearchRole is the enum for the saved searches role.
type SavedSearchRole string

const (
	// SavedSearchOwner indicates the user owns the saved search query.
	SavedSearchOwner SavedSearchRole = "OWNER"
)

const savedSearchUserRolesTable = "SavedSearchUserRoles"

// SavedSearchUserRole represents a user's role in relation to a saved search.
type SavedSearchUserRole struct {
	SavedSearchID string          `spanner:"SavedSearchID"`
	UserID        string          `spanner:"UserID"`
	UserRole      SavedSearchRole `spanner:"UserRole"`
}

func (c *Client) checkForSavedSearchRole(
	ctx context.Context, txn *spanner.ReadWriteTransaction, roleToCheck SavedSearchRole,
	userID string, savedSearchID string) error {
	var role string
	stmt := spanner.Statement{
		SQL: `SELECT UserRole
				  FROM SavedSearchUserRoles
				  WHERE SavedSearchID = @savedSearchID AND UserID = @userID`,
		Params: map[string]interface{}{
			"savedSearchID": savedSearchID,
			"userID":        userID,
		},
	}
	row, err := txn.Query(ctx, stmt).Next()
	if err != nil {
		// No row found. User does not have a role.
		if errors.Is(err, iterator.Done) {
			return errors.Join(ErrMissingRequiredRole, err)
		}
		slog.ErrorContext(ctx, "failed to query user role", "error", err)

		return errors.Join(ErrInternalQueryFailure, err)
	}
	if err := row.Columns(&role); err != nil {
		slog.ErrorContext(ctx, "failed to extract role from row", "error", err)

		return errors.Join(ErrInternalQueryFailure, err)
	}

	if role != string(roleToCheck) {
		return ErrMissingRequiredRole
	}

	return nil
}
