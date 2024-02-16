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

package gds

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

func testWPTMetricsByBrowserByFeature(ctx context.Context, client *Client, t *testing.T) {
	// Step 1b. Pick a range that gets both entries of feature specific metrics.
	featureMetrics, _, err := client.ListWPTMetricsByBrowserByFeature(
		ctx,
		"fooBrowser",
		shared.ExperimentalLabel,
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		"fooFeature",
		nil,
	)
	if err != nil {
		t.Errorf("unable to get metrics for browser by feature. %s", err.Error())
	}
	expectedPageFeatureMetrics := []*WPTRunToMetricsByFeature{
		{
			WPTRunMetadata: WPTRunMetadata{
				RunID:          7,
				TimeStart:      time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
				TimeEnd:        time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
				BrowserName:    "fooBrowser",
				BrowserVersion: "0.0.0",
				Channel:        shared.ExperimentalLabel,
				OSName:         "os",
				OSVersion:      "0.0.0",
			},
			WPTRunMetric: &WPTRunMetric{
				TotalTests: intPtr(2),
				TestPass:   intPtr(2),
			},
			FeatureID: "fooFeature",
		},
		{
			WPTRunMetadata: WPTRunMetadata{
				RunID:          1,
				TimeStart:      time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				TimeEnd:        time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
				BrowserName:    "fooBrowser",
				BrowserVersion: "0.0.0",
				Channel:        shared.ExperimentalLabel,
				OSName:         "os",
				OSVersion:      "0.0.0",
			},
			WPTRunMetric: &WPTRunMetric{
				TotalTests: intPtr(1),
				TestPass:   intPtr(1),
			},
			FeatureID: "fooFeature",
		},
	}
	if !reflect.DeepEqual(expectedPageFeatureMetrics, featureMetrics) {
		t.Errorf("unequal slices")
	}
}
