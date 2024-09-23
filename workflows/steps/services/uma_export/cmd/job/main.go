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

	"cloud.google.com/go/civil"
	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
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
	numWorkers := 1
	// Worker Pool Setup
	pool := workerpool.Pool[workflow.JobArguments]{}

	fetcher, err := workflow.NewHTTPMetricsFetcher(auth.GCPTokenGenerator{})
	if err != nil {
		slog.ErrorContext(ctx, "failed to create metrics fetcher", "error", err.Error())
		os.Exit(1)
	}
	processor := workflow.NewUMAExportJobProcessor(
		spanneradapters.NewUMAMetricConsumer(spannerClient),
		fetcher,
		workflow.XSSIMetricsParser{})

	// Job Generation
	dates, err := generateDateJobs(ctx, os.Getenv("DATE"), os.Getenv("TODAY"))
	if err != nil {
		slog.ErrorContext(ctx, "failed generate arguments for jobs. exiting", "error", err)
		os.Exit(1)
	}
	jobs := []workflow.JobArguments{}
	for _, date := range dates {
		jobs = append(jobs, workflow.NewJobArguments(
			metricdatatypes.WebDXFeaturesQuery,
			date,
			metricdatatypes.WebDXFeatureEnum,
		))
	}

	// Job Execution and Error Handling
	errs := pool.Start(ctx, numWorkers, processor, jobs)
	if len(errs) > 0 {
		slog.ErrorContext(ctx, "workflow returned errors", "error", errs)
		os.Exit(1)
	}
}

// nolint:lll // WONTFIX. Permalink to code section
/*
Inspired by:
https://github.com/GoogleChrome/chromium-dashboard/blob/d82111860aaf8bfa6949b13a00a32035684af2ba/internals/fetchmetrics.py#L189-L214
*/

func generateDateJobs(ctx context.Context, dateStr, todayStr string) ([]civil.Date, error) {
	var dates []civil.Date
	// Case: Allow users to specify an exact date.
	if dateStr != "" {
		date, err := civil.ParseDate(dateStr)
		if err != nil {
			slog.ErrorContext(ctx, "failed to parse input DATE", "input", dateStr, "error", err)

			return nil, err
		}
		dates = append(dates, date)

		return dates, nil
	}

	// Case: Optionally allow users to allow users to specify a starting day and go back 5 days.
	startingDate := civil.DateOf(time.Now())
	if todayStr != "" {
		todayDate, err := civil.ParseDate(todayStr)
		if err != nil {
			slog.WarnContext(ctx, "failed to parse input TODAY. Will use default", "input", todayStr, "error", err)
		} else {
			startingDate = todayDate
		}
	}
	for i := 1; i <= 5; i++ {
		dates = append(dates, startingDate.AddDays(-i))
	}

	return dates, nil
}
