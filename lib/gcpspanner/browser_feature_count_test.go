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
	"testing"
	"time"
)

func loadDataForListBrowserFeatureCountMetric(ctx context.Context, t *testing.T, client *Client) {
	webFeatures := []WebFeature{
		{FeatureKey: "FeatureX", Name: "Cool API", Description: "text", DescriptionHTML: "<html>"},
		{FeatureKey: "FeatureY", Name: "Super API", Description: "text", DescriptionHTML: "<html>"},
		{FeatureKey: "FeatureZ", Name: "Neat API", Description: "text", DescriptionHTML: "<html>"},
		{FeatureKey: "FeatureW", Name: "Amazing API", Description: "text", DescriptionHTML: "<html>"},
		{FeatureKey: "FeatureV", Name: "Awesome API", Description: "text", DescriptionHTML: "<html>"},
	}
	for _, feature := range webFeatures {
		_, err := client.upsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert of features. %s", err.Error())
		}
	}

	browserReleases := []BrowserRelease{
		// Chrome Releases
		{BrowserName: "chrome", BrowserVersion: "99", ReleaseDate: time.Date(2023, 12, 5, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "chrome", BrowserVersion: "100", ReleaseDate: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "chrome", BrowserVersion: "101", ReleaseDate: time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC)},

		// Firefox Releases
		{BrowserName: "firefox", BrowserVersion: "80", ReleaseDate: time.Date(2023, 11, 15, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "firefox", BrowserVersion: "81", ReleaseDate: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "firefox", BrowserVersion: "82", ReleaseDate: time.Date(2024, 2, 20, 0, 0, 0, 0, time.UTC)},

		// Chrome Android releases
		{BrowserName: "chrome_android", BrowserVersion: "100", ReleaseDate: time.Date(2023, 12, 7, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "chrome_android", BrowserVersion: "101", ReleaseDate: time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)},

		// Firefox Android releases
		{BrowserName: "firefox_android", BrowserVersion: "80", ReleaseDate: time.Date(2023, 11, 17, 0, 0, 0, 0, time.UTC)},
	}
	for _, release := range browserReleases {
		err := client.InsertBrowserRelease(ctx, release)
		if err != nil {
			t.Errorf("unexpected error during insert of releases. %s", err.Error())
		}
	}

	browserFeatureAvailabilities := map[string][]BrowserFeatureAvailability{
		"FeatureX": {
			{BrowserName: "chrome", BrowserVersion: "100"},
			{BrowserName: "chrome_android", BrowserVersion: "100"},
		},
		"FeatureY": {
			{BrowserName: "chrome", BrowserVersion: "100"},
			{BrowserName: "firefox", BrowserVersion: "80"},
		},
		"FeatureZ": {
			{BrowserName: "chrome", BrowserVersion: "101"},
		},
		"FeatureV": {
			{BrowserName: "chrome", BrowserVersion: "101"},
			{BrowserName: "chrome_android", BrowserVersion: "101"},
			{BrowserName: "firefox_android", BrowserVersion: "80"},
		},
		"FeatureW": {
			{BrowserName: "firefox", BrowserVersion: "81"},
		},
	}

	err := client.SyncBrowserFeatureAvailabilities(ctx, browserFeatureAvailabilities)
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}
}

