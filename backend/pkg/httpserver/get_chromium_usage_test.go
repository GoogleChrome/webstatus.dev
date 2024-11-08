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

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestGetV(t *testing.T) {
	testCases := []struct {
		name              string
		mockConfig        MockGetFeatureByIDConfig
		expectedCallCount int // For the mock method
		request           backend.GetV1FeaturesIdStatusUsageChromiumDailyStatsRequestObject
		expectedResponse  backend.GetV1FeaturesIdStatusUsageChromiumDailyStatsResponseObject
		expectedError     error
	}{
		{
			name: "Success Case",
			// nolint:dupl // WONTFIX - being explicit for short list of tests.
			mockConfig: MockGetFeatureByIDConfig{
				expectedFeatureID:     "feature1",
				expectedWPTMetricView: backend.SubtestCounts,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
				},
				data: &backend.Feature{
					Baseline: &backend.BaselineInfo{
						Status: valuePtr(backend.Widely),
						LowDate: valuePtr(
							openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
						),
						HighDate: valuePtr(
							openapi_types.Date{Time: time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)},
						),
					},
					BrowserImplementations: &map[string]backend.BrowserImplementation{
						"chrome": {
							Status:  valuePtr(backend.Available),
							Date:    &openapi_types.Date{Time: time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC)},
							Version: valuePtr("100"),
						},
					},
					FeatureId: "feature1",
					Name:      "feature 1",
					Spec:      nil,
					Usage: &backend.BrowserUsage{
						Chromium: &backend.ChromiumUsageInfo{
							Daily: valuePtr[float64](0.91),
						},
					},
					Wpt: nil,
				},
				err: nil,
			},
			expectedCallCount: 1,
			expectedResponse: backend.GetV1FeaturesIdStatusUsageChromiumDailyStats200JSONResponse{
				Usage: valuePtr[float64](0.91),
			},
			request: backend.GetV1FeaturesIdStatusUsageChromiumDailyStatsRequestObject{
				Id: "feature1",
			},
			expectedError: nil,
		},
		{
			name: "Success Case with null usage",
			// nolint:dupl // WONTFIX - being explicit for short list of tests.
			mockConfig: MockGetFeatureByIDConfig{
				expectedFeatureID:     "feature1",
				expectedWPTMetricView: backend.SubtestCounts,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
				},
				data: &backend.Feature{
					Baseline: &backend.BaselineInfo{
						Status: valuePtr(backend.Widely),
						LowDate: valuePtr(
							openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
						),
						HighDate: valuePtr(
							openapi_types.Date{Time: time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)},
						),
					},
					BrowserImplementations: &map[string]backend.BrowserImplementation{
						"chrome": {
							Status:  valuePtr(backend.Available),
							Date:    &openapi_types.Date{Time: time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC)},
							Version: valuePtr("100"),
						},
					},
					FeatureId: "feature1",
					Name:      "feature 1",
					Spec:      nil,
					Wpt:       nil,
					Usage: &backend.BrowserUsage{
						Chromium: &backend.ChromiumUsageInfo{
							Daily: nil,
						},
					},
				},
				err: nil,
			},
			expectedCallCount: 1,
			expectedResponse: backend.GetV1FeaturesIdStatusUsageChromiumDailyStats200JSONResponse{
				Usage: nil,
			},
			request: backend.GetV1FeaturesIdStatusUsageChromiumDailyStatsRequestObject{
				Id: "feature1",
			},
			expectedError: nil,
		},
		{
			name: "404",
			mockConfig: MockGetFeatureByIDConfig{
				expectedFeatureID:     "feature1",
				expectedWPTMetricView: backend.SubtestCounts,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
				},
				data: nil,
				err:  gcpspanner.ErrQueryReturnedNoResults,
			},
			expectedCallCount: 1,
			expectedResponse: backend.GetV1FeaturesIdStatusUsageChromiumDailyStats404JSONResponse{
				Code:    404,
				Message: "feature id feature1 is not found",
			},
			request: backend.GetV1FeaturesIdStatusUsageChromiumDailyStatsRequestObject{
				Id: "feature1",
			},
			expectedError: nil,
		},
		{
			name: "500",
			mockConfig: MockGetFeatureByIDConfig{
				expectedFeatureID:     "feature1",
				expectedWPTMetricView: backend.SubtestCounts,
				expectedBrowsers: []backend.BrowserPathParam{
					backend.Chrome,
					backend.Edge,
					backend.Firefox,
					backend.Safari,
				},
				data: nil,
				err:  errTest,
			},
			expectedCallCount: 1,
			expectedResponse: backend.GetV1FeaturesIdStatusUsageChromiumDailyStats500JSONResponse{
				Code:    500,
				Message: "unable to get feature",
			},
			request: backend.GetV1FeaturesIdStatusUsageChromiumDailyStatsRequestObject{
				Id: "feature1",
			},
			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				getFeatureByIDConfig: tc.mockConfig,
				t:                    t,
			}
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil}

			// Call the function under test
			resp, err := myServer.GetV1FeaturesIDStatusUsageChromiumDailyStats(context.Background(), tc.request)

			// Assertions
			if mockStorer.callCountGetFeature != tc.expectedCallCount {
				t.Errorf("Incorrect call count: expected %d, got %d",
					tc.expectedCallCount,
					mockStorer.callCountGetFeature)
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
