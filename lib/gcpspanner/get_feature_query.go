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

func NewFeatureIDFilter(featureID string) *FeatureIDFilter {
	return &FeatureIDFilter{featureID: featureID}
}

// FeatureIDFilter will limit the search to a particular feature ID.
type FeatureIDFilter struct {
	featureID string
}

func (f FeatureIDFilter) Clause() string {
	return `
wf.FeatureID = @featureID
`
}

func (f FeatureIDFilter) Params() map[string]interface{} {
	return map[string]interface{}{
		"featureID": f.featureID,
	}
}

// GetFeatureQueryBuilder builds a query to search for one feature.
type GetFeatureQueryBuilder struct {
	baseQuery FeatureSearchBaseQuery
}

func (q GetFeatureQueryBuilder) Build(
	prefilter FeatureSearchPrefilterResult,
	filter Filterable) spanner.Statement {
	filterParams := make(map[string]interface{})

	if filter != nil {
		maps.Copy(filterParams, filter.Params())
	}

	sql, params := q.baseQuery.Query(prefilter)
	maps.Copy(filterParams, params)

	stmt := spanner.NewStatement(
		sql + " WHERE " + filter.Clause() + " LIMIT 1")
	stmt.Params = filterParams

	return stmt
}
