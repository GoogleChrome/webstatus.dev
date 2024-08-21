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
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

const umaQueryServer = "https://uma-export.appspot.com/webstatus/"

var errMissingBody = errors.New("missing response body")
var errUnexpectedStatusCode = errors.New("unexpected status code")

type tokenGenerator interface {
	Generate(ctx context.Context, url string) (*string, error)
}

func NewHTTPMetricsFetcher(tokenGen tokenGenerator) (*HTTPMetricsFetcher, error) {
	baseURL, err := url.Parse(umaQueryServer)
	if err != nil {
		return nil, err
	}

	client := http.DefaultClient
	// Use the same timeout as
	// https://github.com/GoogleChrome/chromium-dashboard/blob/main/internals/fetchmetrics.py
	client.Timeout = 120 * time.Second

	return &HTTPMetricsFetcher{
		baseURL:  baseURL,
		tokenGen: tokenGen,

		httpClient: client,
	}, nil
}

type HTTPMetricsFetcher struct {
	baseURL    *url.URL
	tokenGen   tokenGenerator
	httpClient *http.Client
}

func (f HTTPMetricsFetcher) Fetch(ctx context.Context, queryName UMAExportQuery) (io.ReadCloser, error) {
	queryURL := f.queryURL(queryName)

	token, err := f.tokenGen.Generate(ctx, queryURL)
	if err != nil {
		slog.ErrorContext(ctx, "unable to generate token", "error", err)

		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		slog.ErrorContext(ctx, "unable to create request", "error", err)

		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+*token)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	var dumpStr string
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		slog.Error("unable to dump request", "error", err)
	}
	dumpStr = string(dump)

	slog.InfoContext(ctx, "debug", "respcode", resp.StatusCode, "bodynil?", resp.Body == nil, "request", dumpStr)

	if resp.StatusCode != http.StatusOK {
		// Clean up by closing since we will not be returning the body
		if resp.Body != nil {
			resp.Body.Close()
		}

		return nil, errUnexpectedStatusCode
	}

	if resp.Body == nil {
		return nil, errMissingBody
	}

	return resp.Body, nil
}

func (f HTTPMetricsFetcher) queryURL(queryName UMAExportQuery) string {
	return f.baseURL.JoinPath(string(queryName)).String()
}
