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
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/web-platform-tests/wpt.fyi/api/query"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

type MockResultsDownloader struct {
	shouldFail     bool
	resultsSummary ResultsSummaryFile
}

var errDownloadResults = errors.New("download results error")

func (m *MockResultsDownloader) DownloadResults(_ context.Context, _ string) (ResultsSummaryFile, error) {
	if m.shouldFail {
		return nil, errDownloadResults
	}

	return m.resultsSummary, nil
}

type MockWebFeaturesDataGetter struct {
	shouldFail      bool
	webFeaturesData *shared.WebFeaturesData
}

var errGetWebFeaturesData = errors.New("web features test error")

func (m *MockWebFeaturesDataGetter) GetWebFeaturesData(
	_ context.Context,
	_ string) (*shared.WebFeaturesData, error) {
	if m.shouldFail {
		return nil, errGetWebFeaturesData
	}

	return m.webFeaturesData, nil
}

type MockWebFeatureWPTScorer struct {
	metricsPerFeature map[string]WPTFeatureMetric
}

func (m *MockWebFeatureWPTScorer) Score(
	_ context.Context,
	_ ResultsSummaryFile,
	_ *shared.WebFeaturesData) map[string]WPTFeatureMetric {
	return m.metricsPerFeature
}

type insertRunConfig struct {
	run WPTRun
	err error
}

type upsertMetricConfig struct {
	runID             int64
	metricsPerFeature map[string]WPTFeatureMetric
	err               error
}

type MockWebFeatureWPTScoreStorer struct {
	insertRunCfg    *insertRunConfig
	upsertMetricCfg *upsertMetricConfig
	t               *testing.T
}

var (
	errInsertWPTRun    = errors.New("insert wpt run test error")
	errUpsertWPTMetric = errors.New("upsert wpt metric test error")
)

func (m *MockWebFeatureWPTScoreStorer) InsertWPTRun(
	_ context.Context,
	run WPTRun) error {
	if !reflect.DeepEqual(run, m.insertRunCfg.run) {
		m.t.Error("unexpected input to InsertWPTRun")
	}

	return m.insertRunCfg.err
}

func (m *MockWebFeatureWPTScoreStorer) UpsertWPTRunFeatureMetric(
	_ context.Context,
	runID int64,
	metricsPerFeature map[string]WPTFeatureMetric) error {
	if !reflect.DeepEqual(metricsPerFeature, m.upsertMetricCfg.metricsPerFeature) ||
		runID != m.upsertMetricCfg.runID {
		m.t.Error("unexpected input to UpsertWPTRunFeatureMetric")
	}

	return m.upsertMetricCfg.err
}

type processRunTest struct {
	name                      string
	inputRun                  shared.TestRun
	mockResultsDownloader     *MockResultsDownloader
	mockWebFeaturesDataGetter *MockWebFeaturesDataGetter
	mockWebFeatureWPTScorer   *MockWebFeatureWPTScorer
	insertRunConfig           *insertRunConfig
	upsertMetricConfig        *upsertMetricConfig
	expectedErr               error
}

