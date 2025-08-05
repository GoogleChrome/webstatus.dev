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
	"time"

	"cloud.google.com/go/spanner"
)

const wptRunsTable = "WPTRuns"
const indexRunsByExternalRunID = "RunsByExternalRunID"

// spannerWPTRun is a wrapper for the run data that is actually
// stored in spanner. This is useful because the spanner id is not useful to
// return to the end user since it is only used to decouple the primary keys
// between this system and wpt.fyi.
type spannerWPTRun struct {
	ID string `spanner:"ID"`
	WPTRun
}

// WPTRun contains common metadata for a run.
// Columns come from the ../../infra/storage/spanner/migrations/*.sql files.
type WPTRun struct {
	RunID            int64     `spanner:"ExternalRunID"`
	TimeStart        time.Time `spanner:"TimeStart"`
	TimeEnd          time.Time `spanner:"TimeEnd"`
	BrowserName      string    `spanner:"BrowserName"`
	BrowserVersion   string    `spanner:"BrowserVersion"`
	Channel          string    `spanner:"Channel"`
	OSName           string    `spanner:"OSName"`
	OSVersion        string    `spanner:"OSVersion"`
	FullRevisionHash string    `spanner:"FullRevisionHash"`
}

type wptRunSpannerMapper struct{}

func (m wptRunSpannerMapper) SelectOne(externalRunID int64) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID,
		ExternalRunID,
		TimeStart,
		TimeEnd,
		BrowserName,
		BrowserVersion,
		Channel,
		OSName,
		OSVersion,
		FullRevisionHash
	FROM %s
	WHERE ExternalRunID = @externalRunID
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"externalRunID": externalRunID,
	}
	stmt.Params = parameters

	return stmt
}

func (m wptRunSpannerMapper) GetKeyFromExternal(in WPTRun) int64 {
	return in.RunID
}

func (m wptRunSpannerMapper) Table() string {
	return wptRunsTable
}

func (m wptRunSpannerMapper) Merge(_ WPTRun, existing spannerWPTRun) spannerWPTRun {
	// For now, only keep the existing.
	return existing
}

// WPTRunDataForMetrics contains duplicate data from WPTRuns that will be stored
// in the individual metrics. It will allow for quicker look up of metrics.
type WPTRunDataForMetrics struct {
	ID          string    `spanner:"ID"`
	BrowserName string    `spanner:"BrowserName"`
	Channel     string    `spanner:"Channel"`
	TimeStart   time.Time `spanner:"TimeStart"`
}

// InsertWPTRun will insert the given WPT Run.
func (c *Client) InsertWPTRun(ctx context.Context, run WPTRun) error {
	return newEntityWriter[wptRunSpannerMapper](c).upsert(ctx, run)
}

// GetWPTRunDataByRunIDForMetrics is a helper function to help get a subsection of the WPT Run information. This
// information will be used to create the WPT Run metrics.
func (c *Client) GetWPTRunDataByRunIDForMetrics(ctx context.Context, runID int64) (*WPTRunDataForMetrics, error) {
	row, err := newEntityReader[wptRunSpannerMapper, spannerWPTRun, int64](c).readRowByKey(ctx, runID)
	if err != nil {
		return nil, err
	}

	return &WPTRunDataForMetrics{
		ID:          row.ID,
		BrowserName: row.BrowserName,
		Channel:     row.Channel,
		TimeStart:   row.TimeStart,
	}, nil
}
