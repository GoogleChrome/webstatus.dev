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
	"github.com/GoogleChrome/webstatus.dev/lib/wptfyi"
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

	ghClient := github.NewClient(nil)
	numWorkers := 8
	w := workflow.Entrypoint{
		Starter: workflow.NewRunsWorkerManager(
			workflow.NewWptRunsWorker(
				wptfyi.NewHTTPClient(wptFyiHostname, wptPageLimit),
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
		NumWorkers: numWorkers,
	}
	ctx := context.Background()
	// For now only go a year back by default.

	dataWindowDuration := os.Getenv("DATA_WINDOW_DURATION")
	duration, err := time.ParseDuration(dataWindowDuration)
	if err != nil {
		slog.Error("unable to parse DATA_WINDOW_DURATION duration", "input value", dataWindowDuration)
		os.Exit(1)
	}
	startAt := time.Now().UTC().Add(-duration)
	slog.Info("starting wpt workflow",
		"time window", startAt.String(),
		"workers", numWorkers,
		"wpt.fyi hostname", wptFyiHostname,
		"wpt page limit", wptPageLimit)
	errs := w.Start(ctx, startAt)
	if len(errs) > 0 {
		slog.Error("workflow returned errors", "error", errs)
		os.Exit(1)
	}
}
