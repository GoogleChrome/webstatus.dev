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
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters"
	"github.com/GoogleChrome/webstatus.dev/lib/gds"
	"github.com/GoogleChrome/webstatus.dev/lib/gds/datastoreadapters"
	"github.com/GoogleChrome/webstatus.dev/lib/gh"
	"github.com/GoogleChrome/webstatus.dev/lib/workerpool"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/web_feature_consumer/pkg/data"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/web_feature_consumer/pkg/workflow"
)

const (
	defaultRepoOwner        = "web-platform-dx"
	defaultRepoName         = "web-features"
	defaultReleaseAssetName = "data.json"
)

func main() {
	ctx := context.Background()

	// Configuration and Client Setup

	var datastoreDB *string
	if value, found := os.LookupEnv("DATASTORE_DATABASE"); found {
		datastoreDB = &value
	}
	fs, err := gds.NewDatastoreClient(os.Getenv("PROJECT_ID"), datastoreDB)
	if err != nil {
		slog.Error("failed to create datastore client", "error", err.Error())
		os.Exit(1)
	}

	projectID := os.Getenv("PROJECT_ID")
	spannerDB := os.Getenv("SPANNER_DATABASE")
	spannerInstance := os.Getenv("SPANNER_INSTANCE")
	spannerClient, err := gcpspanner.NewSpannerClient(projectID, spannerInstance, spannerDB)
	if err != nil {
		slog.Error("failed to create spanner client", "error", err.Error())
		os.Exit(1)
	}

	// Will be empty if not set and that is okay.
	token := os.Getenv("GITHUB_TOKEN")

	dataWindowDuration := os.Getenv("DATA_WINDOW_DURATION")
	duration, err := time.ParseDuration(dataWindowDuration)
	if err != nil {
		slog.Error("unable to parse DATA_WINDOW_DURATION duration", "input value", dataWindowDuration)
		os.Exit(1)
	}
	endAt := time.Now().UTC()
	startAt := endAt.Add(-duration)

	// Currently, only one worker needed
	numWorkers := 1

	repoName := cmp.Or(os.Getenv("REPO_NAME"), defaultRepoName)

	repoOwner := cmp.Or(os.Getenv("REPO_OWNER"), defaultRepoOwner)

	releaseAssetName := cmp.Or(os.Getenv("RELEASE_ASSET_NAME"), defaultReleaseAssetName)

	// Worker Pool Setup
	pool := workerpool.Pool[workflow.JobArguments]{}

	processor := workflow.NewWebFeaturesJobProcessor(
		gh.NewClient(token),
		spanneradapters.NewWebFeaturesConsumer(spannerClient),
		datastoreadapters.NewWebFeaturesConsumer(fs),
		spanneradapters.NewWebFeatureGroupsConsumer(spannerClient),
		spanneradapters.NewWebFeatureSnapshotsConsumer(spannerClient),
		data.Parser{},
	)

	// Job Generation
	jobs := []workflow.JobArguments{
		workflow.NewJobArguments(
			releaseAssetName,
			repoOwner,
			repoName,
			startAt,
			endAt,
		),
	}
	// Job Execution and Error Handling
	errs := pool.Start(ctx, numWorkers, processor, jobs)
	if len(errs) > 0 {
		slog.ErrorContext(ctx, "workflow returned errors", "error", errs)
		os.Exit(1)
	}
}
