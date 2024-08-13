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
	"fmt"

	"cloud.google.com/go/spanner"
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

// Implements the entityMapper interface for WebFeature and SpannerWebFeature.
type webFeatureSpannerMapper struct{}

func (m webFeatureSpannerMapper) SelectOne(key string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID, FeatureKey, Name
	FROM %s
	WHERE FeatureKey = @featureKey
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"featureKey": key,
	}
	stmt.Params = parameters

	return stmt
}

func (m webFeatureSpannerMapper) Merge(in WebFeature, existing SpannerWebFeature) SpannerWebFeature {
	return SpannerWebFeature{
		ID: existing.ID,
		WebFeature: WebFeature{
			FeatureKey: existing.FeatureKey,
			// Only allow overriding of the feature name.
			Name: cmp.Or[string](in.Name, existing.Name),
		},
	}
}

func (m webFeatureSpannerMapper) Table() string {
	return webFeaturesTable
}

func (m webFeatureSpannerMapper) GetKey(in WebFeature) string {
	return in.FeatureKey
}

func (m webFeatureSpannerMapper) GetID(key string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID
	FROM %s
	WHERE FeatureKey = @featureKey
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"featureKey": key,
	}
	stmt.Params = parameters

	return stmt
}

func (c *Client) UpsertWebFeature(ctx context.Context, feature WebFeature) (*string, error) {
	return newEntityWriterWithIDRetrieval[webFeatureSpannerMapper, string](c).upsertAndGetID(ctx, feature)
}

func (c *Client) GetIDFromFeatureKey(ctx context.Context, filter *FeatureIDFilter) (*string, error) {
	return newEntityWriterWithIDRetrieval[webFeatureSpannerMapper, string](c).getIDByKey(ctx, filter.featureKey)
}
