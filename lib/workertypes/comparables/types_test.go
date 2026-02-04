// Copyright 2026 Google LLC
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

package comparables

import (
	"reflect"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/generic"
	"github.com/google/go-cmp/cmp"
	"github.com/oapi-codegen/runtime/types"
)

func TestNewFeatureFromBackendFeature(t *testing.T) {
	avail := backend.Available
	unavail := backend.Unavailable
	status := backend.Widely
	date := types.Date{Time: time.Now()}

	tests := []struct {
		name string
		in   backend.Feature
		want Feature
	}{
		{
			name: "Fully Populated",
			in: backend.Feature{
				FeatureId:   "feat-1",
				Name:        "Feature One",
				Spec:        nil,
				Discouraged: nil,
				Usage:       nil,
				Wpt:         nil,
				Baseline: &backend.BaselineInfo{
					Status:   &status,
					LowDate:  nil,
					HighDate: nil,
				},
				BrowserImplementations: &map[string]backend.BrowserImplementation{
					"chrome":  {Status: &avail, Date: &date, Version: generic.ValuePtr("version")},
					"firefox": {Status: &unavail, Date: nil, Version: nil},
					"safari":  {Status: &avail, Date: nil, Version: nil},
					"unknown": {Status: &avail, Date: nil, Version: nil}, // Should be ignored
				},
				VendorPositions:            nil,
				DeveloperSignals:           nil,
				SystemManagedSavedSearchId: nil,
			},
			want: createExpectedFeature("feat-1", "Feature One",
				backend.Widely, map[backend.SupportedBrowsers]BrowserState{
					backend.Chrome: {
						Status:  generic.OptionallySet[backend.BrowserImplementationStatus]{Value: backend.Available, IsSet: true},
						Date:    generic.OptionallySet[*time.Time]{Value: &date.Time, IsSet: true},
						Version: generic.OptionallySet[*string]{Value: generic.ValuePtr("version"), IsSet: true},
					},
					backend.ChromeAndroid: zero[BrowserState](),
					backend.Firefox: {
						Status:  generic.OptionallySet[backend.BrowserImplementationStatus]{Value: backend.Unavailable, IsSet: true},
						Date:    generic.UnsetOpt[*time.Time](),
						Version: generic.UnsetOpt[*string](),
					},
					backend.FirefoxAndroid: zero[BrowserState](),
					backend.Safari: {
						Status:  generic.OptionallySet[backend.BrowserImplementationStatus]{Value: backend.Available, IsSet: true},
						Date:    generic.UnsetOpt[*time.Time](),
						Version: generic.UnsetOpt[*string](),
					},
					backend.SafariIos: zero[BrowserState](),
					backend.Edge:      zero[BrowserState](),
				}),
		},
		{
			name: "Minimal (Nil Maps)",
			in: backend.Feature{
				FeatureId:                  "feat-2",
				Name:                       "Minimal Feature",
				Baseline:                   nil,
				Spec:                       nil,
				BrowserImplementations:     nil,
				Discouraged:                nil,
				Usage:                      nil,
				Wpt:                        nil,
				VendorPositions:            nil,
				DeveloperSignals:           nil,
				SystemManagedSavedSearchId: nil,
			},
			want: createExpectedFeature("feat-2", "Minimal Feature", backend.Limited, nil),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NewFeatureFromBackendFeature(tc.in)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("toComparable mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// createExpectedFeature constructs a Feature with all OptionallySet fields initialized.
// This is required to pass the exhaustruct linter in tests.
func createExpectedFeature(id, name string, baseline backend.BaselineInfoStatus,
	browsers map[backend.SupportedBrowsers]BrowserState) Feature {
	cf := Feature{
		ID:   id,
		Name: generic.OptionallySet[string]{Value: name, IsSet: true},
		BaselineStatus: generic.OptionallySet[BaselineState]{Value: BaselineState{
			Status:   generic.OptionallySet[backend.BaselineInfoStatus]{Value: baseline, IsSet: true},
			LowDate:  generic.OptionallySet[*time.Time]{Value: nil, IsSet: true},
			HighDate: generic.OptionallySet[*time.Time]{Value: nil, IsSet: true},
		}, IsSet: true},
		BrowserImpls: generic.UnsetOpt[BrowserImplementations](),
		Docs:         generic.UnsetOpt[Docs](),
	}

	// Override specific browsers if provided
	if browsers != nil {
		cf.BrowserImpls.IsSet = true
		setIfPresent(browsers, backend.Chrome, &cf.BrowserImpls.Value.Chrome)
		setIfPresent(browsers, backend.ChromeAndroid, &cf.BrowserImpls.Value.ChromeAndroid)
		setIfPresent(browsers, backend.Edge, &cf.BrowserImpls.Value.Edge)
		setIfPresent(browsers, backend.Firefox, &cf.BrowserImpls.Value.Firefox)
		setIfPresent(browsers, backend.FirefoxAndroid, &cf.BrowserImpls.Value.FirefoxAndroid)
		setIfPresent(browsers, backend.Safari, &cf.BrowserImpls.Value.Safari)
		setIfPresent(browsers, backend.SafariIos, &cf.BrowserImpls.Value.SafariIos)
	}

	return cf
}

// nolint:ireturn // WONTFIX: used for testing only.
func zero[T any]() T {
	return *new(T)
}

func setIfPresent[K comparable, V any](m map[K]V, key K, target *generic.OptionallySet[V]) {
	var zero V
	if val, ok := m[key]; ok && !reflect.DeepEqual(zero, val) {
		target.IsSet = true
		target.Value = val
	}
}
