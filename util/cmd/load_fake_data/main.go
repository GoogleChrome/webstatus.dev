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
	"math/rand"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/web-platform-tests/wpt.fyi/shared"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const releasesPerBrowser = 50
const runsPerBrowserPerChannel = 100
const numberOfFeatures = 150

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

func generateFeatures(ctx context.Context, client *gcpspanner.Client) ([]gcpspanner.WebFeature, error) {
	features := make([]gcpspanner.WebFeature, 0, numberOfFeatures)
	featureIDMap := make(map[string]interface{})

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
			Name:      featureName,
			FeatureID: featureID,
		}
		err := client.UpsertWebFeature(ctx, feature)
		if err != nil {
			return nil, err
		}
		features = append(features, feature)
	}

	return features, nil
}

func generateFeatureAvailability(
	ctx context.Context,
	client *gcpspanner.Client,
	features []gcpspanner.WebFeature) (int, error) {
	availabilitiesInserted := 0
	for _, browser := range browsers {
		for _, feature := range features {
			// Add availability randomly at a 50% chance
			releaseVersion := r.Int31n(2*releasesPerBrowser) + 1
			if releaseVersion <= releasesPerBrowser {
				err := client.InsertBrowserFeatureAvailability(
					ctx,
					gcpspanner.BrowserFeatureAvailability{
						BrowserName:    browser,
						BrowserVersion: fmt.Sprintf("%d", releaseVersion),
						FeatureID:      feature.FeatureID,
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

func generateData(ctx context.Context, client *gcpspanner.Client) error {
	releasesCount, err := generateReleases(ctx, client)
	if err != nil {
		return fmt.Errorf("release generation failed %w", err)
	}
	slog.Info("releases generated",
		"amount of releases created", releasesCount)

	features, err := generateFeatures(ctx, client)
	if err != nil {
		return fmt.Errorf("feature generation failed %w", err)
	}
	slog.Info("features generated",
		"amount of features created", len(features))

	runsCount, metricsCount, err := generateRunsAndMetrics(ctx, client, features)
	if err != nil {
		return fmt.Errorf("wpt runs generation failed %w", err)
	}
	slog.Info("runs and metrics generated",
		"amount of runs created", runsCount, "amount of metrics created", metricsCount)

	statusCount, err := generateBaselineStatus(ctx, client, features)
	if err != nil {
		return fmt.Errorf("baseline status failed %w", err)
	}
	slog.Info("statuses generated",
		"amount of statuses created", statusCount)

	availabilityCount, err := generateFeatureAvailability(ctx, client, features)
	if err != nil {
		return fmt.Errorf("feature availability generation failed %w", err)
	}
	slog.Info("availabilities generated",
		"amount of availabilities created", availabilityCount)

	return nil
}

func generateBaselineStatus(
	ctx context.Context, client *gcpspanner.Client, features []gcpspanner.WebFeature) (int, error) {
	statusesGenerated := 0
	statuses := []gcpspanner.BaselineStatus{
		gcpspanner.BaselineStatusUndefined,
		gcpspanner.BaselineStatusNone,
		gcpspanner.BaselineStatusLow,
		gcpspanner.BaselineStatusHigh,
	}

	baseDate := startTimeWindow
	for _, feature := range features {
		statusIndex := r.Intn(len(statuses))
		var highDate *time.Time
		var lowDate *time.Time
		switch statuses[statusIndex] {
		case gcpspanner.BaselineStatusHigh:
			adjustedTime := baseDate.AddDate(0, 0, r.Intn(30)) // Add up to 1 month
			lowDate = &adjustedTime
			highAdjustedTime := adjustedTime.AddDate(0, 0, r.Intn(30)) // Add up to another month
			highDate = &highAdjustedTime
		case gcpspanner.BaselineStatusLow:
			adjustedTime := baseDate.AddDate(0, 0, r.Intn(30)) // Add up to 1 month
			lowDate = &adjustedTime
		case gcpspanner.BaselineStatusUndefined, gcpspanner.BaselineStatusNone:
			// Do nothing.
		}
		err := client.UpsertFeatureBaselineStatus(ctx, gcpspanner.FeatureBaselineStatus{
			FeatureID: feature.FeatureID,
			Status:    statuses[statusIndex],
			LowDate:   lowDate,
			HighDate:  highDate,
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
	ctx context.Context, client *gcpspanner.Client, features []gcpspanner.WebFeature) (int, int, error) {
	// For now only generate one run with metrics per browser+channel combination.
	// TODO. Need to think about the graphs we want to draw.
	runsGenerated := 0
	metricsGenerated := 0
	channels := []string{shared.StableLabel, shared.ExperimentalLabel}
	for _, channel := range channels {
		for _, browser := range browsers {
			baseTime := startTimeWindow
			for i := 0; i < runsPerBrowserPerChannel; i++ {
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
					metric := gcpspanner.WPTRunFeatureMetric{
						FeatureID:  feature.FeatureID,
						TotalTests: &testTotal,
						TestPass:   &testPass,
					}
					spannerMetric := client.CreateSpannerWPTRunFeatureMetric(*wptRunData, metric)
					m, err := spanner.InsertOrUpdateStruct(gcpspanner.WPTRunFeatureMetricTable, spannerMetric)
					if err != nil {
						return runsGenerated, metricsGenerated, err
					}
					mutations = append(mutations, m)
				}
				// BatchWrite is not implemented in the emulator.
				// https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/154
				// Instead, do Apply which does multiple statements atomically.
				// Revisit this once the emulator supports BatchWrite.
				_, err = client.Apply(ctx, mutations)
				if err != nil {
					return runsGenerated, metricsGenerated, err
				}
				metricsGenerated += len(mutations)
			}
		}
	}

	return runsGenerated, metricsGenerated, nil
}

func main() {
	// Use the grpc port from spanner in .dev/spanner/skaffold.yaml
	// Describe the command line flags and parse the flags
	var (
		spannerProject  = flag.String("spanner_project", "", "Spanner Project")
		spannerInstance = flag.String("spanner_instance", "", "Spanner Instance")
		spannerDatabase = flag.String("spanner_database", "", "Spanner Database")
	)
	flag.Parse()

	slog.Info("establishing spanner client",
		"project", *spannerProject,
		"instance", *spannerInstance,
		"database", *spannerDatabase)

	client, err := gcpspanner.NewSpannerClient(*spannerProject, *spannerInstance, *spannerDatabase)
	if err != nil {
		slog.Error("unable to create spanner client", "error", err)
		os.Exit(1)
	}

	gofakeit.GlobalFaker = gofakeit.New(seedValue)

	ctx := context.Background()

	err = generateData(ctx, client)
	if err != nil {
		slog.Error("unable to generate data", "error", err)
		os.Exit(1)
	}
	slog.Info("loading fake data successful")
}
