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

// Named comparables instead of comparable to not conflict with the standard library's "comparable" interface
package comparables

import (
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/generic"
)

type Feature struct {
	ID             string
	Name           generic.OptionallySet[string]
	BaselineStatus generic.OptionallySet[BaselineState]
	BrowserImpls   generic.OptionallySet[BrowserImplementations]
	Docs           generic.OptionallySet[Docs]
}

type BaselineState struct {
	Status   generic.OptionallySet[backend.BaselineInfoStatus]
	LowDate  generic.OptionallySet[*time.Time]
	HighDate generic.OptionallySet[*time.Time]
}

// BrowserImplementations defines the specific browsers we track.
type BrowserImplementations struct {
	Chrome         generic.OptionallySet[BrowserState]
	ChromeAndroid  generic.OptionallySet[BrowserState]
	Edge           generic.OptionallySet[BrowserState]
	Firefox        generic.OptionallySet[BrowserState]
	FirefoxAndroid generic.OptionallySet[BrowserState]
	Safari         generic.OptionallySet[BrowserState]
	SafariIos      generic.OptionallySet[BrowserState]
}

// BrowserState captures the implementation details for a specific browser.
type BrowserState struct {
	Status  generic.OptionallySet[backend.BrowserImplementationStatus]
	Date    generic.OptionallySet[*time.Time]
	Version generic.OptionallySet[*string]
}

type MdnDoc struct {
	URL   generic.OptionallySet[string]
	Title generic.OptionallySet[*string]
	Slug  generic.OptionallySet[*string]
}

type Docs struct {
	MdnDocs generic.OptionallySet[[]MdnDoc]
}

func NewFeatureMapFromBackendFeatures(features []backend.Feature) map[string]Feature {
	m := make(map[string]Feature)
	for _, f := range features {
		m[f.FeatureId] = NewFeatureFromBackendFeature(f)
	}

	return m
}

func NewFeatureFromBackendFeature(f backend.Feature) Feature {
	status := backend.Limited
	var lowDate, highDate *time.Time
	if f.Baseline != nil {
		if f.Baseline.Status != nil {
			status = *f.Baseline.Status
		}
		if f.Baseline.LowDate != nil {
			t := f.Baseline.LowDate.Time
			lowDate = &t
		}
		if f.Baseline.HighDate != nil {
			t := f.Baseline.HighDate.Time
			highDate = &t
		}
	}

	baseline := BaselineState{
		Status:   generic.OptionallySet[backend.BaselineInfoStatus]{Value: status, IsSet: true},
		LowDate:  generic.OptionallySet[*time.Time]{Value: lowDate, IsSet: true},
		HighDate: generic.OptionallySet[*time.Time]{Value: highDate, IsSet: true},
	}

	cf := Feature{
		ID:             f.FeatureId,
		Name:           generic.OptionallySet[string]{Value: f.Name, IsSet: true},
		BaselineStatus: generic.OptionallySet[BaselineState]{Value: baseline, IsSet: true},
		// TODO: Handle Docs when https://github.com/GoogleChrome/webstatus.dev/issues/930 is supported.
		Docs:         generic.UnsetOpt[Docs](),
		BrowserImpls: generic.UnsetOpt[BrowserImplementations](),
	}

	if f.BrowserImplementations == nil {
		return cf
	}

	raw := *f.BrowserImplementations
	cf.BrowserImpls = generic.OptionallySet[BrowserImplementations]{
		Value: BrowserImplementations{
			Chrome:         newBrowserStateFromBackendBrowserState(raw[string(backend.Chrome)]),
			ChromeAndroid:  newBrowserStateFromBackendBrowserState(raw[string(backend.ChromeAndroid)]),
			Edge:           newBrowserStateFromBackendBrowserState(raw[string(backend.Edge)]),
			Firefox:        newBrowserStateFromBackendBrowserState(raw[string(backend.Firefox)]),
			FirefoxAndroid: newBrowserStateFromBackendBrowserState(raw[string(backend.FirefoxAndroid)]),
			Safari:         newBrowserStateFromBackendBrowserState(raw[string(backend.Safari)]),
			SafariIos:      newBrowserStateFromBackendBrowserState(raw[string(backend.SafariIos)]),
		},
		IsSet: true,
	}

	return cf
}

// newBrowserStateFromBackendBrowserState converts a single browser implementation from the backend API
// into the canonical comparable format.
func newBrowserStateFromBackendBrowserState(impl backend.BrowserImplementation) generic.OptionallySet[BrowserState] {
	var status backend.BrowserImplementationStatus
	if impl.Status != nil {
		status = *impl.Status
	}

	var date *time.Time
	if impl.Date != nil {
		date = &impl.Date.Time
	}

	// An empty struct from the map lookup indicates the browser was not present.
	// In this case, we return an unset OptionallySet.
	if impl.Status == nil && impl.Date == nil && impl.Version == nil {
		return generic.UnsetOpt[BrowserState]()
	}

	return generic.OptionallySet[BrowserState]{
		Value: BrowserState{
			Status:  generic.OptionallySet[backend.BrowserImplementationStatus]{Value: status, IsSet: true},
			Version: generic.OptionallySet[*string]{Value: impl.Version, IsSet: impl.Version != nil},
			Date:    generic.OptionallySet[*time.Time]{Value: date, IsSet: date != nil},
		},
		IsSet: true,
	}
}
