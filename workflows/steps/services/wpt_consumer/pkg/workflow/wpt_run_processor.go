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
	"strings"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/wptconsumertypes"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

// ResultsSummaryFile contains the results of a given file format.
type ResultsSummaryFile interface {
	Score(context.Context, *shared.WebFeaturesData) map[string]wptconsumertypes.WPTFeatureMetric
}

// WPTRunProcessor contains all the steps for the workflow to consume wpt data
// of a particular WPT Run.
type WPTRunProcessor struct {
	resultsDownloader     ResultsDownloader
	webFeaturesDataGetter WebFeaturesDataGetter
	scoreStorer           WebFeatureWPTScoreStorer
}

// NewWPTRunProcessor constructs a WPTRunProcessor.
func NewWPTRunProcessor(
	resultsDownloader ResultsDownloader,
	webFeaturesDataGetter WebFeaturesDataGetter,
	scoreStorer WebFeatureWPTScoreStorer) *WPTRunProcessor {
	return &WPTRunProcessor{
		resultsDownloader:     resultsDownloader,
		webFeaturesDataGetter: webFeaturesDataGetter,
		scoreStorer:           scoreStorer,
	}
}

// ResultsDownloader will download the results for a given run.
// The url to download the results comes from the API to get runs.
type ResultsDownloader interface {
	// Returns a small interface ResultsSummaryFile that is later used to generate metrics for each feature.
	// TODO. once we start parsing multiple file types, we can revisit not returning an interface.
	DownloadResults(context.Context, string) (ResultsSummaryFile, error)
}

// WebFeaturesDataGetter describes an interface that will get the web features data.
type WebFeaturesDataGetter interface {
	// Get the web features metadata for the particular commit sha.
	GetWebFeaturesData(context.Context, string) (shared.WebFeaturesData, error)
}

// WebFeatureWPTScoreStorer describes the interface to store run data and metrics data.
type WebFeatureWPTScoreStorer interface {
	InsertWPTRun(context.Context, shared.TestRun) error
	UpsertWPTRunFeatureMetrics(
		context.Context,
		int64,
		map[string]wptconsumertypes.WPTFeatureMetric) error
}

func (w WPTRunProcessor) ProcessRun(
	ctx context.Context,
	run shared.TestRun) error {

	if !strings.HasSuffix(run.ResultsURL, "summary_v2.json.gz") {
		slog.WarnContext(ctx, "can only process v2 summary runs. skipping...", "runID", run.ID, "resultsURL", run.ResultsURL)

		return nil
	}

	// Get the results.
	resultsSummaryFile, err := w.resultsDownloader.DownloadResults(ctx, run.ResultsURL)
	if err != nil {
		return err
	}

	// Get the web features data.
	webFeaturesData, err := w.webFeaturesDataGetter.GetWebFeaturesData(ctx, run.FullRevisionHash)
	if err != nil {
		return err
	}

	metricsPerFeature := resultsSummaryFile.Score(ctx, &webFeaturesData)

	// Insert the data.

	// Insert the wpt run data.
	err = w.scoreStorer.InsertWPTRun(ctx, run)
	if err != nil {
		return err
	}

	// Upsert the feature metrics for the given run.
	err = w.scoreStorer.UpsertWPTRunFeatureMetrics(ctx, run.ID, metricsPerFeature)
	if err != nil {
		return err
	}

	return nil
}
