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
	"context"

	"cloud.google.com/go/datastore"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

type webFeaturesFilter struct {
	webFeatureID string
}

func (f webFeaturesFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.FilterField("web_feature_id", "=", f.webFeatureID)
}

func (c *Client) UpsertFeatureData(
	ctx context.Context,
	webFeatureID string,
	data web_platform_dx__web_features.FeatureData,
) error {
	entityClient := entityClient[FeatureData]{c}

	return entityClient.upsert(ctx,
		featureDataKey,
		&FeatureData{
			WebFeatureID: webFeatureID,
			Name:         data.Name,
		},
		webFeaturesFilter{
			webFeatureID: webFeatureID,
		},
	)
}

func (c *Client) ListWebFeataureData(ctx context.Context) ([]backend.Feature, error) {
	var featureData []*FeatureData
	_, err := c.GetAll(ctx, datastore.NewQuery(featureDataKey), &featureData)
	if err != nil {
		return nil, err
	}
	ret := make([]backend.Feature, len(featureData))
	for idx, val := range featureData {
		ret[idx] = backend.Feature{
			FeatureId: val.WebFeatureID,
			Name:      val.Name,
			Spec:      nil,
		}
	}

	return ret, nil
}

func (c *Client) Get(ctx context.Context, webFeatureID string) (*backend.Feature, error) {
	var featureData []*FeatureData
	_, err := c.GetAll(
		ctx, datastore.NewQuery(featureDataKey).
			FilterField("web_feature_id", "=", webFeatureID).Limit(1),
		&featureData)
	if err != nil {
		return nil, err
	}

	return &backend.Feature{
		Name:      featureData[0].WebFeatureID,
		FeatureId: featureData[0].WebFeatureID,
		Spec:      nil,
	}, nil
}
