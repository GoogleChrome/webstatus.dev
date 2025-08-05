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
