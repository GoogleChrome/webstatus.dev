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

const (
	chromiumHistogramEnumValuesTable = "ChromiumHistogramEnumValues"
)

type chromiumHistogramEnumValuesMapper struct{}

func (m chromiumHistogramEnumValuesMapper) Table() string {
	return chromiumHistogramEnumValuesTable
}

func (m chromiumHistogramEnumValuesMapper) SelectOne(key spannerChromiumHistogramEnumValueKey) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ChromiumHistogramEnumID, BucketID, Label
	FROM %s
	WHERE ChromiumHistogramEnumID = @chromiumHistogramEnumID AND BucketID = @bucketID
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"chromiumHistogramEnumID": key.ChromiumHistogramEnumID,
		"bucketID":                key.BucketID,
	}
	stmt.Params = parameters

	return stmt
}

func (m chromiumHistogramEnumValuesMapper) GetKey(in ChromiumHistogramEnumValue) spannerChromiumHistogramEnumValueKey {
	return spannerChromiumHistogramEnumValueKey{
		ChromiumHistogramEnumID: in.ChromiumHistogramEnumID,
		BucketID:                in.BucketID,
	}
}

func (m chromiumHistogramEnumValuesMapper) Merge(
	_ ChromiumHistogramEnumValue, existing spannerChromiumHistogramEnumValue) spannerChromiumHistogramEnumValue {
	// If the histogram exists, it currently does nothing and keeps the existing as-is.
	return existing
}

type ChromiumHistogramEnumValue struct {
	ChromiumHistogramEnumID string `spanner:"ChromiumHistogramEnumID"`
	BucketID                int64  `spanner:"BucketID"`
	Label                   string `spanner:"Label"`
}

type spannerChromiumHistogramEnumValue struct {
	ChromiumHistogramEnumValue
}

type spannerChromiumHistogramEnumValueKey struct {
	ChromiumHistogramEnumID string
	BucketID                int64
}

func (c *Client) UpsertChromiumHistogramEnumValue(ctx context.Context, in ChromiumHistogramEnumValue) error {
	return newEntityWriter[chromiumHistogramEnumValuesMapper](c).upsert(ctx, in)
}
