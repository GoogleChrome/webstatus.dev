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

package wptconsumertypes

import (
	"errors"
)

// WPTFeatureMetric is the representation of the metric that comes from the WPT Consumer
// This is located in the shared lib package so that it can be used in the adapter and the workflow.
type WPTFeatureMetric struct {
	TotalTests *int64
	TestPass   *int64
}

// ErrInvalidDataFromWPT indicates that the data will not be stored because it
// contains unexpected data from WPT.
var ErrInvalidDataFromWPT = errors.New("invalid data from WPT")

// ErrUnableToStoreWPTRun indicates that the storage layer was unable to save
// the wpt run data.
var ErrUnableToStoreWPTRun = errors.New("unable to store wpt run data")

// ErrUnableToStoreWPTRunFeatureMetrics indicates that the storage layer was
// unable to save the wpt run feature metrics.
var ErrUnableToStoreWPTRunFeatureMetrics = errors.New("unable to store wpt run feature metrics")
