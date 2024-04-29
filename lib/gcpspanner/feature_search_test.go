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
	"reflect"
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
	sampleBrowserAvailabilities := []struct {
		BrowserFeatureAvailability
		FeatureID string
	}{
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{
				BrowserName:    "fooBrowser",
				BrowserVersion: "0.0.0",
			},
			FeatureID: "feature1",
		},
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{
				BrowserName:    "barBrowser",
				BrowserVersion: "1.0.0",
			},
			FeatureID: "feature1",
		},
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{
				BrowserName:    "barBrowser",
				BrowserVersion: "2.0.0",
			},
			FeatureID: "feature2",
		},
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{
				BrowserName:    "fooBrowser",
				BrowserVersion: "1.0.0",
			},
			FeatureID: "feature3",
		},
	}
	for _, availability := range sampleBrowserAvailabilities {
		err := client.InsertBrowserFeatureAvailability(ctx, availability.FeatureID, availability.BrowserFeatureAvailability)
		if err != nil {
			t.Errorf("unexpected error during insert of availabilities. %s", err.Error())
		}
	}

	//nolint: dupl // Okay to duplicate for tests
	sampleBaselineStatuses := []struct {
		featureID string
		status    FeatureBaselineStatus
	}{
		{
			featureID: "feature1",
			status: FeatureBaselineStatus{
				Status:   valuePtr(BaselineStatusLow),
				LowDate:  valuePtr[time.Time](time.Date(2000, time.January, 5, 0, 0, 0, 0, time.UTC)),
				HighDate: nil,
			},
		},
		{
			featureID: "feature2",
			status: FeatureBaselineStatus{
				Status:   valuePtr(BaselineStatusHigh),
				LowDate:  valuePtr[time.Time](time.Date(2000, time.January, 4, 0, 0, 0, 0, time.UTC)),
				HighDate: valuePtr[time.Time](time.Date(2000, time.January, 31, 0, 0, 0, 0, time.UTC)),
			},
		},
		{
			featureID: "feature3",
			status: FeatureBaselineStatus{
				Status:   valuePtr(BaselineStatusNone),
				LowDate:  nil,
				HighDate: nil,
			},
		},
		// feature4 will default to nil.
	}
	for _, status := range sampleBaselineStatuses {
		err := client.UpsertFeatureBaselineStatus(ctx, status.featureID, status.status)
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
		Metrics       map[string]WPTRunFeatureMetric
	}{
		// Run 0 metrics - fooBrowser - stable
		{
			ExternalRunID: 0,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:    valuePtr[int64](20),
					TestPass:      valuePtr[int64](10),
					TotalSubtests: valuePtr[int64](220),
					SubtestPass:   valuePtr[int64](110),
				},
				"feature2": {
					TotalTests:    valuePtr[int64](5),
					TestPass:      valuePtr[int64](0),
					TotalSubtests: valuePtr[int64](55),
					SubtestPass:   valuePtr[int64](11),
				},
				"feature3": {
					TotalTests:    valuePtr[int64](50),
					TestPass:      valuePtr[int64](5),
					TotalSubtests: valuePtr[int64](5000),
					SubtestPass:   valuePtr[int64](150),
				},
			},
		},
		// Run 1 metrics - fooBrowser - experimental
		{
			ExternalRunID: 1,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:    valuePtr[int64](20),
					TestPass:      valuePtr[int64](20),
					TotalSubtests: valuePtr[int64](200),
					SubtestPass:   valuePtr[int64](200),
				},
			},
		},
		// Run 2 metrics - barBrowser - stable
		{
			ExternalRunID: 2,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:    valuePtr[int64](20),
					TestPass:      valuePtr[int64](10),
					TotalSubtests: valuePtr[int64](200),
					SubtestPass:   valuePtr[int64](15),
				},
			},
		},
		// Run 3 metrics - barBrowser - experimental
		{
			ExternalRunID: 3,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:    valuePtr[int64](20),
					TestPass:      valuePtr[int64](10),
					TotalSubtests: valuePtr[int64](700),
					SubtestPass:   valuePtr[int64](250),
				},
			},
		},
		// Run 6 metrics - fooBrowser - stable
		{
			ExternalRunID: 6,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:    valuePtr[int64](20),
					TestPass:      valuePtr[int64](20),
					TotalSubtests: valuePtr[int64](1000),
					SubtestPass:   valuePtr[int64](1000),
				},
				"feature2": {
					TotalTests:    valuePtr[int64](10),
					TestPass:      valuePtr[int64](0),
					TotalSubtests: valuePtr[int64](100),
					SubtestPass:   valuePtr[int64](15),
				},
				"feature3": {
					TotalTests:    valuePtr[int64](50),
					TestPass:      valuePtr[int64](35),
					TotalSubtests: valuePtr[int64](9000),
					SubtestPass:   valuePtr[int64](4000),
				},
			},
		},
		// Run 7 metrics - fooBrowser - experimental
		{
			ExternalRunID: 7,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:    valuePtr[int64](11),
					TestPass:      valuePtr[int64](11),
					TotalSubtests: valuePtr[int64](11),
					SubtestPass:   valuePtr[int64](11),
				},
				"feature2": {
					TotalTests:    valuePtr[int64](12),
					TestPass:      valuePtr[int64](12),
					TotalSubtests: valuePtr[int64](12),
					SubtestPass:   valuePtr[int64](12),
				},
			},
		},
		// Run 8 metrics - barBrowser - stable
		{
			ExternalRunID: 8,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:    valuePtr[int64](33),
					TestPass:      valuePtr[int64](33),
					TotalSubtests: valuePtr[int64](333),
					SubtestPass:   valuePtr[int64](333),
				},
				"feature2": {
					TotalTests:    valuePtr[int64](10),
					TestPass:      valuePtr[int64](10),
					TotalSubtests: valuePtr[int64](100),
					SubtestPass:   valuePtr[int64](100),
				},
			},
		},
		// Run 9 metrics - barBrowser - experimental
		{
			ExternalRunID: 9,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:    valuePtr[int64](220),
					TestPass:      valuePtr[int64](220),
					TotalSubtests: valuePtr[int64](2220),
					SubtestPass:   valuePtr[int64](2220),
				},
				"feature2": {
					TotalTests:    valuePtr[int64](120),
					TestPass:      valuePtr[int64](120),
					TotalSubtests: valuePtr[int64](1220),
					SubtestPass:   valuePtr[int64](1220),
				},
			},
		},
	}
	for _, metric := range sampleRunMetrics {
		err := client.UpsertWPTRunFeatureMetrics(
			ctx, metric.ExternalRunID, metric.Metrics)
		if err != nil {
			t.Errorf("unexpected error during insert of metrics. %s", err.Error())
		}
	}
}

