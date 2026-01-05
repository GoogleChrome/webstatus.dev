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

package gcppubsubadapters

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/google/go-cmp/cmp"
)

func testSavedSearchResponse(id string, query string, updatedAt time.Time) *backend.SavedSearchResponse {
	var resp backend.SavedSearchResponse
	resp.Id = id
	resp.Query = query
	resp.UpdatedAt = updatedAt

	return &resp
}
func TestSearchConfigurationPublisherAdapter_Publish(t *testing.T) {
	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		resp         *backend.SavedSearchResponse
		userID       string
		isCreation   bool
		publishErr   error
		wantErr      bool
		expectedJSON string
	}{
		{
			name:       "success creation",
			resp:       testSavedSearchResponse("search-123", "group:css", fixedTime),
			userID:     "user-1",
			isCreation: true,
			publishErr: nil,
			wantErr:    false,
			expectedJSON: `{
				"apiVersion": "v1",
				"kind": "SearchConfigurationChangedEvent",
				"data": {
					"search_id": "search-123",
					"query": "group:css",
					"user_id": "user-1",
					"timestamp": "2025-01-01T00:00:00Z",
					"is_creation": true,
					"frequency": "IMMEDIATE"
				}
			}`,
		},
		{
			name:       "success update",
			resp:       testSavedSearchResponse("search-456", "group:html", fixedTime.Add(24*time.Hour)),
			userID:     "user-1",
			isCreation: false,
			publishErr: nil,
			wantErr:    false,
			expectedJSON: `{
				"apiVersion": "v1",
				"kind": "SearchConfigurationChangedEvent",
				"data": {
					"search_id": "search-456",
					"query": "group:html",
					"user_id": "user-1",
					"timestamp": "2025-01-02T00:00:00Z",
					"is_creation": false,
					"frequency": "IMMEDIATE"
				}
			}`,
		},
		{
			name:         "publish error",
			resp:         testSavedSearchResponse("search-err", "group:html", fixedTime),
			userID:       "user-1",
			isCreation:   false,
			publishErr:   errors.New("pubsub error"),
			wantErr:      true,
			expectedJSON: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			publisher := new(mockPublisher)
			publisher.err = tc.publishErr
			adapter := NewBackendAdapter(publisher, "test-topic")

			err := adapter.PublishSearchConfigurationChanged(context.Background(), tc.resp, tc.userID, tc.isCreation)

			if (err != nil) != tc.wantErr {
				t.Errorf("PublishSearchConfigurationChanged() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.wantErr {
				return
			}

			if publisher.publishedTopic != "test-topic" {
				t.Errorf("Topic mismatch: got %s, want test-topic", publisher.publishedTopic)
			}

			// Unmarshal actual data
			var actual interface{}
			if err := json.Unmarshal(publisher.publishedData, &actual); err != nil {
				t.Fatalf("failed to unmarshal published data: %v", err)
			}

			// Unmarshal expected data
			var expected interface{}
			if err := json.Unmarshal([]byte(tc.expectedJSON), &expected); err != nil {
				t.Fatalf("failed to unmarshal expected data: %v", err)
			}

			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Errorf("Payload mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
