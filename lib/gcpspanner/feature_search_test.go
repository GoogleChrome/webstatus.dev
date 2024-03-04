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

package gcpspanner

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

func setupRequiredTablesForFeaturesSearch(ctx context.Context,
	client *Client, t *testing.T) {
	//nolint: dupl // Okay to duplicate for tests
	sampleFeatures := []WebFeature{
		{
			Name:      "Feature 1",
			FeatureID: "feature1",
		},
		{
			Name:      "Feature 2",
			FeatureID: "feature2",
		},
		{
			Name:      "Feature 3",
			FeatureID: "feature3",
		},
		{
			Name:      "Feature 4",
			FeatureID: "feature4",
		},
	}
	for _, feature := range sampleFeatures {
		err := client.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert of features. %s", err.Error())
		}
	}

	// nolint: dupl // Okay to duplicate for tests
	sampleReleases := []BrowserRelease{
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "0.0.0",
			ReleaseDate:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "0.0.0",
			ReleaseDate:    time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "1.0.0",
			ReleaseDate:    time.Date(2000, time.February, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "1.0.0",
			ReleaseDate:    time.Date(2000, time.February, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "2.0.0",
			ReleaseDate:    time.Date(2000, time.March, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "2.0.0",
			ReleaseDate:    time.Date(2000, time.March, 2, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, release := range sampleReleases {
		err := client.InsertBrowserRelease(ctx, release)
		if err != nil {
			t.Errorf("unexpected error during insert of releases. %s", err.Error())
		}
	}

	//nolint: dupl // Okay to duplicate for tests
	sampleBrowserAvailabilities := []BrowserFeatureAvailability{
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "0.0.0",
			FeatureID:      "feature1",
		},
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "1.0.0",
			FeatureID:      "feature1",
		},
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "2.0.0",
			FeatureID:      "feature2",
		},
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "1.0.0",
			FeatureID:      "feature3",
		},
	}
	for _, availability := range sampleBrowserAvailabilities {
		err := client.InsertBrowserFeatureAvailability(ctx, availability)
		if err != nil {
			t.Errorf("unexpected error during insert of availabilities. %s", err.Error())
		}
	}

	//nolint: dupl // Okay to duplicate for tests
	sampleBaselineStatuses := []FeatureBaselineStatus{
		{
			FeatureID: "feature1",
			Status:    BaselineStatusUndefined,
			LowDate:   nil,
			HighDate:  nil,
		},
		{
			FeatureID: "feature2",
			Status:    BaselineStatusHigh,
			LowDate:   valuePtr[time.Time](time.Date(2000, time.January, 15, 0, 0, 0, 0, time.UTC)),
			HighDate:  valuePtr[time.Time](time.Date(2000, time.January, 31, 0, 0, 0, 0, time.UTC)),
		},
		{
			FeatureID: "feature3",
			Status:    BaselineStatusUndefined,
			LowDate:   nil,
			HighDate:  nil,
		},
		// feature4 will default to undefined.
	}
	for _, status := range sampleBaselineStatuses {
		err := client.UpsertFeatureBaselineStatus(ctx, status)
		if err != nil {
			t.Errorf("unexpected error during insert of statuses. %s", err.Error())
		}
	}

	// nolint: dupl // Okay to duplicate for tests
	sampleRuns := []WPTRun{
		{
			RunID:            0,
			TimeStart:        time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:      "fooBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.StableLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            1,
			TimeStart:        time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:      "fooBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.ExperimentalLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            2,
			TimeStart:        time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:      "barBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.StableLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            3,
			TimeStart:        time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:      "barBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.ExperimentalLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            6,
			TimeStart:        time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:      "fooBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.StableLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            7,
			TimeStart:        time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:      "fooBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.ExperimentalLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            8,
			TimeStart:        time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:      "barBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.StableLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            9,
			TimeStart:        time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:      "barBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.ExperimentalLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
	}

	for _, run := range sampleRuns {
		err := client.InsertWPTRun(ctx, run)
		if err != nil {
			t.Errorf("unexpected error during insert of runs. %s", err.Error())
		}
	}

	// nolint: dupl // Okay to duplicate for tests
	sampleRunMetrics := []struct {
		ExternalRunID int64
		WPTRunFeatureMetric
	}{
		// Run 0 metrics - fooBrowser - stable
		{
			ExternalRunID: 0,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](20),
				TestPass:   valuePtr[int64](10),
			},
		},
		{
			ExternalRunID: 0,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature2",
				TotalTests: valuePtr[int64](5),
				TestPass:   valuePtr[int64](0),
			},
		},
		{
			ExternalRunID: 0,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature3",
				TotalTests: valuePtr[int64](50),
				TestPass:   valuePtr[int64](5),
			},
		},
		// Run 1 metrics - fooBrowser - experimental
		{
			ExternalRunID: 1,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](20),
				TestPass:   valuePtr[int64](20),
			},
		},
		// Run 2 metrics - barBrowser - stable
		{
			ExternalRunID: 2,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](20),
				TestPass:   valuePtr[int64](10),
			},
		},
		// Run 3 metrics - barBrowser - experimental
		{
			ExternalRunID: 3,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](20),
				TestPass:   valuePtr[int64](10),
			},
		},
		// Run 6 metrics - fooBrowser - stable
		{
			ExternalRunID: 6,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](20),
				TestPass:   valuePtr[int64](20),
			},
		},
		{
			ExternalRunID: 6,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature2",
				TotalTests: valuePtr[int64](10),
				TestPass:   valuePtr[int64](0),
			},
		},
		{
			ExternalRunID: 6,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature3",
				TotalTests: valuePtr[int64](50),
				TestPass:   valuePtr[int64](35),
			},
		},
		// Run 7 metrics - fooBrowser - experimental
		{
			ExternalRunID: 7,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](11),
				TestPass:   valuePtr[int64](11),
			},
		},
		{
			ExternalRunID: 7,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature2",
				TotalTests: valuePtr[int64](12),
				TestPass:   valuePtr[int64](12),
			},
		},
		// Run 8 metrics - barBrowser - stable
		{
			ExternalRunID: 8,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](33),
				TestPass:   valuePtr[int64](33),
			},
		},
		{
			ExternalRunID: 8,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature2",
				TotalTests: valuePtr[int64](10),
				TestPass:   valuePtr[int64](10),
			},
		},
		// Run 9 metrics - barBrowser - experimental
		{
			ExternalRunID: 9,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](220),
				TestPass:   valuePtr[int64](220),
			},
		},
		{
			ExternalRunID: 9,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature2",
				TotalTests: valuePtr[int64](120),
				TestPass:   valuePtr[int64](120),
			},
		},
	}
	for _, metric := range sampleRunMetrics {
		err := client.UpsertWPTRunFeatureMetric(ctx, metric.ExternalRunID, metric.WPTRunFeatureMetric)
		if err != nil {
			t.Errorf("unexpected error during insert of metrics. %s", err.Error())
		}
	}
}

