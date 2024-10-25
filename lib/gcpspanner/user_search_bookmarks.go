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

func (m userSavedSearchBookmarkMapper) GetKey(
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

func (c *Client) AddUserSearchBookmark(ctx context.Context, req UserSavedSearchBookmark) error {
	return newEntityWriter[userSavedSearchBookmarkMapper](c).upsert(ctx, req)
}

func (c *Client) DeleteUserSearchBookmark(ctx context.Context, req UserSavedSearchBookmark) error {
	return newEntityRemover[userSavedSearchBookmarkMapper](c).remove(ctx, req)
}