func defaultSorting() Sortable {
	return NewFeatureNameSort(true)
}

func defaultWPTMetricView() WPTMetricView {
	// TODO. For now, default to the view mode. Switch to the subtest later.
	return WPTTestView
}

func sortImplementationStatusesByBrowserName(statuses []*ImplementationStatus) {
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].BrowserName < statuses[j].BrowserName
	})
}

func sortMetricsByBrowserName(metrics []*FeatureResultMetric) {
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].BrowserName < metrics[j].BrowserName
	})
}

func stabilizeFeatureResultPage(page *FeatureResultPage) {
	stabilizeFeatureResults(page.Features)
}

func stabilizeFeatureResults(results []FeatureResult) {
	for _, result := range results {
		stabilizeFeatureResult(result)
	}
}

func stabilizeFeatureResult(result FeatureResult) {
	sortMetricsByBrowserName(result.StableMetrics)
	sortMetricsByBrowserName(result.ExperimentalMetrics)
	sortImplementationStatusesByBrowserName(result.ImplementationStatuses)

}

// FeatureSearchTestFeatureID represents a unique identifier for a feature
// within the following files:
//   - lib/gcpspanner/feature_search_test.go
//   - lib/gcpspanner/get_feature_test.go
type FeatureSearchTestFeatureID int

const (
	FeatureSearchTestFId1 FeatureSearchTestFeatureID = 1
	FeatureSearchTestFId2 FeatureSearchTestFeatureID = 2
	FeatureSearchTestFId3 FeatureSearchTestFeatureID = 3
	FeatureSearchTestFId4 FeatureSearchTestFeatureID = 4
)

