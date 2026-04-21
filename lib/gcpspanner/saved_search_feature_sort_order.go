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
	"fmt"

	"cloud.google.com/go/spanner"
)

const savedSearchFeatureSortOrderTable = "SavedSearchFeatureSortOrder"

// SpannerSavedSearchFeatureSortOrder represents a row in the SavedSearchFeatureSortOrder table.
type SpannerSavedSearchFeatureSortOrder struct {
	SavedSearchID string `spanner:"SavedSearchID"`
	FeatureKey    string `spanner:"FeatureKey"`
	PositionIndex int64  `spanner:"PositionIndex"`
}

type savedSearchFeatureSortOrderMapper struct{}

func (m savedSearchFeatureSortOrderMapper) Table() string {
	return savedSearchFeatureSortOrderTable
}

// SelectAllByKeys returns a statement to query by FeatureKey.
// Implements readAllByKeysMapper[string].
func (m savedSearchFeatureSortOrderMapper) SelectAllByKeys(featureKey string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
		SELECT SavedSearchID, FeatureKey, PositionIndex
		FROM %s
		WHERE FeatureKey = @featureKey`, m.Table()))
	stmt.Params = map[string]any{
		"featureKey": featureKey,
	}

	return stmt
}

// InsertOrUpdateMutation returns a mutation to insert or update a row.
func (m savedSearchFeatureSortOrderMapper) InsertOrUpdateMutation(item SpannerSavedSearchFeatureSortOrder) (
	*spanner.Mutation, error) {
	return spanner.InsertOrUpdateStruct(m.Table(), item)
}

// DeleteMutation returns a mutation to delete a row.
func (m savedSearchFeatureSortOrderMapper) DeleteMutation(savedSearchID string, featureKey string) *spanner.Mutation {
	return spanner.Delete(m.Table(), spanner.Key{savedSearchID, featureKey})
}

// Client methods to be used by consumers or hooks.

// GetSavedSearchFeatureSortOrderByFeatureKey fetches rows for a feature key using a transaction.
func (c *Client) GetSavedSearchFeatureSortOrderByFeatureKey(
	ctx context.Context,
	txn *spanner.ReadOnlyTransaction,
	featureKey string,
) ([]SpannerSavedSearchFeatureSortOrder, error) {
	reader := newAllByKeysEntityReader[savedSearchFeatureSortOrderMapper, string, SpannerSavedSearchFeatureSortOrder](c)

	return reader.readAllByKeysWithTransaction(ctx, featureKey, txn)
}
