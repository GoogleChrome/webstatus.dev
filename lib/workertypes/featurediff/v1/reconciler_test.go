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
		initialDiff  *FeatureDiffV1
		mockResults  map[string]*backendtypes.GetFeatureResult
		mockErrors   map[string]error
		expectedDiff *FeatureDiffV1
		wantErr      bool
	}{
		{
			name: "Scenario 1: Feature Moved (Rename)",
			initialDiff: &FeatureDiffV1{
				Removed:      []FeatureRemoved{{ID: "old-id", Name: "Old Name", Reason: ReasonUnmatched}},
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
			expectedDiff: &FeatureDiffV1{
				Removed: nil, // Should be cleared
				Added:   nil, // Should be cleared
				Moves: []FeatureMoved{
					{FromID: "old-id", FromName: "Old Name", ToID: "new-id", ToName: "New Name"},
				},
				QueryChanged: false,
				Modified:     nil,
				Splits:       nil,
			},
			wantErr: false,
		},
		// ... (other test cases from original file)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &mockReconcileClient{
				results: tc.mockResults,
				errors:  tc.mockErrors,
			}
			comparator := NewComparatorV1(client)

			if tc.initialDiff != nil {
				tc.initialDiff.Sort()
			}

			gotDiffResult, err := comparator.ReconcileHistory(context.Background(), tc.initialDiff)

			if tc.wantErr {
				if err == nil {
					t.Fatal("ReconcileHistory() expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("ReconcileHistory() unexpected error: %v", err)
			}

			// Type assert to sort
			if gotDiffV1, ok := gotDiffResult.Diff.(*FeatureDiffV1); ok {
				gotDiffV1.Sort()
			}

			if tc.expectedDiff != nil {
				tc.expectedDiff.Sort()
			}

			if diff := cmp.Diff(tc.expectedDiff, gotDiffResult.Diff); diff != "" {
				t.Errorf("ReconcileHistory() mismatch. (-want +got):\n%s", diff)
			}
		})
	}
}
