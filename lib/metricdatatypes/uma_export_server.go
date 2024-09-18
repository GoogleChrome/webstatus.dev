// Copyright 2024 Google LLC
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

package metricdatatypes

/*
Data from UMA Export Server
*/

// UMAExportQuery represents the enumeration of queries that can be sent to the UMA Export Server.
type UMAExportQuery string

const (
	WebDXFeaturesQuery UMAExportQuery = "usecounter.webdxfeatures"
)

// BucketDataMetric contains metric details for a specific bucket
// in the UMA Export Server response.  These fields correspond to
// those used in ChromeStatus:
// https://github.com/GoogleChrome/chromium-dashboard/blob/35ccb75ce0ba9a599f9adb71ed93224621a4177a/internals/fetchmetrics.py#L148-L150
// You can also refer to the google3 code for the structure of the body too.
//
//nolint:lll // WONTFIX - url is long.
type BucketDataMetric struct {
	Rate      float64 `json:"rate,omitempty"`
	Milestone string  `json:"milestone,omitempty"`
	LowVolume bool    `json:"low_volume,omitempty"`
}

// BucketDataMetrics maps the bucket IDs (from histograms in https://chromium.googlesource.com/chromium/src/+/main/tools/metrics/histograms/enums.xml)
// to their corresponding metrics. This map is returned as a field in the UMA Export Server response.
//
// Keys: Integer values representing bucket IDs within a histogram. This integer is the "value" for each enum value in enums.xml.
// Values: Metric details (see BucketDataMetric).
//
// Refer to the corresponding mojom file for a specific histogram to see the definitions of these integer values.
// Below is a mapping between histogram and mojom file:
// - WebDXFeatureObserver: https://chromium.googlesource.com/chromium/chromium/src/+/main:third_party/blink/public/mojom/use_counter/metrics/webdx_feature.mojom
//
//nolint:lll // WONTFIX - url is long.
type BucketDataMetrics map[int64]BucketDataMetric
