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
	"log/slog"
	"os"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters"
	"github.com/GoogleChrome/webstatus.dev/lib/gds"
	"github.com/GoogleChrome/webstatus.dev/lib/localcache"
	"github.com/GoogleChrome/webstatus.dev/lib/wptfyi"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/wpt_consumer/pkg/httpserver"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/wpt_consumer/pkg/workflow"
	"github.com/google/go-github/v47/github"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

func main() {
	wptFyiHostname := cmp.Or[string](os.Getenv("WPTFYI_HOSTNAME"), "wpt.fyi")
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

	spannerDB := os.Getenv("SPANNER_DATABASE")
	spannerInstance := os.Getenv("SPANNER_INSTANCE")
	spannerClient, err := gcpspanner.NewSpannerClient(projectID, spannerInstance, spannerDB)
	if err != nil {
		slog.Error("failed to create spanner client", "error", err.Error())
		os.Exit(1)
	}

	ghClient := github.NewClient(nil)

	w := workflow.Entrypoint{
		Starter: workflow.NewRunsWorkerManager(
			workflow.NewWptRunsWorker(
				wptfyi.NewHTTPClient(wptFyiHostname),
				workflow.NewWPTRunsProcessor(
					workflow.NewWPTRunProcessor(
						workflow.NewHTTPResultsGetter(),
						workflow.NewCacheableWebFeaturesDataGetter(
							shared.NewGitHubWebFeaturesClient(ghClient),
							localcache.NewLocalDataCache[shared.WebFeaturesData](),
						),
						workflow.WPTScorerForWebFeatures{},
						spanneradapters.NewWPTWorkflowConsumer(spannerClient),
					),
				),
			),
		),
		NumWorkers: 1,
	}

	srv, err := httpserver.NewHTTPServer(
		"8080",
		w,
		// For now only go a year back by default.
		time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		slog.Error("unable to create server", "error", err.Error())
		os.Exit(1)
	}
	err = srv.ListenAndServe()
	if err != nil {
		slog.Error("unable to start server", "error", err.Error())
		os.Exit(1)
	}
}
