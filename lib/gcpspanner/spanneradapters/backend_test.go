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
	"context"
	"errors"
	"math/big"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// nolint: gochecknoglobals
var (
	nonNilInputPageToken = valuePtr[string]("input-token")
	nonNilNextPageToken  = valuePtr[string]("test-token")
	errTest              = errors.New("something is wrong")
	testStart            = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	testEnd              = time.Date(2000, time.January, 31, 0, 0, 0, 0, time.UTC)
)

type mockFeaturesSearchConfig struct {
	expectedPageToken *string
	expectedPageSize  int
	expectedSortable  gcpspanner.Sortable
	expectedNode      *searchtypes.SearchNode
	result            *gcpspanner.FeatureResultPage
	returnedError     error
}

type mockGetFeatureConfig struct {
	expectedFilterable gcpspanner.Filterable
	result             *gcpspanner.FeatureResult
	returnedError      error
}

type mockGetIDByFeaturesIDConfig struct {
	expectedFilterable gcpspanner.Filterable
	result             *string
	returnedError      error
}

type mockListBrowserFeatureCountMetricConfig struct {
	result        *gcpspanner.BrowserFeatureCountResultPage
	returnedError error
}

type mockBackendSpannerClient struct {
	t                                    *testing.T
	aggregationData                      []gcpspanner.WPTRunAggregationMetricWithTime
	featureData                          []gcpspanner.WPTRunFeatureMetricWithTime
	mockFeaturesSearchCfg                mockFeaturesSearchConfig
	mockGetFeatureCfg                    mockGetFeatureConfig
	mockGetIDByFeaturesIDCfg             mockGetIDByFeaturesIDConfig
	mockListBrowserFeatureCountMetricCfg mockListBrowserFeatureCountMetricConfig
	pageToken                            *string
	err                                  error
}

func (c mockBackendSpannerClient) GetFeature(
	_ context.Context, filter gcpspanner.Filterable) (*gcpspanner.FeatureResult, error) {
	if !reflect.DeepEqual(filter, c.mockGetFeatureCfg.expectedFilterable) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockGetFeatureCfg.result, c.mockFeaturesSearchCfg.returnedError
}

func (c mockBackendSpannerClient) GetIDFromFeatureID(
	_ context.Context, filter *gcpspanner.FeatureIDFilter) (*string, error) {
	if !reflect.DeepEqual(filter, c.mockGetIDByFeaturesIDCfg.expectedFilterable) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockGetIDByFeaturesIDCfg.result, c.mockGetIDByFeaturesIDCfg.returnedError
}

func (c mockBackendSpannerClient) ListBrowserFeatureCountMetric(
	ctx context.Context,
	browser string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) (*gcpspanner.BrowserFeatureCountResultPage, error) {
	if ctx != context.Background() ||
		browser != "mybrowser" ||
		!startAt.Equal(testStart) ||
		!endAt.Equal(testEnd) ||
		pageSize != 100 ||
		pageToken != nonNilInputPageToken {
		c.t.Error("unexpected input to mock")
	}

	return c.mockListBrowserFeatureCountMetricCfg.result, c.mockListBrowserFeatureCountMetricCfg.returnedError
}

func (c mockBackendSpannerClient) ListMetricsForFeatureIDBrowserAndChannel(
	ctx context.Context,
	featureID string,
	browser string,
	channel string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]gcpspanner.WPTRunFeatureMetricWithTime, *string, error) {
	if ctx != context.Background() ||
		featureID != "feature" ||
		browser != "browser" ||
		channel != "channel" ||
		!startAt.Equal(testStart) ||
		!endAt.Equal(testEnd) ||
		pageSize != 100 ||
		pageToken != nonNilInputPageToken {
		c.t.Error("unexpected input to mock")
	}

	return c.featureData, c.pageToken, c.err
}

