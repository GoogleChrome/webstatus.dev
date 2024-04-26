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
	"math/big"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
)

func init() {
	featuresSearchPageCursorFilterOneColTemplate = NewQueryTemplate(featuresSearchPageCursorFilterOneColRawTemplate)
	featuresSearchPageCursorFilterTwoColTemplate = NewQueryTemplate(featuresSearchPageCursorFilterTwoColRawTemplate)
}

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

// nolint: gochecknoglobals // WONTFIX. Caution: Do not override these outside of init()
var (
	// featuresSearchPageCursorFilterOneColTemplate is the compiled version of
	// featuresSearchPageCursorFilterOneColRawTemplate.
	featuresSearchPageCursorFilterOneColTemplate BaseQueryTemplate
	// featuresSearchPageCursorFilterTwoColTemplate is the compiled version of
	// featuresSearchPageCursorFilterTwoColRawTemplate.
	featuresSearchPageCursorFilterTwoColTemplate BaseQueryTemplate
)

// PageFilter templates. For now, we have an explicit template each combination
// of possible sort columns.
// As of now, we can have between 1 or 2 sort columns at a given time.
// TODO: If the number grows, we may want to invest in solving it for N sort columns.
const (
	// featuresSearchPageCursorFilterOneColRawTemplate is the template for resuming features search / get feature
	// queries when using one sort column.
	featuresSearchPageCursorFilterOneColRawTemplate = `
(
	{{ .Column1 }} {{ .Column1Operator }} @{{ .Column1ValueParam }} OR
	({{ .Column1 }} = @{{ .Column1ValueParam }} AND {{ .TieBreakerColumn }} > @{{ .TieBreakerValueParam }})
)
`
	// featuresSearchPageCursorFilterTwoColRawTemplate is the template for resuming features search / get feature
	// queries when using two sort columns.
	featuresSearchPageCursorFilterTwoColRawTemplate = `
(
	{{ .Column1 }} {{ .Column1Operator }} @{{ .Column1ValueParam }} OR
	(
		{{ .Column1 }} = @{{ .Column1ValueParam }} AND
		{{ .Column2 }} {{ .Column2Operator }} @{{ .Column2ValueParam }}
	) OR
	(
		{{ .Column1 }} = @{{ .Column1ValueParam }} AND
		{{ .Column2 }} = @{{ .Column2ValueParam }} AND
		{{ .TieBreakerColumn }} > @{{ .TieBreakerValueParam }}
	)
)
`
)

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
// Disclaimer: External users should be aware that the format of this token is subject to change and should not be
// treated as a stable interface. Instead, external users should rely on the returned pagination token long term.
type FeatureResultOffsetCursor struct {
	Offset int `json:"offset"`
}

// RawFeatureResultCursor: Represents a point for resuming queries based on the last feature ID to enable efficient
// pagination within Spanner.
// RawFeatureResultCursor is a generic representation of a feature-based cursor, used primarily for encoding and
// initial decoding to preserve exact value types for 'LastSortValue'.
type RawFeatureResultCursor struct {
	LastFeatureID string          `json:"last_feature_id"`
	SortTarget    string          `json:"sort_operation"`
	Columns       []LastValueInfo `json:"last_values"`
}

type LastValueInfo struct {
	SortOrderOperator string              `json:"sort_order_operator"`
	LastSortValue     any                 `json:"last_sort_value"`
	Column            FeatureSearchColumn `json:"column"`
}

// FeatureResultCursor provides a non-generic representation of a feature-based cursor, simplifying its use in
// subsequent query building logic.
type FeatureResultCursor struct {
	LastFeatureID string
	SortTarget    FeaturesSearchSortTarget
	Columns       []FeatureResultCursorLastValue
}

