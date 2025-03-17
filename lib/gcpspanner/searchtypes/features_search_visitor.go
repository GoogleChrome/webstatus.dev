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

package searchtypes

import (
	"errors"
	"fmt"
	"strings"

	parser "github.com/GoogleChrome/webstatus.dev/lib/gen/featuresearch/parser/antlr"
	"github.com/antlr4-go/antlr/v4"
)

// https://github.com/antlr/antlr4/pull/1841#issuecomment-576791512
// https://github.com/antlr/antlr4/issues/2504#issuecomment-1299123230
type FeaturesSearchVisitor struct {
	err error
	parser.BaseFeatureSearchVisitor
}

func getOperatorType(text string) SearchKeyword {
	switch text {
	case "AND":
		return KeywordAND
	case "OR":
		return KeywordOR
	}

	return KeywordNone
}

func (v *FeaturesSearchVisitor) addError(err error) {
	if v.err == nil {
		v.err = err

		return
	}
	v.err = errors.Join(v.err, err)
}

type termMissingValueError struct {
	term SearchIdentifier
}

func (e termMissingValueError) Error() string {
	return fmt.Sprintf("term %s is missing value", e.term)
}

type termMissingRangeValueError struct {
	term SearchIdentifier
}

func (e termMissingRangeValueError) Error() string {
	return fmt.Sprintf("term %s is missing value range", e.term)
}

func (v *FeaturesSearchVisitor) handleOperator(current *SearchNode, operatorCtx *parser.OperatorContext) *SearchNode {
	operator := getOperatorType(operatorCtx.GetText())

	newNode := &SearchNode{
		Keyword:  operator,
		Term:     nil,
		Children: nil,
	}

	if current != nil {
		// Detect implicit AND case and prioritize flat groupings.
		// This logic handles the optional chaining structure in the grammar, where
		// terms/groups without explicit operators are implicitly joined using AND.
		// The implicit AND nodes themselves are created primarily within the 'chainWithImplicitAND' function.
		if current.Keyword == KeywordAND && len(current.Children) == 1 {
			current.Children = append(current.Children, newNode)
		} else {
			// Handle the case where a new explicit operator node is encountered:
			// 1. The 'newNode' (representing the explicit operator) becomes the root of a new
			//    operator sub-tree.
			// 2. The previous 'current' node (term, implicit AND chain, or sub-tree) becomes
			//    the first child of this new sub-tree. `current`` at this point would be the left operand
			//    that was already encountered in VisitCombined_search_criteria. That is why it becomes
			//    the child of the explicit operator, newNode.
			// 3. The 'current' reference is updated to the 'newNode', making it the focus of
			//    subsequent parsing within the combined search criteria.
			newNode.Children = append(newNode.Children, current)
			current = newNode // Update current to the new operator node
		}
	}

	return current
}

// chainWithImplicitAND incorporates a new node into a query structure. It creates,
// extends, or prepares implicit AND groupings (originating from optional chaining
// in the 'combined_search_criteria' grammar rule) for handling by 'aggregateNodesImplicitAND'.
func (v *FeaturesSearchVisitor) chainWithImplicitAND(current, newNode *SearchNode) *SearchNode {
	if current == nil {
		// Case: Starting a new chain.
		return newNode
	} else if !current.IsKeyword() {
		// Case: Chaining onto a term node, create an implicit AND.
		return &SearchNode{Term: nil, Keyword: KeywordAND, Children: []*SearchNode{current, newNode}}
	}
	// Case: Continue an existing AND chain.
	current.Children = append(current.Children, newNode)

	return current

}

// aggregateNodesImplicitAND builds on the structure created by 'handleOperator and 'chainWithImplicitAND'.
// It constructs a single SearchNode representing an implicit AND group of search criteria
// (originating from the optional chaining in the 'combined_search_criteria' grammar rule).
func (v *FeaturesSearchVisitor) aggregateNodesImplicitAND(nodes []*SearchNode) *SearchNode {
	if len(nodes) == 0 {
		return nil
	}

	// Create the root node of the implicit AND tree. This becomes the starting
	// point for subsequent chaining.
	rootNode := nodes[0]

	// Iterate through remaining nodes, extending the AND chain.
	for _, node := range nodes[1:] {
		// Chain subsequent nodes as children using implicit AND operators.
		// Relies on 'chainWithImplicitAND' to handle the chaining structure.
		rootNode = &SearchNode{
			Term:     nil,
			Keyword:  KeywordAND,
			Children: []*SearchNode{rootNode, node},
		}
	}

	return rootNode
}

func (v *FeaturesSearchVisitor) createIDNode(idNode antlr.TerminalNode) *SearchNode {
	return v.createSimpleNode(idNode, IdentifierID)
}

