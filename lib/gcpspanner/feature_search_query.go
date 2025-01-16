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
	"time"

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
		node.Keyword != searchtypes.KeywordRoot ||
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
	case node.IsKeyword(): // Handle AND/OR keyword
		var childFilters []string // Collect child filters first
		for _, child := range node.Children {
			childFilters = append(childFilters, b.traverseAndGenerateFilters(child)...)
		}

		// Join child filters using the current node's operator
		if len(childFilters) > 0 {
			joiner := " AND "
			if node.Keyword == searchtypes.KeywordOR {
				joiner = " OR "
			}
			filterString := strings.Join(childFilters, joiner)

			if strings.TrimSpace(filterString) != "" {
				filters = append(filters, "("+filterString+")")
			}

		}

	case node.Term != nil && (node.Keyword == searchtypes.KeywordNone):
		var filter string
		switch node.Term.Identifier {
		case searchtypes.IdentifierAvailableDate:
			// Currently not a terminal identifier.
			break
		case searchtypes.IdentifierAvailableOn:
			filter = b.availabilityFilter(node.Term.Value, node.Term.Operator)
		case searchtypes.IdentifierName:
			filter = b.featureNameFilter(node.Term.Value, node.Term.Operator)
		case searchtypes.IdentifierGroup:
			filter = b.groupFilter(node.Term.Value, node.Term.Operator)
		case searchtypes.IdentifierSnapshot:
			filter = b.snapshotFilter(node.Term.Value, node.Term.Operator)
		case searchtypes.IdentifierID:
			filter = b.idFilter(node.Term.Value, node.Term.Operator)
		case searchtypes.IdentifierBaselineStatus:
			filter = b.baselineStatusFilter(node.Term.Value, node.Term.Operator)
		case searchtypes.IdentifierBaselineDate:
			filter = b.baselineDateFilter(node.Term.Value, node.Term.Operator)
		case searchtypes.IdentifierAvailableBrowserDate:
			filter = b.handleIdentifierAvailableBrowserDateTerm(node)
		}
		if filter != "" {
			filters = append(filters, "("+filter+")")
		}
	}

	return filters
}

func searchOperatorToSpannerBinaryOperator(in searchtypes.SearchOperator) string {
	switch in {
	case searchtypes.OperatorGt:
		return ">"
	case searchtypes.OperatorGtEq:
		return ">="
	case searchtypes.OperatorLt:
		return "<"
	case searchtypes.OperatorLtEq:
		return "<="
	case searchtypes.OperatorEq:
		return "="
	case searchtypes.OperatorNeq:
		return "!="
	case searchtypes.OperatorNone:
		fallthrough
	default:
		// If caller tries to pass a string that actually is not an operator, default to =
		return "="
	}
}

func searchOperatorToSpannerListOperator(in searchtypes.SearchOperator) string {
	switch in {
	case searchtypes.OperatorEq:
		return "IN"
	case searchtypes.OperatorNeq:
		return "NOT IN"
	case searchtypes.OperatorGt,
		searchtypes.OperatorGtEq,
		searchtypes.OperatorLt,
		searchtypes.OperatorLtEq,
		searchtypes.OperatorNone:
		fallthrough
	default:
		// Default to "IN". Callers should know the filter is applying to a list and the searchNode should have
		// either Eq or Neq. Return "IN" to produce correct sql syntax.
		return "IN"
	}
}

func searchOperatorToSpannerStringPatternOperator(in searchtypes.SearchOperator) string {
	switch in {
	case searchtypes.OperatorEq:
		return "LIKE"
	case searchtypes.OperatorNeq:
		return "NOT LIKE"
	case searchtypes.OperatorGt,
		searchtypes.OperatorGtEq,
		searchtypes.OperatorLt,
		searchtypes.OperatorLtEq,
		searchtypes.OperatorNone:
		fallthrough
	default:
		// Default to "NOT LIKE". Callers should know the filter is applying to a string pattern and the
		// searchNode should have either Eq or Neq. Return "LIKE" to produce correct sql syntax.
		return "LIKE"
	}
}

func (b *FeatureSearchFilterBuilder) availabilityFilter(browser string, op searchtypes.SearchOperator) string {
	paramName := b.addParamGetName(browser)

	return fmt.Sprintf(`wf.ID %s (SELECT WebFeatureID FROM BrowserFeatureAvailabilities
WHERE BrowserName = @%s)`, searchOperatorToSpannerListOperator(op), paramName)
}

