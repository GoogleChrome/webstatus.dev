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

package main

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"math/big"
	"math/rand"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters"
	"github.com/GoogleChrome/webstatus.dev/lib/gds"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features_mappings"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
	"github.com/GoogleChrome/webstatus.dev/lib/webfeaturesmappingtypes"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/web-platform-tests/wpt.fyi/shared"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const releasesPerBrowser = 50
const runsPerBrowserPerChannel = 100
const numberOfFeatures = 80

// Feature Key used for feature page tests.
const featurePageFeatureKey = "anchor-positioning"

const discouragedFeatureKey = "discouraged"

// Allows us to regenerate the same values between runs.
const seedValue = 1024

// nolint: gochecknoglobals
var (
	// nolint: gosec // not using the random source for security.
	r               = rand.New(rand.NewSource(seedValue))
	startTimeWindow = time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	browsers        = []string{
		string(backend.Chrome),
		string(backend.Firefox),
		string(backend.Edge),
		string(backend.Safari),
		string(backend.ChromeAndroid),
		string(backend.FirefoxAndroid),
		string(backend.SafariIos),
	}
)

// List of test user emails whose data should be reset.
// nolint: gochecknoglobals
var testUserEmails = []string{
	"test.user.1@example.com",
	"test.user.2@example.com",
	"test.user.3@example.com",
	"fresh.user@example.com",
	"chromium.user@example.com",
	"firefox.user@example.com",
	"webkit.user@example.com",
}

func getSpecialFeatureDetails() []struct {
	name string
	id   string
} {
	return []struct {
		name string
		id   string
	}{
		// Top HTML Interop Issues
		// Copied from TOP_HTML_INTEROP_ISSUES in frontend/src/static/js/utils/constants.ts
		{
			name: "Customizable Select",
			id:   "customizable-select",
		},
		{
			name: "Popover",
			id:   "popover",
		},
		{
			// Used for feature detail page tests.
			name: "Anchor Positioning",
			id:   featurePageFeatureKey,
		},
		{
			name: "Customized Built-in Elements",
			id:   "customized-built-in-elements",
		},
		{
			name: "Shadow DOM",
			id:   "shadow-dom",
		},
		{
			name: "Dialog",
			id:   "dialog",
		},
		{
			name: "View Transitions",
			id:   "view-transitions",
		},
		{
			name: "Cross Document View Transitions",
			id:   "cross-document-view-transitions",
		},
		{
			name: "File System Access",
			id:   "file-system-access",
		},
		{
			name: "Input Date Time",
			id:   "input-date-time",
		},
		{
			name: "Invoker Commands",
			id:   "invoker-commands",
		},
		{
			name: "WebUSB",
			id:   "webusb",
		},
		// Feature evolution
		// old-feature will be moved to new-feature
		{
			name: "Old Feature",
			id:   "old-feature",
		},
		// before-split-feature will be split into after-split-feature-1 and after-split-feature-2
		{
			name: "Before Split Feature",
			id:   "before-split-feature",
		},
		{
			name: "Discouraged Feature",
			id:   discouragedFeatureKey,
		},
	}
}

func getMovedFeaturesDetails() []gcpspanner.MovedWebFeature {
	return []gcpspanner.MovedWebFeature{
		{
			OriginalFeatureKey: "old-feature",
			NewFeatureKey:      "new-feature",
		},
	}
}

func getSplitFeatureDetails() []gcpspanner.SplitWebFeature {
	return []gcpspanner.SplitWebFeature{
		{
			OriginalFeatureKey: "before-split-feature",
			TargetFeatureKeys: []string{
				"after-split-feature-1",
				"after-split-feature-2",
			},
		},
	}
}

func getWebFeaturesToRemoveDuringEvolution() []string {
	return []string{
		"old-feature",
		"before-split-feature",
	}
}

func getAdditionalWebFeaturesDuringEvolution() []gcpspanner.WebFeature {

	return []gcpspanner.WebFeature{
		{
			Name:            "New Feature",
			FeatureKey:      "new-feature",
			Description:     "New Feature comes from Old Feature",
			DescriptionHTML: "New Feature comes from <b>Old Feature</b>",
		},
		{
			Name:            "After Split Feature 1",
			FeatureKey:      "after-split-feature-1",
			Description:     "This feature was split from before-split-feature",
			DescriptionHTML: "This feature was split from <b>before-split-feature</b>",
		},
		{
			Name:            "After Split Feature 2",
			FeatureKey:      "after-split-feature-2",
			Description:     "This feature was split from before-split-feature",
			DescriptionHTML: "This feature was split from <b>before-split-feature</b>",
		},
	}
}

type featuresHelper struct {
	features map[string]gcpspanner.WebFeature
}

func (h *featuresHelper) AddFeature(feature gcpspanner.WebFeature) {
	h.features[feature.FeatureKey] = feature
}

func (h *featuresHelper) RemoveFeature(featureKey string) {
	delete(h.features, featureKey)
}

func (h featuresHelper) Features() []gcpspanner.WebFeature {
	features := make([]gcpspanner.WebFeature, 0, len(h.features))
	for _, feature := range h.features {
		features = append(features, feature)
	}
	slices.SortFunc(features, func(a, b gcpspanner.WebFeature) int { return cmp.Compare(a.FeatureKey, b.FeatureKey) })

	return features
}

func resetTestData(ctx context.Context, spannerClient *gcpspanner.Client, authClient *auth.Client) error {
	slog.InfoContext(ctx, "Resetting test user saved searches and bookmarks...")
	userIDs := make([]string, len(testUserEmails))
	for idx, email := range testUserEmails {
		userID, err := findUserIDByEmail(ctx, email, authClient)
		// It's okay if a user doesn't exist yet, just log it
		if err != nil {
			slog.WarnContext(ctx, "Could not find user for reset, skipping", "email", email, "error", err)

			continue
		}
		userIDs[idx] = userID
	}

	if len(userIDs) == 0 {
		slog.InfoContext(ctx, "No test user IDs found to reset data for.")

		return nil
	}

	for _, userID := range userIDs {
		page, err := spannerClient.ListUserSavedSearches(ctx, userID, 1000, nil)
		if err != nil {
			return fmt.Errorf("failed to list test user saved searches: %w", err)
		}
		for _, savedSearch := range page.Searches {
			if savedSearch.Role != nil && *savedSearch.Role == string(gcpspanner.SavedSearchOwner) {
				// Delete the owned saved searches (which will also clear out the saved search bookmarks on cascade)
				err := spannerClient.DeleteUserSavedSearch(ctx, gcpspanner.DeleteUserSavedSearchRequest{
					RequestingUserID: userID,
					SavedSearchID:    savedSearch.ID,
				})
				if err != nil {
					return fmt.Errorf("failed to delete test user saved search: %w", err)
				}
			}
		}
	}
	slog.InfoContext(ctx, "Deleted saved searches for test users", "count", len(userIDs))

	// Reset subscriptions for each test user.
	slog.InfoContext(ctx, "Resetting test user subscriptions...")
	for _, userID := range userIDs {
		// We don't need to handle pagination here, assuming a test user won't have more than 1000 subscriptions.
		req := gcpspanner.ListSavedSearchSubscriptionsRequest{
			UserID:    userID,
			PageToken: nil,
			PageSize:  1000, // A large enough number to get all subscriptions for a test user.
		}
		subscriptions, _, err := spannerClient.ListSavedSearchSubscriptions(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to list subscriptions for user %s: %w", userID, err)
		}

		for _, sub := range subscriptions {
			err := spannerClient.DeleteSavedSearchSubscription(ctx, sub.ID, userID)
			if err != nil {
				// Log the error but continue trying to delete others.
				slog.WarnContext(ctx, "failed to delete subscription, continuing",
					"subscriptionID", sub.ID, "userID", userID, "error", err)
			}
		}
	}
	slog.InfoContext(ctx, "Test user subscriptions reset.")

	slog.InfoContext(ctx, "Test user data reset complete.")

	return nil
}

