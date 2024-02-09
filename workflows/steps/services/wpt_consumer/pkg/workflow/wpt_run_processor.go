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

type ResultsDownloader interface {
	DownloadResults(context.Context, string) (ResultsSummaryFile, error)
}

type WebFeaturesDataGetter interface {
	GetWebFeaturesData(context.Context) (shared.WebFeaturesData, error)
}

type WebFeatureWPTScorer interface {
	Score(context.Context, ResultsSummaryFile, shared.WebFeaturesData)
}

type WebFeatureWPTScoreStorer interface {
	Store(context.Context) error
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
	// TODO: in the future, get the matching metadata if it exist. Then default to
	//      the latest if it doesn't exist.
	webFeaturesData, err := w.webFeaturesDataGetter.GetWebFeaturesData(ctx)
	if err != nil {
		return err
	}
	w.scorer.Score(ctx, resultsSummaryFile, webFeaturesData)

	return w.scoreStorer.Store(ctx)
}
