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
	"slices"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func (c *Client) readAllGroupDescendantInfo(ctx context.Context, _ *testing.T) ([]spannerGroupDescendantInfo, error) {
	stmt := spanner.NewStatement(
		`SELECT
			GroupID, DescendantGroupIDs
		FROM WebDXGroupDescendants`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []spannerGroupDescendantInfo
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var info spannerGroupDescendantInfo
		if err := row.ToStruct(&info); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		ret = append(ret, info)
	}

	return ret, nil
}

func (c *Client) createSampleGroupDescendantInfo(
	ctx context.Context, t *testing.T, groupKeyToIDMapping map[string]string) {
	infoArr := []struct {
		groupKey string
		info     GroupDescendantInfo
	}{
		{
			groupKey: "child1",
			info: GroupDescendantInfo{
				DescendantGroupIDs: []string{
					groupKeyToIDMapping["grandchild1"],
					groupKeyToIDMapping["grandchild2"],
				},
			},
		},
		{
			groupKey: "parent1",
			info: GroupDescendantInfo{
				DescendantGroupIDs: []string{
					groupKeyToIDMapping["child1"],
					groupKeyToIDMapping["grandchild1"],
					groupKeyToIDMapping["grandchild2"],
				},
			},
		},
		{
			groupKey: "parent2",
			info: GroupDescendantInfo{
				DescendantGroupIDs: []string{
					groupKeyToIDMapping["child2"],
				},
			},
		},
	}
	for _, info := range infoArr {
		err := c.UpsertGroupDescendantInfo(ctx, info.groupKey, info.info)
		if err != nil {
			t.Fatalf("unable to insert group descendant info err %s", err)
		}
	}
}

func TestUpsertGroupDescendantInfo(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	idMapping := spannerClient.createSampleGroups(ctx, t)
	spannerClient.createSampleGroupDescendantInfo(ctx, t, idMapping)

	expectedInfoArr := []spannerGroupDescendantInfo{
		{
			ID: idMapping["child1"],
			GroupDescendantInfo: GroupDescendantInfo{
				DescendantGroupIDs: []string{
					idMapping["grandchild1"],
					idMapping["grandchild2"],
				},
			},
		},
		{
			ID: idMapping["parent1"],
			GroupDescendantInfo: GroupDescendantInfo{
				DescendantGroupIDs: []string{
					idMapping["child1"],
					idMapping["grandchild1"],
					idMapping["grandchild2"],
				},
			},
		},
		{
			ID: idMapping["parent2"],
			GroupDescendantInfo: GroupDescendantInfo{
				DescendantGroupIDs: []string{
					idMapping["child2"],
				},
			},
		},
	}
	infoArr, err := spannerClient.readAllGroupDescendantInfo(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all groups info err: %s", err)
	}
	slices.SortFunc(expectedInfoArr, sortGroupDescendantInfo)
	slices.SortFunc(infoArr, sortGroupDescendantInfo)
	if !slices.EqualFunc(expectedInfoArr, infoArr, groupDescendantInfoEquality) {
		t.Errorf("unequal groups.\nexpected %+v\nreceived %+v", expectedInfoArr, infoArr)
	}

	// Reset the descendants on parent2
	err = spannerClient.UpsertGroupDescendantInfo(ctx, "parent2", GroupDescendantInfo{
		DescendantGroupIDs: nil,
	})
	if err != nil {
		t.Errorf("unable to edit the group descendant info %s", err)
	}

	expectedInfoArr = []spannerGroupDescendantInfo{
		{
			ID: idMapping["child1"],
			GroupDescendantInfo: GroupDescendantInfo{
				DescendantGroupIDs: []string{
					idMapping["grandchild1"],
					idMapping["grandchild2"],
				},
			},
		},
		{
			ID: idMapping["parent1"],
			GroupDescendantInfo: GroupDescendantInfo{
				DescendantGroupIDs: []string{
					idMapping["child1"],
					idMapping["grandchild1"],
					idMapping["grandchild2"],
				},
			},
		},
		{
			ID: idMapping["parent2"],
			GroupDescendantInfo: GroupDescendantInfo{
				DescendantGroupIDs: nil,
			},
		},
	}
	infoArr, err = spannerClient.readAllGroupDescendantInfo(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all groups info err: %s", err)
	}
	slices.SortFunc(expectedInfoArr, sortGroupDescendantInfo)
	slices.SortFunc(infoArr, sortGroupDescendantInfo)
	if !slices.EqualFunc(expectedInfoArr, infoArr, groupDescendantInfoEquality) {
		t.Errorf("unequal groups.\nexpected %+v\nreceived %+v", expectedInfoArr, infoArr)
	}
}

func groupDescendantInfoEquality(left, right spannerGroupDescendantInfo) bool {
	return left.ID == right.ID &&
		slices.Equal(left.DescendantGroupIDs, right.DescendantGroupIDs)
}

func sortGroupDescendantInfo(left, right spannerGroupDescendantInfo) int {
	return cmp.Compare(left.ID, right.ID)
}
