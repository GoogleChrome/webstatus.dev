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
	"net/url"
	"time"

	"cloud.google.com/go/civil"
	"github.com/GoogleChrome/webstatus.dev/lib/httputils"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

const umaQueryServer = "https://uma-export.appspot.com/webstatus/"

var errGeneratingToken = errors.New("token generation error")

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

func (f HTTPMetricsFetcher) Fetch(ctx context.Context,
	queryName metricdatatypes.UMAExportQuery, date civil.Date) (io.ReadCloser, error) {
	queryURL := f.queryURL(queryName, date)

	token, err := f.tokenGen.Generate(ctx, queryURL)
	if err != nil {
		slog.ErrorContext(ctx, "unable to generate token", "error", err)

		return nil, errors.Join(err, errGeneratingToken)
	}

	fetcher, err := httputils.NewHTTPFetcher(queryURL, f.httpClient)
	if err != nil {
		return nil, err
	}

	body, err := fetcher.Fetch(ctx, httputils.WithHeaders(map[string]string{
		"Authorization": "Bearer " + *token,
	}))
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (f HTTPMetricsFetcher) queryURL(queryName metricdatatypes.UMAExportQuery, date civil.Date) string {
	u := f.baseURL.JoinPath(string(queryName))
	q := u.Query()
	// Format the date into YYYYMMDDD
	// More information in https://go.dev/src/time/format.go
	q.Add("date", date.In(time.UTC).Format("20060102"))
	u.RawQuery = q.Encode()

	return u.String()
}