func getFeatureSearchTestFeature(testFeatureID FeatureSearchTestFeatureID) FeatureResult {
	var ret FeatureResult
	switch testFeatureID {
	case FeatureSearchTestFId1:
		ret = FeatureResult{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    valuePtr(string(BaselineStatusLow)),
			LowDate:   valuePtr[time.Time](time.Date(2000, time.January, 5, 0, 0, 0, 0, time.UTC)),
			HighDate:  nil,
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
			ImplementationStatuses: []*ImplementationStatus{
				{
					BrowserName:          "barBrowser",
					ImplementationStatus: Available,
				},
				{
					BrowserName:          "fooBrowser",
					ImplementationStatus: Available,
				},
			},
		}
	case FeatureSearchTestFId2:
		ret = FeatureResult{
			FeatureID: "feature2",
			Name:      "Feature 2",
			Status:    valuePtr(string(BaselineStatusHigh)),
			LowDate:   valuePtr[time.Time](time.Date(2000, time.January, 4, 0, 0, 0, 0, time.UTC)),
			HighDate:  valuePtr[time.Time](time.Date(2000, time.January, 31, 0, 0, 0, 0, time.UTC)),
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
			ImplementationStatuses: []*ImplementationStatus{
				{
					BrowserName:          "barBrowser",
					ImplementationStatus: Available,
				},
			},
		}
	case FeatureSearchTestFId3:
		ret = FeatureResult{
			FeatureID: "feature3",
			Name:      "Feature 3",
			Status:    valuePtr(string(BaselineStatusNone)),
			LowDate:   nil,
			HighDate:  nil,
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(35, 50),
				},
			},
			ExperimentalMetrics: nil,
			ImplementationStatuses: []*ImplementationStatus{
				{
					BrowserName:          "fooBrowser",
					ImplementationStatus: Available,
				},
			},
		}
	case FeatureSearchTestFId4:
		ret = FeatureResult{
			FeatureID:              "feature4",
			Name:                   "Feature 4",
			Status:                 nil,
			LowDate:                nil,
			HighDate:               nil,
			StableMetrics:          nil,
			ExperimentalMetrics:    nil,
			ImplementationStatuses: nil,
		}
	}

	return ret
}

func testFeatureSearchAll(ctx context.Context, t *testing.T, client *Client) {
	// Simple test to get all the features without filters.
	expectedPage := FeatureResultPage{
		Features: []FeatureResult{
			getFeatureSearchTestFeature(FeatureSearchTestFId1),
			getFeatureSearchTestFeature(FeatureSearchTestFId2),
			getFeatureSearchTestFeature(FeatureSearchTestFId3),
			getFeatureSearchTestFeature(FeatureSearchTestFId4),
		},
		Total:         4,
		NextPageToken: nil,
	}
	// Test: Get all the results.
	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      nil,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)
}

func testFeatureSearchPagination(ctx context.Context, t *testing.T, client *Client) {
	type PaginationTestCase struct {
		name         string
		pageSize     int
		pageToken    *string // Optional
		expectedPage *FeatureResultPage
	}
	testCases := []PaginationTestCase{
		{
			name:      "page one",
			pageSize:  2,
			pageToken: nil, // First page does not need a page token.
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: valuePtr(encodeFeatureResultOffsetCursor(2)),
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
				},
			},
		},
		{
			name:     "page two",
			pageSize: 2,
			// The token should be made from the token of the previous page's last item
			pageToken: valuePtr(encodeFeatureResultOffsetCursor(2)),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: valuePtr(encodeFeatureResultOffsetCursor(4)),
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: tc.pageToken,
					pageSize:  tc.pageSize,
					node:      nil,
					sort:      defaultSorting(),
				},
				tc.expectedPage,
			)
		})
	}
}

func testFeatureSearchFilters(ctx context.Context, t *testing.T, client *Client) {
	testFeatureAvailableSearchFilters(ctx, t, client)
	testFeatureNotAvailableSearchFilters(ctx, t, client)
	testFeatureCommonFilterCombos(ctx, t, client)
	testFeatureNameFilters(ctx, t, client)
	testFeatureBaselineStatusFilters(ctx, t, client)
}

