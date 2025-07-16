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
	"context"
	"reflect"
	"slices"
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
		{GroupKey: "group-a", Name: "Group A"},
		{GroupKey: "group-b", Name: "Group B"},
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

func TestUpsertFeatureGroupLookups(t *testing.T) {
	t.Run("group look ups are inserted", func(t *testing.T) {
		restartDatabaseContainer(t)
		ctx := context.Background()
		_, featureKeyToID, _, groupKeyToID := setupTablesForUpsertFeatureGroupLookups(ctx, t)
		err := spannerClient.UpsertFeatureGroupLookups(ctx, []FeatureGroupIDsLookup{
			{ID: groupKeyToID["group-a"], WebFeatureID: featureKeyToID["FeatureX"], Depth: 0},
			{ID: groupKeyToID["group-b"], WebFeatureID: featureKeyToID["FeatureX"], Depth: 1},
			{ID: groupKeyToID["group-b"], WebFeatureID: featureKeyToID["FeatureZ"], Depth: 0},
		})
		if err != nil {
			t.Fatalf("UpsertFeatureGroupLookups failed: %v", err)
		}
		// Assert the expected look ups
		expectedLookups := []FeatureGroupIDsLookup{
			{ID: groupKeyToID["group-a"], WebFeatureID: featureKeyToID["FeatureX"], Depth: 0},
			{ID: groupKeyToID["group-b"], WebFeatureID: featureKeyToID["FeatureX"], Depth: 1},
			{ID: groupKeyToID["group-b"], WebFeatureID: featureKeyToID["FeatureZ"], Depth: 0},
		}

		assertFeatureGroupIDsLookups(ctx, t, expectedLookups)
	})
}

func assertFeatureGroupIDsLookups(ctx context.Context, t *testing.T, expectedEvents []FeatureGroupIDsLookup) {
	actualEvents := spannerClient.readAllFeatureGroupIDsLookups(ctx, t)

	// Assert that the actual events match the expected events
	slices.SortFunc(expectedEvents, sortFeatureGroupIDsLookups)
	slices.SortFunc(actualEvents, sortFeatureGroupIDsLookups)
	if !reflect.DeepEqual(expectedEvents, actualEvents) {
		t.Errorf("Unexpected data in FeatureGroupIDsLookups\nExpected (size: %d):\n%+v\nActual (size: %d):\n%+v",
			len(expectedEvents), expectedEvents, len(actualEvents), actualEvents)
	}
}
func sortFeatureGroupIDsLookups(a, b FeatureGroupIDsLookup) int {
	if a.WebFeatureID != b.WebFeatureID {
		return slices.Compare([]string{a.WebFeatureID}, []string{b.WebFeatureID})
	}
	if a.ID != b.ID {
		return slices.Compare([]string{a.ID}, []string{b.ID})
	}
	if a.Depth != b.Depth {
		return int(a.Depth - b.Depth)
	}

	return 0
}

func (c *Client) readAllFeatureGroupIDsLookups(ctx context.Context, t *testing.T) []FeatureGroupIDsLookup {
	// Fetch all rows from FeatureGroupIDsLookup
	stmt := spanner.Statement{
		SQL: `SELECT *
              FROM FeatureGroupIDsLookup`,
		Params: nil,
	}
	var actualEvents []FeatureGroupIDsLookup
	iter := spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	err := iter.Do(func(row *spanner.Row) error {
		var event FeatureGroupIDsLookup
		if err := row.ToStruct(&event); err != nil {
			return err
		}
		actualEvents = append(actualEvents, event)

		return nil
	})
	if err != nil {
		t.Fatalf("Failed to fetch data from FeatureGroupIDsLookup: %v", err)
	}

	return actualEvents
}
