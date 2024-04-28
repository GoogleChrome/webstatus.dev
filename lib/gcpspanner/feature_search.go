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
	"math/big"
	"slices"

	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"google.golang.org/api/iterator"
)

// SpannerFeatureResult is a wrapper for the feature result that is actually
// stored in spanner. This is useful because the spanner id is not useful to
// return to the end user.
type SpannerFeatureResult struct {
	ID                     string                  `spanner:"ID"`
	FeatureID              string                  `spanner:"FeatureID"`
	Name                   string                  `spanner:"Name"`
	Status                 string                  `spanner:"Status"`
	StableMetrics          []*FeatureResultMetric  `spanner:"StableMetrics"`
	ExperimentalMetrics    []*FeatureResultMetric  `spanner:"ExperimentalMetrics"`
	ImplementationStatuses []*ImplementationStatus `spanner:"ImplementationStatuses"`
}

// BrowserImplementationStatus is an enumeration of the possible implementation states for a feature in a browser.
type BrowserImplementationStatus string

const (
	Available   BrowserImplementationStatus = "available"
	Unavailable BrowserImplementationStatus = "unavailable"
)

// ImplementationStatus contains the implementation status information for a given browser.
type ImplementationStatus struct {
	BrowserName          string                      `spanner:"BrowserName"`
	ImplementationStatus BrowserImplementationStatus `spanner:"ImplementationStatus"`
}

// FeatureResultMetric contains metric information for a feature result query.
// Very similar to WPTRunFeatureMetric.
type FeatureResultMetric struct {
	BrowserName string   `json:"BrowserName"`
	PassRate    *big.Rat `json:"PassRate"`
}

// FeatureResult contains information regarding a particular feature.
type FeatureResult struct {
	FeatureID              string                  `spanner:"FeatureID"`
	Name                   string                  `spanner:"Name"`
	Status                 string                  `spanner:"Status"`
	StableMetrics          []*FeatureResultMetric  `spanner:"StableMetrics"`
	ExperimentalMetrics    []*FeatureResultMetric  `spanner:"ExperimentalMetrics"`
	ImplementationStatuses []*ImplementationStatus `spanner:"ImplementationStatuses"`
}

// FeatureResultPage contains the details for the feature search request.
type FeatureResultPage struct {
	Total         int64
	NextPageToken *string
	Features      []FeatureResult
}

func (c *Client) FeaturesSearch(
	ctx context.Context,
	pageToken *string,
	pageSize int,
	searchNode *searchtypes.SearchNode,
	sortOrder Sortable,
	wptMetricView WPTMetricView,
) (*FeatureResultPage, error) {
	// Build filterable
	filterBuilder := NewFeatureSearchFilterBuilder()
	filter := filterBuilder.Build(searchNode)

	var offsetCursor *FeatureResultOffsetCursor
	var err error
	if pageToken != nil {
		offsetCursor, err = decodeInputFeatureResultCursor(*pageToken)
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
	}

	txn := c.ReadOnlyTransaction()
	defer txn.Close()
	prefilterResults, err := c.featureSearchQuery.Prefilter(ctx, txn)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	queryBuilder := FeatureSearchQueryBuilder{
		baseQuery:     c.featureSearchQuery,
		offsetCursor:  offsetCursor,
		wptMetricView: wptMetricView,
	}

	// Get the total
	total, err := c.getTotalFeatureCount(ctx, queryBuilder, filter, txn)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	// Get the results
	results, err := c.getFeatureResult(
		ctx,
		queryBuilder,
		prefilterResults,
		filter,
		sortOrder,
		pageSize,
		txn)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	page := FeatureResultPage{
		Features:      results,
		Total:         total,
		NextPageToken: nil,
	}

	if len(results) == pageSize {
		previousOffset := 0
		if offsetCursor != nil {
			previousOffset = offsetCursor.Offset
		}
		token := encodeFeatureResultOffsetCursor(previousOffset + pageSize)
		page.NextPageToken = &token

		return &page, nil
	}

	return &page, nil
}

func (c *Client) getTotalFeatureCount(
	ctx context.Context,
	queryBuilder FeatureSearchQueryBuilder,
	filter *FeatureSearchCompiledFilter,
	txn *spanner.ReadOnlyTransaction) (int64, error) {
	stmt := queryBuilder.CountQueryBuild(filter)

	var count int64
	err := txn.Query(ctx, stmt).Do(func(row *spanner.Row) error {
		if err := row.Column(0, &count); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (c *Client) getFeatureResult(
	ctx context.Context,
	queryBuilder FeatureSearchQueryBuilder,
	prefilterResults FeatureSearchPrefilterResult,
	filter *FeatureSearchCompiledFilter,
	sortOrder Sortable,
	pageSize int,
	txn *spanner.ReadOnlyTransaction) ([]FeatureResult, error) {
	stmt := queryBuilder.Build(prefilterResults, filter, sortOrder, pageSize)

	it := txn.Query(ctx, stmt)
	defer it.Stop()

	var results []FeatureResult
	for {
		row, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var result SpannerFeatureResult
		if err := row.ToStruct(&result); err != nil {
			return nil, err
		}
		result.StableMetrics = slices.DeleteFunc[[]*FeatureResultMetric](
			result.StableMetrics, findDefaultPlaceHolder)
		if len(result.StableMetrics) == 0 {
			// If we removed everything, just set it to nil
			result.StableMetrics = nil
		}

		result.ExperimentalMetrics = slices.DeleteFunc[[]*FeatureResultMetric](
			result.ExperimentalMetrics, findDefaultPlaceHolder)
		if len(result.ExperimentalMetrics) == 0 {
			// If we removed everything, just set it to nil
			result.ExperimentalMetrics = nil
		}

		result.ImplementationStatuses = slices.DeleteFunc[[]*ImplementationStatus](
			result.ImplementationStatuses, findImplementationStatusDefaultPlaceHolder)
		if len(result.ImplementationStatuses) == 0 {
			// If we removed everything, just set it to nil
			result.ImplementationStatuses = nil
		}

		actualResult := FeatureResult{
			FeatureID:              result.FeatureID,
			Name:                   result.Name,
			Status:                 result.Status,
			StableMetrics:          result.StableMetrics,
			ExperimentalMetrics:    result.ExperimentalMetrics,
			ImplementationStatuses: result.ImplementationStatuses,
		}
		results = append(results, actualResult)
	}

	return results, nil
}

// nolint: gochecknoglobals // needed for findDefaultPlaceHolder.
var zeroPassRatePlaceholder = big.NewRat(0, 1)

// The base query has a solution that works on both GCP Spanner and Emulator that if it finds
// a null array, put a placeholder in there. This function exists to find it and remove it before returning.
func findDefaultPlaceHolder(in *FeatureResultMetric) bool {
	if in == nil || in.PassRate == nil {
		return false
	}

	return in.BrowserName == "" && in.PassRate.Cmp(zeroPassRatePlaceholder) == 0
}

// The base query has a solution that works on both GCP Spanner and Emulator that if it finds
// a null array, put a placeholder in there. This function exists to find it and remove it before returning.
func findImplementationStatusDefaultPlaceHolder(in *ImplementationStatus) bool {
	if in == nil {
		return false
	}

	return in.BrowserName == "" && in.ImplementationStatus == Unavailable
}
