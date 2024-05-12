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
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"slices"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"google.golang.org/api/iterator"
)

// SpannerFeatureResult is a wrapper for the feature result that is actually
// stored in spanner. This is useful because the spanner id is not useful to
// return to the end user.
type SpannerFeatureResult struct {
	ID                     string                        `spanner:"ID"`
	FeatureKey             string                        `spanner:"FeatureKey"`
	Name                   string                        `spanner:"Name"`
	Status                 *string                       `spanner:"Status"`
	StableMetrics          []*SpannerFeatureResultMetric `spanner:"StableMetrics"`
	ExperimentalMetrics    []*SpannerFeatureResultMetric `spanner:"ExperimentalMetrics"`
	ImplementationStatuses []*ImplementationStatus       `spanner:"ImplementationStatuses"`
	LowDate                *time.Time                    `spanner:"LowDate"`
	HighDate               *time.Time                    `spanner:"HighDate"`
	SpecLinks              []string                      `spanner:"SpecLinks"`
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
	ImplementationDate   *time.Time                  `spanner:"ImplementationDate"`
}

// FeatureResultMetric contains metric information for a feature result query.
// Very similar to WPTRunFeatureMetric.
type FeatureResultMetric struct {
	BrowserName       string                 `spanner:"BrowserName"`
	PassRate          *big.Rat               `spanner:"PassRate"`
	FeatureRunDetails map[string]interface{} `spanner:"-"`
}

type SpannerFeatureResultMetric struct {
	BrowserName       string           `spanner:"BrowserName"`
	PassRate          *big.Rat         `spanner:"PassRate"`
	FeatureRunDetails spanner.NullJSON `spanner:"FeatureRunDetails"`
}

// FeatureResult contains information regarding a particular feature.
type FeatureResult struct {
	FeatureKey             string                  `spanner:"FeatureKey"`
	Name                   string                  `spanner:"Name"`
	Status                 *string                 `spanner:"Status"`
	StableMetrics          []*FeatureResultMetric  `spanner:"StableMetrics"`
	ExperimentalMetrics    []*FeatureResultMetric  `spanner:"ExperimentalMetrics"`
	ImplementationStatuses []*ImplementationStatus `spanner:"ImplementationStatuses"`
	LowDate                *time.Time              `spanner:"LowDate"`
	HighDate               *time.Time              `spanner:"HighDate"`
	SpecLinks              []string                `spanner:"SpecLinks"`
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
	browsers []string,
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

	queryBuilder := FeatureSearchQueryBuilder{
		baseQuery:     c.featureSearchQuery,
		offsetCursor:  offsetCursor,
		wptMetricView: wptMetricView,
		browsers:      browsers,
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
	filter *FeatureSearchCompiledFilter,
	sortOrder Sortable,
	pageSize int,
	txn *spanner.ReadOnlyTransaction) ([]FeatureResult, error) {
	stmt := queryBuilder.Build(filter, sortOrder, pageSize)

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

		stableMetrics := convertSpannerMetrics(result.StableMetrics)
		experimentalMetrics := convertSpannerMetrics(result.ExperimentalMetrics)

		result.ImplementationStatuses = slices.DeleteFunc[[]*ImplementationStatus](
			result.ImplementationStatuses, findImplementationStatusDefaultPlaceHolder)
		if len(result.ImplementationStatuses) == 0 {
			// If we removed everything, just set it to nil
			result.ImplementationStatuses = nil
		}

		if len(result.SpecLinks) == 0 {
			result.SpecLinks = nil
		}

		actualResult := FeatureResult{
			FeatureKey:             result.FeatureKey,
			Name:                   result.Name,
			Status:                 result.Status,
			StableMetrics:          stableMetrics,
			ExperimentalMetrics:    experimentalMetrics,
			ImplementationStatuses: result.ImplementationStatuses,
			LowDate:                result.LowDate,
			HighDate:               result.HighDate,
			SpecLinks:              result.SpecLinks,
		}
		results = append(results, actualResult)
	}

	return results, nil
}

// convertSpannerMetrics converts a slice of SpannerFeatureResultMetric to FeatureResultMetric.
// TODO: Pass in context to be used by slog.ErrorContext.
func convertSpannerMetrics(spannerMetrics []*SpannerFeatureResultMetric) []*FeatureResultMetric {
	featureResults := make([]*FeatureResultMetric, 0, len(spannerMetrics))
	for _, metric := range spannerMetrics {
		if findDefaultPlaceHolder(metric) {
			continue
		}
		featureResultMetric := FeatureResultMetric{
			BrowserName:       metric.BrowserName,
			PassRate:          metric.PassRate,
			FeatureRunDetails: nil,
		}
		if metric.FeatureRunDetails.Valid {
			var detailsMap map[string]interface{}
			if err := json.Unmarshal([]byte(metric.FeatureRunDetails.String()), &detailsMap); err != nil {
				slog.Error("Error unmarshalling FeatureRunDetails", "error", err)
			} else {
				featureResultMetric.FeatureRunDetails = detailsMap
			}
		}
		featureResults = append(featureResults, &featureResultMetric)
	}

	return featureResults
}

// nolint: gochecknoglobals // needed for findDefaultPlaceHolder.
var zeroPassRatePlaceholder = big.NewRat(0, 1)

// The base query has a solution that works on both GCP Spanner and Emulator that if it finds
// a null array, put a placeholder in there. This function exists to find it and remove it before returning.
func findDefaultPlaceHolder(in *SpannerFeatureResultMetric) bool {
	if in == nil {
		return false
	}

	return in.BrowserName == "" ||
		(in.PassRate == nil || (in.PassRate != nil && in.PassRate.Cmp(zeroPassRatePlaceholder) == 0)) &&
			!in.FeatureRunDetails.Valid
}

// The base query has a solution that works on both GCP Spanner and Emulator that if it finds
// a null array, put a placeholder in there. This function exists to find it and remove it before returning.
func findImplementationStatusDefaultPlaceHolder(in *ImplementationStatus) bool {
	if in == nil {
		return false
	}

	return in.BrowserName == "" && in.ImplementationStatus == Unavailable && in.ImplementationDate == nil
}
