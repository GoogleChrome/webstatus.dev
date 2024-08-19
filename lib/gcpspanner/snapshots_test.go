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
	"errors"
	"slices"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func (c *Client) createSampleSnapshots(ctx context.Context, t *testing.T) map[string]string {
	snapshot1 := Snapshot{
		SnapshotKey: "snapshot1",
		Name:        "Snapshot 1",
	}
	snapshot2 := Snapshot{
		SnapshotKey: "snapshot2",
		Name:        "Snapshot 2",
	}
	snapshot1ID, err := c.UpsertSnapshot(ctx, snapshot1)
	if err != nil {
		t.Fatalf("failed to insert snapshot. err: %s snapshot: %v\n", err, snapshot1)
	}
	snapshot2ID, err := c.UpsertSnapshot(ctx, snapshot2)
	if err != nil {
		t.Fatalf("failed to insert snapshot. err: %s snapshot: %v\n", err, snapshot2)
	}

	return map[string]string{
		snapshot1.SnapshotKey: *snapshot1ID,
		snapshot2.SnapshotKey: *snapshot2ID,
	}
}

func (c *Client) ReadAllSnapshots(ctx context.Context, _ *testing.T) ([]Snapshot, error) {
	stmt := spanner.NewStatement(
		`SELECT
			ID, SnapshotKey, Name
		FROM WebDXSnapshots ORDER BY SnapshotKey ASC`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []Snapshot
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var snapshot spannerSnapshot
		if err := row.ToStruct(&snapshot); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		ret = append(ret, snapshot.Snapshot)
	}

	return ret, nil
}

func TestUpsertSnapshot(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	_ = spannerClient.createSampleSnapshots(ctx, t)

	snapshots, err := spannerClient.ReadAllSnapshots(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all snapshots err: %s", err)
	}

	expectedSnapshots := []Snapshot{
		{
			SnapshotKey: "snapshot1",
			Name:        "Snapshot 1",
		},
		{
			SnapshotKey: "snapshot2",
			Name:        "Snapshot 2",
		},
	}

	if !slices.EqualFunc(expectedSnapshots, snapshots, snapshotEquality) {
		t.Errorf("unequal snapshots.\nexpected %+v\nreceived %+v", expectedSnapshots, snapshots)
	}

	// Change one of the snapshots
	_, err = spannerClient.UpsertSnapshot(ctx, Snapshot{
		SnapshotKey: "snapshot2",
		// Change the name
		Name: "Snapshot 2 edit",
	})
	if err != nil {
		t.Errorf("unable to edit the snapshot %s", err)
	}

	expectedSnapshots = []Snapshot{
		{
			SnapshotKey: "snapshot1",
			Name:        "Snapshot 1",
		},
		{
			SnapshotKey: "snapshot2",
			Name:        "Snapshot 2 edit",
		},
	}

	snapshots, err = spannerClient.ReadAllSnapshots(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all snapshots err: %s", err)
	}

	if !slices.EqualFunc(expectedSnapshots, snapshots, snapshotEquality) {
		t.Errorf("unequal snapshots.\nexpected %+v\nreceived %+v", expectedSnapshots, snapshots)
	}
}

func snapshotEquality(left, right Snapshot) bool {
	return left.Name == right.Name &&
		left.SnapshotKey == right.SnapshotKey
}
