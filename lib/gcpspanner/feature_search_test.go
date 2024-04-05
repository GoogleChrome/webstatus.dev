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
			Status:    BaselineStatusLow,
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
			Status:    BaselineStatusNone,
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
		err := client.UpsertWPTRunFeatureMetrics(
			ctx, metric.ExternalRunID,
			// Insert them individually because sampleRunMetrics has metrics from different runs.
			[]WPTRunFeatureMetric{metric.WPTRunFeatureMetric})
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

}

func testFeatureSearchAll(ctx context.Context, t *testing.T, client *Client) {
	// Simple test to get all the features without filters.
	//nolint: dupl // Okay to duplicate for tests
	expectedResults := []FeatureResult{
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusLow),
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
			Status:    string(BaselineStatusNone),
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
	expectedPage := FeatureResultPage{
		Features:      expectedResults,
		Total:         4,
		NextPageToken: nil,
	}
	// Test: Get all the results.
	page, err := client.FeaturesSearch(ctx, nil, 100, nil, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
	}
}

func testFeatureSearchPagination(ctx context.Context, t *testing.T, client *Client) {
	// Test: Get all the results with pagination.
	// nolint: dupl
	expectedResultsPageOne := []FeatureResult{
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusLow),
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
	expectedToken := encodeFeatureResultCursor(
		defaultSorting(),
		expectedResultsPageOne[len(expectedResultsPageOne)-1])
	expectedPage := FeatureResultPage{
		Total:         4,
		NextPageToken: &expectedToken,
		Features:      expectedResultsPageOne,
	}
	page, err := client.FeaturesSearch(ctx, nil, 2, nil, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
	}

	expectedResultsPageTwo := []FeatureResult{
		{
			FeatureID: "feature3",
			Name:      "Feature 3",
			Status:    string(BaselineStatusNone),
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

	expectedToken = encodeFeatureResultCursor(
		defaultSorting(),
		expectedResultsPageTwo[len(expectedResultsPageTwo)-1])
	expectedPageTwo := FeatureResultPage{
		Total:         4,
		Features:      expectedResultsPageTwo,
		NextPageToken: &expectedToken,
	}

	// With regular token
	page, err = client.FeaturesSearch(ctx, page.NextPageToken, 2, nil, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPageTwo, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPageTwo),
			PrettyPrintFeatureResultPage(page))
	}

	// With offset token
	expectedOffsetPageTwo := FeatureResultPage{
		Total:         4,
		Features:      expectedResultsPageTwo,
		NextPageToken: &expectedToken,
	}
	offsetToken := encodeFeatureResultOffsetCursor(2)
	offsetPage, err := client.FeaturesSearch(ctx, &offsetToken, 2, nil, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(offsetPage)
	if !AreFeatureResultPagesEqual(&expectedOffsetPageTwo, offsetPage) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedOffsetPageTwo),
			PrettyPrintFeatureResultPage(page))
	}

	if *offsetPage.NextPageToken != *page.NextPageToken {
		t.Error("pagination from last id and offset should generate the same next token")
	}

	// Last page should have no results and should have no token.
	var expectedResultsPageThree []FeatureResult
	expectedPageThree := FeatureResultPage{
		Total:         4,
		NextPageToken: nil,
		Features:      expectedResultsPageThree,
	}
	page, err = client.FeaturesSearch(ctx, page.NextPageToken, 2, nil, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}

	if !AreFeatureResultPagesEqual(&expectedPageThree, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPageThree),
			PrettyPrintFeatureResultPage(page))
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
	expectedPage := FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
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

	page, err := client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
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
	expectedPage := FeatureResultPage{
		Total:         2,
		NextPageToken: nil,
		Features:      expectedResults,
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
	page, err := client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
	}
}
func testFeatureAvailableSearchFilters(ctx context.Context, t *testing.T, client *Client) {
	// Single browser
	// nolint: dupl
	expectedResults := []FeatureResult{
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusLow),
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
	expectedPage := FeatureResultPage{
		Total:         2,
		NextPageToken: nil,
		Features:      expectedResults,
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
	page, err := client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
	}

	// Multiple browsers.
	expectedResults = []FeatureResult{
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusLow),
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
			Status:    string(BaselineStatusNone),
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

	expectedPage = FeatureResultPage{
		Total:         3,
		NextPageToken: nil,
		Features:      expectedResults,
	}

	page, err = client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
	}
}

