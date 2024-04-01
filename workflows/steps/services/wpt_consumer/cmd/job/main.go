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
	"github.com/GoogleChrome/webstatus.dev/lib/gds"
	"github.com/GoogleChrome/webstatus.dev/lib/localcache"
	"github.com/GoogleChrome/webstatus.dev/lib/workerpool"
	"github.com/GoogleChrome/webstatus.dev/lib/wptfyi"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/wpt_consumer/pkg/workflow"
	"github.com/google/go-github/v47/github"
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
			slog.Error("unable to parse custom page limit", "input", wptPageLimitStr)
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
		slog.Error("failed to create datastore client", "error", err.Error())
		os.Exit(1)
	}
	_ = dsClient

	// Spanner setup
	spannerDB := os.Getenv("SPANNER_DATABASE")
	spannerInstance := os.Getenv("SPANNER_INSTANCE")
	spannerClient, err := gcpspanner.NewSpannerClient(projectID, spannerInstance, spannerDB)
	if err != nil {
		slog.Error("failed to create spanner client", "error", err.Error())
		os.Exit(1)
	}

	ghClient := github.NewClient(nil)
	// Currently, keep it at 8. 4 browsers, 2 channels each.
	numWorkers := 8

	dataWindowDuration := os.Getenv("DATA_WINDOW_DURATION")
	duration, err := time.ParseDuration(dataWindowDuration)
	if err != nil {
		slog.Error("unable to parse DATA_WINDOW_DURATION duration", "input value", dataWindowDuration)
		os.Exit(1)
	}

	// Worker Pool Setup
	pool := workerpool.Pool[workflow.JobArguments]{}

	worker := workflow.NewWptRunsWorker(
		wptfyi.NewHTTPClient(wptFyiHostname),
		workflow.NewWPTRunsProcessor(
			workflow.NewWPTRunProcessor(
				workflow.NewHTTPResultsGetter(),
				workflow.NewCacheableWebFeaturesDataGetter(
					shared.NewGitHubWebFeaturesClient(ghClient),
					localcache.NewLocalDataCache[string, shared.WebFeaturesData](),
				),
				spanneradapters.NewWPTWorkflowConsumer(spannerClient),
			),
		),
	)

	// Job Generation
	jobChan := make(chan workflow.JobArguments)
	go func() {
		startAt := time.Now().UTC().Add(-duration)
		browsers := shared.GetDefaultBrowserNames()
		channels := []string{shared.StableLabel, shared.ExperimentalLabel}
		for _, browser := range browsers {
			for _, channel := range channels {
				args := workflow.NewJobArguments(
					startAt,
					browser,
					channel,
					wptPageLimit,
				)
				slog.Info("sending args to worker pool", "args", args)
				jobChan <- args
			}
		}
		// Close the job channel now that we are done.
		close(jobChan)
	}()

	// Job Execution and Error Handling
	errs := pool.Start(ctx, jobChan, numWorkers, worker)
	if len(errs) > 0 {
		slog.Error("workflow returned errors", "error", errs)
		os.Exit(1)
	}
}
