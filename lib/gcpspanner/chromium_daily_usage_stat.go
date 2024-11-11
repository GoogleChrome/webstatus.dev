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

package gcpspanner

import (
	"context"
	"time"
)

// ChromiumDailyUsageStatsWithTime contains usage stats for a feature at a given time.
type ChromiumDailyUsageStatWithTime struct {
	Date  time.Time `spanner:"Date"`
	Usage *int64    `spanner:"Usage"`
}

// ListMetricsForFeatureIDBrowserAndChannel attempts to return a page of
// metrics based on a web feature key, browser name and channel. A time window
// must be specified to analyze the runs according to the TimeStart of the run.
// If the page size matches the pageSize, a page token is returned. Else,
// no page token is returned.
func (c *Client) ListChromiumDailyUsageStatsForFeatureID(
	ctx context.Context,
	featureKey string,
	startAt time.Time,
	endAt time.Time,
	pageSize int,
	pageToken *string,
) ([]ChromiumDailyUsageStatWithTime, *string, error) {
	var chromiumUsageStats []ChromiumDailyUsageStatWithTime
	chromiumUsageStats = append(chromiumUsageStats, ChromiumDailyUsageStatWithTime{
		Date:  time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		Usage: nil,
	})

	return chromiumUsageStats, nil, nil
}
