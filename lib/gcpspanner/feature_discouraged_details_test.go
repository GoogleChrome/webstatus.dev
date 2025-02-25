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
	"errors"
	"slices"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func setupRequiredTablesForFeatureDiscouragedDetails(
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

func insertTestFeatureDiscouragedDetails(
	ctx context.Context,
	client *Client,
	t *testing.T,
	values []sampleDiscouragedDetail,
) {
	for _, value := range values {
		err := client.UpsertFeatureDiscouragedDetails(
			ctx,
			value.featureID,
			value.details,
		)
		if err != nil {
			t.Errorf("unexpected error during insert of discouraged details. %s", err.Error())
		}
	}
}

func (c *Client) readAllFeatureDiscouragedDetails(
	ctx context.Context, _ *testing.T) ([]spannerFeatureDiscouragedDetails, error) {
	stmt := spanner.NewStatement(
		`SELECT
			WebFeatureID, AccordingTo, Alternatives
		FROM FeatureDiscouragedDetails`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []spannerFeatureDiscouragedDetails
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var details spannerFeatureDiscouragedDetails
		if err := row.ToStruct(&details); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		ret = append(ret, details)
	}

	return ret, nil
}

type sampleDiscouragedDetail struct {
	featureID string
	details   FeatureDiscouragedDetails
}

func getSampleDiscouragedDetails() []sampleDiscouragedDetail {
	return []sampleDiscouragedDetail{
		{
			featureID: "feature1",
			details: FeatureDiscouragedDetails{
				AccordingTo:  []string{"source"},
				Alternatives: nil,
			},
		},
		{
			featureID: "feature2",
			details: FeatureDiscouragedDetails{
				AccordingTo:  nil,
				Alternatives: []string{"featurefoo"},
			},
		},
	}
}

func sortFeatureDiscouragedDetails(left, right spannerFeatureDiscouragedDetails) int {
	return cmp.Compare(left.WebFeatureID, right.WebFeatureID)
}

func featureDiscouragedDetailsEquality(left, right spannerFeatureDiscouragedDetails) bool {
	return left.WebFeatureID == right.WebFeatureID &&
		slices.Equal(left.AccordingTo, right.AccordingTo) &&
		slices.Equal(left.Alternatives, right.Alternatives)
}

func TestUpsertFeatureDiscouragedDetails(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	idMap := setupRequiredTablesForFeatureDiscouragedDetails(ctx, t)
	discouragedDetails := getSampleDiscouragedDetails()
	insertTestFeatureDiscouragedDetails(ctx, spannerClient, t, discouragedDetails)

	expected := []spannerFeatureDiscouragedDetails{
		{
			WebFeatureID: idMap["feature1"],
			FeatureDiscouragedDetails: FeatureDiscouragedDetails{
				AccordingTo:  []string{"source"},
				Alternatives: nil,
			},
		},
		{
			WebFeatureID: idMap["feature2"],
			FeatureDiscouragedDetails: FeatureDiscouragedDetails{
				AccordingTo:  nil,
				Alternatives: []string{"featurefoo"},
			},
		},
	}
	slices.SortFunc(expected, sortFeatureDiscouragedDetails)

	details, err := spannerClient.readAllFeatureDiscouragedDetails(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all discouraged details err: %s", err)
	}
	slices.SortFunc(details, sortFeatureDiscouragedDetails)

	if !slices.EqualFunc(expected, details, featureDiscouragedDetailsEquality) {
		t.Errorf("unequal discouraged details.\nexpected %+v\nreceived %+v", expected, details)
	}

	// Upsert FeatureDiscouragedDetails
	err = spannerClient.UpsertFeatureDiscouragedDetails(ctx, "feature2", FeatureDiscouragedDetails{
		AccordingTo: []string{"source2"},
		// Reset to nil
		Alternatives: nil,
	})
	if err != nil {
		t.Fatalf("unable to update discouraged details err: %s", err)
	}

	expected = []spannerFeatureDiscouragedDetails{
		{
			WebFeatureID: idMap["feature1"],
			FeatureDiscouragedDetails: FeatureDiscouragedDetails{
				AccordingTo:  []string{"source"},
				Alternatives: nil,
			},
		},
		{
			WebFeatureID: idMap["feature2"],
			FeatureDiscouragedDetails: FeatureDiscouragedDetails{
				AccordingTo:  []string{"source2"},
				Alternatives: nil,
			},
		},
	}
	// Should allow resetting of Alternatives. Should allow setting of AccordingTo
	slices.SortFunc(expected, sortFeatureDiscouragedDetails)

	details, err = spannerClient.readAllFeatureDiscouragedDetails(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all discouraged details err: %s", err)
	}
	slices.SortFunc(details, sortFeatureDiscouragedDetails)

	if !slices.EqualFunc(expected, details, featureDiscouragedDetailsEquality) {
		t.Errorf("unequal discouraged details.\nexpected %+v\nreceived %+v", expected, details)
	}
}
