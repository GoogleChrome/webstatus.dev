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
	"slices"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func (c *Client) GetFeature(
	ctx context.Context,
	filter Filterable,
	wptMetricView WPTMetricView,
) (*FeatureResult, error) {
	txn := c.ReadOnlyTransaction()
	defer txn.Close()
	prefilterResults, err := c.featureSearchQuery.Prefilter(ctx, txn)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}
	b := GetFeatureQueryBuilder{
		baseQuery:     c.featureSearchQuery,
		wptMetricView: wptMetricView,
	}
	stmt := b.Build(prefilterResults, filter)

	it := txn.Query(ctx, stmt)
	defer it.Stop()

	row, err := it.Next()
	if err != nil {
		// No row found
		if errors.Is(err, iterator.Done) {
			return nil, errors.Join(ErrQueryReturnedNoResults, err)
		}

		// Catch-all for other errors.
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}
	var result SpannerFeatureResult
	if err := row.ToStruct(&result); err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
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
		FeatureKey:             result.FeatureKey,
		Name:                   result.Name,
		Status:                 result.Status,
		StableMetrics:          result.StableMetrics,
		ExperimentalMetrics:    result.ExperimentalMetrics,
		ImplementationStatuses: result.ImplementationStatuses,
		LowDate:                result.LowDate,
		HighDate:               result.HighDate,
	}

	return &actualResult, nil
}

func (c *Client) GetIDFromFeatureKey(ctx context.Context, filter *FeatureIDFilter) (*string, error) {
	query := `
	SELECT
		ID
	FROM WebFeatures wf ` +
		"WHERE " + filter.Clause() + `
	LIMIT 1
	`
	stmt := spanner.NewStatement(query)

	stmt.Params = filter.Params()

	// Attempt to query for the row.
	txn := c.Single()
	defer txn.Close()
	it := txn.Query(ctx, stmt)
	defer it.Stop()
	row, err := it.Next()
	if err != nil {
		// No row found
		if errors.Is(err, iterator.Done) {
			return nil, errors.Join(ErrQueryReturnedNoResults, err)
		}

		// Catch-all for other errors.
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}
	var id string
	err = row.Column(0, &id)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	return &id, nil
}
