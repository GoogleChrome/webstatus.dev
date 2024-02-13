package main

import (
	"cmp"
	"log/slog"
	"os"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gds"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/wpt_consumer/pkg/httpserver"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/wpt_consumer/pkg/workflow"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/wpt_consumer/pkg/wptfyi"
	"github.com/google/go-github/v47/github"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

func main() {
	wptFyiHostname := cmp.Or[string](os.Getenv("WPTFYI_HOSTNAME"), "wpt.fyi")
	var datastoreDB *string
	if value, found := os.LookupEnv("DATASTORE_DATABASE"); found {
		datastoreDB = &value
	}
	dsClient, err := gds.NewDatastoreClient(os.Getenv("PROJECT_ID"), datastoreDB)
	if err != nil {
		slog.Error("failed to create datastore client", "error", err.Error())
		os.Exit(1)
	}

	ghClient := github.NewClient(nil)
	workflow := workflow.Entrypoint{
		Starter: workflow.NewWptRunsWorker(
			wptfyi.NewHTTPClient(wptFyiHostname),
			workflow.NewWPTRunsProcessor(workflow.NewWPTRunProcessor(
				workflow.NewHTTPResultsGetter(),
				workflow.NewGitHubWebFeaturesDataGetter(
					shared.NewGitHubWebFeaturesClient(ghClient),
				),
				workflow.WPTScorerForWebFeatures{},
				dsClient,
			)),
		),
		NumWorkers: 1,
	}

	srv, err := httpserver.NewHTTPServer(
		"8080",
		workflow,
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
