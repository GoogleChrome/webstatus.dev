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
	"errors"
	"slices"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func TestUpsertBrowserCompatFeatures(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	feature := getSampleFeatures()[0]
	featureID, err := spannerClient.UpsertWebFeature(ctx, feature)
	if err != nil {
		t.Fatalf("failed to insert feature: %v", err)
	}

	initial := []string{"html.elements.address", "html.elements.section"}
	err = spannerClient.UpsertBrowserCompatFeatures(ctx, *featureID, initial)
	if err != nil {
		t.Fatalf("UpsertBrowserCompatFeatures initial insert failed: %v", err)
	}

	expected := slices.Clone(initial)
	details := readAllBrowserCompatFeatures(ctx, t, *featureID)
	slices.Sort(details)
	slices.Sort(expected)
	if !slices.Equal(details, expected) {
		t.Errorf("initial compat features mismatch.\nexpected %+v\nreceived %+v", expected, details)
	}

	updated := []string{"html.elements.article"}
	err = spannerClient.UpsertBrowserCompatFeatures(ctx, *featureID, updated)
	if err != nil {
		t.Fatalf("UpsertBrowserCompatFeatures update failed: %v", err)
	}

	expected = slices.Clone(updated)
	details = readAllBrowserCompatFeatures(ctx, t, *featureID)
	slices.Sort(details)
	slices.Sort(expected)
	if !slices.Equal(details, expected) {
		t.Errorf("updated compat features mismatch.\nexpected %+v\nreceived %+v", expected, details)
	}
}

func readAllBrowserCompatFeatures(ctx context.Context, t *testing.T, featureID string) []string {
	stmt := spanner.NewStatement(`
		SELECT CompatFeature
		FROM WebFeatureBrowserCompatFeatures
		WHERE ID = @id`)
	stmt.Params["id"] = featureID

	iter := spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	var features []string
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}
		var compat string
		if err := row.Columns(&compat); err != nil {
			t.Fatalf("column parse failed: %v", err)
		}
		features = append(features, compat)
	}

	return features
}
