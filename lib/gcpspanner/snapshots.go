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

const snapshotsTable = "WebDXSnapshots"

// Snapshot contains common metadata for a snapshot from the WebDX web-feature
// repository.
// Columns come from the ../../infra/storage/spanner/migrations/*.sql files.
type Snapshot struct {
	SnapshotKey string `spanner:"SnapshotKey"`
	Name        string `spanner:"Name"`
}

// spannerSnapshot is a wrapper for the Snapshot that is actually
// stored in spanner. This is useful because the spanner id is not useful to
// return to the end user since it is only used to decouple the primary keys
// between this system and web features repo.
type spannerSnapshot struct {
	ID string `spanner:"ID"`
	Snapshot
}

// Implements the entityMapper interface for Snapshot and SpannerSnapshot.
type snapshotSpannerMapper struct{}

func (m snapshotSpannerMapper) Merge(in Snapshot, existing spannerSnapshot) spannerSnapshot {
	return spannerSnapshot{
		ID: existing.ID,
		Snapshot: Snapshot{
			Name:        cmp.Or(in.Name, existing.Name),
			SnapshotKey: existing.SnapshotKey,
		},
	}
}

func (m snapshotSpannerMapper) GetKey(in Snapshot) string {
	return in.SnapshotKey
}

func (m snapshotSpannerMapper) SelectOne(key string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID, SnapshotKey, Name
	FROM %s
	WHERE SnapshotKey = @snapshotKey
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"snapshotKey": key,
	}
	stmt.Params = parameters

	return stmt
}

func (m snapshotSpannerMapper) Table() string {
	return snapshotsTable
}

func (m snapshotSpannerMapper) GetID(key string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID
	FROM %s
	WHERE SnapshotKey = @snapshotKey
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"snapshotKey": key,
	}
	stmt.Params = parameters

	return stmt
}

func (c *Client) UpsertSnapshot(ctx context.Context, snapshot Snapshot) (*string, error) {
	return newEntityWriterWithIDRetrieval[snapshotSpannerMapper, string](c).upsertAndGetID(ctx, snapshot)
}

func (c *Client) GetSnapshotIDFromSnapshotKey(ctx context.Context, snapshotKey string) (*string, error) {
	return newEntityWriterWithIDRetrieval[snapshotSpannerMapper, string](c).getIDByKey(ctx, snapshotKey)
}
