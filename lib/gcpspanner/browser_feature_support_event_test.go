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

	"cloud.google.com/go/spanner"
)

func setupTablesForPrecalculateBrowserFeatureSupportEvents(
	ctx context.Context,
	t *testing.T,
) ([]WebFeature, map[string]string) {
	featureKeyToID := map[string]string{}

	// 1. Insert sample data into WebFeatures
	features := []WebFeature{
		{FeatureKey: "FeatureX", Name: "Cool API"},
		{FeatureKey: "FeatureY", Name: "Super API"},
		{FeatureKey: "FeatureZ", Name: "Ultra API"},
	}
	for _, feature := range features {
		id, err := spannerClient.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Fatalf("Failed to insert WebFeature: %v", err)
		}
		featureKeyToID[feature.FeatureKey] = *id
	}

	// 2. Insert sample data into BrowserReleases
	releases := []BrowserRelease{
		{BrowserName: "Chrome", BrowserVersion: "110", ReleaseDate: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "Chrome", BrowserVersion: "111", ReleaseDate: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "Firefox", BrowserVersion: "111", ReleaseDate: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)},
		{BrowserName: "Firefox", BrowserVersion: "112", ReleaseDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, release := range releases {
		err := spannerClient.InsertBrowserRelease(ctx, release)
		if err != nil {
			t.Fatalf("Failed to insert BrowserRelease: %v", err)
		}
	}

	// 3. Insert sample data into BrowserFeatureAvailabilities
	availabilities := []struct {
		WebFeatureKey string
		BrowserFeatureAvailability
	}{
		{
			WebFeatureKey:              features[0].FeatureKey,
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "Chrome", BrowserVersion: "110"},
		},
		{
			WebFeatureKey:              features[2].FeatureKey,
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "Chrome", BrowserVersion: "111"},
		},
		{
			WebFeatureKey:              features[1].FeatureKey,
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "Firefox", BrowserVersion: "111"},
		},
		{
			WebFeatureKey:              features[2].FeatureKey,
			BrowserFeatureAvailability: BrowserFeatureAvailability{BrowserName: "Firefox", BrowserVersion: "112"},
		},
	}
	for _, availability := range availabilities {
		err := spannerClient.UpsertBrowserFeatureAvailability(ctx, availability.WebFeatureKey,
			availability.BrowserFeatureAvailability)
		if err != nil {
			t.Fatalf("Failed to insert BrowserFeatureAvailability: %v", err)
		}
	}

	return features, featureKeyToID
}

func TestPrecalculateBrowserFeatureSupportEvents(t *testing.T) {
	t.Run("all data", func(t *testing.T) {
		restartDatabaseContainer(t)
		ctx := context.Background()
		features, featureKeyToID := setupTablesForPrecalculateBrowserFeatureSupportEvents(ctx, t)

		// 4. Call the function to pre-calculate the data
		err := spannerClient.PrecalculateBrowserFeatureSupportEvents(
			ctx,
			time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC),
		)
		if err != nil {
			t.Fatalf("PrecalculateBrowserFeatureSupportEvents failed: %v", err)
		}

		// 5. Assert the expected data in BrowserFeatureSupportEvents
		expectedEvents := []BrowserFeatureSupportEvent{
			/*
				2024-01-10 - Chrome release
			*/
			// Chrome supports features[0] during it's own release which has features[0]
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[0].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Chrome never supports features[1]
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[1].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
			// Chrome does not support features[2] yet
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[2].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
			// Firefox never supports features[0]
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[0].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
			// Firefox should not support features[1] during the release of Chrome because
			// Firefox doesn't support features[1] until its release later.
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[1].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
			// Firefox does not support features[2] yet
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[2].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
			/*
				2024-02-01 - Firefox and Chrome release
			*/
			// Firefox release
			// Chrome already supports features[0].
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[0].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Chrome never supports features[1]
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[1].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
			// Chrome now supports features[2].
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[2].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Firefox never supports features[0]
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[0].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
			// Firefox supports features[1] during it's own release which has features[1]
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[1].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Firefox does not support features[2] yet
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[2].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},

			// Chrome release
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[0].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Chrome never supports features[1]
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[1].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
			// Chrome now supports features[2].
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[2].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Firefox never supports features[0]
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[0].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
			// Firefox supports features[1] during it's own release which has features[1]
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[1].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Firefox does not support features[2] yet
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[2].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},

			/*
				2024-03-01 - Firefox release
			*/
			// Chrome already supports features[0].
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[0].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Chrome never supports features[1]
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[1].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
			// Chrome now supports features[2].
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[2].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Firefox never supports features[0]
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[0].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
			// Firefox supports features[1] during it's own release which has features[1]
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[1].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Firefox supports features[2] now
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[2].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
		}
		assertBrowserFeatureSupportEvents(ctx, t, expectedEvents)
	})

	t.Run("narrow time window of data", func(t *testing.T) {
		restartDatabaseContainer(t)
		ctx := context.Background()
		features, featureKeyToID := setupTablesForPrecalculateBrowserFeatureSupportEvents(ctx, t)

		// 4. Call the function to pre-calculate the data
		err := spannerClient.PrecalculateBrowserFeatureSupportEvents(
			ctx,
			time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 2, 25, 0, 0, 0, 0, time.UTC),
		)
		if err != nil {
			t.Fatalf("PrecalculateBrowserFeatureSupportEvents failed: %v", err)
		}

		// 5. Assert the expected data in BrowserFeatureSupportEvents
		expectedEvents := []BrowserFeatureSupportEvent{
			/*
				2024-02-01 - Firefox and Chrome release
			*/
			// Firefox release
			// Chrome already supports features[0].
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[0].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Chrome never supports features[1]
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[1].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
			// Chrome now supports features[2].
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[2].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Firefox never supports features[0]
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[0].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
			// Firefox supports features[1] during it's own release which has features[1]
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[1].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Firefox does not support features[2] yet
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Firefox",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[2].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},

			// Chrome release
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[0].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Chrome never supports features[1]
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[1].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
			// Chrome now supports features[2].
			{
				TargetBrowserName: "Chrome",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[2].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Firefox never supports features[0]
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[0].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
			// Firefox supports features[1] during it's own release which has features[1]
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[1].FeatureKey],
				SupportStatus:     SupportedFeatureSupport,
			},
			// Firefox does not support features[2] yet
			{
				TargetBrowserName: "Firefox",
				EventBrowserName:  "Chrome",
				EventReleaseDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				WebFeatureID:      featureKeyToID[features[2].FeatureKey],
				SupportStatus:     UnsupportedFeatureSupport,
			},
		}
		assertBrowserFeatureSupportEvents(ctx, t, expectedEvents)
	})
}

