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
	"fmt"
	"math/big"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

const dailyChromiumHistogramMetricsTable = "DailyChromiumHistogramMetrics"

// Implements the entityMapper interface for DailyChromiumHistogramMetric and SpannerDailyChromiumHistogramMetric.
type dailyChromiumHistogramMetricSpannerMapper struct{}

func (m dailyChromiumHistogramMetricSpannerMapper) Table() string {
	return dailyChromiumHistogramMetricsTable
}

type dailyChromiumHistogramMetricKey struct {
	ChromiumHistogramEnumValueID string
	Day                          civil.Date
}

func (m dailyChromiumHistogramMetricSpannerMapper) GetKey(
	in spannerDailyChromiumHistogramMetric) dailyChromiumHistogramMetricKey {
	return dailyChromiumHistogramMetricKey{
		ChromiumHistogramEnumValueID: in.ChromiumHistogramEnumValueID,
		Day:                          in.Day,
	}
}

func (m dailyChromiumHistogramMetricSpannerMapper) Merge(
	in spannerDailyChromiumHistogramMetric,
	existing spannerDailyChromiumHistogramMetric) spannerDailyChromiumHistogramMetric {
	return spannerDailyChromiumHistogramMetric{
		ChromiumHistogramEnumValueID: existing.ChromiumHistogramEnumValueID,
		DailyChromiumHistogramMetric: DailyChromiumHistogramMetric{
			Day:  existing.Day,
			Rate: in.Rate,
		},
	}
}

func (m dailyChromiumHistogramMetricSpannerMapper) SelectOne(key dailyChromiumHistogramMetricKey) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ChromiumHistogramEnumValueID, Day, Rate
	FROM %s
	WHERE ChromiumHistogramEnumValueID = @chromiumHistogramEnumValueID AND Day = @day
	LIMIT 1`,
		m.Table()))
	parameters := map[string]interface{}{
		"chromiumHistogramEnumValueID": key.ChromiumHistogramEnumValueID,
		"day":                          key.Day,
	}
	stmt.Params = parameters

	return stmt
}

type DailyChromiumHistogramMetric struct {
	Day  civil.Date `spanner:"Day"`
	Rate big.Rat    `spanner:"Rate"`
}

type spannerDailyChromiumHistogramMetric struct {
	DailyChromiumHistogramMetric
	ChromiumHistogramEnumValueID string `spanner:"ChromiumHistogramEnumValueID"`
}

func (c *Client) UpsertDailyChromiumHistogramMetric(
	ctx context.Context,
	histogramName metricdatatypes.HistogramName,
	bucketID int64,
	metric DailyChromiumHistogramMetric) error {
	// TODO: When we have a generic way to do batch upserts, change this to accept an array of metrics.
	chromiumHistogramEnumID, err := c.GetIDFromChromiumHistogramKey(ctx, string(histogramName))
	if err != nil {
		return err
	}
	chromiumHistogramEnumValueID, err := c.GetIDFromChromiumHistogramEnumValueKey(ctx, *chromiumHistogramEnumID, bucketID)
	if err != nil {
		return err
	}

	return newEntityWriter[dailyChromiumHistogramMetricSpannerMapper](c).upsert(ctx, spannerDailyChromiumHistogramMetric{
		DailyChromiumHistogramMetric: metric,
		ChromiumHistogramEnumValueID: *chromiumHistogramEnumValueID,
	})
}
