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

package webdxfeaturetypes

// ProcessedWebFeaturesData is the top-level container for the fully parsed and
// transformed data from the web-features package. It represents a clean,
// application-ready view of the data, with features pre-sorted by kind.
type ProcessedWebFeaturesData struct {
	Snapshots map[string]SnapshotData
	Browsers  Browsers
	Groups    map[string]GroupData
	Features  *FeatureKinds
}

// FeatureKinds is a container that categorizes all parsed web features by
// their specific type. This makes it easy for application logic to consume
// a specific kind of feature without needing to perform type assertions.
type FeatureKinds struct {
	Data  map[string]FeatureValue
	Moved map[string]FeatureMovedData
	Split map[string]FeatureSplitData
}

type FeatureData struct {
	Browsers  Browsers                `json:"browsers"`
	Features  map[string]FeatureValue `json:"features"`
	Groups    map[string]GroupData    `json:"groups"`
	Snapshots map[string]SnapshotData `json:"snapshots"`
}

// Browsers and browser release data.
type Browsers struct {
	Chrome         BrowserData `json:"chrome"`
	ChromeAndroid  BrowserData `json:"chrome_android"`
	Edge           BrowserData `json:"edge"`
	Firefox        BrowserData `json:"firefox"`
	FirefoxAndroid BrowserData `json:"firefox_android"`
	Safari         BrowserData `json:"safari"`
	SafariIos      BrowserData `json:"safari_ios"`
}

// Browser information.
type BrowserData struct {
	// The name of the browser, as in "Edge" or "Safari on iOS"
	Name string `json:"name"`
	// The browser's releases
	Releases []Release `json:"releases"`
}

// Browser release information.
type Release struct {
	// The release date, as in "2023-12-11"
	Date string `json:"date"`
	// The version string, as in "10" or "17.1"
	Version string `json:"version"`
}

type FeatureValue struct {
	// caniuse.com identifier(s)
	Caniuse []string `json:"caniuse"`
	// Sources of support data for this feature
	CompatFeatures []string `json:"compat_features,omitempty"`
	// Short description of the feature, as a plain text string
	Description string `json:"description"`
	// Short description of the feature, as an HTML string
	DescriptionHTML string `json:"description_html"`
	// Whether developers are formally discouraged from using this feature
	Discouraged *Discouraged `json:"discouraged,omitempty"`
	// Group identifier(s)
	Group []string `json:"group"`
	// Short name
	Name string `json:"name"`
	// Snapshot identifier(s)
	Snapshot []string `json:"snapshot"`
	// Specification URL(s)
	Spec []string `json:"spec"`
	// Whether a feature is considered a "baseline" web platform feature and when it achieved
	// that status
	Status Status `json:"status"`
}

// Whether developers are formally discouraged from using this feature.
type Discouraged struct {
	// Links to a formal discouragement notice, such as specification text, intent-to-unship,
	// etc.
	AccordingTo []string `json:"according_to"`
	// IDs for features that substitute some or all of this feature's utility
	Alternatives []string `json:"alternatives,omitempty"`
}

// Whether a feature is considered a "baseline" web platform feature and when it achieved
// that status.
type Status struct {
	// Whether the feature is Baseline (low substatus), Baseline (high substatus), or not (false)
	Baseline *BaselineUnion `json:"baseline"`
	// Date the feature achieved Baseline high status
	BaselineHighDate *string `json:"baseline_high_date,omitempty"`
	// Date the feature achieved Baseline low status
	BaselineLowDate *string `json:"baseline_low_date,omitempty"`
	// Statuses for each key in the feature's compat_features list, if applicable. Not available
	// to the npm release of web-features.
	ByCompatKey map[string]ByCompatKey `json:"by_compat_key,omitempty"`
	// Browser versions that most-recently introduced the feature
	Support StatusSupport `json:"support"`
}

type ByCompatKey struct {
	// Whether the feature is Baseline (low substatus), Baseline (high substatus), or not (false)
	Baseline *BaselineUnion `json:"baseline"`
	// Date the feature achieved Baseline high status
	BaselineHighDate *string `json:"baseline_high_date,omitempty"`
	// Date the feature achieved Baseline low status
	BaselineLowDate *string `json:"baseline_low_date,omitempty"`
	// Browser versions that most-recently introduced the feature
	Support ByCompatKeySupport `json:"support"`
}

// Browser versions that most-recently introduced the feature.
type ByCompatKeySupport struct {
	Chrome         *string `json:"chrome,omitempty"`
	ChromeAndroid  *string `json:"chrome_android,omitempty"`
	Edge           *string `json:"edge,omitempty"`
	Firefox        *string `json:"firefox,omitempty"`
	FirefoxAndroid *string `json:"firefox_android,omitempty"`
	Safari         *string `json:"safari,omitempty"`
	SafariIos      *string `json:"safari_ios,omitempty"`
}

// Browser versions that most-recently introduced the feature.
type StatusSupport struct {
	Chrome         *string `json:"chrome,omitempty"`
	ChromeAndroid  *string `json:"chrome_android,omitempty"`
	Edge           *string `json:"edge,omitempty"`
	Firefox        *string `json:"firefox,omitempty"`
	FirefoxAndroid *string `json:"firefox_android,omitempty"`
	Safari         *string `json:"safari,omitempty"`
	SafariIos      *string `json:"safari_ios,omitempty"`
}

type GroupData struct {
	// Short name
	Name string `json:"name"`
	// Identifier of parent group
	Parent *string `json:"parent,omitempty"`
}

type SnapshotData struct {
	// Short name
	Name string `json:"name"`
	// Specification
	Spec string `json:"spec"`
}

type BaselineEnum string

const (
	High BaselineEnum = "high"
	Low  BaselineEnum = "low"
)

type BaselineUnion struct {
	Bool *bool
	Enum *BaselineEnum
}

type FeatureDataKind string

const (
	Feature FeatureDataKind = "feature"
)

type FeatureMovedDataKind string

const (
	Moved FeatureMovedDataKind = "moved"
)

type FeatureSplitDataKind string

const (
	Split FeatureSplitDataKind = "split"
)

// A feature has permanently moved to exactly one other ID.
type FeatureMovedData struct {
	Kind FeatureMovedDataKind `json:"kind"`
	// The new ID for this feature
	RedirectTarget string `json:"redirect_target"`
}

// A feature has split into two or more other features.
type FeatureSplitData struct {
	Kind FeatureSplitDataKind `json:"kind"`
	// The new IDs for this feature
	RedirectTargets []string `json:"redirect_targets"`
}