func sortMetricsByBrowserName(metrics []*FeatureResultMetric) {
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].BrowserName < metrics[j].BrowserName
	})
}
func stabilizeFeatureResults(results []FeatureResult) {
	for _, result := range results {
		stabilizeFeatureResult(result)
	}
}

func stabilizeFeatureResult(result FeatureResult) {
	sortMetricsByBrowserName(result.StableMetrics)
	sortMetricsByBrowserName(result.ExperimentalMetrics)

}

func testFeatureSearchAll(ctx context.Context, t *testing.T, client *Client) {
	// Simple test to get all the features without filters.
	expectedResults := []FeatureResult{
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusUndefined),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](33),
					TestPass:    valuePtr[int64](33),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](20),
					TestPass:    valuePtr[int64](20),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](220),
					TestPass:    valuePtr[int64](220),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](11),
					TestPass:    valuePtr[int64](11),
				},
			},
		},
		{
			FeatureID: "feature2",
			Name:      "Feature 2",
			Status:    string(BaselineStatusHigh),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](10),
					TestPass:    valuePtr[int64](10),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](10),
					TestPass:    valuePtr[int64](0),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](120),
					TestPass:    valuePtr[int64](120),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](12),
					TestPass:    valuePtr[int64](12),
				},
			},
		},
		{
			FeatureID: "feature3",
			Name:      "Feature 3",
			Status:    string(BaselineStatusUndefined),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](50),
					TestPass:    valuePtr[int64](35),
				},
			},
			ExperimentalMetrics: nil,
		},
		{
			FeatureID:           "feature4",
			Name:                "Feature 4",
			Status:              string(BaselineStatusUndefined),
			StableMetrics:       nil,
			ExperimentalMetrics: nil,
		},
	}
	// Test: Get all the results.
	results, _, err := client.FeaturesSearch(ctx, nil, 100)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !reflect.DeepEqual(expectedResults, results) {
		t.Errorf("unequal results. expected (%+v) received (%+v) ", expectedResults, results)
	}
}

