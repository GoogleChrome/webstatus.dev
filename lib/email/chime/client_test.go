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

package chime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

// mockHTTPClient allows faking HTTP responses for tests.
type mockHTTPClient struct {
	response *http.Response
	err      error
}

func (m *mockHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	return m.response, m.err
}

// mockTokenSource is a dummy token source for tests.
type mockTokenSource struct {
	token *oauth2.Token
	err   error
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	return m.token, m.err
}

func newTestSender(mockClient HTTPClient) *Sender {
	return &Sender{
		bcc: []string{"bcc@example.com"},
		// nolint:exhaustruct  // WONTFIX - external struct.
		tokenSource: &mockTokenSource{token: &oauth2.Token{AccessToken: "fake-token"}, err: nil},
		httpClient:  mockClient,
		fromAddress: "test-from@example.com",
		baseURL:     "https://fake-chime.googleapis.com",
	}
}

func TestSend(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		name          string
		mockClient    *mockHTTPClient
		id            string
		expectedError error
	}{
		{
			name: "Success - SENT outcome",
			mockClient: &mockHTTPClient{
				// nolint:exhaustruct  // WONTFIX - external struct.
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"details": {"outcome": "SENT"}}`)),
					Header:     make(http.Header),
				},
				err: nil,
			},
			id:            "success-id",
			expectedError: nil,
		},
		{
			name: "Duplicate Notification - 409 Conflict",
			mockClient: &mockHTTPClient{
				// nolint:exhaustruct  // WONTFIX - external struct.
				response: &http.Response{
					StatusCode: http.StatusConflict,
					Body:       io.NopCloser(strings.NewReader("Duplicate")),
					Header:     make(http.Header),
				},
				err: nil,
			},
			id:            "duplicate-id",
			expectedError: ErrDuplicate,
		},
		{
			name: "Permanent User Error - PREFERENCE_DROPPED",
			mockClient: &mockHTTPClient{
				// nolint:exhaustruct  // WONTFIX - external struct.
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"details": {"outcome": "PREFERENCE_DROPPED"}}`)),
					Header:     make(http.Header),
				},
				err: nil,
			},
			id:            "user-error-id",
			expectedError: ErrPermanentUser,
		},
		{
			name: "Permanent System Error - INVALID_REQUEST_DROPPED",
			mockClient: &mockHTTPClient{
				// nolint:exhaustruct  // WONTFIX - external struct.
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"details": {"outcome": "INVALID_REQUEST_DROPPED"}}`)),
					Header:     make(http.Header),
				},
				err: nil,
			},
			id:            "system-error-id",
			expectedError: ErrPermanentSystem,
		},
		{
			name: "Permanent System Error - 400 Bad Request",
			mockClient: &mockHTTPClient{
				// nolint:exhaustruct  // WONTFIX - external struct.
				response: &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader("Bad Request")),
					Header:     make(http.Header),
				},
				err: nil,
			},
			id:            "bad-request-id",
			expectedError: ErrPermanentSystem,
		},
		{
			name: "Transient Error - QUOTA_DROPPED",
			mockClient: &mockHTTPClient{
				// nolint:exhaustruct  // WONTFIX - external struct.
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"details": {"outcome": "QUOTA_DROPPED"}}`)),
					Header:     make(http.Header),
				},
				err: nil,
			},
			id:            "transient-quota-id",
			expectedError: ErrTransient,
		},
		{
			name: "Transient Error - 503 Server Error",
			mockClient: &mockHTTPClient{
				// nolint:exhaustruct  // WONTFIX - external struct.
				response: &http.Response{
					StatusCode: http.StatusServiceUnavailable,
					Body:       io.NopCloser(strings.NewReader("Server Error")),
					Header:     make(http.Header),
				},
				err: nil,
			},
			id:            "transient-server-error-id",
			expectedError: ErrTransient,
		},
		{
			name: "Network Error",
			mockClient: &mockHTTPClient{
				response: nil,
				err:      fmt.Errorf("network connection failed"),
			},
			id:            "network-error-id",
			expectedError: ErrTransient,
		},
		{
			name:          "Empty ID Error",
			mockClient:    &mockHTTPClient{response: nil, err: nil},
			id:            "",
			expectedError: ErrPermanentSystem,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sender := newTestSender(tc.mockClient)
			err := sender.Send(ctx, tc.id, "to@example.com", "Test Subject", "<h1>Test</h1>")

			if tc.expectedError != nil {
				if err == nil {
					t.Fatalf("Expected error but got nil")
				}
				if !errors.Is(err, tc.expectedError) {
					t.Errorf("Expected error wrapping %v, but got %v", tc.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got %v", err)
				}
			}
		})
	}
}
