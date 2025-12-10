package differ

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestOptionallySet_Marshaling(t *testing.T) {
	type wrapper struct {
		Field OptionallySet[string] `json:"field,omitempty"`
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
			// MarshalJSON blindly marshals the value.
			// Since "field" is omitempty and the marshaled result of "" is `""` (which is not "empty" in JSON terms
			// relative to the struct field? No, custom marshaler bypasses omitempty logic in complex ways depending on
			// Go version, but usually it writes the value).
			// Actually, OptionallySet[T] is a struct. Structs are only omitted if they match the zero value AND
			// the field has omitempty.
			// But since we implement Marshaler, it depends on what MarshalJSON returns.
			// If Value is "", json.Marshal returns `""`.
			expected: `{"field":""}`,
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
			if !out.Field.IsSet {
				t.Error("Expected IsSet to be true after Unmarshal")
			}
			if out.Field.Value != tc.input.Field.Value {
				t.Errorf("Value mismatch. Got %q, want %q", out.Field.Value, tc.input.Field.Value)
			}
		})
	}
}

func TestOptionallySet_UnmarshalMissing(t *testing.T) {
	type wrapper struct {
		Field OptionallySet[string] `json:"field,omitempty"`
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
			},
			expected: true,
		},
		{
			name: "Added",
			diff: FeatureDiff{
				QueryChanged: false,
				Added:        []FeatureAdded{{ID: "1", Name: "A", Reason: ReasonNewMatch}},
				Removed:      nil,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			expected: true,
		},
		{
			name: "Removed",
			diff: FeatureDiff{
				QueryChanged: false,
				Added:        nil,
				Removed:      []FeatureRemoved{{ID: "1", Name: "A", Reason: ReasonUnmatched}},
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
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
					ID:             "1",
					Name:           "A",
					NameChange:     nil,
					BaselineChange: &Change[backend.BaselineInfoStatus]{From: "a", To: "b"},
					BrowserChanges: nil,
				}},
				Moves:  nil,
				Splits: nil,
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
				Moves:        []FeatureMoved{{FromID: "A", ToID: "B", FromName: "A", ToName: "B"}},
				Splits:       nil,
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

func TestSummarize(t *testing.T) {
	tests := []struct {
		name         string
		diff         FeatureDiff
		expectedText string
		expectedCats SummaryCategories
	}{
		{
			name: "Empty",
			diff: FeatureDiff{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			expectedText: "No changes detected",
			expectedCats: SummaryCategories{
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
			name: "Complex Update",
			diff: FeatureDiff{
				QueryChanged: true,
				Added: []FeatureAdded{
					{ID: "1", Name: "A", Reason: ReasonNewMatch},
					{ID: "2", Name: "B", Reason: ReasonNewMatch},
				},
				Removed: []FeatureRemoved{
					{ID: "3", Name: "C", Reason: ReasonUnmatched},
				},
				Moves: []FeatureMoved{
					{FromID: "4", ToID: "5", FromName: "D", ToName: "E"},
				},
				Splits: []FeatureSplit{
					{FromID: "6", FromName: "F", To: []FeatureAdded{{ID: "7", Name: "G", Reason: ReasonNewMatch}}},
				},
				Modified: []FeatureModified{
					{
						ID:             "8",
						Name:           "H",
						NameChange:     nil,
						BaselineChange: &Change[backend.BaselineInfoStatus]{From: "limited", To: "newly"},
						BrowserChanges: nil,
					},
					{
						ID:             "9",
						Name:           "I",
						NameChange:     nil,
						BaselineChange: nil,
						BrowserChanges: map[string]Change[string]{
							"chrome": {From: "unavailable", To: "available"},
						},
					},
					{
						ID:             "10",
						Name:           "J",
						NameChange:     &Change[string]{From: "Old", To: "New"},
						BaselineChange: nil,
						BrowserChanges: nil,
					},
				},
			},
			// Note: The text output order depends on the implementation of Summarize.
			// "Search criteria updated, 2 features added, 1 features removed, 1 features moved/renamed,
			// 1 features split, 3 features updated"
			expectedText: "Search criteria updated, 2 features added, 1 features removed, 1 features moved/renamed, " +
				"1 features split, 3 features updated",
			expectedCats: SummaryCategories{
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
