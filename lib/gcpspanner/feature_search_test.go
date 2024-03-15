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
	"fmt"
	"math/big"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
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

func defaultSorting() Sortable {
	return NewFeatureNameSort(true)
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
	//nolint: dupl // Okay to duplicate for tests
	expectedResults := []FeatureResult{
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusUndefined),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(33, 33),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(20, 20),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(220, 220),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(11, 11),
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
					PassRate:    big.NewRat(10, 10),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(0, 10),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(120, 120),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(12, 12),
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
					PassRate:    big.NewRat(35, 50),
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
	results, _, err := client.FeaturesSearch(ctx, nil, 100, nil, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !AreFeatureResultsSlicesEqual(expectedResults, results) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResults(expectedResults),
			PrettyPrintFeatureResults(results))
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
					PassRate:    big.NewRat(33, 33),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(20, 20),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(220, 220),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(11, 11),
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
					PassRate:    big.NewRat(10, 10),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(0, 10),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(120, 120),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(12, 12),
				},
			},
		},
	}
	results, token, err := client.FeaturesSearch(ctx, nil, 2, nil, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !AreFeatureResultsSlicesEqual(expectedResultsPageOne, results) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResults(expectedResultsPageOne),
			PrettyPrintFeatureResults(results))
	}

	expectedResultsPageTwo := []FeatureResult{
		{
			FeatureID: "feature3",
			Name:      "Feature 3",
			Status:    string(BaselineStatusUndefined),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(35, 50),
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

	results, token, err = client.FeaturesSearch(ctx, token, 2, nil, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !AreFeatureResultsSlicesEqual(expectedResultsPageTwo, results) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResults(expectedResultsPageTwo),
			PrettyPrintFeatureResults(results))
	}

	// Last page should have no results and should have no token.
	results, token, err = client.FeaturesSearch(ctx, token, 2, nil, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	if token != nil {
		t.Error("expected nil token")
	}
	var expectedResultsPageThree []FeatureResult
	if !AreFeatureResultsSlicesEqual(expectedResultsPageThree, results) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResults(expectedResultsPageThree),
			PrettyPrintFeatureResults(results))
	}

}

func testFeatureSearchFilters(ctx context.Context, t *testing.T, client *Client) {
	testFeatureAvailableSearchFilters(ctx, t, client)
	testFeatureNotAvailableSearchFilters(ctx, t, client)
	testFeatureCommonFilterCombos(ctx, t, client)
	testFeatureNameFilters(ctx, t, client)
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
					PassRate:    big.NewRat(10, 10),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(0, 10),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(120, 120),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(12, 12),
				},
			},
		},
	}
	// available on barBrowser AND not available on fooBrowser
	node := &searchtypes.SearchNode{
		Operator: searchtypes.OperatorRoot,
		Term:     nil,
		Children: []*searchtypes.SearchNode{
			{
				Operator: searchtypes.OperatorAND,
				Term:     nil,
				Children: []*searchtypes.SearchNode{
					{
						Children: nil,
						Term: &searchtypes.SearchTerm{
							Identifier: searchtypes.IdentifierAvailableOn,
							Value:      "barBrowser",
						},
						Operator: searchtypes.OperatorNone,
					},
					{
						Children: nil,
						Term: &searchtypes.SearchTerm{
							Identifier: searchtypes.IdentifierAvailableOn,
							Value:      "fooBrowser",
						},
						Operator: searchtypes.OperatorNegation,
					},
				},
			},
		},
	}

	results, _, err := client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !AreFeatureResultsSlicesEqual(expectedResults, results) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResults(expectedResults),
			PrettyPrintFeatureResults(results))
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
					PassRate:    big.NewRat(10, 10),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(0, 10),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(120, 120),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(12, 12),
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
	// not available on fooBrowser
	node := &searchtypes.SearchNode{
		Operator: searchtypes.OperatorRoot,
		Term:     nil,
		Children: []*searchtypes.SearchNode{
			{
				Children: nil,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierAvailableOn,
					Value:      "fooBrowser",
				},
				Operator: searchtypes.OperatorNegation,
			},
		},
	}
	results, _, err := client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !AreFeatureResultsSlicesEqual(expectedResults, results) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResults(expectedResults),
			PrettyPrintFeatureResults(results))
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
					PassRate:    big.NewRat(33, 33),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(20, 20),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(220, 220),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(11, 11),
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
					PassRate:    big.NewRat(10, 10),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(0, 10),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(120, 120),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(12, 12),
				},
			},
		},
	}
	// available on barBrowser
	node := &searchtypes.SearchNode{
		Operator: searchtypes.OperatorRoot,
		Term:     nil,
		Children: []*searchtypes.SearchNode{
			{
				Children: nil,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierAvailableOn,
					Value:      "barBrowser",
				},
				Operator: searchtypes.OperatorNone,
			},
		},
	}
	results, _, err := client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !AreFeatureResultsSlicesEqual(expectedResults, results) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResults(expectedResults),
			PrettyPrintFeatureResults(results))
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
					PassRate:    big.NewRat(33, 33),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(20, 20),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(220, 220),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(11, 11),
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
					PassRate:    big.NewRat(10, 10),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(0, 10),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(120, 120),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(12, 12),
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
					PassRate:    big.NewRat(35, 50),
				},
			},
			ExperimentalMetrics: nil,
		},
	}
	// available on either barBrowser OR fooBrowser
	node = &searchtypes.SearchNode{
		Operator: searchtypes.OperatorRoot,
		Term:     nil,
		Children: []*searchtypes.SearchNode{
			{
				Operator: searchtypes.OperatorOR,
				Term:     nil,
				Children: []*searchtypes.SearchNode{
					{
						Children: nil,
						Term: &searchtypes.SearchTerm{
							Identifier: searchtypes.IdentifierAvailableOn,
							Value:      "barBrowser",
						},
						Operator: searchtypes.OperatorNone,
					},
					{
						Children: nil,
						Term: &searchtypes.SearchTerm{
							Identifier: searchtypes.IdentifierAvailableOn,
							Value:      "fooBrowser",
						},
						Operator: searchtypes.OperatorNone,
					},
				},
			},
		},
	}

	results, _, err = client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !AreFeatureResultsSlicesEqual(expectedResults, results) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResults(expectedResults),
			PrettyPrintFeatureResults(results))
	}
}

