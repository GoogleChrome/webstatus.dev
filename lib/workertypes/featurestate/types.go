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

package featurestate

import (
	"encoding/json"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// --- Generics ---

// OptionallySet allows distinguishing between "Missing Field" (Schema Cold Start)
// and "Zero Value" (Valid Data).
type OptionallySet[T any] struct {
	Value T
	IsSet bool
}

// IsZero enables the 'omitzero' JSON tag to work correctly.
func (o OptionallySet[T]) IsZero() bool {
	return !o.IsSet
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (o *OptionallySet[T]) UnmarshalJSON(data []byte) error {
	o.IsSet = true

	return json.Unmarshal(data, &o.Value)
}

// MarshalJSON implements the json.Marshaler interface.
func (o OptionallySet[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.Value)
}

// --- In-Memory State Types ---

// BaselineState captures the full status context for a feature's baseline.
type BaselineState struct {
	Status   OptionallySet[backend.BaselineInfoStatus] `json:"status,omitzero"`
	LowDate  OptionallySet[*time.Time]                 `json:"lowDate,omitzero"`
	HighDate OptionallySet[*time.Time]                 `json:"highDate,omitzero"`
}

// ComparableFeature is the in-memory struct used for comparison logic.
type ComparableFeature struct {
	ID             string                                `json:"id"`
	Name           OptionallySet[string]                 `json:"name,omitzero"`
	BaselineStatus OptionallySet[BaselineState]          `json:"baselineStatus,omitzero"`
	BrowserImpls   OptionallySet[BrowserImplementations] `json:"browserImplementations,omitzero"`
	Docs           OptionallySet[Docs]                   `json:"docs,omitzero"`
}

type Docs struct {
	MdnDocs OptionallySet[[]MdnDoc] `json:"mdnDocs,omitzero"`
}

type MdnDoc struct {
	URL   OptionallySet[*string] `json:"url,omitzero"`
	Title OptionallySet[*string] `json:"title,omitzero"`
	Slug  OptionallySet[*string] `json:"slug,omitzero"`
}

// BrowserImplementations defines the specific browsers we track.
type BrowserImplementations struct {
	Chrome         OptionallySet[BrowserState] `json:"chrome"`
	ChromeAndroid  OptionallySet[BrowserState] `json:"chrome_android"`
	Edge           OptionallySet[BrowserState] `json:"edge"`
	Firefox        OptionallySet[BrowserState] `json:"firefox"`
	FirefoxAndroid OptionallySet[BrowserState] `json:"firefox_android"`
	Safari         OptionallySet[BrowserState] `json:"safari"`
	SafariIos      OptionallySet[BrowserState] `json:"safari_ios"`
}

// BrowserState captures the implementation details for a specific browser.
type BrowserState struct {
	Status  OptionallySet[backend.BrowserImplementationStatus] `json:"status,omitzero"`
	Date    OptionallySet[*time.Time]                          `json:"date,omitzero"`
	Version OptionallySet[*string]                             `json:"version,omitzero"`
}

// setBrowserState is a helper to set the correct browser field in BrowserImplementations.
func (b *BrowserImplementations) SetBrowserState(browser backend.SupportedBrowsers, state OptionallySet[BrowserState]) {
	switch browser {
	case backend.Chrome:
		b.Chrome = state
	case backend.ChromeAndroid:
		b.ChromeAndroid = state
	case backend.Edge:
		b.Edge = state
	case backend.Firefox:
		b.Firefox = state
	case backend.FirefoxAndroid:
		b.FirefoxAndroid = state
	case backend.Safari:
		b.Safari = state
	case backend.SafariIos:
		b.SafariIos = state
	}
}