func (c FeatureResultCursor) buildPageFilters(existingParams map[string]interface{}) string {
	sortColumns := make([]FeatureSearchSortColumn, 0, len(c.Columns))
	currentParamCount := 0
	for _, lastValue := range c.Columns {
		columnValueParam := fmt.Sprintf("cursorSortColumn%d", currentParamCount)
		existingParams[columnValueParam] = lastValue.Value
		currentParamCount++

		sortColumn := FeatureSearchSortColumn{
			Column:           lastValue.Column.ToFilterColumn(),
			SortOrder:        lastValue.SortOrder,
			ColumnValueParam: columnValueParam,
		}

		sortColumns = append(sortColumns, sortColumn)
		slog.Info("sort stuff", "column", sortColumn, "value", lastValue.Value)
	}
	tieBreakerValueParam := fmt.Sprintf("cursor%d", currentParamCount)
	existingParams[tieBreakerValueParam] = c.LastFeatureID

	switch len(sortColumns) {
	case 1:
		return featuresSearchPageCursorFilterOneColTemplate.Execute(struct {
			Column1              string
			Column1Operator      string
			Column1ValueParam    string
			TieBreakerColumn     string
			TieBreakerValueParam string
		}{
			Column1:              sortColumns[0].Column,
			Column1Operator:      sortColumns[0].SortOrder,
			Column1ValueParam:    sortColumns[0].ColumnValueParam,
			TieBreakerColumn:     string(featureSearchFeatureIDColumn),
			TieBreakerValueParam: tieBreakerValueParam,
		})
	case 2:
		return featuresSearchPageCursorFilterTwoColTemplate.Execute(struct {
			Column1              string
			Column1Operator      string
			Column1ValueParam    string
			Column2              string
			Column2Operator      string
			Column2ValueParam    string
			TieBreakerColumn     string
			TieBreakerValueParam string
		}{
			Column1:              sortColumns[0].Column,
			Column1Operator:      sortColumns[0].SortOrder,
			Column1ValueParam:    sortColumns[0].ColumnValueParam,
			Column2:              sortColumns[1].Column,
			Column2Operator:      sortColumns[1].SortOrder,
			Column2ValueParam:    sortColumns[1].ColumnValueParam,
			TieBreakerColumn:     string(featureSearchFeatureIDColumn),
			TieBreakerValueParam: tieBreakerValueParam,
		})
	}

	// Default
	return ""
}

type FeatureSearchSortColumn struct {
	Column           string
	SortOrder        string
	ColumnValueParam string
}

// FeatureResultCursorLastValue holds the various representations of the 'LastSortValue,' allowing flexibility without
// the need for generics in the main 'FeatureResultCursor'.
type FeatureResultCursorLastValue struct {
	Value     any
	SortOrder string
	Column    FeatureSearchColumn
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
	cursor string) (*FeatureResultOffsetCursor, *FeatureResultCursor, error) {
	// Try for the offset based cursor
	offsetCursor, err := decodeCursor[FeatureResultOffsetCursor](cursor)
	if err != nil {
		return nil, nil, err
	}
	// If we found something, return early
	if offsetCursor.Offset > 0 {
		return offsetCursor, nil, nil
	}

	decodedCursor, err := decodeCursor[RawFeatureResultCursor](cursor)
	if err != nil {
		return nil, nil, err
	}

	// Sanitize the sort order by the only operators we want.
	for _, value := range decodedCursor.Columns {
		if value.SortOrderOperator != sortOrderASCPaginationOperator &&
			value.SortOrderOperator != sortOrderDESCPaginationOperator {
			return nil, nil, ErrInvalidCursorFormat
		}
	}

	sortTarget := FeaturesSearchSortTarget(decodedCursor.SortTarget)
	switch sortTarget {
	case IDSort, StatusSort, NameSort, StableImplSort, ExperimentalImplSort:
		break
	default:
		slog.Warn("unable to use sort target", "target", sortTarget)

		return nil, nil, ErrInvalidCursorFormat
	}

	lastValues := make([]FeatureResultCursorLastValue, 0, len(decodedCursor.Columns))
	for _, lastValueInfo := range decodedCursor.Columns {
		col := FeatureSearchColumn(lastValueInfo.Column)
		switch col {
		case featureSearchFeatureIDColumn,
			featureSearchFeatureNameColumn,
			featureSearchStatusColumn:
			_, ok := lastValueInfo.LastSortValue.(string)
			if !ok {
				// Type check the value
				return nil, nil, ErrInvalidCursorFormat
			}
		case featureSearcBrowserImplColumn:
			// featureSearcBrowserImplColumn is a nullable string
			if lastValueInfo.LastSortValue == nil {
				break
			}
			// If not null, try to set to string
			_, ok := lastValueInfo.LastSortValue.(string)
			if !ok {
				// Type check the value
				return nil, nil, ErrInvalidCursorFormat
			}
		case featureSearcBrowserMetricColumn:
			// If its null, go ahead and break now.
			if lastValueInfo.LastSortValue == nil {
				break
			}
			rat := &big.Rat{}
			strVal, ok := lastValueInfo.LastSortValue.(string)
			if !ok {
				slog.Error("unable to convert", "big rat", lastValueInfo.LastSortValue)
				// Type check the value
				return nil, nil, ErrInvalidCursorFormat
			}
			rat, ok = rat.SetString(strVal)
			if !ok {
				slog.Error("unable to convert2", "big rat", lastValueInfo.LastSortValue)
				// Type check the value
				return nil, nil, ErrInvalidCursorFormat
			}
			lastValueInfo.LastSortValue = rat

		}
		lastValues = append(lastValues, FeatureResultCursorLastValue{
			Value:     lastValueInfo.LastSortValue,
			SortOrder: lastValueInfo.SortOrderOperator,
			Column:    col,
		})
	}

	return nil, &FeatureResultCursor{
		LastFeatureID: decodedCursor.LastFeatureID,
		SortTarget:    sortTarget,
		Columns:       lastValues,
	}, nil
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
	switch sortOrder.SortTarget() {
	case NameSort:
		return encodeCursor(RawFeatureResultCursor{
			LastFeatureID: lastResult.FeatureID,
			SortTarget:    string(NameSort),
			Columns: []LastValueInfo{
				{
					SortOrderOperator: sortOrderOperator,
					LastSortValue:     lastResult.Name,
					Column:            featureSearchFeatureNameColumn,
				},
			},
		})
	case StatusSort:
		return encodeCursor(RawFeatureResultCursor{
			LastFeatureID: lastResult.FeatureID,
			SortTarget:    string(StatusSort),
			Columns: []LastValueInfo{
				{
					SortOrderOperator: sortOrderOperator,
					LastSortValue:     lastResult.Status,
					Column:            featureSearchStatusColumn,
				},
			},
		})
	case IDSort:
		return encodeCursor(RawFeatureResultCursor{
			LastFeatureID: lastResult.FeatureID,
			SortTarget:    string(IDSort),
			Columns: []LastValueInfo{
				{
					SortOrderOperator: sortOrderOperator,
					LastSortValue:     lastResult.FeatureID,
					Column:            featureSearchFeatureIDColumn,
				},
			},
		})
	case StableImplSort:
		lastMetric := findPassRateForBrowser(lastResult.StableMetrics, sortOrder.browserTarget)
		lastImplStatus := findImplStatusForBrowser(lastResult.ImplementationStatuses, sortOrder.browserTarget)

		cols := addImplSortValuesToSlice(lastMetric, lastImplStatus, sortOrderOperator)

		cursorData := RawFeatureResultCursor{
			LastFeatureID: lastResult.FeatureID,
			SortTarget:    string(StableImplSort),
			Columns:       cols,
		}

		return encodeCursor(cursorData)
	case ExperimentalImplSort:
		lastMetric := findPassRateForBrowser(lastResult.ExperimentalMetrics, sortOrder.browserTarget)
		lastImplStatus := findImplStatusForBrowser(lastResult.ImplementationStatuses, sortOrder.browserTarget)

		cols := addImplSortValuesToSlice(lastMetric, lastImplStatus, sortOrderOperator)

		cursorData := RawFeatureResultCursor{
			LastFeatureID: lastResult.FeatureID,
			SortTarget:    string(StableImplSort),
			Columns:       cols,
		}

		return encodeCursor(cursorData)
	}

	// Should be not reached. Linting should catch all the cases as more are added.
	return ""
}