func testFeatureNameFilters(ctx context.Context, t *testing.T, client *Client) {
	// All lower case with partial "feature" name. Should return all.
	//nolint: dupl // Okay to duplicate for tests
	expectedResults := []FeatureResult{
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusUndefined),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(33, 33),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(20, 20),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(220, 220),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(11, 11),
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
					PassRate:    big.NewRat(10, 10),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(0, 10),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(120, 120),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(12, 12),
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
					PassRate:    big.NewRat(35, 50),
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
	node := &searchtypes.SearchNode{
		Operator: searchtypes.OperatorRoot,
		Term:     nil,
		Children: []*searchtypes.SearchNode{
			{
				Operator: searchtypes.OperatorNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierName,
					Value:      "feature",
				},
				Children: nil,
			},
		},
	}

	results, _, err := client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !AreFeatureResultsSlicesEqual(expectedResults, results) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResults(expectedResults),
			PrettyPrintFeatureResults(results))
	}

	// All upper case with partial "FEATURE" name. Should return same results (all).
	node = &searchtypes.SearchNode{
		Operator: searchtypes.OperatorRoot,
		Term:     nil,
		Children: []*searchtypes.SearchNode{
			{
				Operator: searchtypes.OperatorNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierName,
					Value:      "FEATURE",
				},
				Children: nil,
			},
		},
	}

	results, _, err = client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !AreFeatureResultsSlicesEqual(expectedResults, results) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResults(expectedResults),
			PrettyPrintFeatureResults(results))
	}

	// Search for name with "4" Should return only feature 4.
	expectedResults = []FeatureResult{
		{
			FeatureID:           "feature4",
			Name:                "Feature 4",
			Status:              string(BaselineStatusUndefined),
			StableMetrics:       nil,
			ExperimentalMetrics: nil,
		},
	}
	node = &searchtypes.SearchNode{
		Operator: searchtypes.OperatorRoot,
		Term:     nil,
		Children: []*searchtypes.SearchNode{
			{
				Operator: searchtypes.OperatorNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierName,
					Value:      "4",
				},
				Children: nil,
			},
		},
	}

	results, _, err = client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !AreFeatureResultsSlicesEqual(expectedResults, results) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResults(expectedResults),
			PrettyPrintFeatureResults(results))
	}

}

func testFeatureSearchSort(ctx context.Context, t *testing.T, client *Client) {
	testFeatureSearchSortName(ctx, t, client)
}

