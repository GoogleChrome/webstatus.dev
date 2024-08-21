package workflow

import (
	"context"
	"io"
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

type ChromiumHistogramEnumsWorker struct {
	// Handles the processing of individual jobs
	jobProcessor JobProcessor
}

// NewJobArguments constructor to create JobArguments, encapsulating essential workflow parameters.
func NewJobArguments(histograms []metricdatatypes.HistogramName) JobArguments {
	return JobArguments{
		histograms: histograms,
	}
}

type JobArguments struct {
	histograms []metricdatatypes.HistogramName
}

// JobProcessor defines the contract for processing a single job within the Chromium Histogram Enum workflow.
type JobProcessor interface {
	Process(
		ctx context.Context,
		job JobArguments) error
}

// NewChromiumHistogramEnumsWorker constructs a ChromiumHistogramEnumsWorker, initializing it with a
// ChromiumHistogramEnumsJobProcessor and the provided dependencies for getting and processing metrics.
func NewChromiumHistogramEnumsWorker(
	enumsFetecher EnumsFetecher,
	enumsParser EnumsParser,
	histogramStorer HistogramStorer,
) *ChromiumHistogramEnumsWorker {
	return &ChromiumHistogramEnumsWorker{
		jobProcessor: ChromiumHistogramEnumsJobProcessor{
			enumsFetecher:   enumsFetecher,
			enumsParser:     enumsParser,
			histogramStorer: histogramStorer,
		},
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
	slog.Info("debug", "data", data)

	// Step 3. Save histogram enums
	// err = p.histogramStorer.SaveHistogramEnums(ctx, data)
	// if err != nil {
	// 	slog.ErrorContext(ctx, "unable to save enums", "error", err)

	// 	return err
	// }

	return nil
}
