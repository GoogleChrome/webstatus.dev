package differ

import (
	"encoding/json"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func newBaseFeature(id, name, status string) ComparableFeature {
	return ComparableFeature{
		ID:             id,
		Name:           OptionallySet[string]{Value: name, IsSet: true},
		BaselineStatus: OptionallySet[backend.BaselineInfoStatus]{Value: backend.BaselineInfoStatus(status), IsSet: true},
		BrowserImpls: BrowserImplementations{
			Chrome:         unsetOptionalSet[string](),
			ChromeAndroid:  unsetOptionalSet[string](),
			Edge:           unsetOptionalSet[string](),
			Firefox:        unsetOptionalSet[string](),
			FirefoxAndroid: unsetOptionalSet[string](),
			Safari:         unsetOptionalSet[string](),
			SafariIos:      unsetOptionalSet[string](),
		},
	}
}

func TestCalculateDiff(t *testing.T) {
	tests := []struct {
		name         string
		oldMap       map[string]ComparableFeature
		newMap       map[string]ComparableFeature
		wantAdded    int
		wantRemoved  int
		wantModified int
	}{
		{
			name:         "No Changes",
			oldMap:       map[string]ComparableFeature{"1": newBaseFeature("1", "A", "limited")},
			newMap:       map[string]ComparableFeature{"1": newBaseFeature("1", "A", "limited")},
			wantAdded:    0,
			wantRemoved:  0,
			wantModified: 0,
		},
		{
			name:         "Addition",
			oldMap:       map[string]ComparableFeature{},
			newMap:       map[string]ComparableFeature{"2": newBaseFeature("2", "A", "limited")},
			wantAdded:    1,
			wantRemoved:  0,
			wantModified: 0,
		},
		{
			name:         "Removal",
			oldMap:       map[string]ComparableFeature{"1": newBaseFeature("1", "A", "limited")},
			newMap:       map[string]ComparableFeature{},
			wantAdded:    0,
			wantRemoved:  1,
			wantModified: 0,
		},
		{
			name: "Modification",
			oldMap: map[string]ComparableFeature{
				"1": newBaseFeature("1", "A", "limited"),
			},
			newMap: map[string]ComparableFeature{
				"1": newBaseFeature("1", "A", "widely"),
			},
			wantAdded:    0,
			wantRemoved:  0,
			wantModified: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			diff := calculateDiff(tc.oldMap, tc.newMap)
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

func TestCompareFeature_Fields(t *testing.T) {
	tests := []struct {
		name      string
		oldF      ComparableFeature
		newF      ComparableFeature
		wantMod   bool
		checkDiff func(t *testing.T, m FeatureModified)
	}{
		{
			name:    "Name Change",
			oldF:    newBaseFeature("1", "Old Name", "limited"),
			newF:    newBaseFeature("1", "New Name", "limited"),
			wantMod: true,
			checkDiff: func(t *testing.T, m FeatureModified) {
				if m.NameChange == nil {
					t.Fatal("NameChange is nil")
				}
				if m.NameChange.From != "Old Name" || m.NameChange.To != "New Name" {
					t.Errorf("NameChange mismatch: %v", m.NameChange)
				}
			},
		},
		{
			name:    "Baseline Change",
			oldF:    newBaseFeature("1", "A", "limited"),
			newF:    newBaseFeature("1", "A", "widely"),
			wantMod: true,
			checkDiff: func(t *testing.T, m FeatureModified) {
				if m.BaselineChange == nil {
					t.Fatal("BaselineChange is nil")
				}
				if m.BaselineChange.From != backend.Limited || m.BaselineChange.To != backend.Widely {
					t.Errorf("BaselineChange mismatch: %v", m.BaselineChange)
				}
			},
		},
		{
			name: "Browser Implementation Change",
			oldF: func() ComparableFeature {
				f := newBaseFeature("1", "A", "limited")
				f.BrowserImpls.Chrome = OptionallySet[string]{Value: "unavailable", IsSet: true}

				return f
			}(),
			newF: func() ComparableFeature {
				f := newBaseFeature("1", "A", "limited")
				f.BrowserImpls.Chrome = OptionallySet[string]{Value: "available", IsSet: true}

				return f
			}(),
			wantMod: true,
			checkDiff: func(t *testing.T, m FeatureModified) {
				if len(m.BrowserChanges) == 0 {
					t.Fatal("BrowserChanges is empty")
				}
				if chg, ok := m.BrowserChanges["chrome"]; !ok || chg.To != "available" {
					t.Errorf("Chrome change mismatch: %v", chg)
				}
			},
		},
		{
			name: "Browser Implementation Schema Evolution (Missing in Old)",
			oldF: func() ComparableFeature {
				f := newBaseFeature("1", "A", "limited")
				f.BrowserImpls.Chrome = OptionallySet[string]{Value: "", IsSet: false} // Missing

				return f
			}(),
			newF: func() ComparableFeature {
				f := newBaseFeature("1", "A", "limited")
				f.BrowserImpls.Chrome = OptionallySet[string]{Value: "available", IsSet: true} // Present

				return f
			}(),
			wantMod:   false, // Should NOT detect change
			checkDiff: func(_ *testing.T, _ FeatureModified) {},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mod, changed := compareFeature(tc.oldF, tc.newF)
			if changed != tc.wantMod {
				t.Errorf("compareFeature() changed = %v, want %v", changed, tc.wantMod)
			}
			if changed && tc.checkDiff != nil {
				tc.checkDiff(t, mod)
			}
		})
	}
}

func unsetOptionalSet[T any]() OptionallySet[T] {
	var value T

	return OptionallySet[T]{
		Value: value,
		IsSet: false,
	}
}

// TestSchemaEvolution_QuietRollout demonstrates exactly how the system behaves
// when a new field is added to the code but missing from GCS blobs.
func TestSchemaEvolution_QuietRollout(t *testing.T) {
	// 1. Simulate an "Old Blob" (V1)
	// Missing "browserImplementations" entirely.
	legacyJSON := `{
		"id": "feat-123",
		"name": "My Feature",
		"baselineStatus": "limited"
	}`

	// 2. Unmarshal into Current Struct
	var oldFeature ComparableFeature
	if err := json.Unmarshal([]byte(legacyJSON), &oldFeature); err != nil {
		t.Fatalf("Failed to unmarshal legacy json: %v", err)
	}

	// Verify IsSet=false for the missing fields inside the struct
	if oldFeature.BrowserImpls.Chrome.IsSet {
		t.Fatal("Expected Chrome.IsSet to be false for legacy blob")
	}

	// 3. Create "New Live Data" (V2)
	// We populate the struct manually to simulate a live fetch
	newFeature := ComparableFeature{
		ID:             "feat-123",
		Name:           OptionallySet[string]{Value: "My Feature", IsSet: true},
		BaselineStatus: OptionallySet[backend.BaselineInfoStatus]{Value: "limited", IsSet: true},
		BrowserImpls: BrowserImplementations{
			Chrome:         OptionallySet[string]{Value: "available", IsSet: true},
			ChromeAndroid:  unsetOptionalSet[string](),
			Edge:           unsetOptionalSet[string](),
			Firefox:        unsetOptionalSet[string](),
			FirefoxAndroid: unsetOptionalSet[string](),
			Safari:         unsetOptionalSet[string](),
			SafariIos:      unsetOptionalSet[string](),
		},
	}

	// 4. Run Comparison
	mod, changed := compareFeature(oldFeature, newFeature)

	// 5. Assert "Quiet Rollout"
	// Even though Old.Chrome was unset and New.Chrome is "available",
	// the comparator should skip it because Old.Chrome.IsSet is false.
	if changed {
		t.Errorf("Quiet Rollout Failed! Expected no changes, but got: %+v", mod)
	}

	// 6. Assert Mixed Update Safety
	// Modify an existing field to ensure it is still caught.
	newFeature.BaselineStatus.Value = "newly" // Real update!

	mod, changed = compareFeature(oldFeature, newFeature)

	if !changed {
		t.Error("Mixed Update Failed! Expected baseline change to trigger alert.")
	}
	if mod.BaselineChange == nil {
		t.Error("Expected BaselineChange to be populated.")
	}
	// The new field should STILL be ignored
	if _, ok := mod.BrowserChanges["chrome"]; ok {
		t.Error("Expected BrowserChanges[chrome] to still be ignored (Quiet Rollout).")
	}
}
