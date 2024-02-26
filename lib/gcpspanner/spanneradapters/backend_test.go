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
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
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

type mockBackendSpannerClient struct {
	t               *testing.T
	aggregationData []gcpspanner.WPTRunAggregationMetricWithTime
	featureData     []gcpspanner.WPTRunFeatureMetricWithTime
	pageToken       *string
	err             error
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
