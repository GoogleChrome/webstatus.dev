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
	"encoding/json"
	"errors"
	"testing"
	"time"

	v1 "github.com/GoogleChrome/webstatus.dev/lib/blobtypes/featurelistdiff/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/generic"
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

func TestGenerateJSONSummaryFeatureDiffV1(t *testing.T) {
	tests := []struct {
		name          string
		diff          v1.FeatureDiff
		expected      string
		expectedError error
	}{
		{
			name: "Empty",
			diff: v1.FeatureDiff{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			expected:      `{"schemaVersion":"v1","text":"No changes detected"}`,
			expectedError: nil,
		},
		{
			name: "Complex Update",
			diff: v1.FeatureDiff{
				QueryChanged: true,
				Added: []v1.FeatureAdded{
					{ID: "1", Name: "A", Reason: v1.ReasonNewMatch, Docs: nil},
					{ID: "2", Name: "B", Reason: v1.ReasonNewMatch, Docs: nil},
				},
				Removed: []v1.FeatureRemoved{
					{ID: "3", Name: "C", Reason: v1.ReasonUnmatched},
				},
				Moves: []v1.FeatureMoved{
					{FromID: "4", ToID: "5", FromName: "D", ToName: "E"},
				},
				Splits: []v1.FeatureSplit{
					{FromID: "6", FromName: "F", To: []v1.FeatureAdded{{ID: "7", Name: "G", Reason: v1.ReasonNewMatch, Docs: nil}}},
				},
				Modified: []v1.FeatureModified{
					{
						ID:         "8",
						Name:       "H",
						NameChange: nil,
						BaselineChange: &v1.Change[v1.BaselineState]{
							From: v1.BaselineState{
								Status:   generic.SetOpt(v1.Limited),
								LowDate:  generic.UnsetOpt[*time.Time](),
								HighDate: generic.UnsetOpt[*time.Time](),
							},
							To: v1.BaselineState{
								Status:   generic.SetOpt(v1.Newly),
								LowDate:  generic.UnsetOpt[*time.Time](),
								HighDate: generic.UnsetOpt[*time.Time](),
							},
						},
						BrowserChanges: nil,
						Docs:           nil,
						DocsChange:     nil,
					},
					{
						ID:             "9",
						Name:           "I",
						NameChange:     nil,
						BaselineChange: nil,
						BrowserChanges: map[v1.SupportedBrowsers]*v1.Change[v1.BrowserState]{
							v1.Chrome: {From: v1.BrowserState{
								Status:  generic.SetOpt(v1.Unavailable),
								Date:    generic.UnsetOpt[*time.Time](),
								Version: generic.UnsetOpt[*string](),
							}, To: v1.BrowserState{
								Status:  generic.SetOpt(v1.Available),
								Date:    generic.UnsetOpt[*time.Time](),
								Version: generic.UnsetOpt[*string](),
							}},
							v1.ChromeAndroid:  nil,
							v1.Edge:           nil,
							v1.Firefox:        nil,
							v1.FirefoxAndroid: nil,
							v1.Safari:         nil,
							v1.SafariIos:      nil,
						},
						Docs:       nil,
						DocsChange: nil,
					},
					{
						ID:             "10",
						Name:           "J",
						NameChange:     &v1.Change[string]{From: "Old", To: "New"},
						BaselineChange: nil,
						BrowserChanges: nil,
						Docs:           nil,
						DocsChange:     nil,
					},
				},
			},

			expected: `{
"schemaVersion":"v1",
"text":"Search criteria updated, 2 features added, 1 features removed, ` +
				`1 features moved/renamed, 1 features split, 3 features updated",
"categories":
	{
		"query_changed":1,
		"added":2,
		"removed":1,
		"moved":1,
		"split":1,
		"updated":3,
		"updated_impl":1,
		"updated_rename":1,
		"updated_baseline":1
	}
}`,
			expectedError: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := &FeatureDiffV1SummaryGenerator{}
			got, err := g.GenerateJSONSummary(tc.diff)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("GenerateJSONSummary() error = %v, wantErr %v", err, tc.expectedError)

				return
			}
			if err != nil {
				return
			}

			compareJSONBodies(t, got, []byte(tc.expected))
		})
	}
}

func compareJSONBodies(t *testing.T, actualBody, expectedBody []byte) {
	t.Helper()
	var actualObj, expectedObj interface{}
	err := json.Unmarshal(actualBody, &actualObj)
	if err != nil {
		t.Fatal("failed to parse json from actual response")
	}
	err = json.Unmarshal(expectedBody, &expectedObj)
	if err != nil {
		t.Fatal("failed to parse json from expected response")
	}

	if diff := cmp.Diff(expectedObj, actualObj); diff != "" {
		t.Errorf("JSON mismatch (-want +got):\n%s", diff)
	}
}
