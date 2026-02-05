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
	"errors"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes/comparables"
)

// ReconcileHistory analyzes the "Removed" list to detect if features were actually Moved or Split
// rather than deleted.
//
// It performs a "Detective" pass:
// 1. For every removed feature, ask the DB: "What is your current status?"
// 2. If the DB says "I moved to ID 'B'", we check if 'B' is in our "Added" list.
// 3. If yes, we link them together into a "Move" event and remove them from the raw Added/Removed lists.
//
// This transforms a confusing [Removed: "Grid", Added: "CSS Grid"] diff into a clear
// [Moved: "Grid" -> "CSS Grid"] diff.
func (w *FeatureDiffWorkflow) ReconcileHistory(ctx context.Context,
	oldSnapshot, newSnapshot map[string]comparables.Feature) error {
	renames := make(map[string]string)
	splits := make(map[string][]string)
	visitor := &reconciliationVisitor{renames: renames, splits: splits, currentID: "", targetFeature: nil}

	// Phase 1: Investigation
	// We rebuild the Removed list. Items identified as Deleted will be moved to the Deleted list.
	// Items identified as Moved/Split will be handled in Phase 2.
	finalRemoved := make([]FeatureRemoved, 0, len(w.diff.Removed))
	for _, r := range w.diff.Removed {
		// Check the current status of the removed feature ID in the database.
		result, err := w.fetcher.GetFeature(ctx, r.ID)
		if err != nil {
			// If the entity is completely gone from the DB, it's a true deletion.
			if errors.Is(err, backendtypes.ErrEntityDoesNotExist) {
				w.diff.Deleted = append(w.diff.Deleted, FeatureDeleted{
					ID:     r.ID,
					Name:   r.Name,
					Reason: ReasonDeleted,
				})

				continue
			}

			return err
		}

		// Update the visitor context so it knows which OldID owns the result we are about to visit.
		visitor.currentID = r.ID
		visitor.targetFeature = nil

		// Dispatch based on whether the feature is Regular (exists), Moved, or Split.
		if err := result.Visit(ctx, visitor); err != nil {
			return err
		}

		// If it's a Regular feature (not moved/split), calculate the Diff
		if visitor.targetFeature != nil {
			if oldF, ok := oldSnapshot[r.ID]; ok {
				if mod, changed := compareFeature(oldF, *visitor.targetFeature); changed {
					r.Diff = &mod
				}
			}
		}

		// We keep it in the Removed list for now; Phase 2 will filter it out if it was a Move/Split.
		finalRemoved = append(finalRemoved, r)
	}
	// Update the diff with the (now smaller) list of removed items to check.
	if len(finalRemoved) > 0 {
		w.diff.Removed = finalRemoved
	} else {
		// If no items remain in the finalRemoved list, clear it.
		w.diff.Removed = nil
	}

	// Phase 2: Correlation
	// If we found any history records, try to match them with the 'Added' list.
	if len(renames) > 0 {
		w.reconcileMoves(ctx, renames, newSnapshot)
	}
	if len(splits) > 0 {
		w.reconcileSplits(ctx, splits, newSnapshot)
	}

	return nil
}

// reconciliationVisitor implements the Visitor pattern to populate the renames/splits maps
// based on the polymorphic result returned by GetFeature.
type reconciliationVisitor struct {
	currentID     string
	renames       map[string]string
	splits        map[string][]string
	targetFeature *comparables.Feature
}

