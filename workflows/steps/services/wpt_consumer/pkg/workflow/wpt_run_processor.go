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

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/wptconsumertypes"
	"github.com/web-platform-tests/wpt.fyi/api/query"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

// A copy of summary from wpt.fyi
// https://github.com/web-platform-tests/wpt.fyi/blob/05ddddc52a6b95469131eac5e439af39cbd1200a/api/query/query.go#L30
// TODO export Summary in wpt.fyi and use it here instead.
type ResultsSummaryFile map[string]query.SummaryResult

// WPTRunProcessor contains all the steps for the workflow to consume wpt data
// of a particular WPT Run.
type WPTRunProcessor struct {
	resultsDownloader     ResultsDownloader
	webFeaturesDataGetter WebFeaturesDataGetter
	scorer                WebFeatureWPTScorer
	scoreStorer           WebFeatureWPTScoreStorer
}

// NewWPTRunProcessor constructs a WPTRunProcessor.
func NewWPTRunProcessor(
	resultsDownloader ResultsDownloader,
	webFeaturesDataGetter WebFeaturesDataGetter,
	scorer WebFeatureWPTScorer,
	scoreStorer WebFeatureWPTScoreStorer) *WPTRunProcessor {
	return &WPTRunProcessor{
		resultsDownloader:     resultsDownloader,
		webFeaturesDataGetter: webFeaturesDataGetter,
		scorer:                scorer,
		scoreStorer:           scoreStorer,
	}
}

// ResultsDownloader will download the results for a given run.
// The url to download the results comes from the API to get runs.
type ResultsDownloader interface {
	DownloadResults(context.Context, string) (ResultsSummaryFile, error)
}

// WebFeaturesDataGetter describes an interface that will get the web features data.
type WebFeaturesDataGetter interface {
	// Get the web features metadata for the particular commit sha.
	GetWebFeaturesData(context.Context, string) (*shared.WebFeaturesData, error)
}

// WebFeatureWPTScorer describes an interface that will score the run with the given features data.
type WebFeatureWPTScorer interface {
	Score(context.Context, ResultsSummaryFile, *shared.WebFeaturesData) map[string]wptconsumertypes.WPTFeatureMetric
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
	metricsPerFeature := w.scorer.Score(ctx, resultsSummaryFile, webFeaturesData)

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
