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

	"cloud.google.com/go/spanner"
)

type SpannerFeatureSpec struct {
	WebFeatureID string
	FeatureSpec
}

// FeatureSpec contains availability information for a particular
// feature in a browser.
type FeatureSpec struct {
	Links []string
}

const featureSpecsTable = "FeatureSpecs"

type featureSpecSpannerMapper struct{}

func (featureSpecSpannerMapper) Table() string {
	return featureSpecsTable
}

func (featureSpecSpannerMapper) ToSpanner(entity *SpannerFeatureSpec) map[string]interface{} {
	return map[string]interface{}{
		"WebFeatureID": entity.WebFeatureID,
		"Links":        entity.Links,
	}
}

func (featureSpecSpannerMapper) GetKeyFromExternal(entity *SpannerFeatureSpec) string {
	return entity.WebFeatureID
}

func (featureSpecSpannerMapper) SelectOne(key string) spanner.Statement {
	return spanner.Statement{
		SQL: `SELECT WebFeatureID, Links FROM FeatureSpecs WHERE WebFeatureID = @webFeatureID`,
		Params: map[string]interface{}{
			"webFeatureID": key,
		},
	}
}

func (featureSpecSpannerMapper) Merge(external *SpannerFeatureSpec, _ SpannerFeatureSpec) SpannerFeatureSpec {
	return *external
}

// InsertFeatureSpec will insert the given feature spec information.
// If the spec info, does not exist, it will insert a new spec info.
// If the spec info exists, it currently overwrites the data.
func (c *Client) UpsertFeatureSpec(
	ctx context.Context,
	webFeatureID string,
	input FeatureSpec) error {
	id, err := c.GetIDFromFeatureKey(ctx, NewFeatureKeyFilter(webFeatureID))
	if err != nil {
		return err
	}
	if id == nil {
		return ErrInternalQueryFailure
	}

	featureSpec := &SpannerFeatureSpec{
		WebFeatureID: *id,
		FeatureSpec:  input,
	}

	writer := newEntityWriter[featureSpecSpannerMapper](c)

	return writer.upsert(ctx, featureSpec)
}
