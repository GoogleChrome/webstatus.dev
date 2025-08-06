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

const browserReleasesTable = "BrowserReleases"

// spannerBrowserRelease is a wrapper for the browser release that is actually
// stored in spanner. For now, it is the same. But we keep this structure to be
// consistent to the other database models.
type spannerBrowserRelease struct {
	BrowserRelease
}

// BrowserRelease contains information regarding a certain browser release.
type BrowserRelease struct {
	BrowserName    string    `spanner:"BrowserName"`
	BrowserVersion string    `spanner:"BrowserVersion"`
	ReleaseDate    time.Time `spanner:"ReleaseDate"`
}

type browserReleaseKey struct {
	BrowserName    string
	BrowserVersion string
}

// Implements the entityMapper interface for BrowserRelease and SpannerBrowserRelease.
type browserReleaseSpannerMapper struct{}

func (m browserReleaseSpannerMapper) SelectOne(key browserReleaseKey) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		BrowserName, BrowserVersion, ReleaseDate
	FROM %s
	WHERE BrowserName = @browserName AND BrowserVersion = @browserVersion
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"browserName":    key.BrowserName,
		"browserVersion": key.BrowserVersion,
	}
	stmt.Params = parameters

	return stmt
}

func (m browserReleaseSpannerMapper) Merge(_ BrowserRelease, existing spannerBrowserRelease) spannerBrowserRelease {
	// If the release exists, it currently does nothing and keeps the existing as-is.
	return existing
}

func (m browserReleaseSpannerMapper) GetKeyFromExternal(in BrowserRelease) browserReleaseKey {
	return browserReleaseKey{
		BrowserName:    in.BrowserName,
		BrowserVersion: in.BrowserVersion,
	}
}

func (m browserReleaseSpannerMapper) Table() string {
	return browserReleasesTable
}

// InsertBrowserRelease will insert the given browser release.
func (c *Client) InsertBrowserRelease(ctx context.Context, release BrowserRelease) error {
	return newEntityWriter[browserReleaseSpannerMapper](c).upsert(ctx, release)
}

func (c *Client) fetchAllBrowserReleasesWithTransaction(
	ctx context.Context, txn *spanner.ReadOnlyTransaction) ([]spannerBrowserRelease, error) {
	var releases []spannerBrowserRelease
	iter := txn.Read(ctx, browserReleasesTable, spanner.AllKeys(), []string{
		"BrowserName",
		"BrowserVersion",
		"ReleaseDate",
	})
	defer iter.Stop()
	err := iter.Do(func(row *spanner.Row) error {
		var entry spannerBrowserRelease
		if err := row.ToStruct(&entry); err != nil {
			return err
		}
		releases = append(releases, entry)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return releases, nil
}
