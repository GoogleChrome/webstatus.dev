package webdxfeaturetypes

import (
	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
)

// ProcessedWebFeaturesData is the top-level container for the fully parsed and
// transformed data from the web-features package. It represents a clean,
// application-ready view of the data, with features pre-sorted by kind.
type ProcessedWebFeaturesData struct {
	Snapshots map[string]web_platform_dx__web_features.SnapshotData
	Browsers  map[string]web_platform_dx__web_features.BrowserData
	Groups    map[string]web_platform_dx__web_features.GroupData
	Features  FeatureKinds
}

// FeatureKinds is a container that categorizes all parsed web features by
// their specific type. This makes it easy for application logic to consume
// a specific kind of feature without needing to perform type assertions.
type FeatureKinds struct {
	Data  map[string]web_platform_dx__web_features.FeatureData
	Moved map[string]web_platform_dx__web_features.FeatureMovedData
	Split map[string]web_platform_dx__web_features.FeatureSplitData
}
