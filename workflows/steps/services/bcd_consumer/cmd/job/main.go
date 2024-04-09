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

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/bcdconsumertypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gh"
	"github.com/GoogleChrome/webstatus.dev/lib/workerpool"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/bcd_consumer/pkg/data"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/bcd_consumer/pkg/workflow"
)

const (
	defaultRepoOwner        = "mdn"
	defaultRepoName         = "browser-compat-data"
	defaultReleaseAssetName = "data.json"
)

func main() {
	ctx := context.Background()

	// Configuration and Client Setup

	projectID := os.Getenv("PROJECT_ID")

	// Spanner setup
	spannerDB := os.Getenv("SPANNER_DATABASE")
	spannerInstance := os.Getenv("SPANNER_INSTANCE")
	spannerClient, err := gcpspanner.NewSpannerClient(projectID, spannerInstance, spannerDB)
	if err != nil {
		slog.Error("failed to create spanner client", "error", err.Error())
		os.Exit(1)
	}

	// Currently, only one worker needed
	numWorkers := 1

	repoName := cmp.Or(os.Getenv("REPO_NAME"), defaultRepoName)

	repoOwner := cmp.Or(os.Getenv("REPO_OWNER"), defaultRepoOwner)

	releaseAssetName := cmp.Or(os.Getenv("RELEASE_ASSET_NAME"), defaultReleaseAssetName)

	// Worker Pool Setup
	pool := workerpool.Pool[workflow.JobArguments]{}

	// Will be empty if not set and that is okay.
	token := os.Getenv("GITHUB_TOKEN")
	worker := workflow.NewBCDReleasesWorker(
		gh.NewClient(token),
		data.Parser{},
		workflow.BCDDataFilter{},
		spanneradapters.NewBCDWorkflowConsumer(spannerClient),
		repoOwner,
		repoName,
		releaseAssetName,
	)

	// Job Generation
	jobChan := make(chan workflow.JobArguments)
	go func() {
		args := workflow.NewJobArguments(
			[]string{
				string(bcdconsumertypes.Chrome),
				string(bcdconsumertypes.Edge),
				string(bcdconsumertypes.Firefox),
				string(bcdconsumertypes.Safari),
			},
		)
		slog.Info("sending args to worker pool", "args", args)
		jobChan <- args
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
