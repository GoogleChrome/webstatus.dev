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
	"cmp"
	"context"
	"slices"
	"sort"
	"testing"

	gcmp "github.com/google/go-cmp/cmp"
)

func setupRequiredTablesForDeveloperSignals(t *testing.T) {
	ctx := context.Background()
	sampleFeatures := getSampleFeatures()
	err := spannerClient.SyncWebFeatures(ctx, sampleFeatures)
	if err != nil {
		t.Fatalf("unexpected error during sync. %s", err.Error())
	}
}

func getSampleFeatureDeveloperSignals() []FeatureDeveloperSignal {
	return []FeatureDeveloperSignal{
		{
			WebFeatureKey: "feature1",
			Link:          "https://example.com/feature1",
			Votes:         100,
		},
		{
			WebFeatureKey: "feature2",
			Link:          "https://example.com/feature2",
			Votes:         200,
		},
	}
}

func TestSyncAndGetAllLatestFeatureDeveloperSignals(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	setupRequiredTablesForDeveloperSignals(t)

	// 1. Initial Sync
	initialSignals := getSampleFeatureDeveloperSignals()
	err := spannerClient.SyncLatestFeatureDeveloperSignals(ctx, initialSignals)
	if err != nil {
		t.Fatalf("unexpected error during initial sync: %v", err)
	}

	// 2. Verify initial data with GetAll
	retrievedSignals, err := spannerClient.GetAllLatestFeatureDeveloperSignals(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting all signals: %v", err)
	}

	sortFn := func(a, b FeatureDeveloperSignal) int {
		return cmp.Compare(a.WebFeatureKey, b.WebFeatureKey)
	}
	slices.SortFunc(retrievedSignals, sortFn)

	if diff := gcmp.Diff(initialSignals, retrievedSignals); diff != "" {
		t.Errorf("retrieved signals mismatch after initial sync (-want +got):\n%s", diff)
	}

	// 3. Second Sync (Update, Insert, Delete)
	updatedSignals := []FeatureDeveloperSignal{
		{
			WebFeatureKey: "feature1",
			Votes:         150, // Update
			Link:          "https://example.com/feature1-updated",
		},
		// feature2 is deleted
		{
			WebFeatureKey: "feature3", // Insert
			Votes:         300,
			Link:          "https://example.com/feature3",
		},
	}

	err = spannerClient.SyncLatestFeatureDeveloperSignals(ctx, updatedSignals)
	if err != nil {
		t.Fatalf("unexpected error during second sync: %v", err)
	}

	// 4. Verify updated data
	retrievedSignals, err = spannerClient.GetAllLatestFeatureDeveloperSignals(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting all signals after update: %v", err)
	}

	sort.Slice(retrievedSignals, func(i, j int) bool {
		return retrievedSignals[i].WebFeatureKey < retrievedSignals[j].WebFeatureKey
	})

	expectedSignalsAfterUpdate := []FeatureDeveloperSignal{
		{
			WebFeatureKey: "feature1",
			Link:          "https://example.com/feature1-updated",
			Votes:         150,
		},
		{
			WebFeatureKey: "feature3",
			Link:          "https://example.com/feature3",
			Votes:         300,
		},
	}

	if diff := gcmp.Diff(expectedSignalsAfterUpdate, retrievedSignals); diff != "" {
		t.Errorf("retrieved signals mismatch after second sync (-want +got):\n%s", diff)
	}

	// 5. Test Sync with non-existent feature key
	signalsWithInvalidFeature := []FeatureDeveloperSignal{
		{
			WebFeatureKey: "non-existent-feature",
			Link:          "https://example.com/non-existent-feature",
			Votes:         999,
		},
	}
	err = spannerClient.SyncLatestFeatureDeveloperSignals(ctx, signalsWithInvalidFeature)
	if err == nil {
		t.Error("expected an error when syncing with a non-existent feature key, but got nil")
	}

	// 6. Verify that the data was not modified by the failed sync
	retrievedSignalsAfterFailure, err := spannerClient.GetAllLatestFeatureDeveloperSignals(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting all signals after failed sync: %v", err)
	}
	slices.SortFunc(retrievedSignalsAfterFailure, sortFn)

	if diff := gcmp.Diff(expectedSignalsAfterUpdate, retrievedSignalsAfterFailure); diff != "" {
		t.Errorf("data should not have changed after a failed sync (-want +got):\n%s", diff)
	}

	// 7. Test Empty Sync (deletes all)
	err = spannerClient.SyncLatestFeatureDeveloperSignals(ctx, []FeatureDeveloperSignal{})
	if err != nil {
		t.Fatalf("unexpected error during empty sync: %v", err)
	}

	// 8. Verify all deleted
	retrievedSignalsAfterEmptySync, err := spannerClient.GetAllLatestFeatureDeveloperSignals(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting all signals after empty sync: %v", err)
	}
	if len(retrievedSignalsAfterEmptySync) != 0 {
		t.Errorf("expected 0 signals after empty sync, but got %d", len(retrievedSignalsAfterEmptySync))
	}
}
