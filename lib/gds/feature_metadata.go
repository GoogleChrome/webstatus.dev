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

package gds

import (
	"cmp"
	"context"

	"cloud.google.com/go/datastore"
)

const featureMetadataKey = "FeatureMetadataKey"

// FeatureMetadata contains useful metadata about the feature.
// This information is not stored in the relational database because it is not used in complex queries for
// the overview page.
type FeatureMetadata struct {
	// The ID from WebFeatures table in spanner. Not the web features key.
	WebFeatureID string `datastore:"web_feature_id"`
	// Non-null fields from https://github.com/web-platform-dx/web-features/blob/main/schemas/data.schema.json
	Description string `datastore:"description"`
	// Nullable fields from https://github.com/web-platform-dx/web-features/blob/main/schemas/data.schema.json
	CanIUseIDs []string `datastore:"can_i_use_ids"`
}

// webFeaturesMetadataFilter implements Filterable to filter by web_feature_id.
// Compatible kinds:
// - featureMetadataKey.
type webFeaturesMetadataFilter struct {
	webFeatureID string
}

func (f webFeaturesMetadataFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.FilterField("web_feature_id", "=", f.webFeatureID)
}

// webFeatureMetadataMerge implements Mergeable for FeatureMetadata.
type webFeatureMetadataMerge struct{}

func (m webFeatureMetadataMerge) Merge(existing *FeatureMetadata, incoming *FeatureMetadata) *FeatureMetadata {
	canIUseIDs := existing.CanIUseIDs
	if len(incoming.CanIUseIDs) > 0 {
		canIUseIDs = incoming.CanIUseIDs
	}

	return &FeatureMetadata{
		Description: cmp.Or[string](incoming.Description, existing.Description, ""),
		CanIUseIDs:  canIUseIDs,
		// The below fields cannot be overridden during a merge.
		WebFeatureID: existing.WebFeatureID,
	}
}

// UpsertFeatureMetadata inserts/updates metadata for the given web feature.
func (c *Client) UpsertFeatureMetadata(
	ctx context.Context,
	data FeatureMetadata,
) error {
	entityClient := entityClient[FeatureMetadata]{c}

	return entityClient.upsert(ctx,
		featureMetadataKey,
		&data,
		webFeatureMetadataMerge{},
		webFeaturesMetadataFilter{
			webFeatureID: data.WebFeatureID,
		},
	)
}

// GetWebFeatureMetadata atttempts to get data for a given web feature.
func (c *Client) GetWebFeatureMetadata(ctx context.Context, webFeatureID string) (*FeatureMetadata, error) {
	entityClient := entityClient[FeatureMetadata]{c}
	featureData, err := entityClient.get(ctx, featureMetadataKey, webFeaturesMetadataFilter{
		webFeatureID: webFeatureID,
	})
	if err != nil {
		return nil, err
	}

	return featureData, nil
}