func (b *FeatureSearchFilterBuilder) handleIdentifierAvailableBrowserDateTerm(node *searchtypes.SearchNode) string {
	if len(node.Children) != 2 {
		return ""
	}
	var browserNode, dateNode *searchtypes.SearchNode
	for idx := range node.Children {
		if node.Children[idx].Term.Identifier == searchtypes.IdentifierAvailableOn {
			browserNode = node.Children[idx]
		} else if node.Children[idx].Term.Identifier == searchtypes.IdentifierAvailableDate {
			dateNode = node.Children[idx]
		}
	}
	if browserNode == nil || dateNode == nil {
		return ""
	}

	return b.availableBrowserDateFilter(
		browserNode.Term.Value,
		dateNode.Term.Value,
		browserNode.Term.Operator,
		dateNode.Term.Operator,
	)
}

func (b *FeatureSearchFilterBuilder) availableBrowserDateFilter(
	browser, rawDate string,
	browserOp, dateOp searchtypes.SearchOperator) string {
	date, err := time.Parse(time.DateOnly, rawDate)
	if err != nil {
		// an empty string which will be thrown away by the filter builder
		return ""
	}
	browserParamName := b.addParamGetName(browser)

	dateParamName := b.addParamGetName(date)

	// Can't directly use browser_info here because Spanner doesn't support
	// accessing fields of a struct array within a subquery when that same
	// array is also in the main query's result set.
	// Check this issue: https://github.com/GoogleChrome/webstatus.dev/issues/576
	return fmt.Sprintf(`
	wf.ID IN (
		SELECT bfa.WebFeatureID
		FROM BrowserFeatureAvailabilities bfa
		JOIN BrowserReleases br
			ON bfa.BrowserName = br.BrowserName AND bfa.BrowserVersion = br.BrowserVersion
		WHERE
			br.BrowserName %s @%s
			AND br.ReleaseDate %s @%s
	)
    `,
		searchOperatorToSpannerBinaryOperator(browserOp), browserParamName,
		searchOperatorToSpannerBinaryOperator(dateOp), dateParamName,
	)
}

func (b *FeatureSearchFilterBuilder) featureNameFilter(featureName string, op searchtypes.SearchOperator) string {
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

	opStr := searchOperatorToSpannerStringPatternOperator(op)

	return fmt.Sprintf(`(wf.Name_Lowercase %s @%s OR wf.FeatureKey_Lowercase %s @%s)`, opStr, paramName,
		opStr, paramName)
}

func (b *FeatureSearchFilterBuilder) groupFilter(group string, op searchtypes.SearchOperator) string {
	// Normalize the string to lower case to use the computed column.
	group = strings.ToLower(group)

	paramName := b.addParamGetName(group)

	opStr := searchOperatorToSpannerBinaryOperator(op)

	return fmt.Sprintf(`
    wf.ID IN (
        SELECT wfg.WebFeatureID
        FROM WebFeatureGroups wfg
        WHERE
            EXISTS (
                SELECT 1
                FROM WebDXGroups g
                LEFT JOIN WebDXGroupDescendants gd ON g.ID = gd.GroupID
                WHERE g.GroupKey_Lowercase %s @%s
                  AND (
                      g.ID IN UNNEST(wfg.GroupIDs)
                      OR
                      ARRAY_INCLUDES_ANY(gd.DescendantGroupIDs, wfg.GroupIDs)
                  )
            )
    )
    `, opStr, paramName)
}

func (b *FeatureSearchFilterBuilder) snapshotFilter(snapshot string, op searchtypes.SearchOperator) string {
	// Normalize the string to lower case to use the computed column.
	snapshot = strings.ToLower(snapshot)

	paramName := b.addParamGetName(snapshot)

	opStr := searchOperatorToSpannerBinaryOperator(op)

	return fmt.Sprintf(`
    wf.ID IN (
        SELECT wfs.WebFeatureID
        FROM WebFeatureSnapshots wfs
        WHERE
            EXISTS (
                SELECT 1
                FROM WebDXSnapshots s
                WHERE s.SnapshotKey_Lowercase %s @%s AND s.ID IN UNNEST(wfs.SnapshotIDs)
            )
    )
    `, opStr, paramName)
}

