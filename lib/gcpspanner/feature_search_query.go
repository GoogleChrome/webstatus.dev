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
	params  map[string]interface{}
	filters []string
}

func (f FeatureSearchCompiledFilter) Params() map[string]interface{} {
	return f.params
}

func (f FeatureSearchCompiledFilter) Filters() []string {
	return f.filters
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

// Build constructs a Spanner query for the FeaturesSearch function.
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

	return &FeatureSearchCompiledFilter{
		params:  b.params,
		filters: generatedFilters,
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

	return fmt.Sprintf(`wf.ID IN (SELECT WebFeatureID FROM BrowserFeatureAvailabilities
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

	return fmt.Sprintf(`(wf.Name_Lowercase LIKE @%s OR wf.FeatureKey_Lowercase LIKE @%s)`, paramName, paramName)
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
	baseQuery     FeatureSearchBaseQuery
	offsetCursor  *FeatureResultOffsetCursor
	wptMetricView WPTMetricView
}

func (q FeatureSearchQueryBuilder) CountQueryBuild(
	filter *FeatureSearchCompiledFilter) spanner.Statement {
	filterParams := make(map[string]interface{})
	args := FeatureSearchCountArgs{
		Filters: nil,
	}
	if filter != nil {
		args.Filters = filter.filters
		maps.Copy(filterParams, filter.Params())
	}

	sql := q.baseQuery.CountQuery(args)

	stmt := spanner.NewStatement(sql)
	stmt.Params = filterParams

	return stmt
}

func (q FeatureSearchQueryBuilder) Build(
	prefilter FeatureSearchPrefilterResult,
	filter *FeatureSearchCompiledFilter,
	sort Sortable,
	pageSize int) spanner.Statement {

	var stableBrowserImplDetails, expBrowserImplDetails *SortByBrowserImplDetails

	switch sort.SortTarget() {
	case StableImplSort:
		stableBrowserImplDetails = &SortByBrowserImplDetails{
			BrowserName: sort.BrowserTarget(),
		}
	case ExperimentalImplSort:
		expBrowserImplDetails = &SortByBrowserImplDetails{
			BrowserName: sort.BrowserTarget(),
		}
	case IDSort, NameSort, StatusSort:
		break // do nothing.
	}

	filterParams := make(map[string]interface{})
	queryArgs := FeatureSearchQueryArgs{
		MetricView:  q.wptMetricView,
		Filters:     nil,
		PageFilters: nil,
		Offset:      0,
		PageSize:    pageSize,
		Prefilter:   prefilter,
		SortClause:  sort.Clause(),
		// Special Sort Targets.
		SortByStableBrowserImpl: stableBrowserImplDetails,
		SortByExpBrowserImpl:    expBrowserImplDetails,
	}

	if q.offsetCursor != nil {
		queryArgs.Offset = q.offsetCursor.Offset
	}
	if filter != nil {
		queryArgs.Filters = filter.filters
		maps.Copy(filterParams, filter.Params())
	}

	sql, params := q.baseQuery.Query(queryArgs)
	maps.Copy(filterParams, params)

	stmt := spanner.NewStatement(sql)

	stmt.Params = filterParams

	return stmt
}

// Sortable is a basic class that all/most sortables can include.
type Sortable struct {
	clause         string
	sortTarget     FeaturesSearchSortTarget
	ascendingOrder bool
	browserTarget  *string
}

func (s Sortable) BrowserTarget() string {
	if s.browserTarget == nil {
		return ""
	}

	return *s.browserTarget
}

func (s Sortable) Clause() string {
	return s.clause
}

func (s Sortable) SortTarget() FeaturesSearchSortTarget {
	return s.sortTarget
}

// buildFullClause generates a sorting clause appropriate for Spanner pagination.
// It includes the primary sorting column and the 'WebFeatureID' column as a tiebreaker
// to ensure deterministic page ordering.
func buildFullClause(sortableClauses []string, tieBreakerColumn FeatureSearchColumn) string {
	return strings.Join(append(sortableClauses, string(tieBreakerColumn)), ", ")
}

func buildSortableOrderClause(isAscending bool, column FeatureSearchColumn) string {
	direction := "ASC"
	if !isAscending {
		direction = "DESC"
	}

	return fmt.Sprintf("%s %s", column, direction)
}

// FeatureSearchColumn is the high level column returned in the FeatureSearch Query from spanner.
type FeatureSearchColumn string

func (f FeatureSearchColumn) ToFilterColumn() string {
	switch f {
	case featureSearchFeatureKeyColumn,
		featureSearchFeatureNameColumn,
		featureSearcBrowserMetricColumn,
		featureSearchLowDateColumn,
		featureSearchHighDateColumn,
		featureSearcBrowserImplColumn,
		featureSearchStatusColumn:
		return string(f)
	}

	return ""
}

// FeaturesSearchSortTarget is an enumeration of the data that is being targeted for the sort operation.
// This is used to know which column(s) to encode and decode in the pagination token.
type FeaturesSearchSortTarget string

const (
	IDSort               FeaturesSearchSortTarget = "id"
	NameSort             FeaturesSearchSortTarget = "name"
	StatusSort           FeaturesSearchSortTarget = "status"
	StableImplSort       FeaturesSearchSortTarget = "stable_browser_impl"
	ExperimentalImplSort FeaturesSearchSortTarget = "experimental_browser_impl"
)

const (
	featureSearchFeatureKeyColumn   FeatureSearchColumn = "wf.FeatureKey"
	featureSearchFeatureNameColumn  FeatureSearchColumn = "wf.Name"
	featureSearchStatusColumn       FeatureSearchColumn = "Status"
	featureSearchLowDateColumn      FeatureSearchColumn = "LowDate"
	featureSearchHighDateColumn     FeatureSearchColumn = "HighDate"
	featureSearcBrowserMetricColumn FeatureSearchColumn = "sort_metric_calcs.SortMetric"
	featureSearcBrowserImplColumn   FeatureSearchColumn = "sort_impl_calcs.SortImplStatus"
)

const (
	derviedTableSortMetrics = "sort_metric_calcs"
	derviedTableSortImpl    = "sort_impl_calcs"
)

// NewFeatureNameSort returns a Sortable specifically for the Name column.
func NewFeatureNameSort(isAscending bool) Sortable {
	return Sortable{
		clause: buildFullClause(
			[]string{buildSortableOrderClause(isAscending, featureSearchFeatureNameColumn)},
			featureSearchFeatureKeyColumn),
		ascendingOrder: isAscending,
		sortTarget:     NameSort,
		browserTarget:  nil,
	}
}

// NewBaselineStatusSort returns a Sortable specifically for the Status column.
func NewBaselineStatusSort(isAscending bool) Sortable {
	return Sortable{
		clause: buildFullClause(
			[]string{
				buildSortableOrderClause(isAscending, featureSearchLowDateColumn),
				buildSortableOrderClause(isAscending, featureSearchHighDateColumn),
				buildSortableOrderClause(isAscending, featureSearchStatusColumn),
			},
			featureSearchFeatureKeyColumn,
		),
		ascendingOrder: isAscending,
		sortTarget:     StatusSort,
		browserTarget:  nil,
	}
}

// NewBrowserImplSort creates a Sortable configuration for ordering Web Features.
// The primary sorting criterion is the pass rate of stable or experimental WPT (Web Platform Tests) metrics
// for the specified browser.
// The secondary sorting criterion is the implementation status ("available" or "unavailable") of the feature
// in the specified browser.
//
// Arguments:
//   - isAscending: Whether the sorting should be ascending (true) or descending (false).
//   - browserName: The name of the browser ("chrome", "firefox", etc.).
//   - isStable: Whether to use stable (true) or experimental (false) WPT metrics.
func NewBrowserImplSort(isAscending bool, browserName string, isStable bool) Sortable {
	sortTarget := StableImplSort
	if !isStable {
		sortTarget = ExperimentalImplSort
	}

	return Sortable{
		clause: buildFullClause(
			[]string{
				buildSortableOrderClause(isAscending, featureSearcBrowserMetricColumn),
				buildSortableOrderClause(isAscending, featureSearcBrowserImplColumn),
			},
			featureSearchFeatureKeyColumn,
		),
		browserTarget:  &browserName,
		ascendingOrder: isAscending,
		sortTarget:     sortTarget,
	}
}
