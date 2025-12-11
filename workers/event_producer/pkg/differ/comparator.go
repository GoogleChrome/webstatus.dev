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

import "github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"

func calculateDiff(oldMap, newMap map[string]ComparableFeature) *FeatureDiff {
	diff := new(FeatureDiff)

	for id, newF := range newMap {
		oldF, exists := oldMap[id]
		if !exists {
			diff.Added = append(diff.Added, FeatureAdded{
				ID: id, Name: newF.Name.Value, Reason: ReasonNewMatch,
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
		NameChange:     nil,
		BaselineChange: nil,
		BrowserChanges: nil,
	}
	hasMods := false

	// 1. Name Change
	if oldF.Name.IsSet && oldF.Name.Value != newF.Name.Value {
		mod.NameChange = &Change[string]{From: oldF.Name.Value, To: newF.Name.Value}
		hasMods = true
	}

	// 2. Baseline Status
	if oldF.BaselineStatus.IsSet && oldF.BaselineStatus.Value != newF.BaselineStatus.Value {
		mod.BaselineChange = &Change[backend.BaselineInfoStatus]{From: oldF.BaselineStatus.Value,
			To: newF.BaselineStatus.Value}
		hasMods = true
	}

	// 3. Browser Implementations
	// We check each field individually using a helper.
	// This ensures we respect the IsSet flag for each browser independently.
	if mod.BrowserChanges == nil {
		mod.BrowserChanges = make(map[backend.SupportedBrowsers]*Change[string])
	}

	checkBrowser := func(key backend.SupportedBrowsers, oldB, newB OptionallySet[string]) {
		if oldB.IsSet && oldB.Value != newB.Value {
			mod.BrowserChanges[key] = &Change[string]{From: oldB.Value, To: newB.Value}
			hasMods = true
		}
	}

	// Note for future devs:
	// The 'exhaustive' linter checks that this map literal contains every key from the enum.
	// If we add a new browser, the linter will fail and let us know we should start calculating for that browser too.
	browserMap := map[backend.SupportedBrowsers]struct {
		Old OptionallySet[string]
		New OptionallySet[string]
	}{
		backend.Chrome:         {oldF.BrowserImpls.Chrome, newF.BrowserImpls.Chrome},
		backend.ChromeAndroid:  {oldF.BrowserImpls.ChromeAndroid, newF.BrowserImpls.ChromeAndroid},
		backend.Edge:           {oldF.BrowserImpls.Edge, newF.BrowserImpls.Edge},
		backend.Firefox:        {oldF.BrowserImpls.Firefox, newF.BrowserImpls.Firefox},
		backend.FirefoxAndroid: {oldF.BrowserImpls.FirefoxAndroid, newF.BrowserImpls.FirefoxAndroid},
		backend.Safari:         {oldF.BrowserImpls.Safari, newF.BrowserImpls.Safari},
		backend.SafariIos:      {oldF.BrowserImpls.SafariIos, newF.BrowserImpls.SafariIos},
	}

	// EXECUTE: Iterate over the map.
	for key, data := range browserMap {
		checkBrowser(key, data.Old, data.New)
	}

	return mod, hasMods
}