func (c mockBackendSpannerClient) ListMetricsOverTimeWithAggregatedTotals(
	ctx context.Context,
	featureIDs []string,
	browser string,
	channel string,
	startAt, endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]gcpspanner.WPTRunAggregationMetricWithTime, *string, error) {
	if ctx != context.Background() ||
		!slices.Equal[[]string](featureIDs, []string{"feature1", "feature2"}) ||
		browser != "browser" ||
		channel != "channel" ||
		!startAt.Equal(testStart) ||
		!endAt.Equal(testEnd) ||
		pageSize != 100 ||
		pageToken != nonNilInputPageToken {
		c.t.Error("unexpected input to mock")
	}

	return c.aggregationData, c.pageToken, c.err
}

func (c mockBackendSpannerClient) FeaturesSearch(
	_ context.Context,
	pageToken *string,
	pageSize int,
	searchNode *searchtypes.SearchNode,
	sortOrder gcpspanner.Sortable) (*gcpspanner.FeatureResultPage, error) {
	if pageToken != c.mockFeaturesSearchCfg.expectedPageToken ||
		pageSize != c.mockFeaturesSearchCfg.expectedPageSize ||
		!reflect.DeepEqual(searchNode, c.mockFeaturesSearchCfg.expectedNode) ||
		!reflect.DeepEqual(sortOrder, c.mockFeaturesSearchCfg.expectedSortable) {
		c.t.Error("unexpected input to mock")
	}

	return c.mockFeaturesSearchCfg.result,
		c.mockFeaturesSearchCfg.returnedError
}

func valuePtr[T any](in T) *T { return &in }

func TestListMetricsForFeatureIDBrowserAndChannel(t *testing.T) {
	testCases := []struct {
		name              string
		featureData       []gcpspanner.WPTRunFeatureMetricWithTime
		pageToken         *string
		err               error
		expectedOutput    []backend.WPTRunMetric
		expectedPageToken *string
		expectedErr       error
	}{
		{
			name: "success",
			featureData: []gcpspanner.WPTRunFeatureMetricWithTime{
				{
					TimeStart:  time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
					RunID:      10,
					TotalTests: valuePtr[int64](20),
					TestPass:   valuePtr[int64](10),
				},
				{
					TimeStart:  time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
					RunID:      9,
					TotalTests: valuePtr[int64](19),
					TestPass:   valuePtr[int64](9),
				},
				{
					TimeStart:  time.Date(2000, time.January, 8, 0, 0, 0, 0, time.UTC),
					RunID:      8,
					TotalTests: valuePtr[int64](18),
					TestPass:   valuePtr[int64](8),
				},
			},
			pageToken: nonNilNextPageToken,
			err:       nil,
			expectedOutput: []backend.WPTRunMetric{
				{
					RunTimestamp:    time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: valuePtr[int64](20),
					TestPassCount:   valuePtr[int64](10),
				},
				{
					RunTimestamp:    time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: valuePtr[int64](19),
					TestPassCount:   valuePtr[int64](9),
				},
				{
					RunTimestamp:    time.Date(2000, time.January, 8, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: valuePtr[int64](18),
					TestPassCount:   valuePtr[int64](8),
				},
			},
			expectedPageToken: nonNilNextPageToken,
			expectedErr:       nil,
		},
		{
			name:              "failure",
			featureData:       nil,
			pageToken:         nil,
			err:               errTest,
			expectedOutput:    nil,
			expectedPageToken: nil,
			expectedErr:       errTest,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:           t,
				featureData: tc.featureData,
				pageToken:   tc.pageToken,
				err:         tc.err,
			}
			backend := NewBackend(mock)
			metrics, pageToken, err := backend.ListMetricsForFeatureIDBrowserAndChannel(
				context.Background(), "feature", "browser", "channel", testStart, testEnd, 100, nonNilInputPageToken)
			if !errors.Is(err, tc.expectedErr) {
				t.Error("unexpected error")
			}

			if pageToken != tc.expectedPageToken {
				t.Error("unexpected page token")
			}

			if !reflect.DeepEqual(metrics, tc.expectedOutput) {
				t.Error("unexpected metrics")
			}
		})
	}
}

