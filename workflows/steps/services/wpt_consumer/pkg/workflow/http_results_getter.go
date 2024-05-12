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
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/web-platform-tests/wpt.fyi/api/query"
)

var (
	// ErrFailedToRequestResults indicates the request for results failed.
	ErrFailedToRequestResults = errors.New("failed to send request for results")

	// ErrResultsDownloadBadStatusCode indicates there was an unexpected status code.
	ErrResultsDownloadBadStatusCode = errors.New("unexpected status code when downloading results")

	// ErrFailedToParseResults indicates the results could not be parsed.
	ErrFailedToParseResults = errors.New("failed to parse results")
)

// NewHTTPResultsGetter returns a new instance of HTTPResultsGetter.
func NewHTTPResultsGetter() *HTTPResultsGetter {
	return &HTTPResultsGetter{
		client: *http.DefaultClient,
	}
}

// HTTPResultsGetter is an implementation of the ResultsGetter interface.
// It contains the logic to retrieve the results for a given WPT Run from the http endpoint.
// This endpoint typically is a publicly accessible url to a GCP storage bucket.
type HTTPResultsGetter struct {
	client http.Client
}

// nolint: ireturn
func (h HTTPResultsGetter) DownloadResults(
	ctx context.Context,
	url string) (ResultsSummaryFile, error) {
	// Build the request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Join(ErrFailedToRequestResults, err)
	}

	// Attempt to download the results.
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, errors.Join(ErrFailedToRequestResults, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.WarnContext(ctx, "download failed: unexpected status code", "status code", resp.StatusCode)

		return nil, ErrResultsDownloadBadStatusCode
	}

	// No need to decompress it despite it having the .gz suffix.

	// Attempt to convert the results file from the raw bytes.
	// For now only attempt to parse v2 files.
	var data ResultsSummaryFileV2
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&data); err != nil {
		return nil, errors.Join(ErrFailedToParseResults, err)
	}

	return data, nil
}

// ResultsSummaryFileV2 is the representation of the v2 summary file.
// It is a copy of the `summary` type from wpt.fyi.
// https://github.com/web-platform-tests/wpt.fyi/blob/05ddddc52a6b95469131eac5e439af39cbd1200a/api/query/query.go#L30
// TODO export Summary in the wpt.fyi project and use it here instead.
type ResultsSummaryFileV2 map[string]query.SummaryResult
