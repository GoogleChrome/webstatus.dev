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
	"cmp"
	"context"
	"reflect"
	"slices"
	"sync"
	"testing"

	"cloud.google.com/go/spanner"
)

func setupTablesForUpsertFeatureGroupLookups(
	ctx context.Context,
	t *testing.T,
) ([]WebFeature, map[string]string, []Group, map[string]string) {
	featureKeyToID := map[string]string{}
	groupKeyToID := map[string]string{}

	// 1. Insert sample data into WebFeatures
	features := []WebFeature{
		{FeatureKey: "FeatureX", Name: "Cool API", Description: "text", DescriptionHTML: "<html>"},
		{FeatureKey: "FeatureY", Name: "Super API", Description: "text", DescriptionHTML: "<html>"},
		{FeatureKey: "FeatureZ", Name: "Ultra API", Description: "text", DescriptionHTML: "<html>"},
	}
	for _, feature := range features {
		id, err := spannerClient.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Fatalf("Failed to insert WebFeature: %v", err)
		}
		featureKeyToID[feature.FeatureKey] = *id
	}

	// Insert sample groups
	groups := []Group{
		{GroupKey: "parent", Name: "Group A"},
		{GroupKey: "child", Name: "Group B"},
	}
	for _, group := range groups {
		id, err := spannerClient.UpsertGroup(ctx, group)
		if err != nil {
			t.Fatalf("Failed to insert Group: %v", err)
		}
		groupKeyToID[group.GroupKey] = *id
	}

	return features, featureKeyToID, groups, groupKeyToID
}

func TestCalculateAllFeatureGroupLookups(t *testing.T) {
	type testCase struct {
		name                          string
		featureKeyToID                map[string]string
		featureKeyToGroupsMapping     map[string][]string
		groupKeyToDetails             map[string]spannerGroupIDKeyAndKeyLowercase
		childGroupKeyToParentGroupKey map[string]string
		expectedLookups               []spannerFeatureGroupKeysLookup
	}

	testCases := []testCase{
		{
			name:                      "Deep Hierarchy",
			featureKeyToID:            map[string]string{"feat1": "feature_id_1"},
			featureKeyToGroupsMapping: map[string][]string{"feat1": {"grandchild"}},
			groupKeyToDetails: map[string]spannerGroupIDKeyAndKeyLowercase{
				"root":       {ID: "uuid_root", GroupKey: "root", GroupKeyLowercase: "root"},
				"child":      {ID: "uuid_child", GroupKey: "child", GroupKeyLowercase: "child"},
				"grandchild": {ID: "uuid_grandchild", GroupKey: "grandchild", GroupKeyLowercase: "grandchild"},
			},
			childGroupKeyToParentGroupKey: map[string]string{
				"child":      "root",
				"grandchild": "child",
			},
			expectedLookups: []spannerFeatureGroupKeysLookup{
				{GroupID: "uuid_grandchild", WebFeatureID: "feature_id_1", Depth: 0, GroupKeyLowercase: "grandchild"},
				{GroupID: "uuid_child", WebFeatureID: "feature_id_1", Depth: 1, GroupKeyLowercase: "child"},
				{GroupID: "uuid_root", WebFeatureID: "feature_id_1", Depth: 2, GroupKeyLowercase: "root"},
			},
		},
		{
			name:                      "Multiple Direct Groups",
			featureKeyToID:            map[string]string{"feat1": "feature_id_1"},
			featureKeyToGroupsMapping: map[string][]string{"feat1": {"child1", "child2"}},
			groupKeyToDetails: map[string]spannerGroupIDKeyAndKeyLowercase{
				"root":   {ID: "uuid_root", GroupKey: "root", GroupKeyLowercase: "root"},
				"child1": {ID: "uuid_child1", GroupKey: "child1", GroupKeyLowercase: "child1"},
				"child2": {ID: "uuid_child2", GroupKey: "child2", GroupKeyLowercase: "child2"},
			},
			childGroupKeyToParentGroupKey: map[string]string{
				"child1": "root",
				"child2": "root",
			},
			expectedLookups: []spannerFeatureGroupKeysLookup{
				{GroupID: "uuid_child1", WebFeatureID: "feature_id_1", Depth: 0, GroupKeyLowercase: "child1"},
				{GroupID: "uuid_root", WebFeatureID: "feature_id_1", Depth: 1, GroupKeyLowercase: "root"},
				{GroupID: "uuid_child2", WebFeatureID: "feature_id_1", Depth: 0, GroupKeyLowercase: "child2"},
				{GroupID: "uuid_root", WebFeatureID: "feature_id_1", Depth: 1, GroupKeyLowercase: "root"},
			},
		},
		{
			name:                      "Feature with No Group Mapping",
			featureKeyToID:            map[string]string{"feat1": "feature_id_1"},
			featureKeyToGroupsMapping: map[string][]string{}, // No mapping for feat1
			groupKeyToDetails: map[string]spannerGroupIDKeyAndKeyLowercase{
				"group1": {ID: "uuid_1", GroupKey: "group1", GroupKeyLowercase: "group1"}},
			childGroupKeyToParentGroupKey: map[string]string{},
			expectedLookups:               []spannerFeatureGroupKeysLookup{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entityChan := make(chan spannerFeatureGroupKeysLookup, 100) // Buffered channel
			var actualLookups []spannerFeatureGroupKeysLookup
			var wg sync.WaitGroup
			wg.Add(1)

			// This goroutine reads all results from the channel.
			go func() {
				defer wg.Done()
				for lookup := range entityChan {
					actualLookups = append(actualLookups, lookup)
				}
			}()

			// Run the function under test.
			calculateAllFeatureGroupLookups(
				context.Background(),
				tc.featureKeyToID,
				tc.featureKeyToGroupsMapping,
				tc.groupKeyToDetails,
				entityChan,
				tc.childGroupKeyToParentGroupKey,
			)
			// Close the channel to signal to the reader goroutine that we're done.
			close(entityChan)

			// Wait for the reader goroutine to finish processing.
			wg.Wait()

			// Sort both slices for a deterministic comparison.
			slices.SortFunc(tc.expectedLookups, sortFeatureGroupKeysLookups)
			slices.SortFunc(actualLookups, sortFeatureGroupKeysLookups)

			if !slices.Equal(actualLookups, tc.expectedLookups) {
				t.Errorf("lookup slice mismatch.\ngot= %v\nwant=%v", actualLookups, tc.expectedLookups)
			}
		})
	}
}

