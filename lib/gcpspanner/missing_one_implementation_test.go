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
	"slices"
	"testing"
	"time"
)

func loadDataForListMissingOneImplCounts(ctx context.Context, t *testing.T, client *Client) {
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
		{BrowserName: "fooBrowser", BrowserVersion: "110", ReleaseDate: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "fooBrowser", BrowserVersion: "111", ReleaseDate: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "fooBrowser", BrowserVersion: "112", ReleaseDate: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "fooBrowser", BrowserVersion: "113", ReleaseDate: time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC)},

		// barBrowser Releases
		{BrowserName: "barBrowser", BrowserVersion: "113", ReleaseDate: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "barBrowser", BrowserVersion: "114", ReleaseDate: time.Date(2024, 3, 28, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "barBrowser", BrowserVersion: "115", ReleaseDate: time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)},

		// bazBrowser Releases
		{BrowserName: "bazBrowser", BrowserVersion: "16.4", ReleaseDate: time.Date(2024, 1, 25, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "bazBrowser", BrowserVersion: "16.5", ReleaseDate: time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "bazBrowser", BrowserVersion: "17", ReleaseDate: time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)},
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
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "fooBrowser", BrowserVersion: "111"},
			FeatureKey:                 "FeatureX",
		}, // Available from fooBrowser 111
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "fooBrowser", BrowserVersion: "112"},
			FeatureKey:                 "FeatureY",
		}, // Available from fooBrowser 112
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "fooBrowser", BrowserVersion: "112"},
			FeatureKey:                 "FeatureZ",
		}, // Available from fooBrowser 112
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "fooBrowser", BrowserVersion: "113"},
			FeatureKey:                 "FeatureW",
		}, // Available from fooBrowser 113

		// barBrowser Availabilities
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "barBrowser", BrowserVersion: "113"},
			FeatureKey:                 "FeatureX",
		}, // Available from barBrowser 113
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "barBrowser", BrowserVersion: "113"},
			FeatureKey:                 "FeatureZ",
		}, // Available from barBrowser 113
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "barBrowser", BrowserVersion: "114"},
			FeatureKey:                 "FeatureY",
		}, // Available from barBrowser 114
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "barBrowser", BrowserVersion: "115"},
			FeatureKey:                 "FeatureW",
		}, // Available from barBrowser 115

		// bazBrowser Availabilities
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "bazBrowser", BrowserVersion: "16.4"},
			FeatureKey:                 "FeatureX",
		}, // Available from bazBrowser 16.4
		{
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "bazBrowser", BrowserVersion: "16.5"},
			FeatureKey:                 "FeatureY",
		}, // Available from bazBrowser 16.5
	}
	for _, availability := range browserFeatureAvailabilities {
		err := client.InsertBrowserFeatureAvailability(ctx,
			availability.FeatureKey, availability.BrowserFeatureAvailability)
		if err != nil {
			t.Errorf("unexpected error during insert. %s", err.Error())
		}
	}
	err := spannerClient.PrecalculateBrowserFeatureSupportEvents(ctx,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Errorf("unexpected error during pre-calculate. %s", err.Error())
	}
}

func assertListMissingOneImplCounts(ctx context.Context, t *testing.T, startAt, endAt time.Time, pageToken *string,
	targetBrowser string, otherBrowsers []string, pageSize int, expectedPage *MissingOneImplCountPage) {
	result, err := spannerClient.ListMissingOneImplCounts(
		ctx,
		targetBrowser,
		otherBrowsers,
		startAt,
		endAt,
		pageSize,
		pageToken,
	)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expectedPage, result) {
		t.Errorf("unexpected result.\nExpected %+v\nReceived %+v", expectedPage, result)
	}
}

