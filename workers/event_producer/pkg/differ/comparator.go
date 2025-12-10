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
		mod.BrowserChanges = make(map[string]Change[string])
	}

	checkBrowser := func(key string, oldB, newB OptionallySet[string]) {
		// Only diff if the browser existed in the old blob
		if oldB.IsSet && oldB.Value != newB.Value {
			mod.BrowserChanges[key] = Change[string]{From: oldB.Value, To: newB.Value}
			hasMods = true
		}
	}

	oldB := oldF.BrowserImpls
	newB := newF.BrowserImpls

	// We use the backend string constants/keys for the map output
	checkBrowser("chrome", oldB.Chrome, newB.Chrome)
	checkBrowser("chrome_android", oldB.ChromeAndroid, newB.ChromeAndroid)
	checkBrowser("edge", oldB.Edge, newB.Edge)
	checkBrowser("firefox", oldB.Firefox, newB.Firefox)
	checkBrowser("firefox_android", oldB.FirefoxAndroid, newB.FirefoxAndroid)
	checkBrowser("safari", oldB.Safari, newB.Safari)
	checkBrowser("safari_ios", oldB.SafariIos, newB.SafariIos)

	return mod, hasMods
}