func addImplSortValuesToSlice(lastMetric *big.Rat,
	lastImplStatus BrowserImplementationStatus, sortOrderOperator string) []LastValueInfo {
	ret := []LastValueInfo{}
	// Check for nil metric. If it is nil, do not encode because the database will not
	// find a nil value in the tables to check.
	if lastMetric != nil {
		ret = append(ret, LastValueInfo{
			SortOrderOperator: sortOrderOperator,
			LastSortValue:     lastMetric,
			Column:            featureSearcBrowserMetricColumn,
		})
	}

	// Check for empty impl string. If it is empty, do not encode because the database will not
	// find a empty value in the tables to check.
	var lastImplValue any
	if lastImplStatus != "" {
		lastImplValue = lastImplStatus
	} else {
		lastImplValue = nil
	}
	ret = append(ret, LastValueInfo{
		SortOrderOperator: sortOrderOperator,
		LastSortValue:     lastImplValue,
		Column:            featureSearcBrowserImplColumn,
	})

	return ret
}

func findPassRateForBrowser(metrics []*FeatureResultMetric, browserName *string) *big.Rat {
	var passRate *big.Rat
	if browserName == nil {
		return passRate
	}
	for _, metric := range metrics {
		if strings.EqualFold(metric.BrowserName, *browserName) {
			passRate = metric.PassRate

			continue
		}
	}

	return passRate
}

func findImplStatusForBrowser(statuses []*ImplementationStatus, browserName *string) BrowserImplementationStatus {
	var ret BrowserImplementationStatus
	if browserName == nil {
		return ret
	}
	for _, status := range statuses {
		if strings.EqualFold(status.BrowserName, *browserName) {
			ret = status.ImplementationStatus

			continue
		}
	}

	return ret
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
