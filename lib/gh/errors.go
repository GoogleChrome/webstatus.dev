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
	"net/http"
	"strings"

	"github.com/google/go-github/v79/github"
)

// IsRateLimitError reports whether err represents a GitHub rate limit error,
// including primary rate limits (429), secondary/abuse rate limits (403),
// and ErrSecondaryRateLimit.
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrSecondaryRateLimit) {
		return true
	}

	var rateLimitErr *github.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return true
	}

	var abuseErr *github.AbuseRateLimitError
	if errors.As(err, &abuseErr) {
		return true
	}

	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) && ghErr.Response != nil {
		if ghErr.Response.StatusCode == http.StatusTooManyRequests {
			return true
		}

		if ghErr.Response.StatusCode == http.StatusForbidden &&
			strings.Contains(strings.ToLower(ghErr.Message), "rate limit") {
			return true
		}
	}

	return false
}

// IsPermanentClientError reports whether err represents an unretryable 4XX client error
// (e.g. 400 Bad Request, 401 Unauthorized, 404 Not Found, 422 Unprocessable Entity),
// explicitly excluding transient rate-limiting errors (429 / secondary rate limits).
func IsPermanentClientError(err error) bool {
	if IsRateLimitError(err) {
		return false
	}

	return IsClientError(err)
}

// IsClientError reports whether err represents an HTTP 4XX client error returned by the GitHub API.
// Common examples include 400 Bad Request, 401 Unauthorized, 403 Forbidden, and 404 Not Found.
func IsClientError(err error) bool {
	if err == nil {
		return false
	}

	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) && ghErr.Response != nil {
		code := ghErr.Response.StatusCode

		return code >= http.StatusBadRequest && code < http.StatusInternalServerError
	}

	return false
}

// IsServerError reports whether err represents an HTTP 5XX server error returned by the GitHub API.
// Common examples include 500 Internal Server Error, 502 Bad Gateway, 503 Service Unavailable, and 504 Gateway Timeout.
func IsServerError(err error) bool {
	if err == nil {
		return false
	}

	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) && ghErr.Response != nil {
		code := ghErr.Response.StatusCode

		return code >= http.StatusInternalServerError && code <= 599
	}

	return false
}

// NewHTTPErrorResponseForTest constructs a *github.ErrorResponse with the given HTTP status code and message.
// Primarily used for testing error handling and classification across consumers of lib/gh without requiring
// them to directly import github.com/google/go-github.
//
//nolint:exhaustruct // Mock error response for tests only.
func NewHTTPErrorResponseForTest(statusCode int, message string) error {
	return &github.ErrorResponse{
		Response: &http.Response{
			StatusCode: statusCode,
		},
		Message: message,
	}
}
