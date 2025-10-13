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
	"fmt"

	"cloud.google.com/go/spanner"
)

const webFeatureChromiumHistogramEnumValuesTable = "WebFeatureChromiumHistogramEnumValues"

// WebFeatureChromiumHistogramEnumValue contains the mapping between ChromiumHistogramEnumValues and WebFeatures.
// Columns come from the ../../infra/storage/spanner/migrations/*.sql files.
type WebFeatureChromiumHistogramEnumValue struct {
	WebFeatureID                 string `spanner:"WebFeatureID"`
	ChromiumHistogramEnumValueID string `spanner:"ChromiumHistogramEnumValueID"`
}

// SpannerWebFeatureChromiumHistogramEnum is a wrapper for the WebFeatureChromiumHistogramEnum that is actually
// stored in spanner.
type spannerWebFeatureChromiumHistogramEnum struct {
	WebFeatureChromiumHistogramEnumValue
}

// Implements the Mapping interface for WebFeatureChromiumHistogramEnum and SpannerWebFeatureChromiumHistogramEnum.
type webFeaturesChromiumHistogramEnumSpannerMapper struct{}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) GetKeyFromExternal(
	in WebFeatureChromiumHistogramEnumValue) string {
	return in.WebFeatureID
}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) GetKeyFromInternal(
	in spannerWebFeatureChromiumHistogramEnum) string {
	return in.WebFeatureID
}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) Table() string {
	return webFeatureChromiumHistogramEnumValuesTable
}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) SelectAll() spanner.Statement {
	return spanner.NewStatement(fmt.Sprintf(`SELECT * FROM %s`, m.Table()))
}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) SelectAllByKeys(id string) spanner.Statement {
	stmt := spanner.NewStatement(`
	SELECT
		*
	FROM WebFeatureChromiumHistogramEnumValues
	WHERE WebFeatureID = @webFeatureID`)
	stmt.Params = map[string]interface{}{
		"webFeatureID": id,
	}

	return stmt
}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) MergeAndCheckChanged(
	_ WebFeatureChromiumHistogramEnumValue,
	existing spannerWebFeatureChromiumHistogramEnum) (spannerWebFeatureChromiumHistogramEnum, bool) {
	// This entity only has key columns, so there's nothing to merge or update.
	// The synchronizer will handle inserts and deletes based on the key.
	return existing, false
}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) DeleteMutation(
	in spannerWebFeatureChromiumHistogramEnum) *spanner.Mutation {
	return spanner.Delete(m.Table(), spanner.Key{in.WebFeatureID})
}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) GetChildDeleteKeyMutations(
	_ context.Context,
	_ *Client,
	_ []spannerWebFeatureChromiumHistogramEnum,
) ([]ExtraMutationsGroup, error) {
	return nil, nil
}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) PreDeleteHook(
	_ context.Context, _ *Client, _ []spannerWebFeatureChromiumHistogramEnum) ([]ExtraMutationsGroup, error) {
	return nil, nil
}

func (c *Client) SyncWebFeatureChromiumHistogramEnumValues(
	ctx context.Context,
	in []WebFeatureChromiumHistogramEnumValue,
) error {
	return newEntitySynchronizer[webFeaturesChromiumHistogramEnumSpannerMapper](c).Sync(ctx, in)
}

func (c *Client) getAllWebFeatureChromiumHistogramEnumValuesByFeatureID(
	ctx context.Context, featureID string) ([]WebFeatureChromiumHistogramEnumValue, error) {
	return newAllByKeysEntityReader[
		webFeaturesChromiumHistogramEnumSpannerMapper,
		string,
		WebFeatureChromiumHistogramEnumValue,
	](c).readAllByKeys(ctx, featureID)
}
