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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestListChromiumDailyUsageStats(t *testing.T) {
	testCases := []struct {
		name               string
		mockConfig         *MockListChromiumDailyUsageStatsConfig
		expectedCallCount  int // For the mock method
		expectedCacheCalls []*ExpectedCacheCall
		expectedGetCalls   []*ExpectedGetCall
		request            *http.Request
		expectedResponse   *http.Response
	}{
		{
			name: "Success Case - no optional params - use defaults",
			mockConfig: &MockListChromiumDailyUsageStatsConfig{
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
						Usage:     valuePtr[float64](0.0),
					},
				},
			},
			expectedCacheCalls: nil,
			expectedGetCalls:   nil,
			expectedCallCount:  1,
			expectedResponse: testJSONResponse(200, `
{
	"data":[
		{
			"timestamp":"2000-01-01T00:00:00Z",
			"usage":0
		}
	],
	"metadata":{

	}
}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/features/feature1/stats/usage/chromium/daily_stats?startAt=2000-01-01&endAt=2000-01-10", nil),
		},
		{
			name: "400 case - invalid page token",
			mockConfig: &MockListChromiumDailyUsageStatsConfig{
				expectedFeatureID: "feature1",
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  100,
				expectedPageToken: badPageToken,
				pageToken:         nil,
				err:               backendtypes.ErrInvalidPageToken,
				data:              nil,
			},
			expectedCacheCalls: nil,
			expectedGetCalls:   nil,
			expectedCallCount:  1,
			expectedResponse:   testJSONResponse(400, `{"code":400,"message":"invalid page token"}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/features/feature1/stats/usage/chromium/daily_stats?"+
					"startAt=2000-01-01&endAt=2000-01-10&page_token="+*badPageToken, nil),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				listChromiumDailyUsageStatsCfg: tc.mockConfig,
				t:                              t,
			}
			mockCacher := NewMockRawBytesDataCacher(t, tc.expectedCacheCalls, tc.expectedGetCalls)
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil,
				operationResponseCaches: initOperationResponseCaches(mockCacher)}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse)
			assertMocksExpectations(t, tc.expectedCallCount, mockStorer.callCountListChromiumDailyUsageStats,
				"ListChromiumDailyUsageStats", mockCacher)
		})
	}
}