func testFeatureSearchSortName(ctx context.Context, t *testing.T, client *Client) {
	// Name asc
	sortByAsc := NewFeatureNameSort(true)
	//nolint: dupl // Okay to duplicate for tests
	expectedResults := []FeatureResult{
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusUndefined),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(33, 33),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(20, 20),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(220, 220),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(11, 11),
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
					PassRate:    big.NewRat(10, 10),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(0, 10),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(120, 120),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(12, 12),
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
					PassRate:    big.NewRat(35, 50),
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
	results, _, err := client.FeaturesSearch(ctx, nil, 100, nil, sortByAsc)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !AreFeatureResultsSlicesEqual(expectedResults, results) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResults(expectedResults),
			PrettyPrintFeatureResults(results))
	}

	// Name desc
	sortByDesc := NewFeatureNameSort(false)
	//nolint: dupl // Okay to duplicate for tests
	expectedResults = []FeatureResult{
		{
			FeatureID:           "feature4",
			Name:                "Feature 4",
			Status:              string(BaselineStatusUndefined),
			StableMetrics:       nil,
			ExperimentalMetrics: nil,
		},
		{
			FeatureID: "feature3",
			Name:      "Feature 3",
			Status:    string(BaselineStatusUndefined),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(35, 50),
				},
			},
			ExperimentalMetrics: nil,
		},
		{
			FeatureID: "feature2",
			Name:      "Feature 2",
			Status:    string(BaselineStatusHigh),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(10, 10),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(0, 10),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(120, 120),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(12, 12),
				},
			},
		},
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusUndefined),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(33, 33),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(20, 20),
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(220, 220),
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(11, 11),
				},
			},
		},
	}
	// Test: Get all the results.
	results, _, err = client.FeaturesSearch(ctx, nil, 100, nil, sortByDesc)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResults(results)
	if !AreFeatureResultsSlicesEqual(expectedResults, results) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResults(expectedResults),
			PrettyPrintFeatureResults(results))
	}
}

func TestFeaturesSearch(t *testing.T) {
	client := getTestDatabase(t)
	ctx := context.Background()
	setupRequiredTablesForFeaturesSearch(ctx, client, t)

	// Try with local mode equal to false
	testFeatureSearchAll(ctx, t, client)
	testFeatureSearchPagination(ctx, t, client)
	testFeatureSearchFilters(ctx, t, client)
	testFeatureSearchSort(ctx, t, client)

	// Try with local mode equal to true
	client.SetIsLocal(true)
	testFeatureSearchAll(ctx, t, client)
	testFeatureSearchPagination(ctx, t, client)
	testFeatureSearchFilters(ctx, t, client)
	testFeatureSearchSort(ctx, t, client)
}

func AreFeatureResultsSlicesEqual(a, b []FeatureResult) bool {
	return slices.EqualFunc[[]FeatureResult](a, b, AreFeatureResultsEqual)
}

func AreFeatureResultsEqual(a, b FeatureResult) bool {
	if a.FeatureID != b.FeatureID ||
		a.Name != b.Name ||
		a.Status != b.Status ||
		!AreMetricsEqual(a.StableMetrics, b.StableMetrics) ||
		!AreMetricsEqual(a.ExperimentalMetrics, b.ExperimentalMetrics) {
		return false
	}

	return true
}

func AreMetricsEqual(a, b []*FeatureResultMetric) bool {
	return slices.EqualFunc[[]*FeatureResultMetric](a, b, func(a, b *FeatureResultMetric) bool {
		if (a.PassRate == nil && b.PassRate != nil) || (a.PassRate != nil && b.PassRate == nil) {
			return false
		}

		return a.BrowserName == b.BrowserName &&
			((a.PassRate == nil && b.PassRate == nil) || (a.PassRate.Cmp(b.PassRate) == 0))
	})
}

func PrettyPrintFeatureResult(result FeatureResult) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "FeatureID: %s\n", result.FeatureID)
	fmt.Fprintf(&builder, "Name: %s\n", result.Name)
	fmt.Fprintf(&builder, "Status: %s\n", result.Status)

	fmt.Fprintln(&builder, "Stable Metrics:")
	for _, metric := range result.StableMetrics {
		fmt.Fprint(&builder, PrettyPrintMetric(metric))
	}

	fmt.Fprintln(&builder, "Experimental Metrics:")
	for _, metric := range result.ExperimentalMetrics {
		fmt.Fprint(&builder, PrettyPrintMetric(metric))
	}
	fmt.Fprintln(&builder)

	return builder.String()
}

func PrettyPrintMetric(metric *FeatureResultMetric) string {
	var builder strings.Builder
	if metric == nil {
		return "\tNIL\n"
	}
	fmt.Fprintf(&builder, "\tBrowserName: %s\n", metric.BrowserName)
	fmt.Fprintf(&builder, "\tPassRate: %s\n", PrettyPrintPassRate(metric.PassRate))

	return builder.String()
}

func PrettyPrintPassRate(passRate *big.Rat) string {
	if passRate == nil {
		return "\tNIL\n"
	}

	return passRate.String()
}

// PrettyPrintFeatureResults returns a formatted string representation of a slice of FeatureResult structs.
func PrettyPrintFeatureResults(results []FeatureResult) string {
	var builder strings.Builder
	for _, result := range results {
		fmt.Fprint(&builder, PrettyPrintFeatureResult(result))
	}

	return builder.String()
}