func (v *FeaturesSearchVisitor) createSnapshotNode(snapshotNode antlr.TerminalNode) *SearchNode {
	return v.createSimpleNode(snapshotNode, IdentifierSnapshot)
}

func (v *FeaturesSearchVisitor) createGroupNode(groupNode antlr.TerminalNode) *SearchNode {
	return v.createSimpleNode(groupNode, IdentifierGroup)
}

func (v *FeaturesSearchVisitor) createNameNode(nameNode antlr.TerminalNode) *SearchNode {
	return v.createSimpleNode(nameNode, IdentifierName)
}

func (v *FeaturesSearchVisitor) createSimpleNode(
	node antlr.TerminalNode,
	identifier SearchIdentifier) *SearchNode {
	if node == nil {
		v.addError(termMissingValueError{term: identifier})

		return nil
	}
	value := node.GetText()
	value = strings.Trim(value, `"`)

	return &SearchNode{
		Keyword:  KeywordNone,
		Children: nil,
		Term: &SearchTerm{
			Identifier: identifier,
			Value:      value,
			Operator:   OperatorEq,
		},
	}
}

/*
The below section implements the generated BaseFeatureSearchVisitor methods.
*/

func (v *FeaturesSearchVisitor) VisitQuery(ctx *parser.QueryContext) interface{} {
	// Create root node.
	root := &SearchNode{Keyword: KeywordRoot, Term: nil, Children: nil}
	childResult, ok := v.VisitChildren(ctx).(*SearchNode)
	if !ok {
		v.addError(fmt.Errorf("VisitQuery did not receive a SearchNode"))
	} else {
		root.Children = append(root.Children, childResult)
	}

	return root
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitAvailable_on_term(ctx *parser.Available_on_termContext) interface{} {
	browserNameNode := ctx.BROWSER_NAME()
	if browserNameNode == nil {
		v.addError(termMissingValueError{term: IdentifierAvailableOn})

		return nil
	}
	browserName := strings.ToLower(ctx.BROWSER_NAME().GetText())

	return &SearchNode{
		Keyword: KeywordNone,
		Term: &SearchTerm{
			Identifier: IdentifierAvailableOn,
			Value:      browserName,
			Operator:   OperatorEq,
		},
		Children: nil,
	}
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitBaseline_status_term(ctx *parser.Baseline_status_termContext) interface{} {
	baselineStatusNode := ctx.BASELINE_STATUS()
	if baselineStatusNode == nil {
		v.addError(termMissingValueError{term: IdentifierBaselineStatus})

		return nil
	}
	baselineStatus := baselineStatusNode.GetText()

	return &SearchNode{
		Keyword: KeywordNone,
		Term: &SearchTerm{
			Identifier: IdentifierBaselineStatus,
			Value:      baselineStatus,
			Operator:   OperatorEq,
		},
		Children: nil,
	}
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitAvailable_date_term(ctx *parser.Available_date_termContext) interface{} {
	var browserName string
	if browserCtx := ctx.BROWSER_NAME(); browserCtx != nil {
		browserName = strings.ToLower(browserCtx.GetText())
	}
	if dateRangeCtx := ctx.Date_range_query(); dateRangeCtx != nil {
		if browserName != "" {
			return v.VisitAvailableBrowserDateTerm(dateRangeCtx, browserName)
		}
	}

	// Otherwise, use the default behavior of the visitor.
	return v.VisitChildren(ctx)
}

func (v *FeaturesSearchVisitor) VisitAvailableBrowserDateTerm(
	ctx parser.IDate_range_queryContext, browserName string) interface{} {
	startDate := ctx.GetStartDate()
	endDate := ctx.GetEndDate()

	if startDate == nil || endDate == nil {
		v.addError(termMissingRangeValueError{term: IdentifierAvailableBrowserDate})

		return nil
	}

	// Create two nodes for start and end dates
	startDateNode := &SearchNode{
		Keyword: KeywordNone,
		Term: &SearchTerm{
			Identifier: IdentifierAvailableBrowserDate,
			Operator:   OperatorNone,
			Value:      "",
		},
		Children: []*SearchNode{
			{
				Keyword: KeywordNone,
				Term: &SearchTerm{
					Identifier: IdentifierAvailableOn,
					Value:      browserName,
					Operator:   OperatorEq,
				},
				Children: nil,
			},
			{
				Keyword: KeywordNone,
				Term: &SearchTerm{
					Identifier: IdentifierAvailableDate,
					Value:      startDate.GetText(),
					Operator:   OperatorGtEq,
				},
				Children: nil,
			},
		},
	}

	endDateNode := &SearchNode{
		Keyword: KeywordNone,
		Term: &SearchTerm{
			Identifier: IdentifierAvailableBrowserDate,
			Operator:   OperatorNone,
			Value:      "",
		},
		Children: []*SearchNode{
			{
				Keyword: KeywordNone,
				Term: &SearchTerm{
					Identifier: IdentifierAvailableOn,
					Value:      browserName,
					Operator:   OperatorEq,
				},
				Children: nil,
			},
			{
				Keyword: KeywordNone,
				Term: &SearchTerm{
					Identifier: IdentifierAvailableDate,
					Value:      endDate.GetText(),
					Operator:   OperatorLtEq,
				},
				Children: nil,
			},
		},
	}

	// Create a parent AND node to combine the two date conditions
	return &SearchNode{
		Keyword:  KeywordAND,
		Term:     nil,
		Children: []*SearchNode{startDateNode, endDateNode},
	}
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitBaseline_date_term(ctx *parser.Baseline_date_termContext) interface{} {
	if dateRangeCtx := ctx.Date_range_query(); dateRangeCtx != nil {
		return v.VisitDateRangeQuery(dateRangeCtx, IdentifierBaselineDate)
	}

	// Otherwise, use the default behavior of the visitor.
	return v.VisitChildren(ctx)
}

// VisitDateRangeQuery is not part of the generated methods. It is specialized to handle queries that
// have a date range context. The generated VisitDate_range_query is no longer needed.
func (v *FeaturesSearchVisitor) VisitDateRangeQuery(ctx parser.IDate_range_queryContext,
	identifier SearchIdentifier) *SearchNode {
	startDateNode := ctx.GetStartDate()
	endDateNode := ctx.GetEndDate()

	if startDateNode == nil || endDateNode == nil {
		v.addError(termMissingRangeValueError{term: identifier})

		return nil
	}

	startDate := startDateNode.GetText()
	endDate := endDateNode.GetText()

	return &SearchNode{
		Keyword: KeywordAND,
		Term:    nil,
		Children: []*SearchNode{
			{
				Keyword: KeywordNone,
				Term: &SearchTerm{
					Identifier: identifier,
					Value:      startDate,
					Operator:   OperatorGtEq,
				},
				Children: nil,
			},
			{
				Keyword: KeywordNone,
				Term: &SearchTerm{
					Identifier: identifier,
					Value:      endDate,
					Operator:   OperatorLtEq,
				},
				Children: nil,
			},
		},
	}
}

func (v *FeaturesSearchVisitor) VisitChildren(node antlr.RuleNode) interface{} {
	var resultNodes []*SearchNode
	for _, child := range node.GetChildren() {
		// Only process non-terminal nodes.
		if _, ok := child.(antlr.TerminalNode); ok {
			continue
		}

		if parseTreeNode, ok := child.(antlr.ParseTree); ok {
			if childResult, ok := v.Visit(parseTreeNode).(*SearchNode); ok {
				resultNodes = append(resultNodes, childResult)
			} else {
				v.addError(fmt.Errorf("VisitChildren did not receive a SearchNode"))
			}
		} else {
			v.addError(fmt.Errorf("VisitChildren could not convert to antlr.ParseTree"))
		}

	}

	// Aggregation Logic based on node type.
	if _, ok := node.(*parser.Combined_search_criteriaContext); ok {
		// Constructs a single SearchNode with an AND operator to represent the combined criteria,
		// using the collected child nodes.
		return v.aggregateNodesImplicitAND(resultNodes)
	} else if len(resultNodes) > 0 {
		return resultNodes[0] // Only return the first node for now.
	}

	v.addError(fmt.Errorf("VisitChildren returning nil for node. %s", node.GetText()))

	return nil
}

func (v *FeaturesSearchVisitor) Visit(tree antlr.ParseTree) any {
	return tree.Accept(v)
}

func (v *FeaturesSearchVisitor) VisitParenthesizedCriteria(ctx *parser.ParenthesizedCriteriaContext) interface{} {
	combinedCtx, ok := ctx.Combined_search_criteria().(*parser.Combined_search_criteriaContext)
	if !ok {
		return nil
	}
	node := &SearchNode{
		Keyword:  KeywordParens,
		Term:     nil,
		Children: nil,
	}

	// Recursively visit the combined_search_criteria inside the parentheses.
	if subTree, ok := v.VisitCombined_search_criteria(combinedCtx).(*SearchNode); ok {
		node.Children = append(node.Children, subTree)
	} else {
		v.addError(fmt.Errorf("VisitParenthesizedCriteria did not receive SearchNode for visit"))
	}

	return node
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitCombined_search_criteria(ctx *parser.Combined_search_criteriaContext) interface{} {
	var root *SearchNode
	var current *SearchNode

	for i := 0; i < ctx.GetChildCount(); i++ {
		child := ctx.GetChild(i)
		switch node := child.(type) {
		case *parser.OperatorContext:
			current = v.handleOperator(current, node)
		case *parser.Search_criteriaContext:
			if parseTreeNode, ok := child.(antlr.ParseTree); ok {
				if childNode, ok := v.Visit(parseTreeNode).(*SearchNode); ok {
					current = v.chainWithImplicitAND(current, childNode)
				} else {
					// Handle case where Visit didn't return a SearchNode.
					v.addError(fmt.Errorf("VisitCombined_search_criteria did not receive SearchNode for visit"))
				}
			} else {
				// Handle case where child cannot be converted to ParseTree.
				v.addError(fmt.Errorf("VisitCombined_search_criteria could not convert to antlr.ParseTree"))
			}
		case *parser.Combined_search_criteriaContext:
			// Handle recursive case (subtrees).
			if subTree, ok := v.VisitCombined_search_criteria(node).(*SearchNode); ok {
				current = v.chainWithImplicitAND(current, subTree)
			} else {
				v.addError(fmt.Errorf("VisitCombined_search_criteria did not receive SearchNode for visit"))
			}
		case *parser.ParenthesizedCriteriaContext:
			// Handle parenthesized criteria.
			if subTree, ok := v.VisitParenthesizedCriteria(node).(*SearchNode); ok {
				current = v.chainWithImplicitAND(current, subTree)
			} else {
				v.addError(fmt.Errorf("VisitCombined_search_criteria did not receive SearchNode for visit"))
			}
		default:
			// Future cases will call into the default.
		}
	}

	root = current

	return root
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitId_term(ctx *parser.Id_termContext) interface{} {
	return v.createIDNode(ctx.ANY_VALUE())
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitSnapshot_term(ctx *parser.Snapshot_termContext) interface{} {
	return v.createSnapshotNode(ctx.ANY_VALUE())
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitGroup_term(ctx *parser.Group_termContext) interface{} {
	return v.createGroupNode(ctx.ANY_VALUE())
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitName_term(ctx *parser.Name_termContext) interface{} {
	return v.createNameNode(ctx.ANY_VALUE())
}

func (v *FeaturesSearchVisitor) VisitTerm(ctx *parser.TermContext) interface{} {
	return v.VisitChildren(ctx)
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitGeneric_search_term(ctx *parser.Generic_search_termContext) interface{} {
	node, ok := v.VisitChildren(ctx).(*SearchNode)
	if !ok {
		return nil
	}
	// Add the negation if needed as we come back up from the children.
	if ctx.NOT() != nil {
		node.Keyword.Invert()
		if len(node.Children) == 0 {
			node.Term.Operator.Invert()
		} else {
			// The grammar currently does not allow negation on the combined search term so we do not need to
			// invert beyond the first level of children.
			for idx := range node.Children {
				if node.Children[idx].Term != nil {
					node.Children[idx].Term.Operator.Invert()
				}
			}
		}
	}

	return node
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitSearch_criteria(ctx *parser.Search_criteriaContext) interface{} {
	// Handle the default ANY_VALUE case.
	// This is needed for the feature name that does not have the prefix.
	// Even though createNameNode will return nil if node is nil, it will add an error.
	// So we proactively check for node.
	if node := ctx.ANY_VALUE(); node != nil {
		return v.createNameNode(node)
	}

	return v.VisitChildren(ctx)
}

func (v *FeaturesSearchVisitor) VisitOperator(ctx *parser.OperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

/*
Implements ErrorListener.
Similar to https://github.com/google/mangle/blob/28db3310648ee110b108523b3df943ce22b61e2a/parse/parse.go#L679
*/

// SyntaxError is called by ANTLR generated code when a syntax error is encountered.
func (v *FeaturesSearchVisitor) SyntaxError(_ antlr.Recognizer,
	_ any, line, column int, msg string, _ antlr.RecognitionException) {
	v.addError(fmt.Errorf("msg: %s line: %d column: %d", msg, line, column))
}

// ReportAmbiguity implements error listener interface.
func (v *FeaturesSearchVisitor) ReportAmbiguity(_ antlr.Parser,
	_ *antlr.DFA, _, _ int, _ bool, _ *antlr.BitSet,
	_ *antlr.ATNConfigSet) {
	// Intentional
}

// ReportAttemptingFullContext implements error listener interface.
func (v *FeaturesSearchVisitor) ReportAttemptingFullContext(
	_ antlr.Parser, _ *antlr.DFA, _, _ int,
	_ *antlr.BitSet, _ *antlr.ATNConfigSet) {
	// Intentional
}

// ReportContextSensitivity  implements error listener interface.
func (v *FeaturesSearchVisitor) ReportContextSensitivity(
	_ antlr.Parser, _ *antlr.DFA, _, _, _ int,
	_ *antlr.ATNConfigSet) {
	// Intentional
}
