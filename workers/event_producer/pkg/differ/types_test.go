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

package differ

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

func TestOptionallySet_Marshaling(t *testing.T) {
	type wrapper struct {
		Field OptionallySet[string] `json:"field,omitzero"`
	}

	tests := []struct {
		name     string
		input    wrapper
		expected string
	}{
		{
			name: "IsSet=true",
			input: wrapper{
				Field: OptionallySet[string]{Value: "val", IsSet: true},
			},
			expected: `{"field":"val"}`,
		},
		{
			name: "IsSet=false",
			input: wrapper{
				Field: OptionallySet[string]{Value: "", IsSet: false},
			},
			// With 'omitzero', the zero value of the struct ({Value: "", IsSet: false})
			// is correctly omitted from the JSON output.
			expected: `{}`,
		},
		{
			name: "Pointer IsSet=true, Value=nil",
			input: wrapper{
				Field: OptionallySet[string]{Value: "null", IsSet: true}, // String doesn't support nil, but concept holds.
			},
			expected: `{"field":"null"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			if string(b) != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, string(b))
			}

			// Round trip check for Unmarshal
			var out wrapper
			if err := json.Unmarshal(b, &out); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			// If we expected {}, then IsSet should remain false
			if tc.expected == `{}` {
				if out.Field.IsSet {
					t.Error("Expected IsSet to be false (missing field)")
				}
			} else {
				if !out.Field.IsSet {
					t.Error("Expected IsSet to be true after Unmarshal")
				}
				if out.Field.Value != tc.input.Field.Value {
					t.Errorf("Value mismatch. Got %q, want %q", out.Field.Value, tc.input.Field.Value)
				}
			}
		})
	}
}

func TestOptionallySet_UnmarshalMissing(t *testing.T) {
	type wrapper struct {
		Field OptionallySet[string] `json:"field,omitzero"`
	}

	// Case: Field is missing entirely from JSON
	jsonStr := `{}`
	var out wrapper
	if err := json.Unmarshal([]byte(jsonStr), &out); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if out.Field.IsSet {
		t.Error("Expected IsSet=false for missing field")
	}
}

func TestHasChanges(t *testing.T) {
	tests := []struct {
		name     string
		diff     FeatureDiffV1
		expected bool
	}{
		{
			name: "No Changes",
			diff: FeatureDiffV1{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			expected: false,
		},
		{
			name: "Query Changed",
			diff: FeatureDiffV1{
				QueryChanged: true,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			expected: true,
		},
		{
			name: "Added",
			diff: FeatureDiffV1{
				QueryChanged: false,
				Added:        []FeatureDiffV1FeatureAdded{{ID: "1", Name: "A", Reason: ReasonNewMatch, Docs: nil}},
				Removed:      nil,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			expected: true,
		},
		{
			name: "Removed",
			diff: FeatureDiffV1{
				QueryChanged: false,
				Added:        nil,
				Removed:      []FeatureDiffV1FeatureRemoved{{ID: "1", Name: "A", Reason: ReasonUnmatched}},
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			expected: true,
		},
		{
			name: "Modified",
			diff: FeatureDiffV1{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified: []FeatureDiffV1FeatureModified{{
					ID:         "1",
					Name:       "A",
					Docs:       nil,
					NameChange: nil,
					BaselineChange: &Change[BaselineState]{
						From: BaselineState{
							Status:   OptionallySet[backend.BaselineInfoStatus]{Value: "a", IsSet: true},
							LowDate:  OptionallySet[*time.Time]{Value: nil, IsSet: false},
							HighDate: OptionallySet[*time.Time]{Value: nil, IsSet: false},
						},
						To: BaselineState{
							Status:   OptionallySet[backend.BaselineInfoStatus]{Value: "b", IsSet: true},
							LowDate:  OptionallySet[*time.Time]{Value: nil, IsSet: false},
							HighDate: OptionallySet[*time.Time]{Value: nil, IsSet: false},
						}},
					BrowserChanges: nil,
					DocsChange:     nil,
				}},
				Moves:  nil,
				Splits: nil,
			},
			expected: true,
		},
		{
			name: "Moves",
			diff: FeatureDiffV1{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves:        []FeatureDiffV1FeatureMoved{{FromID: "A", ToID: "B", FromName: "A", ToName: "B"}},
				Splits:       nil,
			},
			expected: true,
		},
		{
			name: "Splits",
			diff: FeatureDiffV1{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves:        nil,
				Splits:       []FeatureDiffV1FeatureSplit{{FromID: "A", FromName: "A", To: nil}},
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

func testBaselineChange() *Change[BaselineState] {
	return &Change[BaselineState]{
		From: BaselineState{
			Status:   OptionallySet[backend.BaselineInfoStatus]{Value: "limited", IsSet: true},
			LowDate:  OptionallySet[*time.Time]{Value: nil, IsSet: false},
			HighDate: OptionallySet[*time.Time]{Value: nil, IsSet: false},
		},
		To: BaselineState{
			Status:   OptionallySet[backend.BaselineInfoStatus]{Value: "newly", IsSet: true},
			LowDate:  OptionallySet[*time.Time]{Value: nil, IsSet: false},
			HighDate: OptionallySet[*time.Time]{Value: nil, IsSet: false},
		},
	}
}

func TestSummarize(t *testing.T) {
	tests := []struct {
		name         string
		diff         FeatureDiffV1
		expectedText string
		expectedCats workertypes.SummaryCategories
	}{
		{
			name: "Empty",
			diff: FeatureDiffV1{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			expectedText: "No changes detected",
			expectedCats: workertypes.SummaryCategories{
				QueryChanged:    0,
				Added:           0,
				Removed:         0,
				Moved:           0,
				Split:           0,
				Updated:         0,
				UpdatedImpl:     0,
				UpdatedRename:   0,
				UpdatedBaseline: 0,
			},
		},
		{
			name: "Only Baseline Change",
			diff: FeatureDiffV1{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified: []FeatureDiffV1FeatureModified{
					{
						ID:             "1",
						Name:           "Feature A",
						Docs:           nil,
						NameChange:     nil,
						BaselineChange: testBaselineChange(),
						BrowserChanges: nil,
						DocsChange:     nil,
					},
				},
				Moves:  nil,
				Splits: nil,
			},
			expectedText: "1 features updated",
			expectedCats: workertypes.SummaryCategories{
				QueryChanged:    0,
				Added:           0,
				Removed:         0,
				Moved:           0,
				Split:           0,
				Updated:         1,
				UpdatedImpl:     0,
				UpdatedRename:   0,
				UpdatedBaseline: 1,
			},
		},
		{
			name: "Only Name Change",
			diff: FeatureDiffV1{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified: []FeatureDiffV1FeatureModified{
					{
						ID:             "1",
						Name:           "Feature A",
						Docs:           nil,
						NameChange:     &Change[string]{From: "Old", To: "New"},
						BaselineChange: nil,
						BrowserChanges: nil,
						DocsChange:     nil,
					},
				},
				Moves:  nil,
				Splits: nil,
			},
			expectedText: "1 features updated",
			expectedCats: workertypes.SummaryCategories{
				QueryChanged:    0,
				Added:           0,
				Removed:         0,
				Moved:           0,
				Split:           0,
				Updated:         1,
				UpdatedImpl:     0,
				UpdatedRename:   1,
				UpdatedBaseline: 0,
			},
		},
		{
			name: "Only Browser Change",
			diff: FeatureDiffV1{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified: []FeatureDiffV1FeatureModified{
					{
						ID:             "1",
						Name:           "Feature A",
						Docs:           nil,
						NameChange:     nil,
						BaselineChange: nil,
						BrowserChanges: map[backend.SupportedBrowsers]*Change[BrowserState]{
							backend.Chrome: {
								From: BrowserState{
									Status: OptionallySet[backend.BrowserImplementationStatus]{
										Value: "unavailable",
										IsSet: true,
									},
									Version: OptionallySet[*string]{
										Value: nil,
										IsSet: false,
									},
									Date: OptionallySet[*time.Time]{
										Value: nil,
										IsSet: false,
									},
								},
								To: BrowserState{
									Status: OptionallySet[backend.BrowserImplementationStatus]{
										Value: "available",
										IsSet: false,
									},
									Version: OptionallySet[*string]{
										Value: nil,
										IsSet: false,
									},
									Date: OptionallySet[*time.Time]{
										Value: nil,
										IsSet: false,
									},
								},
							},
							backend.ChromeAndroid:  nil,
							backend.Firefox:        nil,
							backend.FirefoxAndroid: nil,
							backend.Edge:           nil,
							backend.Safari:         nil,
							backend.SafariIos:      nil,
						},
						DocsChange: nil,
					},
				},
				Moves:  nil,
				Splits: nil,
			},
			expectedText: "1 features updated",
			expectedCats: workertypes.SummaryCategories{
				QueryChanged:    0,
				Added:           0,
				Removed:         0,
				Moved:           0,
				Split:           0,
				Updated:         1,
				UpdatedImpl:     1,
				UpdatedRename:   0,
				UpdatedBaseline: 0,
			},
		},
		{
			name: "Complex Update",
			diff: FeatureDiffV1{
				QueryChanged: true,
				Added: []FeatureDiffV1FeatureAdded{
					{ID: "1", Name: "A", Reason: ReasonNewMatch, Docs: nil},
					{ID: "2", Name: "B", Reason: ReasonNewMatch, Docs: nil},
				},
				Removed: []FeatureDiffV1FeatureRemoved{
					{ID: "3", Name: "C", Reason: ReasonUnmatched},
				},
				Moves: []FeatureDiffV1FeatureMoved{
					{FromID: "4", ToID: "5", FromName: "D", ToName: "E"},
				},
				Splits: []FeatureDiffV1FeatureSplit{
					{FromID: "6", FromName: "F", To: []FeatureDiffV1FeatureAdded{
						{ID: "7", Name: "G", Reason: ReasonNewMatch, Docs: nil}}},
				},
				Modified: []FeatureDiffV1FeatureModified{
					{
						ID:             "8",
						Name:           "H",
						Docs:           nil,
						NameChange:     nil,
						BaselineChange: testBaselineChange(),
						BrowserChanges: nil,
						DocsChange:     nil,
					},
					{
						ID:             "10",
						Name:           "",
						Docs:           nil,
						NameChange:     &Change[string]{From: "Old", To: "New"},
						BaselineChange: nil,
						BrowserChanges: nil,
						DocsChange:     nil,
					},
					{
						ID:             "12",
						Name:           "I",
						Docs:           nil,
						NameChange:     nil,
						BaselineChange: nil,
						BrowserChanges: map[backend.SupportedBrowsers]*Change[BrowserState]{
							backend.Chrome: {
								From: BrowserState{
									Status: OptionallySet[backend.BrowserImplementationStatus]{
										Value: "unavailable",
										IsSet: true,
									},
									Version: OptionallySet[*string]{
										Value: nil,
										IsSet: false,
									},
									Date: OptionallySet[*time.Time]{
										Value: nil,
										IsSet: false,
									},
								},
								To: BrowserState{
									Status: OptionallySet[backend.BrowserImplementationStatus]{
										Value: "available",
										IsSet: false,
									},
									Version: OptionallySet[*string]{
										Value: nil,
										IsSet: false,
									},
									Date: OptionallySet[*time.Time]{
										Value: nil,
										IsSet: false,
									},
								},
							},
							backend.ChromeAndroid:  nil,
							backend.Firefox:        nil,
							backend.FirefoxAndroid: nil,
							backend.Edge:           nil,
							backend.Safari:         nil,
							backend.SafariIos:      nil,
						},
						DocsChange: nil,
					},
				}},
			// Note: The text output order depends on the implementation of Summarize.
			// "Search criteria updated, 2 features added, 1 features removed, 1 features moved/renamed,
			// 1 features split, 3 features updated"
			expectedText: "Search criteria updated, 2 features added, 1 features removed, 1 features moved/renamed, " +
				"1 features split, 3 features updated",
			expectedCats: workertypes.SummaryCategories{
				QueryChanged:    1,
				Added:           2,
				Removed:         1,
				Moved:           1,
				Split:           1,
				Updated:         3,
				UpdatedImpl:     1,
				UpdatedRename:   1,
				UpdatedBaseline: 1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.diff.Summarize()

			if s.Text != tc.expectedText {
				t.Errorf("Text mismatch.\nGot:  %q\nWant: %q", s.Text, tc.expectedText)
			}
			if !reflect.DeepEqual(s.Categories, tc.expectedCats) {
				t.Errorf("Categories mismatch.\nGot:  %+v\nWant: %+v", s.Categories, tc.expectedCats)
			}
		})
	}
}

func TestFeatureDiff_Sort(t *testing.T) {
	diff := FeatureDiffV1{
		QueryChanged: false,
		Added: []FeatureDiffV1FeatureAdded{
			{ID: "2", Name: "B", Reason: ReasonNewMatch, Docs: nil},
			{ID: "1", Name: "A", Reason: ReasonNewMatch, Docs: nil},
			{ID: "3", Name: "A", Reason: ReasonNewMatch, Docs: nil}, // Same Name, Diff ID
		},
		Removed: []FeatureDiffV1FeatureRemoved{
			{ID: "2", Name: "B", Reason: ReasonUnmatched},
			{ID: "1", Name: "A", Reason: ReasonUnmatched},
		},
		Modified: []FeatureDiffV1FeatureModified{
			{ID: "2", Name: "B", NameChange: nil, BaselineChange: nil, BrowserChanges: nil, Docs: nil, DocsChange: nil},
			{ID: "1", Name: "A", NameChange: nil, BaselineChange: nil, BrowserChanges: nil, Docs: nil, DocsChange: nil},
		},
		Moves: []FeatureDiffV1FeatureMoved{
			{FromID: "2", FromName: "B", ToID: "20", ToName: ""},
			{FromID: "1", FromName: "A", ToID: "10", ToName: ""},
		},
		Splits: []FeatureDiffV1FeatureSplit{
			{
				FromID:   "2",
				FromName: "B",
				To: []FeatureDiffV1FeatureAdded{
					{ID: "20", Name: "Y", Reason: ReasonNewMatch, Docs: nil},
					{ID: "10", Name: "X", Reason: ReasonNewMatch, Docs: nil},
				},
			},
			{
				FromID:   "1",
				FromName: "A",
				To:       nil,
			},
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
