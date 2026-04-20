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
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func TestGetSubscriptionRSS(t *testing.T) {
	testCases := []struct {
		name                 string
		subCfg               *MockGetSavedSearchSubscriptionPublicConfig
		searchCfg            *MockGetSavedSearchPublicConfig
		eventsCfg            *MockListSavedSearchNotificationEventsConfig
		expectedStatusCode   int
		expectedContentType  string
		expectedBodyContains []string
	}{
		{
			name: "success",
			subCfg: &MockGetSavedSearchSubscriptionPublicConfig{
				expectedSubscriptionID: "sub-id",
				output: &backend.SubscriptionResponse{
					Id: "sub-id",
					Subscribable: backend.SavedSearchInfo{
						Id:   "search-id",
						Name: "",
					},
					ChannelId: "",
					CreatedAt: time.Time{},
					Frequency: backend.SubscriptionFrequencyImmediate,
					Triggers:  nil,
					UpdatedAt: time.Time{},
				},
				err: nil,
			},
			searchCfg: &MockGetSavedSearchPublicConfig{
				expectedSavedSearchID: "search-id",
				output: &backend.SavedSearchResponse{
					Id:             "search-id",
					Name:           "test search",
					Query:          "query",
					BookmarkStatus: nil,
					CreatedAt:      time.Time{},
					Description:    nil,
					Permissions:    nil,
					UpdatedAt:      time.Time{},
				},
				err: nil,
			},
			eventsCfg: &MockListSavedSearchNotificationEventsConfig{
				expectedSavedSearchID: "search-id",
				expectedSnapshotType:  string(backend.SubscriptionFrequencyImmediate),
				expectedPageSize:      100,
				expectedPageToken:     nil,
				output: []backendtypes.SavedSearchNotificationEvent{
					{
						ID:            "event-1",
						SavedSearchID: "search-id",
						SnapshotType:  string(backend.SubscriptionFrequencyImmediate),
						Timestamp:     time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
						EventType:     "IMMEDIATE_DIFF",
						Summary:       []byte(`"summary"`),
						Reasons:       nil,
						BlobPath:      "",
						DiffBlobPath:  "",
					},
				},
				outputNextPageToken: nil,
				err:                 nil,
			},
			expectedStatusCode:  200,
			expectedContentType: "application/rss+xml",
			expectedBodyContains: []string{
				"<title>WebStatus.dev - test search</title>",
				"<description>RSS feed for saved search: test search</description>",
				"<guid isPermaLink=\"false\">event-1</guid>",
				"<pubDate>Thu, 01 Jan 2026 12:00:00 +0000</pubDate>",
				"<link>http://localhost:8080/features?q=query</link>",
			},
		},
		{
			name: "success with pagination",
			subCfg: &MockGetSavedSearchSubscriptionPublicConfig{
				expectedSubscriptionID: "sub-id",
				output: &backend.SubscriptionResponse{
					Id: "sub-id",
					Subscribable: backend.SavedSearchInfo{
						Id:   "search-id",
						Name: "",
					},
					ChannelId: "",
					CreatedAt: time.Time{},
					Frequency: backend.SubscriptionFrequencyImmediate,
					Triggers:  nil,
					UpdatedAt: time.Time{},
				},
				err: nil,
			},
			searchCfg: &MockGetSavedSearchPublicConfig{
				expectedSavedSearchID: "search-id",
				output: &backend.SavedSearchResponse{
					Id:             "search-id",
					Name:           "test search",
					Query:          "query",
					BookmarkStatus: nil,
					CreatedAt:      time.Time{},
					Description:    nil,
					Permissions:    nil,
					UpdatedAt:      time.Time{},
				},
				err: nil,
			},
			eventsCfg: &MockListSavedSearchNotificationEventsConfig{
				expectedSavedSearchID: "search-id",
				expectedSnapshotType:  string(backend.SubscriptionFrequencyImmediate),
				expectedPageSize:      100,
				expectedPageToken:     nil,
				output: []backendtypes.SavedSearchNotificationEvent{
					{
						ID:            "event-1",
						SavedSearchID: "search-id",
						SnapshotType:  string(backend.SubscriptionFrequencyImmediate),
						Timestamp:     time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
						EventType:     "IMMEDIATE_DIFF",
						Summary:       []byte(`"summary"`),
						Reasons:       nil,
						BlobPath:      "",
						DiffBlobPath:  "",
					},
				},
				outputNextPageToken: &[]string{"next-token"}[0],
				err:                 nil,
			},
			expectedStatusCode:  200,
			expectedContentType: "application/rss+xml",
			expectedBodyContains: []string{
				"<title>WebStatus.dev - test search</title>",
				"<description>RSS feed for saved search: test search</description>",
				"<guid isPermaLink=\"false\">event-1</guid>",
				"<pubDate>Thu, 01 Jan 2026 12:00:00 +0000</pubDate>",
				"<link>http://localhost:8080/features?q=query</link>",
				`<atom:link rel="next" ` +
					`href="http://localhost:8080/v1/subscriptions/sub-id/rss?page_size=100&amp;page_token=next-token">` +
					`</atom:link>`,
			},
		},
		{
			name: "subscription not found",
			subCfg: &MockGetSavedSearchSubscriptionPublicConfig{
				expectedSubscriptionID: "missing-sub",
				output:                 nil,
				err:                    backendtypes.ErrEntityDoesNotExist,
			},
			searchCfg:            nil,
			eventsCfg:            nil,
			expectedStatusCode:   404,
			expectedContentType:  "",
			expectedBodyContains: nil,
		},
		{
			name: "500 error",
			subCfg: &MockGetSavedSearchSubscriptionPublicConfig{
				expectedSubscriptionID: "sub-id",
				output:                 nil,
				err:                    errors.New("db error"),
			},
			searchCfg:            nil,
			eventsCfg:            nil,
			expectedStatusCode:   500,
			expectedContentType:  "",
			expectedBodyContains: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var mockStorer MockWPTMetricsStorer
			mockStorer.getSavedSearchSubscriptionPublicCfg = tc.subCfg
			mockStorer.getSavedSearchPublicCfg = tc.searchCfg
			mockStorer.listSavedSearchNotificationEventsCfg = tc.eventsCfg
			mockStorer.t = t

			myServer := Server{
				wptMetricsStorer:        &mockStorer,
				metadataStorer:          nil,
				userGitHubClientFactory: nil,
				eventPublisher:          nil,
				operationResponseCaches: nil,
				baseURL:                 getTestBaseURL(t),
			}

			req := httptest.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				"/v1/subscriptions/"+tc.subCfg.expectedSubscriptionID+"/rss",
				nil,
			)

			// Fix createOpenAPIServerServer call
			srv := createOpenAPIServerServer("", &myServer, nil, noopMiddleware)

			w := httptest.NewRecorder()

			// Fix router.ServeHTTP to srv.Handler.ServeHTTP
			srv.Handler.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatusCode {
				t.Errorf("expected status code %d, got %d", tc.expectedStatusCode, resp.StatusCode)
			}

			if tc.expectedStatusCode == 200 {
				contentType := resp.Header.Get("Content-Type")
				if contentType != tc.expectedContentType {
					t.Errorf("expected content type %s, got %s", tc.expectedContentType, contentType)
				}

				bodyBytes, _ := io.ReadAll(resp.Body)
				bodyStr := string(bodyBytes)

				for _, searchStr := range tc.expectedBodyContains {
					if !strings.Contains(bodyStr, searchStr) {
						t.Errorf("expected body to contain %q, but it did not.\nBody:\n%s", searchStr, bodyStr)
					}
				}
			}
		})
	}
}
