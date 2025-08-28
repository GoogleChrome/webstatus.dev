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

package gcpspanner

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSyncMovedWebFeatures(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		name          string
		initial       []MovedWebFeature
		sync          []MovedWebFeature
		expected      []MovedWebFeature
		expectedError error
	}

	testCases := []testCase{
		{
			name:    "initial sync",
			initial: []MovedWebFeature{},
			sync: []MovedWebFeature{
				{OriginalFeatureKey: "feature-a", NewFeatureKey: "feature-b"},
				{OriginalFeatureKey: "feature-c", NewFeatureKey: "feature-d"},
			},
			expected: []MovedWebFeature{
				{OriginalFeatureKey: "feature-a", NewFeatureKey: "feature-b"},
				{OriginalFeatureKey: "feature-c", NewFeatureKey: "feature-d"},
			},
			expectedError: nil,
		},
		{
			name: "delete",
			initial: []MovedWebFeature{
				{OriginalFeatureKey: "feature-a", NewFeatureKey: "feature-b"},
				{OriginalFeatureKey: "feature-c", NewFeatureKey: "feature-d"},
			},
			sync: []MovedWebFeature{
				{OriginalFeatureKey: "feature-a", NewFeatureKey: "feature-b"},
			},
			expected: []MovedWebFeature{
				{OriginalFeatureKey: "feature-a", NewFeatureKey: "feature-b"},
			},
			expectedError: nil,
		},
		{
			name: "add and delete",
			initial: []MovedWebFeature{
				{OriginalFeatureKey: "feature-a", NewFeatureKey: "feature-b"},
			},
			sync: []MovedWebFeature{
				{OriginalFeatureKey: "feature-c", NewFeatureKey: "feature-d"},
			},
			expected: []MovedWebFeature{
				{OriginalFeatureKey: "feature-c", NewFeatureKey: "feature-d"},
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restartDatabaseContainer(t)

			// Setup initial web features.
			err := spannerClient.SyncWebFeatures(ctx, []WebFeature{
				{FeatureKey: "feature-b", Name: "Feature B", Description: "", DescriptionHTML: ""},
				{FeatureKey: "feature-d", Name: "Feature D", Description: "", DescriptionHTML: ""},
			})
			if err != nil {
				t.Fatalf("failed to sync web features: %v", err)
			}

			// Setup initial moved features.
			if len(tc.initial) > 0 {
				err = spannerClient.SyncMovedWebFeatures(ctx, tc.initial)
				if err != nil {
					t.Fatalf("failed to sync initial moved features: %v", err)
				}
			}

			// Perform the sync to test.
			err = spannerClient.SyncMovedWebFeatures(ctx, tc.sync)
			if !errors.Is(err, tc.expectedError) {
				t.Fatalf("expected error %v, got %v", tc.expectedError, err)
			}

			// Verify the result.
			result, err := spannerClient.GetAllMovedWebFeatures(ctx)
			if err != nil {
				t.Fatalf("failed to get all moved features: %v", err)
			}

			sortFunc := func(a, b MovedWebFeature) bool {
				return a.OriginalFeatureKey < b.OriginalFeatureKey
			}
			if diff := cmp.Diff(tc.expected, result, cmpopts.SortSlices(sortFunc)); diff != "" {
				t.Errorf("unexpected result (-want +got): %s", diff)
			}
		})
	}
}

func TestGetMovedWebFeatureDetailsByOriginalFeatureKey(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	// Setup initial web features.
	err := spannerClient.SyncWebFeatures(ctx, []WebFeature{
		{FeatureKey: "feature-b", Name: "Feature B", Description: "", DescriptionHTML: ""},
	})
	if err != nil {
		t.Fatalf("failed to sync web features: %v", err)
	}

	// Setup moved feature.
	movedFeature := MovedWebFeature{OriginalFeatureKey: "feature-a", NewFeatureKey: "feature-b"}
	err = spannerClient.SyncMovedWebFeatures(ctx, []MovedWebFeature{movedFeature})
	if err != nil {
		t.Fatalf("failed to sync moved features: %v", err)
	}

	// Test found case.
	result, err := spannerClient.GetMovedWebFeatureDetailsByOriginalFeatureKey(ctx, "feature-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := &MovedWebFeature{OriginalFeatureKey: "feature-a", NewFeatureKey: "feature-b"}

	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("unexpected result (-want +got): %s", diff)
	}

	// Test not found case.
	_, err = spannerClient.GetMovedWebFeatureDetailsByOriginalFeatureKey(ctx, "non-existent-feature")
	if !errors.Is(err, ErrQueryReturnedNoResults) {
		t.Errorf("expected error %v, got %v", ErrQueryReturnedNoResults, err)
	}
}
