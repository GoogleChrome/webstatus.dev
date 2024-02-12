package gds

import (
	"context"
	"time"

	"cloud.google.com/go/datastore"
)

const wptRunsKey = "WptRuns"

// WPTRun contains common metadata for a run.
type WPTRun struct {
	RunID          int64     `datastore:"run_id"`
	TimeStart      time.Time `datastore:"time_start"`
	TimeEnd        time.Time `datastore:"time_end"`
	BrowserName    string    `datastore:"browser_name"`
	BrowserVersion string    `datastore:"browser_version"`
	Channel        string    `datastore:"channel"`
	OSName         string    `datastore:"os_name"`
	OSVersion      string    `datastore:"os_version"`
}

// wptRunIDFilter implements Filterable to filter by run_id.
// Compatible kinds:
// - wptRunsKey.
// - wptRunMetricsKey.
// - wptRunMetricsGroupByFeatureKey.
type wptRunIDFilter struct {
	runID int64
}

func (f wptRunIDFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.FilterField("run_id", "=", f.runID)
}

// wptRunMerge implements Mergeable for WPTRun.
type wptRunMerge struct{}

func (m wptRunMerge) Merge(existing *WPTRun, _ *WPTRun) *WPTRun {
	// The below fields cannot be overridden during a merge.
	return &WPTRun{
		RunID:          existing.RunID,
		TimeStart:      existing.TimeStart,
		TimeEnd:        existing.TimeEnd,
		BrowserName:    existing.BrowserName,
		BrowserVersion: existing.BrowserVersion,
		Channel:        existing.Channel,
		OSName:         existing.OSName,
		OSVersion:      existing.OSVersion,
	}
}

// StoreWPTRun stores the metadata for a given run.
func (c *Client) StoreWPTRun(
	ctx context.Context,
	run WPTRun) error {
	entityClient := entityClient[WPTRun]{c}

	return entityClient.upsert(
		ctx,
		wptRunsKey,
		&run,
		wptRunMerge{},
		wptRunIDFilter{runID: run.RunID},
	)
}

// GetWPTRun gets the metadata for a given run.
func (c *Client) GetWPTRun(
	ctx context.Context,
	runID int64) (*WPTRun, error) {
	entityClient := entityClient[WPTRun]{c}

	return entityClient.get(
		ctx,
		wptRunsKey,
		wptRunIDFilter{runID: runID},
	)
}

// nolint: lll
// wptRunsByBrowserFilter implements Filterable to filter by:
// - browser_name (equality)
// - channel (equality)
// - time_start (startAt >= x < endAt)
// - sort by time_start
// https://github.com/web-platform-tests/wpt.fyi/blob/fb5bae7c6d04563864ef1c28a263a0a8d6637c4e/shared/test_run_query.go#L183-L186
//
// Compatible kinds:
// - wptRunsKey.
type wptRunsByBrowserFilter struct {
	startAt time.Time
	endAt   time.Time
	browser string
	channel string
}

func (f wptRunsByBrowserFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.FilterField("browser_name", "=", f.browser).
		FilterField("channel", "=", f.channel).
		FilterField("time_start", ">=", f.startAt).
		FilterField("time_start", "<", f.endAt).
		Order("-time_start")
}

// ListWPTRunsByBrowser gets the metadata for a given run.
func (c *Client) ListWPTRunsByBrowser(
	ctx context.Context,
	browser string,
	channel string,
	startAt time.Time,
	endAt time.Time,
	pageToken *string) ([]*WPTRun, *string, error) {
	entityClient := entityClient[WPTRun]{c}

	return entityClient.list(
		ctx,
		wptRunsKey,
		pageToken,
		wptRunsByBrowserFilter{
			startAt: startAt,
			endAt:   endAt,
			browser: browser,
			channel: channel,
		},
	)
}
