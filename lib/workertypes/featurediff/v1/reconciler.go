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

package v1

import (
	"context"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
)

// reconciliationVisitor implements the Visitor pattern to populate the renames/splits maps
// based on the polymorphic result returned by GetFeature.
type reconciliationVisitor struct {
	currentID string
	renames   map[string]string
	splits    map[string][]string
}

func (v *reconciliationVisitor) VisitRegularFeature(_ context.Context, _ backendtypes.RegularFeatureResult) error {
	// No-op: The feature exists in the DB (is Regular) but was removed from our snapshot.
	// This implies it simply fell out of scope of the user's query (e.g. tags changed).
	// We leave it as a standard "Removed" item with ReasonUnmatched.
	return nil
}

func (v *reconciliationVisitor) VisitMovedFeature(_ context.Context, result backendtypes.MovedFeatureResult) error {
	v.renames[v.currentID] = result.NewFeatureID()

	return nil
}

func (v *reconciliationVisitor) VisitSplitFeature(_ context.Context, result backendtypes.SplitFeatureResult) error {
	splitFeatures := result.SplitFeature()
	targetIDs := make([]string, 0, len(splitFeatures.Features))
	for _, f := range splitFeatures.Features {
		targetIDs = append(targetIDs, f.Id)
	}
	v.splits[v.currentID] = targetIDs

	return nil
}

// reconcileMoves modifies the diff in-place.
func reconcileMoves(diff *FeatureDiffV1, renames map[string]string) {
	addedMap := make(map[string]FeatureAdded)
	for _, a := range diff.Added {
		addedMap[a.ID] = a
	}

	var newRemoved []FeatureRemoved
	newMoves := diff.Moves

	for _, r := range diff.Removed {
		newID, isRenamed := renames[r.ID]
		target, isAdded := addedMap[newID]

		if isRenamed && isAdded {
			newMoves = append(newMoves, FeatureMoved{
				FromID:   r.ID,
				ToID:     newID,
				FromName: r.Name,
				ToName:   target.Name,
			})
			delete(addedMap, newID)
		} else {
			newRemoved = append(newRemoved, r)
		}
	}

	var newAdded []FeatureAdded
	for _, a := range diff.Added {
		if _, exists := addedMap[a.ID]; exists {
			newAdded = append(newAdded, a)
		}
	}

	diff.Removed = newRemoved
	diff.Added = newAdded
	diff.Moves = newMoves
}

// reconcileSplits modifies the diff in-place.
func reconcileSplits(diff *FeatureDiffV1, splits map[string][]string) {
	addedMap := make(map[string]FeatureAdded)
	for _, a := range diff.Added {
		addedMap[a.ID] = a
	}

	var newRemoved []FeatureRemoved
	newSplits := diff.Splits

	for _, r := range diff.Removed {
		targetIDs, isSplit := splits[r.ID]
		var foundTargets []FeatureAdded
		foundAny := false

		if isSplit {
			for _, targetID := range targetIDs {
				if target, isAdded := addedMap[targetID]; isAdded {
					foundTargets = append(foundTargets, target)
					foundAny = true
					delete(addedMap, targetID)
				}
			}
		}

		if foundAny {
			newSplits = append(newSplits, FeatureSplit{
				FromID:   r.ID,
				FromName: r.Name,
				To:       foundTargets,
			})
		} else {
			newRemoved = append(newRemoved, r)
		}
	}

	var newAdded []FeatureAdded
	for _, a := range diff.Added {
		if _, exists := addedMap[a.ID]; exists {
			newAdded = append(newAdded, a)
		}
	}

	diff.Removed = newRemoved
	diff.Added = newAdded
	diff.Splits = newSplits
}
