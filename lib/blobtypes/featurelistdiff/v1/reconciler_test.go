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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/generic"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes/comparables"
	"github.com/google/go-cmp/cmp"
	"github.com/oapi-codegen/runtime/types"
)

// mockReconcileClient mocks the FeatureFetcher interface for reconciliation tests.
type mockReconcileClient struct {
	// Map featureID -> Result to return
	results map[string]*backendtypes.GetFeatureResult
	// Map featureID -> Error to return
	errors map[string]error
}

func (m *mockReconcileClient) FetchFeatures(_ context.Context, _ string) ([]backend.Feature, error) {
	return nil, nil // Not used in reconciler tests
}

func (m *mockReconcileClient) GetFeature(
	_ context.Context,
	featureID string,
) (*backendtypes.GetFeatureResult, error) {
	if err, ok := m.errors[featureID]; ok {
		return nil, err
	}
	if res, ok := m.results[featureID]; ok {
		return res, nil
	}
	// Default to not found if not mocked
	return nil, backendtypes.ErrEntityDoesNotExist
}

func TestReconcileHistory(t *testing.T) {
	tests := []struct {
		name         string
		initialDiff  *FeatureDiff // Starting point from comparator output before reconciliation
		oldState     map[string]comparables.Feature
		newState     map[string]comparables.Feature
		mockResults  map[string]*backendtypes.GetFeatureResult
		mockErrors   map[string]error
		expectedDiff *FeatureDiff
		wantErr      bool
	}{
		{
			name: "Scenario 1: Feature Moved (Rename) - Matching",
			initialDiff: &FeatureDiff{
				Removed: []FeatureRemoved{{ID: "old-id", Name: "Old Name", Reason: ReasonUnmatched, Diff: nil}},
				Deleted: nil,
				Added: []FeatureAdded{{ID: "new-id", Name: "New Name", Reason: ReasonNewMatch, Docs: nil,
					QueryMatch: QueryMatchMatch}},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			oldState: map[string]comparables.Feature{
				"old-id": newBaseFeature("old-id", "Old Name", "limited"),
			},
			newState: map[string]comparables.Feature{
				"new-id": newBaseFeature("new-id", "New Name", "limited"),
			},
			mockResults: map[string]*backendtypes.GetFeatureResult{
				"old-id": backendtypes.NewGetFeatureResult(
					backendtypes.NewMovedFeatureResult("new-id"),
				),
			},
			mockErrors: nil,
			expectedDiff: &FeatureDiff{
				Removed: nil,
				Added:   nil,
				Moves: []FeatureMoved{
					{
						FromID:     "old-id",
						FromName:   "Old Name",
						ToID:       "new-id",
						ToName:     "New Name",
						QueryMatch: QueryMatchMatch,
					},
				},
				QueryChanged: false,
				Modified:     nil,
				Splits:       nil,
				Deleted:      nil,
			},
			wantErr: false,
		},
		{
			name: "Scenario 2: Feature Split (Full) - All Matching",
			initialDiff: &FeatureDiff{
				Removed: []FeatureRemoved{{ID: "monolith", Name: "Monolith Feature", Reason: ReasonUnmatched,
					Diff: nil}},
				Added: []FeatureAdded{
					{ID: "part-1", Name: "Part 1", Reason: ReasonNewMatch, Docs: nil, QueryMatch: QueryMatchMatch},
					{ID: "part-2", Name: "Part 2", Reason: ReasonNewMatch, Docs: nil, QueryMatch: QueryMatchMatch},
				},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			oldState: nil,
			newState: map[string]comparables.Feature{
				"part-1": newBaseFeature("part-1", "Part 1", "limited"),
				"part-2": newBaseFeature("part-2", "Part 2", "limited"),
			},
			mockResults: map[string]*backendtypes.GetFeatureResult{
				"monolith": backendtypes.NewGetFeatureResult(
					backendtypes.NewSplitFeatureResult(backend.FeatureEvolutionSplit{
						Features: []backend.FeatureSplitInfo{
							{Id: "part-1"},
							{Id: "part-2"},
						},
					}),
				),
			},
			mockErrors: nil,
			expectedDiff: &FeatureDiff{
				Removed: nil,
				Added:   nil,
				Splits: []FeatureSplit{
					{
						FromID:   "monolith",
						FromName: "Monolith Feature",
						To: []FeatureAdded{
							{ID: "part-1", Name: "Part 1", Reason: ReasonNewMatch, Docs: nil, QueryMatch: QueryMatchMatch},
							{ID: "part-2", Name: "Part 2", Reason: ReasonNewMatch, Docs: nil, QueryMatch: QueryMatchMatch},
						},
					},
				},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Deleted:      nil,
			},
			wantErr: false,
		},
		{
			name: "Scenario 3: Feature Split - Partial Matching",
			initialDiff: &FeatureDiff{
				Removed: []FeatureRemoved{{ID: "monolith", Name: "Monolith Feature", Reason: ReasonUnmatched,
					Diff: nil}},
				Added: []FeatureAdded{
					{ID: "part-1", Name: "Part 1", Reason: ReasonNewMatch, Docs: nil, QueryMatch: QueryMatchMatch},
				},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			oldState: nil,
			newState: map[string]comparables.Feature{
				"part-1": newBaseFeature("part-1", "Part 1", "limited"),
			},
			mockResults: map[string]*backendtypes.GetFeatureResult{
				"monolith": backendtypes.NewGetFeatureResult(
					backendtypes.NewSplitFeatureResult(backend.FeatureEvolutionSplit{
						Features: []backend.FeatureSplitInfo{
							{Id: "part-1"},
							{Id: "part-2"},
						},
					}),
				),
				"part-2": backendtypes.NewGetFeatureResult(
					backendtypes.NewRegularFeatureResult(&backend.Feature{
						FeatureId:                  "part-2",
						Name:                       "Part 2",
						Baseline:                   nil,
						BrowserImplementations:     nil,
						DeveloperSignals:           nil,
						Discouraged:                nil,
						Spec:                       nil,
						SystemManagedSavedSearchId: nil,
						Usage:                      nil,
						VendorPositions:            nil,
						Wpt:                        nil,
					}),
				),
			},
			mockErrors: nil,
			expectedDiff: &FeatureDiff{
				Removed: nil,
				Added:   nil,
				Splits: []FeatureSplit{
					{
						FromID:   "monolith",
						FromName: "Monolith Feature",
						To: []FeatureAdded{
							{ID: "part-1", Name: "Part 1", Reason: ReasonNewMatch, Docs: nil,
								QueryMatch: QueryMatchMatch},
							{ID: "part-2", Name: "Part 2", Reason: ReasonUnmatched, Docs: nil,
								QueryMatch: QueryMatchNoMatch},
						},
					},
				},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Deleted:      nil,
			},
			wantErr: false,
		},
		{
			name: "Scenario 4: Regular Removal (Unmatched with Diff)",
			initialDiff: &FeatureDiff{
				Removed: []FeatureRemoved{{ID: "removed-id", Name: "Removed Feature", Reason: ReasonUnmatched,
					Diff: nil}},
				Added:        nil,
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			oldState: map[string]comparables.Feature{
				"removed-id": newBaseFeature("removed-id", "Removed Feature", "limited"),
			},
			newState: nil,
			mockResults: map[string]*backendtypes.GetFeatureResult{
				"removed-id": backendtypes.NewGetFeatureResult(
					backendtypes.NewRegularFeatureResult(&backend.Feature{
						FeatureId: "removed-id",
						Name:      "Removed Feature",
						Baseline: &backend.BaselineInfo{
							Status:   generic.ValuePtr(backend.Newly),
							LowDate:  generic.ValuePtr(types.Date{Time: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}),
							HighDate: nil,
						},
						BrowserImplementations:     nil,
						DeveloperSignals:           nil,
						Discouraged:                nil,
						Spec:                       nil,
						SystemManagedSavedSearchId: nil,
						Usage:                      nil,
						VendorPositions:            nil,
						Wpt:                        nil,
					}),
				),
			},
			mockErrors: nil,
			expectedDiff: &FeatureDiff{
				// Remains in Removed list
				Removed: []FeatureRemoved{{ID: "removed-id", Name: "Removed Feature", Reason: ReasonUnmatched,
					Diff: &FeatureModified{
						ID:   "removed-id",
						Name: "Removed Feature",
						BaselineChange: &Change[BaselineState]{
							From: BaselineState{
								Status:   generic.SetOpt(Limited),
								LowDate:  generic.UnsetOpt[*time.Time](),
								HighDate: generic.UnsetOpt[*time.Time](),
							},
							To: BaselineState{
								Status:   generic.SetOpt(Newly),
								LowDate:  generic.SetOpt(generic.ValuePtr(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))),
								HighDate: generic.SetOpt[*time.Time](nil),
							},
						},
						NameChange:     nil,
						BrowserChanges: nil,
						Docs:           nil,
						DocsChange:     nil,
					}}},
				Added:        nil,
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			wantErr: false,
		},
		{
			name: "Scenario 5: Hard Delete (EntityDoesNotExist)",
			initialDiff: &FeatureDiff{
				Removed: []FeatureRemoved{{ID: "deleted-id", Name: "Deleted Feature",
					Reason: ReasonUnmatched, Diff: nil}},
				Deleted:      nil,
				Added:        nil,
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			oldState: map[string]comparables.Feature{
				"deleted-id": newBaseFeature("deleted-id", "Deleted Feature", "limited"),
			},
			newState:    nil,
			mockResults: nil,
			mockErrors: map[string]error{
				"deleted-id": backendtypes.ErrEntityDoesNotExist,
			},
			expectedDiff: &FeatureDiff{
				// Should be moved to Deleted list
				Removed:      nil,
				Deleted:      []FeatureDeleted{{ID: "deleted-id", Name: "Deleted Feature", Reason: ReasonDeleted}},
				Added:        nil,
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			wantErr: false,
		},
		{
			name: "Scenario 6: Move Destination Out of Scope",
			initialDiff: &FeatureDiff{
				Removed: []FeatureRemoved{{ID: "old-id", Name: "Old Name", Reason: ReasonUnmatched, Diff: nil}},
				Added: []FeatureAdded{{ID: "unrelated-id", Name: "Unrelated", Reason: ReasonNewMatch, Docs: nil,
					QueryMatch: QueryMatchMatch}},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			oldState: nil,
			newState: map[string]comparables.Feature{
				"unrelated-id": newBaseFeature("unrelated-id", "Unrelated", "limited"),
			},
			mockResults: map[string]*backendtypes.GetFeatureResult{
				"old-id": backendtypes.NewGetFeatureResult(
					backendtypes.NewMovedFeatureResult("missing-new-id"),
				),
				"missing-new-id": backendtypes.NewGetFeatureResult(
					backendtypes.NewRegularFeatureResult(&backend.Feature{
						FeatureId:                  "missing-new-id",
						Name:                       "New Name",
						Baseline:                   nil,
						BrowserImplementations:     nil,
						DeveloperSignals:           nil,
						Discouraged:                nil,
						Spec:                       nil,
						SystemManagedSavedSearchId: nil,
						Usage:                      nil,
						VendorPositions:            nil,
						Wpt:                        nil,
					}),
				),
			},
			mockErrors: nil,
			expectedDiff: &FeatureDiff{
				Removed: nil,
				Added: []FeatureAdded{{ID: "unrelated-id", Name: "Unrelated", Reason: ReasonNewMatch, Docs: nil,
					QueryMatch: QueryMatchMatch}},
				Moves: []FeatureMoved{
					{
						FromID:     "old-id",
						FromName:   "Old Name",
						ToID:       "missing-new-id",
						ToName:     "New Name",
						QueryMatch: QueryMatchNoMatch,
					},
				},
				QueryChanged: false,
				Modified:     nil,
				Splits:       nil,
				Deleted:      nil,
			},
			wantErr: false,
		},
		{
			name: "Scenario 7: DB Error",
			initialDiff: &FeatureDiff{
				Removed: []FeatureRemoved{{ID: "error-id", Name: "Error Feature", Reason: ReasonUnmatched,
					Diff: nil}},
				Added:        nil,
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			oldState:    nil,
			newState:    nil,
			mockResults: nil,
			mockErrors: map[string]error{
				"error-id": errors.New("db connection failed"),
			},
			expectedDiff: nil,
			wantErr:      true,
		},
		{
			name: "Scenario 8: Split Destination Completely Out of Scope",
			initialDiff: &FeatureDiff{
				Removed: []FeatureRemoved{{ID: "monolith", Name: "Monolith Feature", Reason: ReasonUnmatched,
					Diff: nil}},
				Added: []FeatureAdded{{ID: "unrelated", Name: "Unrelated",
					Reason: ReasonNewMatch, Docs: nil, QueryMatch: QueryMatchMatch}},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			oldState: nil,
			newState: map[string]comparables.Feature{
				"unrelated": newBaseFeature("unrelated", "Unrelated", "limited"),
			},
			mockResults: map[string]*backendtypes.GetFeatureResult{
				"monolith": backendtypes.NewGetFeatureResult(
					backendtypes.NewSplitFeatureResult(backend.FeatureEvolutionSplit{
						Features: []backend.FeatureSplitInfo{
							{Id: "missing-b"},
						},
					}),
				),
				"missing-b": backendtypes.NewGetFeatureResult(
					backendtypes.NewRegularFeatureResult(&backend.Feature{
						FeatureId:                  "missing-b",
						Name:                       "Part B",
						Baseline:                   nil,
						BrowserImplementations:     nil,
						DeveloperSignals:           nil,
						Discouraged:                nil,
						Spec:                       nil,
						SystemManagedSavedSearchId: nil,
						Usage:                      nil,
						VendorPositions:            nil,
						Wpt:                        nil,
					}),
				),
			},
			mockErrors: nil,
			expectedDiff: &FeatureDiff{
				Removed: nil,
				Added: []FeatureAdded{{ID: "unrelated", Name: "Unrelated", Reason: ReasonNewMatch, Docs: nil,
					QueryMatch: QueryMatchMatch}},
				Splits: []FeatureSplit{
					{
						FromID:   "monolith",
						FromName: "Monolith Feature",
						To: []FeatureAdded{
							{
								ID:         "missing-b",
								Name:       "Part B",
								Reason:     ReasonUnmatched,
								Docs:       nil,
								QueryMatch: QueryMatchNoMatch,
							},
						},
					},
				},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Deleted:      nil,
			},
			wantErr: false,
		},
		{
			name: "Scenario 9: Unrelated Additions Preserved",
			// A moved to B. C is just a new feature.
			// Result should be Move(A->B) + Added(C). B should NOT be in Added list.
			initialDiff: &FeatureDiff{
				Removed: []FeatureRemoved{{ID: "old-id", Name: "Old Name", Reason: ReasonUnmatched, Diff: nil}},
				Added: []FeatureAdded{
					{ID: "new-id", Name: "New Name", Reason: ReasonNewMatch, Docs: nil, QueryMatch: QueryMatchMatch},
					{ID: "extra-id", Name: "Extra Feature", Reason: ReasonNewMatch, Docs: nil, QueryMatch: QueryMatchMatch},
				},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			oldState: nil,
			newState: map[string]comparables.Feature{
				"new-id":   newBaseFeature("new-id", "New Name", "limited"),
				"extra-id": newBaseFeature("extra-id", "Extra Feature", "limited"),
			},
			mockResults: map[string]*backendtypes.GetFeatureResult{
				"old-id": backendtypes.NewGetFeatureResult(
					backendtypes.NewMovedFeatureResult("new-id"),
				),
			},
			mockErrors: nil,
			expectedDiff: &FeatureDiff{
				Removed: nil,
				Added: []FeatureAdded{
					{ID: "extra-id", Name: "Extra Feature", Reason: ReasonNewMatch, Docs: nil, QueryMatch: QueryMatchMatch},
				},
				QueryChanged: false,
				Modified:     nil,
				Moves: []FeatureMoved{
					{FromID: "old-id", FromName: "Old Name", ToID: "new-id", ToName: "New Name", QueryMatch: QueryMatchMatch},
				},
				Splits:  nil,
				Deleted: nil,
			},
			wantErr: false,
		},
		{
			name: "Scenario 10: Mixed Removed and Deleted",
			initialDiff: &FeatureDiff{
				Removed: []FeatureRemoved{
					{ID: "deleted-id", Name: "Deleted Feature", Reason: ReasonUnmatched, Diff: nil},
					{ID: "removed-id", Name: "Removed Feature", Reason: ReasonUnmatched, Diff: nil},
				},
				Added:        nil,
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			oldState: map[string]comparables.Feature{
				"removed-id": newBaseFeature("removed-id", "Removed Feature", "limited"),
				"deleted-id": newBaseFeature("deleted-id", "Deleted Feature", "limited"),
			},
			newState: nil,
			mockResults: map[string]*backendtypes.GetFeatureResult{
				"removed-id": backendtypes.NewGetFeatureResult(
					backendtypes.NewRegularFeatureResult(&backend.Feature{
						FeatureId: "removed-id",
						Name:      "Removed Feature",
						Baseline: &backend.BaselineInfo{
							// This simulates a feature that still exists but has changed baseline status,
							// so it should be reconciled as an unmatched removal with diff rather than a hard delete.
							Status:   generic.ValuePtr(backend.Newly),
							LowDate:  generic.ValuePtr(types.Date{Time: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}),
							HighDate: nil,
						},
						BrowserImplementations:     nil,
						DeveloperSignals:           nil,
						Discouraged:                nil,
						Spec:                       nil,
						SystemManagedSavedSearchId: nil,
						Usage:                      nil,
						VendorPositions:            nil,
						Wpt:                        nil,
					}),
				),
			},
			mockErrors: map[string]error{
				"deleted-id": backendtypes.ErrEntityDoesNotExist,
			},
			expectedDiff: &FeatureDiff{
				Removed: []FeatureRemoved{{ID: "removed-id", Name: "Removed Feature", Reason: ReasonUnmatched,
					Diff: &FeatureModified{
						ID:   "removed-id",
						Name: "Removed Feature",
						BaselineChange: &Change[BaselineState]{
							From: BaselineState{
								Status:   generic.SetOpt(Limited),
								LowDate:  generic.UnsetOpt[*time.Time](),
								HighDate: generic.UnsetOpt[*time.Time](),
							},
							To: BaselineState{
								Status:   generic.SetOpt(Newly),
								LowDate:  generic.SetOpt(generic.ValuePtr(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))),
								HighDate: generic.SetOpt[*time.Time](nil),
							},
						},
						NameChange:     nil,
						BrowserChanges: nil,
						Docs:           nil,
						DocsChange:     nil,
					}}},
				Deleted:      []FeatureDeleted{{ID: "deleted-id", Name: "Deleted Feature", Reason: ReasonDeleted}},
				Added:        nil,
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			wantErr: false,
		},
		{
			name: "Scenario 11: Move to Existing Feature (Not Added, but Matching)",
			// A moves to B. B was ALREADY in the list (so it's not in 'Added').
			// Result should be Move(A->B) with QueryMatchMatch.
			initialDiff: &FeatureDiff{
				Removed:      []FeatureRemoved{{ID: "old-id", Name: "Old Name", Reason: ReasonUnmatched, Diff: nil}},
				Added:        nil, // 'existing-id' is not Added because it was in oldState too
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			oldState: map[string]comparables.Feature{
				"old-id":      newBaseFeature("old-id", "Old Name", "limited"),
				"existing-id": newBaseFeature("existing-id", "Existing Name", "limited"),
			},
			newState: map[string]comparables.Feature{
				"existing-id": newBaseFeature("existing-id", "Existing Name", "limited"),
			},
			mockResults: map[string]*backendtypes.GetFeatureResult{
				"old-id": backendtypes.NewGetFeatureResult(
					backendtypes.NewMovedFeatureResult("existing-id"),
				),
			},
			mockErrors: nil,
			expectedDiff: &FeatureDiff{
				Removed: nil,
				Added:   nil,
				Moves: []FeatureMoved{
					{
						FromID:     "old-id",
						FromName:   "Old Name",
						ToID:       "existing-id",
						ToName:     "Existing Name",
						QueryMatch: QueryMatchMatch,
					},
				},
				QueryChanged: false,
				Modified:     nil,
				Splits:       nil,
				Deleted:      nil,
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &mockReconcileClient{
				results: tc.mockResults,
				errors:  tc.mockErrors,
			}
			w := NewFeatureDiffWorkflow(client, nil)

			if tc.initialDiff != nil {
				tc.initialDiff.Sort()
			}

			w.diff = tc.initialDiff

			err := w.ReconcileHistory(context.Background(), tc.oldState, tc.newState)

			if tc.wantErr {
				if err == nil {
					t.Fatal("reconcileHistory() expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("reconcileHistory() unexpected error: %v", err)
			}

			if tc.expectedDiff != nil {
				tc.expectedDiff.Sort()
			}

			if diff := cmp.Diff(tc.expectedDiff, w.diff); diff != "" {
				t.Errorf("reconcileHistory() mismatch. (-want +got):\n%s", diff)
			}
		})
	}
}
