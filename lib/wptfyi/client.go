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

package wptfyi

import (
	"context"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

type HTTPClient struct {
	hostname  string
	pageLimit int
}

func NewHTTPClient(hostname string, pageLimit int) HTTPClient {
	return HTTPClient{
		hostname:  hostname,
		pageLimit: pageLimit,
	}
}

func (w HTTPClient) GetRuns(
	_ context.Context,
	from time.Time,
	_ int,
	browserName string,
	channelName string,
) (shared.TestRuns, error) {
	//nolint:exhaustruct
	// External struct does not need comply with exhaustruct.
	apiOptions := shared.TestRunFilter{
		From:     &from,
		MaxCount: &w.pageLimit,
		Labels:   mapset.NewSetWith(browserName, channelName),
	}
	allRuns := shared.TestRuns{}
	runs, err := shared.FetchRuns(w.hostname, apiOptions)
	if err != nil {
		return nil, err
	}
	allRuns = append(allRuns, runs...)

	return allRuns, nil
}
