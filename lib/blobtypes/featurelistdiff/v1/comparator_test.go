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
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/generic"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes/comparables"
)

func newBaseFeature(id, name string, status backend.BaselineInfoStatus) comparables.Feature {
	return comparables.Feature{
		ID:   id,
		Name: generic.OptionallySet[string]{Value: name, IsSet: true},
		BaselineStatus: generic.OptionallySet[comparables.BaselineState]{
			Value: comparables.BaselineState{
				Status: generic.OptionallySet[backend.BaselineInfoStatus]{
					Value: status, IsSet: true},
				LowDate:  generic.OptionallySet[*time.Time]{IsSet: false, Value: nil},
				HighDate: generic.OptionallySet[*time.Time]{IsSet: false, Value: nil},
			},
			IsSet: true,
		},
		BrowserImpls: generic.OptionallySet[comparables.BrowserImplementations]{
			IsSet: true,
			Value: comparables.BrowserImplementations{
				Chrome:         unsetBrowserState(),
				ChromeAndroid:  unsetBrowserState(),
				Edge:           unsetBrowserState(),
				Firefox:        unsetBrowserState(),
				FirefoxAndroid: unsetBrowserState(),
				Safari:         unsetBrowserState(),
				SafariIos:      unsetBrowserState(),
			},
		},
		Docs: generic.UnsetOpt[comparables.Docs](),
	}
}

func unsetBrowserState() generic.OptionallySet[comparables.BrowserState] {
	return generic.OptionallySet[comparables.BrowserState]{
		IsSet: false,
		Value: comparables.BrowserState{
			Status:  generic.UnsetOpt[backend.BrowserImplementationStatus](),
			Date:    generic.UnsetOpt[*time.Time](),
			Version: generic.UnsetOpt[*string](),
		},
	}
}

func TestCalculateDiff(t *testing.T) {
	tests := []struct {
		name         string
		oldMap       map[string]comparables.Feature
		newMap       map[string]comparables.Feature
		wantAdded    int
		wantRemoved  int
		wantModified int
	}{
		{
			name:         "No Changes",
			oldMap:       map[string]comparables.Feature{"1": newBaseFeature("1", "A", "limited")},
			newMap:       map[string]comparables.Feature{"1": newBaseFeature("1", "A", "limited")},
			wantAdded:    0,
			wantRemoved:  0,
			wantModified: 0,
		},
		{
			name:         "Addition",
			oldMap:       map[string]comparables.Feature{},
			newMap:       map[string]comparables.Feature{"2": newBaseFeature("2", "A", "limited")},
			wantAdded:    1,
			wantRemoved:  0,
			wantModified: 0,
		},
		{
			name:         "Removal",
			oldMap:       map[string]comparables.Feature{"1": newBaseFeature("1", "A", "limited")},
			newMap:       map[string]comparables.Feature{},
			wantAdded:    0,
			wantRemoved:  1,
			wantModified: 0,
		},
		{
			name: "Modification",
			oldMap: map[string]comparables.Feature{
				"1": newBaseFeature("1", "A", "limited"),
			},
			newMap: map[string]comparables.Feature{
				"1": newBaseFeature("1", "A", "widely"),
			},
			wantAdded:    0,
			wantRemoved:  0,
			wantModified: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := NewFeatureDiffWorkflow(nil, nil)
			w.CalculateDiff(tc.oldMap, tc.newMap)
			diff := w.diff
			if len(diff.Added) != tc.wantAdded {
				t.Errorf("Added count: got %d, want %d", len(diff.Added), tc.wantAdded)
			}
			if len(diff.Removed) != tc.wantRemoved {
				t.Errorf("Removed count: got %d, want %d", len(diff.Removed), tc.wantRemoved)
			}
			if len(diff.Modified) != tc.wantModified {
				t.Errorf("Modified count: got %d, want %d", len(diff.Modified), tc.wantModified)
			}
		})
	}
}

func TestCompareFeature_NameChange(t *testing.T) {
	oldF := newBaseFeature("1", "Old Name", "limited")
	newF := newBaseFeature("1", "New Name", "limited")

	mod, changed := compareFeature(oldF, newF)

	if !changed {
		t.Fatal("expected a change, but none was reported")
	}
	if mod.NameChange == nil {
		t.Fatal("NameChange is nil")
	}
	if mod.NameChange.From != "Old Name" || mod.NameChange.To != "New Name" {
		t.Errorf("NameChange mismatch: got %+v", mod.NameChange)
	}
	if mod.BaselineChange != nil || mod.BrowserChanges != nil || mod.DocsChange != nil {
		t.Error("unexpected changes reported for other fields")
	}
}

func TestCompareFeature_BaselineChange(t *testing.T) {
	oldF := newBaseFeature("1", "A", "limited")
	newF := newBaseFeature("1", "A", "widely")

	mod, changed := compareFeature(oldF, newF)

	if !changed {
		t.Fatal("expected a change, but none was reported")
	}
	if mod.BaselineChange == nil {
		t.Fatal("BaselineChange is nil")
	}
	if mod.BaselineChange.From.Status.Value != Limited || mod.BaselineChange.To.Status.Value != Widely {
		t.Errorf("BaselineChange mismatch: got %+v", mod.BaselineChange)
	}
	if mod.NameChange != nil || mod.BrowserChanges != nil || mod.DocsChange != nil {
		t.Error("unexpected changes reported for other fields")
	}
}

