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
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestFeaturesSearch_HotlistSubquery_AllBuilders(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	// 1. Insert mock features
	insertMockFeatures(ctx, t, map[string]bool{
		"feat-sub1": true,
		"feat-sub2": true,
		"feat-sub3": true,
	})

	// 2. Insert a Hotlist SavedSearch with EMPTY query
	hotlistID := "hotlist-subquery-test"
	_, err := spannerClient.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		m, err := spanner.InsertStruct(savedSearchesTable, &SavedSearch{
			ID:          hotlistID,
			Name:        "Hotlist Subquery Test",
			Description: nil,
			Query:       "", // Empty query triggers subquery path
			Scope:       SystemGlobalScope,
			AuthorID:    "system",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		if err != nil {
			return err
		}

		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		t.Fatalf("failed to insert hotlist saved search: %v", err)
	}

	for i, featID := range []string{"feat-sub1", "feat-sub2"} {
		m := spanner.Insert("SavedSearchFeatureSortOrder",
			[]string{"SavedSearchID", "FeatureKey", "PositionIndex"},
			[]any{hotlistID, featID, int64((i + 1) * 10)},
		)
		_, err = spannerClient.Apply(ctx, []*spanner.Mutation{m})
		if err != nil {
			t.Fatalf("failed to insert saved search feature sort order: %v", err)
		}
	}

	builders := []FeatureSearchBaseQuery{
		GCPFeatureSearchBaseQuery{},
		LocalFeatureBaseQuery{},
	}

	parser := searchtypes.FeaturesSearchQueryParser{}
	node, err := parser.Parse("hotlist:" + hotlistID)
	if err != nil {
		t.Fatalf("failed to parse hotlist query: %v", err)
	}

	for _, builder := range builders {
		t.Run(fmt.Sprintf("%T", builder), func(t *testing.T) {
			spannerClient.SetFeatureSearchBaseQuery(builder)

			page, err := spannerClient.FeaturesSearch(
				ctx,
				nil,
				100,
				node,
				NewFeatureNameSort(true),
				WPTSubtestView,
				[]string{},
			)
			if err != nil {
				t.Fatalf("FeaturesSearch error: %v", err)
			}

			actualKeys := make([]string, 0, len(page.Features))
			for _, f := range page.Features {
				actualKeys = append(actualKeys, f.FeatureKey)
			}

			expectedKeys := []string{"feat-sub1", "feat-sub2"}
			sortOpt := cmpopts.SortSlices(func(a, b string) bool { return a < b })

			if diff := cmp.Diff(expectedKeys, actualKeys, sortOpt); diff != "" {
				t.Fatalf("Hotlist subquery results mismatch (-want +got):\n%s", diff)
			}
		})
	}
	// Reset to default
	spannerClient.SetFeatureSearchBaseQuery(GCPFeatureSearchBaseQuery{})
}

func TestFeaturesSearch_HotlistAll(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	// 1. Insert mock features
	insertMockFeatures(ctx, t, map[string]bool{
		"feat-all1": true,
		"feat-all2": true,
		"feat-all3": true,
	})

	parser := searchtypes.FeaturesSearchQueryParser{}
	node, err := parser.Parse("hotlist:all")
	if err != nil {
		t.Fatalf("failed to parse hotlist query: %v", err)
	}

	page, err := spannerClient.FeaturesSearch(
		ctx,
		nil,
		100,
		node,
		NewFeatureNameSort(true),
		WPTSubtestView,
		[]string{},
	)
	if err != nil {
		t.Fatalf("FeaturesSearch error: %v", err)
	}

	actualKeys := make([]string, 0, len(page.Features))
	for _, f := range page.Features {
		actualKeys = append(actualKeys, f.FeatureKey)
	}

	expectedKeys := []string{"feat-all1", "feat-all2", "feat-all3"}
	sortOpt := cmpopts.SortSlices(func(a, b string) bool { return a < b })

	if diff := cmp.Diff(expectedKeys, actualKeys, sortOpt); diff != "" {
		t.Fatalf("Hotlist all results mismatch (-want +got):\n%s", diff)
	}
}
