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
	"google.golang.org/grpc/codes"
)

const browserReleasesTable = "BrowserReleases"

// SpannerBrowserRelease is a wrapper for the browser release that is actually
// stored in spanner. For now, it is the same. But we keep this structure to be
// consistent to the other database models.
type SpannerBrowserRelease struct {
	BrowserRelease
}

// BrowserRelease contains information regarding a certain browser release.
type BrowserRelease struct {
	BrowserName    string    `spanner:"BrowserName"`
	BrowserVersion string    `spanner:"BrowserVersion"`
	ReleaseDate    time.Time `spanner:"ReleaseDate"`
}

// InsertBrowserRelease will insert the given browser release.
// If the release, does not exist, it will insert a new release.
// If the release exists, it currently does nothing and keeps the existing as-is.
// nolint: dupl // TODO. Will refactor for common patterns.
func (c *Client) InsertBrowserRelease(ctx context.Context, release BrowserRelease) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := txn.ReadRow(
			ctx,
			browserReleasesTable,
			spanner.Key{release.BrowserName, release.BrowserVersion},
			[]string{
				"ReleaseDate",
			})
		if err != nil {
			// Received an error other than not found. Return now.
			if spanner.ErrCode(err) != codes.NotFound {
				return errors.Join(ErrInternalQueryFailure, err)
			}
			m, err := spanner.InsertOrUpdateStruct(browserReleasesTable, release)
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
			err = txn.BufferWrite([]*spanner.Mutation{m})
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
		}
		// For now, do not overwrite anything for releases.
		return nil

	})
	if err != nil {
		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}
