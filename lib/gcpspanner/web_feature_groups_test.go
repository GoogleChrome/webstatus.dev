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

// nolint:dupl // WONTFIX
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

func setupRequiredTablesForWebFeatureGroup(
	ctx context.Context,
	t *testing.T,
) map[string]string {
	ret := map[string]string{}
	sampleFeatures := getSampleFeatures()
	for _, feature := range sampleFeatures {
		id, err := spannerClient.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert. %s", err.Error())

			continue
		}
		ret[feature.FeatureKey] = *id
	}

	return ret
}

func (c *Client) createSampleWebFeatureGroups(
	ctx context.Context, t *testing.T, idMap map[string]string) {
	err := c.UpsertWebFeatureGroup(ctx, WebFeatureGroup{
		WebFeatureID: idMap["feature1"],
		GroupIDs: []string{
			"group1",
			"group2",
		},
	})
	if err != nil {
		t.Fatalf("failed to insert group. err: %s group\n", err)
	}
	err = c.UpsertWebFeatureGroup(ctx, WebFeatureGroup{
		WebFeatureID: idMap["feature2"],
		GroupIDs:     nil,
	})
	if err != nil {
		t.Fatalf("failed to insert group. err: %s group\n", err)
	}
}

func (c *Client) ReadAllWebFeatureGroups(ctx context.Context, _ *testing.T) ([]WebFeatureGroup, error) {
	stmt := spanner.NewStatement(
		`SELECT
			WebFeatureID, GroupIDs
		FROM WebFeatureGroups`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []WebFeatureGroup
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var group spannerWebFeatureGroup
		if err := row.ToStruct(&group); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		ret = append(ret, group.WebFeatureGroup)
	}

	return ret, nil
}

func sortWebFeatureGroups(left, right WebFeatureGroup) int {
	return cmp.Compare(left.WebFeatureID, right.WebFeatureID)
}

func webFeatureGroupEquality(left, right WebFeatureGroup) bool {
	return left.WebFeatureID == right.WebFeatureID &&
		slices.Equal(left.GroupIDs, right.GroupIDs)
}

func TestUpsertWebFeatureGroup(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	idMap := setupRequiredTablesForWebFeatureGroup(ctx, t)
	spannerClient.createSampleWebFeatureGroups(ctx, t, idMap)

	expected := []WebFeatureGroup{
		{
			WebFeatureID: idMap["feature1"],
			GroupIDs: []string{
				"group1",
				"group2",
			},
		},
		{
			WebFeatureID: idMap["feature2"],
			GroupIDs:     nil,
		},
	}
	slices.SortFunc(expected, sortWebFeatureGroups)

	groups, err := spannerClient.ReadAllWebFeatureGroups(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all groups err: %s", err)
	}
	slices.SortFunc(groups, sortWebFeatureGroups)

	if !slices.EqualFunc(expected, groups, webFeatureGroupEquality) {
		t.Errorf("unequal groups.\nexpected %+v\nreceived %+v", expected, groups)
	}

	// Upsert group
	err = spannerClient.UpsertWebFeatureGroup(ctx, WebFeatureGroup{
		WebFeatureID: idMap["feature2"],
		GroupIDs: []string{
			"group3",
		},
	})
	if err != nil {
		t.Fatalf("unable to update group err: %s", err)
	}

	expected = []WebFeatureGroup{
		{
			WebFeatureID: idMap["feature1"],
			GroupIDs: []string{
				"group1",
				"group2",
			},
		},
		{
			WebFeatureID: idMap["feature2"],
			GroupIDs: []string{
				"group3",
			},
		},
	}
	slices.SortFunc(expected, sortWebFeatureGroups)

	groups, err = spannerClient.ReadAllWebFeatureGroups(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all groups err: %s", err)
	}
	slices.SortFunc(groups, sortWebFeatureGroups)

	if !slices.EqualFunc(expected, groups, webFeatureGroupEquality) {
		t.Errorf("unequal groups.\nexpected %+v\nreceived %+v", expected, groups)
	}
}