func testFeatureSearchPagination(ctx context.Context, t *testing.T, client *Client) {
	// Test: Get all the results with pagination.
	// nolint: dupl
	expectedResultsPageOne := []FeatureResult{
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusUndefined),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](33),
					TestPass:    valuePtr[int64](33),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](20),
					TestPass:    valuePtr[int64](20),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](220),
					TestPass:    valuePtr[int64](220),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](11),
					TestPass:    valuePtr[int64](11),
				},
			},
		},
		{
			FeatureID: "feature2",
			Name:      "Feature 2",
			Status:    string(BaselineStatusHigh),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](10),
					TestPass:    valuePtr[int64](10),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](10),
					TestPass:    valuePtr[int64](0),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](120),
					TestPass:    valuePtr[int64](120),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](12),
					TestPass:    valuePtr[int64](12),
				},
			},
		},
	}
	results, token, err := client.FeaturesSearch(ctx, nil, 2)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !reflect.DeepEqual(expectedResultsPageOne, results) {
		t.Errorf("unequal results. expected (%+v) received (%+v) ", expectedResultsPageOne, results)
	}

	expectedResultsPageTwo := []FeatureResult{
		{
			FeatureID: "feature3",
			Name:      "Feature 3",
			Status:    string(BaselineStatusUndefined),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](50),
					TestPass:    valuePtr[int64](35),
				},
			},
			ExperimentalMetrics: nil,
		},
		{
			FeatureID:           "feature4",
			Name:                "Feature 4",
			Status:              string(BaselineStatusUndefined),
			StableMetrics:       nil,
			ExperimentalMetrics: nil,
		},
	}

	results, token, err = client.FeaturesSearch(ctx, token, 2)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !reflect.DeepEqual(expectedResultsPageTwo, results) {
		t.Errorf("unequal results. expected (%+v) received (%+v) ", expectedResultsPageTwo, results)
	}

	// Last page should have no results and should have no token.
	results, token, err = client.FeaturesSearch(ctx, token, 2)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	if token != nil {
		t.Error("expected nil token")
	}
	var expectedResultsPageThree []FeatureResult
	if !reflect.DeepEqual(expectedResultsPageThree, results) {
		t.Errorf("unequal results. expected (%+v) received (%+v) ", expectedResultsPageThree, results)
	}

}

func testFeatureSearchFilters(ctx context.Context, t *testing.T, client *Client) {
	testFeatureAvailableSearchFilters(ctx, t, client)
	testFeatureNotAvailableSearchFilters(ctx, t, client)
	testFeatureCommonFilterCombos(ctx, t, client)
}

func testFeatureCommonFilterCombos(ctx context.Context, t *testing.T, client *Client) {
	// Available and not available filters
	expectedResults := []FeatureResult{
		{
			FeatureID: "feature2",
			Name:      "Feature 2",
			Status:    string(BaselineStatusHigh),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](10),
					TestPass:    valuePtr[int64](10),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](10),
					TestPass:    valuePtr[int64](0),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](120),
					TestPass:    valuePtr[int64](120),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](12),
					TestPass:    valuePtr[int64](12),
				},
			},
		},
	}
	availableFilter := NewAvailabileFilter([]string{"barBrowser"})
	notAvailableFilter := NewNotAvailabileFilter([]string{"fooBrowser"})
	results, _, err := client.FeaturesSearch(ctx, nil, 100, availableFilter, notAvailableFilter)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !reflect.DeepEqual(expectedResults, results) {
		t.Errorf("unequal results. expected (%+v) received (%+v) ", expectedResults, results)
	}
}

