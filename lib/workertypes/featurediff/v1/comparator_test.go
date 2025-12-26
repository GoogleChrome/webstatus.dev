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
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes/featurestate"
)

func newBaseFeature(name string) featurestate.ComparableFeature {
	return featurestate.ComparableFeature{
		ID:   "1",
		Name: featurestate.OptionallySet[string]{Value: name, IsSet: true},
		BaselineStatus: featurestate.OptionallySet[featurestate.BaselineState]{
			Value: featurestate.BaselineState{
				Status: featurestate.OptionallySet[backend.BaselineInfoStatus]{
					Value: backend.BaselineInfoStatus("limited"), IsSet: true},
				LowDate:  featurestate.OptionallySet[*time.Time]{IsSet: false, Value: nil},
				HighDate: featurestate.OptionallySet[*time.Time]{IsSet: false, Value: nil},
			},
			IsSet: true,
		},
		Docs: featurestate.OptionallySet[featurestate.Docs]{
			IsSet: true,
			Value: featurestate.Docs{
				MdnDocs: featurestate.OptionallySet[[]featurestate.MdnDoc]{IsSet: false, Value: nil},
			},
		},
		BrowserImpls: featurestate.OptionallySet[featurestate.BrowserImplementations]{
			IsSet: true,
			Value: featurestate.BrowserImplementations{
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

func unsetBrowserState() featurestate.OptionallySet[featurestate.BrowserState] {
	return featurestate.OptionallySet[featurestate.BrowserState]{
		IsSet: false,
		Value: featurestate.BrowserState{
			Status:  featurestate.OptionallySet[backend.BrowserImplementationStatus]{IsSet: false, Value: ""},
			Date:    featurestate.OptionallySet[*time.Time]{IsSet: false, Value: nil},
			Version: featurestate.OptionallySet[*string]{IsSet: false, Value: nil}},
	}
}

func makeBrowserState(status backend.BrowserImplementationStatus,
	ver *string, date *time.Time) featurestate.OptionallySet[featurestate.BrowserState] {
	return featurestate.OptionallySet[featurestate.BrowserState]{
		IsSet: true,
		Value: featurestate.BrowserState{
			Status:  featurestate.OptionallySet[backend.BrowserImplementationStatus]{Value: status, IsSet: true},
			Date:    featurestate.OptionallySet[*time.Time]{Value: date, IsSet: true},
			Version: featurestate.OptionallySet[*string]{Value: ver, IsSet: true},
		},
	}
}

func TestCompareFeature_Fields(t *testing.T) {
	v110 := "110"
	v111 := "111"
	t1 := time.Now()
	t2 := t1.Add(24 * time.Hour)

	tests := []struct {
		name      string
		oldF      featurestate.ComparableFeature
		newF      featurestate.ComparableFeature
		wantMod   bool
		checkDiff func(t *testing.T, m FeatureModified)
	}{
		{
			name:    "Name Change",
			oldF:    newBaseFeature("Old Name"),
			newF:    newBaseFeature("New Name"),
			wantMod: true,
			checkDiff: func(t *testing.T, m FeatureModified) {
				if m.NameChange == nil {
					t.Fatal("NameChange is nil")
				}
			},
		},
		{
			name: "Browser Status Change",
			oldF: func() featurestate.ComparableFeature {
				f := newBaseFeature("A")
				f.BrowserImpls.Value.SetBrowserState(backend.Chrome, makeBrowserState("unavailable", nil, nil))

				return f
			}(),
			newF: func() featurestate.ComparableFeature {
				f := newBaseFeature("A")
				f.BrowserImpls.Value.SetBrowserState(backend.Chrome, makeBrowserState("available", nil, nil))

				return f
			}(),
			wantMod: true,
			checkDiff: func(t *testing.T, m FeatureModified) {
				if len(m.BrowserChanges) == 0 {
					t.Fatal("BrowserChanges is empty")
				}
				if chg, ok := m.BrowserChanges[backend.Chrome]; !ok || chg.To.Status != "available" {
					t.Errorf("Chrome change mismatch: %v", chg)
				}
			},
		},
		{
			name: "Browser Version Change (Data Refinement)",
			oldF: func() featurestate.ComparableFeature {
				f := newBaseFeature("A")
				f.BrowserImpls.Value.SetBrowserState(backend.Chrome, makeBrowserState("available", &v110, nil))

				return f
			}(),
			newF: func() featurestate.ComparableFeature {
				f := newBaseFeature("A")
				f.BrowserImpls.Value.SetBrowserState(backend.Chrome, makeBrowserState("available", &v111, nil))

				return f
			}(),
			wantMod: true,
			checkDiff: func(t *testing.T, m FeatureModified) {
				if len(m.BrowserChanges) == 0 {
					t.Fatal("BrowserChanges is empty (Version change missed)")
				}
				chg := m.BrowserChanges[backend.Chrome]
				if *chg.From.Version != "110" || *chg.To.Version != "111" {
					t.Errorf("Version change mismatch: %v -> %v", *chg.From.Version, *chg.To.Version)
				}
			},
		},
		{
			name: "Browser Date Change",
			oldF: func() featurestate.ComparableFeature {
				f := newBaseFeature("A")
				f.BrowserImpls.Value.SetBrowserState(backend.Chrome, makeBrowserState("available", nil, &t1))

				return f
			}(),
			newF: func() featurestate.ComparableFeature {
				f := newBaseFeature("A")
				f.BrowserImpls.Value.SetBrowserState(backend.Chrome, makeBrowserState("available", nil, &t2))

				return f
			}(),
			wantMod: true,
			checkDiff: func(t *testing.T, m FeatureModified) {
				if len(m.BrowserChanges) == 0 {
					t.Fatal("BrowserChanges is empty (Date change missed)")
				}
			},
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
