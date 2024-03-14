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
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

const wptRunsTable = "WPTRuns"
const indexRunsByExternalRunID = "RunsByExternalRunID"

// SpannerWPTRun is a wrapper for the run data that is actually
// stored in spanner. This is useful because the spanner id is not useful to
// return to the end user since it is only used to decouple the primary keys
// between this system and wpt.fyi.
type SpannerWPTRun struct {
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

// WPTRunDataForMetrics contains duplicate data from WPTRuns that will be stored
// in the individual metrics. It will allow for quicker look up of metrics.
type WPTRunDataForMetrics struct {
	ID          string    `spanner:"ID"`
	BrowserName string    `spanner:"BrowserName"`
	Channel     string    `spanner:"Channel"`
	TimeStart   time.Time `spanner:"TimeStart"`
}

// InsertWPTRun will insert the given WPT Run.
// If the run, does not exist, it will insert a new run.
// If the run exists, it currently does nothing and keeps the existing as-is.
// The update case should be revisited later on.
// It uses the RunsByExternalRunID index to quickly look up the row.
func (c *Client) InsertWPTRun(ctx context.Context, run WPTRun) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := txn.ReadRowUsingIndex(
			ctx,
			wptRunsTable,
			indexRunsByExternalRunID,
			spanner.Key{run.RunID},
			[]string{
				"ID",
			})
		if err != nil {
			// Received an error other than not found. Return now.
			if spanner.ErrCode(err) != codes.NotFound {
				return errors.Join(ErrInternalQueryFailure, err)
			}
			m, err := spanner.InsertOrUpdateStruct(wptRunsTable, run)
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
			err = txn.BufferWrite([]*spanner.Mutation{m})
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
		}
		// For now, do not overwrite anything for wpt runs.
		// If this is changed in the future, do not allow changes to the data in
		// WPTRunDataForMetrics because it is used in the metrics table.
		return nil

	})
	if err != nil {
		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}

// GetIDOfWPTRunByRunID is a helper function to help get the spanner ID of the
// run. This ID then can be used to create WPT Run Metrics. By linking with this
// ID, we do not have to be coupled with the ID from wpt.fyi.
// It uses the RunsByExternalRunID index to quickly look up the row.
func (c *Client) GetIDOfWPTRunByRunID(ctx context.Context, runID int64) (*string, error) {
	txn := c.Single()
	defer txn.Close()
	row, err := txn.ReadRowUsingIndex(
		ctx,
		wptRunsTable,
		indexRunsByExternalRunID,
		spanner.Key{runID},
		[]string{
			"ID",
		})
	if err != nil {
		// For now, do not check for the "does not exist" error. Treat it as ErrInternalQueryFailure for now.
		// Can revisit whether or not separate that error from the rest of the errors in the future, if needed.

		return nil, errors.Join(ErrInternalQueryFailure, err)
	}
	var id string
	err = row.Column(0, &id)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	return &id, nil
}

// GetIDOfWPTRunByRunID is a helper function to help get the spanner ID of the
// run. This ID then can be used to create WPT Run Metrics. By linking with this
// ID, we do not have to be coupled with the ID from wpt.fyi.
// It uses the RunsByExternalRunID index to quickly look up the row.
func (c *Client) GetWPTRunDataByRunIDForMetrics(ctx context.Context, runID int64) (*WPTRunDataForMetrics, error) {
	query := `
	SELECT
		ID, BrowserName, Channel, TimeStart
	FROM WPTRuns
		WHERE ExternalRunID = @id
	LIMIT 1
	`
	stmt := spanner.NewStatement(query)

	stmt.Params = map[string]interface{}{
		"id": runID,
	}

	// Attempt to query for the row.
	txn := c.Single()
	defer txn.Close()
	it := txn.Query(ctx, stmt)
	defer it.Stop()
	row, err := it.Next()
	if err != nil {
		// No row found
		if errors.Is(err, iterator.Done) {
			return nil, errors.Join(ErrQueryReturnedNoResults, err)
		}

		// Catch-all for other errors.
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}
	var data WPTRunDataForMetrics
	err = row.ToStruct(&data)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	return &data, nil
}