func TestProcessRun(t *testing.T) {
	testCases := []processRunTest{
		// nolint: dupl // Ok to have similar test cases
		{
			name: "Successful Processing",
			// nolint: exhaustruct // WONTFIX: external struct
			inputRun: shared.TestRun{
				ID:        123,
				TimeStart: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				TimeEnd:   time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				Labels:    []string{shared.StableLabel},
				ProductAtRevision: shared.ProductAtRevision{
					FullRevisionHash: "sha",
					Product: shared.Product{
						OSName:         "os",
						OSVersion:      "osversion",
						BrowserName:    "browser",
						BrowserVersion: "browserverion",
					},
				},
			},
			insertRunConfig: &insertRunConfig{
				run: WPTRun{
					ID:               123,
					BrowserName:      "browser",
					BrowserVersion:   "browserverion",
					TimeStart:        time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
					TimeEnd:          time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
					Channel:          "stable",
					OSName:           "os",
					OSVersion:        "osversion",
					FullRevisionHash: "sha",
				},
				err: nil,
			},
			upsertMetricConfig: &upsertMetricConfig{
				runID: 123,
				metricsPerFeature: map[string]WPTFeatureMetric{
					"feature1": {TotalTests: valuePtr[int64](10), TestPass: valuePtr[int64](10)},
				},
				err: nil,
			},
			mockResultsDownloader: &MockResultsDownloader{
				resultsSummary: ResultsSummaryFile{"test1": query.SummaryResult{
					Status: "O",
					Counts: []int{0, 0},
				}},
				shouldFail: false,
			},
			mockWebFeaturesDataGetter: &MockWebFeaturesDataGetter{
				webFeaturesData: &shared.WebFeaturesData{},
				shouldFail:      false,
			},
			mockWebFeatureWPTScorer: &MockWebFeatureWPTScorer{
				metricsPerFeature: map[string]WPTFeatureMetric{
					"feature1": {TotalTests: valuePtr[int64](10), TestPass: valuePtr[int64](10)},
				},
			},
			expectedErr: nil,
		},
		{
			name: "Fail to download data",
			// nolint: exhaustruct // WONTFIX: external struct
			inputRun: shared.TestRun{
				ID:        123,
				TimeStart: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				TimeEnd:   time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				Labels:    []string{shared.StableLabel},
				ProductAtRevision: shared.ProductAtRevision{
					FullRevisionHash: "sha",
					Product: shared.Product{
						OSName:         "os",
						OSVersion:      "osversion",
						BrowserName:    "browser",
						BrowserVersion: "browserverion",
					},
				},
			},
			insertRunConfig:    nil,
			upsertMetricConfig: nil,
			mockResultsDownloader: &MockResultsDownloader{
				resultsSummary: nil,
				shouldFail:     true,
			},
			mockWebFeaturesDataGetter: &MockWebFeaturesDataGetter{
				webFeaturesData: nil,
				shouldFail:      false,
			},
			mockWebFeatureWPTScorer: nil,
			expectedErr:             errDownloadResults,
		},
		{
			name: "Fail to get data",
			// nolint: exhaustruct // WONTFIX: external struct
			inputRun: shared.TestRun{
				ID:        123,
				TimeStart: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				TimeEnd:   time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				Labels:    []string{shared.StableLabel},
				ProductAtRevision: shared.ProductAtRevision{
					FullRevisionHash: "sha",
					Product: shared.Product{
						OSName:         "os",
						OSVersion:      "osversion",
						BrowserName:    "browser",
						BrowserVersion: "browserverion",
					},
				},
			},
			insertRunConfig:    nil,
			upsertMetricConfig: nil,
			mockResultsDownloader: &MockResultsDownloader{
				resultsSummary: ResultsSummaryFile{"test1": query.SummaryResult{
					Status: "O",
					Counts: []int{0, 0},
				}},
				shouldFail: false,
			},
			mockWebFeaturesDataGetter: &MockWebFeaturesDataGetter{
				webFeaturesData: nil,
				shouldFail:      true,
			},
			mockWebFeatureWPTScorer: nil,
			expectedErr:             errGetWebFeaturesData,
		},
		{
			name: "Fail to insert run",
			// nolint: exhaustruct // WONTFIX: external struct
			inputRun: shared.TestRun{
				ID:        123,
				TimeStart: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				TimeEnd:   time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				Labels:    []string{shared.StableLabel},
				ProductAtRevision: shared.ProductAtRevision{
					FullRevisionHash: "sha",
					Product: shared.Product{
						OSName:         "os",
						OSVersion:      "osversion",
						BrowserName:    "browser",
						BrowserVersion: "browserverion",
					},
				},
			},
			insertRunConfig: &insertRunConfig{
				run: WPTRun{
					ID:               123,
					BrowserName:      "browser",
					BrowserVersion:   "browserverion",
					TimeStart:        time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
					TimeEnd:          time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
					Channel:          "stable",
					OSName:           "os",
					OSVersion:        "osversion",
					FullRevisionHash: "sha",
				},
				err: errInsertWPTRun,
			},
			upsertMetricConfig: nil,
			mockResultsDownloader: &MockResultsDownloader{
				resultsSummary: ResultsSummaryFile{"test1": query.SummaryResult{
					Status: "O",
					Counts: []int{0, 0},
				}},
				shouldFail: false,
			},
			mockWebFeaturesDataGetter: &MockWebFeaturesDataGetter{
				webFeaturesData: &shared.WebFeaturesData{},
				shouldFail:      false,
			},
			mockWebFeatureWPTScorer: &MockWebFeatureWPTScorer{
				metricsPerFeature: map[string]WPTFeatureMetric{
					"feature1": {TotalTests: valuePtr[int64](10), TestPass: valuePtr[int64](10)},
				},
			},
			expectedErr: errInsertWPTRun,
		},
		// nolint: dupl // Ok to have similar test cases
		{
			name: "Fail to upsert metric",
			// nolint: exhaustruct // WONTFIX: external struct
			inputRun: shared.TestRun{
				ID:        123,
				TimeStart: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				TimeEnd:   time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				Labels:    []string{shared.StableLabel},
				ProductAtRevision: shared.ProductAtRevision{
					FullRevisionHash: "sha",
					Product: shared.Product{
						OSName:         "os",
						OSVersion:      "osversion",
						BrowserName:    "browser",
						BrowserVersion: "browserverion",
					},
				},
			},
			insertRunConfig: &insertRunConfig{
				run: WPTRun{
					ID:               123,
					BrowserName:      "browser",
					BrowserVersion:   "browserverion",
					TimeStart:        time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
					TimeEnd:          time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
					Channel:          "stable",
					OSName:           "os",
					OSVersion:        "osversion",
					FullRevisionHash: "sha",
				},
				err: nil,
			},
			upsertMetricConfig: &upsertMetricConfig{
				runID: 123,
				metricsPerFeature: map[string]WPTFeatureMetric{
					"feature1": {TotalTests: valuePtr[int64](10), TestPass: valuePtr[int64](10)},
				},
				err: errUpsertWPTMetric,
			},
			mockResultsDownloader: &MockResultsDownloader{
				resultsSummary: ResultsSummaryFile{"test1": query.SummaryResult{
					Status: "O",
					Counts: []int{0, 0},
				}},
				shouldFail: false,
			},
			mockWebFeaturesDataGetter: &MockWebFeaturesDataGetter{
				webFeaturesData: &shared.WebFeaturesData{},
				shouldFail:      false,
			},
			mockWebFeatureWPTScorer: &MockWebFeatureWPTScorer{
				metricsPerFeature: map[string]WPTFeatureMetric{
					"feature1": {TotalTests: valuePtr[int64](10), TestPass: valuePtr[int64](10)},
				},
			},
			expectedErr: errUpsertWPTMetric,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewWPTRunProcessor(
				tt.mockResultsDownloader,
				tt.mockWebFeaturesDataGetter,
				tt.mockWebFeatureWPTScorer,
				&MockWebFeatureWPTScoreStorer{
					insertRunCfg:    tt.insertRunConfig,
					upsertMetricCfg: tt.upsertMetricConfig,
					t:               t,
				},
			)

			err := processor.ProcessRun(context.Background(), tt.inputRun)

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("Expected error: %v, Got: %v", tt.expectedErr, err)
			}
		})
	}
}
