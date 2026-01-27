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
