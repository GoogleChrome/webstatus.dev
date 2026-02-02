// Copyright 2026 Google LLC
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
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
)

const systemSearchDesc = "system search"

func TestGetSystemManagedSavedSearchByFeatureID(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	featureID := uuid.New().String()
	savedSearchID := uuid.New().String()

	_, err := spannerClient.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		featureMutation, err := spanner.InsertStruct(webFeaturesTable, &SpannerWebFeature{
			ID: featureID,
			WebFeature: WebFeature{
				FeatureKey:      "foo",
				Name:            "Foo",
				Description:     "",
				DescriptionHTML: "",
			},
		})
		if err != nil {
			return err
		}

		desc := systemSearchDesc
		savedSearchMutation, err := spanner.InsertStruct(savedSearchesTable, &SavedSearch{
			ID:          savedSearchID,
			Name:        "my system search",
			Query:       "feature:is(\"foo\")",
			Scope:       SystemManagedScope,
			AuthorID:    "system",
			Description: &desc,
			CreatedAt:   spanner.CommitTimestamp,
			UpdatedAt:   spanner.CommitTimestamp,
		})
		if err != nil {
			return err
		}

		systemManagedSearchMutation, err := spanner.InsertStruct(systemManagedSavedSearchesTable, &SystemManagedSavedSearch{
			FeatureID:     featureID,
			SavedSearchID: savedSearchID,
			CreatedAt:     spanner.CommitTimestamp,
			UpdatedAt:     spanner.CommitTimestamp,
		})
		if err != nil {
			return err
		}

		return txn.BufferWrite([]*spanner.Mutation{featureMutation, savedSearchMutation, systemManagedSearchMutation})
	})
	if err != nil {
		t.Fatalf("unexpected error during insert: %s", err)
	}

	systemManagedSearch, err := spannerClient.GetSystemManagedSavedSearchByFeatureID(ctx, featureID)
	if err != nil {
		t.Fatalf("GetSystemManagedSavedSearchByFeatureID() error = %v", err)
	}

	if systemManagedSearch.FeatureID != featureID {
		t.Errorf("FeatureID = %s; want %s", systemManagedSearch.FeatureID, featureID)
	}
	if systemManagedSearch.SavedSearchID != savedSearchID {
		t.Errorf("SavedSearchID = %s; want %s", systemManagedSearch.SavedSearchID, savedSearchID)
	}
}

func TestUpsertAndListSystemManagedSavedSearches(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	featureID := uuid.New().String()
	savedSearchID := uuid.New().String()

	// 1. Create a feature and a saved search
	_, err := spannerClient.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		featureMutation, err := spanner.InsertStruct(webFeaturesTable, &SpannerWebFeature{
			ID: featureID,
			WebFeature: WebFeature{
				FeatureKey:      "foo",
				Name:            "Foo",
				Description:     "",
				DescriptionHTML: "",
			},
		})
		if err != nil {
			return err
		}

		desc := systemSearchDesc
		savedSearchMutation, err := spanner.InsertStruct(savedSearchesTable, &SavedSearch{
			ID:          savedSearchID,
			Name:        "my system search",
			Query:       "feature:is(\"foo\")",
			Scope:       SystemManagedScope,
			AuthorID:    "system",
			Description: &desc,
			CreatedAt:   spanner.CommitTimestamp,
			UpdatedAt:   spanner.CommitTimestamp,
		})
		if err != nil {
			return err
		}

		return txn.BufferWrite([]*spanner.Mutation{featureMutation, savedSearchMutation})
	})
	if err != nil {
		t.Fatalf("unexpected error during insert: %s", err)
	}

	// 2. Create a SystemManagedSavedSearch.
	systemManagedSearch := SystemManagedSavedSearch{
		FeatureID:     featureID,
		SavedSearchID: savedSearchID,
		CreatedAt:     spanner.CommitTimestamp,
		UpdatedAt:     spanner.CommitTimestamp,
	}

	// 3. Call UpsertSystemManagedSavedSearch.
	err = spannerClient.UpsertSystemManagedSavedSearch(ctx, systemManagedSearch)
	if err != nil {
		t.Fatalf("UpsertSystemManagedSavedSearch() error = %v", err)
	}

	// 4. Call ListAllSystemManagedSavedSearches and verify the new system managed search is in the list.
	systemManagedSearches, err := spannerClient.ListAllSystemManagedSavedSearches(ctx)
	if err != nil {
		t.Fatalf("ListAllSystemManagedSavedSearches() error = %v", err)
	}

	expectedState := []SystemManagedSavedSearch{systemManagedSearch}
	// Ignore server-side timestamps.
	opts := cmpopts.IgnoreFields(SystemManagedSavedSearch{
		FeatureID:     "",
		SavedSearchID: "",
		CreatedAt:     time.Time{},
		UpdatedAt:     time.Time{},
	}, "CreatedAt", "UpdatedAt")
	if diff := cmp.Diff(expectedState, systemManagedSearches, opts); diff != "" {
		t.Errorf("Mismatch in system managed saved searches (-want +got):\n%s", diff)
	}

	// 5. Call DeleteSystemManagedSavedSearch.
	err = spannerClient.DeleteSystemManagedSavedSearch(ctx, featureID)
	if err != nil {
		t.Fatalf("DeleteSystemManagedSavedSearch() error = %v", err)
	}

	// 6. Call ListAllSystemManagedSavedSearches and verify the system managed search is gone.
	systemManagedSearches, err = spannerClient.ListAllSystemManagedSavedSearches(ctx)
	if err != nil {
		t.Fatalf("ListAllSystemManagedSavedSearches() error = %v", err)
	}

	if len(systemManagedSearches) != 0 {
		t.Fatalf("len(systemManagedSearches) = %d; want 0", len(systemManagedSearches))
	}
}

