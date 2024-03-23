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
	"net/http"
)

func NewHTTPResultsGetter() *HTTPResultsGetter {
	return &HTTPResultsGetter{
		client: *http.DefaultClient,
	}
}

type HTTPResultsGetter struct {
	client http.Client
}

func (h HTTPResultsGetter) DownloadResults(
	ctx context.Context,
	url string) (ResultsSummaryFile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// No need to decompress it despite it having the .gz suffix.

	var data ResultsSummaryFile
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}
