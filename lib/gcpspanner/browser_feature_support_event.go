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
	"sync"
	"time"

	"cloud.google.com/go/spanner"
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

func calculateBrowserSupportEventsAndSend(
	availabilityMap map[string]map[string]time.Time,
	releases []spannerBrowserRelease,
	ids []string,
	eventChan chan<- BrowserFeatureSupportEvent,
	startAtFilter *time.Time,
	endAtFilter *time.Time,
) {
	count := 0
	for _, targetBrowser := range releases {
		for _, eventBrowser := range releases {
			for _, id := range ids {
				supportStatus := UnsupportedFeatureSupport // Default to unsupported
				if _, ok := availabilityMap[targetBrowser.BrowserName]; ok {
					availabilityTime, supported := availabilityMap[targetBrowser.BrowserName][id]
					if startAtFilter != nil && availabilityTime.Before(*startAtFilter) {
						continue
					}
					if endAtFilter != nil && availabilityTime.After(*endAtFilter) {
						continue
					}
					if supported && (availabilityTime.Equal(eventBrowser.ReleaseDate) ||
						eventBrowser.ReleaseDate.After(availabilityTime)) {
						supportStatus = SupportedFeatureSupport
					}
				}
				count++
				eventChan <- BrowserFeatureSupportEvent{
					TargetBrowserName: targetBrowser.BrowserName,
					EventBrowserName:  eventBrowser.BrowserName,
					EventReleaseDate:  eventBrowser.ReleaseDate,
					WebFeatureID:      id,
					SupportStatus:     supportStatus,
				}
			}
		}
	}
	slog.Info("finished sending", "count", count)
	close(eventChan)
}

func (c *Client) batchWriteBrowserFeatureSupportEvents(
	ctx context.Context, wg *sync.WaitGroup, batchSize int,
	eventChan <-chan BrowserFeatureSupportEvent, errChan chan error, workerID int) {
	defer func() {
		wg.Done()
		slog.InfoContext(ctx, "worker finishing", "id", workerID)
	}()
	slog.InfoContext(ctx, "worker starting", "id", workerID)
	for {
		batch := make([]*spanner.Mutation, 0, batchSize)
		for i := 0; i < batchSize; i++ {
			select {
			case event, isChannelStillOpen := <-eventChan:
				// If the channel is closed, go ahead and apply what we have and return.
				if !isChannelStillOpen {
					if len(batch) > 0 {
						slog.InfoContext(ctx, "sending final batch", "size", len(batch), "id", workerID)
						err := c.BatchWriteMutations(ctx, c.Client, batch)
						if err != nil {
							errChan <- err
						}
					}

					return
				}
				// Else, the channel is still open and it has received a value.
				// Create a mutation and append it to the upcoming batch
				m, err := spanner.InsertOrUpdateStruct(browserFeatureSupportEventsTable, event)
				if err != nil {
					errChan <- err

					return
				}
				batch = append(batch, m)
			case <-ctx.Done():
				// If the system tells us that we are done, we can abort too.
				return
			}
		}
		// The current batch is full. Send the mutations to the database.
		slog.InfoContext(ctx, "sending batch", "size", len(batch), "id", workerID)
		err := c.BatchWriteMutations(ctx, c.Client, batch)
		if err != nil {
			errChan <- err

			return
		}
	}
}

type PrecalculateBrowserFeatureSupportEventFilter struct {
	StartAt *time.Time
	EndAt   *time.Time
}

// PrecalculateBrowserFeatureSupportEvents populates the BrowserFeatureSupportEvents table with pre-calculated data.
func (c *Client) PrecalculateBrowserFeatureSupportEvents(ctx context.Context,
	filter PrecalculateBrowserFeatureSupportEventFilter) error {
	const batchSize = 10000
	eventChan := make(chan BrowserFeatureSupportEvent, batchSize)
	errChan := make(chan error)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	workers := 8
	wg.Add(workers)
	doneChan := make(chan struct{})
	go func() {
		slog.InfoContext(ctx, "waiting for wait group to finish")
		wg.Wait()
		slog.InfoContext(ctx, "wait group to finished")
		close(doneChan)
	}()
	for i := 0; i < workers; i++ {
		go c.batchWriteBrowserFeatureSupportEvents(ctx, &wg, batchSize, eventChan, errChan, i)
	}
	slog.InfoContext(ctx, "About to pre-calculate")
	txn := c.Client.ReadOnlyTransaction()
	// 1. Fetch all BrowserFeatureAvailabilities
	availabilities, err := c.fetchAllBrowserAvailabilitiesWithTransaction(ctx, txn)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "Availabilities fetched")

	// 2. Fetch all BrowserReleases
	releases, err := c.fetchAllBrowserReleasesWithTransaction(ctx, txn)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "releases fetched")

	// 3. Fetch all WebFeatures
	ids, err := c.fetchAllWebFeatureIDsWithTransaction(ctx, txn)
	if err != nil {
		return err
	}

	// 4. Create maps for quick look ups
	availabilityMap := buildAvailabilityMap(releases, availabilities)

	// 4. Generate BrowserFeatureSupportEvents entries (including SupportStatus)
	calculateBrowserSupportEventsAndSend(availabilityMap, releases, ids, eventChan, filter.StartAt, filter.EndAt)

	slog.InfoContext(ctx, "support events sent")

	// 4. Check for errors from the goroutine
	select {
	case err := <-errChan:
		cancel()
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-doneChan:
		return nil
	}

}
