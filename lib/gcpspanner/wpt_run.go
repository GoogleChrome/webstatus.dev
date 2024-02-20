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

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
)

const wptRunsTable = "WPTRuns"

// SpannerWPTRun represents the data stored in spanner.
// To keep the IDs from WPT Runs decoupled from our storage, it has its own ID.
type SpannerWPTRun struct {
	ID string `spanner:"ID"`
	WPTRun
}

// WPTRun contains common metadata for a run.
// Columns come from the ../../infra/storage/spanner.sql file.
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

func (c *Client) UpsertWPTRun(ctx context.Context, run WPTRun) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := txn.ReadRowUsingIndex(
			ctx,
			wptRunsTable,
			"RunsByExternalRunID",
			spanner.Key{run.RunID},
			[]string{
				"ID",
			})
		if err != nil {
			// Received an error other than not found. Return now.
			if spanner.ErrCode(err) != codes.NotFound {
				return err
			}
			m, err := spanner.InsertOrUpdateStruct(wptRunsTable, run)
			if err != nil {
				return err
			}
			err = txn.BufferWrite([]*spanner.Mutation{m})
			if err != nil {
				return err
			}
		}
		// For now, do not overwrite anything for wpt runs.
		return nil

	})
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) GetIDOfWPTRunByRunID(ctx context.Context, runID int64) (*string, error) {
	txn := c.Single()
	defer txn.Close()
	row, err := txn.ReadRowUsingIndex(
		ctx,
		wptRunsTable,
		"RunsByExternalRunID",
		spanner.Key{runID},
		[]string{
			"ID",
		})
	if err != nil {
		return nil, err
	}
	var id string
	err = row.Column(0, &id)
	if err != nil {
		return nil, err
	}

	return &id, nil
}
