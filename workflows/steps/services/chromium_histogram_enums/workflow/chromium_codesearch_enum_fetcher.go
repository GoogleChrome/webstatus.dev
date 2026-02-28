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
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/httputils"
)

func NewChromiumCodesearchEnumFetcher(httpClient *http.Client) (*ChromiumCodesearchEnumFetcher, error) {
	fetcher, err := httputils.NewHTTPFetcher(EnumURL, httpClient)
	if err != nil {
		return nil, err
	}

	return &ChromiumCodesearchEnumFetcher{
		HTTPFetcher: fetcher,
	}, nil
}

// ChromiumCodesearchEnumFetcher fetches the enums from Chromium code search.
// The returned data will be base64 encoded and it is up the consumer to decode
// before reading.
type ChromiumCodesearchEnumFetcher struct {
	*httputils.HTTPFetcher
}

const EnumURL = "https://chromium.googlesource.com/chromium/src/+/main/tools/metrics/histograms/metadata/blink/" +
	"enums.xml?format=TEXT"
