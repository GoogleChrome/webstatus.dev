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

const groupDescendantsTable = "WebDXGroupDescendants"

// Implements the entityMapper interface for GroupDescendantInfo and SpannerGroupDescendantInfo.
type groupDescendantInfoMapper struct{}

func (m groupDescendantInfoMapper) Merge(
	in spannerGroupDescendantInfo, existing spannerGroupDescendantInfo) spannerGroupDescendantInfo {
	return spannerGroupDescendantInfo{
		ID: existing.ID,
		GroupDescendantInfo: GroupDescendantInfo{
			DescendantGroupIDs: in.DescendantGroupIDs,
		},
	}
}

func (m groupDescendantInfoMapper) GetKey(in spannerGroupDescendantInfo) string {
	return in.ID
}

func (m groupDescendantInfoMapper) Table() string {
	return groupDescendantsTable
}

func (m groupDescendantInfoMapper) SelectOne(id string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		GroupID, DescendantGroupIDs
	FROM %s
	WHERE GroupID = @id
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"id": id,
	}
	stmt.Params = parameters

	return stmt
}

type GroupDescendantInfo struct {
	DescendantGroupIDs []string `spanner:"DescendantGroupIDs"`
}

type spannerGroupDescendantInfo struct {
	ID string `spanner:"GroupID"`
	GroupDescendantInfo
}

func (c *Client) UpsertGroupDescendantInfo(
	ctx context.Context, groupKey string, descendantInfo GroupDescendantInfo) error {
	id, err := c.GetGroupIDFromGroupKey(ctx, groupKey)
	if err != nil {
		return err
	}
	info := spannerGroupDescendantInfo{
		ID:                  *id,
		GroupDescendantInfo: descendantInfo,
	}

	return newEntityWriter[groupDescendantInfoMapper](c).upsert(ctx, info)
}
