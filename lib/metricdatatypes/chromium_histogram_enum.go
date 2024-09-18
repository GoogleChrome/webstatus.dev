// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package metricdatatypes

/*
Data from https://chromium.googlesource.com/chromium/src/+/main/tools/metrics/histograms/enums.xml
*/
type HistogramName string

// Names come from the enums file above.
const (
	// Generated from third_party/blink/public/mojom/use_counter/metrics/webdx_feature.mojom.
	WebDXFeatureEnum HistogramName = "WebDXFeatureObserver"
)

// Each histogram in the enums file contains a list of enum values.
type HistogramMapping map[HistogramName][]HistogramEnumValue

// HistogramEnumValue contains the information for a single enumeration inside a given histogram.
type HistogramEnumValue struct {
	Value int64
	Label string
}
