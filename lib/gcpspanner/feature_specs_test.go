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

func getSampleFeatureSpecs() []struct {
	featureKey string
	spec       FeatureSpec
} {
	return []struct {
		featureKey string
		spec       FeatureSpec
	}{
		{
			featureKey: "feature1",
			spec: FeatureSpec{
				Links: nil,
			},
		},
		{
			featureKey: "feature2",
			spec: FeatureSpec{
				Links: []string{
					"http://example1.com",
					"http://example2.com",
				},
			},
		},
	}
}

func setupRequiredTablesForFeatureSpecs(ctx context.Context,
	client *Client, t *testing.T) {
	sampleFeatures := getSampleFeatures()
	for _, feature := range sampleFeatures {
		err := client.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert of features. %s", err.Error())
		}
	}
}

// Helper method to get all the statuses in a stable order.
func (c *Client) ReadAllFeatureSpecs(ctx context.Context, _ *testing.T) ([]FeatureSpec, error) {
	stmt := spanner.NewStatement("SELECT * FROM FeatureSpecs ORDER BY ARRAY_LENGTH(Links) ASC")
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []FeatureSpec
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var spec SpannerFeatureSpec
		if err := row.ToStruct(&spec); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}

		ret = append(ret, spec.FeatureSpec)
	}

	return ret, nil
}

func specEquality(left, right FeatureSpec) bool {
	return slices.Equal(left.Links, right.Links)
}

func TestUpsertFeatureSpec(t *testing.T) {
	client := getTestDatabase(t)
	ctx := context.Background()
	setupRequiredTablesForFeatureSpecs(ctx, client, t)
	sampleSpecs := getSampleFeatureSpecs()

	expectedSpecs := make([]FeatureSpec, 0, len(sampleSpecs))
	for _, spec := range sampleSpecs {
		expectedSpecs = append(expectedSpecs, spec.spec)
		err := client.UpsertFeatureSpec(ctx, spec.featureKey, spec.spec)
		if err != nil {
			t.Errorf("unexpected error during insert. %s", err.Error())
		}
	}

	specs, err := client.ReadAllFeatureSpecs(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	if !slices.EqualFunc[[]FeatureSpec](
		expectedSpecs,
		specs, specEquality) {
		t.Errorf("unequal status.\nexpected %+v\nreceived %+v", expectedSpecs, specs)
	}

	err = client.UpsertFeatureSpec(ctx, "feature1", FeatureSpec{
		Links: []string{
			"https://sample1.com",
			"https://sample2.com",
			"https://sample3.com",
		},
	})
	if err != nil {
		t.Errorf("unexpected error during update. %s", err.Error())
	}

	expectedPageAfterUpdate := []FeatureSpec{
		{
			Links: []string{
				"http://example1.com",
				"http://example2.com",
			},
		},
		{
			Links: []string{
				"https://sample1.com",
				"https://sample2.com",
				"https://sample3.com",
			},
		},
	}

	specs, err = client.ReadAllFeatureSpecs(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all after update. %s", err.Error())
	}
	if !slices.EqualFunc[[]FeatureSpec](
		expectedPageAfterUpdate,
		specs, specEquality) {
		t.Errorf("unequal spec.\nexpected %+v\nreceived %+v", expectedPageAfterUpdate, specs)
	}
}
