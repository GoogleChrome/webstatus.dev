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
	"slices"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func getSampleBrowserReleases() []BrowserRelease {
	// nolint: dupl // Okay to duplicate for tests
	return []BrowserRelease{
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "0.0.0",
			ReleaseDate:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "0.0.0",
			ReleaseDate:    time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "1.0.0",
			ReleaseDate:    time.Date(2000, time.February, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "1.0.0",
			ReleaseDate:    time.Date(2000, time.February, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "2.0.0",
			ReleaseDate:    time.Date(2000, time.March, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "2.0.0",
			ReleaseDate:    time.Date(2000, time.March, 2, 0, 0, 0, 0, time.UTC),
		},
	}
}

// Helper method to get all the browser releases in a stable order.
// nolint: lll
func (c *Client) ReadAllBrowserReleases(ctx context.Context, _ *testing.T) ([]BrowserRelease, error) {
	stmt := spanner.NewStatement("SELECT BrowserName, BrowserVersion, ReleaseDate FROM BrowserReleases ORDER BY ReleaseDate ASC")
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []BrowserRelease
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var release SpannerBrowserRelease
		if err := row.ToStruct(&release); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		ret = append(ret, release.BrowserRelease)
	}

	return ret, nil
}

func TestInsertBrowserRelease(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	sampleBrowserReleases := getSampleBrowserReleases()
	for _, release := range sampleBrowserReleases {
		err := spannerClient.InsertBrowserRelease(ctx, release)
		if err != nil {
			t.Errorf("unexpected error during insert. %s", err.Error())
		}
	}

	releases, err := spannerClient.ReadAllBrowserReleases(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	if !slices.Equal[[]BrowserRelease](sampleBrowserReleases, releases) {
		t.Errorf("unequal releases. expected %+v actual %+v", sampleBrowserReleases, releases)
	}
}