func assertBrowserFeatureSupportEvents(ctx context.Context, t *testing.T, expectedEvents []BrowserFeatureSupportEvent) {
	actualEvents := spannerClient.readAllBrowserFeatureSupportEvents(ctx, t)

	// Assert that the actual events match the expected events
	slices.SortFunc(expectedEvents, sortBrowserFeatureSupportEvents)
	slices.SortFunc(actualEvents, sortBrowserFeatureSupportEvents)
	if !reflect.DeepEqual(expectedEvents, actualEvents) {
		t.Errorf("Unexpected data in BrowserFeatureSupportEvents\nExpected (size: %d):\n%+v\nActual (size: %d):\n%+v",
			len(expectedEvents), expectedEvents, len(actualEvents), actualEvents)
	}
}

func (c *Client) readAllBrowserFeatureSupportEvents(ctx context.Context, t *testing.T) []BrowserFeatureSupportEvent {
	// Fetch all rows from BrowserFeatureSupportEvents
	stmt := spanner.Statement{
		SQL: `SELECT *
              FROM BrowserFeatureSupportEvents`,
		Params: nil,
	}
	var actualEvents []BrowserFeatureSupportEvent
	iter := spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	err := iter.Do(func(row *spanner.Row) error {
		var event BrowserFeatureSupportEvent
		if err := row.ToStruct(&event); err != nil {
			return err
		}
		actualEvents = append(actualEvents, event)

		return nil
	})
	if err != nil {
		t.Fatalf("Failed to fetch data from BrowserFeatureSupportEvents: %v", err)
	}

	return actualEvents
}

func sortBrowserFeatureSupportEvents(left, right BrowserFeatureSupportEvent) int {
	// 1. Sort by EventReleaseDate
	if !left.EventReleaseDate.Equal(right.EventReleaseDate) {
		if left.EventReleaseDate.Before(right.EventReleaseDate) {
			return -1
		}

		return 1
	}

	// 2. Sort by EventBrowserName
	if left.EventBrowserName != right.EventBrowserName {
		if left.EventBrowserName < right.EventBrowserName {
			return -1
		}

		return 1
	}

	// 3. Sort by TargetBrowserName
	if left.TargetBrowserName != right.TargetBrowserName {
		if left.TargetBrowserName < right.TargetBrowserName {
			return -1
		}

		return 1
	}

	// 4. Sort by WebFeatureID
	if left.WebFeatureID < right.WebFeatureID {
		return -1
	} else if left.WebFeatureID > right.WebFeatureID {
		return 1
	}

	return 0 // Equal
}
