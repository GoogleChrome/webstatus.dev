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

	"github.com/stretchr/testify/assert"
)

// nolint:dupl // WONTFIX
func loadDataForListMissingOneImplFeatureList(ctx context.Context, t *testing.T, client *Client) {
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

// nolint:unparam // WONTFIX
func assertMissingOneImplFeatureList(ctx context.Context, t *testing.T, targetDate time.Time,
	targetBrowser string, otherBrowsers []string, expectedPage *MissingOneImplFeatureListPage, token *string,
	pageSize int) {
	result, err := spannerClient.ListMissingOneImplementationFeatures(
		ctx,
		targetBrowser,
		otherBrowsers,
		targetDate,
		pageSize,
		token,
	)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expectedPage.NextPageToken, result.NextPageToken) {
		t.Errorf("unexpected result.\nExpected %+v\nReceived %+v", expectedPage, result)
	}
	if !assert.ElementsMatch(t, expectedPage.FeatureList, result.FeatureList) {
		t.Errorf("unexpected result.\nExpected %+v\nReceived %+v", expectedPage, result)
	}
}

func testMissingOneImplFeatureListSuite(
	ctx context.Context,
	t *testing.T,
) {
	t.Run("Query bazBrowser without exclusions", func(t *testing.T) {
		const targetBrowser = "bazBrowser"
		otherBrowsers := []string{
			"fooBrowser",
			"barBrowser",
		}
		targetDate := time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC)
		pageSize := 25
		token := encodeMissingOneImplFeatureListCursor(0)

		t.Run("simple successful query", func(t *testing.T) {
			expectedResult := &MissingOneImplFeatureListPage{
				NextPageToken: nil,
				FeatureList: []MissingOneImplFeature{
					// fooBrowser 113 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// barBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureW, FeatureZ
					{
						WebFeatureID: "FeatureZ",
					},
					{
						WebFeatureID: "FeatureW",
					},
				},
			}
			assertMissingOneImplFeatureList(
				ctx,
				t,
				targetDate,
				targetBrowser,
				otherBrowsers,
				expectedResult,
				&token,
				pageSize,
			)
		})

		t.Run("empty query result", func(t *testing.T) {
			emptyDate := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)
			expectedResult := &MissingOneImplFeatureListPage{
				NextPageToken: nil,
				FeatureList:   []MissingOneImplFeature{},
			}
			assertMissingOneImplFeatureList(
				ctx,
				t,
				emptyDate,
				targetBrowser,
				otherBrowsers,
				expectedResult,
				&token,
				pageSize,
			)
		})

		t.Run("simple query at a smaller subset of otherBrowsers", func(t *testing.T) {
			subsetBrowsers := []string{
				"barBrowser",
			}

			expectedResult := &MissingOneImplFeatureListPage{
				NextPageToken: nil,
				FeatureList: []MissingOneImplFeature{
					// fooBrowser 113 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// barBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureW, FeatureZ
					{
						WebFeatureID: "FeatureZ",
					},
					{
						WebFeatureID: "FeatureW",
					},
				},
			}
			assertMissingOneImplFeatureList(
				ctx,
				t,
				targetDate,
				targetBrowser,
				subsetBrowsers,
				expectedResult,
				&token,
				pageSize,
			)
		})

		t.Run("simple successful query with pagination", func(t *testing.T) {
			pageToken := encodeMissingOneImplFeatureListCursor(1)
			expectedResult := &MissingOneImplFeatureListPage{
				NextPageToken: nil,
				FeatureList: []MissingOneImplFeature{
					// fooBrowser 113 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// barBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureW, FeatureZ
					{
						WebFeatureID: "FeatureZ",
					},
				},
			}
			assertMissingOneImplFeatureList(
				ctx,
				t,
				targetDate,
				targetBrowser,
				otherBrowsers,
				expectedResult,
				&pageToken,
				pageSize,
			)
		})

		t.Run("Return a page token with page size 1", func(t *testing.T) {
			onePerPage := 1
			returnToken := encodeMissingOneImplFeatureListCursor(1)
			expectedResult := &MissingOneImplFeatureListPage{
				NextPageToken: &returnToken,
				FeatureList: []MissingOneImplFeature{
					// fooBrowser 113 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// barBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureW, FeatureZ
					{
						WebFeatureID: "FeatureW",
					},
				},
			}
			assertMissingOneImplFeatureList(
				ctx,
				t,
				targetDate,
				targetBrowser,
				otherBrowsers,
				expectedResult,
				&token,
				onePerPage,
			)

			pageTwoToken := encodeMissingOneImplFeatureListCursor(2)
			expectedResultPageTwo := &MissingOneImplFeatureListPage{
				NextPageToken: &pageTwoToken,
				FeatureList: []MissingOneImplFeature{
					// fooBrowser 113 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// barBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureW, FeatureZ
					{
						WebFeatureID: "FeatureZ",
					},
				},
			}
			assertMissingOneImplFeatureList(
				ctx,
				t,
				targetDate,
				targetBrowser,
				otherBrowsers,
				expectedResultPageTwo,
				&returnToken,
				onePerPage,
			)
		})

		t.Run("simple query without a token", func(t *testing.T) {
			expectedResult := &MissingOneImplFeatureListPage{
				NextPageToken: nil,
				FeatureList: []MissingOneImplFeature{
					// fooBrowser 113 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// barBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureW, FeatureZ
					{
						WebFeatureID: "FeatureZ",
					},
					{
						WebFeatureID: "FeatureW",
					},
				},
			}
			assertMissingOneImplFeatureList(
				ctx,
				t,
				targetDate,
				targetBrowser,
				otherBrowsers,
				expectedResult,
				nil,
				pageSize,
			)
		})
	})

	t.Run("with excluded/discouraged features", func(t *testing.T) {
		// Exclude Feature X
		excludedFeatures := []string{"FeatureX"}
		for _, featureKey := range excludedFeatures {
			err := spannerClient.InsertExcludedFeatureKey(ctx, featureKey)
			if err != nil {
				t.Fatalf("Failed to insert excluded feature key: %v", err)
			}
		}

		// Discourage FeatureZ
		discouragedFeatures := []string{"FeatureZ"}
		for _, featureKey := range discouragedFeatures {
			err := spannerClient.UpsertFeatureDiscouragedDetails(ctx, featureKey, FeatureDiscouragedDetails{
				AccordingTo:  nil,
				Alternatives: nil,
			})
			if err != nil {
				t.Fatalf("Failed to upsert feature discouraged details: %v", err)
			}
		}

		// nolint:goconst // WONTFIX
		t.Run("simple query", func(t *testing.T) {
			targetBrowser := "bazBrowser"
			otherBrowsers := []string{"fooBrowser", "barBrowser"}
			targetDate := time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC)
			pageSize := 25
			token := encodeMissingOneImplFeatureListCursor(0)

			expectedResult := &MissingOneImplFeatureListPage{
				NextPageToken: nil,
				FeatureList: []MissingOneImplFeature{
					// fooBrowser 113 release
					// Currently supported features:
					// fooBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// barBrowser: FeatureX, FeatureZ, FeatureY, FeatureW
					// bazBrowser: FeatureX, FeatureY
					// Missing in on for bazBrowser: FeatureW (FeatureZ is excluded/discouraged)
					{
						WebFeatureID: "FeatureW",
					},
				},
			}
			// Assert with excluded/discouraged features
			assertMissingOneImplFeatureList(
				ctx,
				t,
				targetDate,
				targetBrowser,
				otherBrowsers,
				expectedResult,
				&token,
				pageSize,
			)
		})

		// Clear the excluded and discouraged features after the test
		err := spannerClient.ClearExcludedFeatureKeys(ctx)
		if err != nil {
			t.Fatalf("Failed to clear excluded feature keys: %v", err)
		}

		err = spannerClient.ClearFeatureDiscouragedDetails(ctx)
		if err != nil {
			t.Fatalf("Failed to clear feature discouraged details: %v", err)
		}
	})
}

func TestListMissingOneImplFeatureList(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	loadDataForListMissingOneImplFeatureList(ctx, t, spannerClient)
	actualEvents := spannerClient.readAllBrowserFeatureSupportEvents(ctx, t)
	slices.SortFunc(actualEvents, sortBrowserFeatureSupportEvents)
	t.Run("MissingOneImplFeatureListQuery", func(t *testing.T) {
		testMissingOneImplFeatureListSuite(ctx, t)
	})
}
