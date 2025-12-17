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
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func newBaseFeature(name, status string) ComparableFeature {

	return ComparableFeature{

		ID:   "1",
		Name: OptionallySet[string]{Value: name, IsSet: true},
		BaselineStatus: OptionallySet[BaselineState]{
			Value: BaselineState{
				Status:   OptionallySet[backend.BaselineInfoStatus]{Value: backend.BaselineInfoStatus(status), IsSet: true},
				LowDate:  OptionallySet[*time.Time]{IsSet: false, Value: nil},
				HighDate: OptionallySet[*time.Time]{IsSet: false, Value: nil},
			},
			IsSet: true,
		},
		Docs: OptionallySet[Docs]{
			IsSet: true,
			Value: Docs{
				MdnDocs: OptionallySet[[]MdnDoc]{IsSet: false, Value: nil},
			},
		},
		BrowserImpls: OptionallySet[BrowserImplementations]{
			IsSet: true,
			Value: BrowserImplementations{
				Chrome:         unsetBrowserState(),
				ChromeAndroid:  unsetBrowserState(),
				Edge:           unsetBrowserState(),
				Firefox:        unsetBrowserState(),
				FirefoxAndroid: unsetBrowserState(),
				Safari:         unsetBrowserState(),
				SafariIos:      unsetBrowserState(),
			},
		},
	}
}

func unsetBrowserState() OptionallySet[BrowserState] {
	return OptionallySet[BrowserState]{
		IsSet: false,
		Value: BrowserState{
			Status:  OptionallySet[backend.BrowserImplementationStatus]{IsSet: false, Value: ""},
			Date:    OptionallySet[*time.Time]{IsSet: false, Value: nil},
			Version: OptionallySet[*string]{IsSet: false, Value: nil}},
	}
}

// Helper to quickly create a populated BrowserState for tests.
func makeBrowserState(status backend.BrowserImplementationStatus,
	ver *string, date *time.Time) OptionallySet[BrowserState] {
	return OptionallySet[BrowserState]{
		IsSet: true,
		Value: BrowserState{
			Status:  OptionallySet[backend.BrowserImplementationStatus]{Value: status, IsSet: true},
			Date:    OptionallySet[*time.Time]{Value: date, IsSet: true},
			Version: OptionallySet[*string]{Value: ver, IsSet: true},
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
			oldMap:       map[string]ComparableFeature{"1": newBaseFeature("A", "limited")},
			newMap:       map[string]ComparableFeature{"1": newBaseFeature("A", "limited")},
			wantAdded:    0,
			wantRemoved:  0,
			wantModified: 0,
		},
		{
			name:         "Modification",
			oldMap:       map[string]ComparableFeature{"1": newBaseFeature("A", "limited")},
			newMap:       map[string]ComparableFeature{"1": newBaseFeature("A", "widely")},
			wantAdded:    0,
			wantRemoved:  0,
			wantModified: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			diff := calculateDiff(tc.oldMap, tc.newMap)
			if len(diff.Modified) != tc.wantModified {
				t.Errorf("Modified count: got %d, want %d", len(diff.Modified), tc.wantModified)
			}
		})
	}
}

func TestCompareFeature_Fields(t *testing.T) {
	v110 := "110"
	v111 := "111"
	t1 := time.Now()
	t2 := t1.Add(24 * time.Hour)

	tests := []struct {
		name      string
		oldF      ComparableFeature
		newF      ComparableFeature
		wantMod   bool
		checkDiff func(t *testing.T, m FeatureModified)
	}{
		{
			name:    "Name Change",
			oldF:    newBaseFeature("Old Name", "limited"),
			newF:    newBaseFeature("New Name", "limited"),
			wantMod: true,
			checkDiff: func(t *testing.T, m FeatureModified) {
				if m.NameChange == nil {
					t.Fatal("NameChange is nil")
				}
			},
		},
		{
			name: "Browser Status Change",
			oldF: func() ComparableFeature {
				f := newBaseFeature("A", "limited")
				f.BrowserImpls.Value.Chrome = makeBrowserState("unavailable", nil, nil)

				return f
			}(),
			newF: func() ComparableFeature {
				f := newBaseFeature("A", "limited")
				f.BrowserImpls.Value.Chrome = makeBrowserState("available", nil, nil)

				return f
			}(),
			wantMod: true,
			checkDiff: func(t *testing.T, m FeatureModified) {
				if len(m.BrowserChanges) == 0 {
					t.Fatal("BrowserChanges is empty")
				}
				if chg, ok := m.BrowserChanges[backend.Chrome]; !ok || chg.To.Status.Value != "available" {
					t.Errorf("Chrome change mismatch: %v", chg)
				}
			},
		},
		{
			name: "Browser Version Change (Data Refinement)",
			oldF: func() ComparableFeature {
				f := newBaseFeature("A", "limited")
				f.BrowserImpls.Value.Chrome = makeBrowserState("available", &v110, nil)

				return f
			}(),
			newF: func() ComparableFeature {
				f := newBaseFeature("A", "limited")
				f.BrowserImpls.Value.Chrome = makeBrowserState("available", &v111, nil)

				return f
			}(),
			wantMod: true,
			checkDiff: func(t *testing.T, m FeatureModified) {
				if len(m.BrowserChanges) == 0 {
					t.Fatal("BrowserChanges is empty (Version change missed)")
				}
				chg := m.BrowserChanges[backend.Chrome]
				if *chg.From.Version.Value != "110" || *chg.To.Version.Value != "111" {
					t.Errorf("Version change mismatch: %v -> %v", chg.From.Version, chg.To.Version)
				}
			},
		},
		{
			name: "Browser Date Change",
			oldF: func() ComparableFeature {
				f := newBaseFeature("A", "limited")
				f.BrowserImpls.Value.Chrome = makeBrowserState("available", nil, &t1)

				return f
			}(),
			newF: func() ComparableFeature {
				f := newBaseFeature("A", "limited")
				f.BrowserImpls.Value.Chrome = makeBrowserState("available", nil, &t2)

				return f
			}(),
			wantMod: true,
			checkDiff: func(t *testing.T, m FeatureModified) {
				if len(m.BrowserChanges) == 0 {
					t.Fatal("BrowserChanges is empty (Date change missed)")
				}
			},
		},
		{
			name: "Browser Version Schema Evolution (Missing in Old)",
			oldF: func() ComparableFeature {
				f := newBaseFeature("A", "limited")
				// Old blob has status, but Version is missing (IsSet=false)
				f.BrowserImpls.Value.Chrome = OptionallySet[BrowserState]{
					IsSet: true,
					Value: BrowserState{
						Status:  OptionallySet[backend.BrowserImplementationStatus]{Value: "available", IsSet: true},
						Version: OptionallySet[*string]{IsSet: false, Value: nil}, // Missing field
						Date:    OptionallySet[*time.Time]{IsSet: false, Value: nil},
					},
				}

				return f
			}(),
			newF: func() ComparableFeature {
				f := newBaseFeature("A", "limited")
				// New blob has Version populated
				f.BrowserImpls.Value.Chrome = makeBrowserState("available", &v111, nil)

				return f
			}(),
			wantMod:   false, // Should NOT detect change because Version field was missing in old
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
