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
