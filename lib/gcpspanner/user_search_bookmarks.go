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

// UserSavedSearchBookmark represents a user's bookmark for a saved search.
type UserSavedSearchBookmark struct {
	UserID        string `spanner:"UserID"`
	SavedSearchID string `spanner:"SavedSearchID"`
}

const userSavedSearchBookmarksTable = "UserSavedSearchBookmarks"

// Implements the entityMapper interface for UserSavedSearchBookmark.
type userSavedSearchBookmarkMapper struct{}

func (m userSavedSearchBookmarkMapper) Table() string {
	return userSavedSearchBookmarksTable
}

type userSavedSearchBookmarkKey struct {
	UserSavedSearchBookmark
}

func (m userSavedSearchBookmarkMapper) GetKeyFromExternal(
	in UserSavedSearchBookmark) userSavedSearchBookmarkKey {
	return userSavedSearchBookmarkKey{
		UserSavedSearchBookmark: in,
	}
}

func (m userSavedSearchBookmarkMapper) Merge(
	_ UserSavedSearchBookmark, existing UserSavedSearchBookmark) UserSavedSearchBookmark {
	return existing
}

func (m userSavedSearchBookmarkMapper) SelectOne(
	key userSavedSearchBookmarkKey) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		SavedSearchID, UserID
	FROM %s
	WHERE UserID = @userID AND SavedSearchID = @savedSearchID
	LIMIT 1`,
		m.Table()))
	parameters := map[string]interface{}{
		"userID":        key.UserID,
		"savedSearchID": key.SavedSearchID,
	}
	stmt.Params = parameters

	return stmt
}

func (m userSavedSearchBookmarkMapper) DeleteKey(key userSavedSearchBookmarkKey) spanner.Key {
	return spanner.Key{key.UserID, key.SavedSearchID}
}

var (
	// ErrUserSearchBookmarkLimitExceeded indicates that the user already has
	// reached the limit of saved searches that a given user can bookmark.
	ErrUserSearchBookmarkLimitExceeded = errors.New("user bookmark limit reached")
	// ErrOwnerCannotDeleteBookmark indicates that the user is the owner of the
	// saved search and cannot delete the bookmark.
	ErrOwnerCannotDeleteBookmark = errors.New("user is the owner of the saved search and cannot delete the bookmark")
)

func (c *Client) AddUserSearchBookmark(ctx context.Context, req UserSavedSearchBookmark) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Check if search id exists for this user
		_, err := newEntityReader[
			authenticatedUserSavedSearchMapper,
			UserSavedSearch,
			authenticatedUserSavedSearchMapperKey,
		](c).readRowByKeyWithTransaction(ctx, authenticatedUserSavedSearchMapperKey{
			UserID: req.UserID,
			ID:     req.SavedSearchID,
		}, txn)
		if err != nil {
			return err
		}

		// 2. Read the current count of bookmarks where the user is not the owner
		var count int64
		stmt := spanner.Statement{
			SQL: fmt.Sprintf(`
			SELECT COUNT(us.SavedSearchID)
			FROM %s us
			LEFT JOIN %s sr ON us.SavedSearchID = sr.SavedSearchID AND us.UserID = sr.UserID
			WHERE us.UserID = @UserID AND (sr.UserRole != @Role OR sr.UserRole IS NULL);
		`, userSavedSearchBookmarksTable, savedSearchUserRolesTable),
			Params: map[string]interface{}{
				"UserID": req.UserID,
				"Role":   SavedSearchOwner,
			},
		}
		row, err := txn.Query(ctx, stmt).Next()
		if err != nil {
			return err
		}
		if err := row.Columns(&count); err != nil {
			return err
		}

		// 3. Check against the limit
		if count >= int64(c.searchCfg.maxBookmarksPerUser) {
			return ErrUserSearchBookmarkLimitExceeded
		}

		// 4. Insert the bookmark
		_, err = newEntityWriter[userSavedSearchBookmarkMapper](c).upsertWithTransaction(ctx, txn, req)

		return err
	})

	return err
}

func (c *Client) DeleteUserSearchBookmark(ctx context.Context, req UserSavedSearchBookmark) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Check if search id exists for this user
		_, err := newEntityReader[
			authenticatedUserSavedSearchMapper,
			UserSavedSearch,
			authenticatedUserSavedSearchMapperKey,
		](c).readRowByKeyWithTransaction(ctx, authenticatedUserSavedSearchMapperKey{
			UserID: req.UserID,
			ID:     req.SavedSearchID,
		}, txn)
		if err != nil {
			return err
		}

		// 2. Check if the user is the owner of the saved search
		var isOwner bool
		stmt := spanner.Statement{
			SQL: fmt.Sprintf(`
				SELECT EXISTS(
					SELECT 1
					FROM %s
					WHERE UserID = @UserID AND SavedSearchID = @SavedSearchID AND UserRole = @Role
				)
			`, savedSearchUserRolesTable),
			Params: map[string]interface{}{
				"UserID":        req.UserID,
				"SavedSearchID": req.SavedSearchID,
				"Role":          SavedSearchOwner,
			},
		}
		row, err := txn.Query(ctx, stmt).Next()
		if err != nil {
			return err
		}
		if err := row.Columns(&isOwner); err != nil {
			return err
		}

		if isOwner {
			return ErrOwnerCannotDeleteBookmark
		}

		// 3. Delete the bookmark
		return newEntityRemover[userSavedSearchBookmarkMapper, UserSavedSearchBookmark](c).
			removeWithTransaction(ctx, txn, req)
	})

	return err
}
