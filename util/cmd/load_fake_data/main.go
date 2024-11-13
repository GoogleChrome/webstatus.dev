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
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gds"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/web-platform-tests/wpt.fyi/shared"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const releasesPerBrowser = 50
const runsPerBrowserPerChannel = 100
const numberOfFeatures = 80

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
	}
)

func generateReleases(ctx context.Context, c *gcpspanner.Client) (int, error) {
	releasesGenerated := 0
	for _, browser := range browsers {
		baseDate := startTimeWindow
		releases := make([]gcpspanner.BrowserRelease, 0, releasesPerBrowser)
		for i := 0; i < releasesPerBrowser; i++ {
			if i > 1 {
				baseDate = releases[i-1].ReleaseDate.AddDate(0, 1, r.Intn(90)) // Add 1 month to ~3 months
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
	ctx context.Context, client *gcpspanner.Client) ([]gcpspanner.SpannerWebFeature, map[string]string, error) {
	features := make([]gcpspanner.SpannerWebFeature, 0, numberOfFeatures)
	featureIDMap := make(map[string]interface{})
	webFeatureKeyToInternalFeatureID := map[string]string{}

	for len(featureIDMap) < numberOfFeatures {
		word := fmt.Sprintf("%s%d", gofakeit.LoremIpsumWord(), len(featureIDMap))
		featureName := cases.Title(language.English).String(word)
		featureID := strings.ToLower(featureName)
		// Check if we already generated this ID.
		if _, alreadyUsed := featureIDMap[word]; alreadyUsed {
			continue
		}
		// Add it to the map.
		featureIDMap[word] = nil
		feature := gcpspanner.WebFeature{
			Name:       featureName,
			FeatureKey: featureID,
		}
		_, err := client.UpsertWebFeature(ctx, feature)
		if err != nil {
			return nil, nil, err
		}
		id, err := client.GetIDFromFeatureKey(ctx, gcpspanner.NewFeatureKeyFilter(featureID))
		if err != nil {
			return nil, nil, err
		}
		webFeatureKeyToInternalFeatureID[featureID] = *id
		features = append(features, gcpspanner.SpannerWebFeature{
			WebFeature: feature,
			ID:         *id,
		})
	}

	return features, webFeatureKeyToInternalFeatureID, nil
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

func generateFeatureAvailability(
	ctx context.Context,
	client *gcpspanner.Client,
	features []gcpspanner.SpannerWebFeature) (int, error) {
	availabilitiesInserted := 0
	for _, browser := range browsers {
		for _, feature := range features {
			// Add availability randomly at a 50% chance
			releaseVersion := r.Int31n(2*releasesPerBrowser) + 1
			if releaseVersion <= releasesPerBrowser {
				err := client.InsertBrowserFeatureAvailability(
					ctx,
					feature.FeatureKey,
					gcpspanner.BrowserFeatureAvailability{
						BrowserName:    browser,
						BrowserVersion: fmt.Sprintf("%d", releaseVersion),
					},
				)
				if err != nil {
					return availabilitiesInserted, err
				}
				availabilitiesInserted++
			}
		}
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
	groupDescArr := []struct {
		groupKey string
		info     gcpspanner.GroupDescendantInfo
	}{
		{
			groupKey: "parent1",
			info: gcpspanner.GroupDescendantInfo{
				DescendantGroupIDs: []string{
					groupKeyToInternalID["child3"],
				},
			},
		},
	}
	for _, info := range groupDescArr {
		err := client.UpsertGroupDescendantInfo(ctx, info.groupKey, info.info)
		if err != nil {
			return nil, err
		}
	}
	for _, feature := range features {
		group := groups[r.Intn(len(groups))]
		err := client.UpsertWebFeatureGroup(ctx, gcpspanner.WebFeatureGroup{
			WebFeatureID: feature.ID,
			GroupIDs: []string{
				groupKeyToInternalID[group.GroupKey],
			},
		})
		if err != nil {
			return nil, err
		}
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

func generateData(ctx context.Context, spannerClient *gcpspanner.Client, datastoreClient *gds.Client) error {
	releasesCount, err := generateReleases(ctx, spannerClient)
	if err != nil {
		return fmt.Errorf("release generation failed %w", err)
	}
	slog.Info("releases generated",
		"amount of releases created", releasesCount)

	features, webFeatureKeyToInternalFeatureID, err := generateFeatures(ctx, spannerClient)
	if err != nil {
		return fmt.Errorf("feature generation failed %w", err)
	}
	slog.Info("features generated",
		"amount of features created", len(features))

	chromiumHistogramEnumIDMap, err := generateChromiumHistogramEnums(ctx, spannerClient)
	if err != nil {
		return fmt.Errorf("chromium histogram enums generation failed %w", err)
	}

	chromiumHistogramEnumValueToIDMap, err := generateChromiumHistogramEnumValues(
		ctx, spannerClient, chromiumHistogramEnumIDMap, features)
	if err != nil {
		return fmt.Errorf("chromium histogram enum values generation failed %w", err)
	}

	err = generateWebFeatureChromiumHistogramEnumValues(
		ctx, spannerClient, webFeatureKeyToInternalFeatureID, chromiumHistogramEnumValueToIDMap, features)
	if err != nil {
		return fmt.Errorf("web feature chromium histogram enums values generation failed %w", err)
	}

	err = generateChromiumHistogramMetrics(ctx, spannerClient, features)
	if err != nil {
		return fmt.Errorf("chromium histogram metrics generation failed %w", err)
	}

	err = generateFeatureMetadata(ctx, datastoreClient, features)
	if err != nil {
		return fmt.Errorf("feature metadata generation failed %w", err)
	}
	slog.Info("feature metadata generated",
		"amount of feature metadata created", len(features))

	runsCount, metricsCount, err := generateRunsAndMetrics(ctx, spannerClient, features)
	if err != nil {
		return fmt.Errorf("wpt runs generation failed %w", err)
	}
	slog.Info("runs and metrics generated",
		"amount of runs created", runsCount, "amount of metrics created", metricsCount)

	statusCount, err := generateBaselineStatus(ctx, spannerClient, features)
	if err != nil {
		return fmt.Errorf("baseline status failed %w", err)
	}
	slog.Info("statuses generated",
		"amount of statuses created", statusCount)

	availabilityCount, err := generateFeatureAvailability(ctx, spannerClient, features)
	if err != nil {
		return fmt.Errorf("feature availability generation failed %w", err)
	}
	slog.Info("availabilities generated",
		"amount of availabilities created", availabilityCount)

	groupKeys, err := generateGroups(ctx, spannerClient, features)
	if err != nil {
		return fmt.Errorf("group generation failed %w", err)
	}
	slog.Info("groups generated",
		"groupKeys", groupKeys)

	snapshotKeys, err := generateSnapshots(ctx, spannerClient, features)
	if err != nil {
		return fmt.Errorf("snapshot generation failed %w", err)
	}
	slog.Info("snapshots generated",
		"snapshotKeys", snapshotKeys)

	return nil
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
	for i, feature := range features {
		ChromiumHistogramEnumValueEntry := gcpspanner.ChromiumHistogramEnumValue{
			ChromiumHistogramEnumID: chromiumHistogramEnumIDMap["WebDXFeatureObserver"],
			BucketID:                int64(i + 1),
			Label:                   feature.FeatureKey,
		}
		enumValueID, err := client.UpsertChromiumHistogramEnumValue(ctx, ChromiumHistogramEnumValueEntry)
		if err != nil {
			return nil, err
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
	for _, feature := range features {
		webFeatureChromiumHistogramEnumValueEntry := gcpspanner.WebFeatureChromiumHistogramEnumValue{
			WebFeatureID:                 webFeatureKeyToInternalFeatureID[feature.FeatureKey],
			ChromiumHistogramEnumValueID: chromiumHistogramEnumValueToIDMap[feature.FeatureKey],
		}
		err := client.UpsertWebFeatureChromiumHistogramEnumValue(
			ctx,
			webFeatureChromiumHistogramEnumValueEntry,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func generateChromiumHistogramMetrics(
	ctx context.Context, client *gcpspanner.Client, features []gcpspanner.SpannerWebFeature) error {
	for i := range len(features) {
		currDate := startTimeWindow
		for currDate.After(time.Date(2020, time.December, 1, 0, 0, 0, 0, time.UTC)) {
			usage := big.NewRat(r.Int63n(100), 100) // Generate usage between 0-100%
			err := client.UpsertDailyChromiumHistogramMetric(
				ctx,
				metricdatatypes.WebDXFeatureEnum,
				int64(i+1),
				gcpspanner.DailyChromiumHistogramMetric{
					Day:  civil.DateOf(currDate),
					Rate: *usage,
				},
			)
			if err != nil {
				return err
			}
			currDate = currDate.AddDate(0, 0, r.Intn(23)+7) // Add up to a month, increasing by at least 1.
		}
	}

	return nil
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
	)
	flag.Parse()

	slog.Info("establishing spanner client",
		"project", *spannerProject,
		"instance", *spannerInstance,
		"database", *spannerDatabase)

	spannerClient, err := gcpspanner.NewSpannerClient(*spannerProject, *spannerInstance, *spannerDatabase)
	if err != nil {
		slog.Error("unable to create spanner client", "error", err)
		os.Exit(1)
	}

	slog.Info("establishing datastore client",
		"project", *datastoreProject,
		"database", *datastoreDatabase)

	datastoreClient, err := gds.NewDatastoreClient(*datastoreProject, datastoreDatabase)
	if err != nil {
		slog.Error("unable to create datastore client", "error", err)
		os.Exit(1)
	}

	gofakeit.GlobalFaker = gofakeit.New(seedValue)

	ctx := context.Background()

	err = generateData(ctx, spannerClient, datastoreClient)
	if err != nil {
		slog.Error("unable to generate data", "error", err)
		os.Exit(1)
	}
	slog.Info("loading fake data successful")
}
