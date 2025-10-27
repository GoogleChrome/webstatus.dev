// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/iterator"
)

func (c *Client) ReadAllWebFeaturesMappingData(ctx context.Context) ([]WebFeaturesMappingData, error) {
	stmt := spanner.NewStatement(`SELECT * FROM WebFeaturesMappingData`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []WebFeaturesMappingData
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {

			return nil, err
		}
		var data WebFeaturesMappingData
		if err := row.ToStruct(&data); err != nil {

			return nil, err
		}
		ret = append(ret, data)
	}

	return ret, nil
}

func TestSyncWebFeaturesMappingData(t *testing.T) {
	ctx := context.Background()

	// Define WebFeature fixtures that correspond to the mapping data.
	webFeatures := []WebFeature{
		{FeatureKey: "feature1", Name: "Feature 1", Description: "", DescriptionHTML: ""},
		{FeatureKey: "feature2", Name: "Feature 2", Description: "", DescriptionHTML: ""},
		{FeatureKey: "feature3", Name: "Feature 3", Description: "", DescriptionHTML: ""},
	}

	restartDatabaseContainer(t)

	// Insert WebFeatures to satisfy foreign key constraints.
	if err := spannerClient.SyncWebFeatures(ctx, webFeatures); err != nil {
		t.Fatalf("Failed to sync web features: %v", err)
	}

	initialState := []WebFeaturesMappingData{
		{
			WebFeatureID: "feature1",
			VendorPositions: spanner.NullJSON{
				Value: `[{"vendor":"mozilla","position":"positive"}]`,
				Valid: true,
			},
		},
		{
			WebFeatureID: "feature2",
			VendorPositions: spanner.NullJSON{
				Value: `[{"vendor":"webkit","position":"negative"}]`,
				Valid: true,
			},
		},
	}

	// 1. Setup initial state
	if err := spannerClient.SyncWebFeaturesMappingData(ctx, initialState); err != nil {
		t.Fatalf("Failed to set up initial state: %v", err)
	}

	// 2. Run the sync with the desired state
	desiredState := []WebFeaturesMappingData{
		{
			WebFeatureID: "feature1",
			VendorPositions: spanner.NullJSON{
				Value: `[{"vendor":"mozilla","position":"neutral"}]`,
				Valid: true,
			},
		},
		{
			WebFeatureID: "feature3",
			VendorPositions: spanner.NullJSON{
				Value: `[{"vendor":"w3c","position":"positive"}]`,
				Valid: true,
			},
		},
	}
	if err := spannerClient.SyncWebFeaturesMappingData(ctx, desiredState); err != nil {
		t.Fatalf("SyncWebFeaturesMappingData failed: %v", err)
	}

	// 3. Verify the final state
	data, err := spannerClient.ReadAllWebFeaturesMappingData(ctx)
	if err != nil {
		t.Fatalf("ReadAllWebFeaturesMappingData failed: %v", err)
	}

	featureKeyToID := make(map[string]string)
	// Get the generated IDs.
	stmt := spanner.NewStatement(`SELECT ID, FeatureKey FROM WebFeatures`)
	iter := spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			t.Fatalf("Failed to read web features: %v", err)
		}
		var feature SpannerWebFeature
		if err := row.ToStruct(&feature); err != nil {
			t.Fatalf("Failed to convert row to SpannerWebFeature: %v", err)
		}
		featureKeyToID[feature.FeatureKey] = feature.ID
	}

	expectedData := []WebFeaturesMappingData{
		{
			WebFeatureID: featureKeyToID["feature1"],
			VendorPositions: spanner.NullJSON{
				Value: `[{"vendor":"mozilla","position":"neutral"}]`,
				Valid: true,
			},
		},
		{
			WebFeatureID: featureKeyToID["feature3"],
			VendorPositions: spanner.NullJSON{
				Value: `[{"vendor":"w3c","position":"positive"}]`,
				Valid: true,
			},
		},
	}

	sortFunc := func(a, b WebFeaturesMappingData) int {
		return strings.Compare(a.WebFeatureID, b.WebFeatureID)
	}

	slices.SortFunc(data, sortFunc)
	slices.SortFunc(expectedData, sortFunc)

	if diff := cmp.Diff(expectedData, data); diff != "" {
		t.Errorf("data mismatch (-want +got):\n%s", diff)
	}
}
