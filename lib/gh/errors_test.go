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

package gh

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-github/v79/github"
)

//nolint:exhaustruct
func TestErrorClassification(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		err        error
		wantClient bool
		wantServer bool
	}{
		{
			name:       "nil error",
			err:        nil,
			wantClient: false,
			wantServer: false,
		},
		{
			name:       "standard non-github error",
			err:        errors.New("arbitrary network error"),
			wantClient: false,
			wantServer: false,
		},
		{
			name: "github error response with nil Response",
			err: &github.ErrorResponse{
				Response: nil,
				Message:  "error with nil response",
			},
			wantClient: false,
			wantServer: false,
		},
		{
			name: "400 Bad Request",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusBadRequest,
				},
				Message: "Bad Request",
			},
			wantClient: true,
			wantServer: false,
		},
		{
			name: "401 Unauthorized",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusUnauthorized,
				},
				Message: "Unauthorized",
			},
			wantClient: true,
			wantServer: false,
		},
		{
			name: "403 Forbidden",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusForbidden,
				},
				Message: "Forbidden",
			},
			wantClient: true,
			wantServer: false,
		},
		{
			name: "404 Not Found",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusNotFound,
				},
				Message: "Not Found",
			},
			wantClient: true,
			wantServer: false,
		},
		{
			name: "422 Unprocessable Entity",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusUnprocessableEntity,
				},
				Message: "Unprocessable Entity",
			},
			wantClient: true,
			wantServer: false,
		},
		{
			name: "wrapped 404 error",
			err: fmt.Errorf("calling github api: %w", &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusNotFound,
				},
				Message: "Not Found",
			}),
			wantClient: true,
			wantServer: false,
		},
		{
			name: "500 Internal Server Error",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusInternalServerError,
				},
				Message: "Internal Server Error",
			},
			wantClient: false,
			wantServer: true,
		},
		{
			name: "502 Bad Gateway",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusBadGateway,
				},
				Message: "Bad Gateway",
			},
			wantClient: false,
			wantServer: true,
		},
		{
			name: "503 Service Unavailable",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusServiceUnavailable,
				},
				Message: "Service Unavailable",
			},
			wantClient: false,
			wantServer: true,
		},
		{
			name: "504 Gateway Timeout",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusGatewayTimeout,
				},
				Message: "Gateway Timeout",
			},
			wantClient: false,
			wantServer: true,
		},
		{
			name: "wrapped 503 error",
			err: fmt.Errorf("service failed: %w", &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusServiceUnavailable,
				},
				Message: "Service Unavailable",
			}),
			wantClient: false,
			wantServer: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := IsClientError(tc.err); got != tc.wantClient {
				t.Errorf("IsClientError() = %v, want %v", got, tc.wantClient)
			}
			if got := IsServerError(tc.err); got != tc.wantServer {
				t.Errorf("IsServerError() = %v, want %v", got, tc.wantServer)
			}
		})
	}
}

//nolint:exhaustruct
func TestRateLimitAndPermanentClientErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		err           error
		wantRateLimit bool
		wantPermanent bool
	}{
		{
			name:          "nil error",
			err:           nil,
			wantRateLimit: false,
			wantPermanent: false,
		},
		{
			name:          "ErrSecondaryRateLimit sentinel",
			err:           ErrSecondaryRateLimit,
			wantRateLimit: true,
			wantPermanent: false,
		},
		{
			name:          "wrapped ErrSecondaryRateLimit",
			err:           fmt.Errorf("request failed: %w", ErrSecondaryRateLimit),
			wantRateLimit: true,
			wantPermanent: false,
		},
		{
			name: "HTTP 429 Too Many Requests ErrorResponse",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusTooManyRequests,
				},
				Message: "API rate limit exceeded",
			},
			wantRateLimit: true,
			wantPermanent: false,
		},
		{
			name: "HTTP 403 Forbidden with Secondary Rate Limit message",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusForbidden,
				},
				Message: "You have exceeded a secondary rate limit. Please wait a few minutes.",
			},
			wantRateLimit: true,
			wantPermanent: false,
		},
		{
			name: "github.RateLimitError",
			err: &github.RateLimitError{
				Response: &http.Response{
					StatusCode: http.StatusForbidden,
				},
				Message: "API rate limit exceeded for user ID",
			},
			wantRateLimit: true,
			wantPermanent: false,
		},
		{
			name: "github.AbuseRateLimitError",
			err: &github.AbuseRateLimitError{
				Response: &http.Response{
					StatusCode: http.StatusForbidden,
				},
				Message: "You have triggered an abuse detection mechanism.",
			},
			wantRateLimit: true,
			wantPermanent: false,
		},
		{
			name: "Standard HTTP 403 Forbidden (Non-Rate Limit)",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusForbidden,
				},
				Message: "Resource not accessible by integration",
			},
			wantRateLimit: false,
			wantPermanent: true,
		},
		{
			name: "HTTP 404 Not Found",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusNotFound,
				},
				Message: "Not Found",
			},
			wantRateLimit: false,
			wantPermanent: true,
		},
		{
			name: "HTTP 401 Unauthorized",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusUnauthorized,
				},
				Message: "Bad credentials",
			},
			wantRateLimit: false,
			wantPermanent: true,
		},
		{
			name: "HTTP 422 Unprocessable Entity",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusUnprocessableEntity,
				},
				Message: "Validation Failed",
			},
			wantRateLimit: false,
			wantPermanent: true,
		},
		{
			name: "HTTP 500 Internal Server Error",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusInternalServerError,
				},
				Message: "Internal Server Error",
			},
			wantRateLimit: false,
			wantPermanent: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := IsRateLimitError(tc.err); got != tc.wantRateLimit {
				t.Errorf("IsRateLimitError() = %v, want %v", got, tc.wantRateLimit)
			}
			if got := IsPermanentClientError(tc.err); got != tc.wantPermanent {
				t.Errorf("IsPermanentClientError() = %v, want %v", got, tc.wantPermanent)
			}
		})
	}
}

func TestNewHTTPErrorResponseForTest(t *testing.T) {
	t.Parallel()

	err := NewHTTPErrorResponseForTest(http.StatusNotFound, "Not Found")
	if err == nil {
		t.Fatalf("expected non-nil error")
	}

	if !IsClientError(err) {
		t.Errorf("expected IsClientError to return true for 404")
	}
	if !IsPermanentClientError(err) {
		t.Errorf("expected IsPermanentClientError to return true for 404")
	}
	if IsServerError(err) {
		t.Errorf("expected IsServerError to return false for 404")
	}
	if IsRateLimitError(err) {
		t.Errorf("expected IsRateLimitError to return false for 404")
	}

	serverErr := NewHTTPErrorResponseForTest(http.StatusServiceUnavailable, "Service Unavailable")
	if !IsServerError(serverErr) {
		t.Errorf("expected IsServerError to return true for 503")
	}
	if IsClientError(serverErr) {
		t.Errorf("expected IsClientError to return false for 503")
	}
}
