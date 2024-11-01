package gcpspanner

import (
	"context"
	"errors"
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

// PrecalculateBrowserFeatureSupportEvents populates the BrowserFeatureSupportEvents table with pre-calculated data.
func (c *Client) PrecalculateBrowserFeatureSupportEvents(ctx context.Context) error {
	_, err := c.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Fetch all BrowserFeatureAvailabilities
		var availabilities []spannerBrowserFeatureAvailability
		iter := txn.Read(ctx, browserFeatureAvailabilitiesTable, spanner.AllKeys(), []string{
			"BrowserName",
			"BrowserVersion",
			"WebFeatureID",
		})
		defer iter.Stop()
		err := iter.Do(func(row *spanner.Row) error {
			var entry spannerBrowserFeatureAvailability
			if err := row.ToStruct(&entry); err != nil {
				return err
			}
			availabilities = append(availabilities, entry)
			return nil
		})
		if err != nil {
			return err
		}

		// // 1b. Create a map for efficient lookup of feature availability
		// availabilityMap := make(map[string]map[string]bool) // map[browserName]map[featureID]bool
		// for _, availability := range availabilities {
		// 	if _, ok := availabilityMap[availability.BrowserName]; !ok {
		// 		availabilityMap[availability.BrowserName] = make(map[string]bool)
		// 	}
		// 	availabilityMap[availability.BrowserName][availability.WebFeatureID] = true
		// }

		// 2. Fetch all BrowserReleases
		var releases []spannerBrowserRelease
		iter = txn.Read(ctx, browserReleasesTable, spanner.AllKeys(), []string{
			"BrowserName",
			"BrowserVersion",
			"ReleaseDate",
		})
		defer iter.Stop()
		err = iter.Do(func(row *spanner.Row) error {
			var entry spannerBrowserRelease
			if err := row.ToStruct(&entry); err != nil {
				return err
			}
			releases = append(releases, entry)
			return nil
		})
		if err != nil {
			return err
		}

		// 3. Fetch all WebFeatures
		var features []SpannerWebFeature
		iter = txn.Read(ctx, webFeaturesTable, spanner.AllKeys(), []string{"ID"})
		defer iter.Stop()
		err = iter.Do(func(row *spanner.Row) error {
			var entry SpannerWebFeature
			if err := row.ToStruct(&entry); err != nil {
				return err
			}
			features = append(features, entry)
			return nil
		})
		if err != nil {
			return err
		}

		// 4. Create maps for quick look ups
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

		// 4. Generate BrowserFeatureSupportEvents entries (including SupportStatus)
		var supportEvents []*BrowserFeatureSupportEvent
		for _, targetBrowser := range releases {
			for _, eventBrowser := range releases {
				for _, feature := range features {
					supportStatus := UnsupportedFeatureSupport // Default to unsupported
					if _, ok := availabilityMap[targetBrowser.BrowserName]; ok {
						if availabilityTime, supported := availabilityMap[targetBrowser.BrowserName][feature.ID]; supported && (availabilityTime.Equal(eventBrowser.ReleaseDate) || eventBrowser.ReleaseDate.After(availabilityTime)) {
							supportStatus = SupportedFeatureSupport
						}
					}
					supportEvents = append(supportEvents, &BrowserFeatureSupportEvent{
						TargetBrowserName: targetBrowser.BrowserName,
						EventBrowserName:  eventBrowser.BrowserName,
						EventReleaseDate:  eventBrowser.ReleaseDate,
						WebFeatureID:      feature.ID,
						SupportStatus:     supportStatus,
					})
				}
			}
		}

		// 5. Insert the new entries into BrowserFeatureSupportEvents
		var mutations []*spanner.Mutation
		for _, entry := range supportEvents {
			m, err := spanner.InsertOrUpdateStruct(browserFeatureSupportEventsTable, entry)
			if err != nil {
				return errors.Join(err, ErrInternalQueryFailure)
			}
			mutations = append(mutations, m)
		}
		return txn.BufferWrite(mutations)

	})
	return err
}
