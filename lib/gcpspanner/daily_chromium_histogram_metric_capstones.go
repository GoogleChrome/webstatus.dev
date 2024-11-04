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
	"errors"
	"fmt"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

const dailyChromiumHistogramEnumCapstonesTable = "DailyChromiumHistogramEnumCapstones"

// Implements the entityMapper interface for DailyChromiumHistogramEnumCapstone
// and SpannerDailyChromiumHistogramEnumCapstone.
type dailyChromiumHistogramEnumCapstonesSpannerMapper struct{}

func (m dailyChromiumHistogramEnumCapstonesSpannerMapper) Table() string {
	return dailyChromiumHistogramEnumCapstonesTable
}

type dailyChromiumHistogramEnumCapstoneKey struct {
	ChromiumHistogramEnumID string
	Day                     civil.Date
}

func (m dailyChromiumHistogramEnumCapstonesSpannerMapper) GetKey(
	in spannerDailyChromiumHistogramEnumCapstone) dailyChromiumHistogramEnumCapstoneKey {
	return dailyChromiumHistogramEnumCapstoneKey{
		ChromiumHistogramEnumID: in.ChromiumHistogramEnumID,
		Day:                     in.Day,
	}
}

func (m dailyChromiumHistogramEnumCapstonesSpannerMapper) Merge(
	_ spannerDailyChromiumHistogramEnumCapstone,
	existing spannerDailyChromiumHistogramEnumCapstone) spannerDailyChromiumHistogramEnumCapstone {
	return existing
}

func (m dailyChromiumHistogramEnumCapstonesSpannerMapper) SelectOne(
	key dailyChromiumHistogramEnumCapstoneKey) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ChromiumHistogramEnumID, Day
	FROM %s
	WHERE ChromiumHistogramEnumID = @chromiumHistogramEnumID AND Day = @day
	LIMIT 1`,
		m.Table()))
	parameters := map[string]interface{}{
		"chromiumHistogramEnumID": key.ChromiumHistogramEnumID,
		"day":                     key.Day,
	}
	stmt.Params = parameters

	return stmt
}

type DailyChromiumHistogramEnumCapstone struct {
	Day           civil.Date
	HistogramName metricdatatypes.HistogramName
}

type spannerDailyChromiumHistogramEnumCapstone struct {
	Day                     civil.Date `spanner:"Day"`
	ChromiumHistogramEnumID string     `spanner:"ChromiumHistogramEnumID"`
}

func (c *Client) HasDailyChromiumHistogramCapstone(
	ctx context.Context, in DailyChromiumHistogramEnumCapstone) (*bool, error) {
	chromiumHistogramEnumID, err := c.GetIDFromChromiumHistogramKey(ctx, string(in.HistogramName))
	if err != nil {
		return nil, err
	}

	_, err = newEntityReader[
		dailyChromiumHistogramEnumCapstonesSpannerMapper,
		spannerDailyChromiumHistogramEnumCapstone,
		dailyChromiumHistogramEnumCapstoneKey](c).readRowByKey(ctx, dailyChromiumHistogramEnumCapstoneKey{
		ChromiumHistogramEnumID: *chromiumHistogramEnumID,
		Day:                     in.Day,
	})
	ret := false
	if err != nil {
		if !errors.Is(err, ErrQueryReturnedNoResults) {
			return nil, err
		}
	} else {
		ret = true
	}

	return &ret, nil
}

func (c *Client) UpsertDailyChromiumHistogramCapstone(
	ctx context.Context, in DailyChromiumHistogramEnumCapstone) error {
	chromiumHistogramEnumID, err := c.GetIDFromChromiumHistogramKey(
		ctx, string(in.HistogramName))
	if err != nil {
		return err
	}

	return newEntityWriter[dailyChromiumHistogramEnumCapstonesSpannerMapper](c).upsert(
		ctx, spannerDailyChromiumHistogramEnumCapstone{
			ChromiumHistogramEnumID: *chromiumHistogramEnumID,
			Day:                     in.Day,
		})
}
