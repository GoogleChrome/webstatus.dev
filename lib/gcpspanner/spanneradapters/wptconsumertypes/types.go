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

import "time"

// WPTRun representation of wpt run that comes from the WPT Consumer
// This is located in the lib package so that it can be used in the adapter.
type WPTRun struct {
	ID               int64
	BrowserName      string
	BrowserVersion   string
	TimeStart        time.Time
	TimeEnd          time.Time
	Channel          string
	OSName           string
	OSVersion        string
	FullRevisionHash string
}

type WPTFeatureMetric struct {
	TotalTests *int64
	TestPass   *int64
}
