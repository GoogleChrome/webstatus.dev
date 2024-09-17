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
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

func NewChromiumCodesearchEnumFetcher(httpClient *http.Client) (*ChromiumCodesearchEnumFetcher, error) {
	u, err := url.Parse(enumURL)
	if err != nil {
		return nil, err
	}

	return &ChromiumCodesearchEnumFetcher{
		httpClient: httpClient,
		enumURL:    u,
	}, nil
}

// ChromiumCodesearchEnumFetcher fetches the enums from Chromium code search.
// The returned data will be base64 encoded and it is up the consumer to decode
// before reading.
type ChromiumCodesearchEnumFetcher struct {
	httpClient *http.Client
	enumURL    *url.URL
}

const enumURL = "https://chromium.googlesource.com/chromium/src/+/main/tools/metrics/histograms/enums.xml?format=TEXT"

func (f ChromiumCodesearchEnumFetcher) Fetch(ctx context.Context) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.enumURL.String(), nil)
	if err != nil {
		slog.ErrorContext(ctx, "unable to create request", "error", err)

		return nil, err
	}
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		// Clean up by closing since we will not be returning the body
		resp.Body.Close()

		return nil, err
	}

	return resp.Body, nil
}
