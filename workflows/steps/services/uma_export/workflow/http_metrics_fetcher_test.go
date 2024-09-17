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
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

func TestHTTPMetricsFetcher_Fetch(t *testing.T) {
	tests := []struct {
		name         string
		queryName    metricdatatypes.UMAExportQuery
		expectedURL  string
		token        string
		tokenErr     error
		httpStatus   int
		responseBody string
		want         string
		err          error
	}{
		{
			name:       "success",
			queryName:  metricdatatypes.WebDXFeaturesQuery,
			token:      "test-token",
			httpStatus: http.StatusOK,
			responseBody: `{
				"queryName": "WebFeatureObserverDailyMetrics",
				"rows": []
			}`,
			want: `{
				"queryName": "WebFeatureObserverDailyMetrics",
				"rows": []
			}`,
			expectedURL: "https://uma-export.appspot.com/webstatus/usecounter.webdxfeatures",
			err:         nil,
			tokenErr:    nil,
		},
		{
			name:        "error generating token",
			queryName:   metricdatatypes.WebDXFeaturesQuery,
			tokenErr:    errors.New("some error"),
			httpStatus:  http.StatusOK,
			expectedURL: "",
			err:         errGeneratingToken,
			// Empty values that won't be checked
			want:         "",
			responseBody: "",
			token:        "",
		},
		{
			name:        "error unexpected status code",
			queryName:   metricdatatypes.WebDXFeaturesQuery,
			token:       "test-token",
			httpStatus:  http.StatusInternalServerError,
			expectedURL: "",
			err:         errUnexpectedStatusCode,
			// Empty values that won't be checked
			want:         "",
			responseBody: "",
			tokenErr:     nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock RoundTripper
			mockTransport := &mockRoundTripper{
				// nolint:exhaustruct // WONTFIX - external struct.
				response: &http.Response{
					StatusCode: tc.httpStatus,
					Body:       io.NopCloser(strings.NewReader(tc.responseBody)),
				},
			}

			// Create a mock token generator
			mockTokenGen := &mockTokenGenerator{
				lastURL: "",
				token:   &tc.token,
				err:     tc.tokenErr,
			}

			// Create the HTTP metrics fetcher with the mock transport
			fetcher, err := NewHTTPMetricsFetcher(mockTokenGen)
			if err != nil {
				t.Fatalf("unable to create fetcher. err %s", err)
			}

			// nolint:exhaustruct // WONTFIX - external struct.
			fetcher.httpClient = &http.Client{
				Transport: mockTransport,
			}

			got, err := fetcher.Fetch(context.Background(), tc.queryName)
			if !errors.Is(err, tc.err) {
				t.Errorf("Fetch() error = %s, expected %s", err, tc.err)

				return
			}

			if err == nil {
				body, err := io.ReadAll(got)
				if err != nil {
					t.Errorf("Failed to read response body: %v", err)

					return
				}
				got.Close()

				if strings.TrimSpace(string(body)) != tc.want {
					t.Errorf("Fetch() got = %v, want %v", string(body), tc.want)
				}

				// Assert that the request URL is correct
				if !strings.HasPrefix(mockTokenGen.lastURL, umaQueryServer) {
					t.Errorf("Fetch() used incorrect URL prefix, got = %v, want prefix %v", mockTokenGen.lastURL, umaQueryServer)
				}

				if mockTokenGen.lastURL != tc.expectedURL {
					t.Errorf("Fetch() used incorrect URL, got = %v, want %v", mockTokenGen.lastURL, tc.expectedURL)
				}
			}
		})
	}
}

// mockRoundTripper is a mock implementation of http.RoundTripper.
type mockRoundTripper struct {
	response *http.Response
}

func (m *mockRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return m.response, nil
}

type mockTokenGenerator struct {
	token   *string
	err     error
	lastURL string
}

func (m *mockTokenGenerator) Generate(_ context.Context, url string) (*string, error) {
	m.lastURL = url

	return m.token, m.err
}
