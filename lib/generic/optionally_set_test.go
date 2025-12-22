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

package generic

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestOptionallySet_Scalars(t *testing.T) {
	type wrapper struct {
		Str    OptionallySet[string]  `json:"str,omitzero"`
		PtrStr OptionallySet[*string] `json:"ptr_str,omitzero"`
		Int    OptionallySet[int]     `json:"int,omitzero"`
	}

	tests := []struct {
		name         string
		input        wrapper
		expectedJSON string
		wantRestored *wrapper
	}{
		{
			name: "All Set",
			input: wrapper{
				Str:    OptionallySet[string]{Value: "A", IsSet: true},
				PtrStr: OptionallySet[*string]{Value: ValuePtr("B"), IsSet: true},
				Int:    OptionallySet[int]{Value: 1, IsSet: true},
			},
			expectedJSON: `{"str":"A","ptr_str":"B","int":1}`,
		},
		{
			name: "All Unset (Omitted)",
			input: wrapper{
				Str:    UnsetOpt[string](),
				PtrStr: UnsetOpt[*string](),
				Int:    UnsetOpt[int](),
			},
			expectedJSON: `{}`,
		},
		{
			name: "Explicit Nulls (IsSet=true, Value=nil/zero)",
			input: wrapper{
				Str:    OptionallySet[string]{Value: "", IsSet: true},
				PtrStr: OptionallySet[*string]{Value: nil, IsSet: true},
				Int:    OptionallySet[int]{Value: 0, IsSet: true},
			},
			// Note: "str" and "int" serialize to their zero values ("" and 0),
			// but ptr_str serializes to null.
			expectedJSON: `{"str":"","ptr_str":null,"int":0}`,
		},
	}

	runTestCases(t, tests)
}

func TestOptionallySet_Complex(t *testing.T) {
	type inner struct {
		ID int `json:"id"`
	}

	type wrapper struct {
		Slice     OptionallySet[[]string]        `json:"slice,omitzero"`
		PtrSlice  OptionallySet[*[]string]       `json:"ptr_slice,omitzero"`
		Map       OptionallySet[map[string]int]  `json:"map,omitzero"`
		PtrMap    OptionallySet[*map[string]int] `json:"ptr_map,omitzero"`
		Struct    OptionallySet[inner]           `json:"struct,omitzero"`
		PtrStruct OptionallySet[*inner]          `json:"ptr_struct,omitzero"`
	}

	tests := []struct {
		name         string
		input        wrapper
		expectedJSON string
		wantRestored *wrapper
	}{
		{
			name: "Complex Types Set",
			input: wrapper{
				Slice:     OptionallySet[[]string]{Value: []string{"a", "b"}, IsSet: true},
				PtrSlice:  OptionallySet[*[]string]{Value: ValuePtr([]string{"c"}), IsSet: true},
				Map:       OptionallySet[map[string]int]{Value: map[string]int{"k": 1}, IsSet: true},
				PtrMap:    OptionallySet[*map[string]int]{Value: ValuePtr(map[string]int{"k": 2}), IsSet: true},
				Struct:    OptionallySet[inner]{Value: inner{ID: 10}, IsSet: true},
				PtrStruct: OptionallySet[*inner]{Value: &inner{ID: 20}, IsSet: true},
			},
			expectedJSON: `{"slice":["a","b"],"ptr_slice":["c"],"map":{"k":1},"ptr_map":{"k":2},"struct":{"id":10},"ptr_struct":{"id":20}}`,
		},
		{
			name: "Complex Types Unset (Omitted)",
			input: wrapper{
				Slice:     UnsetOpt[[]string](),
				PtrSlice:  UnsetOpt[*[]string](),
				Map:       UnsetOpt[map[string]int](),
				PtrMap:    UnsetOpt[*map[string]int](),
				Struct:    UnsetOpt[inner](),
				PtrStruct: UnsetOpt[*inner](),
			},
			expectedJSON: `{}`,
		},
		{
			name: "Explicit Empty/Zero Values (IsSet=true)",
			input: wrapper{
				// Empty slice (not nil)
				Slice: OptionallySet[[]string]{Value: []string{}, IsSet: true},
				// Empty map (not nil)
				Map: OptionallySet[map[string]int]{Value: map[string]int{}, IsSet: true},
				// Zero struct
				Struct: OptionallySet[inner]{Value: inner{}, IsSet: true},
			},
			expectedJSON: `{"slice":[],"map":{},"struct":{"id":0}}`,
			wantRestored: &wrapper{
				// JSON unmarshals `[]` to `[]string{}` (non-nil)
				Slice: OptionallySet[[]string]{Value: []string{}, IsSet: true},
				// JSON unmarshals `{}` to `map[string]int{}` (non-nil)
				Map:    OptionallySet[map[string]int]{Value: map[string]int{}, IsSet: true},
				Struct: OptionallySet[inner]{Value: inner{}, IsSet: true},
				// Others remain unset
				PtrSlice:  UnsetOpt[*[]string](),
				PtrMap:    UnsetOpt[*map[string]int](),
				PtrStruct: UnsetOpt[*inner](),
			},
		},
	}

	runTestCases(t, tests)
}

func runTestCases[T any](t *testing.T, tests []struct {
	name         string
	input        T
	expectedJSON string
	wantRestored *T
}) {
	t.Helper()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Marshal
			b, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			if string(b) != tc.expectedJSON {
				t.Errorf("\nExpected: %s\nGot:      %s", tc.expectedJSON, string(b))
			}

			// 2. Unmarshal
			var out T
			if err := json.Unmarshal(b, &out); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// 3. Compare
			want := tc.input
			if tc.wantRestored != nil {
				want = *tc.wantRestored
			}

			if !reflect.DeepEqual(out, want) {
				t.Errorf("Round trip mismatch.\nWant: %+v\nGot:  %+v", want, out)
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
