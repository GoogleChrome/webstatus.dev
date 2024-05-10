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
	browsers []string,
) (*FeatureResult, error) {
	txn := c.ReadOnlyTransaction()
	defer txn.Close()

	b := GetFeatureQueryBuilder{
		baseQuery:     c.featureSearchQuery,
		wptMetricView: wptMetricView,
		browsers:      browsers,
	}
	stmt := b.Build(filter)

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
