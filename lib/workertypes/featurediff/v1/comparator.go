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
	"context"
	"errors"
	"reflect"
	"slices"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes/featurestate"
)

// ComparatorV1 implements the workertypes.Comparator interface for version 1 of the diff format.
type ComparatorV1 struct {
	client workertypes.FeatureFetcher
}

func NewComparatorV1(client workertypes.FeatureFetcher) *ComparatorV1 {
	return &ComparatorV1{client: client}
}

func (c *ComparatorV1) Compare(
	oldMap, newMap map[string]featurestate.ComparableFeature,
) (*workertypes.DiffResult, error) {
	diff := new(FeatureDiffV1)

	for id, newF := range newMap {
		oldF, exists := oldMap[id]
		if !exists {
			var docs *Docs
			if newF.Docs.IsSet {
				docs = valuePtr(toV1Docs(newF.Docs.Value))
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

	return &workertypes.DiffResult{Diff: diff}, nil
}

func (c *ComparatorV1) ReconcileHistory(ctx context.Context, d workertypes.Diff) (*workertypes.DiffResult, error) {
	diff, ok := d.(*FeatureDiffV1)
	if !ok {
		return nil, errors.New("invalid diff type for V1 reconciler")
	}

	renames := make(map[string]string)
	splits := make(map[string][]string)
	visitor := &reconciliationVisitor{renames: renames, splits: splits, currentID: ""}

	for i := range diff.Removed {
		r := &diff.Removed[i]
		result, err := c.client.GetFeature(ctx, r.ID)
		if err != nil {
			if errors.Is(err, backendtypes.ErrEntityDoesNotExist) {
				r.Reason = ReasonDeleted

				continue
			}

			return nil, err
		}

		visitor.currentID = r.ID
		if err := result.Visit(ctx, visitor); err != nil {
			return nil, err
		}
	}

	if len(renames) > 0 {
		reconcileMoves(diff, renames)
	}
	if len(splits) > 0 {
		reconcileSplits(diff, splits)
	}

	return &workertypes.DiffResult{Diff: diff}, nil
}

func valuePtr[T any](v T) *T {
	return &v
}

func compareFeature(oldF, newF featurestate.ComparableFeature) (FeatureModified, bool) {
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

func compareName(oldName, newName featurestate.OptionallySet[string]) (*Change[string], bool) {
	if oldName.IsSet && oldName.Value != newName.Value {
		return &Change[string]{From: oldName.Value, To: newName.Value}, true
	}

	return nil, false
}

func compareBaseline(
	oldStatus, newStatus featurestate.OptionallySet[featurestate.BaselineState],
) (*Change[BaselineState], bool) {
	if oldStatus.IsSet {
		oldBase := oldStatus.Value
		newBase := newStatus.Value
		if oldBase.Status.IsSet && oldBase.Status.Value != newBase.Status.Value {
			return &Change[BaselineState]{
				From: toV1BaselineState(oldBase),
				To:   toV1BaselineState(newBase),
			}, true
		}
	}

	return nil, false
}

func compareBrowserImpls(
	oldImpls, newImpls featurestate.OptionallySet[featurestate.BrowserImplementations],
) (map[backend.SupportedBrowsers]*Change[BrowserState], bool) {
	changes := make(map[backend.SupportedBrowsers]*Change[BrowserState])
	hasChanged := false

	if !oldImpls.IsSet {
		return changes, false
	}

	oldB := oldImpls.Value
	newB := newImpls.Value

	browserMap := map[backend.SupportedBrowsers]struct {
		Old featurestate.OptionallySet[featurestate.BrowserState]
		New featurestate.OptionallySet[featurestate.BrowserState]
	}{
		backend.Chrome:         {Old: oldB.Chrome, New: newB.Chrome},
		backend.ChromeAndroid:  {Old: oldB.ChromeAndroid, New: newB.ChromeAndroid},
		backend.Edge:           {Old: oldB.Edge, New: newB.Edge},
		backend.Firefox:        {Old: oldB.Firefox, New: newB.Firefox},
		backend.FirefoxAndroid: {Old: oldB.FirefoxAndroid, New: newB.FirefoxAndroid},
		backend.Safari:         {Old: oldB.Safari, New: newB.Safari},
		backend.SafariIos:      {Old: oldB.SafariIos, New: newB.SafariIos},
	}

	for key, data := range browserMap {
		if change, changed := compareBrowserState(data.Old, data.New); changed {
			changes[key] = toV1BrowserChange(change)
			hasChanged = true
		}
	}

	return changes, hasChanged
}

func compareBrowserState(
	oldB, newB featurestate.OptionallySet[featurestate.BrowserState],
) (*Change[featurestate.BrowserState], bool) {
	if !oldB.IsSet {
		return nil, false
	}
	isChanged := oldB.Value.Status.IsSet && oldB.Value.Status.Value != newB.Value.Status.Value
	if !isChanged && oldB.Value.Version.IsSet && !pointersEqual(oldB.Value.Version.Value, newB.Value.Version.Value) {
		isChanged = true
	}
	if !isChanged && oldB.Value.Date.IsSet && !pointersEqualFn(oldB.Value.Date.Value, newB.Value.Date.Value, timeEqual) {
		isChanged = true
	}

	if isChanged {
		return &Change[featurestate.BrowserState]{
			From: oldB.Value,
			To:   newB.Value,
		}, true
	}

	return nil, false
}

func compareDocs(oldDocs, newDocs featurestate.OptionallySet[featurestate.Docs]) (*Change[Docs], bool) {
	if !oldDocs.IsSet || !oldDocs.Value.MdnDocs.IsSet {

		return nil, false
	}

	oldMdnDocs := oldDocs.Value.MdnDocs.Value
	newMdnDocs := newDocs.Value.MdnDocs.Value
	sortMDNDocs := func(a, b featurestate.MdnDoc) int {
		if a.URL.Value == nil && b.URL.Value == nil {
			return 0
		}
		if a.URL.Value == nil {
			return -1
		}
		if b.URL.Value == nil {
			return 1
		}

		return cmp.Compare(*a.URL.Value, *b.URL.Value)
	}
	slices.SortFunc(oldMdnDocs, sortMDNDocs)
	slices.SortFunc(newMdnDocs, sortMDNDocs)

	mdnDocsEqual := func(a, b featurestate.MdnDoc) bool {
		return reflect.DeepEqual(a.URL.Value, b.URL.Value)
	}
	if !slices.EqualFunc(oldMdnDocs, newMdnDocs, mdnDocsEqual) {
		return &Change[Docs]{
			From: toV1Docs(oldDocs.Value),
			To:   toV1Docs(newDocs.Value),
		}, true
	}

	return nil, false
}

func toV1Docs(d featurestate.Docs) Docs {
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

func toV1BaselineState(bs featurestate.BaselineState) BaselineState {
	var status backend.BaselineInfoStatus
	if bs.Status.IsSet {
		status = bs.Status.Value
	}

	return BaselineState{
		Status:   status,
		LowDate:  bs.LowDate.Value,
		HighDate: bs.HighDate.Value,
	}
}

func toV1BrowserChange(change *Change[featurestate.BrowserState]) *Change[BrowserState] {
	if change == nil {
		return nil
	}

	return &Change[BrowserState]{
		From: BrowserState{
			Status:  change.From.Status.Value,
			Version: change.From.Version.Value,
			Date:    change.From.Date.Value,
		},
		To: BrowserState{
			Status:  change.To.Status.Value,
			Version: change.To.Version.Value,
			Date:    change.To.Date.Value,
		},
	}
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
