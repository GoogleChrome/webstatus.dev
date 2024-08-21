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

const webFeatureChromiumHistogramEnumsTable = "WebFeatureChromiumHistogramEnums"

// WebFeatureChromiumHistogramEnum contains the mapping between ChromiumHistogramEnumValues and WebFeatures.
// Columns come from the ../../infra/storage/spanner/migrations/*.sql files.
type WebFeatureChromiumHistogramEnum struct {
	WebFeatureID            string `spanner:"WebFeatureID"`
	ChromiumHistogramEnumID string `spanner:"ChromiumHistogramEnumID"`
}

// SpannerWebFeatureChromiumHistogramEnum is a wrapper for the WebFeatureChromiumHistogramEnum that is actually
// stored in spanner.
type spannerWebFeatureChromiumHistogramEnum struct {
	WebFeatureChromiumHistogramEnum
}

// Implements the Mapping interface for WebFeatureChromiumHistogramEnum and SpannerWebFeatureChromiumHistogramEnum.
type webFeaturesChromiumHistogramEnumSpannerMapper struct{}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) GetKey(in WebFeatureChromiumHistogramEnum) string {
	return in.WebFeatureID
}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) Table() string {
	return webFeatureChromiumHistogramEnumsTable
}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) Merge(
	_ WebFeatureChromiumHistogramEnum,
	existing spannerWebFeatureChromiumHistogramEnum) spannerWebFeatureChromiumHistogramEnum {
	return existing
}

func (m webFeaturesChromiumHistogramEnumSpannerMapper) SelectOne(id string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		WebFeatureID, ChromiumHistogramEnumID
	FROM %s
	WHERE WebFeatureID = @webFeatureID
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"webFeatureID": id,
	}
	stmt.Params = parameters

	return stmt
}

func (c *Client) UpsertWebFeatureChromiumHistogramEnum(ctx context.Context, in WebFeatureChromiumHistogramEnum) error {
	return newEntityWriter[webFeaturesChromiumHistogramEnumSpannerMapper](c).upsert(ctx, in)
}