func testFeatureCommonFilterCombos(ctx context.Context, t *testing.T, client *Client) {
	type FilterComboTestCase struct {
		name         string
		searchNode   *searchtypes.SearchNode
		expectedPage *FeatureResultPage
	}
	testCases := []FilterComboTestCase{
		{
			name: "Available and not available filters",
			// available on barBrowser AND not available on fooBrowser
			searchNode: &searchtypes.SearchNode{
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
			},
			expectedPage: &FeatureResultPage{
				Total:         1,
				NextPageToken: nil,
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: nil,
					pageSize:  100,
					node:      tc.searchNode,
					sort:      defaultSorting(),
				},
				tc.expectedPage,
			)
		})
	}
}

func testFeatureNotAvailableSearchFilters(ctx context.Context, t *testing.T, client *Client) {
	type NotAvailableFilterTestCase struct {
		name         string
		searchNode   *searchtypes.SearchNode
		expectedPage *FeatureResultPage
	}
	testCases := []NotAvailableFilterTestCase{
		{
			name: "single browser: not available on fooBrowser",
			searchNode: &searchtypes.SearchNode{
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
			},
			expectedPage: &FeatureResultPage{
				Total:         2,
				NextPageToken: nil,
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: nil,
					pageSize:  100,
					node:      tc.searchNode,
					sort:      defaultSorting(),
				},
				tc.expectedPage,
			)
		})
	}
}
func testFeatureAvailableSearchFilters(ctx context.Context, t *testing.T, client *Client) {
	type AvailableFilterTestCase struct {
		name         string
		searchNode   *searchtypes.SearchNode
		expectedPage *FeatureResultPage
	}
	testCases := []AvailableFilterTestCase{
		{
			name: "single browser: available on barBrowser",
			// available on barBrowser
			searchNode: &searchtypes.SearchNode{
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
			},
			expectedPage: &FeatureResultPage{
				Total:         2,
				NextPageToken: nil,
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
				},
			},
		},
		{
			name: "multiple browsers: available on either barBrowser OR fooBrowser",
			// available on either barBrowser OR fooBrowser
			searchNode: &searchtypes.SearchNode{
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
			},
			expectedPage: &FeatureResultPage{
				Total:         3,
				NextPageToken: nil,
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: nil,
					pageSize:  100,
					node:      tc.searchNode,
					sort:      defaultSorting(),
				},
				tc.expectedPage,
			)
		})
	}
}

func testFeatureNameFilters(ctx context.Context, t *testing.T, client *Client) {
	// All lower case with partial "feature" name. Should return all.
	expectedResults := []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId1),
		getFeatureSearchTestFeature(FeatureSearchTestFId2),
		getFeatureSearchTestFeature(FeatureSearchTestFId3),
		getFeatureSearchTestFeature(FeatureSearchTestFId4),
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

	expectedPage := FeatureResultPage{
		Total:         4,
		NextPageToken: nil,
		Features:      expectedResults,
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)

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

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)

	// Search for name with "4" Should return only feature 4.
	expectedResults = []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId4),
	}
	expectedPage = FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
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

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)
}

func testFeatureBaselineStatusFilters(ctx context.Context, t *testing.T, client *Client) {
	// Baseline status low only
	expectedResults := []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId1),
	}
	expectedPage := FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node := &searchtypes.SearchNode{
		Operator: searchtypes.OperatorRoot,
		Term:     nil,
		Children: []*searchtypes.SearchNode{
			{
				Operator: searchtypes.OperatorNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierBaselineStatus,
					Value:      "newly",
				},
				Children: nil,
			},
		},
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)

	// baseline_status high only
	expectedResults = []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId2),
	}
	expectedPage = FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node = &searchtypes.SearchNode{
		Operator: searchtypes.OperatorRoot,
		Term:     nil,
		Children: []*searchtypes.SearchNode{
			{
				Operator: searchtypes.OperatorNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierBaselineStatus,
					Value:      "widely",
				},
				Children: nil,
			},
		},
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)

	// Baseline none only, should exclude feature 4 which is nil.
	expectedResults = []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId3),
	}
	expectedPage = FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node = &searchtypes.SearchNode{
		Operator: searchtypes.OperatorRoot,
		Term:     nil,
		Children: []*searchtypes.SearchNode{
			{
				Operator: searchtypes.OperatorNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierBaselineStatus,
					Value:      "limited",
				},
				Children: nil,
			},
		},
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)
}

