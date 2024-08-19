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
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Term: &SearchTerm{
							Identifier: IdentifierAvailableOn,
							Value:      "chrome",
							Operator:   OperatorEq,
						},
						Children: nil,
						Keyword:  KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: "available_on:Chrome",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Term: &SearchTerm{
							Identifier: IdentifierAvailableOn,
							Value:      "chrome",
							Operator:   OperatorEq,
						},
						Children: nil,
						Keyword:  KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: "available_on:CHROME",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Term: &SearchTerm{
							Identifier: IdentifierAvailableOn,
							Value:      "chrome",
							Operator:   OperatorEq,
						},
						Children: nil,
						Keyword:  KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: "-available_on:chrome",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Term: &SearchTerm{
							Identifier: IdentifierAvailableOn,
							Value:      "chrome",
							Operator:   OperatorNeq,
						},
						Children: nil,
						Keyword:  KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: "available_date:chrome:2000-01-01..2000-12-31",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Keyword: KeywordAND,
						Term:    nil,
						Children: []*SearchNode{
							{ // startDateNode
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
											Value:      "chrome",
											Operator:   OperatorEq,
										},
										Children: nil,
									},
									{
										Keyword: KeywordNone,
										Term: &SearchTerm{
											Identifier: IdentifierAvailableDate,
											Value:      "2000-01-01",
											Operator:   OperatorGtEq,
										},
										Children: nil,
									},
								},
							},
							{ // endDateNode
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
											Value:      "chrome",
											Operator:   OperatorEq,
										},
										Children: nil,
									},
									{
										Keyword: KeywordNone,
										Term: &SearchTerm{
											Identifier: IdentifierAvailableDate,
											Value:      "2000-12-31",
											Operator:   OperatorLtEq,
										},
										Children: nil,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "available_date:chrome:2000-01-01..2000-12-31 OR available_date:firefox:2000-01-01..2000-12-31",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Keyword: KeywordOR,
						Term:    nil,
						Children: []*SearchNode{
							{
								Keyword: KeywordAND,
								Term:    nil,
								Children: []*SearchNode{
									{ // startDateNode
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
													Value:      "chrome",
													Operator:   OperatorEq,
												},
												Children: nil,
											},
											{
												Keyword: KeywordNone,
												Term: &SearchTerm{
													Identifier: IdentifierAvailableDate,
													Value:      "2000-01-01",
													Operator:   OperatorGtEq,
												},
												Children: nil,
											},
										},
									},
									{ // endDateNode
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
													Value:      "chrome",
													Operator:   OperatorEq,
												},
												Children: nil,
											},
											{
												Keyword: KeywordNone,
												Term: &SearchTerm{
													Identifier: IdentifierAvailableDate,
													Value:      "2000-12-31",
													Operator:   OperatorLtEq,
												},
												Children: nil,
											},
										},
									},
								},
							},
							{
								Keyword: KeywordAND,
								Term:    nil,
								Children: []*SearchNode{
									{ // startDateNode
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
													Value:      "firefox",
													Operator:   OperatorEq,
												},
												Children: nil,
											},
											{
												Keyword: KeywordNone,
												Term: &SearchTerm{
													Identifier: IdentifierAvailableDate,
													Value:      "2000-01-01",
													Operator:   OperatorGtEq,
												},
												Children: nil,
											},
										},
									},
									{ // endDateNode
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
													Value:      "firefox",
													Operator:   OperatorEq,
												},
												Children: nil,
											},
											{
												Keyword: KeywordNone,
												Term: &SearchTerm{
													Identifier: IdentifierAvailableDate,
													Value:      "2000-12-31",
													Operator:   OperatorLtEq,
												},
												Children: nil,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "available_on:chrome AND baseline_status:widely",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Keyword: KeywordAND,
						Term:    nil,
						Children: []*SearchNode{
							{
								Children: nil,
								Term: &SearchTerm{
									Identifier: IdentifierAvailableOn,
									Value:      "chrome",
									Operator:   OperatorEq,
								},
								Keyword: KeywordNone,
							},
							{
								Children: nil,
								Term: &SearchTerm{
									Identifier: IdentifierBaselineStatus,
									Value:      "widely",
									Operator:   OperatorEq,
								},
								Keyword: KeywordNone,
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "available_on:chrome baseline_status:widely",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Keyword: KeywordAND,
						Term:    nil,
						Children: []*SearchNode{
							{
								Children: nil,
								Term: &SearchTerm{
									Identifier: IdentifierAvailableOn,
									Value:      "chrome",
									Operator:   OperatorEq,
								},
								Keyword: KeywordNone,
							},
							{
								Children: nil,
								Term: &SearchTerm{
									Identifier: IdentifierBaselineStatus,
									Value:      "widely",
									Operator:   OperatorEq,
								},
								Keyword: KeywordNone,
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "available_on:chrome AND baseline_status:widely OR name:grid",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Keyword: KeywordOR,
						Term:    nil,
						Children: []*SearchNode{
							{
								Keyword: KeywordAND,
								Term:    nil,
								Children: []*SearchNode{
									{
										Children: nil,
										Term: &SearchTerm{
											Identifier: IdentifierAvailableOn,
											Value:      "chrome",
											Operator:   OperatorEq,
										},
										Keyword: KeywordNone,
									},
									{
										Children: nil,
										Term: &SearchTerm{
											Identifier: IdentifierBaselineStatus,
											Value:      "widely",
											Operator:   OperatorEq,
										},
										Keyword: KeywordNone,
									},
								},
							},
							{
								Term: &SearchTerm{
									Identifier: IdentifierName,
									Value:      "grid",
									Operator:   OperatorEq,
								},
								Keyword: KeywordNone,
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "(available_on:chrome AND baseline_status:widely) OR name:grid",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Keyword: KeywordOR,
						Term:    nil,
						Children: []*SearchNode{
							{
								Keyword: KeywordAND,
								Children: []*SearchNode{
									{
										Term: &SearchTerm{
											Identifier: IdentifierAvailableOn,
											Value:      "chrome",
											Operator:   OperatorEq,
										},
										Keyword: KeywordNone,
									},
									{
										Term: &SearchTerm{
											Identifier: IdentifierBaselineStatus,
											Value:      "widely",
											Operator:   OperatorEq,
										},
										Keyword: KeywordNone,
									},
								},
							},
							{
								Term: &SearchTerm{
									Identifier: IdentifierName,
									Value:      "grid",
									Operator:   OperatorEq,
								},
								Keyword: KeywordNone,
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "(available_on:chrome AND baseline_status:widely OR name:avif) OR name:grid",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Keyword: KeywordOR,
						Term:    nil,
						Children: []*SearchNode{
							{
								Keyword: KeywordOR,
								Term:    nil,
								Children: []*SearchNode{
									{
										Keyword: KeywordAND,
										Term:    nil,
										Children: []*SearchNode{
											{
												Keyword:  KeywordNone,
												Children: nil,
												Term: &SearchTerm{
													Identifier: IdentifierAvailableOn,
													Value:      "chrome",
													Operator:   OperatorEq,
												},
											},
											{
												Keyword:  KeywordNone,
												Children: nil,
												Term: &SearchTerm{
													Identifier: IdentifierBaselineStatus,
													Value:      "widely",
													Operator:   OperatorEq,
												},
											},
										},
									},
									{
										Keyword:  KeywordNone,
										Children: nil,
										Term: &SearchTerm{
											Identifier: IdentifierName,
											Value:      "avif",
											Operator:   OperatorEq,
										},
									},
								},
							},
							{
								Keyword:  KeywordNone,
								Children: nil,
								Term:     &SearchTerm{Identifier: IdentifierName, Value: "grid", Operator: OperatorEq},
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "available_on:chrome (baseline_status:widely OR name:avif) OR name:grid",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Keyword: KeywordOR,
						Term:    nil,
						Children: []*SearchNode{
							{
								Keyword: KeywordAND,
								Term:    nil,
								Children: []*SearchNode{
									{
										Keyword:  KeywordNone,
										Children: nil,
										Term: &SearchTerm{
											Identifier: IdentifierAvailableOn,
											Value:      "chrome",
											Operator:   OperatorEq,
										},
									},
									{
										Keyword: KeywordOR,
										Term:    nil,
										Children: []*SearchNode{
											{
												Keyword:  KeywordNone,
												Children: nil,
												Term: &SearchTerm{
													Identifier: IdentifierBaselineStatus,
													Value:      "widely",
													Operator:   OperatorEq,
												},
											},
											{
												Keyword:  KeywordNone,
												Children: nil,
												Term: &SearchTerm{
													Identifier: IdentifierName,
													Value:      "avif",
													Operator:   OperatorEq,
												},
											},
										},
									},
								},
							},
							{
								Keyword:  KeywordNone,
								Children: nil,
								Term:     &SearchTerm{Identifier: IdentifierName, Value: "grid", Operator: OperatorEq},
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "name:grid",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "grid",
							Operator:   OperatorEq,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: "group:css",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierGroup,
							Value:      "css",
							Operator:   OperatorEq,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: `baseline_date:2000-01-01..2000-12-31 OR group:css`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Keyword: KeywordOR,
						Term:    nil,
						Children: []*SearchNode{
							{
								Keyword: KeywordAND,
								Term:    nil,
								Children: []*SearchNode{
									{
										Keyword: KeywordNone,
										Term: &SearchTerm{
											Identifier: IdentifierBaselineDate,
											Value:      "2000-01-01",
											Operator:   OperatorGtEq,
										},
										Children: nil,
									},
									{
										Keyword: KeywordNone,
										Term: &SearchTerm{
											Identifier: IdentifierBaselineDate,
											Value:      "2000-12-31",
											Operator:   OperatorLtEq,
										},
										Children: nil,
									},
								},
							},
							{
								Keyword:  KeywordNone,
								Children: nil,
								Term: &SearchTerm{
									Identifier: IdentifierGroup,
									Value:      "css",
									Operator:   OperatorEq,
								},
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "snapshot:ecmascript-5",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierSnapshot,
							Value:      "ecmascript-5",
							Operator:   OperatorEq,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: `baseline_date:2000-01-01..2000-12-31 OR snapshot:ecmascript-5`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Keyword: KeywordOR,
						Term:    nil,
						Children: []*SearchNode{
							{
								Keyword: KeywordAND,
								Term:    nil,
								Children: []*SearchNode{
									{
										Keyword: KeywordNone,
										Term: &SearchTerm{
											Identifier: IdentifierBaselineDate,
											Value:      "2000-01-01",
											Operator:   OperatorGtEq,
										},
										Children: nil,
									},
									{
										Keyword: KeywordNone,
										Term: &SearchTerm{
											Identifier: IdentifierBaselineDate,
											Value:      "2000-12-31",
											Operator:   OperatorLtEq,
										},
										Children: nil,
									},
								},
							},
							{
								Keyword:  KeywordNone,
								Children: nil,
								Term: &SearchTerm{
									Identifier: IdentifierSnapshot,
									Value:      "ecmascript-5",
									Operator:   OperatorEq,
								},
							},
						},
					},
				},
			},
		},
		{
			// Should remove the quotes
			InputQuery: `name:"CSS Grid"`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "CSS Grid",
							Operator:   OperatorEq,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: "grid",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "grid",
							Operator:   OperatorEq,
						},
						Keyword:  KeywordNone,
						Children: nil,
					},
				},
			},
		},
		{
			InputQuery: `"CSS Grid"`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "CSS Grid",
							Operator:   OperatorEq,
						},
						Keyword:  KeywordNone,
						Children: nil,
					},
				},
			},
		},
		{
			InputQuery: "baseline_date:2000-01-01..2000-12-31",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Keyword: KeywordAND,
						Term:    nil,
						Children: []*SearchNode{
							{
								Keyword: KeywordNone,
								Term: &SearchTerm{
									Identifier: IdentifierBaselineDate,
									Value:      "2000-01-01",
									Operator:   OperatorGtEq,
								},
								Children: nil,
							},
							{
								Keyword: KeywordNone,
								Term: &SearchTerm{
									Identifier: IdentifierBaselineDate,
									Value:      "2000-12-31",
									Operator:   OperatorLtEq,
								},
								Children: nil,
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "-baseline_date:2000-01-01..2000-12-31",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Keyword: KeywordOR,
						Term:    nil,
						Children: []*SearchNode{
							{
								Keyword: KeywordNone,
								Term: &SearchTerm{
									Identifier: IdentifierBaselineDate,
									Value:      "2000-01-01",
									Operator:   OperatorLt,
								},
								Children: nil,
							},
							{
								Keyword: KeywordNone,
								Term: &SearchTerm{
									Identifier: IdentifierBaselineDate,
									Value:      "2000-12-31",
									Operator:   OperatorGt,
								},
								Children: nil,
							},
						},
					},
				},
			},
		},
		{
			InputQuery: `baseline_date:2000-01-01..2000-12-31 OR "CSS Grid"`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Keyword: KeywordOR,
						Term:    nil,
						Children: []*SearchNode{
							{
								Keyword: KeywordAND,
								Term:    nil,
								Children: []*SearchNode{
									{
										Keyword: KeywordNone,
										Term: &SearchTerm{
											Identifier: IdentifierBaselineDate,
											Value:      "2000-01-01",
											Operator:   OperatorGtEq,
										},
										Children: nil,
									},
									{
										Keyword: KeywordNone,
										Term: &SearchTerm{
											Identifier: IdentifierBaselineDate,
											Value:      "2000-12-31",
											Operator:   OperatorLtEq,
										},
										Children: nil,
									},
								},
							},
							{
								Keyword:  KeywordNone,
								Children: nil,
								Term: &SearchTerm{
									Identifier: IdentifierName,
									Value:      "CSS Grid",
									Operator:   OperatorEq,
								},
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "-available_on:chrome OR baseline_status:widely",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Keyword: KeywordOR,
						Term:    nil,
						Children: []*SearchNode{
							{
								Keyword: KeywordNone,
								Term: &SearchTerm{
									Identifier: IdentifierAvailableOn,
									Value:      "chrome",
									Operator:   OperatorNeq,
								},
								Children: nil,
							},
							{
								Keyword: KeywordNone,
								Term: &SearchTerm{
									Identifier: IdentifierBaselineStatus,
									Value:      "widely",
									Operator:   OperatorEq,
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
	if node == nil {
		builder.WriteString("NIL NODE")

		return
	}
	operatorStr := ""
	if node.Keyword != KeywordNone {
		operatorStr = fmt.Sprintf(" (%s)", node.Keyword)
	}

	var termStr string
	if node.Term != nil {
		termStr = fmt.Sprintf("%s:%s (%s)", node.Term.Identifier, node.Term.Value, node.Term.Operator)
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
		// Old baseline_status phrases will parse with error now.
		{
			input: "baseline_status:high",
		},
		{
			input: "baseline_status:low",
		},
		{
			input: "baseline_status:none",
		},
		{
			input: "available_on:chrome,edge",
		},
		{
			input: "available_date:chrome",
		},
		{
			input: "available_date:2000-01-01..2000-12-31",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			parser := FeaturesSearchQueryParser{}
			resultTree, err := parser.Parse(tc.input)
			if resultTree != nil {
				t.Error("expected nil node")
			}
			if errors.Is(err, nil) {
				t.Error("expected non nil error")
			}
		})
	}
}
