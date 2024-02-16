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
	"testing"
	"time"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

// nolint: gochecknoglobals
// Set of initial runs that each test suite starts off with.
var sampleWPTRuns = []WPTRun{
	{
		WPTRunMetadata: &WPTRunMetadata{
			RunID:          0,
			TimeStart:      time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:        time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:    "fooBrowser",
			BrowserVersion: "0.0.0",
			Channel:        shared.StableLabel,
			OSName:         "os",
			OSVersion:      "0.0.0",
		},
		TestMetric: &WPTRunMetric{
			TotalTests: intPtr(2),
			TestPass:   intPtr(2),
		},
		FeatureTestMetrics: []WPTRunMetricsGroupByFeature{
			{
				FeatureID: "fooFeature",
				WPTRunMetric: WPTRunMetric{
					TotalTests: intPtr(1),
					TestPass:   intPtr(0),
				},
			},
		},
	},
	{
		WPTRunMetadata: &WPTRunMetadata{
			RunID:          1,
			TimeStart:      time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:        time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:    "fooBrowser",
			BrowserVersion: "0.0.0",
			Channel:        shared.ExperimentalLabel,
			OSName:         "os",
			OSVersion:      "0.0.0",
		},
		TestMetric: &WPTRunMetric{
			TotalTests: intPtr(3),
			TestPass:   intPtr(3),
		},
		FeatureTestMetrics: []WPTRunMetricsGroupByFeature{
			{
				FeatureID: "fooFeature",
				WPTRunMetric: WPTRunMetric{
					TotalTests: intPtr(1),
					TestPass:   intPtr(1),
				},
			},
		},
	},
	{
		WPTRunMetadata: &WPTRunMetadata{
			RunID:          2,
			TimeStart:      time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:        time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:    "barBrowser",
			BrowserVersion: "0.0.0",
			Channel:        shared.StableLabel,
			OSName:         "os",
			OSVersion:      "0.0.0",
		},
		TestMetric: &WPTRunMetric{
			TotalTests: intPtr(2),
			TestPass:   intPtr(2),
		},
		FeatureTestMetrics: []WPTRunMetricsGroupByFeature{
			{
				FeatureID: "fooFeature",
				WPTRunMetric: WPTRunMetric{
					TotalTests: intPtr(1),
					TestPass:   intPtr(1),
				},
			},
		},
	},
	{
		WPTRunMetadata: &WPTRunMetadata{
			RunID:          3,
			TimeStart:      time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:        time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:    "barBrowser",
			BrowserVersion: "0.0.0",
			Channel:        shared.ExperimentalLabel,
			OSName:         "os",
			OSVersion:      "0.0.0",
		},
		TestMetric: &WPTRunMetric{
			TotalTests: intPtr(3),
			TestPass:   intPtr(3),
		},
		FeatureTestMetrics: []WPTRunMetricsGroupByFeature{
			{
				FeatureID: "fooFeature",
				WPTRunMetric: WPTRunMetric{
					TotalTests: intPtr(1),
					TestPass:   intPtr(1),
				},
			},
		},
	},
	{
		WPTRunMetadata: &WPTRunMetadata{
			RunID:          6,
			TimeStart:      time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:        time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:    "fooBrowser",
			BrowserVersion: "0.0.0",
			Channel:        shared.StableLabel,
			OSName:         "os",
			OSVersion:      "0.0.0",
		},
		TestMetric: &WPTRunMetric{
			TotalTests: intPtr(2),
			TestPass:   intPtr(2),
		},
		FeatureTestMetrics: []WPTRunMetricsGroupByFeature{
			{
				FeatureID: "fooFeature",
				WPTRunMetric: WPTRunMetric{
					TotalTests: intPtr(1),
					TestPass:   intPtr(1),
				},
			},
			{
				FeatureID: "barFeature",
				WPTRunMetric: WPTRunMetric{
					TotalTests: intPtr(1),
					TestPass:   intPtr(1),
				},
			},
		},
	},
	{
		WPTRunMetadata: &WPTRunMetadata{
			RunID:          7,
			TimeStart:      time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:        time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:    "fooBrowser",
			BrowserVersion: "0.0.0",
			Channel:        shared.ExperimentalLabel,
			OSName:         "os",
			OSVersion:      "0.0.0",
		},
		TestMetric: &WPTRunMetric{
			TotalTests: intPtr(3),
			TestPass:   intPtr(3),
		},
		FeatureTestMetrics: []WPTRunMetricsGroupByFeature{
			{
				FeatureID: "fooFeature",
				WPTRunMetric: WPTRunMetric{
					TotalTests: intPtr(2),
					TestPass:   intPtr(2),
				},
			},
			{
				FeatureID: "barFeature",
				WPTRunMetric: WPTRunMetric{
					TotalTests: intPtr(1),
					TestPass:   intPtr(1),
				},
			},
		},
	},
	{
		WPTRunMetadata: &WPTRunMetadata{
			RunID:          8,
			TimeStart:      time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:        time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:    "barBrowser",
			BrowserVersion: "0.0.0",
			Channel:        shared.StableLabel,
			OSName:         "os",
			OSVersion:      "0.0.0",
		},
		TestMetric: &WPTRunMetric{
			TotalTests: intPtr(2),
			TestPass:   intPtr(2),
		},
		FeatureTestMetrics: []WPTRunMetricsGroupByFeature{
			{
				FeatureID: "barFeature",
				WPTRunMetric: WPTRunMetric{
					TotalTests: intPtr(1),
					TestPass:   intPtr(1),
				},
			},
			{
				FeatureID: "fooFeature",
				WPTRunMetric: WPTRunMetric{
					TotalTests: intPtr(1),
					TestPass:   intPtr(1),
				},
			},
		},
	},
	{
		WPTRunMetadata: &WPTRunMetadata{
			RunID:          9,
			TimeStart:      time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:        time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:    "barBrowser",
			BrowserVersion: "0.0.0",
			Channel:        shared.ExperimentalLabel,
			OSName:         "os",
			OSVersion:      "0.0.0",
		},
		TestMetric: &WPTRunMetric{
			TotalTests: intPtr(3),
			TestPass:   intPtr(3),
		},
		FeatureTestMetrics: []WPTRunMetricsGroupByFeature{
			{
				FeatureID: "fooFeature",
				WPTRunMetric: WPTRunMetric{
					TotalTests: intPtr(1),
					TestPass:   intPtr(1),
				},
			},
			{
				FeatureID: "barFeature",
				WPTRunMetric: WPTRunMetric{
					TotalTests: intPtr(2),
					TestPass:   intPtr(2),
				},
			},
		},
	},
}

func setupEntities(ctx context.Context, t *testing.T, client *Client) {
	for _, run := range sampleWPTRuns {
		err := client.StoreWPTRunMetadata(ctx, run.WPTRunMetadata)
		if err != nil {
			t.Errorf("unable to store wpt run %s", err.Error())
		}
		err = client.StoreWPTRunMetrics(ctx, run.RunID,
			run.TestMetric)
		if err != nil {
			t.Errorf("unable to store wpt run metric %s", err.Error())
		}
		featureMap := make(map[string]WPTRunMetric)
		for _, featureMetric := range run.FeatureTestMetrics {
			featureMap[featureMetric.FeatureID] = featureMetric.WPTRunMetric
		}
		err = client.StoreWPTRunMetricsForFeatures(ctx, run.RunID, featureMap)
		if err != nil {
			t.Errorf("unable to store wpt run metrics per feature %s", err.Error())
		}
	}
}
