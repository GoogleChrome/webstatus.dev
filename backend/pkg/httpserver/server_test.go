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
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
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

type MockFeaturesSearchConfig struct {
	expectedPageToken  *string
	expectedPageSize   int
	expectedSearchNode *searchtypes.SearchNode
	expectedSortBy     *backend.GetV1FeaturesParamsSort
	page               *backend.FeaturePage
	err                error
}

type MockGetFeatureByIDConfig struct {
	expectedFeatureID string
	data              *backend.Feature
	err               error
}

type MockListBrowserFeatureCountMetricConfig struct {
	expectedBrowser   string
	expectedStartAt   time.Time
	expectedEndAt     time.Time
	expectedPageSize  int
	expectedPageToken *string
	pageToken         *string
	page              *backend.BrowserReleaseFeatureMetricsPage
	err               error
}

type MockWPTMetricsStorer struct {
	featureCfg                                        MockListMetricsForFeatureIDBrowserAndChannelConfig
	aggregateCfg                                      MockListMetricsOverTimeWithAggregatedTotalsConfig
	featuresSearchCfg                                 MockFeaturesSearchConfig
	listBrowserFeatureCountMetricCfg                  MockListBrowserFeatureCountMetricConfig
	getFeatureByIDConfig                              MockGetFeatureByIDConfig
	t                                                 *testing.T
	callCountListBrowserFeatureCountMetric            int
	callCountFeaturesSearch                           int
	callCountListMetricsForFeatureIDBrowserAndChannel int
	callCountListMetricsOverTimeWithAggregatedTotals  int
	callCountGetFeature                               int
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

func (m *MockWPTMetricsStorer) FeaturesSearch(
	_ context.Context,
	pageToken *string,
	pageSize int,
	node *searchtypes.SearchNode,
	sortBy *backend.GetV1FeaturesParamsSort,
) (*backend.FeaturePage, error) {
	m.callCountFeaturesSearch++

	if pageToken != m.featuresSearchCfg.expectedPageToken ||
		pageSize != m.featuresSearchCfg.expectedPageSize ||
		!reflect.DeepEqual(node, m.featuresSearchCfg.expectedSearchNode) ||
		!reflect.DeepEqual(sortBy, m.featuresSearchCfg.expectedSortBy) {
		m.t.Errorf("Incorrect arguments. Expected: %v, Got: { %v %d %v %v }",
			m.featuresSearchCfg, pageSize, pageToken, node, sortBy)
	}

	return m.featuresSearchCfg.page, m.featuresSearchCfg.err
}

func (m *MockWPTMetricsStorer) GetFeature(
	_ context.Context,
	featureID string,
) (*backend.Feature, error) {
	m.callCountGetFeature++

	if featureID != m.getFeatureByIDConfig.expectedFeatureID {
		m.t.Errorf("Incorrect arguments. Expected: %v, Got: { %s }",
			m.getFeatureByIDConfig, featureID)
	}

	return m.getFeatureByIDConfig.data, m.getFeatureByIDConfig.err
}

func (m *MockWPTMetricsStorer) ListBrowserFeatureCountMetric(
	_ context.Context,
	browser string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) (*backend.BrowserReleaseFeatureMetricsPage, error) {
	m.callCountListBrowserFeatureCountMetric++

	if browser != m.listBrowserFeatureCountMetricCfg.expectedBrowser ||
		!startAt.Equal(m.listBrowserFeatureCountMetricCfg.expectedStartAt) ||
		!endAt.Equal(m.listBrowserFeatureCountMetricCfg.expectedEndAt) ||
		pageSize != m.listBrowserFeatureCountMetricCfg.expectedPageSize ||
		pageToken != m.listBrowserFeatureCountMetricCfg.expectedPageToken {

		m.t.Errorf("Incorrect arguments. Expected: %v, Got: { %v, %s, %s, %d %v }",
			m.listBrowserFeatureCountMetricCfg, browser, startAt, endAt, pageSize, pageToken)
	}

	return m.listBrowserFeatureCountMetricCfg.page, m.listBrowserFeatureCountMetricCfg.err
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
