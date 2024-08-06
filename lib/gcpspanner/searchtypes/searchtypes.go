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
	KeywordAND  SearchKeyword = "AND"
	KeywordOR   SearchKeyword = "OR"
	KeywordRoot SearchKeyword = "ROOT"
	// Placeholder for nil.
	KeywordNone SearchKeyword = "NONE"
)

func (k *SearchKeyword) Invert() {
	switch *k {
	case KeywordAND:
		*k = KeywordOR
	case KeywordOR:
		*k = KeywordAND
	case KeywordRoot, KeywordNone:
		// Do nothing
		return
	}
}

type SearchOperator string

const (
	OperatorGtEq SearchOperator = "GT_EQ"
	OperatorGt   SearchOperator = "GT"
	OperatorLtEq SearchOperator = "LT_EQ"
	OperatorLt   SearchOperator = "LT"
	OperatorEq   SearchOperator = "EQ"
	OperatorNeq  SearchOperator = "NEQ"
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
	}
}

type SearchNode struct {
	Keyword  SearchKeyword
	Term     *SearchTerm
	Children []*SearchNode
}

func (n SearchNode) IsKeyword() bool {
	return n.Keyword == KeywordAND || n.Keyword == KeywordOR
}

type SearchTerm struct {
	Identifier SearchIdentifier
	Operator   SearchOperator
	Value      string
}

type SearchIdentifier string

const (
	IdentifierAvailableDate  SearchIdentifier = "available_date"
	IdentifierAvailableOn    SearchIdentifier = "available_on"
	IdentifierBaselineDate   SearchIdentifier = "baseline_date"
	IdentifierBaselineStatus SearchIdentifier = "baseline_status"
	IdentifierName           SearchIdentifier = "name"
)
