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
	"cmp"
	"slices"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/generic"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes/comparables"
)

// CalculateDiff computes the difference between two sets of features (old and new snapshots).
//
// Comparison Logic & Schema Evolution:
// The comparator relies heavily on generic.OptionallySet[T] to handle schema evolution.
//  1. Cold Start / New Fields: If a field (like BaselineStatus) was "Unset" in the old snapshot
//     (e.g. because it didn't exist in the schema then) and is "Set" in the new snapshot,
//     this is treated as a Change. This ensures users are notified when new data becomes available.
//  2. Quiet Rollouts (Browsers): A specific exception exists for BrowserImplementations.
//     If a browser moves from "Unset" to "Set(Unavailable)" with no extra details, we IGNORE it.
//     This prevents spamming users when we add a new browser column to the DB that is mostly empty.
//
// Guide for Adding New Fields:
// When adding a new field to comparables.Feature:
// 1. Wrap it in generic.OptionallySet[T].
// 2. In your compareXYZ function, explicitly handle the 4 transition cases:
//   - !old.IsSet && !new.IsSet: Return nil (No change).
//   - !old.IsSet && new.IsSet: Return Change (Added). This handles the "Cold Start".
//   - old.IsSet && !new.IsSet: Return Change (Removed).
//   - Both Set: Compare actual values.
//     3. Consider "Quiet Rollout": If your new field will be backfilled with "default/empty" values
//     (like "Unavailable" or "Unknown") that users shouldn't be bothered about, add a check
//     in the "Added" case to return no change for those specific values.
func (w *FeatureDiffWorkflow) CalculateDiff(oldMap, newMap map[string]comparables.Feature) {
	for id, newF := range newMap {
		oldF, exists := oldMap[id]
		if !exists {
			var docs *Docs
			if newF.Docs.IsSet {
				v1Docs := toV1Docs(newF.Docs.Value)
				docs = &v1Docs
			}
			w.diff.Added = append(w.diff.Added, FeatureAdded{
				ID:         id,
				Name:       newF.Name.Value,
				Reason:     ReasonNewMatch,
				Docs:       docs,
				QueryMatch: QueryMatchMatch,
			})

			continue
		}

		if mod, changed := compareFeature(oldF, newF); changed {
			w.diff.Modified = append(w.diff.Modified, mod)
		}
	}

	for id, oldF := range oldMap {
		if _, exists := newMap[id]; !exists {
			w.diff.Removed = append(w.diff.Removed, FeatureRemoved{
				ID: id, Name: oldF.Name.Value, Reason: ReasonUnmatched,
				// Diff is not populated during comparison, only used in reconciliation for move/split detection
				Diff: nil,
			})
		}
	}
}

func compareFeature(oldF, newF comparables.Feature) (FeatureModified, bool) {
	var docs *Docs
	if newF.Docs.IsSet {
		v1Docs := toV1Docs(newF.Docs.Value)
		docs = &v1Docs
	}
	mod := FeatureModified{
		ID:             newF.ID,
		Name:           newF.Name.Value,
		Docs:           docs,
		NameChange:     nil,
		BaselineChange: nil,
		BrowserChanges: nil,
		DocsChange:     nil,
	}
	hasMods := false

	// 1. Name Change
	oldName := ""
	if oldF.Name.IsSet {
		oldName = oldF.Name.Value
	}
	newName := ""
	if newF.Name.IsSet {
		newName = newF.Name.Value
	}

	if oldName != newName {
		mod.NameChange = &Change[string]{From: oldName, To: newName}
		hasMods = true
	}

	// 2. Baseline Status
	baselineChange, baselineHasChanged := compareBaseline(oldF.BaselineStatus, newF.BaselineStatus)
	if baselineHasChanged {
		mod.BaselineChange = baselineChange
		hasMods = true
	}

	// 3. Browser Implementations
	browserChanges, browserHasChanged := compareBrowserImpls(oldF.BrowserImpls, newF.BrowserImpls)
	if browserHasChanged {
		mod.BrowserChanges = browserChanges
		hasMods = true
	}

	docsChange, docsHaveChanged := compareDocs(oldF.Docs, newF.Docs)
	if docsHaveChanged {
		mod.DocsChange = docsChange
		// Docs do not trigger a feature change
	}

	return mod, hasMods
}

