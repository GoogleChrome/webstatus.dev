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

	"cloud.google.com/go/spanner"
)

const chromiumHistogramEnumsTable = "ChromiumHistogramEnums"

type chromiumHistogramMapper struct{}

func (m chromiumHistogramMapper) Table() string {
	return chromiumHistogramEnumsTable
}

func (m chromiumHistogramMapper) SelectOne(key spannerChromiumHistogramKey) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID, HistogramName, BucketID, Label
	FROM %s
	WHERE HistogramName = @histogramName AND BucketID = @bucketID
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"histogramName": key.HistogramName,
		"bucketID":      key.BucketID,
	}
	stmt.Params = parameters

	return stmt
}

func (m chromiumHistogramMapper) GetKey(in ChromiumHistogramEnum) spannerChromiumHistogramKey {
	return spannerChromiumHistogramKey{
		HistogramName: in.HistogramName,
		BucketID:      in.BucketID,
	}
}

func (m chromiumHistogramMapper) Merge(
	_ ChromiumHistogramEnum, existing spannerChromiumHistogramEnum) spannerChromiumHistogramEnum {
	// If the histogram exists, it currently does nothing and keeps the existing as-is.
	return existing
}

type ChromiumHistogramEnum struct {
	HistogramName string `spanner:"HistogramName"`
	BucketID      int64  `spanner:"BucketID"`
	Label         string `spanner:"Label"`
}

type spannerChromiumHistogramEnum struct {
	ID string `spanner:"ID"`
	ChromiumHistogramEnum
}

type spannerChromiumHistogramKey struct {
	HistogramName string
	BucketID      int64
}

func (m chromiumHistogramMapper) GetID(key spannerChromiumHistogramKey) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID
	FROM %s
	WHERE HistogramName = @histogramName AND BucketID = @bucketID
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"histogramName": key.HistogramName,
		"bucketID":      key.BucketID,
	}
	stmt.Params = parameters

	return stmt
}

func (c *Client) UpsertChromiumHistogramEnum(ctx context.Context, in ChromiumHistogramEnum) (*string, error) {
	return newEntityWriterWithIDRetrieval[chromiumHistogramMapper, string](c).upsertAndGetID(ctx, in)
}

func (c *Client) GetIDFromChromiumHistogramKey(
	ctx context.Context, histogramName string, bucketID int64) (*string, error) {
	return newEntityWriterWithIDRetrieval[chromiumHistogramMapper, string](c).
		getIDByKey(ctx, spannerChromiumHistogramKey{
			HistogramName: histogramName,
			BucketID:      bucketID,
		})
}
