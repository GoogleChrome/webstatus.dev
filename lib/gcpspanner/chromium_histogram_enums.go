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
	chromiumHistogramEnumsTable = "ChromiumHistogramEnums"
)

type chromiumHistogramEnumsMapper struct{}

func (m chromiumHistogramEnumsMapper) Table() string {
	return chromiumHistogramEnumsTable
}

func (m chromiumHistogramEnumsMapper) SelectOne(histogramName string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID, HistogramName
	FROM %s
	WHERE HistogramName = @histogramName
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"histogramName": histogramName,
	}
	stmt.Params = parameters

	return stmt
}

func (m chromiumHistogramEnumsMapper) GetKey(in ChromiumHistogramEnum) string {
	return in.HistogramName
}

func (m chromiumHistogramEnumsMapper) Merge(
	_ ChromiumHistogramEnum, existing spannerChromiumHistogramEnum) spannerChromiumHistogramEnum {
	// If the histogram exists, it currently does nothing and keeps the existing as-is.
	return existing
}

type ChromiumHistogramEnum struct {
	HistogramName string `spanner:"HistogramName"`
}

type spannerChromiumHistogramEnum struct {
	ID string `spanner:"ID"`
	ChromiumHistogramEnum
}

func (m chromiumHistogramEnumsMapper) GetID(histogramName string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID
	FROM %s
	WHERE HistogramName = @histogramName
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"histogramName": histogramName,
	}
	stmt.Params = parameters

	return stmt
}

func (c *Client) UpsertChromiumHistogramEnum(ctx context.Context, in ChromiumHistogramEnum) (*string, error) {
	return newEntityWriterWithIDRetrieval[chromiumHistogramEnumsMapper, string](c).upsertAndGetID(ctx, in)
}

func (c *Client) GetIDFromChromiumHistogramKey(
	ctx context.Context, histogramName string) (*string, error) {
	return newEntityWriterWithIDRetrieval[chromiumHistogramEnumsMapper, string](c).
		getIDByKey(ctx, histogramName)
}
