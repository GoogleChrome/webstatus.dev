// Copyright 2026 Google LLC
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
	"fmt"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/iterator"
)

type sortOrderResult struct {
	SavedSearchID string
	FeatureKey    string
	PositionIndex int64
}

func TestFeaturesSearch_SavedSearchSortOrder(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	// Expected exact mapping results. If a PR adds another saved search sort order,
	// this test will explicitly break here until they document the expected order.
	expectedCustomSortOrders := map[string][]string{
		"top-css-interop": {
			"anchor-positioning",
			"scroll-driven-animations",
			"view-transitions",
			"cross-document-view-transitions",
			"container-style-queries",
			"nesting",
			"has",
			"container-queries",
			"scope",
			"if",
			"grid",
		},
		"top-html-interop": {
			"customizable-select",
			"popover",
			"anchor-positioning",
			"customized-built-in-elements",
			"shadow-dom",
			"dialog",
			"view-transitions",
			"cross-document-view-transitions",
			"file-system-access",
			"input-date-time",
			"invoker-commands",
			"webusb",
		},
	}

	sortOrders := getSavedSearchSortOrders(ctx, t)
	uniqueFeatureKeys := getUniqueFeatureKeys(sortOrders)
	insertMockFeatures(ctx, t, uniqueFeatureKeys)

	actualSearchIDs := make(map[string]bool)
	for _, s := range sortOrders {
		actualSearchIDs[s.SavedSearchID] = true
	}

	// Fail the test if the DB migration contains a saved search mapping that the test isn't tracking!
	for actualID := range actualSearchIDs {
		if _, ok := expectedCustomSortOrders[actualID]; !ok {
			t.Fatalf("Integration test failure: found unexpected SavedSearchID '%s'."+
				" Please add its expected order to expectedCustomSortOrders.", actualID)
		}
	}
	for expectedID := range expectedCustomSortOrders {
		if _, ok := actualSearchIDs[expectedID]; !ok {
			t.Fatalf("Integration test failure: expected SavedSearchID '%s'"+
				" is missing from the database migration mapping.", expectedID)
		}
	}

	builders := []FeatureSearchBaseQuery{
		GCPFeatureSearchBaseQuery{},
		LocalFeatureBaseQuery{},
	}

	for _, builder := range builders {
		t.Run(fmt.Sprintf("%T", builder), func(t *testing.T) {
			spannerClient.SetFeatureSearchBaseQuery(builder)
			assertCustomSortOrders(ctx, t, expectedCustomSortOrders)
		})
	}
	// Reset to default
	spannerClient.SetFeatureSearchBaseQuery(GCPFeatureSearchBaseQuery{})
}

func getSavedSearchSortOrders(ctx context.Context, t *testing.T) []sortOrderResult {
	stmt := spanner.NewStatement(
		"SELECT SavedSearchID, FeatureKey, PositionIndex " +
			"FROM SavedSearchFeatureSortOrder ORDER BY SavedSearchID, PositionIndex ASC")
	iter := spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	var sortOrders []sortOrderResult
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			t.Fatalf("failed iterating SavedSearchFeatureSortOrder: %v", err)
		}
		var s sortOrderResult
		if err := row.Columns(&s.SavedSearchID, &s.FeatureKey, &s.PositionIndex); err != nil {
			t.Fatalf("failed scanning row: %v", err)
		}
		sortOrders = append(sortOrders, s)
	}

	return sortOrders
}

func getUniqueFeatureKeys(sortOrders []sortOrderResult) map[string]bool {
	uniqueFeatureKeys := make(map[string]bool)
	for _, s := range sortOrders {
		uniqueFeatureKeys[s.FeatureKey] = true
	}

	return uniqueFeatureKeys
}

func insertMockFeatures(ctx context.Context, t *testing.T, keys map[string]bool) {
	_, err := spannerClient.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		var mutations []*spanner.Mutation
		for key := range keys {
			m, err := spanner.InsertStruct(webFeaturesTable, &SpannerWebFeature{
				ID: "id-" + key,
				WebFeature: WebFeature{
					FeatureKey:      key,
					Name:            "Name for " + key,
					Description:     "",
					DescriptionHTML: "",
				},
			})
			if err != nil {
				return err
			}
			mutations = append(mutations, m)
		}

		return txn.BufferWrite(mutations)
	})
	if err != nil {
		t.Fatalf("failed to insert fake features: %v", err)
	}
}

func assertCustomSortOrders(ctx context.Context, t *testing.T, expected map[string][]string) {
	for searchID, expectedOrderedKeys := range expected {
		sortParam := NewSearchIDOrderSort(true, searchID)

		page, err := spannerClient.FeaturesSearch(
			ctx,
			nil,
			1000,
			nil, // nil search node equals 'all wildcard'
			sortParam,
			WPTSubtestView,
			[]string{})

		if err != nil {
			t.Fatalf("FeaturesSearch error for %s: %v", searchID, err)
		}

		expectedSet := make(map[string]bool)
		for _, k := range expectedOrderedKeys {
			expectedSet[k] = true
		}

		var actualOrderedKeys []string
		for _, f := range page.Features {
			if expectedSet[f.FeatureKey] {
				actualOrderedKeys = append(actualOrderedKeys, f.FeatureKey)
			}
		}

		if diff := cmp.Diff(expectedOrderedKeys, actualOrderedKeys); diff != "" {
			t.Fatalf("Search %s sort order mismatch (-want +got):\n%s", searchID, diff)
		}
	}
}
