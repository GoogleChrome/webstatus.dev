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
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type splitWebFeaturesTestCase struct {
	name          string
	initial       []SplitWebFeature
	sync          []SplitWebFeature
	expected      []SplitWebFeature
	expectedError error
}

func TestSyncSplitWebFeatures(t *testing.T) {
	testCases := []splitWebFeaturesTestCase{
		{
			name:    "initial sync",
			initial: []SplitWebFeature{},
			sync: []SplitWebFeature{
				{OriginalFeatureKey: "feature-a", TargetFeatureKeys: []string{"feature-b", "feature-c"}},
			},
			expected: []SplitWebFeature{
				{OriginalFeatureKey: "feature-a", TargetFeatureKeys: []string{"feature-b", "feature-c"}},
			},
			expectedError: nil,
		},
		{
			name: "delete one target",
			initial: []SplitWebFeature{
				{OriginalFeatureKey: "feature-a", TargetFeatureKeys: []string{"feature-b", "feature-c"}},
			},
			sync: []SplitWebFeature{
				{OriginalFeatureKey: "feature-a", TargetFeatureKeys: []string{"feature-b"}},
			},
			expected: []SplitWebFeature{
				{OriginalFeatureKey: "feature-a", TargetFeatureKeys: []string{"feature-b"}},
			},
			expectedError: nil,
		},
		{
			name: "add one target",
			initial: []SplitWebFeature{
				{OriginalFeatureKey: "feature-a", TargetFeatureKeys: []string{"feature-b"}},
			},
			sync: []SplitWebFeature{
				{OriginalFeatureKey: "feature-a", TargetFeatureKeys: []string{"feature-b", "feature-c"}},
			},
			expected: []SplitWebFeature{
				{OriginalFeatureKey: "feature-a", TargetFeatureKeys: []string{"feature-b", "feature-c"}},
			},
			expectedError: nil,
		},
		{
			name: "delete original feature",
			initial: []SplitWebFeature{
				{OriginalFeatureKey: "feature-a", TargetFeatureKeys: []string{"feature-b", "feature-c"}},
			},
			sync:          []SplitWebFeature{},
			expected:      []SplitWebFeature{},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runSplitWebFeaturesTestCase(t, tc)
		})
	}
}

func runSplitWebFeaturesTestCase(t *testing.T, tc splitWebFeaturesTestCase) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	// Setup initial web features.
	err := spannerClient.SyncWebFeatures(ctx, []WebFeature{
		{FeatureKey: "feature-b", Name: "Feature B", Description: "", DescriptionHTML: ""},
		{FeatureKey: "feature-c", Name: "Feature C", Description: "", DescriptionHTML: ""},
	})
	if err != nil {
		t.Fatalf("failed to sync web features: %v", err)
	}

	// Setup initial split features.
	if len(tc.initial) > 0 {
		err = spannerClient.SyncSplitWebFeatures(ctx, tc.initial)
		if err != nil {
			t.Fatalf("failed to sync initial split features: %v", err)
		}
	}

	// Perform the sync to test.
	err = spannerClient.SyncSplitWebFeatures(ctx, tc.sync)
	if !errors.Is(err, tc.expectedError) {
		t.Fatalf("expected error %v, got %v", tc.expectedError, err)
	}

	// Verify the result.
	var originalKeys []string
	if len(tc.expected) > 0 {
		for _, f := range tc.expected {
			originalKeys = append(originalKeys, f.OriginalFeatureKey)
		}
	} else if len(tc.initial) > 0 {
		// If we expect empty, check the original keys from the initial state
		for _, f := range tc.initial {
			originalKeys = append(originalKeys, f.OriginalFeatureKey)
		}
	}

	result := make([]SplitWebFeature, 0, len(originalKeys))
	for _, key := range originalKeys {
		feature, err := spannerClient.GetSplitWebFeatureByOriginalFeatureKey(ctx, key)
		if err != nil {
			// If we expect no results, this is okay.
			if errors.Is(err, ErrQueryReturnedNoResults) && len(tc.expected) == 0 {
				continue
			}
			t.Fatalf("failed to get split feature %s: %v", key, err)
		}
		result = append(result, *feature)
	}

	for i := range result {
		sortFeatureKeys(result[0].TargetFeatureKeys)
		sortFeatureKeys(tc.expected[0].TargetFeatureKeys)
		if diff := cmp.Diff(
			tc.expected[i].TargetFeatureKeys,
			result[i].TargetFeatureKeys); diff != "" {
			t.Errorf("unexpected target keys for %s (-want +got): %s", result[i].OriginalFeatureKey, diff)
		}
	}
}

func TestGetSplitWebFeatureByOriginalFeatureKey(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	// Setup initial web features.
	err := spannerClient.SyncWebFeatures(ctx, []WebFeature{
		{FeatureKey: "feature-b", Name: "Feature B", Description: "", DescriptionHTML: ""},
		{FeatureKey: "feature-c", Name: "Feature C", Description: "", DescriptionHTML: ""},
	})
	if err != nil {
		t.Fatalf("failed to sync web features: %v", err)
	}

	// Setup split feature.
	splitFeature := SplitWebFeature{OriginalFeatureKey: "feature-a", TargetFeatureKeys: []string{"feature-b", "feature-c"}}
	err = spannerClient.SyncSplitWebFeatures(ctx, []SplitWebFeature{splitFeature})
	if err != nil {
		t.Fatalf("failed to sync split features: %v", err)
	}

	// Test found case.
	result, err := spannerClient.GetSplitWebFeatureByOriginalFeatureKey(ctx, "feature-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sortFeatureKeys(splitFeature.TargetFeatureKeys)
	sortFeatureKeys(result.TargetFeatureKeys)
	if diff := cmp.Diff(
		splitFeature.TargetFeatureKeys, result.TargetFeatureKeys); diff != "" {
		t.Errorf("unexpected target keys (-want +got): %s", diff)
	}
	if result.OriginalFeatureKey != "feature-a" {
		t.Errorf("expected original feature key 'feature-a', got '%s'", result.OriginalFeatureKey)
	}

	// Test not found case.
	_, err = spannerClient.GetSplitWebFeatureByOriginalFeatureKey(ctx, "non-existent-feature")
	if !errors.Is(err, ErrQueryReturnedNoResults) {
		t.Errorf("expected error %v, got %v", ErrQueryReturnedNoResults, err)
	}
}

func sortFeatureKeys(keys []string) {
	slices.Sort(keys)
}