func testFeatureSearchSortAndPagination(ctx context.Context, t *testing.T, client *Client) {
	type SortAndPaginationTestCase struct {
		name         string
		sortable     Sortable
		pageToken    *string
		expectedPage *FeatureResultPage
	}
	testCases := []SortAndPaginationTestCase{
		{
			name:      "BaselineStatus asc - page 1",
			sortable:  NewBaselineStatusSort(true),
			pageToken: nil,
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: valuePtr(encodeFeatureResultOffsetCursor(2)),
				Features: []FeatureResult{
					// nil status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// none status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
				},
			},
		},
		{
			name:     "BaselineStatus asc - page 2",
			sortable: NewBaselineStatusSort(true),
			// Same page token as the next page token from the previous page.
			pageToken: valuePtr(encodeFeatureResultOffsetCursor(2)),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: valuePtr(encodeFeatureResultOffsetCursor(4)),
				Features: []FeatureResult{
					// high status low date 2000-01-04 high date 2000-01-31
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// low status low date 2000-01-05
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
				},
			},
		},
		{
			name:      "BaselineStatus desc - page 1",
			sortable:  NewBaselineStatusSort(false),
			pageToken: nil,
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: valuePtr(encodeFeatureResultOffsetCursor(2)),
				Features: []FeatureResult{
					// low status low date 2000-01-05
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// high status low date 2000-01-04 high date 2000-01-31
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
				},
			},
		},
		{
			name:     "BaselineStatus desc - page 2",
			sortable: NewBaselineStatusSort(false),
			// Same page token as the next page token from the previous page.
			pageToken: valuePtr(encodeFeatureResultOffsetCursor(2)),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: valuePtr(encodeFeatureResultOffsetCursor(4)),
				Features: []FeatureResult{
					// none status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// nil status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: tc.pageToken,
					pageSize:  2,
					node:      nil,
					sort:      tc.sortable,
				},
				tc.expectedPage,
			)
		})
	}
}

func testFeatureSearchComplexQueries(ctx context.Context, t *testing.T, client *Client) {
	testFeatureSearchSortAndPagination(ctx, t, client)
}

func testFeatureSearchSort(ctx context.Context, t *testing.T, client *Client) {
	testFeatureSearchSortName(ctx, t, client)
	testFeatureSearchSortBaselineStatus(ctx, t, client)
	testFeatureSearchSortBrowserImpl(ctx, t, client)
}

// nolint: dupl // WONTFIX. Only duplicated because the feature filter test yields similar results.
func testFeatureSearchSortName(ctx context.Context, t *testing.T, client *Client) {
	type NameSortTestCase struct {
		name         string
		sortable     Sortable
		expectedPage *FeatureResultPage
	}
	testCases := []NameSortTestCase{
		{
			name:     "Name asc",
			sortable: NewFeatureNameSort(true),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
				},
			},
		},
		{
			name:     "Name desc",
			sortable: NewFeatureNameSort(false),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: nil,
					pageSize:  100,
					node:      nil,
					sort:      tc.sortable,
				},
				tc.expectedPage,
			)
		})
	}
}

// nolint: dupl // Okay to duplicate for tests
func testFeatureSearchSortBaselineStatus(ctx context.Context, t *testing.T, client *Client) {
	type BaselineStatusSortCase struct {
		name         string
		sortable     Sortable
		expectedPage *FeatureResultPage
	}
	testCases := []BaselineStatusSortCase{
		{
			name:     "BaselineStatus asc",
			sortable: NewBaselineStatusSort(true),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// nil status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// none status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// high status low date 2000-01-04 high date 2000-01-31
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// low status low date 2000-01-05
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
				},
			},
		},
		{
			name:     "BaselineStatus desc",
			sortable: NewBaselineStatusSort(false),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// low status low date 2000-01-05
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// high status low date 2000-01-04 high date 2000-01-31
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// none status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// nil status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: nil,
					pageSize:  100,
					node:      nil,
					sort:      tc.sortable,
				},
				tc.expectedPage,
			)
		})
	}
}

