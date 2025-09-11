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
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/wptconsumertypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gds"
	"github.com/GoogleChrome/webstatus.dev/lib/localcache"
	"github.com/GoogleChrome/webstatus.dev/lib/workerpool"
	"github.com/GoogleChrome/webstatus.dev/lib/wptfyi"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/wpt_consumer/pkg/workflow"
	"github.com/google/go-github/v74/github"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

func main() {
	ctx := context.Background()

	// Configuration and Client Setup

	// wpt.fyi setup
	wptFyiHostname := cmp.Or[string](os.Getenv("WPT_FYI_HOSTNAME"), "wpt.fyi")

	wptPageLimitStr := os.Getenv("WPT_FYI_PAGE_LIMIT")
	wptPageLimit := shared.MaxCountMaxValue
	if wptPageLimitStr != "" {
		var parseErr error
		wptPageLimit, parseErr = strconv.Atoi(wptPageLimitStr)
		if parseErr != nil {
			slog.ErrorContext(ctx, "unable to parse custom page limit", "input", wptPageLimitStr)
			os.Exit(1)
		}
	}

	// Datastore setup
	var datastoreDB *string
	if value, found := os.LookupEnv("DATASTORE_DATABASE"); found {
		datastoreDB = &value
	}
	projectID := os.Getenv("PROJECT_ID")
	dsClient, err := gds.NewDatastoreClient(projectID, datastoreDB)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create datastore client", "error", err.Error())
		os.Exit(1)
	}
	_ = dsClient

	// Spanner setup
	spannerDB := os.Getenv("SPANNER_DATABASE")
	spannerInstance := os.Getenv("SPANNER_INSTANCE")
	spannerClient, err := gcpspanner.NewSpannerClient(projectID, spannerInstance, spannerDB)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create spanner client", "error", err.Error())
		os.Exit(1)
	}

	ghClient := github.NewClient(nil)
	// Currently, keep it at 8. 4 browsers, 2 channels each.
	numWorkers := 8

	dataWindowDuration := os.Getenv("DATA_WINDOW_DURATION")
	duration, err := time.ParseDuration(dataWindowDuration)
	if err != nil {
		slog.ErrorContext(ctx, "unable to parse DATA_WINDOW_DURATION duration", "input value", dataWindowDuration)
		os.Exit(1)
	}

	// Worker Pool Setup
	pool := workerpool.Pool[workflow.JobArguments]{}

	webFeaturesDataCopier := func(in shared.WebFeaturesData) shared.WebFeaturesData {
		dataCopy := make(shared.WebFeaturesData, len(in))
		for testName, featuresMap := range in {
			newFeaturesMap := make(map[string]interface{}, len(featuresMap))
			for featureKey, featureData := range featuresMap {
				newFeaturesMap[featureKey] = featureData
			}
			dataCopy[testName] = newFeaturesMap
		}

		return dataCopy
	}

	processor := workflow.NewWPTJobProcessor(
		wptfyi.NewHTTPClient(wptFyiHostname),
		workflow.NewWPTRunsProcessor(
			workflow.NewWPTRunProcessor(
				workflow.NewHTTPResultsGetter(),
				workflow.NewCacheableWebFeaturesDataGetter(
					shared.NewGitHubWebFeaturesClient(ghClient),
					localcache.NewLocalDataCache[string, shared.WebFeaturesData](webFeaturesDataCopier),
				),
				spanneradapters.NewWPTWorkflowConsumer(spannerClient),
			),
		),
	)

	// Job Generation
	jobs := []workflow.JobArguments{}
	startAt := time.Now().UTC().Add(-duration)
	browsers := []string{
		string(wptconsumertypes.Chrome),
		string(wptconsumertypes.Edge),
		string(wptconsumertypes.Firefox),
		string(wptconsumertypes.Safari),
		string(wptconsumertypes.ChromeAndroid),
		string(wptconsumertypes.FirefoxAndroid),
	}
	channels := []string{shared.StableLabel, shared.ExperimentalLabel}
	for _, browser := range browsers {
		for _, channel := range channels {
			jobs = append(jobs, workflow.NewJobArguments(
				startAt,
				browser,
				channel,
				wptPageLimit,
			))
		}
	}

	// Job Execution and Error Handling
	errs := pool.Start(ctx, numWorkers, processor, jobs)
	if len(errs) > 0 {
		slog.ErrorContext(ctx, "workflow returned errors", "error", errs)
		os.Exit(1)
	}
}
