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
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
)

type TestTree struct {
	Query     string
	InputTree *searchtypes.SearchNode
}

// nolint: gochecknoglobals // for testing only.
var (
	simpleAvailableOnQuery = TestTree{
		Query: "available_on:chrome",
		InputTree: &searchtypes.SearchNode{
			Keyword: searchtypes.KeywordRoot,
			Term:    nil,
			Children: []*searchtypes.SearchNode{
				{
					Term: &searchtypes.SearchTerm{
						Identifier: searchtypes.IdentifierAvailableOn,
						Operator:   searchtypes.OperatorEq,
						Value:      "chrome",
					},
					Children: nil,
					Keyword:  searchtypes.KeywordNone,
				},
			},
		},
	}

	simpleNameQuery = TestTree{
		Query: `name:"CSS Grid"`,
		InputTree: &searchtypes.SearchNode{
			Keyword: searchtypes.KeywordRoot,
			Term:    nil,
			Children: []*searchtypes.SearchNode{
				{
					Children: nil,
					Term: &searchtypes.SearchTerm{
						Identifier: searchtypes.IdentifierName,
						Value:      "CSS Grid",
						Operator:   searchtypes.OperatorEq,
					},
					Keyword: searchtypes.KeywordNone,
				},
			},
		},
	}

	simpleNameByIDQuery = TestTree{
		Query: `name:grid`,
		InputTree: &searchtypes.SearchNode{
			Keyword: searchtypes.KeywordRoot,
			Term:    nil,
			Children: []*searchtypes.SearchNode{
				{
					Children: nil,
					Term: &searchtypes.SearchTerm{
						Identifier: searchtypes.IdentifierName,
						Value:      "grid",
						Operator:   searchtypes.OperatorEq,
					},
					Keyword: searchtypes.KeywordNone,
				},
			},
		},
	}

	simpleBCDQuery = TestTree{
		Query: `bcd:"html.elements.address"`,
		InputTree: &searchtypes.SearchNode{
			Keyword: searchtypes.KeywordRoot,
			Term:    nil,
			Children: []*searchtypes.SearchNode{
				{
					Keyword: searchtypes.KeywordNone,
					Term: &searchtypes.SearchTerm{
						Identifier: "bcd",
						Value:      "html.elements.address",
						Operator:   searchtypes.OperatorEq,
					},
				},
			},
		},
	}

	availableOnBaselineStatus = TestTree{
		Query: "available_on:chrome AND baseline_status:widely",
		InputTree: &searchtypes.SearchNode{
			Keyword: searchtypes.KeywordRoot,
			Term:    nil,
			Children: []*searchtypes.SearchNode{
				{
					Keyword: searchtypes.KeywordAND,
					Term:    nil,
					Children: []*searchtypes.SearchNode{
						{
							Children: nil,
							Term: &searchtypes.SearchTerm{
								Identifier: searchtypes.IdentifierAvailableOn,
								Value:      "chrome",
								Operator:   searchtypes.OperatorEq,
							},
							Keyword: searchtypes.KeywordNone,
						},
						{
							Children: nil,
							Term: &searchtypes.SearchTerm{
								Identifier: searchtypes.IdentifierBaselineStatus,
								Value:      "widely",
								Operator:   searchtypes.OperatorEq,
							},
							Keyword: searchtypes.KeywordNone,
						},
					},
				},
			},
		},
	}

	availableOnBaselineStatusWithNegation = TestTree{
		Query: "-available_on:chrome AND baseline_status:widely",
		InputTree: &searchtypes.SearchNode{
			Keyword: searchtypes.KeywordRoot,
			Term:    nil,
			Children: []*searchtypes.SearchNode{
				{
					Keyword: searchtypes.KeywordAND,
					Term:    nil,
					Children: []*searchtypes.SearchNode{
						{
							Children: nil,
							Term: &searchtypes.SearchTerm{
								Identifier: searchtypes.IdentifierAvailableOn,
								Value:      "chrome",
								Operator:   searchtypes.OperatorNeq,
							},
							Keyword: searchtypes.KeywordNone,
						},
						{
							Children: nil,
							Term: &searchtypes.SearchTerm{
								Identifier: searchtypes.IdentifierBaselineStatus,
								Value:      "widely",
								Operator:   searchtypes.OperatorEq,
							},
							Keyword: searchtypes.KeywordNone,
						},
					},
				},
			},
		},
	}

	baselineDateRange = TestTree{
		Query: "baseline_date:2000-01-01..2000-12-31",
		InputTree: &searchtypes.SearchNode{
			Keyword: searchtypes.KeywordRoot,
			Term:    nil,
			Children: []*searchtypes.SearchNode{
				{
					Keyword: searchtypes.KeywordAND,
					Term:    nil,
					Children: []*searchtypes.SearchNode{
						{
							Keyword: searchtypes.KeywordNone,
							Term: &searchtypes.SearchTerm{
								Identifier: searchtypes.IdentifierBaselineDate,
								Value:      "2000-01-01",
								Operator:   searchtypes.OperatorGtEq,
							},
							Children: nil,
						},
						{
							Keyword: searchtypes.KeywordNone,
							Term: &searchtypes.SearchTerm{
								Identifier: searchtypes.IdentifierBaselineDate,
								Value:      "2000-12-31",
								Operator:   searchtypes.OperatorLtEq,
							},
							Children: nil,
						},
					},
				},
			},
		},
	}

	baselineDateRangeNegation = TestTree{
		Query: "-baseline_date:2000-01-01..2000-12-31",
		InputTree: &searchtypes.SearchNode{
			Keyword: searchtypes.KeywordRoot,
			Term:    nil,
			Children: []*searchtypes.SearchNode{
				{
					Keyword: searchtypes.KeywordOR,
					Term:    nil,
					Children: []*searchtypes.SearchNode{
						{
							Keyword: searchtypes.KeywordNone,
							Term: &searchtypes.SearchTerm{
								Identifier: searchtypes.IdentifierBaselineDate,
								Value:      "2000-01-01",
								Operator:   searchtypes.OperatorLt,
							},
							Children: nil,
						},
						{
							Keyword: searchtypes.KeywordNone,
							Term: &searchtypes.SearchTerm{
								Identifier: searchtypes.IdentifierBaselineDate,
								Value:      "2000-12-31",
								Operator:   searchtypes.OperatorGt,
							},
							Children: nil,
						},
					},
				},
			},
		},
	}

	complexQuery = TestTree{
		Query: "available_on:chrome (baseline_status:widely OR name:avif) OR name:grid",
		InputTree: &searchtypes.SearchNode{
			Keyword: searchtypes.KeywordRoot,
			Term:    nil,
			Children: []*searchtypes.SearchNode{
				{
					Keyword: searchtypes.KeywordOR,
					Term:    nil,
					Children: []*searchtypes.SearchNode{
						{
							Keyword: searchtypes.KeywordAND,
							Term:    nil,
							Children: []*searchtypes.SearchNode{
								{
									Keyword:  searchtypes.KeywordNone,
									Children: nil,
									Term: &searchtypes.SearchTerm{
										Identifier: searchtypes.IdentifierAvailableOn,
										Value:      "chrome",
										Operator:   searchtypes.OperatorEq,
									},
								},
								{
									Keyword: searchtypes.KeywordParens,
									Term:    nil,
									Children: []*searchtypes.SearchNode{
										{
											Keyword: searchtypes.KeywordOR,
											Term:    nil,
											Children: []*searchtypes.SearchNode{
												{
													Keyword:  searchtypes.KeywordNone,
													Children: nil,
													Term: &searchtypes.SearchTerm{
														Identifier: searchtypes.IdentifierBaselineStatus,
														Value:      "widely",
														Operator:   searchtypes.OperatorEq,
													},
												},
												{
													Keyword:  searchtypes.KeywordNone,
													Children: nil,
													Term: &searchtypes.SearchTerm{
														Identifier: searchtypes.IdentifierName,
														Value:      "avif",
														Operator:   searchtypes.OperatorEq,
													},
												},
											},
										},
									},
								},
							},
						},
						{
							Keyword:  searchtypes.KeywordNone,
							Children: nil,
							Term: &searchtypes.SearchTerm{
								Identifier: searchtypes.IdentifierName,
								Value:      "grid",
								Operator:   searchtypes.OperatorEq,
							},
						},
					},
				},
			},
		},
	}

	complexNestedQuery = TestTree{
		Query: "(available_on:chrome AND (baseline_status:widely OR baseline_status:limited)) OR name:grid",
		InputTree: &searchtypes.SearchNode{
			Keyword: searchtypes.KeywordRoot,
			Term:    nil,
			Children: []*searchtypes.SearchNode{
				{
					Keyword: searchtypes.KeywordOR,
					Term:    nil,
					Children: []*searchtypes.SearchNode{
						{
							Keyword: searchtypes.KeywordParens,
							Term:    nil,
							Children: []*searchtypes.SearchNode{
								{
									Keyword: searchtypes.KeywordAND,
									Children: []*searchtypes.SearchNode{
										{
											Term: &searchtypes.SearchTerm{
												Identifier: searchtypes.IdentifierAvailableOn,
												Value:      "chrome",
												Operator:   searchtypes.OperatorEq,
											},
											Keyword: searchtypes.KeywordNone,
										},
										{
											Term:    nil,
											Keyword: searchtypes.KeywordParens,
											Children: []*searchtypes.SearchNode{
												{
													Keyword: searchtypes.KeywordOR,
													Term:    nil,
													Children: []*searchtypes.SearchNode{
														{
															Children: nil,
															Term: &searchtypes.SearchTerm{
																Identifier: searchtypes.IdentifierBaselineStatus,
																Value:      "widely",
																Operator:   searchtypes.OperatorEq,
															},
															Keyword: searchtypes.KeywordNone,
														},
														{
															Children: nil,
															Term: &searchtypes.SearchTerm{
																Identifier: searchtypes.IdentifierBaselineStatus,
																Value:      "limited",
																Operator:   searchtypes.OperatorEq,
															},
															Keyword: searchtypes.KeywordNone,
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
							Term: &searchtypes.SearchTerm{
								Identifier: searchtypes.IdentifierName,
								Value:      "grid",
								Operator:   searchtypes.OperatorEq,
							},
							Keyword: searchtypes.KeywordNone,
						},
					},
				},
			},
		},
	}

	repeatedSimpleTermQuery = TestTree{
		Query: "id:html OR id:css OR id:typescript OR id:javascript",
		InputTree: &searchtypes.SearchNode{
			Keyword: searchtypes.KeywordRoot,
			Term:    nil,
			Children: []*searchtypes.SearchNode{
				{
					Term:    nil,
					Keyword: searchtypes.KeywordOR,
					Children: []*searchtypes.SearchNode{
						{
							Term:    nil,
							Keyword: searchtypes.KeywordOR,
							Children: []*searchtypes.SearchNode{
								{
									Term:    nil,
									Keyword: searchtypes.KeywordOR,
									Children: []*searchtypes.SearchNode{
										{
											Keyword: searchtypes.KeywordNone,
											Term: &searchtypes.SearchTerm{
												Identifier: searchtypes.IdentifierID,
												Value:      "html",
												Operator:   searchtypes.OperatorEq,
											},
											Children: nil,
										},
										{
											Keyword: searchtypes.KeywordNone,
											Term: &searchtypes.SearchTerm{
												Identifier: searchtypes.IdentifierID,
												Value:      "css",
												Operator:   searchtypes.OperatorEq,
											},
											Children: nil,
										},
									},
								},
								{
									Keyword: searchtypes.KeywordNone,
									Term: &searchtypes.SearchTerm{
										Identifier: searchtypes.IdentifierID,
										Value:      "typescript",
										Operator:   searchtypes.OperatorEq,
									},
									Children: nil,
								},
							},
						},
						{
							Keyword: searchtypes.KeywordNone,
							Term: &searchtypes.SearchTerm{
								Identifier: searchtypes.IdentifierID,
								Value:      "javascript",
								Operator:   searchtypes.OperatorEq,
							},
							Children: nil,
						},
					},
				},
			},
		},
	}
)

// nolint:lll // Some queries will be long lines.
func TestBuild(t *testing.T) {
	testCases := []struct {
		inputTestTree   TestTree
		expectedClauses []string
		expectedParams  map[string]interface{}
	}{
		{
			inputTestTree: simpleAvailableOnQuery,
			expectedClauses: []string{`wf.ID IN (SELECT WebFeatureID FROM BrowserFeatureAvailabilities
WHERE BrowserName = @param0)`},
			expectedParams: map[string]interface{}{
				"param0": "chrome",
			},
		},
		{
			inputTestTree:   simpleNameQuery,
			expectedClauses: []string{`(wf.Name_Lowercase LIKE @param0 OR wf.FeatureKey_Lowercase LIKE @param0)`},
			expectedParams: map[string]interface{}{
				"param0": "%" + "css grid" + "%",
			},
		},
		{
			inputTestTree:   simpleNameByIDQuery,
			expectedClauses: []string{`(wf.Name_Lowercase LIKE @param0 OR wf.FeatureKey_Lowercase LIKE @param0)`},
			expectedParams: map[string]interface{}{
				"param0": "%" + "grid" + "%",
			},
		},
		{
			inputTestTree:   simpleBCDQuery,
			expectedClauses: []string{`wf.ID IN (SELECT ID FROM WebFeatureBrowserCompatFeatures WHERE CompatFeature = @param0)`},
			expectedParams: map[string]interface{}{
				"param0": "html.elements.address",
			},
		},
		{
			inputTestTree: availableOnBaselineStatus,
			expectedClauses: []string{`wf.ID IN (SELECT WebFeatureID FROM BrowserFeatureAvailabilities
WHERE BrowserName = @param0) AND fbs.Status = @param1`},
			expectedParams: map[string]interface{}{
				"param0": "chrome",
				"param1": "high",
			},
		},
		{
			inputTestTree: availableOnBaselineStatusWithNegation,
			expectedClauses: []string{`wf.ID NOT IN (SELECT WebFeatureID FROM BrowserFeatureAvailabilities
WHERE BrowserName = @param0) AND fbs.Status = @param1`},
			expectedParams: map[string]interface{}{
				"param0": "chrome",
				"param1": "high",
			},
		},
		{
			inputTestTree: complexQuery,
			expectedClauses: []string{`wf.ID IN (SELECT WebFeatureID FROM BrowserFeatureAvailabilities
WHERE BrowserName = @param0) AND (fbs.Status = @param1 OR (wf.Name_Lowercase LIKE @param2 OR wf.FeatureKey_Lowercase LIKE @param2)) OR (wf.Name_Lowercase LIKE @param3 OR wf.FeatureKey_Lowercase LIKE @param3)`},
			expectedParams: map[string]interface{}{
				"param0": "chrome",
				"param1": "high",
				"param2": "%" + "avif" + "%",
				"param3": "%" + "grid" + "%",
			},
		},
		{
			inputTestTree: baselineDateRange,
			expectedClauses: []string{
				`LowDate >= @param0 AND LowDate <= @param1`,
			},
			expectedParams: map[string]interface{}{
				"param0": time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				"param1": time.Date(2000, 12, 31, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			inputTestTree: baselineDateRangeNegation,
			expectedClauses: []string{
				`LowDate < @param0 OR LowDate > @param1`,
			},
			expectedParams: map[string]interface{}{
				"param0": time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				"param1": time.Date(2000, 12, 31, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			inputTestTree:   repeatedSimpleTermQuery,
			expectedClauses: []string{`(wf.FeatureKey_Lowercase = @param0) OR (wf.FeatureKey_Lowercase = @param1) OR (wf.FeatureKey_Lowercase = @param2) OR (wf.FeatureKey_Lowercase = @param3)`},
			expectedParams: map[string]interface{}{
				"param0": "html",
				"param1": "css",
				"param2": "typescript",
				"param3": "javascript",
			},
		},
		{
			inputTestTree: complexNestedQuery,
			expectedClauses: []string{`(wf.ID IN (SELECT WebFeatureID FROM BrowserFeatureAvailabilities
WHERE BrowserName = @param0) AND (fbs.Status = @param1 OR fbs.Status = @param2)) OR (wf.Name_Lowercase LIKE @param3 OR wf.FeatureKey_Lowercase LIKE @param3)`,
			},
			expectedParams: map[string]interface{}{
				"param0": "chrome",
				"param1": "high",
				"param2": "none",
				"param3": "%" + "grid" + "%",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.inputTestTree.Query, func(t *testing.T) {
			b := NewFeatureSearchFilterBuilder()
			filter := b.Build(tc.inputTestTree.InputTree)
			if !slices.Equal[[]string](filter.Filters(), tc.expectedClauses) {
				t.Errorf("\nexpected clause [%s]\n  actual clause [%s]", tc.expectedClauses, filter.Filters())
			}
			if !reflect.DeepEqual(tc.expectedParams, filter.Params()) {
				t.Errorf("expected params (%+v) actual params (%+v)", tc.expectedParams, filter.Params())
			}
		})
	}
}
