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
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestListFeatureWPTMetrics(t *testing.T) {
	testCases := []struct {
		name              string
		mockConfig        MockListMetricsForFeatureIDBrowserAndChannelConfig
		expectedCallCount int // For the mock method
		request           backend.ListFeatureWPTMetricsRequestObject
		expectedResponse  backend.ListFeatureWPTMetricsResponseObject
		expectedError     error
	}{
		{
			name: "Success Case - no optional params - use defaults",
			mockConfig: MockListMetricsForFeatureIDBrowserAndChannelConfig{
				expectedFeatureID: "feature1",
				expectedBrowser:   "chrome",
				expectedChannel:   "experimental",
				expectedMetric:    backend.SubtestCounts,
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  100,
				expectedPageToken: nil,
				pageToken:         nil,
				err:               nil,

				data: []backend.WPTRunMetric{
					{
						RunTimestamp:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						TestPassCount:   valuePtr[int64](2),
						TotalTestsCount: valuePtr[int64](2),
					},
				},
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListFeatureWPTMetrics200JSONResponse{
				Data: []backend.WPTRunMetric{
					{
						RunTimestamp:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						TestPassCount:   valuePtr[int64](2),
						TotalTestsCount: valuePtr[int64](2),
					},
				},
				Metadata: &backend.PageMetadata{
					NextPageToken: nil,
				},
			},
			request: backend.ListFeatureWPTMetricsRequestObject{
				Params: backend.ListFeatureWPTMetricsParams{
					StartAt:   openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
					EndAt:     openapi_types.Date{Time: time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC)},
					PageToken: nil,
					PageSize:  nil,
				},
				Browser:    backend.Chrome,
				Channel:    backend.Experimental,
				MetricView: backend.SubtestCounts,
				FeatureId:  "feature1",
			},
			expectedError: nil,
		},
		{
			name: "Success Case - include optional params",
			mockConfig: MockListMetricsForFeatureIDBrowserAndChannelConfig{
				expectedFeatureID: "feature1",
				expectedBrowser:   "chrome",
				expectedChannel:   "experimental",
				expectedMetric:    backend.SubtestCounts,
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  50,
				expectedPageToken: inputPageToken,
				err:               nil,

				data: []backend.WPTRunMetric{
					{
						RunTimestamp:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						TestPassCount:   valuePtr[int64](2),
						TotalTestsCount: valuePtr[int64](2),
					},
				},
				pageToken: nextPageToken,
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListFeatureWPTMetrics200JSONResponse{
				Data: []backend.WPTRunMetric{
					{
						RunTimestamp:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						TestPassCount:   valuePtr[int64](2),
						TotalTestsCount: valuePtr[int64](2),
					},
				},
				Metadata: &backend.PageMetadata{
					NextPageToken: nextPageToken,
				},
			},
			request: backend.ListFeatureWPTMetricsRequestObject{
				Params: backend.ListFeatureWPTMetricsParams{
					StartAt:   openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
					EndAt:     openapi_types.Date{Time: time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC)},
					PageToken: inputPageToken,
					PageSize:  valuePtr[int](50),
				},
				Browser:    backend.Chrome,
				Channel:    backend.Experimental,
				MetricView: backend.SubtestCounts,
				FeatureId:  "feature1",
			},
			expectedError: nil,
		},
		{
			name: "500 case",
			mockConfig: MockListMetricsForFeatureIDBrowserAndChannelConfig{
				expectedFeatureID: "feature1",
				expectedBrowser:   "chrome",
				expectedChannel:   "experimental",
				expectedMetric:    backend.SubtestCounts,
				expectedPageToken: nil,
				data:              nil,
				pageToken:         nil,
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  100,
				err:               errTest,
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListFeatureWPTMetrics500JSONResponse{
				Code:    500,
				Message: "unable to get feature metrics",
			},
			request: backend.ListFeatureWPTMetricsRequestObject{
				Params: backend.ListFeatureWPTMetricsParams{
					StartAt:   openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
					EndAt:     openapi_types.Date{Time: time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC)},
					PageToken: nil,
					PageSize:  nil,
				},
				MetricView: backend.SubtestCounts,
				Browser:    backend.Chrome,
				Channel:    backend.Experimental,
				FeatureId:  "feature1",
			},
			expectedError: nil,
		},
		{
			name: "400 case - invalid page token",
			mockConfig: MockListMetricsForFeatureIDBrowserAndChannelConfig{
				expectedFeatureID: "feature1",
				expectedBrowser:   "chrome",
				expectedChannel:   "experimental",
				expectedMetric:    backend.SubtestCounts,
				expectedPageToken: badPageToken,
				data:              nil,
				pageToken:         nil,
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  100,
				err:               backendtypes.ErrInvalidPageToken,
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListFeatureWPTMetrics400JSONResponse{
				Code:    400,
				Message: "invalid page token",
			},
			request: backend.ListFeatureWPTMetricsRequestObject{
				Params: backend.ListFeatureWPTMetricsParams{
					StartAt:   openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
					EndAt:     openapi_types.Date{Time: time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC)},
					PageToken: badPageToken,
					PageSize:  nil,
				},
				MetricView: backend.SubtestCounts,
				Browser:    backend.Chrome,
				Channel:    backend.Experimental,
				FeatureId:  "feature1",
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				featureCfg: tc.mockConfig,
				t:          t,
			}
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil}

			// Call the function under test
			resp, err := myServer.ListFeatureWPTMetrics(context.Background(), tc.request)

			// Assertions
			if mockStorer.callCountListMetricsForFeatureIDBrowserAndChannel != tc.expectedCallCount {
				t.Errorf("Incorrect call count: expected %d, got %d",
					tc.expectedCallCount,
					mockStorer.callCountListMetricsOverTimeWithAggregatedTotals)
			}

			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(tc.expectedResponse, resp) {
				t.Errorf("Unexpected response: %v", resp)
			}
		})
	}
}
