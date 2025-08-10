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

import "github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"

// ProcessedWebFeaturesData is the top-level container for the fully parsed and
// transformed data from the web-features package. It represents a clean,
// application-ready view of the data, with features pre-sorted by kind.
type ProcessedWebFeaturesData struct {
	Snapshots map[string]web_platform_dx__web_features.SnapshotData
	Browsers  web_platform_dx__web_features.Browsers
	Groups    map[string]web_platform_dx__web_features.GroupData
	Features  *FeatureKinds
}

// FeatureKinds is a container that categorizes all parsed web features by
// their specific type. This makes it easy for application logic to consume
// a specific kind of feature without needing to perform type assertions.
type FeatureKinds struct {
	Data  map[string]web_platform_dx__web_features.FeatureValue
	Moved map[string]web_platform_dx__web_features.FeatureMovedData
	Split map[string]web_platform_dx__web_features.FeatureSplitData
}