func TestSyncSystemManagedSavedQuery(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	// 1. Setup initial state: 2 features
	feature1ID := uuid.New().String()
	feature2ID := uuid.New().String()

	_, err := spannerClient.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		m1, _ := spanner.InsertStruct(webFeaturesTable, &SpannerWebFeature{
			ID: feature1ID,
			WebFeature: WebFeature{
				FeatureKey:      "f1",
				Name:            "Feature 1",
				Description:     "",
				DescriptionHTML: "",
			},
		})
		m2, _ := spanner.InsertStruct(webFeaturesTable, &SpannerWebFeature{
			ID: feature2ID,
			WebFeature: WebFeature{
				FeatureKey:      "f2",
				Name:            "Feature 2",
				Description:     "",
				DescriptionHTML: "",
			},
		})

		return txn.BufferWrite([]*spanner.Mutation{m1, m2})
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// 2. Sync. Should create 2 saved searches and 2 mappings.
	err = spannerClient.SyncSystemManagedSavedQuery(ctx)
	if err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	mappings, err := spannerClient.ListAllSystemManagedSavedSearches(ctx)
	if err != nil || len(mappings) != 2 {
		t.Fatalf("expected 2 mappings, got %d (err: %v)", len(mappings), err)
	}

	// Verify one search
	m1, _ := spannerClient.GetSystemManagedSavedSearchByFeatureID(ctx, feature1ID)
	m2, _ := spannerClient.GetSystemManagedSavedSearchByFeatureID(ctx, feature2ID)
	savedSearch2ID := m2.SavedSearchID

	s1, _ := spannerClient.GetSavedSearch(ctx, m1.SavedSearchID)
	if s1.Name != systemSavedSearchName("f1") {
		t.Errorf("expected name %s, got %s", systemSavedSearchName("f1"), s1.Name)
	}

	// 3. Update a feature key. Sync should update the saved search.
	_, err = spannerClient.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		m, _ := spanner.UpdateStruct(webFeaturesTable, &SpannerWebFeature{
			ID: feature1ID,
			WebFeature: WebFeature{
				FeatureKey:      "f1-new",
				Name:            "Feature 1 New",
				Description:     "",
				DescriptionHTML: "",
			},
		})

		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	err = spannerClient.SyncSystemManagedSavedQuery(ctx)
	if err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	s1updated, _ := spannerClient.GetSavedSearch(ctx, m1.SavedSearchID)
	if s1updated.Name != systemSavedSearchName("f1-new") {
		t.Errorf("expected updated name %s, got %s", systemSavedSearchName("f1-new"), s1updated.Name)
	}
	if s1updated.Query != systemSavedSearchQuery("f1-new") {
		t.Errorf("expected updated query %s, got %s", systemSavedSearchQuery("f1-new"), s1updated.Query)
	}

	// 4. Delete a feature. Sync should remove the mapping and the saved search.
	_, err = spannerClient.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{spanner.Delete(webFeaturesTable, spanner.Key{feature2ID})})
	})
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	err = spannerClient.SyncSystemManagedSavedQuery(ctx)
	if err != nil {
		t.Fatalf("third sync failed: %v", err)
	}

	mappingsFinal, _ := spannerClient.ListAllSystemManagedSavedSearches(ctx)
	if len(mappingsFinal) != 1 {
		t.Errorf("expected 1 mapping remaining, got %d", len(mappingsFinal))
	}
	if mappingsFinal[0].FeatureID != feature1ID {
		t.Errorf("expected feature 1 mapping to remain, got %s", mappingsFinal[0].FeatureID)
	}

	// Verify saved search 1 remains
	_, err = spannerClient.GetSavedSearch(ctx, m1.SavedSearchID)
	if err != nil {
		t.Errorf("expected saved search 1 to remain, got err %v", err)
	}

	// Verify saved search 2 is gone
	_, err = spannerClient.GetSavedSearch(ctx, savedSearch2ID)
	if !errors.Is(err, ErrQueryReturnedNoResults) {
		t.Errorf("expected saved search 2 to be deleted, got err %v", err)
	}
}