func TestListBrowserFeatureCountMetric(t *testing.T) {
	testCases := []struct {
		name         string
		cfg          mockListBrowserFeatureCountMetricConfig
		expectedPage *backend.BrowserReleaseFeatureMetricsPage
		expectedErr  error
	}{
		{
			name: "success",
			cfg: mockListBrowserFeatureCountMetricConfig{
				result: &gcpspanner.BrowserFeatureCountResultPage{
					NextPageToken: nonNilNextPageToken,
					Metrics: []gcpspanner.BrowserFeatureCountMetric{
						{
							ReleaseDate:  time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
							FeatureCount: 10,
						},
						{
							ReleaseDate:  time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
							FeatureCount: 9,
						},
					},
				},
				returnedError: nil,
			},
			expectedPage: &backend.BrowserReleaseFeatureMetricsPage{
				Metadata: &backend.PageMetadata{
					NextPageToken: nonNilNextPageToken,
				},
				Data: []backend.BrowserReleaseFeatureMetric{
					{
						Count:     valuePtr[int64](10),
						Timestamp: time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
					},
					{
						Count:     valuePtr[int64](9),
						Timestamp: time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "failure",
			cfg: mockListBrowserFeatureCountMetricConfig{
				result:        nil,
				returnedError: errTest,
			},
			expectedPage: nil,
			expectedErr:  errTest,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                                    t,
				mockListBrowserFeatureCountMetricCfg: tc.cfg,
			}
			backend := NewBackend(mock)
			page, err := backend.ListBrowserFeatureCountMetric(
				context.Background(),
				"mybrowser",
				testStart,
				testEnd,
				100,
				nonNilInputPageToken)
			if !errors.Is(err, tc.expectedErr) {
				t.Error("unexpected error")
			}

			if !reflect.DeepEqual(page, tc.expectedPage) {
				t.Error("unexpected metrics")
			}
		})
	}
}

func TestListMetricsOverTimeWithAggregatedTotals(t *testing.T) {

	testCases := []struct {
		name              string
		aggregatedData    []gcpspanner.WPTRunAggregationMetricWithTime
		pageToken         *string
		err               error
		expectedOutput    []backend.WPTRunMetric
		expectedPageToken *string
		expectedErr       error
	}{
		{
			name: "success",
			aggregatedData: []gcpspanner.WPTRunAggregationMetricWithTime{
				{
					WPTRunFeatureMetricWithTime: gcpspanner.WPTRunFeatureMetricWithTime{
						TimeStart:  time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
						RunID:      10,
						TotalTests: valuePtr[int64](20),
						TestPass:   valuePtr[int64](10),
					},
				},
				{
					WPTRunFeatureMetricWithTime: gcpspanner.WPTRunFeatureMetricWithTime{
						TimeStart:  time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
						RunID:      9,
						TotalTests: valuePtr[int64](19),
						TestPass:   valuePtr[int64](9),
					},
				},
				{
					WPTRunFeatureMetricWithTime: gcpspanner.WPTRunFeatureMetricWithTime{
						TimeStart:  time.Date(2000, time.January, 8, 0, 0, 0, 0, time.UTC),
						RunID:      8,
						TotalTests: valuePtr[int64](18),
						TestPass:   valuePtr[int64](8),
					},
				},
			},
			pageToken: nonNilNextPageToken,
			err:       nil,
			expectedOutput: []backend.WPTRunMetric{
				{
					RunTimestamp:    time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: valuePtr[int64](20),
					TestPassCount:   valuePtr[int64](10),
				},
				{
					RunTimestamp:    time.Date(2000, time.January, 9, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: valuePtr[int64](19),
					TestPassCount:   valuePtr[int64](9),
				},
				{
					RunTimestamp:    time.Date(2000, time.January, 8, 0, 0, 0, 0, time.UTC),
					TotalTestsCount: valuePtr[int64](18),
					TestPassCount:   valuePtr[int64](8),
				},
			},
			expectedPageToken: nonNilNextPageToken,
			expectedErr:       nil,
		},
		{
			name:              "failure",
			aggregatedData:    nil,
			pageToken:         nil,
			err:               errTest,
			expectedOutput:    nil,
			expectedPageToken: nil,
			expectedErr:       errTest,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:               t,
				aggregationData: tc.aggregatedData,
				pageToken:       tc.pageToken,
				err:             tc.err,
			}
			backend := NewBackend(mock)
			metrics, pageToken, err := backend.ListMetricsOverTimeWithAggregatedTotals(
				context.Background(),
				[]string{"feature1", "feature2"},
				"browser",
				"channel",
				testStart,
				testEnd,
				100,
				nonNilInputPageToken)
			if !errors.Is(err, tc.expectedErr) {
				t.Error("unexpected error")
			}

			if pageToken != tc.expectedPageToken {
				t.Error("unexpected page token")
			}

			if !reflect.DeepEqual(metrics, tc.expectedOutput) {
				t.Error("unexpected metrics")
			}
		})
	}
}

