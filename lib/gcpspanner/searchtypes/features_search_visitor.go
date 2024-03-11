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

func getOperatorType(text string) SearchOperator {
	switch text {
	case "AND":
		return OperatorAND
	case "OR":
		return OperatorOR
	}

	return OperatorNone
}

func (v *FeaturesSearchVisitor) addError(err error) {
	if v.err == nil {
		v.err = err

		return
	}
	v.err = errors.Join(v.err, err)
}

func (v *FeaturesSearchVisitor) handleOperator(current *SearchNode, operatorCtx *parser.OperatorContext) *SearchNode {
	operator := getOperatorType(operatorCtx.GetText())

	newNode := &SearchNode{
		Operator: operator,
		Term:     nil,
		Children: nil,
	}

	if current != nil {
		// Detect implicit AND case and prioritize flat groupings.
		// This logic handles the optional chaining structure in the grammar, where
		// terms/groups without explicit operators are implicitly joined using AND.
		// The implicit AND nodes themselves are created primarily within the 'chainWithImplicitAND' function.
		if current.Operator == OperatorAND && len(current.Children) == 1 {
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
	} else if !current.IsOperator() {
		// Case: Chaining onto a term node, create an implicit AND.
		return &SearchNode{Term: nil, Operator: OperatorAND, Children: []*SearchNode{current, newNode}}
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
			Operator: OperatorAND,
			Children: []*SearchNode{rootNode, node},
		}
	}

	return rootNode
}

func (v *FeaturesSearchVisitor) createNameNode(name string) *SearchNode {
	name = strings.Trim(name, `"`)

	return &SearchNode{
		Operator: OperatorNone,
		Children: nil,
		Term: &SearchTerm{
			Identifier: IdentifierName,
			Value:      name,
		},
	}
}

/*
The below section implements the generated BaseFeatureSearchVisitor methods.
*/

func (v *FeaturesSearchVisitor) VisitQuery(ctx *parser.QueryContext) interface{} {
	// Create root node.
	root := &SearchNode{Operator: OperatorRoot, Term: nil, Children: nil}
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
	browserName := ctx.BROWSER_NAME().GetText()

	return &SearchNode{
		Operator: OperatorNone,
		Term: &SearchTerm{
			Identifier: IdentifierAvailableOn,
			Value:      browserName,
		},
		Children: nil,
	}
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitBaseline_status_term(ctx *parser.Baseline_status_termContext) interface{} {
	baselineStatus := ctx.BASELINE_STATUS().GetText()

	return &SearchNode{
		Operator: OperatorNone,
		Term: &SearchTerm{
			Identifier: IdentifierBaselineStatus,
			Value:      baselineStatus,
		},
		Children: nil,
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

// Similar to https://github.com/google/mangle/blob/28db3310648ee110b108523b3df943ce22b61e2a/parse/parse.go#L154
func (v *FeaturesSearchVisitor) Visit(tree antlr.ParseTree) any {
	switch tree := tree.(type) {
	case *parser.Available_on_termContext:
		return v.VisitAvailable_on_term(tree)
	case *parser.Baseline_status_termContext:
		return v.VisitBaseline_status_term(tree)
	case *parser.Combined_search_criteriaContext:
		return v.VisitCombined_search_criteria(tree)
	case *parser.Generic_search_termContext:
		return v.VisitGeneric_search_term(tree)
	case *parser.Missing_in_one_ofContext:
		return v.VisitMissing_in_one_of(tree)
	case *parser.Name_termContext:
		return v.VisitName_term(tree)
	case *parser.OperatorContext:
		return v.VisitOperator(tree)
	case *parser.QueryContext:
		return v.VisitQuery(tree)
	case *parser.Search_criteriaContext:
		return v.VisitSearch_criteria(tree)
	case *parser.TermContext:
		return v.VisitTerm(tree)
	}

	return tree.Accept(v)
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

		default:
			// Future cases will call into the default.
		}
	}

	root = current

	return root
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitName_term(ctx *parser.Name_termContext) interface{} {
	return v.createNameNode(ctx.ANY_VALUE().GetText())
}

func (v *FeaturesSearchVisitor) VisitTerm(ctx *parser.TermContext) interface{} {
	return v.VisitChildren(ctx)
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitGeneric_search_term(ctx *parser.Generic_search_termContext) interface{} {
	// Should only be a single item.
	// Add the negation if ndeeded.
	node := v.VisitChildren(ctx).(*SearchNode)
	if ctx.NOT() != nil {
		node.Operator = OperatorNegation
	}

	return node
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitSearch_criteria(ctx *parser.Search_criteriaContext) interface{} {
	// Handle the default ANY_VALUE case.
	// This is needed for the feature name that does not have the prefix.
	if node := ctx.ANY_VALUE(); node != nil {
		return v.createNameNode(node.GetText())
	}

	return v.VisitChildren(ctx)
}

// nolint: revive // Method signature is generated.
func (v *FeaturesSearchVisitor) VisitMissing_in_one_of(ctx *parser.Missing_in_one_ofContext) interface{} {
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
