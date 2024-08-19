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

const webFeatureGroupsTable = "WebFeatureGroups"

// WebFeatureGroup contains the mapping between WebDXGroups and WebFeatures.
// Columns come from the ../../infra/storage/spanner/migrations/*.sql files.
type WebFeatureGroup struct {
	WebFeatureID string   `spanner:"WebFeatureID"`
	GroupIDs     []string `spanner:"GroupIDs"`
}

// Implements the Mapping interface for WebFeatureGroup and SpannerWebFeatureGroup.
type webFeaturesGroupSpannerMapper struct{}

func (m webFeaturesGroupSpannerMapper) GetKey(in WebFeatureGroup) string { return in.WebFeatureID }

func (m webFeaturesGroupSpannerMapper) Merge(
	in WebFeatureGroup, existing spannerWebFeatureGroup) spannerWebFeatureGroup {
	var groupIDs []string
	if in.GroupIDs != nil {
		groupIDs = in.GroupIDs
	} else {
		groupIDs = existing.GroupIDs
	}

	return spannerWebFeatureGroup{
		WebFeatureGroup: WebFeatureGroup{
			WebFeatureID: existing.WebFeatureID,
			GroupIDs:     groupIDs,
		},
	}
}

func (m webFeaturesGroupSpannerMapper) SelectOne(id string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		WebFeatureID, GroupIDs
	FROM %s
	WHERE WebFeatureID = @webFeatureID
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"webFeatureID": id,
	}
	stmt.Params = parameters

	return stmt
}

func (m webFeaturesGroupSpannerMapper) Table() string {
	return webFeatureGroupsTable
}

// SpannerGroup is a wrapper for the WebFeatureGroup that is actually
// stored in spanner.
type spannerWebFeatureGroup struct {
	WebFeatureGroup
}

func (c *Client) UpsertWebFeatureGroup(ctx context.Context, group WebFeatureGroup) error {
	return newEntityWriter[webFeaturesGroupSpannerMapper](c).upsert(ctx, group)
}
