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

// FeatureCursorLastSortValueType defines the valid types for the 'LastSortValue' field in a FeatureResultCursor.
// As more are added, also add to FeatureResultCursorLastValue.
type FeatureCursorLastSortValueType interface {
	string // Currently only supports 'string'
}

// FeatureResultOffsetCursor: A numerical offset from the start of the result set. Enables the construction of
// human-friendly URLs specifying an exact page offset.
type FeatureResultOffsetCursor struct {
	Offset int `json:"offset"`
}

// RawFeatureResultCursor: Represents a point for resuming queries based on the last feature ID to enable efficient
// pagination within Spanner.
// RawFeatureResultCursor is a generic representation of a feature-based cursor, used primarily for encoding and
// initial decoding to preserve exact value types for 'LastSortValue'.
type RawFeatureResultCursor[T FeatureCursorLastSortValueType] struct {
	LastFeatureID     string `json:"last_feature_id"`
	SortColumn        string `json:"sort_column"`
	SortOrderOperator string `json:"sort_order_operator"`
	LastSortValue     T      `json:"last_sort_value"`
}

// FeatureResultCursor provides a non-generic representation of a feature-based cursor, simplifying its use in
// subsequent query building logic.
type FeatureResultCursor struct {
	LastFeatureID     string
	SortColumn        string
	SortOrderOperator string
	FeatureResultCursorLastValue
}

func (c FeatureResultCursor) addLastSortValueParam(params map[string]interface{}, paramName string) {
	switch FeatureSearchColumn(c.SortColumn) {
	case featureSearchFeatureIDColumn, featureSearchFeatureNameColumn, featureSearchStatusColumn:
		if c.FeatureResultCursorLastValue.StringValue != nil {
			params[paramName] = *c.FeatureResultCursorLastValue.StringValue
		}
	case featureSearchNone:
		return
	}
}

// getLastSortColumn checks against the allowed list of values of the last sort columns and returns it.
// If spanner ever allows parameterization of the actual column names in queries, we should use that.
// In the meantime, we need to sanitize the input and make sure we only allow columns that we explicitly support.
func (c FeatureResultCursor) getLastSortColumn() FeatureSearchColumn {
	in := FeatureSearchColumn(c.SortColumn)
	switch in {
	case featureSearchFeatureIDColumn,
		featureSearchFeatureNameColumn,
		featureSearchStatusColumn:
		return in
	case featureSearchNone:
		return featureSearchNone
	}

	return featureSearchNone
}

// FeatureResultCursorLastValue holds the various representations of the 'LastSortValue,' allowing flexibility without
// the need for generics in the main 'FeatureResultCursor'.
type FeatureResultCursorLastValue struct {
	StringValue *string
}

// decodeWPTRunCursor provides a wrapper around the generic decodeCursor.
func decodeWPTRunCursor(cursor string) (*WPTRunCursor, error) {
	return decodeCursor[WPTRunCursor](cursor)
}

const (
	sortOrderASCPaginationOperator  = ">"
	sortOrderDESCPaginationOperator = "<"
)

// decodeInputFeatureResultCursor provides a wrapper around the generic decodeCursor.
func decodeInputFeatureResultCursor(
	cursor string, sortOrder Sortable) (*FeatureResultOffsetCursor, *FeatureResultCursor, error) {
	// Try for the offset based cursor
	offsetCursor, err := decodeCursor[FeatureResultOffsetCursor](cursor)
	if err != nil {
		return nil, nil, err
	}
	// If we found something, return early
	if offsetCursor.Offset > 0 {
		return offsetCursor, nil, nil
	}
	switch sortOrder.SortColumn() {
	case featureSearchFeatureIDColumn, featureSearchFeatureNameColumn, featureSearchStatusColumn:
		cursor, err := decodeCursor[RawFeatureResultCursor[string]](cursor)
		if err != nil {
			return nil, nil, err
		}

		// Sanitize the sort order by the only operators we want.
		if cursor.SortOrderOperator != sortOrderASCPaginationOperator &&
			cursor.SortOrderOperator != sortOrderDESCPaginationOperator {
			return nil, nil, ErrInvalidCursorFormat
		}

		return nil, &FeatureResultCursor{
			LastFeatureID:     cursor.LastFeatureID,
			SortColumn:        cursor.SortColumn,
			SortOrderOperator: cursor.SortOrderOperator,
			FeatureResultCursorLastValue: FeatureResultCursorLastValue{
				StringValue: &cursor.LastSortValue,
			},
		}, nil
	case featureSearchNone:
		return nil, nil, ErrInvalidCursorFormat
	}

	return nil, nil, ErrInvalidCursorFormat
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

// encodeFeatureResultCursor encodes a feature-based cursor, selecting the appropriate
// field in 'RawFeatureResultCursor' to use as 'LastSortValue' based on the sortOrder.
func encodeFeatureResultCursor(sortOrder Sortable, lastResult FeatureResult) string {
	var sortOrderOperator string
	if sortOrder.ascendingOrder {
		sortOrderOperator = sortOrderASCPaginationOperator
	} else {
		sortOrderOperator = sortOrderDESCPaginationOperator
	}
	switch sortOrder.SortColumn() {
	case featureSearchFeatureNameColumn:
		return encodeCursor(RawFeatureResultCursor[string]{
			LastFeatureID:     lastResult.FeatureID,
			SortColumn:        string(sortOrder.SortColumn()),
			SortOrderOperator: sortOrderOperator,
			LastSortValue:     lastResult.Name,
		})
	case featureSearchStatusColumn:
		return encodeCursor(RawFeatureResultCursor[string]{
			LastFeatureID:     lastResult.FeatureID,
			SortColumn:        string(sortOrder.SortColumn()),
			SortOrderOperator: sortOrderOperator,
			LastSortValue:     lastResult.Status,
		})
	case featureSearchFeatureIDColumn:
		return encodeCursor(RawFeatureResultCursor[string]{
			LastFeatureID:     lastResult.FeatureID,
			SortColumn:        string(sortOrder.SortColumn()),
			SortOrderOperator: sortOrderOperator,
			LastSortValue:     lastResult.FeatureID,
		})
	case featureSearchNone:
		return ""
	}

	// Should be not reached. Linting should catch all the cases as more are added.
	return ""
}

// BrowserFeatureCountCursor: Represents a point for resuming queries based on the last
// Release Date and cumulative count. Useful for pagination.
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
