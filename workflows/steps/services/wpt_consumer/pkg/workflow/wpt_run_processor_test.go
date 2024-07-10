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

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/wptconsumertypes"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

func valuePtr[T any](in T) *T { return &in }

type MockResultsDownloader struct {
	shouldFail     bool
	resultsSummary ResultsSummaryFile
}

var errDownloadResults = errors.New("download results error")

// nolint: ireturn
func (m *MockResultsDownloader) DownloadResults(_ context.Context, _ string) (ResultsSummaryFile, error) {
	if m.shouldFail {
		return nil, errDownloadResults
	}

	return m.resultsSummary, nil
}

type MockWebFeaturesDataGetter struct {
	shouldFail      bool
	webFeaturesData shared.WebFeaturesData
}

var errGetWebFeaturesData = errors.New("web features test error")

func (m *MockWebFeaturesDataGetter) GetWebFeaturesData(
	_ context.Context,
	_ string) (shared.WebFeaturesData, error) {
	if m.shouldFail {
		return nil, errGetWebFeaturesData
	}

	return m.webFeaturesData, nil
}

type MockResultsFile struct {
	metricsPerFeature map[string]wptconsumertypes.WPTFeatureMetric
}

func (m MockResultsFile) Score(
	_ context.Context,
	_ *shared.WebFeaturesData) map[string]wptconsumertypes.WPTFeatureMetric {
	return m.metricsPerFeature
}

type insertRunConfig struct {
	run shared.TestRun
	err error
}

type upsertMetricConfig struct {
	runID             int64
	metricsPerFeature map[string]wptconsumertypes.WPTFeatureMetric
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
	run shared.TestRun) error {
	if !reflect.DeepEqual(run, m.insertRunCfg.run) {
		m.t.Error("unexpected input to InsertWPTRun")
	}

	return m.insertRunCfg.err
}

func (m *MockWebFeatureWPTScoreStorer) UpsertWPTRunFeatureMetrics(
	_ context.Context,
	runID int64,
	metricsPerFeature map[string]wptconsumertypes.WPTFeatureMetric) error {
	if !reflect.DeepEqual(metricsPerFeature, m.upsertMetricCfg.metricsPerFeature) ||
		runID != m.upsertMetricCfg.runID {
		m.t.Error("unexpected input to UpsertWPTRunFeatureMetrics")
	}

	return m.upsertMetricCfg.err
}

type processRunTest struct {
	name                      string
	inputRun                  shared.TestRun
	mockResultsDownloader     *MockResultsDownloader
	mockWebFeaturesDataGetter *MockWebFeaturesDataGetter
	insertRunConfig           *insertRunConfig
	upsertMetricConfig        *upsertMetricConfig
	expectedErr               error
}

// nolint: lll // WONTFIX
const v2ResultsURL = "https://storage.googleapis.com/wptd/ddd7a27d89d29d2f4573213fa9b757952efd75f1/chrome-128.0.6583.0-linux-20.04-bdd263c4fe-summary_v2.json.gz"

// nolint: lll // WONTFIX
const v1ResultsURL = "https://storage.googleapis.com/wptd/2ca67d39e538810661cd0c9a024f5ce605aa2ab1/chrome-105.0.5148.2_dev-linux-20.04-8a205efcc5-summary.json.gz"

