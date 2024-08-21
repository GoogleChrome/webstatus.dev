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
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

// Heavily inspired by https://github.com/GoogleChrome/chromium-dashboard/blob/main/internals/fetchmetrics.py

type UMAExportWorker struct {
	// Handles the processing of individual jobs
	jobProcessor JobProcessor
}

// NewUMAExportWorker constructs a UMAExportWorker, initializing it with a UMAExportJobProcessor and
// the provided dependencies for getting and processing metrics.
func NewUMAExportWorker(
	metricStorer MetricStorer, metricFetcher MetricFetecher, metricParser MetricParser) *UMAExportWorker {
	return &UMAExportWorker{
		jobProcessor: UMAExportJobProcessor{
			metricStorer:  metricStorer,
			metricFetcher: metricFetcher,
			metricsParser: metricParser,
		},
	}
}

func (w UMAExportWorker) Work(
	ctx context.Context, id int, wg *sync.WaitGroup, jobs <-chan JobArguments, errChan chan<- error) {
	slog.InfoContext(ctx, "starting worker", "worker id", id)
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

// NewJobArguments constructor to create JobArguments, encapsulating essential workflow parameters.
func NewJobArguments(queryName UMAExportQuery, day time.Time) JobArguments {
	return JobArguments{
		queryName: queryName,
		day:       day,
	}
}

// JobProcessor defines the contract for processing a single job within the UMA Export workflow.
type JobProcessor interface {
	Process(
		ctx context.Context,
		job JobArguments) error
}

type UMAExportQuery string

const (
	WebDXFeaturesQuery UMAExportQuery = "usecounter.webdxfeatures"
)

type JobArguments struct {
	queryName UMAExportQuery
	day       time.Time
}

// MetricStorer represents the behavior to the storage layer.
type MetricStorer interface {
	HasCapstone(context.Context, time.Time) (bool, error)
	SaveCapstone(context.Context) error
	SaveMetrics(context.Context, metricdatatypes.BucketDataMetrics) error
}

type MetricFetecher interface {
	Fetch(context.Context, UMAExportQuery) (io.ReadCloser, error)
}

type MetricParser interface {
	Parse(context.Context, io.ReadCloser) (metricdatatypes.BucketDataMetrics, error)
}

type UMAExportJobProcessor struct {
	metricStorer  MetricStorer
	metricFetcher MetricFetecher
	metricsParser MetricParser
}

func (p UMAExportJobProcessor) Process(ctx context.Context, job JobArguments) error {
	// Step 1. Check if already processed.
	// found, err := p.metricStorer.HasCapstone(ctx, job.day)
	// if err != nil {
	// 	slog.ErrorContext(ctx, "unable to parse metrics file", "error", err)

	// 	return err
	// }
	// if found {
	// 	slog.InfoContext(ctx, "Found existing capstone entry", "date", job.day)

	// 	return nil
	// }
	slog.InfoContext(ctx, "No capstone entry found. Will fetch", "date", job.day)

	// Step 2. Fetch results
	rawData, err := p.metricFetcher.Fetch(ctx, job.queryName)
	if err != nil {
		slog.ErrorContext(ctx, "unable to fetch metrics", "error", err)

		return err
	}

	slog.InfoContext(ctx, "debug after fetch", "rawdata nil?", rawData == nil)

	// Step 3. Parse the data.
	data, err := p.metricsParser.Parse(ctx, rawData)
	if err != nil {
		slog.ErrorContext(ctx, "unable to parse metrics response", "error", err)

		return err
	}

	slog.InfoContext(ctx, "debug", "data", data)

	// Step 4. Save the data.
	// err = p.metricStorer.SaveMetrics(ctx, data)
	// if err != nil {
	// 	slog.ErrorContext(ctx, "unable to save the metrics", "error", err)

	// 	return err
	// }

	return nil
}