func TestCompareFeature_BrowserStatusChange(t *testing.T) {
	oldF := newBaseFeature("1", "A", "limited")
	oldF.BrowserImpls.Value.Chrome = newBrowserState(backend.Unavailable, nil, nil)

	newF := newBaseFeature("1", "A", "limited")
	newF.BrowserImpls.Value.Chrome = newBrowserState(backend.Available, nil, nil)

	mod, changed := compareFeature(oldF, newF)

	if !changed {
		t.Fatal("expected a change, but none was reported")
	}
	if len(mod.BrowserChanges) == 0 {
		t.Fatal("BrowserChanges is empty")
	}
	chg, ok := mod.BrowserChanges[Chrome]
	if !ok {
		t.Fatal("Chrome change not detected")
	}
	if chg.To.Status.Value != Available {
		t.Errorf("Chrome status change mismatch: got %v", chg.To.Status.Value)
	}
}

func TestCompareFeature_BrowserVersionChange(t *testing.T) {
	oldF := newBaseFeature("1", "A", "limited")
	oldF.BrowserImpls.Value.Chrome = newBrowserState(backend.Available, generic.ValuePtr("120"), nil)

	newF := newBaseFeature("1", "A", "limited")
	newF.BrowserImpls.Value.Chrome = newBrowserState(backend.Available, generic.ValuePtr("121"), nil)

	mod, changed := compareFeature(oldF, newF)

	if !changed {
		t.Fatal("expected a change")
	}
	chg := mod.BrowserChanges[Chrome]
	if chg.To.Version.Value == nil || *chg.To.Version.Value != "121" {
		t.Errorf("Chrome version change mismatch: got %v", chg.To.Version.Value)
	}
}

func TestCompareFeature_BrowserDateChange(t *testing.T) {
	oldDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newDate := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)

	oldF := newBaseFeature("1", "A", "limited")
	oldF.BrowserImpls.Value.Chrome = newBrowserState(backend.Available, nil, &oldDate)

	newF := newBaseFeature("1", "A", "limited")
	newF.BrowserImpls.Value.Chrome = newBrowserState(backend.Available, nil, &newDate)

	mod, changed := compareFeature(oldF, newF)

	if !changed {
		t.Fatal("expected a change")
	}
	chg := mod.BrowserChanges[Chrome]
	if chg.To.Date.Value == nil || !chg.To.Date.Value.Equal(newDate) {
		t.Errorf("Chrome date change mismatch: got %v", chg.To.Date.Value)
	}
}

func TestCompareFeature_DocsChange_DoesNotTriggerModification(t *testing.T) {
	oldF := newBaseFeature("1", "A", "limited")
	oldF.Docs = newDocs("https://example-old.com")

	newF := newBaseFeature("1", "A", "limited")
	newF.Docs = newDocs("https://example-new.com")

	mod, changed := compareFeature(oldF, newF)

	if changed {
		t.Error("docs-only change should not trigger a modification")
	}
	if mod.DocsChange == nil {
		t.Fatal("DocsChange was not populated")
	}
	if mod.DocsChange.To.MdnDocs[0].URL != "https://example-new.com" {
		t.Errorf("DocsChange.To has wrong URL: %s", mod.DocsChange.To.MdnDocs[0].URL)
	}
}

func TestCompareFeature_QuietRollout_NewBrowser(t *testing.T) {
	// Old feature is missing any data for Chrome
	oldF := newBaseFeature("1", "A", "limited")

	// New feature now has data for Chrome
	newF := newBaseFeature("1", "A", "limited")
	newF.BrowserImpls.Value.Chrome = newBrowserState(backend.Available, nil, nil)

	_, changed := compareFeature(oldF, newF)

	if changed {
		t.Error("quiet rollout of a new browser should not trigger a change")
	}
}

func TestCompareFeature_QuietRollout_NewTopLevelField(t *testing.T) {
	// Old feature is missing the entire BrowserImpls struct
	oldF := newBaseFeature("1", "A", "limited")
	oldF.BrowserImpls = generic.UnsetOpt[comparables.BrowserImplementations]()

	// New feature now has the struct and data for a browser
	newF := newBaseFeature("1", "A", "limited")
	newF.BrowserImpls.Value.Chrome = newBrowserState(backend.Available, nil, nil)

	_, changed := compareFeature(oldF, newF)

	if changed {
		t.Error("quiet rollout of a new top-level field should not trigger a change")
	}
}

// --- Test Helpers ---

func newBrowserState(
	status backend.BrowserImplementationStatus,
	version *string,
	date *time.Time,
) generic.OptionallySet[comparables.BrowserState] {
	return generic.OptionallySet[comparables.BrowserState]{
		Value: comparables.BrowserState{
			Status:  generic.OptionallySet[backend.BrowserImplementationStatus]{Value: status, IsSet: true},
			Version: generic.OptionallySet[*string]{Value: version, IsSet: version != nil},
			Date:    generic.OptionallySet[*time.Time]{Value: date, IsSet: date != nil},
		},
		IsSet: true,
	}
}

func newDocs(url string) generic.OptionallySet[comparables.Docs] {
	return generic.OptionallySet[comparables.Docs]{
		IsSet: true,
		Value: comparables.Docs{
			MdnDocs: generic.OptionallySet[[]comparables.MdnDoc]{
				IsSet: true,
				Value: []comparables.MdnDoc{
					{
						URL:   generic.OptionallySet[string]{Value: url, IsSet: true},
						Title: generic.OptionallySet[*string]{Value: generic.ValuePtr("Example"), IsSet: true},
						Slug:  generic.OptionallySet[*string]{Value: generic.ValuePtr("example"), IsSet: true},
					},
				},
			},
		},
	}
}
