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

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestListChromiumDailyUsageStats(t *testing.T) {
	testCases := []struct {
		name              string
		mockConfig        MockListChromiumDailyUsageStatsConfig
		expectedCallCount int // For the mock method
		request           backend.ListChromiumDailyUsageStatsRequestObject
		expectedResponse  backend.ListChromiumDailyUsageStatsResponseObject
		expectedError     error
	}{
		{
			name: "Success Case - no optional params - use defaults",
			mockConfig: MockListChromiumDailyUsageStatsConfig{
				expectedFeatureID: "feature1",
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  100,
				expectedPageToken: nil,
				pageToken:         nil,
				err:               nil,

				data: []backend.ChromiumUsageStat{
					{
						Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						Usage:     nil,
					},
				},
			},
			expectedCallCount: 1,
			expectedResponse: backend.ListChromiumDailyUsageStats200JSONResponse{
				Data: []backend.ChromiumUsageStat{
					{
						Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
						Usage:     nil,
					},
				},
				Metadata: &backend.PageMetadata{
					NextPageToken: nil,
				},
			},
			request: backend.ListChromiumDailyUsageStatsRequestObject{
				Params: backend.ListChromiumDailyUsageStatsParams{
					StartAt:   openapi_types.Date{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)},
					EndAt:     openapi_types.Date{Time: time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC)},
					PageToken: nil,
					PageSize:  nil,
				},
				FeatureId: "feature1",
			},
			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				listChromiumDailyUsageStatsCfg: tc.mockConfig,
				t:                              t,
			}
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil}

			// Call the function under test
			resp, err := myServer.ListChromiumDailyUsageStats(context.Background(), tc.request)

			// Assertions
			if mockStorer.callCountListChromiumDailyUsageStats != tc.expectedCallCount {
				t.Errorf("Incorrect call count: expected %d, got %d",
					tc.expectedCallCount,
					mockStorer.callCountListChromiumDailyUsageStats)
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
