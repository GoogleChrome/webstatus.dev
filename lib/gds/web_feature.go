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
	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

const featureDataKey = "FeatureData"

// FeatureData contains:
// - basic metadata about the web feature.
// - snapshot of latest metrics.
type FeatureData struct {
	WebFeatureID *string `datastore:"web_feature_id"`
	Name         *string `datastore:"name"`
}

// webFeaturesFilter implements Filterable to filter by web_feature_id.
// Compatible kinds:
// - featureDataKey.
type webFeaturesFilter struct {
	webFeatureID string
}

func (f webFeaturesFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.FilterField("web_feature_id", "=", f.webFeatureID)
}

// webFeatureMerge implements Mergeable for FeatureData.
type webFeatureMerge struct{}

func (m webFeatureMerge) Merge(existing *FeatureData, new *FeatureData) *FeatureData {
	return &FeatureData{
		Name: cmp.Or[*string](new.Name, existing.Name),
		// The below fields cannot be overridden during a merge.
		WebFeatureID: existing.WebFeatureID,
	}
}

// UpsertFeatureData inserts/updates data for the given web feature.
func (c *Client) UpsertFeatureData(
	ctx context.Context,
	webFeatureID string,
	data web_platform_dx__web_features.FeatureData,
) error {
	entityClient := entityClient[FeatureData]{c}

	return entityClient.upsert(ctx,
		featureDataKey,
		&FeatureData{
			WebFeatureID: &webFeatureID,
			Name:         &data.Name,
		},
		webFeatureMerge{},
		webFeaturesFilter{
			webFeatureID: webFeatureID,
		},
	)
}

// ListWebFeataureData lists web features data.
func (c *Client) ListWebFeataureData(ctx context.Context, pageToken *string) ([]backend.Feature, *string, error) {
	entityClient := entityClient[FeatureData]{c}
	featureData, nextPageToken, err := entityClient.list(ctx, featureDataKey, pageToken)
	if err != nil {
		return nil, nil, err
	}
	ret := make([]backend.Feature, len(featureData))
	for idx, val := range featureData {
		// nolint: exhaustruct // TODO revisit once we adjust the ingestion data to incorporate the new fields.
		ret[idx] = backend.Feature{
			FeatureId: *val.WebFeatureID,
			Name:      *val.Name,
			Spec:      nil,
		}
	}

	return ret, nextPageToken, nil
}

// GetWebFeatureData atttempts to get data for a given web feature.
func (c *Client) GetWebFeatureData(ctx context.Context, webFeatureID string) (*backend.Feature, error) {
	entityClient := entityClient[FeatureData]{c}
	featureData, err := entityClient.get(ctx, featureDataKey, webFeaturesFilter{
		webFeatureID: webFeatureID,
	})
	if err != nil {
		return nil, err
	}

	// nolint: exhaustruct // TODO revisit once we adjust the ingestion data to incorporate the new fields.
	return &backend.Feature{
		Name:      *featureData.Name,
		FeatureId: *featureData.WebFeatureID,
		Spec:      nil,
	}, nil
}
