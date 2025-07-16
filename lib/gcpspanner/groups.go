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
	"fmt"

	"cloud.google.com/go/spanner"
)

const groupsTable = "WebDXGroups"

// Implements the entityMapper interface for Group and SpannerGroup.
type groupSpannerMapper struct{}

func (m groupSpannerMapper) Merge(in Group, existing spannerGroup) spannerGroup {
	return spannerGroup{
		ID: existing.ID,
		Group: Group{
			Name:     cmp.Or[string](in.Name, existing.Name),
			GroupKey: existing.GroupKey,
		},
	}
}

func (m groupSpannerMapper) GetKey(in Group) string {
	return in.GroupKey
}

func (m groupSpannerMapper) SelectOne(key string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID, GroupKey, Name
	FROM %s
	WHERE GroupKey = @groupKey
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"groupKey": key,
	}
	stmt.Params = parameters

	return stmt
}

func (m groupSpannerMapper) Table() string {
	return groupsTable
}

func (m groupSpannerMapper) GetID(key string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID
	FROM %s
	WHERE GroupKey = @groupKey
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"groupKey": key,
	}
	stmt.Params = parameters

	return stmt
}

// Group contains common metadata for a group from the WebDX web-feature
// repository.
// Columns come from the ../../infra/storage/spanner/migrations/*.sql files.
type Group struct {
	GroupKey string `spanner:"GroupKey"`
	Name     string `spanner:"Name"`
}

// spannerGroup is a wrapper for the group that is actually
// stored in spanner. This is useful because the spanner id is not useful to
// return to the end user since it is only used to decouple the primary keys
// between this system and web features repo.
type spannerGroup struct {
	ID string `spanner:"ID"`
	Group
}

func (c *Client) UpsertGroup(ctx context.Context, group Group) (*string, error) {
	return newEntityWriterWithIDRetrieval[groupSpannerMapper, string](c).upsertAndGetID(ctx, group)
}

func (c *Client) GetGroupIDFromGroupKey(ctx context.Context, groupKey string) (*string, error) {
	return newEntityWriterWithIDRetrieval[groupSpannerMapper, string](c).getIDByKey(ctx, groupKey)
}

type spannerGroupIDKeyAndKeyLowercase struct {
	ID                string `spanner:"ID"`
	GroupKey          string `spanner:"GroupKey"`
	GroupKeyLowercase string `spanner:"GroupKey_Lowercase"`
}

func (c *Client) fetchAllGroupIDsAndKeysWithTransaction(
	ctx context.Context, txn *spanner.ReadOnlyTransaction) ([]spannerGroupIDKeyAndKeyLowercase, error) {
	return fetchColumnValuesWithTransaction[spannerGroupIDKeyAndKeyLowercase](
		ctx, txn, groupsTable, []string{"ID", "GroupKey", "GroupKey_Lowercase"})
}
