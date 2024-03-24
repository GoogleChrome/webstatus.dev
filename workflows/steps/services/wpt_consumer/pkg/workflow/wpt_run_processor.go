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
	"time"

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

type ResultsDownloader interface {
	DownloadResults(context.Context, string) (ResultsSummaryFile, error)
}

type WebFeaturesDataGetter interface {
	// Get the web features metadata for the particular commit sha.
	GetWebFeaturesData(context.Context, string) (*shared.WebFeaturesData, error)
}

type WebFeatureWPTScorer interface {
	Score(context.Context, ResultsSummaryFile, *shared.WebFeaturesData) map[string]WPTFeatureMetric
}

type WPTRun struct {
	ID               int64
	BrowserName      string
	BrowserVersion   string
	TimeStart        time.Time
	TimeEnd          time.Time
	Channel          string
	OSName           string
	OSVersion        string
	FullRevisionHash string
}

func NewWPTRun(testRun shared.TestRun) WPTRun {
	return WPTRun{
		ID:               testRun.ID,
		BrowserName:      testRun.BrowserName,
		BrowserVersion:   testRun.BrowserVersion,
		TimeStart:        testRun.TimeStart,
		TimeEnd:          testRun.TimeEnd,
		Channel:          testRun.Channel(),
		OSName:           testRun.OSName,
		OSVersion:        testRun.OSVersion,
		FullRevisionHash: testRun.FullRevisionHash,
	}
}

type WebFeatureWPTScoreStorer interface {
	InsertWPTRun(context.Context, WPTRun) error
	UpsertWPTRunFeatureMetric(
		context.Context,
		int64,
		map[string]WPTFeatureMetric) error
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

	err = w.scoreStorer.InsertWPTRun(ctx, NewWPTRun(run))
	if err != nil {
		return err
	}

	err = w.scoreStorer.UpsertWPTRunFeatureMetric(ctx, run.ID, metricsPerFeature)
	if err != nil {
		return err
	}

	return nil
}
