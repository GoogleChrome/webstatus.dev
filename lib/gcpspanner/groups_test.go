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
	"slices"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func (c *Client) createSampleGroups(ctx context.Context, t *testing.T) map[string]string {
	grandChild1 := Group{
		GroupKey: "grandchild1",
		Name:     "Grand Child 1",
	}
	grandChild2 := Group{
		GroupKey: "grandchild2",
		Name:     "Grand Child 2",
	}
	grandChild1ID, err := c.UpsertGroup(ctx, grandChild1)
	if err != nil {
		t.Fatalf("failed to insert group. err: %s group: %v\n", err, grandChild1)
	}
	grandChild2ID, err := c.UpsertGroup(ctx, grandChild2)
	if err != nil {
		t.Fatalf("failed to insert group. err: %s group: %v\n", err, grandChild2)
	}
	child1 := Group{
		GroupKey: "child1",
		Name:     "Child 1",
	}
	child2 := Group{
		GroupKey: "child2",
		Name:     "Child 2",
	}
	child1ID, err := c.UpsertGroup(ctx, child1)
	if err != nil {
		t.Fatalf("failed to insert group. err: %s group: %v\n", err, child1)
	}
	child2ID, err := c.UpsertGroup(ctx, child2)
	if err != nil {
		t.Fatalf("failed to insert group. err: %s group: %v\n", err, child2)
	}
	parent1 := Group{
		GroupKey: "parent1",
		Name:     "Parent 1",
	}
	parent2 := Group{
		GroupKey: "parent2",
		Name:     "Parent 2",
	}
	parent1ID, err := c.UpsertGroup(ctx, parent1)
	if err != nil {
		t.Fatalf("failed to insert group. err: %s group: %v\n", err, parent1)
	}
	parent2ID, err := c.UpsertGroup(ctx, parent2)
	if err != nil {
		t.Fatalf("failed to insert group. err: %s group: %v\n", err, parent2)
	}

	return map[string]string{
		grandChild1.GroupKey: *grandChild1ID,
		grandChild2.GroupKey: *grandChild2ID,
		child1.GroupKey:      *child1ID,
		child2.GroupKey:      *child2ID,
		parent1.GroupKey:     *parent1ID,
		parent2.GroupKey:     *parent2ID,
	}
}

func (c *Client) ReadAllGroups(ctx context.Context, _ *testing.T) ([]Group, error) {
	stmt := spanner.NewStatement(
		`SELECT
			ID, GroupKey, Name
		FROM WebDXGroups ORDER BY GroupKey ASC`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []Group
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var group spannerGroup
		if err := row.ToStruct(&group); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		ret = append(ret, group.Group)
	}

	return ret, nil
}

func TestUpsertGroup(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	_ = spannerClient.createSampleGroups(ctx, t)

	groups, err := spannerClient.ReadAllGroups(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all groups err: %s", err)
	}

	expectedGroups := []Group{
		{
			GroupKey: "child1",
			Name:     "Child 1",
		},
		{
			GroupKey: "child2",
			Name:     "Child 2",
		},
		{
			GroupKey: "grandchild1",
			Name:     "Grand Child 1",
		},
		{
			GroupKey: "grandchild2",
			Name:     "Grand Child 2",
		},
		{
			GroupKey: "parent1",
			Name:     "Parent 1",
		},
		{
			GroupKey: "parent2",
			Name:     "Parent 2",
		},
	}

	if !slices.EqualFunc(expectedGroups, groups, groupEquality) {
		t.Errorf("unequal groups.\nexpected %+v\nreceived %+v", expectedGroups, groups)
	}

	// Change one of the groups
	_, err = spannerClient.UpsertGroup(ctx, Group{
		GroupKey: "parent2",
		// Change the name
		Name: "Parent 2 edit",
	})
	if err != nil {
		t.Errorf("unable to edit the group %s", err)
	}

	expectedGroups = []Group{
		{
			GroupKey: "child1",
			Name:     "Child 1",
		},
		{
			GroupKey: "child2",
			Name:     "Child 2",
		},
		{
			GroupKey: "grandchild1",
			Name:     "Grand Child 1",
		},
		{
			GroupKey: "grandchild2",
			Name:     "Grand Child 2",
		},
		{
			GroupKey: "parent1",
			Name:     "Parent 1",
		},
		{
			GroupKey: "parent2",
			Name:     "Parent 2 edit",
		},
	}

	groups, err = spannerClient.ReadAllGroups(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all groups err: %s", err)
	}

	if !slices.EqualFunc(expectedGroups, groups, groupEquality) {
		t.Errorf("unequal groups.\nexpected %+v\nreceived %+v", expectedGroups, groups)
	}
}

func groupEquality(left, right Group) bool {
	return left.Name == right.Name &&
		left.GroupKey == right.GroupKey
}
