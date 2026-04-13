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
	page, _, err := spannerClient.ListSystemGlobalSavedSearches(ctx, 100, nil)
	if err != nil {
		t.Fatalf("ListSystemGlobalSavedSearches error: %v", err)
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
