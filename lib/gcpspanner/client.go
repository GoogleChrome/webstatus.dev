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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/spanner"
)

// ErrQueryReturnedNoResults indicates no results were returned.
var ErrQueryReturnedNoResults = errors.New("query returned no results")

// ErrInternalQueryFailure is a catch-all error for now.
var ErrInternalQueryFailure = errors.New("internal spanner query failure")

// ErrBadClientConfig indicates the the config to setup a Client is invalid.
var ErrBadClientConfig = errors.New("projectID, instanceID and name must not be empty")

// ErrFailedToEstablishClient indicates the spanner client failed to create.
var ErrFailedToEstablishClient = errors.New("failed to establish spanner client")

// ErrInvalidCursorFormat indicates the cursor is not the correct format.
var ErrInvalidCursorFormat = errors.New("invalid cursor format")

// Client is the client for interacting with GCP Spanner.
type Client struct {
	*spanner.Client
	featureSearchQuery FeatureSearchBaseQuery
}

// NewSpannerClient returns a Client for the Google Spanner service.
func NewSpannerClient(projectID string, instanceID string, name string) (*Client, error) {
	if projectID == "" || instanceID == "" || name == "" {
		return nil, ErrBadClientConfig
	}

	client, err := spanner.NewClient(
		context.TODO(),
		fmt.Sprintf(
			"projects/%s/instances/%s/databases/%s",
			projectID, instanceID, name))
	if err != nil {
		return nil, errors.Join(ErrFailedToEstablishClient, err)
	}

	return &Client{
		client,
		GCPFeatureSearchBaseQuery{},
	}, nil
}

func (c *Client) SetFeatureSearchBaseQuery(query FeatureSearchBaseQuery) {
	c.featureSearchQuery = query
}

// WPTRunCursor: Represents a point for resuming queries based on the last
// TimeStart and ExternalRunID. Useful for pagination.
type WPTRunCursor struct {
	LastTimeStart time.Time `json:"last_time_start"`
	LastRunID     int64     `json:"last_run_id"`
}

// FeatureResultOffsetCursor: A numerical offset from the start of the result set. Enables the construction of
// human-friendly URLs specifying an exact page offset.
// Disclaimer: External users should be aware that the format of this token is subject to change and should not be
// treated as a stable interface. Instead, external users should rely on the returned pagination token long term.
type FeatureResultOffsetCursor struct {
	Offset int `json:"offset"`
}

// decodeWPTRunCursor provides a wrapper around the generic decodeCursor.
func decodeWPTRunCursor(cursor string) (*WPTRunCursor, error) {
	return decodeCursor[WPTRunCursor](cursor)
}

// decodeInputFeatureResultCursor provides a wrapper around the generic decodeCursor.
func decodeInputFeatureResultCursor(
	cursor string) (*FeatureResultOffsetCursor, error) {
	// Try for the offset based cursor
	offsetCursor, err := decodeCursor[FeatureResultOffsetCursor](cursor)
	if err != nil {
		return nil, err
	}

	if offsetCursor == nil || offsetCursor.Offset < 0 {
		return nil, ErrInvalidCursorFormat
	}

	return offsetCursor, nil
}

// decodeCursor: Decodes a base64-encoded cursor string into a Cursor struct.
func decodeCursor[T any](cursor string) (*T, error) {
	data, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, errors.Join(ErrInvalidCursorFormat, err)
	}
	var decoded T
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		return nil, errors.Join(ErrInvalidCursorFormat, err)
	}

	return &decoded, nil
}

// BrowserFeatureCountCursor: Represents a point for resuming feature count queries. Designed for efficient pagination
// by storing the following:
//   - LastReleaseDate: The release date of the last result from the previous page, used to continue fetching from the
//     correct point.
//   - LastCumulativeCount: The cumulative count of features up to (and including) the 'LastReleaseDate'.
//     This eliminates the need to recalculate the count for prior pages.
type BrowserFeatureCountCursor struct {
	LastReleaseDate     time.Time `json:"last_release_date"`
	LastCumulativeCount int64     `json:"last_cumulative_count"`
}

// decodeBrowserFeatureCountCursor provides a wrapper around the generic decodeCursor.
func decodeBrowserFeatureCountCursor(cursor string) (*BrowserFeatureCountCursor, error) {
	return decodeCursor[BrowserFeatureCountCursor](cursor)
}

// encodeBrowserFeatureCountCursor provides a wrapper around the generic encodeCursor.
func encodeBrowserFeatureCountCursor(releaseDate time.Time, lastCount int64) string {
	return encodeCursor[BrowserFeatureCountCursor](BrowserFeatureCountCursor{
		LastReleaseDate:     releaseDate,
		LastCumulativeCount: lastCount,
	})
}

// encodeWPTRunCursor provides a wrapper around the generic encodeCursor.
func encodeWPTRunCursor(timeStart time.Time, id int64) string {
	return encodeCursor[WPTRunCursor](WPTRunCursor{LastTimeStart: timeStart, LastRunID: id})
}

// encodeCursor: Encodes a Cursor into a base64-encoded string.
// Returns an empty string if is unable to create a token.
func encodeCursor[T any](in T) string {
	data, err := json.Marshal(in)
	if err != nil {
		slog.Error("unable to encode cursor", "error", err)

		return ""
	}

	return base64.RawURLEncoding.EncodeToString(data)
}

// encodeFeatureResultOffsetCursor provides a wrapper around the generic encodeCursor.
func encodeFeatureResultOffsetCursor(offset int) string {
	return encodeCursor(FeatureResultOffsetCursor{
		Offset: offset,
	})
}
