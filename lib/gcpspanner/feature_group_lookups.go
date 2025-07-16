// Copyright 2025 Google LLC
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
	"log/slog"
)

const featureGroupKeysLookupTable = "FeatureGroupKeysLookup"

type spannerFeatureGroupKeysLookup struct {
	GroupKeyLowercase string `spanner:"GroupKey_Lowercase"`
	WebFeatureID      string `spanner:"WebFeatureID"`
	GroupID           string `spanner:"GroupID"`
	Depth             int64  `spanner:"Depth"`
}

func (c *Client) UpsertFeatureGroupLookups(
	ctx context.Context, featureKeyToGroupsMapping map[string][]string,
	childGroupKeyToParentGroupKey map[string]string) error {
	// TODO: We should do a diff and delete group lookups no longer needed.
	// This hasn't happened yet.
	txn := c.ReadOnlyTransaction()
	defer txn.Close()

	featureDetails, err := c.fetchAllWebFeatureIDsAndKeysWithTransaction(ctx, txn)
	if err != nil {
		slog.ErrorContext(ctx, "unable to get all feature details for feature group lookups", "error", err)

		return err
	}
	featureKeyToID := make(map[string]string, len(featureDetails))
	for _, featureDetail := range featureDetails {
		featureKeyToID[featureDetail.FeatureKey] = featureDetail.ID
	}

	groupDetails, err := c.fetchAllGroupIDsAndKeysWithTransaction(ctx, txn)
	if err != nil {
		slog.ErrorContext(ctx, "unable to get all group details for feature group lookups", "error", err)

		return err
	}
	groupKeyToGroupDetails := make(map[string]spannerGroupIDKeyAndKeyLowercase, len(featureDetails))
	for _, groupDetail := range groupDetails {
		groupKeyToGroupDetails[groupDetail.GroupKey] = groupDetail
	}

	return runConcurrentBatch(ctx,
		c, func(entityChan chan<- spannerFeatureGroupKeysLookup) {
			calculateAllFeatureGroupLookups(
				ctx,
				featureKeyToID,
				featureKeyToGroupsMapping,
				groupKeyToGroupDetails,
				entityChan,
				childGroupKeyToParentGroupKey)
		}, featureGroupKeysLookupTable)
}

func calculateAllFeatureGroupLookups(
	ctx context.Context,
	featureKeyToID map[string]string,
	featureKeyToGroupsMapping map[string][]string,
	groupKeyToDetails map[string]spannerGroupIDKeyAndKeyLowercase,
	entityChan chan<- spannerFeatureGroupKeysLookup,
	childGroupKeyToParentGroupKey map[string]string,
) {

	for featureKey, featureID := range featureKeyToID {
		featureGroups, found := featureKeyToGroupsMapping[featureKey]
		if !found {
			slog.WarnContext(ctx, "Unable to find feature group data for feature key. "+
				"This is okay if the feature is not associated with a group", "featureKey", featureKey)

			continue
		}

		for _, directGroupKey := range featureGroups {
			currentGroupKey := directGroupKey
			currentDepth := int64(0)

			for {
				groupData, found := groupKeyToDetails[currentGroupKey]
				if !found {
					slog.WarnContext(ctx, "group key not found during hierarchy traversal", "groupKey", currentGroupKey)

					break
				}

				entityChan <- spannerFeatureGroupKeysLookup{
					GroupID:           groupData.ID,
					GroupKeyLowercase: groupData.GroupKeyLowercase,
					WebFeatureID:      featureID,
					Depth:             currentDepth,
				}

				// Move up to the parent.
				parentGroupKey, hasParent := childGroupKeyToParentGroupKey[currentGroupKey]
				if !hasParent {
					break
				}
				currentGroupKey = parentGroupKey
				currentDepth++
			}
		}
	}
}
