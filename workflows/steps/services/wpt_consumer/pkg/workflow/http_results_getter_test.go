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

package workflow

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/fetchtypes"
	"github.com/web-platform-tests/wpt.fyi/api/query"
)

func TestHTTPResultsGetter_DownloadResults(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		responseBody  string
		statusCode    int
		wantData      ResultsSummaryFile
		wantErrOfKind error
	}{
		{
			name:         "Success v2 file",
			url:          "http://test.example/results.json",
			responseBody: `{"test1": {"c": [1, 1], "s": "O"}}`,
			statusCode:   http.StatusOK,
			wantData: ResultsSummaryFileV2{
				"test1": query.SummaryResult{
					Counts: []int{1, 1},
					Status: "O",
				},
			},
			wantErrOfKind: nil,
		},
		{
			name:          "Invalid JSON",
			url:           "http://test.example/invalid.json",
			responseBody:  `{not valid JSON}`,
			statusCode:    http.StatusOK,
			wantData:      nil,
			wantErrOfKind: ErrFailedToParseResults,
		},
		{
			name:          "Context Cancellation",
			url:           "http://test.example/timeout",
			responseBody:  ``, // No response due to timeout
			statusCode:    http.StatusOK,
			wantData:      nil,
			wantErrOfKind: fetchtypes.ErrFailedToFetch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Server Setup
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, writeErr := w.Write([]byte(tt.responseBody))
				if writeErr != nil {
					t.Fatalf("failed to send data in mock server. %s", writeErr)
				}
			}))
			defer server.Close()

			// HTTP Client Mocking
			getter := NewHTTPResultsGetter()
			getter.client = *server.Client()

			// Test with Context (if applicable)
			var ctx context.Context
			var cancel context.CancelFunc
			if tt.name == "Context Cancellation" {
				ctx, cancel = context.WithCancel(context.Background())
				cancel() // Immediately cancel
			} else {
				ctx = context.Background()
			}

			gotData, err := getter.DownloadResults(ctx, server.URL)

			// Assertions
			if !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("Unexpected results. got = %v, want = %v", gotData, tt.wantData)
			}
			if !errors.Is(err, tt.wantErrOfKind) {
				t.Errorf("Unexpected error. got = %v, want = %v", err, tt.wantErrOfKind)
			}
		})
	}
}