func TestListBrowserFeatureCountMetric(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	loadDataForListBrowserFeatureCountMetric(ctx, t, spannerClient)

	// TODO Currently we are not clearing the tables between test cases.
	// We should change this in the future. Be careful.
	// In the meantime, be careful with the order of the test cases.
	testCases := []struct {
		testName                       string
		targetBrowser                  string
		targetMobileBrowser            *string
		startAt                        time.Time
		endAt                          time.Time
		pageSize                       int
		excludedFeatureKeysToInsert    []string
		discouragedFeatureKeysToInsert []string
		inputCursor                    *string
		expectedResult                 *BrowserFeatureCountResultPage
	}{
		{
			testName:                       "Test 1a. First Page",
			targetBrowser:                  "chrome",
			targetMobileBrowser:            valuePtr("chrome_android"),
			startAt:                        time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
			endAt:                          time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			excludedFeatureKeysToInsert:    nil,
			discouragedFeatureKeysToInsert: nil,
			pageSize:                       2,
			inputCursor:                    nil,
			expectedResult: &BrowserFeatureCountResultPage{
				NextPageToken: valuePtr(encodeBrowserFeatureCountCursor(
					time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), 1)),
				Metrics: []BrowserFeatureCountMetric{
					{
						ReleaseDate:  time.Date(2023, 12, 5, 0, 0, 0, 0, time.UTC),
						FeatureCount: 0,
					},
					{
						ReleaseDate:  time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
						FeatureCount: 1,
					},
				},
			},
		},
		{
			testName:                       "Test 1b. Second Page",
			targetBrowser:                  "chrome",
			targetMobileBrowser:            valuePtr("chrome_android"),
			startAt:                        time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
			endAt:                          time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			excludedFeatureKeysToInsert:    nil,
			discouragedFeatureKeysToInsert: nil,
			pageSize:                       2,
			inputCursor: valuePtr(encodeBrowserFeatureCountCursor(
				time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), 1)),
			expectedResult: &BrowserFeatureCountResultPage{
				NextPageToken: nil,
				Metrics: []BrowserFeatureCountMetric{
					{
						ReleaseDate:  time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC),
						FeatureCount: 2,
					},
				},
			},
		},
		{
			testName:                       "Test 2. Get the point but still count all the features beforehand.",
			targetBrowser:                  "chrome",
			targetMobileBrowser:            nil,
			startAt:                        time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
			endAt:                          time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			excludedFeatureKeysToInsert:    nil,
			discouragedFeatureKeysToInsert: nil,
			pageSize:                       100,
			inputCursor:                    nil,
			expectedResult: &BrowserFeatureCountResultPage{
				NextPageToken: nil,
				Metrics: []BrowserFeatureCountMetric{
					{
						ReleaseDate:  time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC),
						FeatureCount: 4,
					},
				},
			},
		},
		// Ensure that the `ListBrowserFeatureCountMetric` query correctly handles a scenario where a browser
		// (`Firefox` in this case) *has* releases within the specified date range but *does not have any new*
		// `BrowserFeatureAvailabilities` entries for those releases within that range.
		{
			testName:                       "Test 3. No availabilities for one browser.",
			targetBrowser:                  "firefox",
			targetMobileBrowser:            nil,
			startAt:                        time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
			endAt:                          time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			excludedFeatureKeysToInsert:    nil,
			discouragedFeatureKeysToInsert: nil,
			pageSize:                       3,
			inputCursor:                    nil,
			expectedResult: &BrowserFeatureCountResultPage{
				NextPageToken: nil,
				Metrics: []BrowserFeatureCountMetric{
					{
						ReleaseDate:  time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
						FeatureCount: 2,
					},
					// No increase
					{
						ReleaseDate:  time.Date(2024, 2, 20, 0, 0, 0, 0, time.UTC),
						FeatureCount: 2,
					},
				},
			},
		},
		{
			testName:                    "Test 4. With Excluded Features",
			targetBrowser:               "chrome",
			targetMobileBrowser:         nil,
			startAt:                     time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
			endAt:                       time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			pageSize:                    3,
			inputCursor:                 nil,
			excludedFeatureKeysToInsert: []string{"FeatureY", "FeatureZ"},
			// Have overlap between excludedFeatureKeysToInsert and discouragedFeatureKeysToInsert
			discouragedFeatureKeysToInsert: []string{"FeatureZ"},
			expectedResult: &BrowserFeatureCountResultPage{
				NextPageToken: valuePtr(encodeBrowserFeatureCountCursor(
					time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC), 2)),
				Metrics: []BrowserFeatureCountMetric{
					{
						ReleaseDate:  time.Date(2023, 12, 5, 0, 0, 0, 0, time.UTC),
						FeatureCount: 0,
					},
					{
						ReleaseDate:  time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
						FeatureCount: 1, // FeatureY excluded / discouraged
					},
					{
						ReleaseDate:  time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC),
						FeatureCount: 2, // FeatureY and FeatureZ excluded / discouraged
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			var result *BrowserFeatureCountResultPage
			var err error

			// Insert excluded feature keys into the ExcludedFeatureKeys table
			for _, featureKey := range tc.excludedFeatureKeysToInsert {
				if err := spannerClient.InsertExcludedFeatureKey(ctx, featureKey); err != nil {
					t.Fatalf("Failed to insert excluded feature key: %v", err)
				}
			}

			// Insert discouraged feature keys into the FeatureDiscouragedDetails table
			for _, featureKey := range tc.discouragedFeatureKeysToInsert {
				if err := spannerClient.UpsertFeatureDiscouragedDetails(
					ctx, featureKey, FeatureDiscouragedDetails{
						AccordingTo:  nil,
						Alternatives: nil,
					}); err != nil {
					t.Fatalf("Failed to insert feature discouraged details: %v", err)
				}
			}

			result, err = spannerClient.ListBrowserFeatureCountMetric(
				ctx, tc.targetBrowser, tc.targetMobileBrowser, tc.startAt, tc.endAt, tc.pageSize, tc.inputCursor)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(tc.expectedResult, result) {
				t.Errorf("unexpected result.\nExpected %+v\nReceived %+v", tc.expectedResult, result)
			}

		})
	}

}
