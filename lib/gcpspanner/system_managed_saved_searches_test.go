// Copyright 2024 Google LLC
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
