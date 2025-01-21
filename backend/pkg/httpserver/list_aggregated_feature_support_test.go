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

func TestListAggregatedFeatureSupport(t *testing.T) {
	testCases := []struct {
		name              string
		mockConfig        MockListBrowserFeatureCountMetricConfig
		expectedCallCount int // For the mock method
		request           backend.ListAggregatedFeatureSupportRequestObject
		expectedResponse  backend.ListAggregatedFeatureSupportResponseObject
		expectedError     error
	}{
		{
			name: "Success Case - no optional params - use defaults",
			mockConfig: MockListBrowserFeatureCountMetricConfig{
				expectedBrowser:   "chrome",
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  100,
				expectedPageToken: nil,
				pageToken:         nil,
				err:               nil,
				page: &backend.BrowserReleaseFeatureMetricsPage{
					Metadata: &backend.PageMetadata{
						NextPageToken: nil,
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
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListAggregatedFeatureSupport200JSONResponse{
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
				Metadata: &backend.PageMetadata{
					NextPageToken: nil,
				},
			},
			request: backend.ListAggregatedFeatureSupportRequestObject{
				Params: backend.ListAggregatedFeatureSupportParams{
					StartAt:   openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
					EndAt:     openapi_types.Date{Time: time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC)},
					PageToken: nil,
					PageSize:  nil,
				},
				Browser: backend.Chrome,
			},
			expectedError: nil,
		},
		{
			name: "Success Case - include optional params",
			mockConfig: MockListBrowserFeatureCountMetricConfig{
				expectedBrowser:   "chrome",
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  50,
				expectedPageToken: inputPageToken,
				err:               nil,
				page: &backend.BrowserReleaseFeatureMetricsPage{
					Metadata: &backend.PageMetadata{
						NextPageToken: nextPageToken,
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
				pageToken: nextPageToken,
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListAggregatedFeatureSupport200JSONResponse{
				Metadata: &backend.PageMetadata{
					NextPageToken: nextPageToken,
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
			request: backend.ListAggregatedFeatureSupportRequestObject{
				Params: backend.ListAggregatedFeatureSupportParams{
					StartAt:   openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
					EndAt:     openapi_types.Date{Time: time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC)},
					PageToken: inputPageToken,
					PageSize:  valuePtr[int](50),
				},
				Browser: backend.Chrome,
			},
			expectedError: nil,
		},
		{
			name: "500 case",
			mockConfig: MockListBrowserFeatureCountMetricConfig{
				expectedBrowser:   "chrome",
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  100,
				expectedPageToken: nil,
				page:              nil,
				pageToken:         nil,
				err:               errTest,
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListAggregatedFeatureSupport500JSONResponse{
				Code:    500,
				Message: "unable to get feature support metrics",
			},
			request: backend.ListAggregatedFeatureSupportRequestObject{
				Params: backend.ListAggregatedFeatureSupportParams{
					StartAt:   openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
					EndAt:     openapi_types.Date{Time: time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC)},
					PageToken: nil,
					PageSize:  nil,
				},
				Browser: backend.Chrome,
			},
			expectedError: nil,
		},
		{
			name: "400 case - invalid page token",
			mockConfig: MockListBrowserFeatureCountMetricConfig{
				expectedBrowser:   "chrome",
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  100,
				expectedPageToken: badPageToken,
				pageToken:         nil,
				err:               backendtypes.ErrInvalidPageToken,
				page:              nil,
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListAggregatedFeatureSupport400JSONResponse{
				Code:    400,
				Message: "invalid page token",
			},
			request: backend.ListAggregatedFeatureSupportRequestObject{
				Params: backend.ListAggregatedFeatureSupportParams{
					StartAt:   openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
					EndAt:     openapi_types.Date{Time: time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC)},
					PageToken: badPageToken,
					PageSize:  nil,
				},
				Browser: backend.Chrome,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				listBrowserFeatureCountMetricCfg: tc.mockConfig,
				t:                                t,
			}
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil}

			// Call the function under test
			resp, err := myServer.ListAggregatedFeatureSupport(context.Background(), tc.request)

			// Assertions
			if mockStorer.callCountListBrowserFeatureCountMetric != tc.expectedCallCount {
				t.Errorf("Incorrect call count: expected %d, got %d",
					tc.expectedCallCount,
					mockStorer.callCountListBrowserFeatureCountMetric)
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
