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
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

func getSampleRuns() []WPTRun {
	// nolint: dupl // Okay to duplicate for tests
	return []WPTRun{
		{
			RunID:            0,
			TimeStart:        time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:      "fooBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.StableLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            1,
			TimeStart:        time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:      "fooBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.ExperimentalLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            2,
			TimeStart:        time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:      "barBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.StableLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            3,
			TimeStart:        time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:      "barBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.ExperimentalLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            6,
			TimeStart:        time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:      "fooBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.StableLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            7,
			TimeStart:        time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:      "fooBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.ExperimentalLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            8,
			TimeStart:        time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:      "barBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.StableLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            9,
			TimeStart:        time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:      "barBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.ExperimentalLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
	}
}

func TestInsertWPTRun(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	for _, run := range getSampleRuns() {
		err := spannerClient.InsertWPTRun(ctx, run)
		if !errors.Is(err, nil) {
			t.Errorf("expected no error upon insert. received %s", err.Error())
		}
		// TODO: test update if we decide to allow updates in the future.
		id, err := spannerClient.GetIDOfWPTRunByRunID(ctx, run.RunID)
		if !errors.Is(err, nil) {
			t.Errorf("expected no error upon insert. received %s", err.Error())
		}
		if id == nil {
			t.Error("expected an id")

			continue
		}
		if len(*id) != 36 {
			// TODO. Assert it is indeed a uuid. For now, check the length.
			t.Errorf("expected auto-generated uuid. id is only length %d", len(*id))
		}
	}
}

// GetIDOfWPTRunByRunID is a test helper function to help get the spanner ID of the
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