// compareBaseline checks for a baseline status change.
func compareBaseline(
	oldStatus, newStatus generic.OptionallySet[comparables.BaselineState],
) (*Change[BaselineState], bool) {
	if !oldStatus.IsSet && !newStatus.IsSet {
		return nil, false
	}

	// Case 2: Added (Old Unset, New Set)
	if !oldStatus.IsSet && newStatus.IsSet {
		zero := new(BaselineState)

		return &Change[BaselineState]{
			From: *zero, // Zero value represents "None"
			To:   toV1BaselineState(newStatus.Value),
		}, true
	}

	// Case 3: Removed (Old Set, New Unset)
	if oldStatus.IsSet && !newStatus.IsSet {
		zero := new(BaselineState)

		return &Change[BaselineState]{
			From: toV1BaselineState(oldStatus.Value),
			To:   *zero, // Zero value represents "None"
		}, true
	}

	// Case 4: Both Set -> Compare Values
	oldBase := oldStatus.Value
	newBase := newStatus.Value
	if oldBase.Status.IsSet && oldBase.Status.Value != newBase.Status.Value {
		return &Change[BaselineState]{
			From: toV1BaselineState(oldBase),
			To:   toV1BaselineState(newBase),
		}, true
	}

	return nil, false
}

func toV1BaselineState(bs comparables.BaselineState) BaselineState {
	return BaselineState{
		Status:   toV1BaselineInfoStatus(bs.Status),
		LowDate:  bs.LowDate,
		HighDate: bs.HighDate,
	}
}

func toV1BaselineInfoStatus(
	status generic.OptionallySet[backend.BaselineInfoStatus],
) generic.OptionallySet[BaselineInfoStatus] {
	if !status.IsSet {
		return generic.UnsetOpt[BaselineInfoStatus]()
	}

	return generic.OptionallySet[BaselineInfoStatus]{
		IsSet: true,
		Value: toV1BaselineInfoStatusValue(status.Value),
	}
}

func toV1BaselineInfoStatusValue(status backend.BaselineInfoStatus) BaselineInfoStatus {
	switch status {
	case backend.Limited:
		return Limited
	case backend.Newly:
		return Newly
	case backend.Widely:
		return Widely
	}

	return Limited
}

// compareBrowserImpls checks for changes in browser implementations.
func compareBrowserImpls(
	oldImpls, newImpls generic.OptionallySet[comparables.BrowserImplementations],
) (map[SupportedBrowsers]*Change[BrowserState], bool) {
	changes := make(map[SupportedBrowsers]*Change[BrowserState])
	hasChanged := false

	var oldB comparables.BrowserImplementations
	if oldImpls.IsSet {
		oldB = oldImpls.Value
	}

	var newB comparables.BrowserImplementations
	if newImpls.IsSet {
		newB = newImpls.Value
	}

	browserMap := map[SupportedBrowsers]struct {
		Old generic.OptionallySet[comparables.BrowserState]
		New generic.OptionallySet[comparables.BrowserState]
	}{
		Chrome:         {Old: oldB.Chrome, New: newB.Chrome},
		ChromeAndroid:  {Old: oldB.ChromeAndroid, New: newB.ChromeAndroid},
		Edge:           {Old: oldB.Edge, New: newB.Edge},
		Firefox:        {Old: oldB.Firefox, New: newB.Firefox},
		FirefoxAndroid: {Old: oldB.FirefoxAndroid, New: newB.FirefoxAndroid},
		Safari:         {Old: oldB.Safari, New: newB.Safari},
		SafariIos:      {Old: oldB.SafariIos, New: newB.SafariIos},
	}

	for key, data := range browserMap {
		if change, changed := compareBrowserState(data.Old, data.New); changed {
			changes[key] = toV1BrowserChange(change)
			hasChanged = true
		}
	}

	return changes, hasChanged
}

// compareDocs checks for changes in the documentation links.
func compareDocs(oldDocs, newDocs generic.OptionallySet[comparables.Docs]) (*Change[Docs], bool) {
	oldMdnDocs := resolveMdnDocs(oldDocs)
	newMdnDocs := resolveMdnDocs(newDocs)

	if len(oldMdnDocs) == 0 && len(newMdnDocs) == 0 {
		return nil, false
	}

	// Sort both lists for deterministic comparison
	sortMDNDocs := func(a, b comparables.MdnDoc) int {
		return cmp.Compare(a.URL.Value, b.URL.Value)
	}
	slices.SortFunc(oldMdnDocs, sortMDNDocs)
	slices.SortFunc(newMdnDocs, sortMDNDocs)

	mdnDocsEqual := func(a, b comparables.MdnDoc) bool {
		return a.URL.Value == b.URL.Value
	}

	if !slices.EqualFunc(oldMdnDocs, newMdnDocs, mdnDocsEqual) {
		// Construct the Change object using the original wrappers (or create valid wrappers)
		return &Change[Docs]{
			From: toV1DocsFromList(oldMdnDocs),
			To:   toV1DocsFromList(newMdnDocs),
		}, true
	}

	return nil, false
}

// resolveMdnDocs extracts the MDN doc list safely from the nested Option structure.
func resolveMdnDocs(docs generic.OptionallySet[comparables.Docs]) []comparables.MdnDoc {
	if !docs.IsSet {
		return nil
	}
	if !docs.Value.MdnDocs.IsSet {
		return nil
	}

	return docs.Value.MdnDocs.Value
}

