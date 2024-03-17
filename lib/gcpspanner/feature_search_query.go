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
	"fmt"
	"maps"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
)

type FeatureSearchFilterBuilder struct {
	paramCounter int
	params       map[string]interface{}
}

func NewFeatureSearchFilterBuilder() *FeatureSearchFilterBuilder {
	return &FeatureSearchFilterBuilder{
		paramCounter: 0,
		params:       nil,
	}
}

type FeatureSearchCompiledFilter struct {
	params map[string]interface{}
	clause string
}

func (f FeatureSearchCompiledFilter) Params() map[string]interface{} {
	return f.params
}

func (f FeatureSearchCompiledFilter) Clause() string {
	return f.clause
}

// addParamGetName adds a parameter to the map that will be used in the spanner params map.
// Afterwards, get the name of the parameter. Then increment the counter.
func (b *FeatureSearchFilterBuilder) addParamGetName(param interface{}) string {
	name := fmt.Sprintf("param%d", b.paramCounter)
	b.params[name] = param

	// Increment for the next parameter.
	b.paramCounter++

	return name
}

func (b *FeatureSearchFilterBuilder) Build(node *searchtypes.SearchNode) *FeatureSearchCompiledFilter {
	// Ensure it is not nil
	if node == nil ||
		// Check for our root node.
		node.Operator != searchtypes.OperatorRoot ||
		// Currently root should only have at most one child.
		// lib/gcpspanner/searchtypes/features_search_visitor.go
		len(node.Children) != 1 {
		return nil
	}

	//  Initialize the map and (re)set counter to 0
	b.params = make(map[string]interface{})
	b.paramCounter = 0

	generatedFilters := b.traverseAndGenerateFilters(node.Children[0])
	filterClause := strings.Join(generatedFilters, " AND ")

	return &FeatureSearchCompiledFilter{
		params: b.params,
		clause: filterClause,
	}
}

func (b *FeatureSearchFilterBuilder) traverseAndGenerateFilters(node *searchtypes.SearchNode) []string {
	var filters []string

	switch {
	case node.IsOperator(): // Handle AND/OR operators
		var childFilters []string // Collect child filters first
		for _, child := range node.Children {
			childFilters = append(childFilters, b.traverseAndGenerateFilters(child)...)
		}

		// Join child filters using the current node's operator
		if len(childFilters) > 0 {
			joiner := " AND "
			if node.Operator == searchtypes.OperatorOR {
				joiner = " OR "
			}
			filterString := strings.Join(childFilters, joiner)

			if strings.TrimSpace(filterString) != "" {
				filters = append(filters, "("+filterString+")")
			}

		}

	case node.Term != nil && (node.Operator == searchtypes.OperatorNone || node.Operator == searchtypes.OperatorNegation):
		var filter string
		switch node.Term.Identifier {
		case searchtypes.IdentifierAvailableOn:
			filter = b.availabilityFilter(node.Term.Value)
		case searchtypes.IdentifierName:
			filter = b.featureNameFilter(node.Term.Value)
		case searchtypes.IdentifierBaselineStatus:
			filter = b.baselineStatusFilter(node.Term.Value)
		}
		if filter != "" {
			if node.Operator == searchtypes.OperatorNegation {
				filter = "NOT (" + filter + ")"
			}
			filters = append(filters, "("+filter+")")
		}
	}

	return filters
}

func (b *FeatureSearchFilterBuilder) availabilityFilter(browser string) string {
	paramName := b.addParamGetName(browser)

	return fmt.Sprintf(`wf.FeatureID IN (SELECT FeatureID FROM BrowserFeatureAvailabilities
WHERE BrowserName = @%s)`, paramName)
}

func (b *FeatureSearchFilterBuilder) featureNameFilter(featureName string) string {
	// Normalize the string to lower case to use the computed column.
	featureName = strings.ToLower(featureName)
	// Safely add the database % wildcards if they do not already exist.
	if !strings.HasPrefix(featureName, "%") {
		featureName = "%" + featureName
	}
	if !strings.HasSuffix(featureName, "%") {
		featureName = featureName + "%"
	}

	paramName := b.addParamGetName(featureName)

	return fmt.Sprintf(`(wf.Name_Lowercase LIKE @%s OR wf.FeatureID_Lowercase LIKE @%s)`, paramName, paramName)
}

func (b *FeatureSearchFilterBuilder) baselineStatusFilter(baselineStatus string) string {
	var status BaselineStatus
	// baseline status is limited to the values in antlr/FeatureSearch.g4.
	switch baselineStatus {
	case "limited":
		status = BaselineStatusNone
	case "newly":
		status = BaselineStatusLow
	case "widely":
		status = BaselineStatusHigh
	default:
		// Catch-all returns an empty string which will be thrown away.
		return ""
	}
	paramName := b.addParamGetName(string(status))

	return fmt.Sprintf(`fbs.Status = @%s`, paramName)
}

// Filterable modifies a query with a given filter.
type Filterable interface {
	Params() map[string]interface{}
	Clause() string
}

// FeatureSearchQueryBuilder builds a query to search for features.
type FeatureSearchQueryBuilder struct {
	baseQuery FeatureSearchBaseQuery
	cursor    *FeatureResultCursor
	pageSize  int
}

func (q FeatureSearchQueryBuilder) Build(
	prefilter FeatureSearchPrefilterResult,
	filter *FeatureSearchCompiledFilter,
	sort Sortable) spanner.Statement {
	filterQuery := ""

	filterParams := make(map[string]interface{})
	if q.cursor != nil {
		filterParams["cursorId"] = q.cursor.LastFeatureID
		filterQuery += " wf.FeatureID > @cursorId"
	}

	filterParams["pageSize"] = q.pageSize

	if filter != nil {
		filterQuery = filter.Clause()
		maps.Copy(filterParams, filter.Params())
	}
	if len(filterQuery) > 0 {
		filterQuery = "WHERE " + filterQuery
	}

	sql, params := q.baseQuery.Query(prefilter)
	maps.Copy(filterParams, params)

	stmt := spanner.NewStatement(
		sql + " " + filterQuery + " ORDER BY " + sort.Clause() + " LIMIT @pageSize")

	stmt.Params = filterParams

	return stmt
}

// Sortable is a basic class that all/most sortables can include.
type Sortable struct {
	clause string
}

func (s Sortable) Clause() string {
	return s.clause
}

func buildSortableOrderClause(isAscending bool, column string) string {
	direction := "ASC"
	if !isAscending {
		direction = "DESC"
	}

	return fmt.Sprintf("%s %s", column, direction)
}

// NewFeatureNameSort returns a Sortable specifically for the Name column.
func NewFeatureNameSort(isAscending bool) Sortable {
	return Sortable{
		clause: buildSortableOrderClause(isAscending, "wf.Name"),
	}
}

// NewBaselineStatusSort returns a Sortable specifically for the Status column.
func NewBaselineStatusSort(isAscending bool) Sortable {
	return Sortable{
		clause: buildSortableOrderClause(isAscending, "Status"),
	}
}
