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
	"math/big"
	"time"
)

// ChromiumDailyUsageStatsWithTime contains usage stats for a feature at a given time.
type ChromiumDailyUsageStatWithTime struct {
	Date  time.Time `spanner:"Date"`
	Usage *big.Rat  `spanner:"Usage"`
}

// nolint: revive // method currently returns fake data.
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
