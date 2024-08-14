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
	UpsertWebFeatureGroup(ctx context.Context, group gcpspanner.WebFeatureGroup) error
	UpsertGroupDescendantInfo(ctx context.Context, groupKey string, descendantInfo gcpspanner.GroupDescendantInfo) error
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

func (c *WebFeatureGroupConsumer) InsertWebFeatureGroups(
	ctx context.Context,
	featureKeyToID map[string]string,
	featureData map[string]web_platform_dx__web_features.FeatureValue,
	groupData map[string]web_platform_dx__web_features.GroupData) error {
	groupKeyToInternalID := make(map[string]string, len(groupData))
	// Upsert basic group data and get group ids
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
	}
	// Upsert the group descendant info.
	groupDescMap := c.buildGroupDescendants(groupData, groupKeyToInternalID)
	for key, groupDescInfo := range groupDescMap {
		err := c.client.UpsertGroupDescendantInfo(ctx, key, groupDescInfo)
		if err != nil {
			slog.ErrorContext(ctx, "unable to upsert group descendant info",
				"error", err, "groupKey", key, "info", groupDescInfo)

			return err
		}
	}
	// Upsert the web-feature to group mappings
	for featureKey, featureID := range featureKeyToID {
		feature := featureData[featureKey]
		if feature.Group == nil {
			continue
		}
		var groupIDs []string
		if feature.Group.String != nil {
			internalID, found := groupKeyToInternalID[*feature.Group.String]
			if !found {
				slog.WarnContext(ctx, "unable to find internal group ID", "groupKey", *feature.Group.String)

				continue
			}
			groupIDs = append(groupIDs, internalID)
		} else if feature.Group.StringArray != nil {
			for _, groupKey := range feature.Group.StringArray {
				internalID, found := groupKeyToInternalID[groupKey]
				if !found {
					slog.WarnContext(ctx, "unable to find internal group ID", "groupKey", groupKey)

					continue
				}
				groupIDs = append(groupIDs, internalID)
			}
		}
		err := c.client.UpsertWebFeatureGroup(ctx, gcpspanner.WebFeatureGroup{
			WebFeatureID: featureID,
			GroupIDs:     groupIDs,
		})
		if err != nil {
			slog.ErrorContext(ctx, "unable to upsert web feature group", "webFeatureID",
				featureID, "featureKey", featureKey, "groupIDs", groupIDs, "error", err)

			return err
		}
	}

	return nil
}

func (c *WebFeatureGroupConsumer) buildGroupDescendants(
	data map[string]web_platform_dx__web_features.GroupData,
	groupKeyToID map[string]string,
) map[string]gcpspanner.GroupDescendantInfo {
	m := make(map[string]gcpspanner.GroupDescendantInfo, len(data))
	groupToChildrenGroupKeys := make(map[string][]string, len(data))
	var rootGroupKeys []string
	for groupKey, groupData := range data {
		info := gcpspanner.GroupDescendantInfo{
			DescendantGroupIDs: nil,
		}
		m[groupKey] = info
		if groupData.Parent != nil {
			groupToChildrenGroupKeys[*groupData.Parent] = append(groupToChildrenGroupKeys[*groupData.Parent], groupKey)
		} else {
			rootGroupKeys = append(rootGroupKeys, groupKey)
		}
	}
	for _, rootGroupKey := range rootGroupKeys {
		c.populateDescendants(rootGroupKey, m, groupToChildrenGroupKeys, groupKeyToID)
	}

	return m
}

func (c *WebFeatureGroupConsumer) populateDescendants(
	groupKey string,
	groupDescendantMap map[string]gcpspanner.GroupDescendantInfo,
	groupToChildrenGroupKeys map[string][]string,
	groupKeyToInternalID map[string]string) {
	info := groupDescendantMap[groupKey]

	// Base case: if no children, descendants are empty (no descendants, only itself)
	if _, exists := groupToChildrenGroupKeys[groupKey]; !exists {
		// DescendantGroupIDs is already nil. No need to update.
		return
	}

	// Recursive case: collect descendants from children
	for _, childGroupKey := range groupToChildrenGroupKeys[groupKey] {
		c.populateDescendants(childGroupKey, groupDescendantMap, groupToChildrenGroupKeys, groupKeyToInternalID)
		childInfo := groupDescendantMap[childGroupKey]
		info.DescendantGroupIDs = append(info.DescendantGroupIDs, groupKeyToInternalID[childGroupKey])
		info.DescendantGroupIDs = append(info.DescendantGroupIDs, childInfo.DescendantGroupIDs...)
	}

	groupDescendantMap[groupKey] = info
}
