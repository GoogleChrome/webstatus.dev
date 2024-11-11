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
	"log/slog"
	"time"
)

const browserFeatureSupportEventsTable = "BrowserFeatureSupportEvents"

type BrowserFeatureSupportStatus string

const (
	UnsupportedFeatureSupport BrowserFeatureSupportStatus = "unsupported"
	SupportedFeatureSupport   BrowserFeatureSupportStatus = "supported"
)

type BrowserFeatureSupportEvent struct {
	TargetBrowserName string                      `spanner:"TargetBrowserName"`
	EventBrowserName  string                      `spanner:"EventBrowserName"`
	EventReleaseDate  time.Time                   `spanner:"EventReleaseDate"`
	WebFeatureID      string                      `spanner:"WebFeatureID"`
	SupportStatus     BrowserFeatureSupportStatus `spanner:"SupportStatus"`
}

func buildAvailabilityMap(
	releases []spannerBrowserRelease,
	availabilities []spannerBrowserFeatureAvailability) map[string]map[string]time.Time {
	// Create a map for efficient lookup of browser releases
	releaseMap := make(map[string]map[string]time.Time) // map[browserName]map[browserVersion]releaseDate
	for _, release := range releases {
		if _, ok := releaseMap[release.BrowserName]; !ok {
			releaseMap[release.BrowserName] = make(map[string]time.Time)
		}
		releaseMap[release.BrowserName][release.BrowserVersion] = release.ReleaseDate
	}

	// Create a map for efficient lookup of feature availability with release dates
	availabilityMap := make(map[string]map[string]time.Time) // map[browserName]map[featureID]time.Time
	for _, availability := range availabilities {
		if _, ok := availabilityMap[availability.BrowserName]; !ok {
			availabilityMap[availability.BrowserName] = make(map[string]time.Time)
		}
		// Use releaseMap to get the release date for this availability
		if releaseDate, ok := releaseMap[availability.BrowserName][availability.BrowserVersion]; ok {
			availabilityMap[availability.BrowserName][availability.WebFeatureID] = releaseDate
		}
	}

	return availabilityMap
}

func calculateBrowserSupportEvents(
	availabilityMap map[string]map[string]time.Time,
	releases []spannerBrowserRelease,
	ids []string,
	eventChan chan<- BrowserFeatureSupportEvent,
) {
	for targetBrowserName := range availabilityMap {
		for _, eventBrowser := range releases {
			for _, id := range ids {
				supportStatus := UnsupportedFeatureSupport // Default to unsupported
				if _, ok := availabilityMap[targetBrowserName]; ok {
					availabilityTime, supported := availabilityMap[targetBrowserName][id]
					if supported && (availabilityTime.Equal(eventBrowser.ReleaseDate) ||
						eventBrowser.ReleaseDate.After(availabilityTime)) {
						supportStatus = SupportedFeatureSupport
					}
				}
				eventChan <- BrowserFeatureSupportEvent{
					TargetBrowserName: targetBrowserName,
					EventBrowserName:  eventBrowser.BrowserName,
					EventReleaseDate:  eventBrowser.ReleaseDate,
					WebFeatureID:      id,
					SupportStatus:     supportStatus,
				}
			}
		}
	}
}

// PrecalculateBrowserFeatureSupportEvents populates the BrowserFeatureSupportEvents table with pre-calculated data.
func (c *Client) PrecalculateBrowserFeatureSupportEvents(ctx context.Context) error {
	txn := c.Client.ReadOnlyTransaction()
	// 1. Fetch all BrowserFeatureAvailabilities
	availabilities, err := c.fetchAllBrowserAvailabilitiesWithTransaction(ctx, txn)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "availabilities fetched", "size", len(availabilities))

	// 2. Fetch all BrowserReleases
	releases, err := c.fetchAllBrowserReleasesWithTransaction(ctx, txn)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "releases fetched", "size", len(releases))

	// 3. Fetch all WebFeatures
	ids, err := c.fetchAllWebFeatureIDsWithTransaction(ctx, txn)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "feature ids fetched", "size", len(ids))

	// 4. Create maps for quick look ups
	availabilityMap := buildAvailabilityMap(releases, availabilities)

	// 5. Generate BrowserFeatureSupportEvents entries (including SupportStatus)
	return runConcurrentBatch[BrowserFeatureSupportEvent](ctx,
		c, func(entityChan chan<- BrowserFeatureSupportEvent) {
			calculateBrowserSupportEvents(availabilityMap, releases, ids, entityChan)
		}, browserFeatureSupportEventsTable)
}
