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

package workertypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// assertingVisitor used for TestParseEventSummary to verify visitor calls.
type assertingVisitor struct {
	t           *testing.T
	wantSummary EventSummary
	visitedV1   bool
}

func (v *assertingVisitor) VisitV1(got EventSummary) error {
	v.visitedV1 = true
	if diff := cmp.Diff(v.wantSummary, got); diff != "" {
		v.t.Errorf("VisitV1 argument mismatch (-want +got):\n%s", diff)
	}

	return nil
}

func TestParseEventSummary(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantVisit   bool
		wantSummary *EventSummary
		wantErr     bool
	}{
		{
			name:      "Explicit V1",
			input:     `{"schemaVersion": "v1", "text": "Hello"}`,
			wantVisit: true,
			wantSummary: &EventSummary{
				SchemaVersion: "v1",
				Text:          "Hello",
				Categories: SummaryCategories{
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
			wantErr: false,
		},
		{
			name:        "Unknown Version",
			input:       `{"schemaVersion": "v99", "text": "Future"}`,
			wantSummary: nil,
			wantVisit:   false,
			wantErr:     true,
		},
		{
			name:        "Invalid JSON",
			input:       `{broken`,
			wantSummary: nil,
			wantVisit:   false,
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var summary EventSummary
			if tc.wantSummary != nil {
				summary = *tc.wantSummary
			}
			v := &assertingVisitor{
				t:           t,
				visitedV1:   false,
				wantSummary: summary,
			}
			err := ParseEventSummary([]byte(tc.input), v)

			if tc.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if v.visitedV1 != tc.wantVisit {
				t.Errorf("VisistedV1 = %v, want %v", v.visitedV1, tc.wantVisit)
			}
		})
	}
}
