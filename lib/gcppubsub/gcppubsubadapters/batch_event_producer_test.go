// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/google/go-cmp/cmp"
)

func TestBatchFanOutPublisherAdapter_PublishRefreshCommand(t *testing.T) {
	tests := []struct {
		name         string
		cmd          workertypes.RefreshSearchCommand
		publishErr   error
		wantErr      bool
		expectedJSON string
	}{
		{
			name: "success",
			cmd: workertypes.RefreshSearchCommand{
				SearchID:  "search-123",
				Query:     "query=abc",
				Frequency: workertypes.FrequencyImmediate,
				Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			publishErr: nil,
			wantErr:    false,
			expectedJSON: `{
				"apiVersion": "v1",
				"kind": "RefreshSearchCommand",
				"data": {
					"search_id": "search-123",
					"query": "query=abc",
					"frequency": "IMMEDIATE",
					"timestamp": "2025-01-01T00:00:00Z"
				}
			}`,
		},
		{
			name: "publish error",
			cmd: workertypes.RefreshSearchCommand{
				SearchID:  "search-123",
				Query:     "query=abc",
				Frequency: workertypes.FrequencyImmediate,
				Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			publishErr:   errors.New("pubsub error"),
			wantErr:      true,
			expectedJSON: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			publisher := new(mockPublisher)
			publisher.err = tc.publishErr
			adapter := NewBatchFanOutPublisherAdapter(publisher, "test-topic")

			err := adapter.PublishRefreshCommand(context.Background(), tc.cmd)

			if (err != nil) != tc.wantErr {
				t.Errorf("PublishRefreshCommand() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr {
				verifyJSONPayload(t, publisher.publishedData, tc.expectedJSON)
			}
		})
	}
}

func verifyJSONPayload(t *testing.T, actualBytes []byte, expectedJSON string) {
	t.Helper()

	var actual interface{}
	if err := json.Unmarshal(actualBytes, &actual); err != nil {
		t.Fatalf("failed to unmarshal actual data: %v", err)
	}

	var expected interface{}
	if err := json.Unmarshal([]byte(expectedJSON), &expected); err != nil {
		t.Fatalf("failed to unmarshal expected data: %v", err)
	}

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("Payload mismatch (-want +got):\n%s", diff)
	}
}
