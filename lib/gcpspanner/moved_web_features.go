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
	"log/slog"

	"cloud.google.com/go/spanner"
)

const movedFeaturesTable = "MovedWebFeatures"

type MovedWebFeature struct {
	OriginalFeatureKey string `spanner:"OriginalFeatureKey"`
	NewFeatureKey      string `spanner:"NewFeatureKey"`
}

type spannerNewMovedWebFeature struct {
	OriginalFeatureKey string `spanner:"OriginalFeatureKey"`
	TargetWebFeatureID string `spanner:"TargetWebFeatureID"`
}

type spannerMovedWebFeature struct {
	ID                 string `spanner:"ID"`
	OriginalFeatureKey string `spanner:"OriginalFeatureKey"`
	TargetWebFeatureID string `spanner:"TargetWebFeatureID"`
}

type movedWebFeatureMapper struct{}

func (m movedWebFeatureMapper) SelectAll() spanner.Statement {
	return spanner.NewStatement(
		`SELECT
			ID,
			OriginalFeatureKey,
			TargetWebFeatureID
		FROM MovedWebFeatures`)
}

// MergeAndCheckChanged will merge the entity and return if the entity has changed.
func (m movedWebFeatureMapper) MergeAndCheckChanged(
	// Right now, we treat these as immutable for now. Differences should yield a different entity
	_ spannerNewMovedWebFeature, existing spannerMovedWebFeature) (spannerMovedWebFeature, bool) {
	return existing, false
}

func (m movedWebFeatureMapper) PreDeleteHook(
	_ context.Context,
	_ *Client,
	_ []spannerMovedWebFeature,
) ([]ExtraMutationsGroup, error) {
	return nil, nil
}

func (m movedWebFeatureMapper) DeleteMutation(in spannerMovedWebFeature) *spanner.Mutation {
	return spanner.Delete(movedFeaturesTable, spanner.Key{in.ID})
}

func (m movedWebFeatureMapper) GetKeyFromExternal(in spannerNewMovedWebFeature) string {
	return in.OriginalFeatureKey
}

func (m movedWebFeatureMapper) GetKeyFromInternal(in spannerMovedWebFeature) string {
	return in.ID
}

func (m movedWebFeatureMapper) GetChildDeleteKeyMutations(
	_ context.Context, _ *Client, _ []spannerMovedWebFeature) ([]ExtraMutationsGroup, error) {
	return nil, nil
}

func (m movedWebFeatureMapper) Table() string {
	return movedFeaturesTable
}

// SyncMovedWebFeatures reconciles the MovedWebFeatures table with the provided list of features.
// It will insert new details for moved web features, update existing ones, and delete any moved web features
// that are no longer present in the provided list.
func (c *Client) SyncMovedWebFeatures(ctx context.Context, movedWebFeatures []MovedWebFeature) error {
	slog.InfoContext(ctx, "Starting moved web features synchronization")
	synchronizer := newEntitySynchronizer[movedWebFeatureMapper](c)

	spannerMovedWebFeatures := make([]spannerNewMovedWebFeature, 0, len(movedWebFeatures))
	for _, movedWebFeature := range movedWebFeatures {
		// Get the web feature id from the target feature key.
		targetWebFeatureID, err := c.GetIDFromFeatureKey(ctx, NewFeatureKeyFilter(movedWebFeature.NewFeatureKey))
		if err != nil {
			return err
		}

		spannerMovedWebFeatures = append(spannerMovedWebFeatures, spannerNewMovedWebFeature{
			OriginalFeatureKey: movedWebFeature.OriginalFeatureKey,
			TargetWebFeatureID: *targetWebFeatureID,
		})
	}

	return synchronizer.Sync(ctx, spannerMovedWebFeatures)
}

type movedWebFeatureByOriginalKeyMapper struct{}

func (m movedWebFeatureByOriginalKeyMapper) SelectOne(featureKey string) spanner.Statement {
	stmt := spanner.NewStatement(
		`SELECT
			mwf.OriginalFeatureKey,
			wf.FeatureKey AS NewFeatureKey
		FROM
			MovedWebFeatures AS mwf
		JOIN
			WebFeatures AS wf ON mwf.TargetWebFeatureID = wf.ID
		WHERE
			OriginalFeatureKey = @OriginalFeatureKey`)
	stmt.Params["OriginalFeatureKey"] = featureKey

	return stmt
}

// GetMovedWebFeatureDetailsByOriginalFeatureKey returns the details about the moved feature.
// If details are not found for the feature key, it returns ErrQueryReturnedNoResults.
// Other errors should be investigated and handled appropriately.
func (c *Client) GetMovedWebFeatureDetailsByOriginalFeatureKey(
	ctx context.Context, originalFeatureKey string) (*MovedWebFeature, error) {
	feature, err := newEntityReader[
		movedWebFeatureByOriginalKeyMapper,
		MovedWebFeature,
		string,
	](c).readRowByKey(ctx, originalFeatureKey)
	if err != nil {
		return nil, err
	}

	return feature, nil
}

type movedWebFeaturesGetAllMapper struct{}

func (m movedWebFeaturesGetAllMapper) SelectAll() spanner.Statement {
	return spanner.NewStatement(
		`SELECT
			mwf.OriginalFeatureKey,
			wf.FeatureKey AS NewFeatureKey
		FROM
			MovedWebFeatures AS mwf
		JOIN
			WebFeatures AS wf ON mwf.TargetWebFeatureID = wf.ID`)
}

func (c *Client) GetAllMovedWebFeatures(ctx context.Context) ([]MovedWebFeature, error) {
	return newAllEntityReader[movedWebFeaturesGetAllMapper, MovedWebFeature](c).readAll(ctx)

}
