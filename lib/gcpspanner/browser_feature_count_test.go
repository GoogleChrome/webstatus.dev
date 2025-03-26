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
		{FeatureKey: "FeatureX", Name: "Cool API"},
		{FeatureKey: "FeatureY", Name: "Super API"},
		{FeatureKey: "FeatureZ", Name: "Neat API"},
		{FeatureKey: "FeatureW", Name: "Amazing API"},
	}
	for _, feature := range webFeatures {
		_, err := client.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert of features. %s", err.Error())
		}
	}

	browserReleases := []BrowserRelease{
		// fooBrowser Releases
		{BrowserName: "fooBrowser", BrowserVersion: "99", ReleaseDate: time.Date(2023, 12, 5, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "fooBrowser", BrowserVersion: "100", ReleaseDate: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "fooBrowser", BrowserVersion: "101", ReleaseDate: time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC)},

		// barBrowser Releases
		{BrowserName: "barBrowser", BrowserVersion: "80", ReleaseDate: time.Date(2023, 11, 15, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "barBrowser", BrowserVersion: "81", ReleaseDate: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "barBrowser", BrowserVersion: "82", ReleaseDate: time.Date(2024, 2, 20, 0, 0, 0, 0, time.UTC)},
	}
	for _, release := range browserReleases {
		err := client.InsertBrowserRelease(ctx, release)
		if err != nil {
			t.Errorf("unexpected error during insert of releases. %s", err.Error())
		}
	}

	browserFeatureAvailabilities := []struct {
		FeatureKey string
		BrowserFeatureAvailability
	}{
		// fooBrowser Availabilities
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "fooBrowser", BrowserVersion: "100"},
			FeatureKey:                 "FeatureX",
		}, // Available from fooBrowser 100
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "fooBrowser", BrowserVersion: "100"},
			FeatureKey:                 "FeatureY",
		},
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "fooBrowser", BrowserVersion: "101"},
			FeatureKey:                 "FeatureZ",
		}, // Available from fooBrowser 101

		// barBrowser Availabilities
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "barBrowser", BrowserVersion: "80"},
			FeatureKey:                 "FeatureY",
		}, // Available from barBrowser 80
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "barBrowser", BrowserVersion: "81"},
			FeatureKey:                 "FeatureW",
		}, // Available from barBrowser 81
	}
	for _, availability := range browserFeatureAvailabilities {
		err := client.UpsertBrowserFeatureAvailability(ctx,
			availability.FeatureKey, availability.BrowserFeatureAvailability)
		if err != nil {
			t.Errorf("unexpected error during insert. %s", err.Error())
		}
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
		browser                        string
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
			browser:                        "fooBrowser",
			startAt:                        time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
			endAt:                          time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			excludedFeatureKeysToInsert:    nil,
			discouragedFeatureKeysToInsert: nil,
			pageSize:                       2,
			inputCursor:                    nil,
			expectedResult: &BrowserFeatureCountResultPage{
				NextPageToken: valuePtr(encodeBrowserFeatureCountCursor(
					time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), 2)),
				Metrics: []BrowserFeatureCountMetric{
					{
						ReleaseDate:  time.Date(2023, 12, 5, 0, 0, 0, 0, time.UTC),
						FeatureCount: 0,
					},
					{
						ReleaseDate:  time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
						FeatureCount: 2,
					},
				},
			},
		},
		{
			testName:                       "Test 1b. Second Page",
			browser:                        "fooBrowser",
			startAt:                        time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
			endAt:                          time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			excludedFeatureKeysToInsert:    nil,
			discouragedFeatureKeysToInsert: nil,
			pageSize:                       3,
			inputCursor: valuePtr(encodeBrowserFeatureCountCursor(
				time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), 2)),
			expectedResult: &BrowserFeatureCountResultPage{
				NextPageToken: nil,
				Metrics: []BrowserFeatureCountMetric{
					{
						ReleaseDate:  time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC),
						FeatureCount: 3,
					},
				},
			},
		},
		{
			testName:                       "Test 2. Get the point but still count all the features beforehand.",
			browser:                        "fooBrowser",
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
						FeatureCount: 3,
					},
				},
			},
		},
		// Ensure that the `ListBrowserFeatureCountMetric` query correctly handles a scenario where a browser
		// (`barBrowser` in this case) *has* releases within the specified date range but *does not have any new*
		// `BrowserFeatureAvailabilities` entries for those releases within that range.
		{
			testName:                       "Test 3. No availabilities for one browser.",
			browser:                        "barBrowser",
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
			browser:                     "fooBrowser",
			startAt:                     time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
			endAt:                       time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			pageSize:                    3,
			inputCursor:                 nil,
			excludedFeatureKeysToInsert: []string{"FeatureY", "FeatureZ"},
			// Have overlap between excludedFeatureKeysToInsert and discouragedFeatureKeysToInsert
			discouragedFeatureKeysToInsert: []string{"FeatureZ"},
			expectedResult: &BrowserFeatureCountResultPage{
				NextPageToken: valuePtr(encodeBrowserFeatureCountCursor(
					time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC), 1)),
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
						FeatureCount: 1, // FeatureY and FeatureZ excluded / discouraged
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
				ctx, tc.browser, tc.startAt, tc.endAt, tc.pageSize, tc.inputCursor)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(tc.expectedResult, result) {
				t.Errorf("unexpected result.\nExpected %+v\nReceived %+v", tc.expectedResult, result)
			}

		})
	}

}
