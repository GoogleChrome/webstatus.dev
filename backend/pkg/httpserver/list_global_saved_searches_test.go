// Copyright 2026 Google LLC
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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestListGlobalSavedSearches(t *testing.T) {
	testTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	token := "next-page-token"

	testCases := []struct {
		name               string
		mockCfg            *MockListGlobalSavedSearchesConfig
		expectedCallCount  int
		expectedGetCalls   []*ExpectedGetCall
		expectedCacheCalls []*ExpectedCacheCall
		request            *http.Request
		expectedResponse   *http.Response
	}{
		{
			name: "Success - Cache Miss - return list of global saved searches",
			mockCfg: &MockListGlobalSavedSearchesConfig{
				expectedPageSize:  100, // default page size
				expectedPageToken: nil,
				output: &backend.GlobalSavedSearchPage{
					Metadata: &backend.PageMetadata{
						NextPageToken: &token,
					},
					Data: &[]backend.GlobalSavedSearch{
						{
							Id:           new("id-1"),
							Name:         "Search 1",
							Description:  new("Description 1"),
							Query:        "q1",
							CreatedAt:    &testTime,
							UpdatedAt:    &testTime,
							DisplayOrder: new(int64(1)),
						},
					},
				},
				err: nil,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `listGlobalSavedSearches-{"Params":{}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: []*ExpectedCacheCall{
				{
					Key: `listGlobalSavedSearches-{"Params":{}}`,
					Value: []byte(
						`{"data":[{"created_at":"2026-01-01T00:00:00Z","description":"Description 1","display_order":1,` +
							`"id":"id-1","name":"Search 1","query":"q1","updated_at":"2026-01-01T00:00:00Z"}],` +
							`"metadata":{"next_page_token":"next-page-token"}}`,
					),
					CacheCfg: getTestAggregatedCacheConfig(),
				},
			},
			expectedCallCount: 1,
			request: httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"/v1/global-saved-searches",
				nil,
			),
			expectedResponse: testJSONResponse(200, `
{
	"data": [
		{
			"created_at": "2026-01-01T00:00:00Z",
			"description": "Description 1",
			"display_order": 1,
			"id": "id-1",
			"name": "Search 1",
			"query": "q1",
			"updated_at": "2026-01-01T00:00:00Z"
		}
	],
	"metadata": {
		"next_page_token": "next-page-token"
	}
}
`),
		},
		{
			name:    "Success - Cache Hit - return list of global saved searches from cache",
			mockCfg: nil, // Should not call DB
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key: `listGlobalSavedSearches-{"Params":{}}`,
					Value: []byte(
						`{"data":[{"created_at":"2026-01-01T00:00:00Z","description":"Description 1","display_order":1,` +
							`"id":"id-1","name":"Search 1","query":"q1","updated_at":"2026-01-01T00:00:00Z"}],` +
							`"metadata":{"next_page_token":"next-page-token"}}`,
					),
					Err: nil,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  0,
			request: httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"/v1/global-saved-searches",
				nil,
			),
			expectedResponse: testJSONResponse(200, `
{
	"data": [
		{
			"created_at": "2026-01-01T00:00:00Z",
			"description": "Description 1",
			"display_order": 1,
			"id": "id-1",
			"name": "Search 1",
			"query": "q1",
			"updated_at": "2026-01-01T00:00:00Z"
		}
	],
	"metadata": {
		"next_page_token": "next-page-token"
	}
}
`),
		},
		{
			name: "Error - invalid page token maps to 400 (Cache Miss)",
			mockCfg: &MockListGlobalSavedSearchesConfig{
				expectedPageSize:  100,
				expectedPageToken: nil,
				output:            nil,
				err:               backendtypes.ErrInvalidPageToken,
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `listGlobalSavedSearches-{"Params":{}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  1,
			request: httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"/v1/global-saved-searches",
				nil,
			),
			expectedResponse: testJSONResponse(400, `
{
	"code": 400,
	"message": "invalid page token"
}
`),
		},
		{
			name: "Error - other errors map to 500 (Cache Miss)",
			mockCfg: &MockListGlobalSavedSearchesConfig{
				expectedPageSize:  100,
				expectedPageToken: nil,
				output:            nil,
				err:               errors.New("db error"),
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `listGlobalSavedSearches-{"Params":{}}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedCacheCalls: nil,
			expectedCallCount:  1,
			request: httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"/v1/global-saved-searches",
				nil,
			),
			expectedResponse: testJSONResponse(500, `
{
	"code": 500,
	"message": "unable to get list of global saved searches"
}
`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// nolint:exhaustruct
			mockStorer := &MockWPTMetricsStorer{
				listGlobalSavedSearchesCfg: tc.mockCfg,
				t:                          t,
			}
			mockCacher := NewMockRawBytesDataCacher(t, tc.expectedCacheCalls, tc.expectedGetCalls)
			myServer := setupTestServer(t,
				withCustomStorer(mockStorer),
				withCustomCaches(initOperationResponseCaches(mockCacher, getTestRouteCacheOptions())),
			)

			assertTestServerRequest(t, myServer, tc.request, tc.expectedResponse)
			assertMocksExpectations(t, tc.expectedCallCount, mockStorer.callCountListGlobalSavedSearches,
				"ListGlobalSavedSearches", mockCacher)
		})
	}
}
