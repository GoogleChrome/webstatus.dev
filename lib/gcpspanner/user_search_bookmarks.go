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

type UserSavedSearchBookmarkInfo struct {
	UserID        string `spanner:"UserID"`
	SavedSearchID string `spanner:"SavedSearchID"`
}

const userSavedSearchBookmarkTable = "UserSavedSearchBookmarks"

// Implements the entityMapper interface for UserSavedSearchBookmarkInfo.
type userSavedSearchBookmarkInfoMapper struct{}

func (m userSavedSearchBookmarkInfoMapper) Table() string {
	return userSavedSearchBookmarkTable
}

type userSavedSearchBookmarkInfoKey struct {
	UserSavedSearchBookmarkInfo
}

func (m userSavedSearchBookmarkInfoMapper) GetKey(
	in UserSavedSearchBookmarkInfo) userSavedSearchBookmarkInfoKey {
	return userSavedSearchBookmarkInfoKey{
		UserSavedSearchBookmarkInfo: in,
	}
}

func (m userSavedSearchBookmarkInfoMapper) Merge(
	_ UserSavedSearchBookmarkInfo, existing UserSavedSearchBookmarkInfo) UserSavedSearchBookmarkInfo {
	return existing
}

func (m userSavedSearchBookmarkInfoMapper) SelectOne(
	key userSavedSearchBookmarkInfoKey) spanner.Statement {
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

func (m userSavedSearchBookmarkInfoMapper) DeleteKey(key userSavedSearchBookmarkInfoKey) spanner.Key {
	return spanner.Key{key.UserID, key.SavedSearchID}
}

func (c *Client) AddUserSearchBookmark(ctx context.Context, req UserSavedSearchBookmarkInfo) error {
	return newEntityWriter[userSavedSearchBookmarkInfoMapper](c).upsert(ctx, req)
}

func (c *Client) DeleteUserSearchBookmark(ctx context.Context, req UserSavedSearchBookmarkInfo) error {
	return newEntityRemover[userSavedSearchBookmarkInfoMapper](c).remove(ctx, req)
}
