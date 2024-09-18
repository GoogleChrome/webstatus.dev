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

	"cloud.google.com/go/civil"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

// Heavily inspired by https://github.com/GoogleChrome/chromium-dashboard/blob/main/internals/fetchmetrics.py

// NewUMAExportJobProcessor constructs a UMAExportJobProcessor.
func NewUMAExportJobProcessor(
	metricStorer MetricStorer, metricFetcher MetricFetecher, metricParser MetricParser) UMAExportJobProcessor {
	return UMAExportJobProcessor{
		metricStorer:  metricStorer,
		metricFetcher: metricFetcher,
		metricsParser: metricParser,
	}
}

// NewJobArguments constructor to create JobArguments, encapsulating essential workflow parameters.
func NewJobArguments(
	queryName metricdatatypes.UMAExportQuery,
	day civil.Date,
	histogramName metricdatatypes.HistogramName) JobArguments {
	return JobArguments{
		queryName:     queryName,
		day:           day,
		histogramName: histogramName,
	}
}

type JobArguments struct {
	queryName     metricdatatypes.UMAExportQuery
	day           civil.Date
	histogramName metricdatatypes.HistogramName
}

// MetricStorer represents the behavior to the storage layer.
type MetricStorer interface {
	HasCapstone(context.Context, civil.Date, metricdatatypes.HistogramName) (bool, error)
	SaveCapstone(context.Context, civil.Date, metricdatatypes.HistogramName) error
	SaveMetrics(context.Context, civil.Date, metricdatatypes.BucketDataMetrics) error
}

type MetricFetecher interface {
	Fetch(context.Context, metricdatatypes.UMAExportQuery) (io.ReadCloser, error)
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
	found, err := p.metricStorer.HasCapstone(ctx, job.day, job.histogramName)
	if err != nil {
		slog.ErrorContext(ctx, "unable to parse metrics file", "error", err)

		return err
	}
	if found {
		slog.InfoContext(ctx, "Found existing capstone entry", "date", job.day)

		return nil
	}
	slog.InfoContext(ctx, "No capstone entry found. Will fetch", "date", job.day)

	// Step 2. Fetch results
	rawData, err := p.metricFetcher.Fetch(ctx, job.queryName)
	if err != nil {
		slog.ErrorContext(ctx, "unable to fetch metrics", "error", err)

		return err
	}

	// Step 3. Parse the data.
	data, err := p.metricsParser.Parse(ctx, rawData)
	if err != nil {
		slog.ErrorContext(ctx, "unable to parse metrics response", "error", err)

		return err
	}

	// Step 4. Save the data.
	err = p.metricStorer.SaveMetrics(ctx, job.day, data)
	if err != nil {
		slog.ErrorContext(ctx, "unable to save the metrics", "error", err)

		return err
	}

	// Step 5. Save the capstone.
	err = p.metricStorer.SaveCapstone(ctx, job.day, job.histogramName)
	if err != nil {
		slog.ErrorContext(ctx, "unable to save the capstone", "error", err)

		return err
	}

	return nil
}