func TestUpsertFeatureGroupLookups(t *testing.T) {
	t.Run("group look ups are inserted", func(t *testing.T) {
		restartDatabaseContainer(t)
		ctx := context.Background()
		_, featureKeyToID, _, groupKeyToID := setupTablesForUpsertFeatureGroupLookups(ctx, t)
		err := spannerClient.UpsertFeatureGroupLookups(ctx,
			map[string][]string{
				"FeatureX": {"parent"},
				"FeatureZ": {"child"},
			},
			map[string]string{
				"child": "parent",
			},
		)
		if err != nil {
			t.Fatalf("UpsertFeatureGroupLookups failed: %v", err)
		}
		// Assert the expected look ups
		expectedLookups := []spannerFeatureGroupKeysLookup{
			{GroupID: groupKeyToID["parent"], WebFeatureID: featureKeyToID["FeatureX"], Depth: 0, GroupKeyLowercase: "parent"},
			{GroupID: groupKeyToID["parent"], WebFeatureID: featureKeyToID["FeatureZ"], Depth: 1, GroupKeyLowercase: "parent"},
			{GroupID: groupKeyToID["child"], WebFeatureID: featureKeyToID["FeatureZ"], Depth: 0, GroupKeyLowercase: "child"},
		}

		assertFeatureGroupKeysLookups(ctx, t, expectedLookups)
	})
}

func assertFeatureGroupKeysLookups(ctx context.Context, t *testing.T, expectedLookups []spannerFeatureGroupKeysLookup) {
	actualLookups := spannerClient.readAllFeatureGroupKeysLookups(ctx, t)

	// Assert that the actual events match the expected events
	slices.SortFunc(expectedLookups, sortFeatureGroupKeysLookups)
	slices.SortFunc(actualLookups, sortFeatureGroupKeysLookups)
	if !reflect.DeepEqual(expectedLookups, actualLookups) {
		t.Errorf("Unexpected data in FeatureGroupKeysLookup\nExpected (size: %d):\n%+v\nActual (size: %d):\n%+v",
			len(expectedLookups), expectedLookups, len(actualLookups), actualLookups)
	}
}
func sortFeatureGroupKeysLookups(a, b spannerFeatureGroupKeysLookup) int {
	if a.GroupKeyLowercase != b.GroupKeyLowercase {
		return cmp.Compare(a.GroupKeyLowercase, b.GroupKeyLowercase)
	}
	if a.WebFeatureID != b.WebFeatureID {
		return cmp.Compare(a.WebFeatureID, b.WebFeatureID)
	}
	if a.GroupID != b.GroupID {
		return cmp.Compare(a.GroupID, b.GroupID)
	}
	if a.Depth != b.Depth {
		return cmp.Compare(a.Depth, b.Depth)
	}

	return 0
}

func (c *Client) readAllFeatureGroupKeysLookups(ctx context.Context, t *testing.T) []spannerFeatureGroupKeysLookup {
	stmt := spanner.Statement{
		SQL: `SELECT *
              FROM FeatureGroupKeysLookup`,
		Params: nil,
	}
	var actualLookups []spannerFeatureGroupKeysLookup
	iter := spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	err := iter.Do(func(row *spanner.Row) error {
		var lookup spannerFeatureGroupKeysLookup
		if err := row.ToStruct(&lookup); err != nil {
			return err
		}
		actualLookups = append(actualLookups, lookup)

		return nil
	})
	if err != nil {
		t.Fatalf("Failed to fetch data from FeatureGroupKeysLookup: %v", err)
	}

	return actualLookups
}
