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

package spanneradapters

import (
	"cmp"
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/wptconsumertypes"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

type MockWPTWorkflowSpannerClient struct {
	InsertWPTRunConfig               *InsertWPTRunConfig
	UpsertWPTRunFeatureMetricsConfig *UpsertWPTRunFeatureMetricsConfig
	t                                *testing.T
}

type InsertWPTRunConfig struct {
	input *gcpspanner.WPTRun
	err   error
}

type UpsertWPTRunFeatureMetricsConfig struct {
	inputID      int64
	inputMetrics []gcpspanner.WPTRunFeatureMetric

	err error
}

// Implementations of the interface methods.
func (m *MockWPTWorkflowSpannerClient) InsertWPTRun(_ context.Context, run gcpspanner.WPTRun) error {
	if !reflect.DeepEqual(run, *m.InsertWPTRunConfig.input) {
		m.t.Error("unexpected input to InsertWPTRun")
	}

	return m.InsertWPTRunConfig.err
}

func (m *MockWPTWorkflowSpannerClient) UpsertWPTRunFeatureMetrics(
	_ context.Context, externalRunID int64, in []gcpspanner.WPTRunFeatureMetric) error {
	// Sort the input to make it stable given the input is originally built from an unordered map.
	slices.SortFunc(in, func(a, b gcpspanner.WPTRunFeatureMetric) int {
		return cmp.Compare(a.FeatureID, b.FeatureID)
	})
	if externalRunID != m.UpsertWPTRunFeatureMetricsConfig.inputID ||
		!reflect.DeepEqual(in, m.UpsertWPTRunFeatureMetricsConfig.inputMetrics) {
		m.t.Error("unexpected input to UpsertWPTRunFeatureMetrics")
	}

	return m.UpsertWPTRunFeatureMetricsConfig.err
}

func getSampleTestRun() shared.TestRun {
	// nolint: exhaustruct // WONTFIX: external struct
	return shared.TestRun{
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
	}
}

func TestWPTConsumer_InsertWPTRun(t *testing.T) {
	testCases := []struct {
		name          string
		mockConfig    InsertWPTRunConfig
		input         shared.TestRun
		expectedError error
	}{
		{
			name: "Success",
			mockConfig: InsertWPTRunConfig{
				input: &gcpspanner.WPTRun{
					RunID:            123,
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

			input:         getSampleTestRun(),
			expectedError: nil,
		},
		{
			name: "Invalid Channel",
			mockConfig: InsertWPTRunConfig{
				input: nil,
				err:   nil,
			},
			// nolint: exhaustruct // WONTFIX: external struct
			input: shared.TestRun{
				ID:        123,
				TimeStart: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				TimeEnd:   time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
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
			expectedError: wptconsumertypes.ErrInvalidDataFromWPT,
		},
		{
			name: "Database Error",
			mockConfig: InsertWPTRunConfig{
				input: &gcpspanner.WPTRun{
					RunID:            123,
					BrowserName:      "browser",
					BrowserVersion:   "browserverion",
					TimeStart:        time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
					TimeEnd:          time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
					Channel:          "stable",
					OSName:           "os",
					OSVersion:        "osversion",
					FullRevisionHash: "sha",
				},
				err: errors.New("database error"),
			},
			input:         getSampleTestRun(),
			expectedError: wptconsumertypes.ErrUnableToStoreWPTRun,
		},
	}

	for idx, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockWPTWorkflowSpannerClient{
				InsertWPTRunConfig:               &testCases[idx].mockConfig,
				UpsertWPTRunFeatureMetricsConfig: nil,
				t:                                t,
			}
			consumer := NewWPTWorkflowConsumer(mockClient)

			err := consumer.InsertWPTRun(context.Background(), tc.input)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Expected error: %v, got: %v", tc.expectedError, err)
			}
		})
	}
}