func (b *FeatureSearchFilterBuilder) idFilter(id string, op searchtypes.SearchOperator) string {
	// Normalize the string to lower case to use the computed column.
	id = strings.ToLower(id)
	paramName := b.addParamGetName(id)
	opStr := searchOperatorToSpannerBinaryOperator(op)

	return fmt.Sprintf(`(wf.FeatureKey_Lowercase %s @%s)`, opStr, paramName)
}

func (b *FeatureSearchFilterBuilder) baselineStatusFilter(baselineStatus string, op searchtypes.SearchOperator) string {
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

	return fmt.Sprintf(`fbs.Status %s @%s`, searchOperatorToSpannerBinaryOperator(op), paramName)
}

func (b *FeatureSearchFilterBuilder) baselineDateFilter(rawDate string, op searchtypes.SearchOperator) string {
	date, err := time.Parse(time.DateOnly, rawDate)
	if err != nil {
		// an empty string which will be thrown away by the filter builder
		return ""
	}

	paramName := b.addParamGetName(date)

	return fmt.Sprintf(`LowDate %s @%s`, searchOperatorToSpannerBinaryOperator(op), paramName)
}

// Exclude all that do not have an entry in ExcludedFeatureKeys.
const removeExcludedKeyFilter = "efk.FeatureKey IS NULL"
const removeExcludedKeyFilterAND = "AND " + removeExcludedKeyFilter

func defaultFeatureSearchFilters() []string {
	return []string{
		removeExcludedKeyFilter,
	}
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
	browsers      []string
}

func (q FeatureSearchQueryBuilder) CountQueryBuild(
	filter *FeatureSearchCompiledFilter) spanner.Statement {
	filterParams := make(map[string]interface{})
	args := FeatureSearchCountArgs{
		Filters: nil,
	}
	args.Filters = defaultFeatureSearchFilters()
	if filter != nil {
		args.Filters = append(args.Filters, filter.filters...)
		maps.Copy(filterParams, filter.Params())
	}

	sql := q.baseQuery.CountQuery(args)

	stmt := spanner.NewStatement(sql)
	stmt.Params = filterParams

	return stmt
}

