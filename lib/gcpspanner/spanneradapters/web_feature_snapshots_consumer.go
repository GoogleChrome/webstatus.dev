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

package spanneradapters

import (
	"context"
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
	"github.com/GoogleChrome/webstatus.dev/lib/webdxfeaturetypes"
)

// WebFeatureSnapshotsClient expects a subset of the functionality from lib/gcpspanner that only apply to Snapshots.
type WebFeatureSnapshotsClient interface {
	UpsertSnapshot(ctx context.Context, snapshot gcpspanner.Snapshot) (*string, error)
	UpsertWebFeatureSnapshot(ctx context.Context, snapshot gcpspanner.WebFeatureSnapshot) error
}

// NewWebFeatureSnapshotsConsumer constructs an adapter for the web feature snapshots consumer service.
func NewWebFeatureSnapshotsConsumer(client WebFeatureSnapshotsClient) *WebFeatureSnapshotConsumer {
	return &WebFeatureSnapshotConsumer{client: client}
}

// WebFeatureSnapshotConsumer handles the conversion of snapshot data between the workflow/API input
// format and the format used by the GCP Spanner client.
type WebFeatureSnapshotConsumer struct {
	client WebFeatureSnapshotsClient
}

func (c *WebFeatureSnapshotConsumer) upsertSnapshots(
	ctx context.Context,
	snapshotData map[string]web_platform_dx__web_features.SnapshotData,
	snapshotKeyToInternalID map[string]string,
) error {
	for key, snapshot := range snapshotData {
		id, err := c.client.UpsertSnapshot(ctx, gcpspanner.Snapshot{
			SnapshotKey: key,
			Name:        snapshot.Name,
		})
		if err != nil {
			slog.ErrorContext(ctx, "unable to upsert snapshot", "error", err, "snapshotKey", key)

			return err
		}
		snapshotKeyToInternalID[key] = *id
	}

	return nil
}

// nolint:dupl // TODO - we should fix this.
func (c *WebFeatureSnapshotConsumer) upsertSnapshotMappings(
	ctx context.Context,
	featureKeyToID map[string]string,
	featureData webdxfeaturetypes.FeatureKinds,
	snapshotKeyToInternalID map[string]string,
) error {
	for featureKey, featureID := range featureKeyToID {
		feature := featureData[featureKey]
		if feature.Snapshot == nil {
			continue
		}
		var snapshotIDs []string
		if feature.Snapshot.String != nil {
			internalID, found := snapshotKeyToInternalID[*feature.Snapshot.String]
			if !found {
				slog.WarnContext(ctx, "unable to find internal snapshot ID", "snapshotKey", *feature.Snapshot.String)

				continue
			}
			snapshotIDs = append(snapshotIDs, internalID)
		} else if feature.Snapshot.StringArray != nil {
			for _, snapshotKey := range feature.Snapshot.StringArray {
				internalID, found := snapshotKeyToInternalID[snapshotKey]
				if !found {
					slog.WarnContext(ctx, "unable to find internal snapshot ID", "snapshotKey", snapshotKey)

					continue
				}
				snapshotIDs = append(snapshotIDs, internalID)
			}
		}
		err := c.client.UpsertWebFeatureSnapshot(ctx, gcpspanner.WebFeatureSnapshot{
			WebFeatureID: featureID,
			SnapshotIDs:  snapshotIDs,
		})
		if err != nil {
			slog.ErrorContext(ctx, "unable to upsert web feature snapshot", "webFeatureID",
				featureID, "featureKey", featureKey, "snapshotIDs", snapshotIDs, "error", err)

			return err
		}
	}

	return nil
}

func (c *WebFeatureSnapshotConsumer) InsertWebFeatureSnapshots(
	ctx context.Context,
	featureKeyToID map[string]string,
	featureData webdxfeaturetypes.FeatureKinds,
	snapshotData map[string]web_platform_dx__web_features.SnapshotData) error {
	snapshotKeyToInternalID := make(map[string]string, len(snapshotData))
	// Upsert basic snapshot data and get snapshot ids.
	err := c.upsertSnapshots(ctx, snapshotData, snapshotKeyToInternalID)
	if err != nil {
		return err
	}

	// Upsert the web-feature to snapshot mappings.
	err = c.upsertSnapshotMappings(ctx, featureKeyToID, featureData, snapshotKeyToInternalID)
	if err != nil {
		return err
	}

	return nil
}
