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

package v1

import (
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/generic"
)

func TestHasChanges(t *testing.T) {
	tests := []struct {
		name     string
		diff     FeatureDiff
		expected bool
	}{
		{
			name: "No Changes",
			diff: FeatureDiff{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			expected: false,
		},
		{
			name: "Query Changed",
			diff: FeatureDiff{
				QueryChanged: true,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			expected: true,
		},
		{
			name: "Added",
			diff: FeatureDiff{
				QueryChanged: false,
				Added: []FeatureAdded{{ID: "1", Name: "A", Reason: ReasonNewMatch, Docs: nil,
					QueryMatch: QueryMatchMatch}},
				Removed:  nil,
				Modified: nil,
				Moves:    nil,
				Splits:   nil,
				Deleted:  nil,
			},
			expected: true,
		},
		{
			name: "Removed",
			diff: FeatureDiff{
				QueryChanged: false,
				Added:        nil,
				Removed:      []FeatureRemoved{{ID: "1", Name: "A", Reason: ReasonUnmatched, Diff: nil}},
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			expected: true,
		},
		{
			name: "Deleted",
			diff: FeatureDiff{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted: []FeatureDeleted{
					{ID: "1", Name: "A", Reason: ReasonDeleted},
				},
			},
			expected: true,
		},
		{
			name: "Modified",
			diff: FeatureDiff{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified: []FeatureModified{{
					ID:         "1",
					Name:       "A",
					Docs:       nil,
					NameChange: nil,
					BaselineChange: &Change[BaselineState]{
						From: BaselineState{
							Status:   generic.SetOpt(Limited),
							LowDate:  generic.UnsetOpt[*time.Time](),
							HighDate: generic.UnsetOpt[*time.Time](),
						},
						To: BaselineState{
							Status:   generic.SetOpt(Widely),
							LowDate:  generic.UnsetOpt[*time.Time](),
							HighDate: generic.UnsetOpt[*time.Time](),
						},
					},
					BrowserChanges: nil,
					DocsChange:     nil,
				}},
				Moves:   nil,
				Splits:  nil,
				Deleted: nil,
			},
			expected: true,
		},
		{
			name: "Moves",
			diff: FeatureDiff{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves: []FeatureMoved{{FromID: "A", ToID: "B", FromName: "A", ToName: "B",
					QueryMatch: QueryMatchMatch}},
				Splits:  nil,
				Deleted: nil,
			},
			expected: true,
		},
		{
			name: "Splits",
			diff: FeatureDiff{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves:        nil,
				Splits:       []FeatureSplit{{FromID: "A", FromName: "A", To: nil}},
				Deleted:      nil,
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.diff.HasChanges(); got != tc.expected {
				t.Errorf("HasChanges() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestFeatureDiff_Sort(t *testing.T) {
	diff := FeatureDiff{
		QueryChanged: false,
		Added: []FeatureAdded{
			{ID: "2", Name: "B", Reason: ReasonNewMatch, Docs: nil, QueryMatch: QueryMatchMatch},
			{ID: "1", Name: "A", Reason: ReasonNewMatch, Docs: nil, QueryMatch: QueryMatchMatch},
			{ID: "3", Name: "A", Reason: ReasonNewMatch, Docs: nil, QueryMatch: QueryMatchMatch}, // Same Name, Diff ID
		},
		Removed: []FeatureRemoved{
			{ID: "2", Name: "B", Reason: ReasonUnmatched, Diff: nil},
			{ID: "1", Name: "A", Reason: ReasonUnmatched, Diff: nil},
		},
		Modified: []FeatureModified{
			{ID: "2", Name: "B", Docs: nil, NameChange: nil, BaselineChange: nil, BrowserChanges: nil, DocsChange: nil},
			{ID: "1", Name: "A", Docs: nil, NameChange: nil, BaselineChange: nil, BrowserChanges: nil, DocsChange: nil},
		},
		Moves: []FeatureMoved{
			{FromID: "2", FromName: "B", ToID: "20", ToName: "", QueryMatch: QueryMatchMatch},
			{FromID: "1", FromName: "A", ToID: "10", ToName: "", QueryMatch: QueryMatchMatch},
		},
		Splits: []FeatureSplit{
			{
				FromID:   "2",
				FromName: "B",
				To: []FeatureAdded{
					{ID: "20", Name: "Y", Reason: ReasonNewMatch, Docs: nil, QueryMatch: QueryMatchMatch},
					{ID: "10", Name: "X", Reason: ReasonNewMatch, Docs: nil, QueryMatch: QueryMatchMatch},
				},
			},
			{
				FromID:   "1",
				FromName: "A",
				To:       nil,
			},
		},
		Deleted: []FeatureDeleted{
			{ID: "2", Name: "B", Reason: ReasonDeleted},
			{ID: "1", Name: "A", Reason: ReasonDeleted},
		},
	}

	diff.Sort()

	// Added: A(1), A(3), B(2)
	if diff.Added[0].ID != "1" || diff.Added[1].ID != "3" || diff.Added[2].ID != "2" {
		t.Errorf("Added sort failed: %+v", diff.Added)
	}

	// Removed: A(1), B(2)
	if diff.Removed[0].ID != "1" || diff.Removed[1].ID != "2" {
		t.Errorf("Removed sort failed: %+v", diff.Removed)
	}

	// Deleted: A(1), B(2)
	if diff.Deleted[0].ID != "1" || diff.Deleted[1].ID != "2" {
		t.Errorf("Deleted sort failed: %+v", diff.Deleted)
	}

	// Modified: A(1), B(2)
	if diff.Modified[0].ID != "1" || diff.Modified[1].ID != "2" {
		t.Errorf("Modified sort failed: %+v", diff.Modified)
	}

	// Moves: A(1), B(2)
	if diff.Moves[0].FromID != "1" || diff.Moves[1].FromID != "2" {
		t.Errorf("Moves sort failed: %+v", diff.Moves)
	}

	// Splits: A(1), B(2)
	if diff.Splits[0].FromID != "1" || diff.Splits[1].FromID != "2" {
		t.Errorf("Splits sort failed: %+v", diff.Splits)
	}

	// Check Nested Split Sort: B(2) -> [X(10), Y(20)]
	// Originally B had [Y, X], should be sorted to [X, Y] by Name
	to := diff.Splits[1].To
	if to[0].Name != "X" || to[1].Name != "Y" {
		t.Errorf("Splits[1].To sort failed: %+v", to)
	}
}
