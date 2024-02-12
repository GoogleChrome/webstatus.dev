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
var sampleWPTRunMetrics = []WPTRunMetrics{
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
		WPTRunMetric: WPTRunMetric{
			TotalTests: intPtr(2),
			TestPass:   intPtr(2),
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
		WPTRunMetric: WPTRunMetric{
			TotalTests: intPtr(3),
			TestPass:   intPtr(3),
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
		WPTRunMetric: WPTRunMetric{
			TotalTests: intPtr(2),
			TestPass:   intPtr(2),
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
		WPTRunMetric: WPTRunMetric{
			TotalTests: intPtr(3),
			TestPass:   intPtr(3),
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
		WPTRunMetric: WPTRunMetric{
			TotalTests: intPtr(2),
			TestPass:   intPtr(2),
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
		WPTRunMetric: WPTRunMetric{
			TotalTests: intPtr(3),
			TestPass:   intPtr(3),
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
		WPTRunMetric: WPTRunMetric{
			TotalTests: intPtr(2),
			TestPass:   intPtr(2),
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
		WPTRunMetric: WPTRunMetric{
			TotalTests: intPtr(3),
			TestPass:   intPtr(3),
		},
	},
}

func TestWPTRunMetricsOperations(t *testing.T) {
	ctx := context.Background()
	client, cleanup := getTestDatabase(ctx, t)
	defer cleanup()
	for _, metric := range sampleWPTRunMetrics {
		err := client.StoreWPTRunMetrics(ctx, metric.WPTRunMetadata, metric.WPTRunMetric)
		if err != nil {
			t.Errorf("unable to store wpt run metric %s", err.Error())
		}
	}
	// Get the foo browser
	// Step 1. Pick a range that gets both entries.
	metrics, _, err := client.GetWPTMetricsByBrowser(
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
	expectedPageBoth := []*WPTRunMetrics{
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
			WPTRunMetric: WPTRunMetric{
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
			WPTRunMetric: WPTRunMetric{
				TotalTests: intPtr(2),
				TestPass:   intPtr(2),
			},
		},
	}
	if !reflect.DeepEqual(expectedPageBoth, metrics) {
		t.Error("unequal slices")
	}
	// Step 2. Pick a range that only gets one.
	metrics, _, err = client.GetWPTMetricsByBrowser(
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
	expectedPageLast := []*WPTRunMetrics{
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
			WPTRunMetric: WPTRunMetric{
				TotalTests: intPtr(2),
				TestPass:   intPtr(2),
			},
		},
	}
	if !reflect.DeepEqual(expectedPageLast, metrics) {
		t.Error("unequal slices")
	}
}
