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
	"reflect"
	"strings"
	"testing"
)

func TestParseQuery(t *testing.T) {
	testCases := []struct {
		InputQuery   string
		ExpectedTree *SearchNode
	}{
		{
			InputQuery: "available_on:chrome",
			ExpectedTree: &SearchNode{
				Operator: OperatorRoot,
				Term:     nil,
				Children: []*SearchNode{
					{
						Term: &SearchTerm{
							Identifier: IdentifierAvailableOn,
							Value:      "chrome",
						},
						Children: nil,
						Operator: OperatorNone,
					},
				},
			},
		},
		{
			InputQuery: "available_on:chrome AND baseline_status:high",
			ExpectedTree: &SearchNode{
				Operator: OperatorRoot,
				Term:     nil,
				Children: []*SearchNode{
					{
						Operator: OperatorAND,
						Term:     nil,
						Children: []*SearchNode{
							{
								Children: nil,
								Term: &SearchTerm{
									Identifier: IdentifierAvailableOn,
									Value:      "chrome",
								},
								Operator: OperatorNone,
							},
							{
								Children: nil,
								Term: &SearchTerm{
									Identifier: IdentifierBaselineStatus,
									Value:      "high",
								},
								Operator: OperatorNone,
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "available_on:chrome baseline_status:high",
			ExpectedTree: &SearchNode{
				Operator: OperatorRoot,
				Term:     nil,
				Children: []*SearchNode{
					{
						Operator: OperatorAND,
						Term:     nil,
						Children: []*SearchNode{
							{
								Children: nil,
								Term: &SearchTerm{
									Identifier: IdentifierAvailableOn,
									Value:      "chrome",
								},
								Operator: OperatorNone,
							},
							{
								Children: nil,
								Term: &SearchTerm{
									Identifier: IdentifierBaselineStatus,
									Value:      "high",
								},
								Operator: OperatorNone,
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "available_on:chrome AND baseline_status:high OR name:grid",
			ExpectedTree: &SearchNode{
				Operator: OperatorRoot,
				Term:     nil,
				Children: []*SearchNode{
					{
						Operator: OperatorOR,
						Term:     nil,
						Children: []*SearchNode{
							{
								Operator: OperatorAND,
								Term:     nil,
								Children: []*SearchNode{
									{
										Children: nil,
										Term: &SearchTerm{
											Identifier: IdentifierAvailableOn,
											Value:      "chrome",
										},
										Operator: OperatorNone,
									},
									{
										Children: nil,
										Term: &SearchTerm{
											Identifier: IdentifierBaselineStatus,
											Value:      "high",
										},
										Operator: OperatorNone,
									},
								},
							},
							{
								Term: &SearchTerm{
									Identifier: IdentifierName,
									Value:      "grid",
								},
								Operator: OperatorNone,
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "(available_on:chrome AND baseline_status:high) OR name:grid",
			ExpectedTree: &SearchNode{
				Operator: OperatorRoot,
				Term:     nil,
				Children: []*SearchNode{
					{
						Operator: OperatorOR,
						Term:     nil,
						Children: []*SearchNode{
							{
								Operator: OperatorAND,
								Children: []*SearchNode{
									{
										Term: &SearchTerm{
											Identifier: IdentifierAvailableOn,
											Value:      "chrome",
										},
										Operator: OperatorNone,
									},
									{
										Term: &SearchTerm{
											Identifier: IdentifierBaselineStatus,
											Value:      "high",
										},
										Operator: OperatorNone,
									},
								},
							},
							{
								Term: &SearchTerm{
									Identifier: IdentifierName,
									Value:      "grid",
								},
								Operator: OperatorNone,
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "(available_on:chrome AND baseline_status:high OR name:avif) OR name:grid",
			ExpectedTree: &SearchNode{
				Operator: OperatorRoot,
				Term:     nil,
				Children: []*SearchNode{
					{
						Operator: OperatorOR,
						Term:     nil,
						Children: []*SearchNode{
							{
								Operator: OperatorOR,
								Term:     nil,
								Children: []*SearchNode{
									{
										Operator: OperatorAND,
										Term:     nil,
										Children: []*SearchNode{
											{
												Operator: OperatorNone,
												Children: nil,
												Term:     &SearchTerm{Identifier: IdentifierAvailableOn, Value: "chrome"},
											},
											{
												Operator: OperatorNone,
												Children: nil,
												Term:     &SearchTerm{Identifier: IdentifierBaselineStatus, Value: "high"},
											},
										},
									},
									{
										Operator: OperatorNone,
										Children: nil,
										Term:     &SearchTerm{Identifier: IdentifierName, Value: "avif"},
									},
								},
							},
							{
								Operator: OperatorNone,
								Children: nil,
								Term:     &SearchTerm{Identifier: IdentifierName, Value: "grid"},
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "available_on:chrome (baseline_status:high OR name:avif) OR name:grid",
			ExpectedTree: &SearchNode{
				Operator: OperatorRoot,
				Term:     nil,
				Children: []*SearchNode{
					{
						Operator: OperatorOR,
						Term:     nil,
						Children: []*SearchNode{
							{
								Operator: OperatorAND,
								Term:     nil,
								Children: []*SearchNode{
									{
										Operator: OperatorNone,
										Children: nil,
										Term:     &SearchTerm{Identifier: IdentifierAvailableOn, Value: "chrome"},
									},
									{
										Operator: OperatorOR,
										Term:     nil,
										Children: []*SearchNode{
											{
												Operator: OperatorNone,
												Children: nil,
												Term:     &SearchTerm{Identifier: IdentifierBaselineStatus, Value: "high"},
											},
											{
												Operator: OperatorNone,
												Children: nil,
												Term:     &SearchTerm{Identifier: IdentifierName, Value: "avif"},
											},
										},
									},
								},
							},
							{
								Operator: OperatorNone,
								Children: nil,
								Term:     &SearchTerm{Identifier: IdentifierName, Value: "grid"},
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "name:grid",
			ExpectedTree: &SearchNode{
				Operator: OperatorRoot,
				Term:     nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "grid",
						},
						Operator: OperatorNone,
					},
				},
			},
		},
		{
			// Should remove the quotes
			InputQuery: `name:"CSS Grid"`,
			ExpectedTree: &SearchNode{
				Operator: OperatorRoot,
				Term:     nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "CSS Grid",
						},
						Operator: OperatorNone,
					},
				},
			},
		},
		{
			InputQuery: "grid",
			ExpectedTree: &SearchNode{
				Operator: OperatorRoot,
				Term:     nil,
				Children: []*SearchNode{
					{
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "grid",
						},
						Operator: OperatorNone,
						Children: nil,
					},
				},
			},
		},
		{
			InputQuery: `"CSS Grid"`,
			ExpectedTree: &SearchNode{
				Operator: OperatorRoot,
				Term:     nil,
				Children: []*SearchNode{
					{
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "CSS Grid",
						},
						Operator: OperatorNone,
						Children: nil,
					},
				},
			},
		},
		{
			InputQuery: "-available_on:chrome OR baseline_status:high",
			ExpectedTree: &SearchNode{
				Operator: OperatorRoot,
				Term:     nil,
				Children: []*SearchNode{
					{
						Operator: OperatorOR,
						Term:     nil,
						Children: []*SearchNode{
							{
								Operator: OperatorNegation,
								Term: &SearchTerm{
									Identifier: IdentifierAvailableOn,
									Value:      "chrome",
								},
								Children: nil,
							},
							{
								Operator: OperatorNone,
								Term: &SearchTerm{
									Identifier: IdentifierBaselineStatus,
									Value:      "high",
								},
								Children: nil,
							},
						},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		parser := FeaturesSearchQueryParser{}
		resultTree, err := parser.Parse(testCase.InputQuery)

		if !reflect.DeepEqual(resultTree, testCase.ExpectedTree) {
			t.Errorf("Query: %s\nExpected:\n%+v\nGot:\n%+v",
				testCase.InputQuery, testCase.ExpectedTree.PrettyPrint(), resultTree.PrettyPrint())
		}
		if !errors.Is(err, nil) {
			t.Errorf("expected nil error got: %s", err.Error())
		}
	}
}

func (node *SearchNode) PrettyPrint() string {
	var builder strings.Builder
	printNode(&builder, node, "")

	return builder.String()
}

func printNode(builder *strings.Builder, node *SearchNode, indent string) {
	operatorStr := ""
	if node.Operator != OperatorNone {
		operatorStr = fmt.Sprintf(" (%s)", node.Operator)
	}

	var termStr string
	if node.Term != nil {
		termStr = fmt.Sprintf("%s:%s", node.Term.Identifier, node.Term.Value)
	}

	builder.WriteString(indent + termStr + operatorStr + "\n")

	newIndent := indent + "  "
	for _, child := range node.Children {
		printNode(builder, child, newIndent)
	}
}

func TestParseQueryBadInput(t *testing.T) {
	testCases := []struct {
		input string
	}{
		{
			input: `badterm:foo`,
		},
		{
			input: ``,
		},
		{
			input: `""`,
		},
		{
			input: "badterm:",
		},
		// unbalanced parenthesis.
		{
			input: "(name:grid ()",
		},
	}
	for _, tc := range testCases {
		parser := FeaturesSearchQueryParser{}
		resultTree, err := parser.Parse(tc.input)
		if resultTree != nil {
			t.Error("expected nil node")
		}
		if errors.Is(err, nil) {
			t.Error("expected non nil error")
		}

	}
}
