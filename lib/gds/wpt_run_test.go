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

// nolint: gochecknoglobals
var sampleWPTRuns = []WPTRun{
	{
		WPTRunMetadata: WPTRunMetadata{
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
		WPTRunMetadata: WPTRunMetadata{
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
		WPTRunMetadata: WPTRunMetadata{
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
		WPTRunMetadata: WPTRunMetadata{
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
		WPTRunMetadata: WPTRunMetadata{
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
		WPTRunMetadata: WPTRunMetadata{
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

func testWPTMetricsByBrowser(ctx context.Context, client *Client, t *testing.T) {
	// Get the foo browser
	// Step 1. Pick a range that gets both entries of run wide metrics.
	metrics, _, err := client.ListWPTMetricsByBrowser(
		ctx,
		"fooBrowser",
		shared.StableLabel,
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Errorf("unable to get metrics for browser. %s", err.Error())
	}
	expectedPageBoth := []WPTRunToMetrics{
		{
			WPTRunMetadata: WPTRunMetadata{
				RunID:          6,
				TimeStart:      time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
				TimeEnd:        time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
				BrowserName:    "fooBrowser",
				BrowserVersion: "0.0.0",
				Channel:        shared.StableLabel,
				OSName:         "os",
				OSVersion:      "0.0.0",
			},
			WPTRunMetric: &WPTRunMetric{
				TotalTests: intPtr(2),
				TestPass:   intPtr(2),
			},
		},
		{
			WPTRunMetadata: WPTRunMetadata{
				RunID:          0,
				TimeStart:      time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				TimeEnd:        time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
				BrowserName:    "fooBrowser",
				BrowserVersion: "0.0.0",
				Channel:        shared.StableLabel,
				OSName:         "os",
				OSVersion:      "0.0.0",
			},
			WPTRunMetric: &WPTRunMetric{
				TotalTests: intPtr(2),
				TestPass:   intPtr(2),
			},
		},
	}
	if !reflect.DeepEqual(expectedPageBoth, metrics) {
		t.Error("unequal slices")
	}

	// Step 2. Pick a range that only gets run-wide metric.
	metrics, _, err = client.ListWPTMetricsByBrowser(
		ctx,
		"fooBrowser",
		shared.StableLabel,
		time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Errorf("unable to get metrics for browser. %s", err.Error())
	}
	expectedPageLast := []WPTRunToMetrics{
		{
			WPTRunMetadata: WPTRunMetadata{
				RunID:          6,
				TimeStart:      time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
				TimeEnd:        time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
				BrowserName:    "fooBrowser",
				BrowserVersion: "0.0.0",
				Channel:        shared.StableLabel,
				OSName:         "os",
				OSVersion:      "0.0.0",
			},
			WPTRunMetric: &WPTRunMetric{
				TotalTests: intPtr(2),
				TestPass:   intPtr(2),
			},
		},
	}
	if !reflect.DeepEqual(expectedPageLast, metrics) {
		t.Error("unequal slices")
	}
}

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

func TestWPTRunMetricsOperations(t *testing.T) {
	ctx := context.Background()

	// Getting the test database is expensive that is why the methods below aren't their own tests.
	// For now, get it once and use it in the sub tests below.
	client, cleanup := getTestDatabase(ctx, t)
	defer cleanup()
	setupEntities(ctx, t, client)

	testWPTMetricsByBrowser(ctx, client, t)
	testWPTMetricsByBrowserByFeature(ctx, client, t)
}
