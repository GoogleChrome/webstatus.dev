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
		err := client.InsertBrowserFeatureAvailability(ctx,
			availability.FeatureKey, availability.BrowserFeatureAvailability)
		if err != nil {
			t.Errorf("unexpected error during insert. %s", err.Error())
		}
	}
}

func TestListBrowserFeatureCountMetric(t *testing.T) {
	client := getTestDatabase(t)
	ctx := context.Background()

	loadDataForListBrowserFeatureCountMetric(ctx, t, client)

	// Test 1a. First Page
	browser := "fooBrowser"
	startAt := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
	endAt := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	pageSize := 2

	result, err := client.ListBrowserFeatureCountMetric(ctx, browser, startAt, endAt, pageSize, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expectedResult := &BrowserFeatureCountResultPage{
		NextPageToken: valuePtr(encodeBrowserFeatureCountCursor(time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC), 3)),
		Metrics: []BrowserFeatureCountMetric{
			{
				ReleaseDate:  time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
				FeatureCount: 2,
			},
			{
				ReleaseDate:  time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC),
				FeatureCount: 3,
			},
		},
	}

	if !reflect.DeepEqual(expectedResult, result) {
		t.Errorf("unexpected result.\nExpected %+v\nReceived %+v", expectedResult, result)
	}

	// Test 1b. Second Page
	result, err = client.ListBrowserFeatureCountMetric(ctx, browser, startAt, endAt, pageSize, result.NextPageToken)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expectedResult = &BrowserFeatureCountResultPage{
		NextPageToken: nil,
		Metrics:       nil,
	}

	if !reflect.DeepEqual(expectedResult, result) {
		t.Errorf("unexpected result.\nExpected %+v\nReceived %+v", expectedResult, result)
	}

	// Test 2. Let's try to get the last one and it should get one point but still count all the features beforehand.
	startAt = time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	endAt = time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)

	result, err = client.ListBrowserFeatureCountMetric(ctx, browser, startAt, endAt, 100, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expectedResult = &BrowserFeatureCountResultPage{
		NextPageToken: nil,
		Metrics: []BrowserFeatureCountMetric{
			{
				ReleaseDate:  time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC),
				FeatureCount: 3,
			},
		},
	}

	if !reflect.DeepEqual(expectedResult, result) {
		t.Errorf("unexpected result.\nExpected %+v\nReceived %+v", expectedResult, result)
	}

}
