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
	"cmp"
	"reflect"
	"slices"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func calculateDiff(oldMap, newMap map[string]ComparableFeature) *LatestFeatureDiff {
	diff := new(LatestFeatureDiff)

	for id, newF := range newMap {
		oldF, exists := oldMap[id]
		if !exists {
			var docs *Docs
			if newF.Docs.IsSet {
				docs = &newF.Docs.Value
			}
			diff.Added = append(diff.Added, FeatureAdded{
				ID: id, Name: newF.Name.Value, Docs: docs, Reason: ReasonNewMatch,
			})

			continue
		}

		if mod, changed := compareFeature(oldF, newF); changed {
			diff.Modified = append(diff.Modified, mod)
		}
	}

	for id, oldF := range oldMap {
		if _, exists := newMap[id]; !exists {
			diff.Removed = append(diff.Removed, FeatureRemoved{
				ID: id, Name: oldF.Name.Value, Reason: ReasonUnmatched,
			})
		}
	}

	return diff
}

func compareFeature(oldF, newF ComparableFeature) (FeatureModified, bool) {
	mod := FeatureModified{
		ID:             newF.ID,
		Name:           newF.Name.Value,
		Docs:           nil,
		NameChange:     nil,
		BaselineChange: nil,
		BrowserChanges: nil,
		DocsChange:     nil,
	}
	hasMods := false

	// Compare each part of the feature.
	nameChange, nameHasChanged := compareName(oldF.Name, newF.Name)
	if nameHasChanged {
		mod.NameChange = nameChange
		hasMods = true
	}

	baselineChange, baselineHasChanged := compareBaseline(oldF.BaselineStatus, newF.BaselineStatus)
	if baselineHasChanged {
		mod.BaselineChange = baselineChange
		hasMods = true

	}

	browserChanges, browserHasChanged := compareBrowserImpls(oldF.BrowserImpls, newF.BrowserImpls)
	if browserHasChanged {
		mod.BrowserChanges = browserChanges
		hasMods = true
	}

	docsChange, docsHaveChanged := compareDocs(oldF.Docs, newF.Docs)
	if docsHaveChanged {
		mod.DocsChange = docsChange
		hasMods = true
	}

	return mod, hasMods
}

// compareName checks for a name change.
func compareName(oldName, newName OptionallySet[string]) (*Change[string], bool) {
	if oldName.IsSet && oldName.Value != newName.Value {
		return &Change[string]{From: oldName.Value, To: newName.Value}, true
	}

	return nil, false
}

// compareBaseline checks for a baseline status change.
func compareBaseline(
	oldStatus, newStatus OptionallySet[BaselineState]) (*Change[BaselineState], bool) {
	if oldStatus.IsSet {
		oldBase := oldStatus.Value
		newBase := newStatus.Value
		if oldBase.Status.IsSet && oldBase.Status.Value != newBase.Status.Value {
			return &Change[BaselineState]{
				From: oldBase,
				To:   newBase,
			}, true
		}
	}

	return nil, false
}

// compareBrowserImpls checks for changes in browser implementations.
func compareBrowserImpls(
	oldImpls, newImpls OptionallySet[BrowserImplementations]) (map[backend.SupportedBrowsers]*Change[BrowserState], bool) {
	changes := make(map[backend.SupportedBrowsers]*Change[BrowserState])
	hasChanged := false

	if !oldImpls.IsSet {
		return changes, false
	}

	oldB := oldImpls.Value
	newB := newImpls.Value

	browserMap := map[backend.SupportedBrowsers]struct {
		Old OptionallySet[BrowserState]
		New OptionallySet[BrowserState]
	}{
		backend.Chrome:         {oldB.Chrome, newB.Chrome},
		backend.ChromeAndroid:  {oldB.ChromeAndroid, newB.ChromeAndroid},
		backend.Edge:           {oldB.Edge, newB.Edge},
		backend.Firefox:        {oldB.Firefox, newB.Firefox},
		backend.FirefoxAndroid: {oldB.FirefoxAndroid, newB.FirefoxAndroid},
		backend.Safari:         {oldB.Safari, newB.Safari},
		backend.SafariIos:      {oldB.SafariIos, newB.SafariIos},
	}

	for key, data := range browserMap {
		if change, changed := compareBrowserState(data.Old, data.New); changed {
			changes[key] = change
			hasChanged = true
		}
	}

	return changes, hasChanged
}

// compareBrowserState checks for changes in a single browser's state.
func compareBrowserState(oldB, newB OptionallySet[BrowserState]) (*Change[BrowserState], bool) {
	if !oldB.IsSet {
		return nil, false
	}
	// Check Status
	isChanged := oldB.Value.Status.IsSet && oldB.Value.Status.Value != newB.Value.Status.Value
	// Check Version
	if !isChanged && oldB.Value.Version.IsSet && !pointersEqual(
		oldB.Value.Version.Value, newB.Value.Version.Value) {
		isChanged = true
	}
	// Check Date
	if !isChanged && oldB.Value.Date.IsSet && !pointersEqualFn(
		oldB.Value.Date.Value, newB.Value.Date.Value, timeEqual) {
		isChanged = true
	}

	if isChanged {
		return &Change[BrowserState]{
			From: oldB.Value,
			To:   newB.Value,
		}, true
	}

	return nil, false
}

// compareDocs checks for changes in the documentation links.
func compareDocs(oldDocs, newDocs OptionallySet[Docs]) (*Change[Docs], bool) {
	if !oldDocs.IsSet {
		return nil, false
	}
	if !oldDocs.Value.MdnDocs.IsSet {
		return nil, false
	}

	oldMdnDocs := oldDocs.Value.MdnDocs.Value
	newMdnDocs := newDocs.Value.MdnDocs.Value
	sortMDNDocs := func(a, b MdnDoc) int {
		if a.URL.Value == nil && b.URL.Value == nil {
			return 0
		}
		if a.URL.Value == nil && b.URL.Value != nil {
			return -1
		}
		if a.URL.Value != nil && b.URL.Value == nil {
			return 1
		}

		return cmp.Compare(*a.URL.Value, *b.URL.Value)
	}
	slices.SortFunc(oldMdnDocs, sortMDNDocs)
	slices.SortFunc(newMdnDocs, sortMDNDocs)
	mdnDocsEqual := func(a, b MdnDoc) bool {
		return reflect.DeepEqual(a.URL.Value, b.URL.Value)
	}
	if !slices.EqualFunc(oldMdnDocs, newMdnDocs, mdnDocsEqual) {
		return &Change[Docs]{
			From: oldDocs.Value,
			To:   newDocs.Value,
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
