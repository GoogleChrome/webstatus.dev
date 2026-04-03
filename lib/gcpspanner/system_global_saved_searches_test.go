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
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/api/iterator"
)

func TestListSystemGlobalSavedSearches(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	// Expected lists defined to catch anyone modifying global structure via migrations without tests
	expectedListed := []struct {
		ID                 string
		HasCustomSortOrder bool
	}{
		{"baseline-2026", false},
		{"baseline-2025", false},
		{"baseline-2024", false},
		{"baseline-2023", false},
		{"baseline-2022", false},
		{"baseline-2021", false},
		{"baseline-2020", false},
		{"top-css-interop", true},
		{"top-html-interop", true},
	}

	expectedUnlisted := []string{
		"all",
	}

	// 1. Get explicitly LISTED searches via the official func and assert precise order and properties
	page, total, _, err := spannerClient.ListSystemGlobalSavedSearches(ctx, 100, nil)
	if err != nil {
		t.Fatalf("ListSystemGlobalSavedSearches error: %v", err)
	}

	if total != int64(len(expectedListed)) {
		t.Errorf("expected total count %d, got %d", len(expectedListed), total)
	}

	if len(page) != len(expectedListed) {
		t.Fatalf("expected %d page items, got %d", len(expectedListed), len(page))
	}

	for i, s := range page {
		expected := expectedListed[i]
		if s.ID != expected.ID {
			t.Errorf("at index %d: expected ID %s, got %s", i, expected.ID, s.ID)
		}
		res, err := spannerClient.GetSystemGlobalSavedSearch(ctx, s.ID)
		if err != nil {
			t.Errorf("at index %d: GetSystemGlobalSavedSearch error for %s: %v", i, s.ID, err)
		} else if res.HasCustomSortOrder != expected.HasCustomSortOrder {
			t.Errorf("at index %d (%s): expected HasCustomSortOrder %v, got %v",
				i, s.ID, expected.HasCustomSortOrder, res.HasCustomSortOrder)
		}
	}
	if total != int64(len(expectedListed)) {
		t.Errorf("expected total count %d, got %d", len(expectedListed), total)
	}

	// 2. Fetch UNLISTED separately directly via spanner to assert they correctly exist
	stmt := spanner.NewStatement("SELECT SavedSearchID FROM SystemGlobalSavedSearches WHERE Status = 'UNLISTED'")
	iter := spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	var actualUnlisted []string
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			t.Fatalf("failed iterating unlisted searches: %v", err)
		}
		var id string
		if err := row.Columns(&id); err != nil {
			t.Fatalf("failed scanning row: %v", err)
		}
		actualUnlisted = append(actualUnlisted, id)
	}

	sortOpt := cmpopts.SortSlices(func(a, b string) bool { return a < b })
	if diff := cmp.Diff(expectedUnlisted, actualUnlisted, sortOpt); diff != "" {
		t.Fatalf("Unlisted System Global Searches mismatch (-want +got):\n%s", diff)
	}

	// 3. Verify no other unknown status types exist
	stmtCnt := spanner.NewStatement(
		"SELECT count(*) FROM SystemGlobalSavedSearches WHERE Status NOT IN ('LISTED', 'UNLISTED')")
	iterCnt := spannerClient.Single().Query(ctx, stmtCnt)
	defer iterCnt.Stop()
	row, err := iterCnt.Next()
	if err != nil {
		t.Fatalf("failed iterating unknown status count: %v", err)
	}
	var unknownCount int64
	if err := row.Columns(&unknownCount); err != nil {
		t.Fatalf("failed scanning unknown status count: %v", err)
	}
	if unknownCount > 0 {
		t.Fatalf(
			"found %d SystemGlobalSavedSearches with unknown statuses (expected only LISTED or UNLISTED)",
			unknownCount,
		)
	}
}

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
