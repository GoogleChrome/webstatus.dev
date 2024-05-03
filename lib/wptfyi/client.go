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

// HTTPClient is a client that communicates with the HTTP API for wpt.fyi.
type HTTPClient struct {
	hostname string
}

// NewHTTPClient returns a HTTPClient that is used to communicate with wpt.fyi.
func NewHTTPClient(hostname string) HTTPClient {
	return HTTPClient{
		hostname: hostname,
	}
}

func (w HTTPClient) GetRuns(
	_ context.Context,
	from time.Time,
	pageSize int,
	browserName string,
	channelName string,
) (shared.TestRuns, error) {
	//nolint:exhaustruct
	// External struct does not need comply with exhaustruct.
	apiOptions := shared.TestRunFilter{
		From:     &from,
		MaxCount: &pageSize,
		Labels:   mapset.NewSetWith(browserName, channelName),
	}

	allRuns := shared.TestRuns{}
	var to *time.Time
	var finalIDOfLastPage *int64
	for {
		if to != nil {
			apiOptions.To = to
		}
		runs, err := shared.FetchRuns(w.hostname, apiOptions)
		if err != nil {
			return nil, err
		}

		lastRunOfPage := runs[len(runs)-1]
		// Edge case:
		// We are unable to get a page token back to start the next page. So
		// there is a possibility that as we manually shift the "to" variable,
		// we get the previous page. This can happen if the number of items per
		// page equals "pageSize" exactly on every call. To mitigate, we track
		// the last ID of the last page. If we see it again, we can stop.
		if finalIDOfLastPage != nil && *finalIDOfLastPage == lastRunOfPage.ID {
			break
		}

		allRuns = append(allRuns, runs...)

		if len(runs) < pageSize {
			break
		}

		to = &lastRunOfPage.CreatedAt
		finalIDOfLastPage = &lastRunOfPage.ID
	}

	return allRuns, nil
}
