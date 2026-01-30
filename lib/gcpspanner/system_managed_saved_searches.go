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
	"errors"
	"log/slog"
	"time"

	"cloud.google.com/go/spanner"
)

const systemManagedSavedSearchesTable = "SystemManagedSavedSearches"

// SystemManagedSavedSearch represents a row in the SystemManagedSavedSearches table.
type SystemManagedSavedSearch struct {
	FeatureID     string    `spanner:"FeatureID"`
	SavedSearchID string    `spanner:"SavedSearchID"`
	CreatedAt     time.Time `spanner:"CreatedAt"`
	UpdatedAt     time.Time `spanner:"UpdatedAt"`
}

// systemManagedSavedSearchMapper implements the necessary interfaces for the generic helpers.
type systemManagedSavedSearchMapper struct{}

func (m systemManagedSavedSearchMapper) Table() string {
	return systemManagedSavedSearchesTable
}

func (m systemManagedSavedSearchMapper) SelectAll() spanner.Statement {
	return spanner.NewStatement(
		"SELECT FeatureID, SavedSearchID, CreatedAt, UpdatedAt FROM " + systemManagedSavedSearchesTable)
}

func (m systemManagedSavedSearchMapper) SelectOne(featureID string) spanner.Statement {
	stmt := spanner.NewStatement(`
		SELECT FeatureID, SavedSearchID, CreatedAt, UpdatedAt
		FROM SystemManagedSavedSearches
		WHERE FeatureID = @featureID
	`)
	stmt.Params["featureID"] = featureID

	return stmt
}

func (m systemManagedSavedSearchMapper) SelectAllByKeys(featureIDs []string) spanner.Statement {
	stmt := spanner.NewStatement(`
		SELECT FeatureID, SavedSearchID, CreatedAt, UpdatedAt
		FROM SystemManagedSavedSearches
		WHERE FeatureID IN UNNEST(@featureIDs)
	`)
	stmt.Params["featureIDs"] = featureIDs

	return stmt
}

func (m systemManagedSavedSearchMapper) GetKeyFromExternal(in SystemManagedSavedSearch) string {
	return in.FeatureID
}

func (m systemManagedSavedSearchMapper) Merge(
	in SystemManagedSavedSearch,
	existing SystemManagedSavedSearch,
) SystemManagedSavedSearch {
	if in.SavedSearchID != "" {
		existing.SavedSearchID = in.SavedSearchID
	}

	return existing
}

func (m systemManagedSavedSearchMapper) DeleteKey(featureID string) spanner.Key {
	return spanner.Key{featureID}
}

func (m systemManagedSavedSearchMapper) GetKeyFromInternal(in SystemManagedSavedSearch) string {
	return in.FeatureID
}

func (m systemManagedSavedSearchMapper) GetChildDeleteKeyMutations(
	_ context.Context,
	_ *Client,
	_ []SystemManagedSavedSearch,
) ([]ExtraMutationsGroup, error) {
	return nil, nil
}

func (m systemManagedSavedSearchMapper) PreDeleteHook(
	_ context.Context, _ *Client, _ []SystemManagedSavedSearch) ([]ExtraMutationsGroup, error) {
	return nil, nil
}

func (m systemManagedSavedSearchMapper) DeleteMutation(
	in SystemManagedSavedSearch) *spanner.Mutation {
	return spanner.Delete(m.Table(), spanner.Key{in.FeatureID})
}

// ListAllSystemManagedSavedSearches returns all system managed saved searches.
func (c *Client) ListAllSystemManagedSavedSearches(
	ctx context.Context) ([]SystemManagedSavedSearch, error) {
	return newAllEntityReader[systemManagedSavedSearchMapper, SystemManagedSavedSearch](c).readAll(ctx)
}

// UpsertSystemManagedSavedSearch inserts or updates a system managed saved search.
func (c *Client) UpsertSystemManagedSavedSearch(
	ctx context.Context,
	systemManagedSearch SystemManagedSavedSearch,
) error {
	return newEntityWriter[systemManagedSavedSearchMapper, SystemManagedSavedSearch, SystemManagedSavedSearch, string](c).
		upsert(ctx, systemManagedSearch)
}

// DeleteSystemManagedSavedSearch deletes a system managed saved search.
func (c *Client) DeleteSystemManagedSavedSearch(
	ctx context.Context,
	featureID string,
) error {
	return newEntityRemover[systemManagedSavedSearchMapper,
		SystemManagedSavedSearch,
		SystemManagedSavedSearch, string](c).
		remove(ctx, SystemManagedSavedSearch{
			FeatureID:     featureID,
			SavedSearchID: "",
			CreatedAt:     spanner.CommitTimestamp,
			UpdatedAt:     spanner.CommitTimestamp})
}

func (c *Client) GetSystemManagedSavedSearchByFeatureID(
	ctx context.Context,
	featureID string,
) (*SystemManagedSavedSearch, error) {
	return newEntityReader[systemManagedSavedSearchMapper, SystemManagedSavedSearch, string](c).
		readRowByKey(ctx, featureID)
}

func (c *Client) getSystemManagedSavedSearchByFeatureIDAndTransaction(
	ctx context.Context,
	txn transaction,
	featureID string,
) (*SystemManagedSavedSearch, error) {
	return newEntityReader[systemManagedSavedSearchMapper, SystemManagedSavedSearch, string](c).
		readRowByKeyWithTransaction(ctx, featureID, txn)
}