func testFeatureSearchSortBrowserImpl(ctx context.Context, t *testing.T, client *Client) {
	type BaselineStatusSortCase struct {
		name         string
		sortable     Sortable
		expectedPage *FeatureResultPage
	}
	testCases := []BaselineStatusSortCase{
		{
			name:     "BrowserImpl fooBrowser Stable asc",
			sortable: NewBrowserImplSort(true, "fooBrowser", true),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// null metric, null status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// 0 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// 0.7 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// 1.0 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
				},
			},
		},
		{
			name:     "BrowserImpl fooBrowser Stable desc",
			sortable: NewBrowserImplSort(false, "fooBrowser", true),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// 1.0 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// 0.7 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// 0 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// null metric, null status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
				},
			},
		},
		{
			name:     "BrowserImpl fooBrowser Experimental asc",
			sortable: NewBrowserImplSort(true, "fooBrowser", false),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// null metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// null metric, null status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// 1.0 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// 1.0 metric, null status
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
				},
			},
		},
		{
			name:     "BrowserImpl fooBrowser Experimental desc",
			sortable: NewBrowserImplSort(false, "fooBrowser", false),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// 1.0 metric, null status
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// 1.0 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// null metric, null status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// null metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: nil,
					pageSize:  100,
					node:      nil,
					sort:      tc.sortable,
				},
				tc.expectedPage,
			)
		})
	}
}

func TestFeaturesSearch(t *testing.T) {
	client := getTestDatabase(t)
	ctx := context.Background()
	setupRequiredTablesForFeaturesSearch(ctx, client, t)

	// Try with default GCPSpannerBaseQuery
	t.Run("gcp spanner queries", func(t *testing.T) {
		testFeatureSearchAll(ctx, t, client)
		testFeatureSearchPagination(ctx, t, client)
		testFeatureSearchFilters(ctx, t, client)
		testFeatureSearchSort(ctx, t, client)
		testFeatureSearchComplexQueries(ctx, t, client)
	})

	// Try with LocalFeatureBaseQuery
	t.Run("local spanner queries", func(t *testing.T) {
		client.SetFeatureSearchBaseQuery(LocalFeatureBaseQuery{})
		testFeatureSearchAll(ctx, t, client)
		testFeatureSearchPagination(ctx, t, client)
		testFeatureSearchFilters(ctx, t, client)
		testFeatureSearchSort(ctx, t, client)
		testFeatureSearchComplexQueries(ctx, t, client)
	})
}

type featureSearchArgs struct {
	pageToken *string
	pageSize  int
	node      *searchtypes.SearchNode
	sort      Sortable
}

func assertFeatureSearch(
	ctx context.Context,
	t *testing.T,
	client *Client,
	args featureSearchArgs,
	expectedPage *FeatureResultPage) {
	page, err := client.FeaturesSearch(
		ctx,
		args.pageToken,
		args.pageSize,
		args.node,
		args.sort,
		// TODO. When the tests assert both views, remove this and allow the test
		// to pass this.
		defaultWPTMetricView(),
	)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(expectedPage),
			PrettyPrintFeatureResultPage(page))
	}
}

func AreFeatureResultPagesEqual(a, b *FeatureResultPage) bool {
	return a.Total == b.Total &&
		((a.NextPageToken == nil && b.NextPageToken == nil) ||
			((a.NextPageToken != nil && b.NextPageToken != nil) && *a.NextPageToken == *b.NextPageToken)) &&
		AreFeatureResultsSlicesEqual(a.Features, b.Features)
}

func AreFeatureResultsSlicesEqual(a, b []FeatureResult) bool {
	return slices.EqualFunc[[]FeatureResult](a, b, AreFeatureResultsEqual)
}

