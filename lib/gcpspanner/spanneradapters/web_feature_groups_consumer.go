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
	"github.com/GoogleChrome/webstatus.dev/lib/webdxfeaturetypes"
)

// WebFeatureGroupsClient expects a subset of the functionality from lib/gcpspanner that only apply to Groups.
type WebFeatureGroupsClient interface {
	UpsertGroup(ctx context.Context, group gcpspanner.Group) (*string, error)
	UpsertFeatureGroupLookups(ctx context.Context,
		featureKeyToGroupsMapping map[string][]string, childGroupKeyToParentGroupKey map[string]string) error
}

// NewWebFeaturesConsumer constructs an adapter for the web features consumer service.
func NewWebFeatureGroupsConsumer(client WebFeatureGroupsClient) *WebFeatureGroupConsumer {
	return &WebFeatureGroupConsumer{client: client}
}

// WebFeatureGroupConsumer handles the conversion of group data between the workflow/API input
// format and the format used by the GCP Spanner client.
type WebFeatureGroupConsumer struct {
	client WebFeatureGroupsClient
}

func extractFeatureKeyToGroupsMapping(
	featuresData map[string]webdxfeaturetypes.FeatureValue,
) map[string][]string {
	m := make(map[string][]string, len(featuresData))

	for featureKey, feature := range featuresData {
		if feature.Group == nil {
			continue
		}

		m[featureKey] = feature.Group
	}

	return m
}

func (c *WebFeatureGroupConsumer) InsertWebFeatureGroups(
	ctx context.Context,
	featureData map[string]webdxfeaturetypes.FeatureValue,
	groupData map[string]webdxfeaturetypes.GroupData) error {
	groupKeyToInternalID := make(map[string]string, len(groupData))
	childToParentMap := make(map[string]string)
	// Step 1. Upsert basic group data and get group ids
	for key, group := range groupData {
		id, err := c.client.UpsertGroup(ctx, gcpspanner.Group{
			GroupKey: key,
			Name:     group.Name,
		})
		if err != nil {
			slog.ErrorContext(ctx, "unable to upsert group", "error", err, "groupKey", key)

			return err
		}
		groupKeyToInternalID[key] = *id

		if group.Parent != nil {
			childToParentMap[key] = *group.Parent
		}
	}

	// Step 2: Upsert the feature group lookups for feature search
	featureKeyToGroupsMapping := extractFeatureKeyToGroupsMapping(featureData)
	err := c.client.UpsertFeatureGroupLookups(ctx, featureKeyToGroupsMapping, childToParentMap)
	if err != nil {
		slog.ErrorContext(ctx, "unable to UpsertFeatureGroupLookups", "error", err)

		return err
	}

	return nil
}
