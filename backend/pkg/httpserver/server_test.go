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

package httpserver

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func valuePtr[T any](in T) *T { return &in }

type MockListMetricsForFeatureIDBrowserAndChannelConfig struct {
	expectedFeatureID string
	expectedBrowser   string
	expectedChannel   string
	expectedStartAt   time.Time
	expectedEndAt     time.Time
	expectedPageSize  int
	expectedPageToken *string
	data              []backend.WPTRunMetric
	pageToken         *string
	err               error
}

type MockListMetricsOverTimeWithAggregatedTotalsConfig struct {
	expectedFeatureIDs []string
	expectedBrowser    string
	expectedChannel    string
	expectedStartAt    time.Time
	expectedEndAt      time.Time
	expectedPageSize   int
	expectedPageToken  *string
	data               []backend.WPTRunMetric
	pageToken          *string
	err                error
}

type MockWPTMetricsStorer struct {
	featureCfg                                        MockListMetricsForFeatureIDBrowserAndChannelConfig
	aggregateCfg                                      MockListMetricsOverTimeWithAggregatedTotalsConfig
	t                                                 *testing.T
	callCountListMetricsForFeatureIDBrowserAndChannel int
	callCountListMetricsOverTimeWithAggregatedTotals  int
}

func (m *MockWPTMetricsStorer) ListMetricsForFeatureIDBrowserAndChannel(_ context.Context,
	featureID string, browser string, channel string,
	startAt time.Time, endAt time.Time,
	pageSize int, pageToken *string) ([]backend.WPTRunMetric, *string, error) {
	m.callCountListMetricsForFeatureIDBrowserAndChannel++

	if featureID != m.featureCfg.expectedFeatureID ||
		browser != m.featureCfg.expectedBrowser ||
		channel != m.featureCfg.expectedChannel ||
		!startAt.Equal(m.featureCfg.expectedStartAt) ||
		!endAt.Equal(m.featureCfg.expectedEndAt) ||
		pageSize != m.featureCfg.expectedPageSize ||
		pageToken != m.featureCfg.expectedPageToken {

		m.t.Errorf("Incorrect arguments. Expected: %v, Got: { %s, %s, %s, %s, %s, %d %v }",
			m.featureCfg, featureID, browser, channel, startAt, endAt, pageSize, pageToken)
	}

	return m.featureCfg.data, m.featureCfg.pageToken, m.featureCfg.err
}

func (m *MockWPTMetricsStorer) ListMetricsOverTimeWithAggregatedTotals(
	_ context.Context,
	featureIDs []string,
	browser string,
	channel string,
	startAt, endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]backend.WPTRunMetric, *string, error) {
	m.callCountListMetricsOverTimeWithAggregatedTotals++

	if !slices.Equal(featureIDs, m.aggregateCfg.expectedFeatureIDs) ||
		browser != m.aggregateCfg.expectedBrowser ||
		channel != m.aggregateCfg.expectedChannel ||
		!startAt.Equal(m.aggregateCfg.expectedStartAt) ||
		!endAt.Equal(m.aggregateCfg.expectedEndAt) ||
		pageSize != m.aggregateCfg.expectedPageSize ||
		pageToken != m.aggregateCfg.expectedPageToken {

		m.t.Errorf("Incorrect arguments. Expected: %v, Got: { %v, %s, %s, %s, %s, %d %v }",
			m.aggregateCfg, featureIDs, browser, channel, startAt, endAt, pageSize, pageToken)
	}

	return m.aggregateCfg.data, m.aggregateCfg.pageToken, m.aggregateCfg.err
}

func TestGetPageSizeOrDefault(t *testing.T) {
	testCases := []struct {
		name          string
		inputPageSize *int
		expected      int
	}{
		{"Nil input", nil, 100},
		{"Input below min", valuePtr[int](0), 100},
		{"Valid input (below max)", valuePtr[int](25), 25},
		{"Input above max", valuePtr[int](100), 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getPageSizeOrDefault(tc.inputPageSize)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}

// nolint: gochecknoglobals
var (
	inputPageToken = valuePtr[string]("input-token")
	nextPageToken  = valuePtr[string]("next-page-token")
	errTest        = errors.New("test error")
)