func testFeatureNameFilters(ctx context.Context, t *testing.T, client *Client) {
	// All lower case with partial "feature" name. Should return all.
	//nolint: dupl // Okay to duplicate for tests
	expectedResults := []FeatureResult{
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusLow),
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
			Status:    string(BaselineStatusNone),
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

	expectedPage := FeatureResultPage{
		Total:         4,
		NextPageToken: nil,
		Features:      expectedResults,
	}

	page, err := client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
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

	page, err = client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
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

	page, err = client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
	}
}

func testFeatureBaselineStatusFilters(ctx context.Context, t *testing.T, client *Client) {
	// Baseline status low only
	//nolint: dupl // Okay to duplicate for tests
	expectedResults := []FeatureResult{
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusLow),
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

	page, err := client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
	}

	// baseline_status high only
	expectedResults = []FeatureResult{
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

	page, err = client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
	}

	// Baseline none only, should exclude feature 4 which is undefined.
	expectedResults = []FeatureResult{
		{
			FeatureID: "feature3",
			Name:      "Feature 3",
			Status:    string(BaselineStatusNone),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(35, 50),
				},
			},
			ExperimentalMetrics: nil,
		},
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

	page, err = client.FeaturesSearch(ctx, nil, 100, node, defaultSorting())
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
	}

}

func testFeatureSearchSortAndPagination(ctx context.Context, t *testing.T, client *Client) {
	// BaselineStatus asc
	sortByAsc := NewBaselineStatusSort(true)
	//nolint: dupl // Okay to duplicate for tests
	expectedPageOneResults := []FeatureResult{
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
			Status:    string(BaselineStatusLow),
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
	expectedToken := encodeFeatureResultCursor(
		sortByAsc,
		expectedPageOneResults[len(expectedPageOneResults)-1])
	expectedPageOne := FeatureResultPage{
		Total:         4,
		NextPageToken: &expectedToken,
		Features:      expectedPageOneResults,
	}
	// Test: Get the first page of results.
	page, err := client.FeaturesSearch(ctx, nil, 2, nil, sortByAsc)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPageOne, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPageOne),
			PrettyPrintFeatureResultPage(page))
	}

	// Page 2
	expectedPageTwoResults := []FeatureResult{
		{
			FeatureID: "feature3",
			Name:      "Feature 3",
			Status:    string(BaselineStatusNone),
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
	expectedToken = encodeFeatureResultCursor(
		sortByAsc,
		expectedPageTwoResults[len(expectedPageTwoResults)-1])
	expectedPageTwo := FeatureResultPage{
		Total:         4,
		NextPageToken: &expectedToken,
		Features:      expectedPageTwoResults,
	}
	// Get the page 2 of results using the cursor token
	page, err = client.FeaturesSearch(ctx, page.NextPageToken, 2, nil, sortByAsc)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPageTwo, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPageTwo),
			PrettyPrintFeatureResultPage(page))
	}

	// Get the page 2 of results using the offset
	expectedOffsetPageTwo := FeatureResultPage{
		Total:         4,
		Features:      expectedPageTwoResults,
		NextPageToken: &expectedToken,
	}
	offsetToken := encodeFeatureResultOffsetCursor(2)
	offsetPage, err := client.FeaturesSearch(ctx, &offsetToken, 2, nil, sortByAsc)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(offsetPage)
	if !AreFeatureResultPagesEqual(&expectedOffsetPageTwo, offsetPage) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedOffsetPageTwo),
			PrettyPrintFeatureResultPage(page))
	}
}

func testFeatureSearchComplexQueries(ctx context.Context, t *testing.T, client *Client) {
	testFeatureSearchSortAndPagination(ctx, t, client)
}