func TestProcessRun(t *testing.T) {
	testCases := []processRunTest{
		// nolint: dupl // Ok to have similar test cases
		{
			name: "Successful Processing v2 file",
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
				ResultsURL: v2ResultsURL,
			},
			insertRunConfig: &insertRunConfig{
				// nolint: exhaustruct // WONTFIX: external struct
				run: shared.TestRun{
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
					ResultsURL: v2ResultsURL,
				},
				err: nil,
			},
			upsertMetricConfig: &upsertMetricConfig{
				runID: 123,
				metricsPerFeature: map[string]wptconsumertypes.WPTFeatureMetric{
					"feature1": {
						TotalTests:        valuePtr[int64](10),
						TestPass:          valuePtr[int64](10),
						TotalSubtests:     valuePtr[int64](100),
						SubtestPass:       valuePtr[int64](100),
						FeatureRunDetails: nil,
					},
				},
				err: nil,
			},
			mockResultsDownloader: &MockResultsDownloader{
				resultsSummary: MockResultsFile{
					metricsPerFeature: map[string]wptconsumertypes.WPTFeatureMetric{
						"feature1": {
							TotalTests:        valuePtr[int64](10),
							TestPass:          valuePtr[int64](10),
							TotalSubtests:     valuePtr[int64](100),
							SubtestPass:       valuePtr[int64](100),
							FeatureRunDetails: nil,
						},
					},
				},
				shouldFail: false,
			},
			mockWebFeaturesDataGetter: &MockWebFeaturesDataGetter{
				webFeaturesData: shared.WebFeaturesData{
					"test1.html": {
						"feature1": nil,
					},
				},
				shouldFail: false,
			},
			expectedErr: nil,
		},
		{
			name: "skip non v2 file",
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
				ResultsURL: v1ResultsURL,
			},
			insertRunConfig:    nil,
			upsertMetricConfig: nil,
			mockResultsDownloader: &MockResultsDownloader{
				resultsSummary: nil,
				shouldFail:     false,
			},
			mockWebFeaturesDataGetter: &MockWebFeaturesDataGetter{
				webFeaturesData: shared.WebFeaturesData{},
				shouldFail:      false,
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
				ResultsURL: v2ResultsURL,
			},
			insertRunConfig:    nil,
			upsertMetricConfig: nil,
			mockResultsDownloader: &MockResultsDownloader{
				resultsSummary: nil,
				shouldFail:     true,
			},
			mockWebFeaturesDataGetter: &MockWebFeaturesDataGetter{
				webFeaturesData: shared.WebFeaturesData{},
				shouldFail:      false,
			},
			expectedErr: errDownloadResults,
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
				ResultsURL: v2ResultsURL,
			},
			insertRunConfig:    nil,
			upsertMetricConfig: nil,
			mockResultsDownloader: &MockResultsDownloader{
				resultsSummary: MockResultsFile{
					metricsPerFeature: map[string]wptconsumertypes.WPTFeatureMetric{
						"feature1": {
							TotalTests:        valuePtr[int64](10),
							TestPass:          valuePtr[int64](10),
							TotalSubtests:     valuePtr[int64](100),
							SubtestPass:       valuePtr[int64](100),
							FeatureRunDetails: nil,
						},
					},
				},
				shouldFail: false,
			},
			mockWebFeaturesDataGetter: &MockWebFeaturesDataGetter{
				webFeaturesData: shared.WebFeaturesData{},
				shouldFail:      true,
			},
			expectedErr: errGetWebFeaturesData,
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
				ResultsURL: v2ResultsURL,
			},
			insertRunConfig: &insertRunConfig{
				// nolint: exhaustruct // WONTFIX: external struct
				run: shared.TestRun{
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
					ResultsURL: v2ResultsURL,
				},
				err: errInsertWPTRun,
			},
			upsertMetricConfig: nil,
			mockResultsDownloader: &MockResultsDownloader{
				resultsSummary: MockResultsFile{
					metricsPerFeature: map[string]wptconsumertypes.WPTFeatureMetric{
						"feature1": {
							TotalTests:        valuePtr[int64](10),
							TestPass:          valuePtr[int64](10),
							TotalSubtests:     valuePtr[int64](100),
							SubtestPass:       valuePtr[int64](100),
							FeatureRunDetails: nil,
						},
					},
				},
				shouldFail: false,
			},
			mockWebFeaturesDataGetter: &MockWebFeaturesDataGetter{
				webFeaturesData: shared.WebFeaturesData{},
				shouldFail:      false,
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
				ResultsURL: v2ResultsURL,
			},
			insertRunConfig: &insertRunConfig{
				// nolint: exhaustruct // WONTFIX: external struct
				run: shared.TestRun{
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
					ResultsURL: v2ResultsURL,
				},
				err: nil,
			},
			upsertMetricConfig: &upsertMetricConfig{
				runID: 123,
				metricsPerFeature: map[string]wptconsumertypes.WPTFeatureMetric{
					"feature1": {
						TotalTests:        valuePtr[int64](10),
						TestPass:          valuePtr[int64](10),
						TotalSubtests:     valuePtr[int64](100),
						SubtestPass:       valuePtr[int64](100),
						FeatureRunDetails: nil,
					},
				},
				err: errUpsertWPTMetric,
			},
			mockResultsDownloader: &MockResultsDownloader{
				resultsSummary: MockResultsFile{
					metricsPerFeature: map[string]wptconsumertypes.WPTFeatureMetric{
						"feature1": {
							TotalTests:        valuePtr[int64](10),
							TestPass:          valuePtr[int64](10),
							TotalSubtests:     valuePtr[int64](100),
							SubtestPass:       valuePtr[int64](100),
							FeatureRunDetails: nil,
						},
					},
				},
				shouldFail: false,
			},
			mockWebFeaturesDataGetter: &MockWebFeaturesDataGetter{
				webFeaturesData: shared.WebFeaturesData{},
				shouldFail:      false,
			},
			expectedErr: errUpsertWPTMetric,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewWPTRunProcessor(
				tt.mockResultsDownloader,
				tt.mockWebFeaturesDataGetter,
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
