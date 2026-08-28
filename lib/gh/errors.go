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

	"github.com/google/go-github/v79/github"
)

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