func (q FeatureSearchQueryBuilder) Build(
	filter *FeatureSearchCompiledFilter,
	sort Sortable,
	pageSize int) spanner.Statement {

	var stableBrowserImplDetails, expBrowserImplDetails *SortByBrowserImplDetails
	var browserFeatureSupportDetails *SortByBrowserFeatureSupportDetails

	switch sort.SortTarget() {
	case StableImplSort:
		stableBrowserImplDetails = &SortByBrowserImplDetails{
			BrowserName: sort.BrowserTarget(),
		}
	case ExperimentalImplSort:
		expBrowserImplDetails = &SortByBrowserImplDetails{
			BrowserName: sort.BrowserTarget(),
		}
	case BrowserFeatureSupportSort:
		browserFeatureSupportDetails = &SortByBrowserFeatureSupportDetails{
			BrowserName: sort.BrowserTarget(),
		}
	case ChromiumUsageSort, IDSort, NameSort, StatusSort:
		break // do nothing.
	}

	filterParams := make(map[string]interface{})
	queryArgs := FeatureSearchQueryArgs{
		MetricView:  q.wptMetricView,
		Filters:     nil,
		PageFilters: nil,
		Offset:      0,
		PageSize:    pageSize,
		Browsers:    q.browsers,
		SortClause:  sort.Clause(),
		// Special Sort Targets.
		SortByStableBrowserImpl:     stableBrowserImplDetails,
		SortByExpBrowserImpl:        expBrowserImplDetails,
		SortByBrowserFeatureSupport: browserFeatureSupportDetails,
	}

	if q.offsetCursor != nil {
		queryArgs.Offset = q.offsetCursor.Offset
	}
	queryArgs.Filters = defaultFeatureSearchFilters()
	if filter != nil {
		queryArgs.Filters = append(queryArgs.Filters, filter.filters...)
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
func buildFullClause(sortableClauses []string) string {
	return strings.Join(append(sortableClauses, string(featureSearchFeatureKeyColumn)), ", ")
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
		featureSearchBrowserMetricColumn,
		featureSearchLowDateColumn,
		featureSearchHighDateColumn,
		featureSearchBrowserImplColumn,
		featureSearchStatusColumn,
		featureSearchChromiumUsageColumn,
		featureSearchBrowserFeatureSupportDateColumn:
		return string(f)
	}

	return ""
}

// FeaturesSearchSortTarget is an enumeration of the data that is being targeted for the sort operation.
// This is used to know which column(s) to encode and decode in the pagination token.
type FeaturesSearchSortTarget string

const (
	IDSort                    FeaturesSearchSortTarget = "id"
	NameSort                  FeaturesSearchSortTarget = "name"
	StatusSort                FeaturesSearchSortTarget = "status"
	StableImplSort            FeaturesSearchSortTarget = "stable_browser_impl"
	ExperimentalImplSort      FeaturesSearchSortTarget = "experimental_browser_impl"
	ChromiumUsageSort         FeaturesSearchSortTarget = "chromium_usage"
	BrowserFeatureSupportSort FeaturesSearchSortTarget = "browser_feature_support"
)

const (
	featureSearchFeatureKeyColumn                FeatureSearchColumn = "wf.FeatureKey"
	featureSearchFeatureNameColumn               FeatureSearchColumn = "wf.Name"
	featureSearchStatusColumn                    FeatureSearchColumn = "Status"
	featureSearchLowDateColumn                   FeatureSearchColumn = "LowDate"
	featureSearchHighDateColumn                  FeatureSearchColumn = "HighDate"
	featureSearchBrowserMetricColumn             FeatureSearchColumn = "sort_metric_calcs.SortMetric"
	featureSearchBrowserImplColumn               FeatureSearchColumn = "sort_impl_calcs.SortImplStatus"
	featureSearchChromiumUsageColumn             FeatureSearchColumn = "chromium_usage_metrics.ChromiumUsage"
	featureSearchBrowserFeatureSupportDateColumn FeatureSearchColumn = "sort_browser_feature_support_calcs.SortDate"
)

const (
	derviedTableSortMetrics               = "sort_metric_calcs"
	derviedTableSortImpl                  = "sort_impl_calcs"
	derivedTableSortBrowserFeatureSupport = "sort_browser_feature_support_calcs"
)

// NewFeatureNameSort returns a Sortable specifically for the Name column.
func NewFeatureNameSort(isAscending bool) Sortable {
	return Sortable{
		clause: buildFullClause(
			[]string{buildSortableOrderClause(isAscending, featureSearchFeatureNameColumn)}),
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
				buildSortableOrderClause(isAscending, featureSearchBrowserMetricColumn),
				buildSortableOrderClause(isAscending, featureSearchBrowserImplColumn),
			},
		),
		browserTarget:  &browserName,
		ascendingOrder: isAscending,
		sortTarget:     sortTarget,
	}
}

// NewChromiumUsageSort returns a Sortable specifically for the ChromiumUsage column.
func NewChromiumUsageSort(isAscending bool) Sortable {
	return Sortable{
		clause: buildFullClause(
			[]string{
				buildSortableOrderClause(isAscending, featureSearchChromiumUsageColumn),
			},
		),
		ascendingOrder: isAscending,
		sortTarget:     ChromiumUsageSort,
		browserTarget:  nil,
	}
}

// NewBrowserFeatureSupportSort creates a Sortable configuration for ordering Web Features.
// The primary sorting criterion is the date of when the feature became available.
//
// Arguments:
//   - isAscending: Whether the sorting should be ascending (true) or descending (false).
//   - browserName: The name of the browser ("chrome", "firefox", etc.).
func NewBrowserFeatureSupportSort(isAscending bool, browserName string) Sortable {
	sortTarget := BrowserFeatureSupportSort

	return Sortable{
		clause: buildFullClause(
			[]string{
				buildSortableOrderClause(isAscending, featureSearchBrowserFeatureSupportDateColumn),
				// TODO. Pass isAscending to buildFullClause so that featureSearchFeatureKeyColumn there gets it.
				buildSortableOrderClause(isAscending, featureSearchFeatureKeyColumn),
			},
		),
		browserTarget:  &browserName,
		ascendingOrder: isAscending,
		sortTarget:     sortTarget,
	}
}