func AreFeatureResultsEqual(a, b FeatureResult) bool {
	if a.FeatureID != b.FeatureID ||
		a.Name != b.Name ||
		!reflect.DeepEqual(a.Status, b.Status) ||
		!reflect.DeepEqual(a.LowDate, b.LowDate) ||
		!reflect.DeepEqual(a.HighDate, b.HighDate) ||
		!AreMetricsEqual(a.StableMetrics, b.StableMetrics) ||
		!AreMetricsEqual(a.ExperimentalMetrics, b.ExperimentalMetrics) ||
		!AreImplementationStatusesEqual(a.ImplementationStatuses, b.ImplementationStatuses) {
		return false
	}

	return true
}

func AreImplementationStatusesEqual(a, b []*ImplementationStatus) bool {
	return slices.EqualFunc[[]*ImplementationStatus](a, b, func(a, b *ImplementationStatus) bool {
		return a.BrowserName == b.BrowserName &&
			(a.ImplementationStatus == b.ImplementationStatus)
	})
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

func PrintNullableField[T any](in *T) string {
	if in == nil {
		return "NIL"
	}

	return fmt.Sprintf("%v", *in)
}

func PrettyPrintFeatureResult(result FeatureResult) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "\tFeatureID: %s\n", result.FeatureID)
	fmt.Fprintf(&builder, "\tName: %s\n", result.Name)

	fmt.Fprintf(&builder, "\tStatus: %s\n", PrintNullableField(result.Status))
	fmt.Fprintf(&builder, "\tLowDate: %s\n", PrintNullableField(result.LowDate))
	fmt.Fprintf(&builder, "\tHighDate: %s\n", PrintNullableField(result.HighDate))

	fmt.Fprintln(&builder, "\tStable Metrics:")
	for _, metric := range result.StableMetrics {
		fmt.Fprint(&builder, PrettyPrintMetric(metric))
	}

	fmt.Fprintln(&builder, "\tExperimental Metrics:")
	for _, metric := range result.ExperimentalMetrics {
		fmt.Fprint(&builder, PrettyPrintMetric(metric))
	}
	fmt.Fprintln(&builder, "\tImplementation Statuses:")
	for _, status := range result.ImplementationStatuses {
		fmt.Fprint(&builder, PrettyPrintImplementationStatus(status))
	}
	fmt.Fprintln(&builder)

	return builder.String()
}

func PrettyPrintImplementationStatus(status *ImplementationStatus) string {
	var builder strings.Builder
	if status == nil {
		return "\t\tNIL STATUS\n"
	}
	fmt.Fprintf(&builder, "\t\tBrowserName: %s\n", status.BrowserName)
	fmt.Fprintf(&builder, "\t\tStatus: %s\n", status.ImplementationStatus)

	return builder.String()
}

func PrettyPrintMetric(metric *FeatureResultMetric) string {
	var builder strings.Builder
	if metric == nil {
		return "\t\tNIL\n"
	}
	fmt.Fprintf(&builder, "\t\tBrowserName: %s\n", metric.BrowserName)
	fmt.Fprintf(&builder, "\t\tPassRate: %s\n", PrettyPrintPassRate(metric.PassRate))

	return builder.String()
}

func PrettyPrintPassRate(passRate *big.Rat) string {
	if passRate == nil {
		return "\t\tNIL\n"
	}

	return passRate.String() + "\n"
}

func PrettyPrintPageToken(token *string) string {
	if token == nil {
		return "NIL\n"
	}

	return *token + "\n"
}

func PrettyPrintFeatureResultPage(page *FeatureResultPage) string {
	if page == nil {
		return ""
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "Total: %d\n", page.Total)
	fmt.Fprintf(&builder, "NextPageToken: %s\n", PrettyPrintPageToken(page.NextPageToken))
	fmt.Fprint(&builder, PrettyPrintFeatureResults(page.Features))

	return builder.String()
}

// PrettyPrintFeatureResults returns a formatted string representation of a slice of FeatureResult structs.
func PrettyPrintFeatureResults(results []FeatureResult) string {
	var builder strings.Builder
	for _, result := range results {
		fmt.Fprint(&builder, PrettyPrintFeatureResult(result))
	}

	return builder.String()
}