func testFeatureNotAvailableSearchFilters(ctx context.Context, t *testing.T, client *Client) {
	// Single browser
	expectedResults := []FeatureResult{
		{
			FeatureID: "feature2",
			Name:      "Feature 2",
			Status:    string(BaselineStatusHigh),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](10),
					TestPass:    valuePtr[int64](10),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](10),
					TestPass:    valuePtr[int64](0),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](120),
					TestPass:    valuePtr[int64](120),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](12),
					TestPass:    valuePtr[int64](12),
				},
			},
		},
		{
			FeatureID:           "feature4",
			Name:                "Feature 4",
			Status:              string(BaselineStatusUndefined),
			StableMetrics:       nil,
			ExperimentalMetrics: nil,
		},
	}
	notAvailableFilter := NewNotAvailabileFilter([]string{"fooBrowser"})
	results, _, err := client.FeaturesSearch(ctx, nil, 100, notAvailableFilter)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !reflect.DeepEqual(expectedResults, results) {
		t.Errorf("unequal results. expected (%+v) received (%+v) ", expectedResults, results)
	}
}
func testFeatureAvailableSearchFilters(ctx context.Context, t *testing.T, client *Client) {
	// Single browser
	// nolint: dupl
	expectedResults := []FeatureResult{
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusUndefined),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](33),
					TestPass:    valuePtr[int64](33),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](20),
					TestPass:    valuePtr[int64](20),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](220),
					TestPass:    valuePtr[int64](220),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](11),
					TestPass:    valuePtr[int64](11),
				},
			},
		},
		{
			FeatureID: "feature2",
			Name:      "Feature 2",
			Status:    string(BaselineStatusHigh),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](10),
					TestPass:    valuePtr[int64](10),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](10),
					TestPass:    valuePtr[int64](0),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](120),
					TestPass:    valuePtr[int64](120),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](12),
					TestPass:    valuePtr[int64](12),
				},
			},
		},
	}
	availableFilter := NewAvailabileFilter([]string{"barBrowser"})
	results, _, err := client.FeaturesSearch(ctx, nil, 100, availableFilter)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !reflect.DeepEqual(expectedResults, results) {
		t.Errorf("unequal results. expected (%+v) received (%+v) ", expectedResults, results)
	}

	// Multiple browsers.
	expectedResults = []FeatureResult{
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusUndefined),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](33),
					TestPass:    valuePtr[int64](33),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](20),
					TestPass:    valuePtr[int64](20),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](220),
					TestPass:    valuePtr[int64](220),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](11),
					TestPass:    valuePtr[int64](11),
				},
			},
		},
		{
			FeatureID: "feature2",
			Name:      "Feature 2",
			Status:    string(BaselineStatusHigh),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](10),
					TestPass:    valuePtr[int64](10),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](10),
					TestPass:    valuePtr[int64](0),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					TotalTests:  valuePtr[int64](120),
					TestPass:    valuePtr[int64](120),
				},
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](12),
					TestPass:    valuePtr[int64](12),
				},
			},
		},
		{
			FeatureID: "feature3",
			Name:      "Feature 3",
			Status:    string(BaselineStatusUndefined),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "fooBrowser",
					TotalTests:  valuePtr[int64](50),
					TestPass:    valuePtr[int64](35),
				},
			},
			ExperimentalMetrics: nil,
		},
	}
	availableFilter = NewAvailabileFilter([]string{"barBrowser", "fooBrowser"})
	results, _, err = client.FeaturesSearch(ctx, nil, 100, availableFilter)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !reflect.DeepEqual(expectedResults, results) {
		t.Errorf("unequal results. expected (%+v) received (%+v) ", expectedResults, results)
	}
}

func TestFeaturesSearch(t *testing.T) {
	client := getTestDatabase(t)
	ctx := context.Background()
	setupRequiredTablesForFeaturesSearch(ctx, client, t)

	testFeatureSearchAll(ctx, t, client)
	testFeatureSearchPagination(ctx, t, client)
	testFeatureSearchFilters(ctx, t, client)
}