func TestListMissingOneImplCounts(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	loadDataForListMissingOneImplCounts(ctx, t, spannerClient)
	actualEvents := spannerClient.readAllBrowserFeatureSupportEvents(ctx, t)
	slices.SortFunc(actualEvents, sortBrowserFeatureSupportEvents)
	defaultStartAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	defaultEndAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	defaultPageSize := 100

	t.Run("bazBrowser ", func(t *testing.T) {
		targetBrowser := "bazBrowser"
		otherBrowsers := []string{
			"fooBrowser",
			"barBrowser",
		}

		// nolint:dupl // WONTFIX - false positive
		t.Run("all data in one page", func(t *testing.T) {
			expectedResult := &MissingOneImplCountPage{
				NextPageToken: nil,
				Metrics: []MissingOneImplCount{
					// fooBrowser 113 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// barBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureW, FeatureZ
					{
						EventReleaseDate: time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC),
						Count:            2,
					},
					// barBrowser 115 AND bazBrowser 17 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureZ, FeatureY
					// barBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureZ
					{
						EventReleaseDate: time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
						Count:            1,
					},
					// barBrowser 114 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureZ, FeatureY
					// barBrowser: FeatureX, FeatureZ, FeatureY
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureZ
					{
						EventReleaseDate: time.Date(2024, 3, 28, 0, 0, 0, 0, time.UTC),
						Count:            1,
					},
					// fooBrowser 112 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureY, FeatureZ
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureZ
					{
						EventReleaseDate: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
						Count:            1,
					},
					// bazBrowser 16.5 release
					// Currently supported features:
					// fooBrowser: FeatureX
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: None
					{
						EventReleaseDate: time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
						Count:            0,
					},
					// fooBrowser 111 release
					// Currently supported features:
					// fooBrowser: FeatureX
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: FeatureX
					// Missing in on for bazBrowser: None
					{
						EventReleaseDate: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
						Count:            0,
					},
					// bazBrowser 16.4 release
					// Currently supported features:
					// fooBrowser: None
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: FeatureX
					// Missing in on for bazBrowser: None
					{
						EventReleaseDate: time.Date(2024, 1, 25, 0, 0, 0, 0, time.UTC),
						Count:            0,
					},
					// barBrowser 113 release
					// Currently supported features:
					// fooBrowser: None
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: None
					// Missing in on for bazBrowser: None
					{
						EventReleaseDate: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
						Count:            0,
					},
					// fooBrowser 110 release
					// Currently supported features:
					// fooBrowser: None
					// barBrowser: None
					// bazBrowser: None
					// Missing in on for bazBrowser: None
					{
						EventReleaseDate: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
						Count:            0,
					},
				},
			}
			assertListMissingOneImplCounts(
				ctx,
				t,
				defaultStartAt,
				defaultEndAt,
				nil,
				targetBrowser,
				otherBrowsers,
				defaultPageSize,
				expectedResult,
			)
		})

		t.Run("pagination", func(t *testing.T) {
			// Page One
			pageOneToken := encodeMissingOneImplCursor(time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC))
			expectedPageOne := &MissingOneImplCountPage{
				NextPageToken: &pageOneToken,
				Metrics: []MissingOneImplCount{
					// fooBrowser 113 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// barBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureW, FeatureZ
					{
						EventReleaseDate: time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC),
						Count:            2,
					},
					// barBrowser 115 AND bazBrowser 17 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureZ, FeatureY
					// barBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureZ
					{
						EventReleaseDate: time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
						Count:            1,
					},
					// barBrowser 114 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureZ, FeatureY
					// barBrowser: FeatureX, FeatureZ, FeatureY
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureZ
					{
						EventReleaseDate: time.Date(2024, 3, 28, 0, 0, 0, 0, time.UTC),
						Count:            1,
					},
					// fooBrowser 112 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureY, FeatureZ
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureZ
					{
						EventReleaseDate: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
						Count:            1,
					},
				},
			}

			assertListMissingOneImplCounts(
				ctx,
				t,
				defaultStartAt,
				defaultEndAt,
				nil,
				targetBrowser,
				otherBrowsers,
				4,
				expectedPageOne,
			)

			// Page Two
			pageTwoToken := encodeMissingOneImplCursor(time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC))
			expectedPageTwo := &MissingOneImplCountPage{
				NextPageToken: &pageTwoToken,
				Metrics: []MissingOneImplCount{
					// bazBrowser 16.5 release
					// Currently supported features:
					// fooBrowser: FeatureX
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: None
					{
						EventReleaseDate: time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
						Count:            0,
					},
					// fooBrowser 111 release
					// Currently supported features:
					// fooBrowser: FeatureX
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: FeatureX
					// Missing in on for bazBrowser: None
					{
						EventReleaseDate: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
						Count:            0,
					},
					// bazBrowser 16.4 release
					// Currently supported features:
					// fooBrowser: None
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: FeatureX
					// Missing in on for bazBrowser: None
					{
						EventReleaseDate: time.Date(2024, 1, 25, 0, 0, 0, 0, time.UTC),
						Count:            0,
					},
					// barBrowser 113 release
					// Currently supported features:
					// fooBrowser: None
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: None
					// Missing in on for bazBrowser: None
					{
						EventReleaseDate: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
						Count:            0,
					},
				},
			}
			assertListMissingOneImplCounts(
				ctx,
				t,
				defaultStartAt,
				defaultEndAt,
				&pageOneToken,
				targetBrowser,
				otherBrowsers,
				4,
				expectedPageTwo,
			)

			// Page Three
			expectedPageThree := &MissingOneImplCountPage{
				NextPageToken: nil,
				Metrics: []MissingOneImplCount{
					// fooBrowser 110 release
					// Currently supported features:
					// fooBrowser: None
					// barBrowser: None
					// bazBrowser: None
					// Missing in on for bazBrowser: None
					{
						EventReleaseDate: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
						Count:            0,
					},
				},
			}
			assertListMissingOneImplCounts(
				ctx,
				t,
				defaultStartAt,
				defaultEndAt,
				&pageTwoToken,
				targetBrowser,
				otherBrowsers,
				4,
				expectedPageThree,
			)
		})

		t.Run("should reduce the number of results by constraining startAt and endAt", func(t *testing.T) {
			expectedResult := &MissingOneImplCountPage{
				NextPageToken: nil,
				Metrics: []MissingOneImplCount{
					// barBrowser 114 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureZ, FeatureY
					// barBrowser: FeatureX, FeatureZ, FeatureY
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureZ
					{
						EventReleaseDate: time.Date(2024, 3, 28, 0, 0, 0, 0, time.UTC),
						Count:            1,
					},
					// fooBrowser 112 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureY, FeatureZ
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureZ
					{
						EventReleaseDate: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
						Count:            1,
					},
					// bazBrowser 16.5 release
					// Currently supported features:
					// fooBrowser: FeatureX
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: None
					{
						EventReleaseDate: time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
						Count:            0,
					},
					// fooBrowser 111 release
					// Currently supported features:
					// fooBrowser: FeatureX
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: FeatureX
					// Missing in on for bazBrowser: None
					{
						EventReleaseDate: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
						Count:            0,
					},
				},
			}
			assertListMissingOneImplCounts(
				ctx,
				t,
				time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
				nil,
				targetBrowser,
				otherBrowsers,
				defaultPageSize,
				expectedResult,
			)
		})

		t.Run("should show less data points when looking at a smaller subset of browsers", func(t *testing.T) {
			otherBrowsers := []string{
				"barBrowser",
			}

			expectedResult := &MissingOneImplCountPage{
				NextPageToken: nil,
				Metrics: []MissingOneImplCount{
					// barBrowser 115 AND bazBrowser 17 release
					// Currently supported features:
					// barBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureZ, FeatureW
					{
						EventReleaseDate: time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
						Count:            2,
					},
					// barBrowser 114 release
					// Currently supported features:
					// barBrowser: FeatureX, FeatureZ, FeatureY
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureZ
					{
						EventReleaseDate: time.Date(2024, 3, 28, 0, 0, 0, 0, time.UTC),
						Count:            1,
					},
					// bazBrowser 16.5 release
					// Currently supported features:
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureZ
					{
						EventReleaseDate: time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
						Count:            1,
					},
					// bazBrowser 16.4 release
					// Currently supported features:
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: FeatureX
					// Missing in on for bazBrowser: FeatureZ
					{
						EventReleaseDate: time.Date(2024, 1, 25, 0, 0, 0, 0, time.UTC),
						Count:            1,
					},
					// barBrowser 113 release
					// Currently supported features:
					// barBrowser: FeatureX, FeatureZ
					// bazBrowser: None
					// Missing in on for bazBrowser: FeatureX, FeatureZ
					{
						EventReleaseDate: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
						Count:            2,
					},
				},
			}

			assertListMissingOneImplCounts(
				ctx,
				t,
				defaultStartAt,
				defaultEndAt,
				nil,
				targetBrowser,
				otherBrowsers,
				defaultPageSize,
				expectedResult,
			)
		})
	})

	// Misc tests just to make sure we can get other browser info.
	// nolint:dupl // WONTFIX - false positive
	t.Run("all fooBrowser data", func(t *testing.T) {
		targetBrowser := "fooBrowser"
		otherBrowsers := []string{
			"barBrowser",
			"bazBrowser",
		}

		expectedResult := &MissingOneImplCountPage{
			NextPageToken: nil,
			Metrics: []MissingOneImplCount{
				// fooBrowser 113 release
				// Currently supported features:
				// fooBrowser: FeatureX, FeatureY, FeatureZ
				// barBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
				// bazBrowser: FeatureX, FeatureY
				// Missing in on for fooBrowser: None
				{
					EventReleaseDate: time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC),
					Count:            0,
				},
				// barBrowser 115 AND bazBrowser 17 release
				// Currently supported features:
				// fooBrowser: FeatureX, FeatureY, FeatureZ
				// barBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
				// bazBrowser: FeatureX, FeatureY
				// Missing in on for fooBrowser: None
				{
					EventReleaseDate: time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
					Count:            0,
				},
				// barBrowser 114 release
				// Currently supported features:
				// fooBrowser: FeatureX, FeatureY, FeatureZ
				// barBrowser: FeatureX, FeatureZ, FeatureY
				// bazBrowser: FeatureX, FeatureY
				// Missing in on for fooBrowser: None
				{
					EventReleaseDate: time.Date(2024, 3, 28, 0, 0, 0, 0, time.UTC),
					Count:            0,
				},
				// fooBrowser 112 release
				// Currently supported features:
				// fooBrowser: FeatureX, FeatureY, FeatureZ
				// barBrowser: FeatureX, FeatureZ
				// bazBrowser: FeatureX, FeatureY
				// Missing in on for fooBrowser: None
				{
					EventReleaseDate: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
					Count:            0,
				},
				// bazBrowser 16.5 release
				// Currently supported features:
				// fooBrowser: FeatureX
				// barBrowser: FeatureX, FeatureZ
				// bazBrowser: FeatureX, FeatureY
				// Missing in on for fooBrowser: None
				{
					EventReleaseDate: time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
					Count:            0,
				},
				// fooBrowser 111 release
				// Currently supported features:
				// fooBrowser: FeatureX
				// barBrowser: FeatureX, FeatureZ
				// bazBrowser: FeatureX
				// Missing in on for fooBrowser: None
				{
					EventReleaseDate: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
					Count:            0,
				},
				// bazBrowser 16.4  release
				// Currently supported features:
				// fooBrowser: None
				// barBrowser: FeatureX, FeatureZ
				// bazBrowser: FeatureX
				// Missing in on for fooBrowser: FeatureX
				{
					EventReleaseDate: time.Date(2024, 1, 25, 0, 0, 0, 0, time.UTC),
					Count:            1,
				},
				// barBrowser 113 release
				// Currently supported features:
				// fooBrowser: None
				// barBrowser: FeatureX, FeatureZ
				// bazBrowser: None
				// Missing in on for fooBrowser: None
				{
					EventReleaseDate: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
					Count:            0,
				},
				// fooBrowser 110 release
				// Currently supported features:
				// fooBrowser: None
				// barBrowser: None
				// bazBrowser: None
				// Missing in on for fooBrowser: None
				{
					EventReleaseDate: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
					Count:            0,
				},
			},
		}

		assertListMissingOneImplCounts(
			ctx,
			t,
			defaultStartAt,
			defaultEndAt,
			nil,
			targetBrowser,
			otherBrowsers,
			defaultPageSize,
			expectedResult,
		)
	})
}
