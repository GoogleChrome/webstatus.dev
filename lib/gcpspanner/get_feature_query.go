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
	"maps"

	"cloud.google.com/go/spanner"
)

func NewFeatureKeyFilter(featureKey string) *FeatureIDFilter {
	return &FeatureIDFilter{featureKey: featureKey}
}

// FeatureIDFilter will limit the search to a particular feature ID.
type FeatureIDFilter struct {
	featureKey string
}

func (f FeatureIDFilter) Clause() string {
	return `
wf.FeatureKey = @featureKey
`
}

func (f FeatureIDFilter) Params() map[string]interface{} {
	return map[string]interface{}{
		"featureKey": f.featureKey,
	}
}

// GetFeatureQueryBuilder builds a query to search for one feature.
type GetFeatureQueryBuilder struct {
	baseQuery     FeatureSearchBaseQuery
	wptMetricView WPTMetricView
	browsers      []string
}

func (q GetFeatureQueryBuilder) Build(
	filter Filterable) spanner.Statement {
	filterParams := make(map[string]interface{})

	queryArgs := FeatureSearchQueryArgs{
		MetricView:                  q.wptMetricView,
		Filters:                     nil,
		PageFilters:                 nil,
		Offset:                      0,
		PageSize:                    1,
		Browsers:                    q.browsers,
		SortClause:                  "",
		SortByStableBrowserImpl:     nil,
		SortByExpBrowserImpl:        nil,
		SortByBrowserFeatureSupport: nil,
	}
	queryArgs.Filters = defaultFeatureSearchFilters()
	if filter != nil {
		queryArgs.Filters = append(queryArgs.Filters, filter.Clause())
		maps.Copy(filterParams, filter.Params())
	}

	sql, params := q.baseQuery.Query(queryArgs)
	maps.Copy(filterParams, params)

	stmt := spanner.NewStatement(sql)
	stmt.Params = filterParams

	return stmt
}
