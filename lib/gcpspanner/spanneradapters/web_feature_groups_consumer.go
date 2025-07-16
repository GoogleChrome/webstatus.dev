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
)

// WebFeatureGroupsClient expects a subset of the functionality from lib/gcpspanner that only apply to Groups.
type WebFeatureGroupsClient interface {
	UpsertGroup(ctx context.Context, group gcpspanner.Group) (*string, error)
	UpsertFeatureGroupLookups(ctx context.Context, lookups []gcpspanner.FeatureGroupIDsLookup) error
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

func (c *WebFeatureGroupConsumer) calculateAllLookups(
	ctx context.Context,
	featureKeyToID map[string]string,
	featureData map[string]web_platform_dx__web_features.FeatureValue,
	groupKeyToID map[string]string,
	childToParentMap map[string]string,
) []gcpspanner.FeatureGroupIDsLookup {
	var allLookups []gcpspanner.FeatureGroupIDsLookup

	for featureKey, featureID := range featureKeyToID {
		feature := featureData[featureKey]
		if feature.Group == nil {
			continue
		}

		var directGroupKeys []string
		if feature.Group.String != nil {
			directGroupKeys = append(directGroupKeys, *feature.Group.String)
		} else if feature.Group.StringArray != nil {
			directGroupKeys = feature.Group.StringArray
		}

		// For each direct group associated with the feature...
		for _, directGroupKey := range directGroupKeys {
			currentGroupKey := directGroupKey
			currentDepth := int64(0)

			for {
				groupID, found := groupKeyToID[currentGroupKey]
				if !found {
					slog.WarnContext(ctx, "group key not found during hierarchy traversal", "groupKey", currentGroupKey)

					break
				}

				allLookups = append(allLookups, gcpspanner.FeatureGroupIDsLookup{
					ID:           groupID,
					WebFeatureID: featureID,
					Depth:        currentDepth,
				})

				// Move up to the parent.
				parentGroupKey, hasParent := childToParentMap[currentGroupKey]
				if !hasParent {
					break
				}
				currentGroupKey = parentGroupKey
				currentDepth++
			}
		}
	}

	return allLookups
}

func (c *WebFeatureGroupConsumer) InsertWebFeatureGroups(
	ctx context.Context,
	featureKeyToID map[string]string,
	featureData map[string]web_platform_dx__web_features.FeatureValue,
	groupData map[string]web_platform_dx__web_features.GroupData) error {
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

	// Step 2: Perform the core calculation in its own helper.
	lookupsToUpsert := c.calculateAllLookups(
		ctx, featureKeyToID, featureData, groupKeyToInternalID, childToParentMap,
	)

	// Step 3: Call the client to perform the final database write.
	if len(lookupsToUpsert) > 0 {
		return c.client.UpsertFeatureGroupLookups(ctx, lookupsToUpsert)
	}

	return nil
}
