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

type SearchKeyword string

const (
	KeywordAND    SearchKeyword = "AND"
	KeywordOR     SearchKeyword = "OR"
	KeywordRoot   SearchKeyword = "ROOT"
	KeywordParens SearchKeyword = "PARENS"
	// Placeholder for nil.
	KeywordNone SearchKeyword = "NONE"
)

func (k *SearchKeyword) Invert() {
	switch *k {
	case KeywordAND:
		*k = KeywordOR
	case KeywordOR:
		*k = KeywordAND
	case KeywordRoot, KeywordNone, KeywordParens:
		// Do nothing
		return
	}
}

type SearchOperator string

const (
	OperatorGtEq    SearchOperator = "GT_EQ"
	OperatorGt      SearchOperator = "GT"
	OperatorLtEq    SearchOperator = "LT_EQ"
	OperatorLt      SearchOperator = "LT"
	OperatorEq      SearchOperator = "EQ"
	OperatorLike    SearchOperator = "LIKE"
	OperatorNotLike SearchOperator = "NOT_LIKE"
	OperatorNeq     SearchOperator = "NEQ"
	OperatorNone    SearchOperator = "NONE"
)

func (o *SearchOperator) Invert() {
	switch *o {
	case OperatorEq:
		*o = OperatorNeq
	case OperatorGt:
		*o = OperatorLtEq
	case OperatorGtEq:
		*o = OperatorLt
	case OperatorLt:
		*o = OperatorGtEq
	case OperatorLtEq:
		*o = OperatorGt
	case OperatorNeq:
		*o = OperatorEq
	case OperatorLike:
		*o = OperatorNotLike
	case OperatorNotLike:
		*o = OperatorLike
	case OperatorNone:
		break
	}
}

type SearchNode struct {
	Keyword  SearchKeyword
	Term     *SearchTerm
	Children []*SearchNode
}

// EmptySearchNode returns the representation of an empty search query AST.
// Currently, an empty query is represented by nil.
func EmptySearchNode() *SearchNode {
	return nil
}

// IsEmptySearchNode returns true if the node represents an empty search query.
func IsEmptySearchNode(node *SearchNode) bool {
	return node == nil
}

func (n SearchNode) IsKeyword() bool {
	return n.Keyword == KeywordAND || n.Keyword == KeywordOR
}

// CountNodes returns the total number of nodes in the SearchNode AST tree.
// Used for structural complexity validation to protect Cloud Spanner against query amplification attacks.
func CountNodes(node *SearchNode) int {
	if node == nil {
		return 0
	}
	count := 1
	for _, child := range node.Children {
		count += CountNodes(child)
	}

	return count
}

// EqualSearchNode checks if two SearchNode trees are structurally identical.
func EqualSearchNode(a, b *SearchNode) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Keyword != b.Keyword {
		return false
	}
	if (a.Term == nil) != (b.Term == nil) {
		return false
	}
	if a.Term != nil {
		if a.Term.Identifier != b.Term.Identifier ||
			a.Term.Operator != b.Term.Operator ||
			a.Term.Value != b.Term.Value {
			return false
		}
	}
	if len(a.Children) != len(b.Children) {
		return false
	}
	for i := range a.Children {
		if !EqualSearchNode(a.Children[i], b.Children[i]) {
			return false
		}
	}

	return true
}

// Deduplicate recursively simplifies an AST by removing duplicate child nodes under AND and OR keywords.
// By the Idempotent Law of Boolean Algebra (A AND A = A, A OR A = A), deduplicating identical terms
// produces a 100% logically equivalent query that returns the exact same set of web features,
// while preventing redundant subquery generation and unnecessary Spanner CPU execution overhead.
func Deduplicate(node *SearchNode) *SearchNode {
	if node == nil {
		return nil
	}

	newChildren := make([]*SearchNode, 0, len(node.Children))
	for _, child := range node.Children {
		dedupChild := Deduplicate(child)
		if dedupChild == nil {
			continue
		}

		if node.IsKeyword() {
			duplicate := false
			for _, existing := range newChildren {
				if EqualSearchNode(existing, dedupChild) {
					duplicate = true

					break
				}
			}
			if duplicate {
				continue
			}
		}

		newChildren = append(newChildren, dedupChild)
	}

	if node.IsKeyword() && len(newChildren) == 1 {
		return newChildren[0]
	}

	return &SearchNode{
		Keyword:  node.Keyword,
		Term:     node.Term,
		Children: newChildren,
	}
}

type SearchTerm struct {
	Identifier SearchIdentifier
	Operator   SearchOperator
	Value      string
}

type SearchIdentifier string

const (
	IdentifierAvailableBrowserDate SearchIdentifier = "available_browser_date"
	IdentifierAvailableDate        SearchIdentifier = "available_date"
	IdentifierAvailableOn          SearchIdentifier = "available_on"
	IdentifierBaselineDate         SearchIdentifier = "baseline_date"
	IdentifierBaselineStatus       SearchIdentifier = "baseline_status"
	IdentifierName                 SearchIdentifier = "name"
	IdentifierDescription          SearchIdentifier = "description"
	IdentifierGroup                SearchIdentifier = "group"
	IdentifierSnapshot             SearchIdentifier = "snapshot"
	IdentifierID                   SearchIdentifier = "id"
	IdentifierSavedSearch          SearchIdentifier = "saved"
	IdentifierHotlist              SearchIdentifier = "hotlist"
)
