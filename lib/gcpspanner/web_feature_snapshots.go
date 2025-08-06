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

const webFeatureSnapshotsTable = "WebFeatureSnapshots"

// WebFeatureSnapshot contains the mapping between WebDXSnapshots and WebFeatures.
// Columns come from the ../../infra/storage/spanner/migrations/*.sql files.
type WebFeatureSnapshot struct {
	WebFeatureID string   `spanner:"WebFeatureID"`
	SnapshotIDs  []string `spanner:"SnapshotIDs"`
}

// SpannerWebFeatureSnapshot is a wrapper for the WebFeatureSnapshot that is actually
// stored in spanner.
type spannerWebFeatureSnapshot struct {
	WebFeatureSnapshot
}

// Implements the Mapping interface for WebFeatureSnapshot and SpannerWebFeatureSnapshot.
type webFeaturesSnapshotSpannerMapper struct{}

func (m webFeaturesSnapshotSpannerMapper) GetKeyFromExternal(in WebFeatureSnapshot) string {
	return in.WebFeatureID
}

func (m webFeaturesSnapshotSpannerMapper) Table() string {
	return webFeatureSnapshotsTable
}

func (m webFeaturesSnapshotSpannerMapper) Merge(
	in WebFeatureSnapshot, existing spannerWebFeatureSnapshot) spannerWebFeatureSnapshot {
	var snapshotIDs []string
	if in.SnapshotIDs != nil {
		snapshotIDs = in.SnapshotIDs
	} else {
		snapshotIDs = existing.SnapshotIDs
	}

	return spannerWebFeatureSnapshot{
		WebFeatureSnapshot: WebFeatureSnapshot{
			WebFeatureID: existing.WebFeatureID,
			SnapshotIDs:  snapshotIDs,
		},
	}
}

func (m webFeaturesSnapshotSpannerMapper) SelectOne(id string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		WebFeatureID, SnapshotIDs
	FROM %s
	WHERE WebFeatureID = @webFeatureID
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"webFeatureID": id,
	}
	stmt.Params = parameters

	return stmt
}

func (c *Client) UpsertWebFeatureSnapshot(ctx context.Context, snapshot WebFeatureSnapshot) error {
	return newEntityWriter[webFeaturesSnapshotSpannerMapper](c).upsert(ctx, snapshot)
}