func (v *reconciliationVisitor) VisitRegularFeature(_ context.Context, res backendtypes.RegularFeatureResult) error {
	// Feature exists and is regular. Convert it so we can calculate the diff.
	feat := comparables.NewFeatureFromBackendFeature(*res.Feature())
	v.targetFeature = &feat

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
func (w *FeatureDiffWorkflow) reconcileMoves(ctx context.Context,
	renames map[string]string, newSnapshot map[string]comparables.Feature) {
	// Index the Added list for O(1) lookups
	addedMap := make(map[string]FeatureAdded)
	for _, a := range w.diff.Added {
		addedMap[a.ID] = a
	}

	var newRemoved []FeatureRemoved
	newMoves := w.diff.Moves

	for _, r := range w.diff.Removed {
		newID, isRenamed := renames[r.ID]
		if !isRenamed {
			newRemoved = append(newRemoved, r)

			continue
		}

		target, isAdded := addedMap[newID]
		if !isAdded {
			// The feature moved to something that wasn't "Added".
			// It might be an existing feature (Match) or completely out of scope (NoMatch).
			toName := newID
			matchStatus := QueryMatchNoMatch

			if feat, ok := newSnapshot[newID]; ok {
				toName = feat.Name.Value
				matchStatus = QueryMatchMatch
			} else {
				// Fetch the name from the DB since it's not in the snapshot.
				if res, err := w.fetcher.GetFeature(ctx, newID); err == nil {
					_ = res.Visit(ctx, &nameFetcherVisitor{name: &toName})
				}
			}

			newMoves = append(newMoves, FeatureMoved{
				FromID:     r.ID,
				ToID:       newID,
				FromName:   r.Name,
				ToName:     toName,
				QueryMatch: matchStatus,
			})

			continue
		}

		newMoves = append(newMoves, FeatureMoved{
			FromID:     r.ID,
			ToID:       newID,
			FromName:   r.Name,
			ToName:     target.Name,
			QueryMatch: QueryMatchMatch,
		})
		// Consume the added item so it doesn't appear as a standalone "Added" event later.
		delete(addedMap, newID)
	}

	// Reconstruct the Added list with only the remaining (unclaimed) items.
	var newAdded []FeatureAdded
	for _, a := range w.diff.Added {
		if _, exists := addedMap[a.ID]; exists {
			newAdded = append(newAdded, a)
		}
	}

	w.diff.Removed = newRemoved
	w.diff.Added = newAdded
	w.diff.Moves = newMoves
}

// reconcileSplits modifies the diff in-place. It pairs Removed items with one or more Added items
// based on the provided splits map (OldID -> [NewID...]).
func (w *FeatureDiffWorkflow) reconcileSplits(ctx context.Context,
	splits map[string][]string, newSnapshot map[string]comparables.Feature) {
	addedMap := make(map[string]FeatureAdded)
	for _, a := range w.diff.Added {
		addedMap[a.ID] = a
	}

	var newRemoved []FeatureRemoved
	newSplits := w.diff.Splits

	for _, r := range w.diff.Removed {
		targetIDs, isSplit := splits[r.ID]
		if !isSplit {
			newRemoved = append(newRemoved, r)

			continue
		}

		var foundTargets []FeatureAdded
		for _, targetID := range targetIDs {
			if target, isAdded := addedMap[targetID]; isAdded {
				target.QueryMatch = QueryMatchMatch
				foundTargets = append(foundTargets, target)
				// Consume the added item
				delete(addedMap, targetID)

				continue
			}

			// Not in Added list. Check snapshot or DB.
			toName := targetID
			matchStatus := QueryMatchNoMatch

			if feat, ok := newSnapshot[targetID]; ok {
				toName = feat.Name.Value
				matchStatus = QueryMatchMatch
			} else {
				// Fetch the name from the DB.
				if res, err := w.fetcher.GetFeature(ctx, targetID); err == nil {
					_ = res.Visit(ctx, &nameFetcherVisitor{name: &toName})
				}
			}
			foundTargets = append(foundTargets, FeatureAdded{
				ID:         targetID,
				Name:       toName,
				Reason:     ReasonUnmatched,
				Docs:       nil,
				QueryMatch: matchStatus,
			})
		}

		newSplits = append(newSplits, FeatureSplit{
			FromID:   r.ID,
			FromName: r.Name,
			To:       foundTargets,
		})
	}

	var newAdded []FeatureAdded
	for _, a := range w.diff.Added {
		if _, exists := addedMap[a.ID]; exists {
			newAdded = append(newAdded, a)
		}
	}

	w.diff.Removed = newRemoved
	w.diff.Added = newAdded
	w.diff.Splits = newSplits
}

type nameFetcherVisitor struct {
	name *string
}

func (v *nameFetcherVisitor) VisitRegularFeature(_ context.Context, res backendtypes.RegularFeatureResult) error {
	*v.name = res.Feature().Name

	return nil
}

func (v *nameFetcherVisitor) VisitMovedFeature(_ context.Context, _ backendtypes.MovedFeatureResult) error {
	return nil
}

func (v *nameFetcherVisitor) VisitSplitFeature(_ context.Context, _ backendtypes.SplitFeatureResult) error {
	return nil
}
