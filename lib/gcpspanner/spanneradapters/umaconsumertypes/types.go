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

package umaconsumertypes

import "errors"

// ErrCapstoneLookupFailed indicates an internal error trying to find the capstone.
var ErrCapstoneLookupFailed = errors.New("failed to look up capstone")

// ErrCapstoneSaveFailed indicates an internal error trying to save the capstone.
var ErrCapstoneSaveFailed = errors.New("failed to save capstone")

// ErrMetricsSaveFailed indicates an internal error trying to save the metrics.
var ErrMetricsSaveFailed = errors.New("failed to save metrics")

// ErrInvalidRate indicates an internal error when parsing the rate.
var ErrInvalidRate = errors.New("invalid rate")