func TestConvertBaselineStatusBackendToSpanner(t *testing.T) {
	var backendToSpannerTests = []struct {
		name     string
		input    backend.FeatureBaselineStatus
		expected gcpspanner.BaselineStatus
	}{
		{"Widely to High", backend.Widely, gcpspanner.BaselineStatusHigh},
		{"Newly to Low", backend.Newly, gcpspanner.BaselineStatusLow},
		{"Limited to None", backend.Limited, gcpspanner.BaselineStatusNone},
		{"Invalid to Undefined", backend.FeatureBaselineStatus("invalid"),
			gcpspanner.BaselineStatusUndefined}, // Test default case
	}
	for _, tt := range backendToSpannerTests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertBaselineStatusBackendToSpanner(tt.input)
			if result != tt.expected {
				t.Errorf("convertBaselineStatusBackendToSpanner(%v): got %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvertBaselineStatusSpannerToBackend(t *testing.T) {
	var spannerToBackendTests = []struct {
		name     string
		input    gcpspanner.BaselineStatus
		expected backend.FeatureBaselineStatus
	}{
		{"High to Widely", gcpspanner.BaselineStatusHigh, backend.Widely},
		{"Low to Newly", gcpspanner.BaselineStatusLow, backend.Newly},
		{"None to Limited", gcpspanner.BaselineStatusNone, backend.Limited},
		{"Invalid to Undefined", gcpspanner.BaselineStatus("invalid"), backend.Undefined}, // Test default case
	}
	for _, tt := range spannerToBackendTests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertBaselineStatusSpannerToBackend(tt.input)
			if result != tt.expected {
				t.Errorf("convertBaselineStatusSpannerToBackend(%v): got %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFeaturesSearch(t *testing.T) {
	testCases := []struct {
		name           string
		cfg            mockFeaturesSearchConfig
		inputPageToken *string
		inputPageSize  int
		searchNode     *searchtypes.SearchNode
		sortOrder      *backend.GetV1FeaturesParamsSort
		expectedPage   *backend.FeaturePage
	}{
		{
			name: "regular",
			cfg: mockFeaturesSearchConfig{
				expectedPageToken: nonNilInputPageToken,
				expectedPageSize:  100,
				expectedNode: &searchtypes.SearchNode{
					Operator: searchtypes.OperatorRoot,
					Term:     nil,
					Children: nil,
				},
				expectedSortable: gcpspanner.NewFeatureNameSort(true),
				result: &gcpspanner.FeatureResultPage{
					Total:         100,
					NextPageToken: nonNilNextPageToken,
					Features: []gcpspanner.FeatureResult{
						{
							Name:      "feature 1",
							FeatureID: "feature1",
							Status:    "low",
							StableMetrics: []*gcpspanner.FeatureResultMetric{
								{
									BrowserName: "browser3",
									PassRate:    big.NewRat(10, 20),
								},
							},
							ExperimentalMetrics: []*gcpspanner.FeatureResultMetric{
								{
									BrowserName: "browser3",
									PassRate:    big.NewRat(10, 50),
								},
							},
						},
						{
							Name:      "feature 2",
							FeatureID: "feature2",
							Status:    "high",
							StableMetrics: []*gcpspanner.FeatureResultMetric{
								{
									BrowserName: "browser1",
									PassRate:    big.NewRat(10, 20),
								},
								{
									BrowserName: "browser2",
									PassRate:    big.NewRat(5, 20),
								},
							},
							ExperimentalMetrics: []*gcpspanner.FeatureResultMetric{
								{
									BrowserName: "browser1",
									PassRate:    big.NewRat(10, 20),
								},
								{
									BrowserName: "browser2",
									PassRate:    big.NewRat(2, 20),
								},
							},
						},
					},
				},
				returnedError: nil,
			},
			inputPageToken: nonNilInputPageToken,
			inputPageSize:  100,
			searchNode: &searchtypes.SearchNode{
				Operator: searchtypes.OperatorRoot,
				Term:     nil,
				Children: nil,
			},
			sortOrder: nil,
			expectedPage: &backend.FeaturePage{
				Metadata: backend.PageMetadataWithTotal{
					NextPageToken: nonNilNextPageToken,
					Total:         100,
				},
				Data: []backend.Feature{
					{
						BaselineStatus: backend.Newly,
						FeatureId:      "feature1",
						Name:           "feature 1",
						Spec:           nil,
						Usage:          nil,
						Wpt: &backend.FeatureWPTSnapshots{
							Experimental: &map[string]backend.WPTFeatureData{
								"browser3": {
									Score: valuePtr[float64](0.2),
								},
							},
							Stable: &map[string]backend.WPTFeatureData{
								"browser3": {
									Score: valuePtr[float64](0.5),
								},
							},
						},
						// TODO(https://github.com/GoogleChrome/webstatus.dev/issues/160)
						BrowserImplementations: nil,
					},
					{
						BaselineStatus: backend.Widely,
						FeatureId:      "feature2",
						Name:           "feature 2",
						Spec:           nil,
						Usage:          nil,
						Wpt: &backend.FeatureWPTSnapshots{
							Experimental: &map[string]backend.WPTFeatureData{
								"browser1": {
									Score: valuePtr[float64](0.5),
								},
								"browser2": {
									Score: valuePtr[float64](0.1),
								},
							},
							Stable: &map[string]backend.WPTFeatureData{
								"browser1": {
									Score: valuePtr[float64](0.5),
								},
								"browser2": {
									Score: valuePtr[float64](0.25),
								},
							},
						},
						// TODO(https://github.com/GoogleChrome/webstatus.dev/issues/160)
						BrowserImplementations: nil,
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                     t,
				mockFeaturesSearchCfg: tc.cfg,
			}
			bk := NewBackend(mock)
			page, err := bk.FeaturesSearch(
				context.Background(),
				tc.inputPageToken,
				tc.inputPageSize,
				tc.searchNode,
				tc.sortOrder)
			if !errors.Is(err, tc.cfg.returnedError) {
				t.Error("unexpected error")
			}

			if !reflect.DeepEqual(page, tc.expectedPage) {
				t.Error("unexpected page")
			}

		})
	}
}

// CompareFeatures checks if two backend.Feature structs are deeply equal.
func CompareFeatures(f1, f2 backend.Feature) bool {
	// 1. Basic Equality Checks
	if f1.BaselineStatus != f2.BaselineStatus ||
		f1.FeatureId != f2.FeatureId ||
		f1.Name != f2.Name ||
		f1.Usage != f2.Usage {
		return false
	}

	// 2. Compare 'spec' (slice of strings)
	if !reflect.DeepEqual(f1.Spec, f2.Spec) {
		return false
	}

	// 3. Compare FeatureWPTSnapshots (nested structs)
	if !compareWPTSnapshots(f1.Wpt, f2.Wpt) {
		return false
	}

	// All fields match
	return true
}

// compareWPTSnapshots helps compare FeatureWPTSnapshots structs.
func compareWPTSnapshots(w1, w2 *backend.FeatureWPTSnapshots) bool {
	// Handle nil cases
	if (w1 == nil && w2 != nil) || (w1 != nil && w2 == nil) {
		return false
	}

	if w1 == nil && w2 == nil { // Both nil
		return true
	}

	// Compare 'Experimental' maps
	if !compareFeatureDataMap(w1.Experimental, w2.Experimental) {
		return false
	}

	// Compare 'Stable' maps
	if !compareFeatureDataMap(w1.Stable, w2.Stable) {
		return false
	}

	return true
}

// compareFeatureDataMap helps compare maps of WPTFeatureData.
func compareFeatureDataMap(m1, m2 *map[string]backend.WPTFeatureData) bool {
	// Handle nil cases
	if (m1 == nil && m2 != nil) || (m1 != nil && m2 == nil) {
		return false
	}

	if m1 == nil && m2 == nil { // Both nil
		return true
	}

	// Check if lengths are equal
	if len(*m1) != len(*m2) {
		return false
	}

	// Compare each key-value pair
	for k, v1 := range *m1 {
		v2, ok := (*m2)[k]
		if !ok || *v1.Score != *v2.Score {
			return false
		}
	}

	return true
}

func TestGetFeature(t *testing.T) {
	testCases := []struct {
		name            string
		cfg             mockGetFeatureConfig
		inputFeatureID  string
		expectedFeature *backend.Feature
	}{
		{
			name:           "regular",
			inputFeatureID: "feature1",
			cfg: mockGetFeatureConfig{
				expectedFilterable: gcpspanner.NewFeatureIDFilter("feature1"),
				result: &gcpspanner.FeatureResult{
					Name:      "feature 1",
					FeatureID: "feature1",
					Status:    "low",
					StableMetrics: []*gcpspanner.FeatureResultMetric{
						{
							BrowserName: "browser3",
							PassRate:    big.NewRat(10, 20),
						},
					},
					ExperimentalMetrics: []*gcpspanner.FeatureResultMetric{
						{
							BrowserName: "browser3",
							PassRate:    big.NewRat(10, 50),
						},
					},
				},
				returnedError: nil,
			},
			expectedFeature: &backend.Feature{
				BaselineStatus: backend.Newly,
				FeatureId:      "feature1",
				Name:           "feature 1",
				Spec:           nil,
				Usage:          nil,
				Wpt: &backend.FeatureWPTSnapshots{
					Experimental: &map[string]backend.WPTFeatureData{
						"browser3": {
							Score: valuePtr[float64](0.2),
						},
					},
					Stable: &map[string]backend.WPTFeatureData{
						"browser3": {
							Score: valuePtr[float64](0.5),
						},
					},
				},
				// TODO(https://github.com/GoogleChrome/webstatus.dev/issues/160)
				BrowserImplementations: nil,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//nolint: exhaustruct
			mock := mockBackendSpannerClient{
				t:                 t,
				mockGetFeatureCfg: tc.cfg,
			}
			bk := NewBackend(mock)
			feature, err := bk.GetFeature(
				context.Background(),
				tc.inputFeatureID)
			if !errors.Is(err, tc.cfg.returnedError) {
				t.Error("unexpected error")
			}

			if !CompareFeatures(*feature, *tc.expectedFeature) {
				t.Error("unexpected feature")
			}

		})
	}
}

func TestGetFeatureSearchSortOrder(t *testing.T) {
	sortOrderTests := []struct {
		input *backend.GetV1FeaturesParamsSort
		want  gcpspanner.Sortable
	}{
		{input: nil, want: gcpspanner.NewFeatureNameSort(true)},
		{
			input: valuePtr[backend.GetV1FeaturesParamsSort](backend.NameAsc),
			want:  gcpspanner.NewFeatureNameSort(true),
		},
		{
			input: valuePtr[backend.GetV1FeaturesParamsSort](backend.NameDesc),
			want:  gcpspanner.NewFeatureNameSort(false),
		},
		{
			input: valuePtr[backend.GetV1FeaturesParamsSort](backend.BaselineStatusAsc),
			want:  gcpspanner.NewBaselineStatusSort(true),
		},
		{
			input: valuePtr[backend.GetV1FeaturesParamsSort](backend.BaselineStatusDesc),
			want:  gcpspanner.NewBaselineStatusSort(false),
		},
	}

	for _, tt := range sortOrderTests {
		got := getFeatureSearchSortOrder(tt.input)

		// Compare 'got' and 'tt.want' (Consider using a deep equality check library)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("got: %v, want: %v", got, tt.want)
		}
	}
}
