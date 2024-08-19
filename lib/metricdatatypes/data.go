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

// BucketDataMetric contains the metric details for a particular bucket.
type BucketDataMetric struct {
	Rate      float64 `json:"rate,omitempty"`
	Milestone string  `json:"milestone,omitempty"`
	LowVolume bool    `json:"low_volume,omitempty"`
}

// BucketDataMetrics is a map between the bucket ID and the metric.
// For WebDX Feature metrics, these identifiers can be found at:
// - https://source.chromium.org/chromium/chromium/src/+/main:third_party/blink/public/mojom/use_counter/metrics/webdx_feature.mojom
//
//nolint:lll // WONTFIX - url is long.
type BucketDataMetrics map[int64]BucketDataMetric
