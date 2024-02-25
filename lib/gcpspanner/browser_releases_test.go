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
func (c *Client) ReadAllBrowserReleases(ctx context.Context, t *testing.T) ([]BrowserRelease, error) {
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
	client := getTestDatabase(t)
	ctx := context.Background()
	sampleBrowserReleases := getSampleBrowserReleases()
	for _, release := range sampleBrowserReleases {
		err := client.InsertBrowserRelease(ctx, release)
		if err != nil {
			t.Errorf("unexpected error during insert. %s", err.Error())
		}
	}

	releases, err := client.ReadAllBrowserReleases(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	if !slices.Equal[[]BrowserRelease](sampleBrowserReleases, releases) {
		t.Errorf("unequal releases. expected %+v actual %+v", sampleBrowserReleases, releases)
	}
}
