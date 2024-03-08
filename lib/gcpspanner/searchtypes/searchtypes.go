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

type SearchOperator string

const (
	OperatorNone     SearchOperator = "NONE"
	OperatorAND      SearchOperator = "AND"
	OperatorOR       SearchOperator = "OR"
	OperatorNegation SearchOperator = "NEGATION"
	OperatorRoot     SearchOperator = "ROOT"
)

type SearchNode struct {
	Operator SearchOperator
	Term     *SearchTerm
	Children []*SearchNode
}

func (n SearchNode) IsOperator() bool {
	return n.Operator == OperatorAND || n.Operator == OperatorOR
}

type SearchTerm struct {
	Identifier SearchIdentifier
	Value      string
}

type SearchIdentifier string

const (
	IdentifierAvailableOn    SearchIdentifier = "available_on"
	IdentifierBaselineStatus SearchIdentifier = "baseline_status"
	IdentifierName           SearchIdentifier = "name"
)
