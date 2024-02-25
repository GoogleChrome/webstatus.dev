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
	FeatureID string `spanner:"FeatureID"`
	Name      string `spanner:"Name"`
}

// UpsertWebFeature will insert the given web feature.
// If the feature, does not exist, it will insert a new feature.
// If the run exists, it will at most update the name.
func (c *Client) UpsertWebFeature(ctx context.Context, feature WebFeature) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.NewStatement(`
		SELECT
			ID, FeatureID, Name
		FROM WebFeatures
		WHERE FeatureID = @featureID
		LIMIT 1`)
		parameters := map[string]interface{}{
			"featureID": feature.FeatureID,
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
