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

	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

// NewJobArguments constructor to create JobArguments, encapsulating essential workflow parameters.
func NewJobArguments(histograms []metricdatatypes.HistogramName) JobArguments {
	return JobArguments{
		histograms: histograms,
	}
}

type JobArguments struct {
	histograms []metricdatatypes.HistogramName
}

// NewChromiumHistogramEnumsJobProcessor constructs a ChromiumHistogramEnumsJobProcessor.
func NewChromiumHistogramEnumsJobProcessor(
	enumsFetecher EnumsFetecher,
	enumsParser EnumsParser,
	histogramStorer HistogramStorer,
) ChromiumHistogramEnumsJobProcessor {
	return ChromiumHistogramEnumsJobProcessor{
		enumsFetecher:   enumsFetecher,
		enumsParser:     enumsParser,
		histogramStorer: histogramStorer,
	}
}

type EnumsFetecher interface {
	Fetch(context.Context) (io.ReadCloser, error)
}

type EnumsParser interface {
	Parse(context.Context, io.ReadCloser, []metricdatatypes.HistogramName) (metricdatatypes.HistogramMapping, error)
}

// HistogramStorer represents the behavior to the storage layer.
type HistogramStorer interface {
	SaveHistogramEnums(context.Context, metricdatatypes.HistogramMapping) error
}

type ChromiumHistogramEnumsJobProcessor struct {
	enumsFetecher   EnumsFetecher
	enumsParser     EnumsParser
	histogramStorer HistogramStorer
}

func (p ChromiumHistogramEnumsJobProcessor) Process(ctx context.Context, job JobArguments) error {
	// Step 1. Fetch enums
	rawData, err := p.enumsFetecher.Fetch(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "unable to fetch enums", "error", err)

		return err
	}

	// Step 2. Parse enums
	data, err := p.enumsParser.Parse(ctx, rawData, job.histograms)
	if err != nil {
		slog.ErrorContext(ctx, "unable to parse enums response", "error", err)

		return err
	}

	// Step 3. Save histogram enums
	err = p.histogramStorer.SaveHistogramEnums(ctx, data)
	if err != nil {
		slog.ErrorContext(ctx, "unable to save enums", "error", err)

		return err
	}

	return nil
}
