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
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

func TestWPTRunOperations(t *testing.T) {
	ctx := context.Background()

	// Getting the test database is expensive that is why the methods below aren't their own tests.
	// For now, get it once and use it in the sub tests below.
	client, resetDB, cleanup := getTestDatabase(ctx, t)
	defer cleanup()

	// Test methods at the WPT Runs and WPT Metadata level
	setupEntities(ctx, t, client)
	testWPTRuns(ctx, client, t)
	resetDB()

	// Test methods at the WPT Metrics level
	setupEntities(ctx, t, client)
	testWPTMetricsByBrowser(ctx, client, t)
	resetDB()

	// Test methods at the Feature WPT metrics level
	setupEntities(ctx, t, client)
	testWPTMetricsByBrowserByFeature(ctx, client, t)
	resetDB()
}

func testWPTRuns(ctx context.Context, client *Client, t *testing.T) {
	// Try to get a non existant run.
	_, err := client.GetWPTRun(ctx, 1000)
	if !errors.Is(err, ErrEntityNotFound) {
		t.Error("expected ErrEntityNotFound")
	}
	// Get a known run
	run, err := client.GetWPTRun(ctx, 8)
	if !errors.Is(err, nil) {
		t.Error("expected no error")
	}
	expectedRunID8 := &WPTRun{
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
	}
	if !reflect.DeepEqual(expectedRunID8, run) {
		t.Errorf("expected run id 8 does not equal actual run. expected (%+v) actual (%+v)", expectedRunID8, run)
	}
}
