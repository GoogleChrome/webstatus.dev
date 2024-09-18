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
	"time"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

func NewWPTJobProcessor(runsGetter RunsGetter, runsProcessor RunsProcessor) WPTJobProcessor {
	return WPTJobProcessor{
		runsGetter:    runsGetter,
		runsProcessor: runsProcessor,
	}
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

func (w WPTJobProcessor) Process(
	ctx context.Context,
	job JobArguments) error {

	// 1. Fetch runs using the provided job arguments
	slog.InfoContext(ctx, "fetching runs", "browser", job.browser, "channel", job.channel)
	runs, err := w.runsGetter.GetRuns(ctx, job.from, job.pageSize, job.browser, job.channel)
	if err != nil {
		return err
	}

	// 2. Process fetched runs
	slog.InfoContext(ctx, "processing runs", "browser", job.browser, "channel", job.channel, "run_count", len(runs))
	err = w.runsProcessor.ProcessRuns(ctx, runs)
	if err != nil {
		return err
	}

	return nil // Indicates successful processing
}
