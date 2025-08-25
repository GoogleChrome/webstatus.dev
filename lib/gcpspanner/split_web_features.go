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

const splitWebFeaturesTable = "SplitWebFeatures"

type SplitWebFeature struct {
	OriginalFeatureKey string   `spanner:"OriginalFeatureKey"`
	TargetFeatureKeys  []string `spanner:"TargetFeatureKeys"`
}

type spannerNewSplitWebFeature struct {
	OriginalFeatureKey string `spanner:"OriginalFeatureKey"`
	TargetWebFeatureID string `spanner:"TargetWebFeatureID"`
}

type spannerSplitWebFeature struct {
	ID                 string `spanner:"ID"`
	OriginalFeatureKey string `spanner:"OriginalFeatureKey"`
	TargetWebFeatureID string `spanner:"TargetWebFeatureID"`
}

type splitWebFeatureMapper struct{}

func (m splitWebFeatureMapper) SelectAll() spanner.Statement {
	return spanner.NewStatement(
		`SELECT
			ID,
			OriginalFeatureKey,
			TargetWebFeatureID
		FROM SplitWebFeatures`)
}

func (m splitWebFeatureMapper) MergeAndCheckChanged(
	_ spannerNewSplitWebFeature, existing spannerSplitWebFeature) (spannerSplitWebFeature, bool) {
	// Right now, we treat these as immutable for now. Differences should yield a different entity
	return existing, false
}

func (m splitWebFeatureMapper) PreDeleteHook(
	_ context.Context,
	_ *Client,
	_ []spannerSplitWebFeature,
) ([]ExtraMutationsGroup, error) {
	return nil, nil
}

func (m splitWebFeatureMapper) GetKeyFromExternal(in spannerNewSplitWebFeature) splitWebFeatureKey {
	return splitWebFeatureKey(in)
}

func (m splitWebFeatureMapper) GetKeyFromInternal(in spannerSplitWebFeature) splitWebFeatureKey {
	return splitWebFeatureKey{
		OriginalFeatureKey: in.OriginalFeatureKey,
		TargetWebFeatureID: in.TargetWebFeatureID,
	}
}

func (m splitWebFeatureMapper) GetChildDeleteKeyMutations(
	_ context.Context, _ *Client, _ []spannerSplitWebFeature) ([]ExtraMutationsGroup, error) {
	return nil, nil
}

func (m splitWebFeatureMapper) Table() string {
	return splitWebFeaturesTable
}

func (m splitWebFeatureMapper) DeleteMutation(in spannerSplitWebFeature) *spanner.Mutation {
	return spanner.Delete(splitWebFeaturesTable, spanner.Key{in.ID})
}

type splitWebFeatureKey struct {
	OriginalFeatureKey string
	TargetWebFeatureID string
}

// SyncSplitWebFeatures reconciles the SplitWebFeatures table with the provided list of features.
// It will insert new details for split web features, update existing ones, and delete any split web features
// that are in the database but not in the provided list.
func (c *Client) SyncSplitWebFeatures(ctx context.Context, splitWebFeatures []SplitWebFeature) error {
	slog.InfoContext(ctx, "Starting split web features synchronization")
	spannerSplitWebFeatures := []spannerNewSplitWebFeature{}
	for _, splitWebFeatures := range splitWebFeatures {
		for _, targetFeatureKey := range splitWebFeatures.TargetFeatureKeys {
			// Get the web feature id from the target feature key.
			targetWebFeatureID, err := c.GetIDFromFeatureKey(ctx, NewFeatureKeyFilter(targetFeatureKey))
			if err != nil {
				return err
			}
			spannerSplitWebFeatures = append(spannerSplitWebFeatures, spannerNewSplitWebFeature{
				OriginalFeatureKey: splitWebFeatures.OriginalFeatureKey,
				TargetWebFeatureID: *targetWebFeatureID,
			})
		}
	}

	synchronizer := newEntitySynchronizer[splitWebFeatureMapper](c)

	return synchronizer.Sync(ctx, spannerSplitWebFeatures)
}

type splitWebFeatureByOriginalKeyMapper struct{}

func (m splitWebFeatureByOriginalKeyMapper) SelectOne(featureKey string) spanner.Statement {
	stmt := spanner.NewStatement(
		`SELECT
			swf.OriginalFeatureKey,
			ARRAY_AGG(wf.FeatureKey) AS TargetFeatureKeys
		FROM
			SplitWebFeatures AS swf
		JOIN
			WebFeatures AS wf ON swf.TargetWebFeatureID = wf.ID
		WHERE
			swf.OriginalFeatureKey = @OriginalFeatureKey
		GROUP BY
			swf.OriginalFeatureKey`)
	stmt.Params["OriginalFeatureKey"] = featureKey

	return stmt
}

// GetSplitWebFeatureByOriginalFeatureKey returns the details about the split feature.
// If details are not found for the feature key, it returns ErrQueryReturnedNoResults.
// Other errors should be investigated and handled appropriately.
func (c *Client) GetSplitWebFeatureByOriginalFeatureKey(
	ctx context.Context, featureKey string) (*SplitWebFeature, error) {
	return newEntityReader[splitWebFeatureByOriginalKeyMapper, SplitWebFeature, string](c).readRowByKey(ctx, featureKey)
}
