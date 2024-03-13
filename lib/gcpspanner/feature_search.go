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

	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"google.golang.org/api/iterator"
)

// SpannerFeatureResult is a wrapper for the feature result that is actually
// stored in spanner. This is useful because the spanner id is not useful to
// return to the end user.
type SpannerFeatureResult struct {
	ID                  string           `spanner:"ID"`
	FeatureID           string           `spanner:"FeatureID"`
	Name                string           `spanner:"Name"`
	Status              string           `spanner:"Status"`
	StableMetrics       spanner.NullJSON `spanner:"StableMetrics"`
	ExperimentalMetrics spanner.NullJSON `spanner:"ExperimentalMetrics"`
}

// FeatureResultMetric contains metric information for a feature result query.
// Very similar to WPTRunFeatureMetric.
type FeatureResultMetric struct {
	BrowserName string `json:"BrowserName"`
	TotalTests  *int64 `json:"TotalTests"`
	TestPass    *int64 `json:"TestPass"`
}

// FeatureResult contains information regarding a particular feature.
type FeatureResult struct {
	FeatureID           string                 `spanner:"FeatureID"`
	Name                string                 `spanner:"Name"`
	Status              string                 `spanner:"Status"`
	StableMetrics       []*FeatureResultMetric `spanner:"StableMetrics"`
	ExperimentalMetrics []*FeatureResultMetric `spanner:"ExperimentalMetrics"`
}

func (c *Client) FeaturesSearch(
	ctx context.Context,
	pageToken *string,
	pageSize int,
	searchNode *searchtypes.SearchNode,
	sortOrder Sortable,
) ([]FeatureResult, *string, error) {
	// Build filterable
	filterBuilder := NewFeatureSearchFilterBuilder()
	filter := filterBuilder.Build(searchNode)

	var cursor *FeatureResultCursor
	var err error
	if pageToken != nil {
		cursor, err = decodeFeatureResultCursor(*pageToken)
		if err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
	}
	queryBuilder := FeatureSearchQueryBuilder{
		baseQuery: FeatureBaseQuery{},
		cursor:    cursor,
		pageSize:  pageSize,
	}
	stmt := queryBuilder.Build(filter, sortOrder)
	txn := c.Single()
	defer txn.Close()
	it := txn.Query(ctx, stmt)
	defer it.Stop()

	var results []FeatureResult
	for {
		row, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var result SpannerFeatureResult
		if err := row.ToStruct(&result); err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var stableMetrics []*FeatureResultMetric
		if err := json.Unmarshal([]byte(result.StableMetrics.String()), &stableMetrics); err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var experimentalMetrics []*FeatureResultMetric
		if err := json.Unmarshal([]byte(result.ExperimentalMetrics.String()), &experimentalMetrics); err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		actualResult := FeatureResult{
			FeatureID:           result.FeatureID,
			Name:                result.Name,
			Status:              result.Status,
			StableMetrics:       stableMetrics,
			ExperimentalMetrics: experimentalMetrics,
		}
		results = append(results, actualResult)
	}

	if len(results) == pageSize {
		lastResult := results[len(results)-1]
		newCursor := encodeFeatureResultCursor(lastResult.FeatureID)

		return results, &newCursor, nil
	}

	return results, nil, nil
}