func testFeatureSearchSort(ctx context.Context, t *testing.T, client *Client) {
	testFeatureSearchSortName(ctx, t, client)
	testFeatureSearchSortBaselineStatus(ctx, t, client)
}

// nolint: dupl // Okay to duplicate for tests
func testFeatureSearchSortName(ctx context.Context, t *testing.T, client *Client) {
	// Name asc
	sortByAsc := NewFeatureNameSort(true)
	expectedResults := []FeatureResult{
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusLow),
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
			Status:    string(BaselineStatusNone),
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
	expectedPage := FeatureResultPage{
		Total:         4,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	// Test: Get all the results.
	page, err := client.FeaturesSearch(ctx, nil, 100, nil, sortByAsc)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
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
			Status:    string(BaselineStatusNone),
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
			Status:    string(BaselineStatusLow),
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
	expectedPage = FeatureResultPage{
		Total:         4,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	// Test: Get all the results.
	page, err = client.FeaturesSearch(ctx, nil, 100, nil, sortByDesc)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
	}
}

// nolint: dupl // Okay to duplicate for tests
func testFeatureSearchSortBaselineStatus(ctx context.Context, t *testing.T, client *Client) {
	// BaselineStatus asc
	sortByAsc := NewBaselineStatusSort(true)
	//nolint: dupl // Okay to duplicate for tests
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
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusLow),
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
			FeatureID: "feature3",
			Name:      "Feature 3",
			Status:    string(BaselineStatusNone),
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
	expectedPage := FeatureResultPage{
		Total:         4,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	// Test: Get all the results.
	page, err := client.FeaturesSearch(ctx, nil, 100, nil, sortByAsc)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
	}

	// BaselineStatus desc
	sortByDesc := NewBaselineStatusSort(false)
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
			Status:    string(BaselineStatusNone),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(35, 50),
				},
			},
			ExperimentalMetrics: nil,
		},
		{
			FeatureID: "feature1",
			Name:      "Feature 1",
			Status:    string(BaselineStatusLow),
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
	expectedPage = FeatureResultPage{
		Total:         4,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	// Test: Get all the results.
	page, err = client.FeaturesSearch(ctx, nil, 100, nil, sortByDesc)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(&expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(&expectedPage),
			PrettyPrintFeatureResultPage(page))
	}
}

func TestFeaturesSearch(t *testing.T) {
	client := getTestDatabase(t)
	ctx := context.Background()
	setupRequiredTablesForFeaturesSearch(ctx, client, t)

	// Try with default GCPSpannerBaseQuery
	testFeatureSearchAll(ctx, t, client)
	testFeatureSearchPagination(ctx, t, client)
	testFeatureSearchFilters(ctx, t, client)
	testFeatureSearchSort(ctx, t, client)
	testFeatureSearchComplexQueries(ctx, t, client)

	// Try with LocalFeatureBaseQuery
	client.SetFeatureSearchBaseQuery(LocalFeatureBaseQuery{})
	testFeatureSearchAll(ctx, t, client)
	testFeatureSearchPagination(ctx, t, client)
	testFeatureSearchFilters(ctx, t, client)
	testFeatureSearchSort(ctx, t, client)
	testFeatureSearchComplexQueries(ctx, t, client)
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
	fmt.Fprintf(&builder, "\tFeatureID: %s\n", result.FeatureID)
	fmt.Fprintf(&builder, "\tName: %s\n", result.Name)
	fmt.Fprintf(&builder, "\tStatus: %s\n", result.Status)

	fmt.Fprintln(&builder, "\tStable Metrics:")
	for _, metric := range result.StableMetrics {
		fmt.Fprint(&builder, PrettyPrintMetric(metric))
	}

	fmt.Fprintln(&builder, "\tExperimental Metrics:")
	for _, metric := range result.ExperimentalMetrics {
		fmt.Fprint(&builder, PrettyPrintMetric(metric))
	}
	fmt.Fprintln(&builder)

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

// encodeFeatureResultOffsetCursor provides a wrapper around the generic encodeCursor.
func encodeFeatureResultOffsetCursor(offset int) string {
	return encodeCursor(FeatureResultOffsetCursor{
		Offset: offset,
	})
}