func TestWPTConsumer_UpsertWPTRunFeatureMetrics(t *testing.T) {
	testCases := []struct {
		name          string
		mockConfig    UpsertWPTRunFeatureMetricsConfig
		externalRunID int64
		metrics       map[string]wptconsumertypes.WPTFeatureMetric
		expectedError error
	}{
		{
			name: "Success",
			mockConfig: UpsertWPTRunFeatureMetricsConfig{
				inputID: 123,
				inputMetrics: []gcpspanner.WPTRunFeatureMetric{
					{
						FeatureID:  "feature1",
						TotalTests: valuePtr[int64](1),
						TestPass:   valuePtr[int64](0),
					},
					{
						FeatureID:  "feature2",
						TotalTests: valuePtr[int64](11),
						TestPass:   valuePtr[int64](10),
					},
				},
				err: nil,
			},
			externalRunID: 123,
			metrics: map[string]wptconsumertypes.WPTFeatureMetric{
				"feature1": {
					TotalTests: valuePtr[int64](1),
					TestPass:   valuePtr[int64](0),
				},
				"feature2": {
					TotalTests: valuePtr[int64](11),
					TestPass:   valuePtr[int64](10),
				},
			},
			expectedError: nil,
		},
		{
			name: "Database error",
			mockConfig: UpsertWPTRunFeatureMetricsConfig{
				inputID: 123,
				inputMetrics: []gcpspanner.WPTRunFeatureMetric{
					{
						FeatureID:  "feature1",
						TotalTests: valuePtr[int64](1),
						TestPass:   valuePtr[int64](0),
					},
					{
						FeatureID:  "feature2",
						TotalTests: valuePtr[int64](11),
						TestPass:   valuePtr[int64](10),
					},
				},
				err: errors.New("database error"),
			},
			externalRunID: 123,
			metrics: map[string]wptconsumertypes.WPTFeatureMetric{
				"feature1": {
					TotalTests: valuePtr[int64](1),
					TestPass:   valuePtr[int64](0),
				},
				"feature2": {
					TotalTests: valuePtr[int64](11),
					TestPass:   valuePtr[int64](10),
				},
			},
			expectedError: wptconsumertypes.ErrUnableToStoreWPTRunFeatureMetrics,
		},
	}

	for idx, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockWPTWorkflowSpannerClient{
				UpsertWPTRunFeatureMetricsConfig: &testCases[idx].mockConfig,
				InsertWPTRunConfig:               nil,
				t:                                t,
			}
			consumer := NewWPTWorkflowConsumer(mockClient)

			err := consumer.UpsertWPTRunFeatureMetrics(context.Background(), tc.externalRunID, tc.metrics)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Expected error: %v, got: %v", tc.expectedError, err)
			}
		})
	}
}

func TestNewWPTRun(t *testing.T) {
	testCases := []struct {
		name           string
		input          shared.TestRun
		expectedOutput gcpspanner.WPTRun
	}{
		{
			name: "stable channel",
			// nolint: exhaustruct // WONTFIX: external struct
			input: shared.TestRun{
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
			expectedOutput: gcpspanner.WPTRun{
				RunID:            123,
				BrowserName:      "browser",
				BrowserVersion:   "browserverion",
				TimeStart:        time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				TimeEnd:          time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				Channel:          "stable",
				OSName:           "os",
				OSVersion:        "osversion",
				FullRevisionHash: "sha",
			},
		},
		{
			name: "experimental channel",
			// nolint: exhaustruct // WONTFIX: external struct
			input: shared.TestRun{
				ID:        123,
				TimeStart: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				TimeEnd:   time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				Labels:    []string{shared.ExperimentalLabel},
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
			expectedOutput: gcpspanner.WPTRun{
				RunID:            123,
				BrowserName:      "browser",
				BrowserVersion:   "browserverion",
				TimeStart:        time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				TimeEnd:          time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				Channel:          "experimental",
				OSName:           "os",
				OSVersion:        "osversion",
				FullRevisionHash: "sha",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := NewWPTRun(tc.input)
			if !reflect.DeepEqual(tc.expectedOutput, output) {
				t.Error("unexpected output")
			}
		})
	}
}
