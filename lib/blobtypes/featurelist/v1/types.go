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
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/generic"
)

const (
	// KindFeatureListSnapshot identifies a full state dump of features.
	KindFeatureListSnapshot = "FeatureListSnapshot"

	// VersionFeatureListSnapshot identifies v1 of the FeatureListSnapshot schema.
	V1FeatureListSnapshot = "v1"
)

// FeatureListSnapshot represents the persisted state of a search.
type FeatureListSnapshot struct {
	Metadata StateMetadata   `json:"metadata"`
	Data     FeatureListData `json:"data"`
}

func (s FeatureListSnapshot) Kind() string    { return KindFeatureListSnapshot }
func (s FeatureListSnapshot) Version() string { return V1FeatureListSnapshot }
func (s FeatureListSnapshot) ID() string      { return s.Metadata.ID }

type FeatureListData struct {
	Features map[string]Feature `json:"features"`
}

type SupportedBrowsers string

const (
	Chrome         SupportedBrowsers = "chrome"
	ChromeAndroid  SupportedBrowsers = "chrome_android"
	Edge           SupportedBrowsers = "edge"
	Firefox        SupportedBrowsers = "firefox"
	FirefoxAndroid SupportedBrowsers = "firefox_android"
	Safari         SupportedBrowsers = "safari"
	SafariIos      SupportedBrowsers = "safari_ios"
)

type BrowserImplementationStatus string

const (
	Available   BrowserImplementationStatus = "available"
	Unavailable BrowserImplementationStatus = "unavailable"
)

type BaselineInfoStatus string

const (
	Limited BaselineInfoStatus = "limited"
	Newly   BaselineInfoStatus = "newly"
	Widely  BaselineInfoStatus = "widely"
)

type BaselineState struct {
	Status   generic.OptionallySet[BaselineInfoStatus] `json:"status,omitzero"`
	LowDate  generic.OptionallySet[*time.Time]         `json:"lowDate,omitzero"`
	HighDate generic.OptionallySet[*time.Time]         `json:"highDate,omitzero"`
}

type Feature struct {
	ID             string                                        `json:"id"`
	Name           generic.OptionallySet[string]                 `json:"name,omitzero"`
	BaselineStatus generic.OptionallySet[BaselineState]          `json:"baselineStatus,omitzero"`
	BrowserImpls   generic.OptionallySet[BrowserImplementations] `json:"browserImplementations,omitzero"`
	Docs           generic.OptionallySet[Docs]                   `json:"docs,omitzero"`
}

// BrowserImplementations defines the specific browsers we track.
type BrowserImplementations struct {
	Chrome         generic.OptionallySet[BrowserState] `json:"chrome,omitzero"`
	ChromeAndroid  generic.OptionallySet[BrowserState] `json:"chrome_android,omitzero"`
	Edge           generic.OptionallySet[BrowserState] `json:"edge,omitzero"`
	Firefox        generic.OptionallySet[BrowserState] `json:"firefox,omitzero"`
	FirefoxAndroid generic.OptionallySet[BrowserState] `json:"firefox_android,omitzero"`
	Safari         generic.OptionallySet[BrowserState] `json:"safari,omitzero"`
	SafariIos      generic.OptionallySet[BrowserState] `json:"safari_ios,omitzero"`
}

// BrowserState captures the implementation details for a specific browser.
type BrowserState struct {
	Status  generic.OptionallySet[BrowserImplementationStatus] `json:"status,omitzero"`
	Date    generic.OptionallySet[*time.Time]                  `json:"date,omitzero"`
	Version generic.OptionallySet[*string]                     `json:"version,omitzero"`
}

type StateMetadata struct {
	ID             string    `json:"id"`
	GeneratedAt    time.Time `json:"generatedAt"`
	SearchID       string    `json:"searchId"`
	QuerySignature string    `json:"querySignature"`
	EventID        string    `json:"eventId,omitempty"`
}

type Docs struct {
	MdnDocs generic.OptionallySet[[]MdnDoc] `json:"mdnDocs,omitzero"`
}

type MdnDoc struct {
	URL   generic.OptionallySet[string]  `json:"url,omitzero"`
	Title generic.OptionallySet[*string] `json:"title,omitzero"`
	Slug  generic.OptionallySet[*string] `json:"slug,omitzero"`
}