func toV1Docs(d comparables.Docs) Docs {
	var mdnDocs []MdnDoc
	if d.MdnDocs.IsSet {
		for _, doc := range d.MdnDocs.Value {
			mdnDocs = append(mdnDocs, MdnDoc{
				URL:   doc.URL.Value,
				Title: doc.Title.Value,
				Slug:  doc.Slug.Value,
			})
		}
	}

	return Docs{MdnDocs: mdnDocs}
}

// toV1DocsFromList creates a V1 Docs struct directly from a slice of comparables.MdnDoc.
func toV1DocsFromList(list []comparables.MdnDoc) Docs {
	mdnDocs := make([]MdnDoc, 0, len(list))
	for _, doc := range list {
		mdnDocs = append(mdnDocs, MdnDoc{
			URL:   doc.URL.Value,
			Title: doc.Title.Value,
			Slug:  doc.Slug.Value,
		})
	}

	return Docs{MdnDocs: mdnDocs}
}

func toV1BrowserImplementationStatus(status generic.OptionallySet[backend.BrowserImplementationStatus]) generic.
	OptionallySet[BrowserImplementationStatus] {
	if !status.IsSet {
		return generic.UnsetOpt[BrowserImplementationStatus]()
	}

	return generic.OptionallySet[BrowserImplementationStatus]{
		IsSet: true,
		Value: toV1BrowserImplementationStatusValue(status.Value),
	}
}

func toV1BrowserImplementationStatusValue(status backend.BrowserImplementationStatus) BrowserImplementationStatus {
	switch status {
	case backend.Available:
		return Available
	case backend.Unavailable:
		return Unavailable
	}

	return Unavailable
}

func toV1BrowserChange(change *Change[comparables.BrowserState]) *Change[BrowserState] {
	if change == nil {
		return nil
	}

	return &Change[BrowserState]{
		From: BrowserState{
			Status:  toV1BrowserImplementationStatus(change.From.Status),
			Version: change.From.Version,
			Date:    change.From.Date,
		},
		To: BrowserState{
			Status:  toV1BrowserImplementationStatus(change.To.Status),
			Version: change.To.Version,
			Date:    change.To.Date,
		},
	}
}

// compareBrowserState checks for changes in a single browser's state.
func compareBrowserState(
	oldB, newB generic.OptionallySet[comparables.BrowserState],
) (*Change[comparables.BrowserState], bool) {
	// Case 1: Both Unset -> No Change
	if !oldB.IsSet && !newB.IsSet {
		return nil, false
	}

	// Case 2: Added (Old Unset, New Set)
	if !oldB.IsSet && newB.IsSet {
		// Quiet Rollout Support:
		// If a new browser is added to the system (Unset -> Set), we only report it
		// if it provides meaningful info (Available, or has Version/Date).
		// If it's just "Unavailable" with no other info, we treat it as no change
		// to avoid spamming the user with "New Browser Added: Unavailable" notifications.
		val := newB.Value
		isUnavailable := val.Status.IsSet && val.Status.Value == backend.Unavailable
		hasDetails := val.Version.IsSet || val.Date.IsSet

		if isUnavailable && !hasDetails {
			return nil, false
		}
		zero := new(comparables.BrowserState)

		return &Change[comparables.BrowserState]{
			From: *zero, // Zero value represents "None"
			To:   newB.Value,
		}, true
	}

	// Case 3: Removed (Old Set, New Unset)
	if oldB.IsSet && !newB.IsSet {
		zero := new(comparables.BrowserState)

		return &Change[comparables.BrowserState]{
			From: oldB.Value,
			To:   *zero, // Zero value represents "None"
		}, true
	}

	// Case 4: Both Set -> Compare Values
	// Check Status
	isChanged := oldB.Value.Status.IsSet && oldB.Value.Status.Value != newB.Value.Status.Value
	// Check Version
	if !isChanged && oldB.Value.Version.IsSet && !pointersEqual(oldB.Value.Version.Value, newB.Value.Version.Value) {
		isChanged = true
	}
	// Check Date
	if !isChanged && oldB.Value.Date.IsSet && !pointersEqualFn(oldB.Value.Date.Value, newB.Value.Date.Value, timeEqual) {
		isChanged = true
	}

	if isChanged {
		return &Change[comparables.BrowserState]{
			From: oldB.Value,
			To:   newB.Value,
		}, true
	}

	return nil, false
}

func pointersEqual[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return *a == *b
}

func pointersEqualFn[T any](a, b *T, isEqual func(a, b T) bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return isEqual(*a, *b)
}

func timeEqual(a, b time.Time) bool { return a.Equal(b) }
