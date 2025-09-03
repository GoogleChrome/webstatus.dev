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

package httputils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/GoogleChrome/webstatus.dev/lib/fetchtypes"
)

var (
	ErrUnableToParseURL = errors.New("unable to parse URL")
)

type HTTPFetcher struct {
	httpClient *http.Client
	endpoint   *url.URL
}

// RequestOptions holds options for the HTTP request.
type RequestOptions struct {
	Headers map[string]string
}

// ResponseOptions holds options for handling the HTTP response.
type ResponseOptions struct {
	// A list of status codes that are considered successful. If empty, only http.StatusOK is considered successful.
	ExpectedStatusCodes []int
}

// FetchOptions holds options for an HTTP fetch operation.
type FetchOptions struct {
	Request  RequestOptions
	Response ResponseOptions
}

// FetchOption is a function that modifies the FetchOptions.
type FetchOption func(*FetchOptions)

// WithHeaders sets the headers for the HTTP request.
func WithHeaders(headers map[string]string) FetchOption {
	return func(o *FetchOptions) {
		o.Request.Headers = headers
	}
}

// WithExpectedStatusCodes sets the expected status codes for the HTTP response.
func WithExpectedStatusCodes(codes []int) FetchOption {
	return func(o *FetchOptions) {
		o.Response.ExpectedStatusCodes = codes
	}
}

// NewHTTPFetcher creates a new HTTPFetcher with the given endpoint and HTTP client.
func NewHTTPFetcher(
	endpoint string, httpClient *http.Client) (*HTTPFetcher, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.Join(ErrUnableToParseURL, err)
	}

	return &HTTPFetcher{
		httpClient: httpClient,
		endpoint:   u,
	}, nil
}

// Fetch performs an HTTP GET request to the configured endpoint.
// It returns the response body as an io.ReadCloser on success.
// The caller is responsible for closing the reader.
// It accepts a variable number of FetchOption functions to customize the request and response handling.
func (f HTTPFetcher) Fetch(ctx context.Context, opts ...FetchOption) (io.ReadCloser, error) {
	// Default options
	options := &FetchOptions{
		Response: ResponseOptions{
			ExpectedStatusCodes: []int{http.StatusOK},
		},
		Request: RequestOptions{
			Headers: nil,
		},
	}

	// Apply custom options
	for _, opt := range opts {
		opt(options)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.endpoint.String(), nil)
	if err != nil {
		slog.ErrorContext(ctx, "unable to create request", "error", err, "url", f.endpoint.String())

		return nil, errors.Join(fetchtypes.ErrFailedToBuildRequest, err)
	}

	// Apply request options
	for k, v := range options.Request.Headers {
		req.Header.Add(k, v)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "unable to fetch", "error", err, "url", f.endpoint.String())

		return nil, errors.Join(fetchtypes.ErrFailedToFetch, err)
	}

	// Apply response options
	statusCodeMatch := false
	for _, code := range options.Response.ExpectedStatusCodes {
		if resp.StatusCode == code {
			statusCodeMatch = true

			break
		}
	}

	if !statusCodeMatch {
		slog.ErrorContext(ctx, "bad status code while fetching", "status", resp.StatusCode, "url", f.endpoint.String())
		// Clean up by closing since we will not be returning the body
		resp.Body.Close()

		return nil, errors.Join(fetchtypes.ErrUnexpectedResult, fmt.Errorf("bad status code:%d", resp.StatusCode))
	}

	if resp.Body == nil {
		slog.ErrorContext(ctx, "missing body", "url", f.endpoint.String())

		return nil, fetchtypes.ErrMissingBody
	}

	return resp.Body, nil
}
