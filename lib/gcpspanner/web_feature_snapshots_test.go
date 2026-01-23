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

// nolint:dupl // WONTFIX
package gcpspanner

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func setupRequiredTablesForWebFeatureSnapshot(
	ctx context.Context,
	t *testing.T,
) map[string]string {
	ret := map[string]string{}
	sampleFeatures := getSampleFeatures()
	for _, feature := range sampleFeatures {
		id, err := spannerClient.upsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert. %s", err.Error())

			continue
		}
		ret[feature.FeatureKey] = *id
	}

	return ret
}

func (c *Client) createSampleWebFeatureSnapshots(
	ctx context.Context, t *testing.T, idMap map[string]string) {
	err := c.UpsertWebFeatureSnapshot(ctx, WebFeatureSnapshot{
		WebFeatureID: idMap["feature1"],
		SnapshotIDs: []string{
			"snapshot1",
			"snapshot2",
		},
	})
	if err != nil {
		t.Fatalf("failed to insert snapshot. err: %s snapshot\n", err)
	}
	err = c.UpsertWebFeatureSnapshot(ctx, WebFeatureSnapshot{
		WebFeatureID: idMap["feature2"],
		SnapshotIDs:  nil,
	})
	if err != nil {
		t.Fatalf("failed to insert snapshot. err: %s snapshot\n", err)
	}
}

func (c *Client) ReadAllWebFeatureSnapshots(ctx context.Context, _ *testing.T) ([]WebFeatureSnapshot, error) {
	stmt := spanner.NewStatement(
		`SELECT
			WebFeatureID, SnapshotIDs
		FROM WebFeatureSnapshots`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []WebFeatureSnapshot
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var snapshot spannerWebFeatureSnapshot
		if err := row.ToStruct(&snapshot); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		ret = append(ret, snapshot.WebFeatureSnapshot)
	}

	return ret, nil
}

func sortWebFeatureSnapshots(left, right WebFeatureSnapshot) int {
	return cmp.Compare(left.WebFeatureID, right.WebFeatureID)
}

func webFeatureSnapshotEquality(left, right WebFeatureSnapshot) bool {
	return left.WebFeatureID == right.WebFeatureID &&
		slices.Equal(left.SnapshotIDs, right.SnapshotIDs)
}

func TestUpsertWebFeatureSnapshot(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	idMap := setupRequiredTablesForWebFeatureSnapshot(ctx, t)
	spannerClient.createSampleWebFeatureSnapshots(ctx, t, idMap)

	expected := []WebFeatureSnapshot{
		{
			WebFeatureID: idMap["feature1"],
			SnapshotIDs: []string{
				"snapshot1",
				"snapshot2",
			},
		},
		{
			WebFeatureID: idMap["feature2"],
			SnapshotIDs:  nil,
		},
	}
	slices.SortFunc(expected, sortWebFeatureSnapshots)

	snapshots, err := spannerClient.ReadAllWebFeatureSnapshots(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all snapshots err: %s", err)
	}
	slices.SortFunc(snapshots, sortWebFeatureSnapshots)

	if !slices.EqualFunc(expected, snapshots, webFeatureSnapshotEquality) {
		t.Errorf("unequal snapshots.\nexpected %+v\nreceived %+v", expected, snapshots)
	}

	// Upsert snapshot
	err = spannerClient.UpsertWebFeatureSnapshot(ctx, WebFeatureSnapshot{
		WebFeatureID: idMap["feature2"],
		SnapshotIDs: []string{
			"snapshot3",
		},
	})
	if err != nil {
		t.Fatalf("unable to update snapshot err: %s", err)
	}

	expected = []WebFeatureSnapshot{
		{
			WebFeatureID: idMap["feature1"],
			SnapshotIDs: []string{
				"snapshot1",
				"snapshot2",
			},
		},
		{
			WebFeatureID: idMap["feature2"],
			SnapshotIDs: []string{
				"snapshot3",
			},
		},
	}
	slices.SortFunc(expected, sortWebFeatureSnapshots)

	snapshots, err = spannerClient.ReadAllWebFeatureSnapshots(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all snapshots err: %s", err)
	}
	slices.SortFunc(snapshots, sortWebFeatureSnapshots)

	if !slices.EqualFunc(expected, snapshots, webFeatureSnapshotEquality) {
		t.Errorf("unequal snapshots.\nexpected %+v\nreceived %+v", expected, snapshots)
	}
}