// ListSystemManagedSavedSearchesByFeatureIDs returns the mappings for the given feature IDs.
// It processes the IDs in chunks to stay well within Spanner statement size limits.
func (c *Client) ListSystemManagedSavedSearchesByFeatureIDs(
	ctx context.Context,
	featureIDs []string,
) ([]SystemManagedSavedSearch, error) {
	if len(featureIDs) == 0 {
		return nil, nil
	}

	const chunkSize = 1000
	var allResults []SystemManagedSavedSearch

	for i := 0; i < len(featureIDs); i += chunkSize {
		end := i + chunkSize
		if end > len(featureIDs) {
			end = len(featureIDs)
		}
		chunk := featureIDs[i:end]

		stmt := systemManagedSavedSearchMapper{}.SelectAllByKeys(chunk)
		iter := c.Single().Query(ctx, stmt)

		err := iter.Do(func(row *spanner.Row) error {
			var res SystemManagedSavedSearch
			if err := row.ToStruct(&res); err != nil {
				return err
			}
			allResults = append(allResults, res)

			return nil
		})
		iter.Stop() // Ensure iterator is stopped after each chunk

		if err != nil {
			return nil, err
		}
	}

	return allResults, nil
}

// SyncSystemManagedSavedQuery ensures that every WebFeature has a corresponding system-managed saved search.
// It also updates existing searches if the feature key has changed and removes searches for deleted features.
func (c *Client) SyncSystemManagedSavedQuery(ctx context.Context) error {
	slog.InfoContext(ctx, "Starting system-managed saved query synchronization")

	features, err := c.listAllWebFeaturesForSync(ctx)
	if err != nil {
		return err
	}

	existingMappings, err := c.ListAllSystemManagedSavedSearches(ctx)
	if err != nil {
		return err
	}
	mappingMap := make(map[string]SystemManagedSavedSearch)
	for _, m := range existingMappings {
		mappingMap[m.FeatureID] = m
	}

	// 3. Identify missing or outdated searches
	// Pre-allocate mutations. At most, we might have 1 mutation per feature (create/update)
	// plus the orphan cleanup.
	mutations := make([]*spanner.Mutation, 0, len(features))
	for _, f := range features {
		mapping, found := mappingMap[f.ID]
		ms, err := c.reconcileSystemManagedSavedSearch(ctx, f, mapping, found)
		if err != nil {
			return err
		}
		mutations = append(mutations, ms...)
	}

	// 4. Identify and delete orphaned system-managed saved searches.
	// These are searches with SYSTEM_MANAGED scope that are no longer linked to a feature.
	// This covers both features deleted by SyncWebFeatures (handled by the loop above)
	// and features deleted directly from the WebFeatures table (handled by DB cascade on mapping,
	// but leaving the SavedSearch itself).
	orphanedSearchIDs, err := c.findOrphanedSystemManagedSavedSearchIDs(ctx)
	if err != nil {
		return err
	}

	for _, id := range orphanedSearchIDs {
		mutations = append(mutations, spanner.Delete(savedSearchesTable, spanner.Key{id}))
	}

	_, err = c.Apply(ctx, mutations)

	return err
}

func (c *Client) reconcileSystemManagedSavedSearch(
	ctx context.Context,
	f SpannerWebFeatureSyncInfo,
	mapping SystemManagedSavedSearch,
	found bool,
) ([]*spanner.Mutation, error) {
	if !found {
		// Missing search - create it.
		var mapper webFeatureSpannerMapper

		return mapper.createSystemManagedSavedSearchMutations(f.ID, WebFeature{
			FeatureKey:      f.FeatureKey,
			Name:            f.Name,
			Description:     "",
			DescriptionHTML: "",
		})
	}

	// Existing search. Check if it needs updating (e.g. name or query change due to feature key change).
	savedSearch, err := c.GetSavedSearch(ctx, mapping.SavedSearchID)
	if err != nil {
		if errors.Is(err, ErrQueryReturnedNoResults) {
			// Orphaned mapping. Re-create the search.
			var mapper webFeatureSpannerMapper

			return mapper.createSystemManagedSavedSearchMutations(f.ID, WebFeature{
				FeatureKey:      f.FeatureKey,
				Name:            f.Name,
				Description:     "",
				DescriptionHTML: "",
			})
		}

		return nil, err
	}

	expectedName := systemSavedSearchName(f.FeatureKey)
	expectedQuery := systemSavedSearchQuery(f.FeatureKey)
	if savedSearch.Name != expectedName || savedSearch.Query != expectedQuery {
		savedSearch.Name = expectedName
		savedSearch.Query = expectedQuery
		savedSearch.UpdatedAt = spanner.CommitTimestamp
		m, err := spanner.UpdateStruct(savedSearchesTable, savedSearch)
		if err != nil {
			return nil, err
		}

		return []*spanner.Mutation{m}, nil
	}

	return nil, nil
}

func (c *Client) findOrphanedSystemManagedSavedSearchIDs(ctx context.Context) ([]string, error) {
	stmt := spanner.NewStatement(`
		SELECT s.ID
		FROM SavedSearches s
		LEFT JOIN SystemManagedSavedSearches m ON s.ID = m.SavedSearchID
		WHERE s.Scope = @scope AND m.SavedSearchID IS NULL
	`)
	stmt.Params["scope"] = string(SystemManagedScope)

	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ids []string
	err := iter.Do(func(r *spanner.Row) error {
		var id string
		if err := r.Column(0, &id); err != nil {
			return err
		}
		ids = append(ids, id)

		return nil
	})

	return ids, err
}
