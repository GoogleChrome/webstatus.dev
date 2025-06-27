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
			InputQuery: "available_on:chrome_android",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Term: &SearchTerm{
							Identifier: IdentifierAvailableOn,
							Value:      "chrome_android",
							Operator:   OperatorEq,
						},
						Children: nil,
						Keyword:  KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: "-available_on:chrome_android",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Term: &SearchTerm{
							Identifier: IdentifierAvailableOn,
							Value:      "chrome_android",
							Operator:   OperatorNeq,
						},
						Children: nil,
						Keyword:  KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: "available_on:firefox_android",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Term: &SearchTerm{
							Identifier: IdentifierAvailableOn,
							Value:      "firefox_android",
							Operator:   OperatorEq,
						},
						Children: nil,
						Keyword:  KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: "-available_on:firefox_android",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Term: &SearchTerm{
							Identifier: IdentifierAvailableOn,
							Value:      "firefox_android",
							Operator:   OperatorNeq,
						},
						Children: nil,
						Keyword:  KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: "available_on:safari_ios",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Term: &SearchTerm{
							Identifier: IdentifierAvailableOn,
							Value:      "safari_ios",
							Operator:   OperatorEq,
						},
						Children: nil,
						Keyword:  KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: "-available_on:safari_ios",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Term: &SearchTerm{
							Identifier: IdentifierAvailableOn,
							Value:      "safari_ios",
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
									Operator:   OperatorLike,
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
								Keyword: KeywordParens,
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
								},
							},
							{
								Term: &SearchTerm{
									Identifier: IdentifierName,
									Value:      "grid",
									Operator:   OperatorLike,
								},
								Keyword: KeywordNone,
							},
						},
					},
				},
			},
		},
		{
			InputQuery: "(available_on:chrome AND (baseline_status:widely OR baseline_status:limited)) OR name:grid",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Keyword: KeywordOR,
						Term:    nil,
						Children: []*SearchNode{
							{
								Keyword: KeywordParens,
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
												Term:    nil,
												Keyword: KeywordParens,
												Children: []*SearchNode{
													{
														Keyword: KeywordOR,
														Term:    nil,
														Children: []*SearchNode{
															{
																Children: nil,
																Term: &SearchTerm{
																	Identifier: IdentifierBaselineStatus,
																	Value:      "widely",
																	Operator:   OperatorEq,
																},
																Keyword: KeywordNone,
															},
															{
																Children: nil,
																Term: &SearchTerm{
																	Identifier: IdentifierBaselineStatus,
																	Value:      "limited",
																	Operator:   OperatorEq,
																},
																Keyword: KeywordNone,
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
								Term: &SearchTerm{
									Identifier: IdentifierName,
									Value:      "grid",
									Operator:   OperatorLike,
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
								Keyword: KeywordParens,
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
													Operator:   OperatorLike,
												},
											},
										},
									},
								},
							},
							{
								Keyword:  KeywordNone,
								Children: nil,
								Term: &SearchTerm{
									Identifier: IdentifierName,
									Value:      "grid",
									Operator:   OperatorLike,
								},
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
										Keyword: KeywordParens,
										Term:    nil,
										Children: []*SearchNode{
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
															Operator:   OperatorLike,
														},
													},
												},
											},
										},
									},
								},
							},
							{
								Keyword:  KeywordNone,
								Children: nil,
								Term: &SearchTerm{
									Identifier: IdentifierName,
									Value:      "grid",
									Operator:   OperatorLike,
								},
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
							Operator:   OperatorLike,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: "-name:grid",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "grid",
							Operator:   OperatorNotLike,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: "desc:@container",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierDescription,
							Value:      "@container",
							Operator:   OperatorLike,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: "-desc:@container",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierDescription,
							Value:      "@container",
							Operator:   OperatorNotLike,
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
			InputQuery: "id:html",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierID,
							Value:      "html",
							Operator:   OperatorEq,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: "id:html OR id:css OR id:typescript OR id:javascript",
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Term:    nil,
						Keyword: KeywordOR,
						Children: []*SearchNode{
							{
								Term:    nil,
								Keyword: KeywordOR,
								Children: []*SearchNode{
									{
										Term:    nil,
										Keyword: KeywordOR,
										Children: []*SearchNode{
											{
												Keyword: KeywordNone,
												Term: &SearchTerm{
													Identifier: IdentifierID,
													Value:      "html",
													Operator:   OperatorEq,
												},
												Children: nil,
											},
											{
												Keyword: KeywordNone,
												Term: &SearchTerm{
													Identifier: IdentifierID,
													Value:      "css",
													Operator:   OperatorEq,
												},
												Children: nil,
											},
										},
									},
									{
										Keyword: KeywordNone,
										Term: &SearchTerm{
											Identifier: IdentifierID,
											Value:      "typescript",
											Operator:   OperatorEq,
										},
										Children: nil,
									},
								},
							},
							{
								Keyword: KeywordNone,
								Term: &SearchTerm{
									Identifier: IdentifierID,
									Value:      "javascript",
									Operator:   OperatorEq,
								},
								Children: nil,
							},
						},
					},
				},
			},
		},
		{
			InputQuery: `baseline_date:2000-01-01..2000-12-31 OR id:css`,
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
									Identifier: IdentifierID,
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
							Operator:   OperatorLike,
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
							Identifier: IdentifierDescription,
							Value:      "grid",
							Operator:   OperatorLike,
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
							Identifier: IdentifierDescription,
							Value:      "CSS Grid",
							Operator:   OperatorLike,
						},
						Keyword:  KeywordNone,
						Children: nil,
					},
				},
			},
		},
		{
			InputQuery: `"::backdrop"`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierDescription,
							Value:      "::backdrop",
							Operator:   OperatorLike,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: `name:"::backdrop"`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "::backdrop",
							Operator:   OperatorLike,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: `":has()"`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierDescription,
							Value:      ":has()",
							Operator:   OperatorLike,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: `<a>`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierDescription,
							Value:      "<a>",
							Operator:   OperatorLike,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: `name:<a>`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "<a>",
							Operator:   OperatorLike,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: `name:"<a>"`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "<a>",
							Operator:   OperatorLike,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: `@charset`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierDescription,
							Value:      "@charset",
							Operator:   OperatorLike,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: `name:@charset`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "@charset",
							Operator:   OperatorLike,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: `name:"@charset"`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "@charset",
							Operator:   OperatorLike,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: `intl.segmenter`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierDescription,
							Value:      "intl.segmenter",
							Operator:   OperatorLike,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: `"intl.segmenter"`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierDescription,
							Value:      "intl.segmenter",
							Operator:   OperatorLike,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: `name:"intl.segmenter"`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "intl.segmenter",
							Operator:   OperatorLike,
						},
						Keyword: KeywordNone,
					},
				},
			},
		},
		{
			InputQuery: `name:intl.segmenter`,
			ExpectedTree: &SearchNode{
				Keyword: KeywordRoot,
				Term:    nil,
				Children: []*SearchNode{
					{
						Children: nil,
						Term: &SearchTerm{
							Identifier: IdentifierName,
							Value:      "intl.segmenter",
							Operator:   OperatorLike,
						},
						Keyword: KeywordNone,
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
									Identifier: IdentifierDescription,
									Value:      "CSS Grid",
									Operator:   OperatorLike,
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
		{
			InputQuery: `name:"has()" OR name:light-dark`,
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
									Identifier: IdentifierName,
									Value:      "has()",
									Operator:   OperatorLike,
								},
								Children: nil,
							},
							{
								Keyword: KeywordNone,
								Term: &SearchTerm{
									Identifier: IdentifierName,
									Value:      "light-dark",
									Operator:   OperatorLike,
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
	if node.Keyword == KeywordParens {
		builder.WriteString(newIndent + "(\n")
	}
	for _, child := range node.Children {
		printNode(builder, child, newIndent)
	}
	if node.Keyword == KeywordParens {
		builder.WriteString(newIndent + "}\n")
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
		// Terms missing colon and values
		{
			input: "available_on",
		},
		{
			input: "available_date",
		},
		{
			input: "baseline_status",
		},
		{
			input: "baseline_date",
		},
		{
			input: "name",
		},
		{
			input: "group",
		},
		{
			input: "id",
		},
		{
			input: "snapshot",
		},
		// Terms missing values
		{
			input: "available_on:",
		},
		{
			input: "available_date:",
		},
		{
			input: "baseline_status:",
		},
		{
			input: "baseline_date:",
		},
		{
			input: "name:",
		},
		{
			input: "group:",
		},
		{
			input: "id:",
		},
		{
			input: "snapshot:",
		},
		{
			input: "available_date:chrome",
		},
		// Other input from https://github.com/GoogleChrome/webstatus.dev/issues/286
		{
			// nolint:lll // WONTFIX. Repro from issue.
			input: `available_on:chrome available_on:firefox available_on:safari baseline_status:widely name:"a substring" baseline_status:widely -`,
		},
		{
			input: `available_on:chrome available_on Secured AI available_on:safari available_on:firefox`,
		},
		{
			input: `available_on:chrome available_on generative AI available_on:safari available_on:firefox`,
		},
		{
			input: `baseline_date:2023-01-01`,
		},
		{
			input: `baseline_date:12-26-2024..1-2-2025`,
		},
		// Other input from 2025-01-27 weekly review
		{
			input: `available_on:chrome -available_date:chrome available_on:safari`,
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
