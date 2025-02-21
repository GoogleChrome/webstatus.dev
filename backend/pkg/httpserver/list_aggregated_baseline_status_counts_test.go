// Copyright 2025 Google LLC
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

func TestListAggregatedBaselineStatusCounts(t *testing.T) {
	testCases := []struct {
		name               string
		mockConfig         *MockListBaselineStatusCountsConfig
		expectedCallCount  int
		expectedCacheCalls []*ExpectedCacheCall
		expectedGetCalls   []*ExpectedGetCall
		request            *http.Request
		expectedResponse   *http.Response
	}{
		{
			name: "Success Case - no optional params - use defaults",
			mockConfig: &MockListBaselineStatusCountsConfig{
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  100,
				expectedPageToken: nil,
				pageToken:         nil,
				err:               nil,
				page: &backend.BaselineStatusMetricsPage{
					Metadata: &backend.PageMetadata{
						NextPageToken: nil,
					},
					Data: []backend.BaselineStatusMetric{
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
			expectedCacheCalls: nil,
			expectedGetCalls:   nil,
			expectedCallCount:  1,
			expectedResponse: testJSONResponse(200, `
{
	"data":[
		{
			"count":10,
			"timestamp":"2000-01-10T00:00:00Z"
		},
		{
			"count":9,
			"timestamp":"2000-01-09T00:00:00Z"
		}
	],
	"metadata":{

	}
}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/baseline_status/low_date_feature_counts?"+
					"startAt=2000-01-01&endAt=2000-01-10", nil),
		},
		{
			name: "Success Case - include optional params",
			mockConfig: &MockListBaselineStatusCountsConfig{
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  50,
				expectedPageToken: inputPageToken,
				err:               nil,
				page: &backend.BaselineStatusMetricsPage{
					Metadata: &backend.PageMetadata{
						NextPageToken: nextPageToken,
					},
					Data: []backend.BaselineStatusMetric{
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
			expectedCacheCalls: nil,
			expectedGetCalls:   nil,
			expectedCallCount:  1,
			expectedResponse: testJSONResponse(200, `
{
	"data":[
		{
			"count":10,
			"timestamp":"2000-01-10T00:00:00Z"
		},
		{
			"count":9,
			"timestamp":"2000-01-09T00:00:00Z"
		}
	],
	"metadata":{
		"next_page_token":"next-page-token"
	}
}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/baseline_status/low_date_feature_counts?"+
					"startAt=2000-01-01&endAt=2000-01-10&page_size=50&page_token="+*inputPageToken, nil),
		},
		{
			name: "500 case",
			mockConfig: &MockListBaselineStatusCountsConfig{
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  100,
				expectedPageToken: nil,
				page:              nil,
				pageToken:         nil,
				err:               errTest,
			},
			expectedCacheCalls: nil,
			expectedGetCalls:   nil,
			expectedCallCount:  1,
			expectedResponse: testJSONResponse(
				500, `{"code":500,"message":"unable to get missing one implementation metrics"}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/baseline_status/low_date_feature_counts?"+
					"startAt=2000-01-01&endAt=2000-01-10", nil),
		},
		{
			name: "400 case - invalid page token",
			mockConfig: &MockListBaselineStatusCountsConfig{
				expectedStartAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				expectedEndAt:     time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC),
				expectedPageSize:  100,
				expectedPageToken: badPageToken,
				page:              nil,
				pageToken:         nil,
				err:               backendtypes.ErrInvalidPageToken,
			},
			expectedCacheCalls: nil,
			expectedGetCalls:   nil,
			expectedCallCount:  1,
			expectedResponse:   testJSONResponse(400, `{"code":400,"message":"invalid page token"}`),
			request: httptest.NewRequest(http.MethodGet,
				"/v1/stats/baseline_status/low_date_feature_counts?"+
					"startAt=2000-01-01&endAt=2000-01-10&page_token"+*badPageToken, nil),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint: exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				listBaselineStatusCountsCfg: tc.mockConfig,
				t:                           t,
			}
			mockCacher := NewMockRawBytesDataCacher(t, tc.expectedCacheCalls, tc.expectedGetCalls)
			myServer := Server{wptMetricsStorer: mockStorer, metadataStorer: nil,
				operationResponseCaches: initOperationResponseCaches(mockCacher)}
			assertTestServerRequest(t, &myServer, tc.request, tc.expectedResponse)
			assertMocksExpectations(t, tc.expectedCallCount, mockStorer.callCountListBaselineStatusCounts,
				"ListBaselineStatusCounts", mockCacher)
		})
	}
}
