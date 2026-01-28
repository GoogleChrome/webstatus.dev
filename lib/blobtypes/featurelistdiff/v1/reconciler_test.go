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

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/google/go-cmp/cmp"
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
		initialDiff  *FeatureDiff
		mockResults  map[string]*backendtypes.GetFeatureResult
		mockErrors   map[string]error
		expectedDiff *FeatureDiff
		wantErr      bool
	}{
		{
			name: "Scenario 1: Feature Moved (Rename)",
			initialDiff: &FeatureDiff{
				Removed:      []FeatureRemoved{{ID: "old-id", Name: "Old Name", Reason: ReasonUnmatched}},
				Deleted:      nil,
				Added:        []FeatureAdded{{ID: "new-id", Name: "New Name", Reason: ReasonNewMatch, Docs: nil}},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			mockResults: map[string]*backendtypes.GetFeatureResult{
				"old-id": backendtypes.NewGetFeatureResult(
					backendtypes.NewMovedFeatureResult("new-id"),
				),
			},
			mockErrors: nil,
			expectedDiff: &FeatureDiff{
				Removed: nil, // Should be cleared
				Added:   nil, // Should be cleared
				Moves: []FeatureMoved{
					{FromID: "old-id", FromName: "Old Name", ToID: "new-id", ToName: "New Name"},
				},
				QueryChanged: false,
				Modified:     nil,
				Splits:       nil,
				Deleted:      nil,
			},
			wantErr: false,
		},
		{
			name: "Scenario 2: Feature Split (Full)",
			initialDiff: &FeatureDiff{
				Removed: []FeatureRemoved{{ID: "monolith", Name: "Monolith Feature", Reason: ReasonUnmatched}},
				Added: []FeatureAdded{
					{ID: "part-1", Name: "Part 1", Reason: ReasonNewMatch, Docs: nil},
					{ID: "part-2", Name: "Part 2", Reason: ReasonNewMatch, Docs: nil},
				},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
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
							{ID: "part-1", Name: "Part 1", Reason: ReasonNewMatch, Docs: nil},
							{ID: "part-2", Name: "Part 2", Reason: ReasonNewMatch, Docs: nil},
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
			name: "Scenario 3: Feature Split (Partial / Out of Scope)",
			// 'part-2' matches the split definition but isn't in the Added list (maybe filtered out by query)
			initialDiff: &FeatureDiff{
				Removed: []FeatureRemoved{{ID: "monolith", Name: "Monolith Feature", Reason: ReasonUnmatched}},
				Added: []FeatureAdded{
					{ID: "part-1", Name: "Part 1", Reason: ReasonNewMatch, Docs: nil},
				},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
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
						// Only part-1 is included because part-2 wasn't in the 'Added' list
						To: []FeatureAdded{
							{ID: "part-1", Name: "Part 1", Reason: ReasonNewMatch, Docs: nil},
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
			name: "Scenario 4: Regular Removal (No Move/Split)",
			initialDiff: &FeatureDiff{
				Removed:      []FeatureRemoved{{ID: "removed-id", Name: "Removed Feature", Reason: ReasonUnmatched}},
				Added:        nil,
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			mockResults: map[string]*backendtypes.GetFeatureResult{
				"removed-id": backendtypes.NewGetFeatureResult(
					backendtypes.NewRegularFeatureResult(&backend.Feature{
						FeatureId:                  "removed-id",
						Name:                       "",
						Spec:                       nil,
						Baseline:                   nil,
						BrowserImplementations:     nil,
						Discouraged:                nil,
						Usage:                      nil,
						Wpt:                        nil,
						VendorPositions:            nil,
						DeveloperSignals:           nil,
						SystemManagedSavedSearchId: nil,
					}),
				),
			},
			mockErrors: nil,
			expectedDiff: &FeatureDiff{
				// Remains in Removed list
				Removed:      []FeatureRemoved{{ID: "removed-id", Name: "Removed Feature", Reason: ReasonUnmatched}},
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
				Deleted:      []FeatureDeleted{{ID: "deleted-id", Name: "Deleted Feature", Reason: ReasonDeleted}},
				Added:        nil,
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Removed:      nil,
			},
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
			name: "Scenario 6: Move Target Missing from Added List",
			// History says A moved to B, but B is NOT in the Added list.
			// Should act as a regular removal.
			initialDiff: &FeatureDiff{
				Removed:      []FeatureRemoved{{ID: "old-id", Name: "Old Name", Reason: ReasonUnmatched}},
				Added:        []FeatureAdded{{ID: "unrelated-id", Name: "Unrelated", Reason: ReasonNewMatch, Docs: nil}},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			mockResults: map[string]*backendtypes.GetFeatureResult{
				"old-id": backendtypes.NewGetFeatureResult(
					backendtypes.NewMovedFeatureResult("missing-new-id"),
				),
			},
			mockErrors: nil,
			expectedDiff: &FeatureDiff{
				Removed:      []FeatureRemoved{{ID: "old-id", Name: "Old Name", Reason: ReasonUnmatched}},
				Added:        []FeatureAdded{{ID: "unrelated-id", Name: "Unrelated", Reason: ReasonNewMatch, Docs: nil}},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			wantErr: false,
		},
		{
			name: "Scenario 7: DB Error",
			initialDiff: &FeatureDiff{
				Removed:      []FeatureRemoved{{ID: "error-id", Name: "Error Feature", Reason: ReasonUnmatched}},
				Added:        nil,
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			mockResults: nil,
			mockErrors: map[string]error{
				"error-id": errors.New("db connection failed"),
			},
			expectedDiff: nil,
			wantErr:      true,
		},
		{
			name: "Scenario 8: Split Targets Completely Missing",
			// History says A split into B, but B is NOT in the Added list.
			// Should act as a regular removal (hitting the 'else' block).
			initialDiff: &FeatureDiff{
				Removed:      []FeatureRemoved{{ID: "monolith", Name: "Monolith Feature", Reason: ReasonUnmatched}},
				Added:        []FeatureAdded{{ID: "unrelated", Name: "Unrelated", Reason: ReasonNewMatch, Docs: nil}},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			mockResults: map[string]*backendtypes.GetFeatureResult{
				"monolith": backendtypes.NewGetFeatureResult(
					backendtypes.NewSplitFeatureResult(backend.FeatureEvolutionSplit{
						Features: []backend.FeatureSplitInfo{
							{Id: "missing-part"},
						},
					}),
				),
			},
			mockErrors: nil,
			expectedDiff: &FeatureDiff{
				Removed:      []FeatureRemoved{{ID: "monolith", Name: "Monolith Feature", Reason: ReasonUnmatched}},
				Added:        []FeatureAdded{{ID: "unrelated", Name: "Unrelated", Reason: ReasonNewMatch, Docs: nil}},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			wantErr: false,
		},
		{
			name: "Scenario 9: Unrelated Additions Preserved",
			// A moved to B. C is just a new feature.
			// Result should be Move(A->B) + Added(C). B should NOT be in Added list.
			initialDiff: &FeatureDiff{
				Removed: []FeatureRemoved{{ID: "old-id", Name: "Old Name", Reason: ReasonUnmatched}},
				Added: []FeatureAdded{
					{ID: "new-id", Name: "New Name", Reason: ReasonNewMatch, Docs: nil},
					{ID: "extra-id", Name: "Extra Feature", Reason: ReasonNewMatch, Docs: nil},
				},
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			mockResults: map[string]*backendtypes.GetFeatureResult{
				"old-id": backendtypes.NewGetFeatureResult(
					backendtypes.NewMovedFeatureResult("new-id"),
				),
			},
			mockErrors: nil,
			expectedDiff: &FeatureDiff{
				Removed: nil,
				Added:   []FeatureAdded{{ID: "extra-id", Name: "Extra Feature", Reason: ReasonNewMatch, Docs: nil}},
				Moves: []FeatureMoved{
					{FromID: "old-id", FromName: "Old Name", ToID: "new-id", ToName: "New Name"},
				},
				QueryChanged: false,
				Modified:     nil,
				Splits:       nil,
				Deleted:      nil,
			},
			wantErr: false,
		},
		{
			name: "Scenario 10: Mixed Removed and Deleted",
			initialDiff: &FeatureDiff{
				Removed: []FeatureRemoved{
					{ID: "deleted-id", Name: "Deleted Feature", Reason: ReasonUnmatched},
					{ID: "removed-id", Name: "Removed Feature", Reason: ReasonUnmatched},
				},
				Added:        nil,
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
				Deleted:      nil,
			},
			mockResults: map[string]*backendtypes.GetFeatureResult{
				"removed-id": backendtypes.NewGetFeatureResult(
					backendtypes.NewRegularFeatureResult(&backend.Feature{
						FeatureId:                  "removed-id",
						Name:                       "",
						Spec:                       nil,
						Baseline:                   nil,
						BrowserImplementations:     nil,
						Discouraged:                nil,
						Usage:                      nil,
						Wpt:                        nil,
						VendorPositions:            nil,
						DeveloperSignals:           nil,
						SystemManagedSavedSearchId: nil,
					}),
				),
			},
			mockErrors: map[string]error{
				"deleted-id": backendtypes.ErrEntityDoesNotExist,
			},
			expectedDiff: &FeatureDiff{
				Removed:      []FeatureRemoved{{ID: "removed-id", Name: "Removed Feature", Reason: ReasonUnmatched}},
				Deleted:      []FeatureDeleted{{ID: "deleted-id", Name: "Deleted Feature", Reason: ReasonDeleted}},
				Added:        nil,
				QueryChanged: false,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
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

			// We manually sort inputs here to ensure the test case inputs match
			// what a real system might produce before reconciliation.
			// (Though in reality, the Comparator output is usually unsorted until Run() finishes).
			if tc.initialDiff != nil {
				tc.initialDiff.Sort()
			}

			w.diff = tc.initialDiff

			err := w.ReconcileHistory(context.Background())

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
