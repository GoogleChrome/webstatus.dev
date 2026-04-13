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
				SchemaVersion:  "v1",
				SnapshotOrigin: "",
				Text:           "Hello",
				Categories: SummaryCategories{
					QueryChanged:    0,
					Added:           0,
					Removed:         0,
					Deleted:         0,
					Moved:           0,
					Split:           0,
					Updated:         0,
					UpdatedImpl:     0,
					UpdatedRename:   0,
					UpdatedBaseline: 0,
				},
				Truncated:   false,
				Highlights:  nil,
				QueryErrors: nil,
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
	newlyDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	browserImplDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name          string
		diff          v1.FeatureDiff
		expected      string
		expectedError error
	}{
		{
			name: "Empty",
			diff: v1.FeatureDiff{
				SnapshotOrigin: v1.OriginLive,
				QueryChanged:   false,
				Added:          nil,
				Removed:        nil,
				Modified:       nil,
				Moves:          nil,
				Splits:         nil,
				Deleted:        nil,
				QueryErrors:    nil,
			},
			expected: `{"schemaVersion":"v1","snapshotOrigin":"LIVE",` +
				`"text":"No changes detected","truncated":false,"highlights":null}`,
			expectedError: nil,
		},
		{
			name: "Complex Update",
			diff: v1.FeatureDiff{
				SnapshotOrigin: v1.OriginLive,
				QueryChanged:   true,
				Added: []v1.FeatureAdded{
					{ID: "1", Name: "A", Reason: v1.ReasonNewMatch, Docs: nil, QueryMatch: v1.QueryMatchMatch},
					{ID: "2", Name: "B", Reason: v1.ReasonNewMatch, Docs: &v1.Docs{
						MdnDocs: []v1.MdnDoc{{URL: "https://mdn.io/B", Title: new("B"), Slug: new("slug-b")}},
					}, QueryMatch: v1.QueryMatchMatch},
				},
				Removed: []v1.FeatureRemoved{
					{ID: "3", Name: "C", Reason: v1.ReasonUnmatched, Diff: nil},
					{ID: "31", Name: "K", Reason: v1.ReasonUnmatched, Diff: &v1.FeatureModified{
						ID:         "31",
						Name:       "K",
						NameChange: nil,
						Docs:       nil,
						BaselineChange: &v1.Change[v1.BaselineState]{
							From: v1.BaselineState{
								Status:   generic.SetOpt(v1.Limited),
								LowDate:  generic.UnsetOpt[*time.Time](),
								HighDate: generic.UnsetOpt[*time.Time](),
							},
							To: v1.BaselineState{
								Status:   generic.SetOpt(v1.Newly),
								LowDate:  generic.SetOpt(&newlyDate),
								HighDate: generic.UnsetOpt[*time.Time](),
							},
						},
						BrowserChanges: map[v1.SupportedBrowsers]*v1.Change[v1.BrowserState]{
							v1.Chrome: {From: v1.BrowserState{
								Status:  generic.SetOpt(v1.Unavailable),
								Date:    generic.UnsetOpt[*time.Time](),
								Version: generic.UnsetOpt[*string](),
							}, To: v1.BrowserState{
								Status:  generic.SetOpt(v1.Available),
								Date:    generic.SetOpt(&browserImplDate),
								Version: generic.SetOpt(new("132")),
							}},
							v1.ChromeAndroid:  nil,
							v1.Edge:           nil,
							v1.Firefox:        nil,
							v1.FirefoxAndroid: nil,
							v1.Safari:         nil,
							v1.SafariIos:      nil,
						},
						DocsChange: nil,
					}},
				},
				Deleted: []v1.FeatureDeleted{
					{ID: "4", Name: "D", Reason: v1.ReasonDeleted},
				},
				QueryErrors: nil,
				Moves: []v1.FeatureMoved{
					{FromID: "4", ToID: "5", FromName: "D", ToName: "E", QueryMatch: v1.QueryMatchMatch},
				},
				Splits: []v1.FeatureSplit{
					{
						FromID:   "6",
						FromName: "F",
						To: []v1.FeatureAdded{
							{
								ID:         "7",
								Name:       "G",
								Reason:     v1.ReasonNewMatch,
								Docs:       nil,
								QueryMatch: v1.QueryMatchMatch,
							},
						},
					},
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
								LowDate:  generic.SetOpt(&newlyDate),
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
								Date:    generic.SetOpt(&browserImplDate),
								Version: generic.SetOpt(new("123")),
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
    "schemaVersion": "v1",
    "snapshotOrigin": "LIVE",
    "text": "Search criteria updated, 2 new features matched your search, 2 features no longer matched your search ` +
				`(1 became Baseline newly available), 1 feature deleted, 1 feature moved/renamed, 1 feature split, ` +
				`3 features updated (1 became Baseline newly available)",
    "categories": {
        "query_changed": 1,
        "added": 2,
        "removed": 2,
		"deleted": 1,
        "moved": 1,
        "split": 1,
        "updated": 3,
        "updated_impl": 1,
        "updated_rename": 1,
        "updated_baseline": 1
    },
    "truncated": false,
    "highlights": [
        {
            "type": "Changed",
            "feature_id": "8",
            "feature_name": "H",
            "baseline_change": {
                "from": {
                    "status": "limited"
                },
                "to": {
                    "status": "newly",
                    "low_date": "2025-01-01T00:00:00Z"
                }
            }
        },
        {
            "type": "Changed",
            "feature_id": "9",
            "feature_name": "I",
            "browser_changes": {
                "chrome": {
                    "from": {
                        "status": "unavailable"
                    },
                    "to": {
						"date": "2024-01-01T00:00:00Z",
                        "status": "available",
                        "version": "123"
                    }
                }
            }
        },
        {
            "type": "Changed",
            "feature_id": "10",
            "feature_name": "J",
            "name_change": {
                "from": "Old",
                "to": "New"
            }
        },
        {
            "type": "Added",
            "feature_id": "1",
            "feature_name": "A"
        },
        {
            "type": "Added",
            "feature_id": "2",
            "feature_name": "B",
            "docs": {
                "mdn_docs": [
					{
						"url": "https://mdn.io/B",
						"title": "B",
						"slug": "slug-b"
					}
            	]
			}
        },
        {
            "type": "Removed",
            "feature_id": "3",
            "feature_name": "C"
        },
        {
            "type": "Removed",
            "feature_id": "31",
            "feature_name": "K",
            "baseline_change": {
                "from": {
                    "status": "limited"
                },
                "to": {
                    "status": "newly",
                    "low_date": "2025-01-01T00:00:00Z"
                }
            },
            "browser_changes": {
                "chrome": {
                    "from": {
                        "status": "unavailable"
                    },
                    "to": {
						"date": "2024-01-01T00:00:00Z",
                        "status": "available",
                        "version": "132"
                    }
                }
            }
        },
		{
            "type": "Deleted",
            "feature_id": "4",
            "feature_name": "D"
        },
        {
            "type": "Moved",
            "feature_id": "5",
            "feature_name": "E",
            "moved": {
                "from": {
                    "id": "4",
                    "name": "D"
                },
                "to": {
                    "id": "5",
                    "name": "E",
                    "query_match": "match"
                }
            }
        },
        {
            "type": "Split",
            "feature_id": "6",
            "feature_name": "F",
            "split": {
                "from": {
                    "id": "6",
                    "name": "F"
                },
                "to": [
                    {
                        "id": "7",
                        "name": "G",
                        "query_match": "match"
                    }
                ]
            }
        }
    ]
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
	var actualObj, expectedObj any
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

func TestGenerateCategoryDetails(t *testing.T) {
	newlyDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	widelyDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		modifications []v1.FeatureModified
		expected      []string
	}{
		{
			name:          "Empty list",
			modifications: []v1.FeatureModified{},
			// generateCategoryDetails builds a slice, if empty conditions, it returns nil from buildBaselineDetails
			expected: nil,
		},
		{
			name: "Only Newly",
			modifications: []v1.FeatureModified{
				{
					ID:             "1",
					Name:           "A",
					NameChange:     nil,
					BrowserChanges: nil,
					Docs:           nil,
					DocsChange:     nil,
					BaselineChange: &v1.Change[v1.BaselineState]{
						From: v1.BaselineState{
							Status:   generic.UnsetOpt[v1.BaselineInfoStatus](),
							LowDate:  generic.UnsetOpt[*time.Time](),
							HighDate: generic.UnsetOpt[*time.Time](),
						},
						To: v1.BaselineState{
							Status:   generic.SetOpt(v1.Newly),
							LowDate:  generic.SetOpt(&newlyDate),
							HighDate: generic.UnsetOpt[*time.Time](),
						},
					},
				},
				{
					ID:             "2",
					Name:           "B",
					NameChange:     nil,
					BrowserChanges: nil,
					Docs:           nil,
					DocsChange:     nil,
					BaselineChange: &v1.Change[v1.BaselineState]{
						From: v1.BaselineState{
							Status:   generic.UnsetOpt[v1.BaselineInfoStatus](),
							LowDate:  generic.UnsetOpt[*time.Time](),
							HighDate: generic.UnsetOpt[*time.Time](),
						},
						To: v1.BaselineState{
							Status:   generic.SetOpt(v1.Newly),
							LowDate:  generic.SetOpt(&newlyDate),
							HighDate: generic.UnsetOpt[*time.Time](),
						},
					},
				},
			},
			expected: []string{"2 became Baseline newly available"},
		},
		{
			name: "Newly and Widely",
			modifications: []v1.FeatureModified{
				{
					ID:             "3",
					Name:           "C",
					NameChange:     nil,
					BrowserChanges: nil,
					Docs:           nil,
					DocsChange:     nil,
					BaselineChange: &v1.Change[v1.BaselineState]{
						From: v1.BaselineState{
							Status:   generic.UnsetOpt[v1.BaselineInfoStatus](),
							LowDate:  generic.UnsetOpt[*time.Time](),
							HighDate: generic.UnsetOpt[*time.Time](),
						},
						To: v1.BaselineState{
							Status:   generic.SetOpt(v1.Newly),
							LowDate:  generic.SetOpt(&newlyDate),
							HighDate: generic.UnsetOpt[*time.Time](),
						},
					},
				},
				{
					ID:             "4",
					Name:           "D",
					NameChange:     nil,
					BrowserChanges: nil,
					Docs:           nil,
					DocsChange:     nil,
					BaselineChange: &v1.Change[v1.BaselineState]{
						From: v1.BaselineState{
							Status:   generic.UnsetOpt[v1.BaselineInfoStatus](),
							LowDate:  generic.UnsetOpt[*time.Time](),
							HighDate: generic.UnsetOpt[*time.Time](),
						},
						To: v1.BaselineState{
							Status:   generic.SetOpt(v1.Widely),
							LowDate:  generic.SetOpt(&newlyDate),
							HighDate: generic.SetOpt(&widelyDate),
						},
					},
				},
			},
			expected: []string{"1 became Baseline newly available", "1 became Baseline widely available"},
		},
		{
			name: "No baseline changes",
			modifications: []v1.FeatureModified{
				{
					ID:             "5",
					Name:           "E",
					NameChange:     &v1.Change[string]{From: "A", To: "B"},
					BaselineChange: nil,
					BrowserChanges: nil,
					Docs:           nil,
					DocsChange:     nil,
				},
			},
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := generateCategoryDetails(tc.modifications)
			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Errorf("generateCategoryDetails() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFilterHighlights(t *testing.T) {
	newlyDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	widelyDate := time.Date(2025, 12, 27, 0, 0, 0, 0, time.UTC)
	availableDate := time.Date(2025, 12, 28, 0, 0, 0, 0, time.UTC)

	// Reusable highlight definitions
	hNewly := SummaryHighlight{
		Type:        SummaryHighlightTypeChanged,
		FeatureID:   "h1",
		FeatureName: "Newly Feature",
		BaselineChange: &Change[BaselineValue]{
			From: BaselineValue{Status: BaselineStatusLimited, LowDate: nil, HighDate: nil},
			To: BaselineValue{Status: BaselineStatusNewly, LowDate: &newlyDate,
				HighDate: nil},
		},
		Docs:           nil,
		NameChange:     nil,
		BrowserChanges: nil,
		Moved:          nil,
		Split:          nil,
	}
	hWidely := SummaryHighlight{
		Type:        SummaryHighlightTypeChanged,
		FeatureID:   "h2",
		FeatureName: "Widely Feature",
		BaselineChange: &Change[BaselineValue]{
			From: BaselineValue{Status: BaselineStatusNewly, LowDate: &newlyDate,
				HighDate: nil},
			To: BaselineValue{Status: BaselineStatusWidely, LowDate: &newlyDate,
				HighDate: &widelyDate},
		},
		Docs:           nil,
		NameChange:     nil,
		BrowserChanges: nil,
		Moved:          nil,
		Split:          nil,
	}
	hRegression := SummaryHighlight{
		Type:        SummaryHighlightTypeChanged,
		FeatureID:   "h3",
		FeatureName: "Regression Feature",
		BaselineChange: &Change[BaselineValue]{
			From: BaselineValue{Status: BaselineStatusWidely, LowDate: &newlyDate,
				HighDate: &widelyDate},
			To: BaselineValue{Status: BaselineStatusLimited, LowDate: nil, HighDate: nil},
		},
		Docs:           nil,
		NameChange:     nil,
		BrowserChanges: nil,
		Moved:          nil,
		Split:          nil,
	}
	hBrowser := SummaryHighlight{
		Type:        SummaryHighlightTypeChanged,
		FeatureID:   "h4",
		FeatureName: "Browser Feature",
		BrowserChanges: map[BrowserName]*Change[BrowserValue]{
			BrowserChrome: {
				From: BrowserValue{Status: BrowserStatusUnavailable, Version: nil, Date: nil},
				To:   BrowserValue{Status: BrowserStatusAvailable, Version: nil, Date: &availableDate},
			},
			BrowserEdge:           nil,
			BrowserFirefox:        nil,
			BrowserSafari:         nil,
			BrowserChromeAndroid:  nil,
			BrowserFirefoxAndroid: nil,
			BrowserSafariIos:      nil,
		},
		BaselineChange: nil,
		Docs:           nil,
		NameChange:     nil,
		Moved:          nil,
		Split:          nil,
	}
	hGenericAdded := SummaryHighlight{
		Type:           SummaryHighlightTypeAdded,
		FeatureID:      "h5",
		FeatureName:    "Generic Added",
		Docs:           nil,
		NameChange:     nil,
		BaselineChange: nil,
		BrowserChanges: nil,
		Moved:          nil,
		Split:          nil,
	}

	allHighlights := []SummaryHighlight{hNewly, hWidely, hRegression, hBrowser, hGenericAdded}

	tests := []struct {
		name     string
		triggers []JobTrigger
		wantIDs  []string
	}{
		{
			name:     "No Triggers (Default) - Should Return All",
			triggers: nil,
			wantIDs:  []string{"h1", "h2", "h3", "h4", "h5"},
		},
		{
			name:     "Empty Triggers List - Should Return All",
			triggers: []JobTrigger{},
			wantIDs:  []string{"h1", "h2", "h3", "h4", "h5"},
		},
		{
			name:     "Newly Trigger",
			triggers: []JobTrigger{FeaturePromotedToNewly},
			wantIDs:  []string{"h1"},
		},
		{
			name:     "Widely Trigger",
			triggers: []JobTrigger{FeaturePromotedToWidely},
			wantIDs:  []string{"h2"},
		},
		{
			name:     "Regression Trigger",
			triggers: []JobTrigger{FeatureRegressedToLimited},
			wantIDs:  []string{"h3"},
		},
		{
			name:     "Browser Implementation Trigger",
			triggers: []JobTrigger{BrowserImplementationAnyComplete},
			wantIDs:  []string{"h4"},
		},
		{
			name: "Multiple Triggers",
			triggers: []JobTrigger{
				FeaturePromotedToNewly,
				FeaturePromotedToWidely,
			},
			wantIDs: []string{"h1", "h2"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FilterHighlights(allHighlights, tc.triggers)

			if len(got) != len(tc.wantIDs) {
				t.Errorf("Count mismatch: got %d, want %d", len(got), len(tc.wantIDs))
			}

			for i, h := range got {
				if i < len(tc.wantIDs) && h.FeatureID != tc.wantIDs[i] {
					t.Errorf("Index %d mismatch: got ID %s, want %s", i, h.FeatureID, tc.wantIDs[i])
				}
			}
		})
	}
}
