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
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/iterator"
)

func getSampleFeatures() []WebFeature {
	return []WebFeature{
		{
			Name:            "Feature 1",
			FeatureKey:      "feature1",
			Description:     "Wow what a feature description",
			DescriptionHTML: "Feature <b>1</b> description",
		},
		{
			Name:            "Feature 2",
			FeatureKey:      "feature2",
			Description:     "Feature 2 description",
			DescriptionHTML: "Feature <b>2</b> description",
		},
		{
			Name:            "Feature 3",
			FeatureKey:      "feature3",
			Description:     "Feature 3 description",
			DescriptionHTML: "Feature <b>3</b> description",
		},
		{
			Name:            "Feature 4",
			FeatureKey:      "feature4",
			Description:     "Feature 4 description",
			DescriptionHTML: "Feature <b>4</b> description",
		},
	}
}

// Helper method to get all the features in a stable order.
func (c *Client) ReadAllWebFeatures(ctx context.Context, t *testing.T) ([]WebFeature, error) {
	stmt := spanner.NewStatement(`SELECT
		ID, FeatureKey, Name, Description, DescriptionHtml
	FROM WebFeatures ORDER BY FeatureKey ASC`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []WebFeature
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var feature SpannerWebFeature
		if err := row.ToStruct(&feature); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		if feature.ID == "" {
			t.Error("retrieved feature ID is empty")
		}
		ret = append(ret, feature.WebFeature)
	}

	return ret, nil
}

func (c *Client) DeleteWebFeature(ctx context.Context, internalID string) error {
	_, err := c.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		mutation := spanner.Delete(webFeaturesTable, spanner.Key{internalID})

		return txn.BufferWrite([]*spanner.Mutation{mutation})
	})
	if err != nil {
		// TODO wrap the error and return it

		return err
	}

	return nil
}

func TestUpsertWebFeature(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	sampleFeatures := getSampleFeatures()
	for _, feature := range sampleFeatures {
		_, err := spannerClient.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert. %s", err.Error())
		}
	}
	features, err := spannerClient.ReadAllWebFeatures(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	if !slices.Equal(sampleFeatures, features) {
		t.Errorf("unequal features. expected %+v actual %+v", sampleFeatures, features)
	}

	_, err = spannerClient.UpsertWebFeature(ctx, WebFeature{
		Name:            "Feature 1!!",
		FeatureKey:      "feature1",
		Description:     "Feature 1 description!",
		DescriptionHTML: "Feature <i>1</i> description!",
	})
	if err != nil {
		t.Errorf("unexpected error during update. %s", err.Error())
	}

	features, err = spannerClient.ReadAllWebFeatures(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}

	expectedPageAfterUpdate := []WebFeature{
		{
			Name:            "Feature 1!!", // Updated field
			FeatureKey:      "feature1",
			Description:     "Feature 1 description!", // Updated field
			DescriptionHTML: "Feature <i>1</i> description!",
		},
		{
			Name:            "Feature 2",
			FeatureKey:      "feature2",
			Description:     "Feature 2 description",
			DescriptionHTML: "Feature <b>2</b> description",
		},
		{
			Name:            "Feature 3",
			FeatureKey:      "feature3",
			Description:     "Feature 3 description",
			DescriptionHTML: "Feature <b>3</b> description",
		},
		{
			Name:            "Feature 4",
			FeatureKey:      "feature4",
			Description:     "Feature 4 description",
			DescriptionHTML: "Feature <b>4</b> description",
		},
	}
	if !slices.Equal[[]WebFeature](expectedPageAfterUpdate, features) {
		t.Errorf("unequal features after update. expected %+v actual %+v", sampleFeatures, features)
	}

	expectedKeys := []string{
		"feature1",
		"feature2",
		"feature3",
		"feature4",
	}
	keys, err := spannerClient.FetchAllFeatureKeys(ctx)
	if err != nil {
		t.Errorf("unexpected error fetching all keys")
	}
	slices.Sort(keys)
	if !slices.Equal(keys, expectedKeys) {
		t.Errorf("unequal keys. expected %+v actual %+v", expectedKeys, keys)
	}
}

func TestSyncWebFeatures(t *testing.T) {
	ctx := context.Background()

	type syncTestCase struct {
		name          string
		initialState  []WebFeature
		desiredState  []WebFeature
		expectedState []WebFeature
	}

	testCases := []syncTestCase{
		{
			name:          "Initial creation",
			initialState:  nil, // No initial state
			desiredState:  getSampleFeatures(),
			expectedState: getSampleFeatures(),
		},
		{
			name:         "Deletes features not in desired state",
			initialState: getSampleFeatures(),
			desiredState: []WebFeature{
				getSampleFeatures()[0], // feature1
				getSampleFeatures()[2], // feature3
			},
			expectedState: []WebFeature{
				getSampleFeatures()[0],
				getSampleFeatures()[2],
			},
		},
		{
			name:         "Updates existing features",
			initialState: getSampleFeatures(),
			desiredState: func() []WebFeature {
				features := getSampleFeatures()
				features[1].Name = "UPDATED Feature 2"
				features[3].Description = "UPDATED Description 4"

				return features
			}(),
			expectedState: func() []WebFeature {
				features := getSampleFeatures()
				features[1].Name = "UPDATED Feature 2"
				features[3].Description = "UPDATED Description 4"

				return features
			}(),
		},
		{
			name:         "Performs mixed insert, update, and delete",
			initialState: getSampleFeatures(),
			desiredState: []WebFeature{
				{FeatureKey: "feature1", Name: "Updated Feature 1 Name", Description: "", DescriptionHTML: ""},
				getSampleFeatures()[2], // Keep feature3
				{FeatureKey: "feature5", Name: "New Feature 5", Description: "", DescriptionHTML: ""},
			},
			expectedState: []WebFeature{
				{
					FeatureKey:      "feature1",
					Name:            "Updated Feature 1 Name",
					Description:     "Wow what a feature description", // Preserved by merge logic
					DescriptionHTML: "Feature <b>1</b> description",   // Preserved by merge logic
				},
				getSampleFeatures()[2], // feature3 is unchanged
				{
					FeatureKey:      "feature5",
					Name:            "New Feature 5",
					Description:     "", // New fields are empty
					DescriptionHTML: "",
				},
			},
		},
		{
			name:          "No changes when desired state matches current state",
			initialState:  getSampleFeatures(),
			desiredState:  getSampleFeatures(),
			expectedState: getSampleFeatures(),
		},
		{
			name:          "Deletes all features when desired state is empty",
			initialState:  getSampleFeatures(),
			desiredState:  []WebFeature{},
			expectedState: []WebFeature{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restartDatabaseContainer(t)

			// 1. Setup initial state if provided
			if tc.initialState != nil {
				if err := spannerClient.SyncWebFeatures(ctx, tc.initialState); err != nil {
					t.Fatalf("Failed to set up initial state: %v", err)
				}
			}

			// 2. Run the sync with the desired state
			if err := spannerClient.SyncWebFeatures(ctx, tc.desiredState); err != nil {
				t.Fatalf("SyncWebFeatures failed: %v", err)
			}

			// 3. Verify the final state
			featuresInDB, err := spannerClient.ReadAllWebFeatures(ctx, t)
			if err != nil {
				t.Fatalf("ReadAllWebFeatures failed: %v", err)
			}

			if diff := cmp.Diff(tc.expectedState, featuresInDB); diff != "" {
				t.Errorf("features mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
