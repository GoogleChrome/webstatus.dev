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

package workflow

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

// NewWptRunsWorker constructs a WptRunsWorker, initializing it with a WPTJobProcessor and
// the provided dependencies for getting and processing runs.
func NewWptRunsWorker(runsGetter RunsGetter, runsProcessor RunsProcessor) *WptRunsWorker {
	return &WptRunsWorker{
		jobProcessor: WPTJobProcessor{
			runsGetter:    runsGetter,
			runsProcessor: runsProcessor,
		},
	}
}

type WptRunsWorker struct {
	// Handles the processing of individual jobs
	jobProcessor JobProcessor
}

// JobProcessor defines the contract for processing a single job within the WPT workflow.
type JobProcessor interface {
	Process(
		ctx context.Context,
		job JobArguments) error
}

type WPTJobProcessor struct {
	runsGetter    RunsGetter    // Dependency for fetching runs from the WPT system
	runsProcessor RunsProcessor // Dependency for processing fetched runs
}

// NewJobArguments constructor to create JobArguments, encapsulating essential workflow parameters.
func NewJobArguments(from time.Time, browser, channel string, pageSize int) JobArguments {
	return JobArguments{
		from:     from,
		browser:  browser,
		channel:  channel,
		pageSize: pageSize,
	}
}

type JobArguments struct {
	from     time.Time // Start date for fetching runs
	pageSize int       // Number of runs to fetch per request
	browser  string    // Browser to filter runs by
	channel  string    // Channel to filter runs by
}

type RunsProcessor interface {
	ProcessRuns(context.Context, shared.TestRuns) error
}

// RunsGetter represents the behavior to get all the runs up until the given
// date.
type RunsGetter interface {
	GetRuns(
		ctx context.Context,
		from time.Time,
		runsPerPage int,
		browserName string,
		channelName string,
	) (shared.TestRuns, error)
}

func (w WptRunsWorker) Work(
	ctx context.Context, id int, wg *sync.WaitGroup, jobs <-chan JobArguments, errChan chan<- error) {
	slog.Info("starting worker", "worker id", id)
	defer wg.Done()

	// Processes jobs received on the 'jobs' channel
	for job := range jobs {
		err := w.jobProcessor.Process(ctx, job)
		if err != nil {
			errChan <- err
		}
	}
	// Do not close the shared error channel here.
	// It will prevent others from returning their errors.
}

func (w WPTJobProcessor) Process(
	ctx context.Context,
	job JobArguments) error {

	// 1. Fetch runs using the provided job arguments
	runs, err := w.runsGetter.GetRuns(ctx, job.from, job.pageSize, job.browser, job.channel)
	if err != nil {
		return err
	}

	// 2. Process fetched runs
	err = w.runsProcessor.ProcessRuns(ctx, runs)
	if err != nil {
		return err
	}

	return nil // Indicates successful processing
}
