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
	"cmp"
	"context"
	"errors"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const webFeaturesTable = "WebFeatures"

// SpannerWebFeature is a wrapper for the feature that is actually
// stored in spanner. This is useful because the spanner id is not useful to
// return to the end user since it is only used to decouple the primary keys
// between this system and web features repo.
type SpannerWebFeature struct {
	ID string `spanner:"ID"`
	WebFeature
}

// WebFeature contains common metadata for a Web Feature.
// Columns come from the ../../infra/storage/spanner/migrations/*.sql files.
type WebFeature struct {
	FeatureKey string `spanner:"FeatureKey"`
	Name       string `spanner:"Name"`
}

// UpsertWebFeature will upsert the given web feature.
// If the feature, does not exist, it will insert a new feature.
// If the run exists, it will only update the name.
func (c *Client) UpsertWebFeature(ctx context.Context, feature WebFeature) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.NewStatement(`
		SELECT
			ID, FeatureKey, Name
		FROM WebFeatures
		WHERE FeatureKey = @featureKey
		LIMIT 1`)
		parameters := map[string]interface{}{
			"featureKey": feature.FeatureKey,
		}
		stmt.Params = parameters

		// Attempt to query for the row.
		it := txn.Query(ctx, stmt)
		defer it.Stop()
		var m *spanner.Mutation

		row, err := it.Next()
		// nolint: nestif // TODO: fix in the future.
		if err != nil {
			if errors.Is(err, iterator.Done) {
				// No rows returned. Act as if this is an insertion.
				var err error
				m, err = spanner.InsertOrUpdateStruct(webFeaturesTable, feature)
				if err != nil {
					return errors.Join(ErrInternalQueryFailure, err)
				}
			} else {
				// An unexpected error occurred.

				return errors.Join(ErrInternalQueryFailure, err)
			}
		} else {
			// Read the existing feature and merge the values.
			var existingFeature SpannerWebFeature
			err = row.ToStruct(&existingFeature)
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
			// Only allow overriding of the feature name.
			existingFeature.Name = cmp.Or[string](feature.Name, existingFeature.Name)
			m, err = spanner.InsertOrUpdateStruct(webFeaturesTable, existingFeature)
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
		}
		// Buffer the mutation to be committed.
		err = txn.BufferWrite([]*spanner.Mutation{m})
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}

		return nil
	})
	if err != nil {
		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}