func generateReleases(ctx context.Context, c *gcpspanner.Client) (int, error) {
	releasesGenerated := 0
	for _, browser := range browsers {
		baseDate := startTimeWindow
		releases := make([]gcpspanner.BrowserRelease, 0, releasesPerBrowser)
		for i := 0; i < releasesPerBrowser; i++ {
			if i > 1 {
				baseDate = releases[i-1].ReleaseDate.AddDate(0, 2, r.Intn(90)) // Add 2 months to ~5 months
			}
			release := gcpspanner.BrowserRelease{
				BrowserName:    browser,
				BrowserVersion: fmt.Sprintf("%d", i+1),
				ReleaseDate:    baseDate.AddDate(0, 0, r.Intn(30)), // Add up to 1 month
			}
			releases = append(releases, release)

			err := c.InsertBrowserRelease(ctx, release)
			if err != nil {
				return releasesGenerated, err
			}
			releasesGenerated++
		}
	}

	return releasesGenerated, nil
}

func generateFeatures(
	ctx context.Context, client *gcpspanner.Client, helper *featuresHelper) (
	[]gcpspanner.SpannerWebFeature, map[string]string, error) {
	features := make([]gcpspanner.SpannerWebFeature, 0, numberOfFeatures)
	featureIDMap := make(map[string]interface{})
	webFeatureKeyToInternalFeatureID := map[string]string{}

	realFeatureDetails := getSpecialFeatureDetails()
	for idx := 0; idx < numberOfFeatures; idx++ {
		word := fmt.Sprintf("%s%d", gofakeit.LoremIpsumWord(), len(featureIDMap))
		featureName := cases.Title(language.English).String(word)
		featureID := strings.ToLower(featureName)
		if idx < len(realFeatureDetails) {
			word = realFeatureDetails[idx].name
			featureID = realFeatureDetails[idx].id
			featureName = realFeatureDetails[idx].name
		}

		// Check if we already generated this ID.
		if _, alreadyUsed := featureIDMap[word]; alreadyUsed {
			continue
		}
		// Add it to the map.
		featureIDMap[word] = nil
		feature := gcpspanner.WebFeature{
			Name:            featureName,
			FeatureKey:      featureID,
			Description:     fmt.Sprintf("description for %s", featureName),
			DescriptionHTML: fmt.Sprintf("description for <b>%s</b>", featureName),
		}
		helper.AddFeature(feature)
	}

	inputFeatures := helper.Features()
	err := client.SyncWebFeatures(ctx, inputFeatures)
	if err != nil {
		return nil, nil, err
	}
	for _, feature := range inputFeatures {
		id, err := client.GetIDFromFeatureKey(ctx, gcpspanner.NewFeatureKeyFilter(feature.FeatureKey))
		if err != nil {
			return nil, nil, err
		}
		webFeatureKeyToInternalFeatureID[feature.FeatureKey] = *id
		features = append(features, gcpspanner.SpannerWebFeature{
			WebFeature: feature,
			ID:         *id,
		})
	}

	return features, webFeatureKeyToInternalFeatureID, nil
}

func generateEvolutionOfFeatures(ctx context.Context, client *gcpspanner.Client, helper *featuresHelper) error {
	featuresToRemove := getWebFeaturesToRemoveDuringEvolution()
	for _, featureToRemove := range featuresToRemove {
		helper.RemoveFeature(featureToRemove)

	}
	featuresToAdd := getAdditionalWebFeaturesDuringEvolution()
	for _, featureToAdd := range featuresToAdd {
		helper.AddFeature(featureToAdd)
	}
	redirectTargets := map[string]string{}
	movedFeatures := getMovedFeaturesDetails()
	for _, movedFeature := range movedFeatures {
		redirectTargets[movedFeature.OriginalFeatureKey] = movedFeature.NewFeatureKey
	}
	slog.InfoContext(ctx, "Performing web feature sync with redirect targets", "target_size", len(redirectTargets))
	err := client.SyncWebFeatures(ctx, helper.Features(), gcpspanner.WithRedirectTargets(redirectTargets))
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "Performing moved web feature sync")
	err = client.SyncMovedWebFeatures(ctx, movedFeatures)
	if err != nil {
		return err
	}

	splitFeatures := getSplitFeatureDetails()
	slog.InfoContext(ctx, "Performing split web feature sync")
	err = client.SyncSplitWebFeatures(ctx, splitFeatures)
	if err != nil {
		return err
	}

	return nil
}

