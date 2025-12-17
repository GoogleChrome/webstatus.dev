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
	v1 "github.com/GoogleChrome/webstatus.dev/lib/workertypes/featurediff/v1"
)

func calculateDiff(oldMap, newMap map[string]ComparableFeature) *v1.LatestFeatureDiff {
	diff := new(v1.LatestFeatureDiff)

	for id, newF := range newMap {
		oldF, exists := oldMap[id]
		if !exists {
			var docs *v1.Docs
			if newF.Docs.IsSet {
				docs = valuePtr(toV1Docs(newF.Docs.Value))
			}
			diff.Added = append(diff.Added, v1.FeatureAdded{
				ID: id, Name: newF.Name.Value, Docs: docs, Reason: v1.ReasonNewMatch,
			})

			continue
		}

		if mod, changed := compareFeature(oldF, newF); changed {
			diff.Modified = append(diff.Modified, mod)
		}
	}

	for id, oldF := range oldMap {
		if _, exists := newMap[id]; !exists {
			diff.Removed = append(diff.Removed, v1.FeatureRemoved{
				ID: id, Name: oldF.Name.Value, Reason: v1.ReasonUnmatched,
			})
		}
	}

	return diff
}

func valuePtr[T any](v T) *T {
	return &v
}

func compareFeature(oldF, newF ComparableFeature) (v1.FeatureModified, bool) {
	mod := v1.FeatureModified{
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
func compareName(oldName, newName OptionallySet[string]) (*v1.Change[string], bool) {
	if oldName.IsSet && oldName.Value != newName.Value {
		return &v1.Change[string]{From: oldName.Value, To: newName.Value}, true
	}

	return nil, false
}

// compareBaseline checks for a baseline status change.
func compareBaseline(
	oldStatus, newStatus OptionallySet[BaselineState]) (*v1.Change[v1.BaselineState], bool) {
	if oldStatus.IsSet {
		oldBase := oldStatus.Value
		newBase := newStatus.Value
		if oldBase.Status.IsSet && oldBase.Status.Value != newBase.Status.Value { //nolint:staticcheck
			return &v1.Change[v1.BaselineState]{
				From: toV1BaselineState(oldBase),
				To:   toV1BaselineState(newBase),
			}, true
		}
	}

	return nil, false
}

// compareBrowserImpls checks for changes in browser implementations.
func compareBrowserImpls(
	oldImpls, newImpls OptionallySet[BrowserImplementations],
) (map[backend.SupportedBrowsers]*v1.Change[v1.BrowserState], bool) {
	changes := make(map[backend.SupportedBrowsers]*v1.Change[v1.BrowserState])
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
			changes[key] = toV1BrowserChange(change)
			hasChanged = true
		}
	}

	return changes, hasChanged
}

// compareBrowserState checks for changes in a single browser's state.
func compareBrowserState(oldB, newB OptionallySet[BrowserState]) (*v1.Change[BrowserState], bool) {
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
		return &v1.Change[BrowserState]{
			From: oldB.Value,
			To:   newB.Value,
		}, true
	}

	return nil, false
}

// compareDocs checks for changes in the documentation links.
func compareDocs(oldDocs, newDocs OptionallySet[Docs]) (*v1.Change[v1.Docs], bool) {
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
		return &v1.Change[v1.Docs]{
			From: toV1Docs(oldDocs.Value),
			To:   toV1Docs(newDocs.Value),
		}, true
	}

	return nil, false
}

func toV1BaselineState(bs BaselineState) v1.BaselineState {
	var status backend.BaselineInfoStatus
	if bs.Status.IsSet {
		status = bs.Status.Value
	}

	// Pointers are copied directly, as they are already *time.Time
	return v1.BaselineState{
		Status:   status,
		LowDate:  bs.LowDate.Value,
		HighDate: bs.HighDate.Value,
	}
}

func toV1BrowserChange(change *v1.Change[BrowserState]) *v1.Change[v1.BrowserState] {
	if change == nil {
		return nil
	}

	return &v1.Change[v1.BrowserState]{
		From: v1.BrowserState{
			Status:  change.From.Status.Value,
			Version: change.From.Version.Value,
			Date:    change.From.Date.Value,
		},
		To: v1.BrowserState{
			Status:  change.To.Status.Value,
			Version: change.To.Version.Value,
			Date:    change.To.Date.Value,
		},
	}
}

func toV1Docs(d Docs) v1.Docs {
	var mdnDocs []v1.MdnDoc
	if d.MdnDocs.IsSet {
		for _, doc := range d.MdnDocs.Value {
			var url, title, slug *string
			if doc.URL.IsSet {
				url = doc.URL.Value
			}
			if doc.Title.IsSet {
				title = doc.Title.Value
			}
			if doc.Slug.IsSet {
				slug = doc.Slug.Value
			}
			mdnDocs = append(mdnDocs, v1.MdnDoc{URL: url, Title: title, Slug: slug})
		}
	}

	return v1.Docs{MdnDocs: mdnDocs}
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
