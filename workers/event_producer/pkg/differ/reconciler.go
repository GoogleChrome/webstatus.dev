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

package differ

import (
	"context"
	"errors"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
)

// reconcileHistory analyzes the "Removed" list to detect if features were actually Moved or Split
// rather than deleted.
//
// It performs a "Detective" pass:
// 1. For every removed feature, ask the DB: "What is your current status?"
// 2. If the DB says "I moved to ID 'B'", we check if 'B' is in our "Added" list.
// 3. If yes, we link them together into a "Move" event and remove them from the raw Added/Removed lists.
//
// This transforms a confusing [Removed: "Grid", Added: "CSS Grid"] diff into a clear
// [Moved: "Grid" -> "CSS Grid"] diff.
func (d *FeatureDiffer) reconcileHistory(ctx context.Context, diff *FeatureDiff) (*FeatureDiff, error) {
	renames := make(map[string]string)
	splits := make(map[string][]string)
	visitor := &reconciliationVisitor{renames: renames, splits: splits, currentID: ""}

	// Phase 1: Investigation
	// Iterate through all removed features to build a map of their historical outcomes.
	for i := range diff.Removed {
		r := &diff.Removed[i]

		// Check the current status of the removed feature ID in the database.
		result, err := d.client.GetFeature(ctx, r.ID)
		if err != nil {
			// If the entity is completely gone from the DB, it's a true deletion.
			// We update the reason to allow for specific UI messaging (e.g. "Deleted from platform").
			if errors.Is(err, backendtypes.ErrEntityDoesNotExist) {
				r.Reason = ReasonDeleted

				continue
			}

			return nil, err
		}

		// Update the visitor context so it knows which OldID owns the result we are about to visit.
		visitor.currentID = r.ID

		// Dispatch based on whether the feature is Regular (exists), Moved, or Split.
		if err := result.Visit(ctx, visitor); err != nil {
			return nil, err
		}
	}

	// Phase 2: Correlation
	// If we found any history records, try to match them with the 'Added' list.
	if len(renames) > 0 {
		reconcileMoves(diff, renames)
	}
	if len(splits) > 0 {
		reconcileSplits(diff, splits)
	}

	return diff, nil
}

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

// reconcileMoves modifies the diff in-place. It pairs Removed items with Added items
// based on the provided renames map (OldID -> NewID).
func reconcileMoves(diff *FeatureDiff, renames map[string]string) {
	// Index the Added list for O(1) lookups
	addedMap := make(map[string]FeatureAdded)
	for _, a := range diff.Added {
		addedMap[a.ID] = a
	}

	var newRemoved []FeatureRemoved
	newMoves := diff.Moves

	for _, r := range diff.Removed {
		newID, isRenamed := renames[r.ID]
		target, isAdded := addedMap[newID]

		// A Move is only valid for *this* diff if the target ID is actually present in the 'Added' list.
		// If the target ID is missing (e.g. filtered out by the user's query), we treat the original
		// item as simply Removed.
		if isRenamed && isAdded {
			newMoves = append(newMoves, FeatureMoved{
				FromID:   r.ID,
				ToID:     newID,
				FromName: r.Name,
				ToName:   target.Name,
			})
			// Consume the added item so it doesn't appear as a standalone "Added" event later.
			delete(addedMap, newID)
		} else {
			newRemoved = append(newRemoved, r)
		}
	}

	// Reconstruct the Added list with only the remaining (unclaimed) items.
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

// reconcileSplits modifies the diff in-place. It pairs Removed items with one or more Added items
// based on the provided splits map (OldID -> [NewID...]).
func reconcileSplits(diff *FeatureDiff, splits map[string][]string) {
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
				// Check if any of the split targets are in our Added list.
				// Even if only 1 of 5 targets matches the user's query, we still report it as a Split.
				if target, isAdded := addedMap[targetID]; isAdded {
					foundTargets = append(foundTargets, target)
					foundAny = true
					// Consume the added item
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
