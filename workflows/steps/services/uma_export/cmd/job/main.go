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
	"log/slog"
	"os"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/workerpool"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/uma_export/workflow"
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
		slog.ErrorContext(ctx, "failed to create spanner client", "error", err.Error())
		os.Exit(1)
	}
	_ = spannerClient
	numWorkers := 1
	// Worker Pool Setup
	pool := workerpool.Pool[workflow.JobArguments]{}

	fetcher, err := workflow.NewHTTPMetricsFetcher(workflow.GCPTokenGenerator{})
	if err != nil {
		slog.ErrorContext(ctx, "failed to create metrics fetcher", "error", err.Error())
		os.Exit(1)
	}
	worker := workflow.NewUMAExportWorker(nil, fetcher, workflow.XSSIMetricsParser{})

	// Job Generation
	jobChan := make(chan workflow.JobArguments)
	go func() {
		args := workflow.NewJobArguments(workflow.WebDXFeaturesQuery, time.Now().Add(-24*5*time.Hour))
		slog.InfoContext(ctx, "sending args to worker pool", "args", args)
		jobChan <- args
		// Close the job channel now that we are done.
		close(jobChan)
	}()

	// Job Execution and Error Handling
	errs := pool.Start(ctx, jobChan, numWorkers, worker)
	if len(errs) > 0 {
		slog.ErrorContext(ctx, "workflow returned errors", "error", errs)
		os.Exit(1)
	}
}
