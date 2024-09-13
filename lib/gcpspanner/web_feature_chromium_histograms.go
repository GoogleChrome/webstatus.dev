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

func (m webFeaturesChromiumHistogramEnumSpannerMapper) GetKey(in WebFeatureChromiumHistogramEnumValue) string {
	return in.WebFeatureID
}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) Table() string {
	return webFeatureChromiumHistogramEnumValuesTable
}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) Merge(
	_ WebFeatureChromiumHistogramEnumValue,
	existing spannerWebFeatureChromiumHistogramEnum) spannerWebFeatureChromiumHistogramEnum {
	return existing
}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) SelectOne(id string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		WebFeatureID, ChromiumHistogramEnumValueID
	FROM %s
	WHERE WebFeatureID = @webFeatureID
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"webFeatureID": id,
	}
	stmt.Params = parameters

	return stmt
}

func (c *Client) UpsertWebFeatureChromiumHistogramEnumValue(
	ctx context.Context, in WebFeatureChromiumHistogramEnumValue) error {
	return newEntityWriter[webFeaturesChromiumHistogramEnumSpannerMapper](c).upsert(ctx, in)
}