func generateFeatureMetadata(ctx context.Context, client *gds.Client, features []gcpspanner.SpannerWebFeature) error {
	for _, feature := range features {
		err := client.UpsertFeatureMetadata(ctx, gds.FeatureMetadata{
			Description:  "Test description for " + feature.Name,
			CanIUseIDs:   []string{"sample1"},
			WebFeatureID: feature.ID,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func generateMissingOneImplementations(
	featureAvailability map[string]map[string]int,
	features []gcpspanner.SpannerWebFeature,
) {
	for i := 0; i < releasesPerBrowser; i++ {
		// Choose a random browser to be the "missing one" for this release
		missingOneBrowserIndex := r.Intn(len(browsers))

		// Choose a random feature to be the "missing one" for this release
		missingOneFeatureIndex := r.Intn(len(features))
		missingOneFeatureKey := features[missingOneFeatureIndex].FeatureKey

		for j, browser := range browsers {
			// This browser will be the "missing one"
			if j == missingOneBrowserIndex {
				continue
			}

			// Make all browsers except the chosen one support the chosen feature
			// Only mark it as supported if it hasn't been marked before.
			// The browser has a 70% chance of supporting it.
			if _, ok := featureAvailability[browser][missingOneFeatureKey]; !ok && r.Intn(10) < 7 {
				// Mark as supported from this release onwards
				featureAvailability[browser][missingOneFeatureKey] = i + 1
			}

			// For the remaining features, given a 10% chance, assign support status to the current browser
			// only if it hasn't been assigned before
			for k, feature := range features {
				if k != missingOneFeatureIndex { // Skip the "missing one" feature
					if _, ok := featureAvailability[browser][feature.FeatureKey]; !ok && r.Intn(10) == 0 {
						featureAvailability[browser][feature.FeatureKey] = i + 1
					}
				}
			}
		}

		// Mark the "missing one" feature as supported by the "missing one" browser on the next release
		// (if it's not the last release AND it's not already supported AND given a 10% chance)
		if i < releasesPerBrowser-1 {
			missingOneBrowser := browsers[missingOneBrowserIndex]
			if _, ok := featureAvailability[missingOneBrowser][missingOneFeatureKey]; !ok && r.Intn(10) == 0 {
				// Mark as supported from the next release onwards
				featureAvailability[missingOneBrowser][missingOneFeatureKey] = i + 2
			}
		}
	}
}

func generateUnimplementedFeatures(featureAvailability map[string]map[string]int, browsers []string) {
	// Iterate over browsers in a fixed order.
	// If we iterate directly over featureAvailability, the order is not guaranteed.
	for _, browser := range browsers {
		featureReleases := featureAvailability[browser]

		// Extract the keys from the featureReleases map.
		keys := make([]string, 0, len(featureReleases))
		for k := range featureReleases {
			keys = append(keys, k)
		}

		// Sort the keys alphabetically to ensure a consistent iteration order.
		sort.Strings(keys)

		// Iterate over the sorted keys.
		for _, k := range keys {
			// 10% chance of removing the feature.
			if r.Intn(10) == 0 {
				delete(featureReleases, k)
			}
		}
	}
}

func generateFeatureAvailability(
	ctx context.Context,
	client *gcpspanner.Client,
	features []gcpspanner.SpannerWebFeature,
) (int, error) {
	availabilitiesInserted := 0
	// Create a map to track feature availability per browser and release
	featureAvailability := make(map[string]map[string]int) // map[browserName]map[featureKey]releaseNumber

	// Initialize the map with all features marked as unsupported for each browser and release
	for _, browser := range browsers {
		featureAvailability[browser] = make(map[string]int)
	}

	// Ensure at least one "missing one" implementation per release, and vary feature support
	generateMissingOneImplementations(featureAvailability, features)

	// Ensure that some features are never implemented in a browser.
	generateUnimplementedFeatures(featureAvailability, browsers)

	// Insert the availabilities into Spanner
	availabilities := make(map[string][]gcpspanner.BrowserFeatureAvailability)
	for _, browser := range browsers {
		for featureKey, releaseNumber := range featureAvailability[browser] {
			var featureAvailabilities []gcpspanner.BrowserFeatureAvailability
			var found bool
			featureAvailabilities, found = availabilities[featureKey]
			if !found {
				featureAvailabilities = []gcpspanner.BrowserFeatureAvailability{}
			}
			featureAvailabilities = append(featureAvailabilities, gcpspanner.BrowserFeatureAvailability{
				BrowserName:    browser,
				BrowserVersion: fmt.Sprintf("%d", releaseNumber),
			})
			availabilities[featureKey] = featureAvailabilities
			availabilitiesInserted++
		}
	}

	err := client.SyncBrowserFeatureAvailabilities(
		ctx,
		availabilities,
	)
	if err != nil {
		return 0, err
	}

	return availabilitiesInserted, nil
}

func generateGroups(ctx context.Context,
	client *gcpspanner.Client,
	features []gcpspanner.SpannerWebFeature) ([]string, error) {
	groupKeyToInternalID := map[string]string{}
	groups := []gcpspanner.Group{
		{
			GroupKey: "parent1",
			Name:     "Parent 1",
		},
		{
			GroupKey: "parent2",
			Name:     "Parent 2",
		},
		{
			GroupKey: "child3",
			Name:     "Child 3",
		},
	}
	for _, group := range groups {
		id, err := client.UpsertGroup(ctx, group)
		if err != nil {
			return nil, err
		}
		groupKeyToInternalID[group.GroupKey] = *id
	}
	featureKeyToGroupsMapping := make(map[string][]string)
	childGroupKeyToParentGroupKey := map[string]string{
		"child3": "parent1",
	}

	for _, feature := range features {
		var groupKeys []string
		if _, found := featureKeyToGroupsMapping[feature.FeatureKey]; !found {
			groupKeys = []string{}
		}
		group := groups[r.Intn(len(groups))]
		groupKeys = append(groupKeys, group.GroupKey)
		featureKeyToGroupsMapping[feature.FeatureKey] = groupKeys
	}

	err := client.UpsertFeatureGroupLookups(ctx, featureKeyToGroupsMapping, childGroupKeyToParentGroupKey)
	if err != nil {
		return nil, err
	}

	groupKeys := []string{}
	for _, group := range groups {
		groupKeys = append(groupKeys, group.GroupKey)
	}

	return groupKeys, nil
}

func generateSnapshots(ctx context.Context,
	client *gcpspanner.Client,
	features []gcpspanner.SpannerWebFeature) ([]string, error) {
	snapshotKeyToInternalID := map[string]string{}
	snapshots := []gcpspanner.Snapshot{
		{
			SnapshotKey: "parent1",
			Name:        "Parent 1",
		},
		{
			SnapshotKey: "parent2",
			Name:        "Parent 2",
		},
	}
	for _, snapshot := range snapshots {
		id, err := client.UpsertSnapshot(ctx, snapshot)
		if err != nil {
			return nil, err
		}
		snapshotKeyToInternalID[snapshot.SnapshotKey] = *id
	}
	for _, feature := range features {
		snapshot := snapshots[r.Intn(len(snapshots))]
		err := client.UpsertWebFeatureSnapshot(ctx, gcpspanner.WebFeatureSnapshot{
			WebFeatureID: feature.ID,
			SnapshotIDs: []string{
				snapshotKeyToInternalID[snapshot.SnapshotKey],
			},
		})
		if err != nil {
			return nil, err
		}
	}

	snapshotKeys := []string{}
	for _, snapshot := range snapshots {
		snapshotKeys = append(snapshotKeys, snapshot.SnapshotKey)
	}

	return snapshotKeys, nil
}

func findUserIDByEmail(ctx context.Context, email string, authClient *auth.Client) (string, error) {
	record, err := authClient.GetUserByEmail(ctx, email)
	if err != nil {
		slog.ErrorContext(ctx, "error trying to get user", "error", err, "email", email)

		return "", err
	}

	return record.UID, nil
}

func valuePtr[T any](in T) *T { return &in }

func generateSavedSearches(ctx context.Context,
	spannerClient *gcpspanner.Client,
	authClient *auth.Client) (int, error) {
	savedSearchesToInsert := []struct {
		Email       string
		Name        string
		Query       string
		Description *string
		UUID        string
	}{
		{
			Email:       "test.user.1@example.com",
			Name:        "my first project query",
			Query:       "baseline_status:newly",
			Description: nil,
			UUID:        "74bdb85f-59d3-43b0-8061-20d5818e8c97",
		},
		{
			Email: "test.user.1@example.com",
			Name:  "I like queries",
			Query: "baseline_status:limited OR available_on:chrome",
			Description: valuePtr(
				"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed non risus. " +
					"Suspendisse lectus tortor, dignissim sit amet, adipiscing nec, ultricies sed, dolor. " +
					"Cras elementum ultrices diam. Maecenas ligula massa, varius a, semper congue, euismod " +
					"non, mi. Proin porttitor, orci nec nonummy molestie, enim est eleifend mi, non fermentum " +
					"diam nisl sit amet erat. Duis semper. Duis arcu massa, scelerisque vitae, consequat in, " +
					"pretium a, enim. Pellentesque congue. Ut in risus volutpat libero pharetra tempor. Cras " +
					"vestibulum bibendum augue. Praesent egestas leo in pede. Praesent blandit odio eu enim. " +
					"Pellentesque sed dui ut augue blandit sodales. Vestibulum ante ipsum primis in faucibus " +
					"orci luctus et ultrices posuere cubilia Curae; Aliquam nibh. Mauris ac mauris sed pede " +
					"pellentesque fermentum. Maecenas adipiscing ante non diam sodales hendrerit. Ut velit " +
					"mauris, egestas sed, gravida nec, ornare ut, mi. Aenean ut orci vel massa suscipit " +
					"pulvinar. Nulla sollicitudin. Fusce varius, ligula non tempus aliquam, nunc turpis " +
					"ullamcorper nibh, in tempus sapien eros vitae ligula. Pellentesque rhoncus nunc et augue. " +
					"Integer id felis. Curabitur aliquet pellentesque diam. Integer quis metus vitae elit " +
					"lobortis egestas. Integer egestas risus ut lectus. Nam viverra, erat vitae porta " +
					"sodales, nulla diam tincidunt sem, et dictum felis nunc nec ligula. Sed nec lectus. " +
					"Donec in velit. Curabitur tempus. Sed consequat, leo eget bibendum sodales, augue velit " +
					"cursus nunc, quis gravida magna mi a libero. Duis vulputate elit eu elit. Donec interdum, " +
					"metus et hendrerit aliquet, dolor diam sagittis ligula, eget egestas libero turpis vel " +
					"mi. Nunc nulla. Maecenas vitae neque. Vivamus ultrices luctus nunc. Vivamus cursus, metus " +
					"quis ullamcorper sodales, lectus lectus tempor enim, vitae gravida nibh purus ut nibh. " +
					"Duis in augue. Cras nulla. Vivamus laoreet. Curabitur suscipit suscipit tellus."),
			UUID: "a09386fe-65f1-4640-b28d-3cf2f2de69c9",
		},
		{
			Email:       "test.user.2@example.com",
			Name:        "test user 2's query",
			Query:       "baseline_status:limited",
			Description: valuePtr("other users can create queries too"),
			UUID:        "bb85baf7-aa1e-42bf-ada0-cf9d2811dd42",
		},
	}

	for _, savedSearch := range savedSearchesToInsert {
		userID, err := findUserIDByEmail(ctx, savedSearch.Email, authClient)
		if err != nil {
			return 0, err
		}
		id, err := spannerClient.CreateNewUserSavedSearchWithUUID(ctx, gcpspanner.CreateUserSavedSearchRequest{
			OwnerUserID: userID,
			Name:        savedSearch.Name,
			Query:       savedSearch.Query,
			Description: savedSearch.Description,
		}, savedSearch.UUID)
		if err != nil {
			return 0, err
		}
		slog.InfoContext(ctx, "saved search created", "id", *id)
	}

	return len(savedSearchesToInsert), nil
}

func generateSavedSearchBookmarks(ctx context.Context, spannerClient *gcpspanner.Client,
	authClient *auth.Client) (int, error) {
	bookmarksToInsert := []struct {
		UUID  string
		Email string
	}{
		{
			UUID:  "bb85baf7-aa1e-42bf-ada0-cf9d2811dd42",
			Email: "test.user.1@example.com",
		},
	}
	for _, bookmarkToInsert := range bookmarksToInsert {
		userID, err := findUserIDByEmail(ctx, bookmarkToInsert.Email, authClient)
		if err != nil {
			return 0, err
		}
		err = spannerClient.AddUserSearchBookmark(ctx, gcpspanner.UserSavedSearchBookmark{
			UserID:        userID,
			SavedSearchID: bookmarkToInsert.UUID,
		})
		if err != nil {
			return 0, err
		}
	}

	return len(bookmarksToInsert), nil
}

func generateUserData(ctx context.Context, spannerClient *gcpspanner.Client,
	authClient *auth.Client) error {
	savedSearchesCount, err := generateSavedSearches(ctx, spannerClient, authClient)
	if err != nil {
		return fmt.Errorf("saved searches generation failed %w", err)
	}
	slog.InfoContext(ctx, "saved searches generated",
		"amount of searches created", savedSearchesCount)

	bookmarkCount, err := generateSavedSearchBookmarks(ctx, spannerClient, authClient)
	if err != nil {
		return fmt.Errorf("saved search bookmarks generation failed %w", err)

	}
	slog.InfoContext(ctx, "saved search bookmarks generated",
		"amount of bookmarks created", bookmarkCount)

	subscriptionsCount, err := generateSubscriptions(ctx, spannerClient, authClient)
	if err != nil {
		return fmt.Errorf("subscriptions generation failed %w", err)
	}
	slog.InfoContext(ctx, "subscriptions generated",
		"amount of subscriptions created", subscriptionsCount)

	return nil
}

func generateSubscriptions(ctx context.Context,
	spannerClient *gcpspanner.Client,
	authClient *auth.Client) (int, error) {
	subscriptionsToInsert := []struct {
		Email         string
		SavedSearchID string
		Frequency     gcpspanner.SavedSearchSnapshotType
		Triggers      []gcpspanner.SubscriptionTrigger
		UUID          string
	}{
		{
			Email:         "test.user.1@example.com",
			SavedSearchID: "74bdb85f-59d3-43b0-8061-20d5818e8c97", // "my first project query"
			Frequency:     gcpspanner.SavedSearchSnapshotTypeWeekly,
			Triggers: []gcpspanner.SubscriptionTrigger{
				gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToNewly,
			},
			UUID: "c1aa6418-1229-43a1-9a98-3f3604efe2ae",
		},
	}

	subscriptionsCreated := 0
	for _, sub := range subscriptionsToInsert {
		userID, err := findUserIDByEmail(ctx, sub.Email, authClient)
		if err != nil {
			slog.WarnContext(ctx, "failed to find user for subscription, skipping", "email", sub.Email, "error", err)

			continue
		}

		// The notification channel is created on login. If it doesn't exist, we can't create a subscription.
		// This is not an error, as some test setups may not involve logging in users.
		channels, _, err := spannerClient.ListNotificationChannels(ctx, gcpspanner.ListNotificationChannelsRequest{
			UserID:    userID,
			PageSize:  100,
			PageToken: nil,
		})
		if err != nil {
			slog.ErrorContext(ctx, "failed to list notification channels, skipping subscription creation",
				"user", sub.Email, "error", err)

			return 0, err
		}
		if len(channels) == 0 {
			slog.WarnContext(ctx, "no notification channels found for user, skipping subscription creation",
				"user", sub.Email)

			continue
		}
		var channel gcpspanner.NotificationChannel
		var found bool
		for _, ch := range channels {
			if ch.EmailConfig != nil && ch.EmailConfig.Address == sub.Email {
				channel = ch
				found = true

				break
			}
		}
		if !found {
			slog.WarnContext(ctx, "no notification channel found for user with matching email address, "+
				"skipping subscription creation",
				"user", sub.Email)

			continue
		}

		_, err = spannerClient.CreateSubscriptionWithUUID(ctx, gcpspanner.CreateSavedSearchSubscriptionRequest{
			UserID:        userID,
			SavedSearchID: sub.SavedSearchID,
			ChannelID:     channel.ID,
			Frequency:     sub.Frequency,
			Triggers:      sub.Triggers,
		}, sub.UUID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to create subscription with UUID", "uuid", sub.UUID, "error", err)

			return 0, err
		}
		subscriptionsCreated++
	}

	return subscriptionsCreated, nil
}

func generateData(ctx context.Context, spannerClient *gcpspanner.Client, datastoreClient *gds.Client) error {
	releasesCount, err := generateReleases(ctx, spannerClient)
	if err != nil {
		return fmt.Errorf("release generation failed %w", err)
	}
	slog.InfoContext(ctx, "releases generated",
		"amount of releases created", releasesCount)

	fh := &featuresHelper{
		features: make(map[string]gcpspanner.WebFeature),
	}
	features, webFeatureKeyToInternalFeatureID, err := generateFeatures(ctx, spannerClient, fh)
	if err != nil {
		return fmt.Errorf("feature generation failed %w", err)
	}
	slog.InfoContext(ctx, "features generated",
		"amount of features created", len(features))

	err = generateFeatureMetadata(ctx, datastoreClient, features)
	if err != nil {
		return fmt.Errorf("feature metadata generation failed %w", err)
	}
	slog.InfoContext(ctx, "feature metadata generated",
		"amount of feature metadata created", len(features))

	runsCount, metricsCount, err := generateRunsAndMetrics(ctx, spannerClient, features)
	if err != nil {
		return fmt.Errorf("wpt runs generation failed %w", err)
	}
	slog.InfoContext(ctx, "runs and metrics generated",
		"amount of runs created", runsCount, "amount of metrics created", metricsCount)

	statusCount, err := generateBaselineStatus(ctx, spannerClient, features)
	if err != nil {
		return fmt.Errorf("baseline status failed %w", err)
	}
	slog.InfoContext(ctx, "statuses generated",
		"amount of statuses created", statusCount)

	availabilityCount, err := generateFeatureAvailability(ctx, spannerClient, features)
	if err != nil {
		return fmt.Errorf("feature availability generation failed %w", err)
	}
	slog.InfoContext(ctx, "availabilities generated",
		"amount of availabilities created", availabilityCount)

	// Only ~12 months
	err = spannerClient.PrecalculateBrowserFeatureSupportEvents(ctx, startTimeWindow, startTimeWindow.Add(
		12*30*24*time.Hour))
	if err != nil {
		return fmt.Errorf("browser feature support precalculation failed %w", err)
	}
	slog.InfoContext(ctx, "browser feature support precalculation complete")

	groupKeys, err := generateGroups(ctx, spannerClient, features)
	if err != nil {
		return fmt.Errorf("group generation failed %w", err)
	}
	slog.InfoContext(ctx, "groups generated",
		"groupKeys", groupKeys)

	snapshotKeys, err := generateSnapshots(ctx, spannerClient, features)
	if err != nil {
		return fmt.Errorf("snapshot generation failed %w", err)
	}
	slog.InfoContext(ctx, "snapshots generated",
		"snapshotKeys", snapshotKeys)

	chromiumHistogramEnumIDMap, err := generateChromiumHistogramEnums(ctx, spannerClient)
	if err != nil {
		return fmt.Errorf("chromium histogram enums generation failed %w", err)
	}
	slog.InfoContext(ctx, "enums generated", "size", len(chromiumHistogramEnumIDMap))

	chromiumHistogramEnumValueToIDMap, err := generateChromiumHistogramEnumValues(
		ctx, spannerClient, chromiumHistogramEnumIDMap, features)
	if err != nil {
		return fmt.Errorf("chromium histogram enum values generation failed %w", err)
	}
	slog.InfoContext(ctx, "enum values generated", "size", len(chromiumHistogramEnumValueToIDMap))

	err = generateWebFeatureChromiumHistogramEnumValues(
		ctx, spannerClient, webFeatureKeyToInternalFeatureID, chromiumHistogramEnumValueToIDMap, features)
	if err != nil {
		return fmt.Errorf("web feature chromium histogram enums values generation failed %w", err)
	}
	slog.InfoContext(ctx, "web feature to enum mapping generated")

	chromiumMetricsCount, err := generateChromiumHistogramMetrics(ctx, spannerClient, features)
	if err != nil {
		return fmt.Errorf("chromium histogram metrics generation failed %w", err)
	}
	slog.InfoContext(ctx, "chromium histogram metrics generated",
		"amount of metrics generated", chromiumMetricsCount)

	signalsGenerated, err := generateFeatureDeveloperSignals(ctx, spannerClient, features)
	if err != nil {
		return fmt.Errorf("feature developer signals generation failed %w", err)
	}
	slog.InfoContext(ctx, "feature developer signals generated",
		"amount of signals generated", signalsGenerated)

	discouragedFeaturesCount, err := generateDiscouragedFeatures(ctx, spannerClient, features)
	if err != nil {
		return fmt.Errorf("discouraged features generation failed %w", err)
	}
	slog.InfoContext(ctx, "discouraged features generated",
		"amount of discouraged features generated", discouragedFeaturesCount)

	mappingDataCount, err := generateWebFeaturesMappingData(ctx, spannerClient, features)
	if err != nil {
		return fmt.Errorf("web features mapping data generation failed %w", err)
	}
	slog.InfoContext(ctx, "web features mapping data generated",
		"amount of mapping data created", mappingDataCount)

	err = generateEvolutionOfFeatures(ctx, spannerClient, fh)
	if err != nil {
		return fmt.Errorf("feature evolution generation failed %w", err)
	}
	slog.InfoContext(ctx, "feature evolution generation complete")

	return nil
}

func generateWebFeaturesMappingData(
	ctx context.Context, client *gcpspanner.Client, features []gcpspanner.SpannerWebFeature) (int, error) {
	mappings := []gcpspanner.WebFeaturesMappingData{}
	for _, feature := range features {
		positions := createFakeVendorPositions(feature.FeatureKey)
		vendorPositions, err := spanneradapters.VendorPositionsToNullJSON(positions)
		if err != nil {
			return 0, err
		}
		if vendorPositions.Valid {
			mappings = append(mappings, gcpspanner.WebFeaturesMappingData{
				WebFeatureID:    feature.FeatureKey,
				VendorPositions: vendorPositions,
			})
		}
	}

	err := client.SyncWebFeaturesMappingData(ctx, mappings)
	if err != nil {
		return 0, err
	}

	return len(mappings), nil
}

// createFakeVendorPositionsNullJSON creates a spanner.NullJSON object containing vendor positions.
// It returns a fixed position for the featurePageFeatureKey and random positions for other features.
func createFakeVendorPositions(featureKey string) []webfeaturesmappingtypes.StandardsPosition {
	var positions []webfeaturesmappingtypes.StandardsPosition

	if featureKey == featurePageFeatureKey {
		// Hardcode the positions for the special feature key
		positions = []webfeaturesmappingtypes.StandardsPosition{
			{
				Vendor:   string(web_platform_dx__web_features_mappings.Mozilla),
				Position: string(web_platform_dx__web_features_mappings.Positive),
				URL:      "https://example.com/mozilla",
				Concerns: nil,
			},
			{
				Vendor:   string(web_platform_dx__web_features_mappings.Apple),
				Position: string(web_platform_dx__web_features_mappings.Negative),
				URL:      "https://example.com/webkit",
				Concerns: nil,
			},
		}
	} else {
		// 90% chance of having a mapping for other features
		if r.Intn(10) < 9 {
			allVendors := getAllVendors()
			allVendorPositions := getAllVendorPositions()
			for _, vendor := range allVendors {
				// 50% chance for each vendor to have a position
				if r.Intn(2) == 0 {
					position := allVendorPositions[r.Intn(len(allVendorPositions))]
					positions = append(positions, webfeaturesmappingtypes.StandardsPosition{
						Vendor:   vendor,
						Position: position,
						URL:      gofakeit.URL(),
						Concerns: nil,
					})
				}
			}
		}
	}

	return positions
}

// getVendorToStringSet provides a string representation for each Vendor enum.
// The exhaustive linter is configured to check that this map is complete.
func getVendorToStringSet() map[web_platform_dx__web_features_mappings.Vendor]any {
	return map[web_platform_dx__web_features_mappings.Vendor]any{
		web_platform_dx__web_features_mappings.Mozilla: nil,
		web_platform_dx__web_features_mappings.Apple:   nil,
	}

}

// getAllVendors returns a slice of all Vendor enums, generated from the map above.
func getAllVendors() []string {
	vendorMap := getVendorToStringSet()
	keys := make([]string, 0, len(vendorMap))
	for k := range vendorMap {
		keys = append(keys, string(k))
	}
	slices.Sort(keys)

	return keys

}

// getPositionToStringSet provides a string representation for each Position enum.
// The exhaustive linter is configured to check that this map is complete.
func getPositionToStringSet() map[web_platform_dx__web_features_mappings.Position]any {
	return map[web_platform_dx__web_features_mappings.Position]any{
		web_platform_dx__web_features_mappings.Positive: nil,
		web_platform_dx__web_features_mappings.Negative: nil,
		web_platform_dx__web_features_mappings.Neutral:  nil,
		web_platform_dx__web_features_mappings.Support:  nil,
		web_platform_dx__web_features_mappings.Oppose:   nil,
		web_platform_dx__web_features_mappings.Defer:    nil,
		web_platform_dx__web_features_mappings.Blocked:  nil,
		web_platform_dx__web_features_mappings.Empty:    nil,
	}

}

// getAllVendorPositions returns a slice of all Position enums, generated from the map above.
// It excludes the empty position, which is not useful for random generation.
func getAllVendorPositions() []string {
	positionMap := getPositionToStringSet()
	keys := make([]string, 0, len(positionMap))
	for k := range positionMap {
		if k != web_platform_dx__web_features_mappings.Empty {
			keys = append(keys, string(k))
		}
	}
	slices.Sort(keys)

	return keys

}

func generateBaselineStatus(
	ctx context.Context, client *gcpspanner.Client, features []gcpspanner.SpannerWebFeature) (int, error) {
	statusesGenerated := 0
	noneValue := gcpspanner.BaselineStatusNone
	lowValue := gcpspanner.BaselineStatusLow
	highValue := gcpspanner.BaselineStatusHigh
	statuses := []*gcpspanner.BaselineStatus{
		nil,
		&noneValue,
		&lowValue,
		&highValue,
	}

	baseDate := startTimeWindow
	for _, feature := range features {
		statusIndex := r.Intn(len(statuses))
		var highDate *time.Time
		var lowDate *time.Time
		if feature.FeatureKey == featurePageFeatureKey {
			statusIndex = 2
		}
		switch statuses[statusIndex] {
		case &highValue:
			adjustedTime := baseDate.AddDate(0, 0, r.Intn(30)) // Add up to 1 month
			lowDate = &adjustedTime
			highAdjustedTime := adjustedTime.AddDate(0, 0, r.Intn(30)) // Add up to another month
			highDate = &highAdjustedTime
		case &lowValue:
			adjustedTime := baseDate.AddDate(0, 0, r.Intn(30)) // Add up to 1 month
			lowDate = &adjustedTime
		case nil, &noneValue:
			// Do nothing.
		}
		err := client.UpsertFeatureBaselineStatus(ctx, feature.FeatureKey, gcpspanner.FeatureBaselineStatus{
			Status:   statuses[statusIndex],
			LowDate:  lowDate,
			HighDate: highDate,
		})
		if err != nil {
			return statusesGenerated, err
		}
		statusesGenerated++

		baseDate = baseDate.AddDate(0, 1, r.Intn(90)) // Add 1 month to ~3 months

	}

	return statusesGenerated, nil
}

func generateRunsAndMetrics(
	ctx context.Context, client *gcpspanner.Client, features []gcpspanner.SpannerWebFeature) (int, int, error) {
	// For now only generate one run with metrics per browser+channel combination.
	// TODO. Need to think about the graphs we want to draw.
	runsGenerated := 0
	metricsGenerated := 0
	channels := []string{shared.StableLabel, shared.ExperimentalLabel}
	for _, channel := range channels {
		for _, browser := range browsers {
			totalDuration := runsPerBrowserPerChannel * 3
			baseTime := startTimeWindow
			for i := 0; i < totalDuration; i += 3 {
				timeStart := baseTime.AddDate(0, 0, i)
				timeEnd := timeStart.Add(time.Duration(r.Intn(5)) * time.Hour)
				runID := r.Int63n(1000000)
				run := gcpspanner.WPTRun{
					RunID:            runID,
					TimeStart:        timeStart,
					TimeEnd:          timeEnd,
					BrowserName:      browser,
					BrowserVersion:   "0.0.0",
					Channel:          channel,
					OSName:           "os",
					OSVersion:        "0.0.0",
					FullRevisionHash: "abcdef0123456789",
				}
				err := client.InsertWPTRun(ctx, run)
				if err != nil {
					return runsGenerated, metricsGenerated, err
				}

				runsGenerated++

				wptRunData, err := client.GetWPTRunDataByRunIDForMetrics(ctx, runID)
				if err != nil {
					return runsGenerated, metricsGenerated, err
				}

				var mutations []*spanner.Mutation
				for _, feature := range features {
					testPass := r.Int63n(1000)
					testTotal := testPass + r.Int63n(1000)
					subtestPass := testPass * 10
					subtestTotal := testTotal * 10
					metric := gcpspanner.WPTRunFeatureMetric{
						TotalTests:        &testTotal,
						TestPass:          &testPass,
						TotalSubtests:     &subtestTotal,
						SubtestPass:       &subtestPass,
						FeatureRunDetails: nil,
					}
					spannerMetric := client.CreateSpannerWPTRunFeatureMetric(feature.ID, *wptRunData, metric)
					m, err := spanner.InsertOrUpdateStruct(gcpspanner.WPTRunFeatureMetricTable, spannerMetric)
					if err != nil {
						return runsGenerated, metricsGenerated, err
					}
					mutations = append(mutations, m)
				}
				writer := gcpspanner.LocalBatchWriter{}
				err = writer.BatchWriteMutations(ctx, client.Client, mutations)
				if err != nil {
					return runsGenerated, metricsGenerated, err
				}
				metricsGenerated += len(mutations)
			}
		}
	}

	return runsGenerated, metricsGenerated, nil
}

func generateChromiumHistogramEnums(
	ctx context.Context, client *gcpspanner.Client) (map[string]string, error) {
	sampleChromiumHistogramEnums := []gcpspanner.ChromiumHistogramEnum{
		{
			HistogramName: "WebDXFeatureObserver",
		},
	}
	chromiumHistogramEnumIDMap := make(map[string]string, len(sampleChromiumHistogramEnums))
	for _, enum := range sampleChromiumHistogramEnums {
		id, err := client.UpsertChromiumHistogramEnum(ctx, enum)
		if err != nil {
			return nil, err
		}
		chromiumHistogramEnumIDMap[enum.HistogramName] = *id
	}

	return chromiumHistogramEnumIDMap, nil
}

func generateChromiumHistogramEnumValues(
	ctx context.Context,
	client *gcpspanner.Client,
	chromiumHistogramEnumIDMap map[string]string,
	features []gcpspanner.SpannerWebFeature,
) (map[string]string, error) {
	chromiumHistogramEnumValueToIDMap := make(map[string]string, len(features))
	enumValuesToSync := make([]gcpspanner.ChromiumHistogramEnumValue, 0, len(features))
	for i, feature := range features {
		ChromiumHistogramEnumValueEntry := gcpspanner.ChromiumHistogramEnumValue{
			ChromiumHistogramEnumID: chromiumHistogramEnumIDMap["WebDXFeatureObserver"],
			BucketID:                int64(i + 1),
			Label:                   feature.FeatureKey,
		}
		enumValuesToSync = append(enumValuesToSync, ChromiumHistogramEnumValueEntry)
	}

	err := client.SyncChromiumHistogramEnumValues(ctx, enumValuesToSync)
	if err != nil {
		return nil, err
	}

	for i, feature := range features {
		enumValueID, err := client.GetIDFromChromiumHistogramEnumValueKey(
			ctx,
			chromiumHistogramEnumIDMap["WebDXFeatureObserver"],
			int64(i+1),
		)
		if err != nil {
			// It is possible that the enum value was deleted in the sync.
			// In that case, we just log it and continue.
			slog.WarnContext(ctx, "unable to get enum value id", "featureKey", feature.FeatureKey)

			continue
		}
		chromiumHistogramEnumValueToIDMap[feature.FeatureKey] = *enumValueID
	}

	return chromiumHistogramEnumValueToIDMap, nil
}

func generateWebFeatureChromiumHistogramEnumValues(
	ctx context.Context,
	client *gcpspanner.Client,
	webFeatureKeyToInternalFeatureID map[string]string,
	chromiumHistogramEnumValueToIDMap map[string]string,
	features []gcpspanner.SpannerWebFeature,
) error {
	values := make([]gcpspanner.WebFeatureChromiumHistogramEnumValue, 0, len(features))
	for _, feature := range features {
		webFeatureChromiumHistogramEnumValueEntry := gcpspanner.WebFeatureChromiumHistogramEnumValue{
			WebFeatureID:                 webFeatureKeyToInternalFeatureID[feature.FeatureKey],
			ChromiumHistogramEnumValueID: chromiumHistogramEnumValueToIDMap[feature.FeatureKey],
		}
		values = append(values, webFeatureChromiumHistogramEnumValueEntry)

	}

	err := client.SyncWebFeatureChromiumHistogramEnumValues(ctx, values)
	if err != nil {
		return err
	}

	return nil
}

func generateChromiumHistogramMetrics(
	ctx context.Context, client *gcpspanner.Client, features []gcpspanner.SpannerWebFeature) (int, error) {
	metricsCount := 0
	for i := range len(features) {
		currDate := startTimeWindow
		// For testing, some features (~20%) have no usage data.
		var modifier = r.Intn(5)
		if modifier == 0 && features[i].FeatureKey != featurePageFeatureKey {
			continue
		}
		for currDate.Before(time.Date(2020, time.December, 1, 0, 0, 0, 0, time.UTC)) {
			var usage *big.Rat
			var modifier = r.Intn(4)

			switch modifier {
			case 0:
				usage = big.NewRat(0, 1) // explicitly zero usage.
			case 1:
				usage = big.NewRat(1, 100000) // very tiny amount (<0.1%).
			default:
				usage = big.NewRat(r.Int63n(10000), 10000) // Generate usage between 0-100%
			}

			err := client.StoreDailyChromiumHistogramMetrics(
				ctx,
				metricdatatypes.WebDXFeatureEnum, map[int64]gcpspanner.DailyChromiumHistogramMetric{
					int64(i + 1): {
						Day:  civil.DateOf(currDate),
						Rate: *usage,
					},
				},
			)

			if err != nil {
				return metricsCount, err
			}
			if features[i].FeatureKey == featurePageFeatureKey {
				// Add more data points to assert pagination of metrics for feature page.
				currDate = currDate.AddDate(0, 0, 1) // Add 1 day.
			} else {
				currDate = currDate.AddDate(0, 0, r.Intn(23)+7) // Add up to a month, increasing by at least 7 days.
			}
			metricsCount++
		}
	}

	err := client.SyncLatestDailyChromiumHistogramMetrics(ctx)
	if err != nil {
		return metricsCount, err
	}

	return metricsCount, nil
}

func generateFeatureDeveloperSignals(
	ctx context.Context, client *gcpspanner.Client, features []gcpspanner.SpannerWebFeature) (int, error) {
	signals := []gcpspanner.FeatureDeveloperSignal{}
	for _, feature := range features {
		var modifier = r.Intn(5)
		if modifier == 0 && feature.FeatureKey != featurePageFeatureKey {
			continue
		}

		signals = append(signals, gcpspanner.FeatureDeveloperSignal{
			WebFeatureKey: feature.FeatureKey,
			Upvotes:       r.Int63n(1000),
			Link:          "https://fake-github.com/org/repo/issues/4",
		})

	}

	err := client.SyncLatestFeatureDeveloperSignals(ctx, signals)
	if err != nil {
		return 0, err
	}

	return len(signals), nil
}

func generateDiscouragedFeatures(
	ctx context.Context, client *gcpspanner.Client, features []gcpspanner.SpannerWebFeature) (int, error) {
	discouragedFeatures := 0
	for _, feature := range features {

		// Only 10 percent of the features are discouraged. Or if it is the discouragedFeatureKey.
		// Also, ensure that it is not the feature page feature key.
		var modifier = r.Intn(10)
		if (modifier == 0 || feature.FeatureKey == discouragedFeatureKey) &&
			feature.FeatureKey != featurePageFeatureKey {
			err := client.UpsertFeatureDiscouragedDetails(ctx, feature.FeatureKey,
				gcpspanner.FeatureDiscouragedDetails{
					AccordingTo: []string{
						"https://webstatus.dev",
						"https://example.com",
					},
					Alternatives: []string{
						featurePageFeatureKey,
					},
				})
			if err != nil {
				return 0, err
			}
			discouragedFeatures++
		}
	}

	return discouragedFeatures, nil
}

func initFirebaseAuthClient(ctx context.Context, projectID string) *auth.Client {
	// nolint:exhaustruct // WONTFIX - will rely on the defaults on this third party struct.
	firebaseApp, err := firebase.NewApp(context.Background(), &firebase.Config{
		ProjectID: projectID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "error initializing firebase app", "error", err)
		os.Exit(1)
	}

	// Access Auth service from default app
	firebaseAuthClient, err := firebaseApp.Auth(context.Background())
	if err != nil {
		slog.ErrorContext(context.TODO(), "error getting Auth client", "error", err)
		os.Exit(1)
	}

	return firebaseAuthClient
}

func main() {
	// Use the grpc port from spanner in .dev/spanner/skaffold.yaml
	// Describe the command line flags and parse the flags
	var (
		spannerProject    = flag.String("spanner_project", "", "Spanner Project")
		spannerInstance   = flag.String("spanner_instance", "", "Spanner Instance")
		spannerDatabase   = flag.String("spanner_database", "", "Spanner Database")
		datastoreProject  = flag.String("datastore_project", "", "Datastore Project")
		datastoreDatabase = flag.String("datastore_database", "", "Datastore Database")
		scope             = flag.String("scope", "all", "Scope of data generation: all, user")
		resetFlag         = flag.Bool("reset", false, "Reset test user data before loading")
	)
	flag.Parse()

	slog.InfoContext(context.TODO(), "establishing spanner client",
		"project", *spannerProject,
		"instance", *spannerInstance,
		"database", *spannerDatabase)

	spannerClient, err := gcpspanner.NewSpannerClient(*spannerProject, *spannerInstance, *spannerDatabase)
	if err != nil {
		slog.ErrorContext(context.TODO(), "unable to create spanner client", "error", err)
		os.Exit(1)
	}

	slog.InfoContext(context.TODO(), "establishing datastore client",
		"project", *datastoreProject,
		"database", *datastoreDatabase)

	datastoreClient, err := gds.NewDatastoreClient(*datastoreProject, datastoreDatabase)
	if err != nil {
		slog.ErrorContext(
			context.TODO(), "unable to create datastore client", "error", err)
		os.Exit(1)
	}

	// Use the same project as spanner
	slog.InfoContext(context.TODO(), "establishing firebase auth client", "project", *spannerProject)

	firebaseAuthClient := initFirebaseAuthClient(context.Background(), *spannerProject)

	gofakeit.GlobalFaker = gofakeit.New(seedValue)

	ctx := context.Background()

	var finalErr error

	switch *scope {
	case "user":
		if *resetFlag {
			err := resetTestData(ctx, spannerClient, firebaseAuthClient)
			if err != nil {
				finalErr = fmt.Errorf("failed during test user data reset: %w", err)

				break
			}
		}
		err := generateUserData(ctx, spannerClient, firebaseAuthClient)
		if err != nil {
			finalErr = fmt.Errorf("failed during user data generation: %w", err)
		}
	case "all":
		slog.InfoContext(ctx, "Generating all data (base + user)...")
		errUser := generateUserData(ctx, spannerClient, firebaseAuthClient)
		if errUser != nil {
			finalErr = errUser

			break
		}
		errBase := generateData(ctx, spannerClient, datastoreClient)
		if errBase != nil {
			finalErr = errors.Join(finalErr, errBase)
		}

	default:
		finalErr = fmt.Errorf("invalid scope specified: %s", *scope)
	}

	if finalErr != nil {
		slog.ErrorContext(ctx, "Data generation failed", "scope", *scope, "reset", *resetFlag, "error", finalErr)
		os.Exit(1)
	}
	slog.InfoContext(ctx, "loading fake data successful")
}
