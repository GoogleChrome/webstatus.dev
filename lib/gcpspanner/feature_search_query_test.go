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
			Operator: searchtypes.OperatorRoot,
			Term:     nil,
			Children: []*searchtypes.SearchNode{
				{
					Term: &searchtypes.SearchTerm{
						Identifier: searchtypes.IdentifierAvailableOn,
						Value:      "chrome",
					},
					Children: nil,
					Operator: searchtypes.OperatorNone,
				},
			},
		},
	}

	simpleNameQuery = TestTree{
		Query: `name:"CSS Grid"`,
		InputTree: &searchtypes.SearchNode{
			Operator: searchtypes.OperatorRoot,
			Term:     nil,
			Children: []*searchtypes.SearchNode{
				{
					Children: nil,
					Term: &searchtypes.SearchTerm{
						Identifier: searchtypes.IdentifierName,
						Value:      "CSS Grid",
					},
					Operator: searchtypes.OperatorNone,
				},
			},
		},
	}

	simpleNameByIDQuery = TestTree{
		Query: `name:grid`,
		InputTree: &searchtypes.SearchNode{
			Operator: searchtypes.OperatorRoot,
			Term:     nil,
			Children: []*searchtypes.SearchNode{
				{
					Children: nil,
					Term: &searchtypes.SearchTerm{
						Identifier: searchtypes.IdentifierName,
						Value:      "grid",
					},
					Operator: searchtypes.OperatorNone,
				},
			},
		},
	}

	availableOnBaselineStatus = TestTree{
		Query: "available_on:chrome AND baseline_status:widely",
		InputTree: &searchtypes.SearchNode{
			Operator: searchtypes.OperatorRoot,
			Term:     nil,
			Children: []*searchtypes.SearchNode{
				{
					Operator: searchtypes.OperatorAND,
					Term:     nil,
					Children: []*searchtypes.SearchNode{
						{
							Children: nil,
							Term: &searchtypes.SearchTerm{
								Identifier: searchtypes.IdentifierAvailableOn,
								Value:      "chrome",
							},
							Operator: searchtypes.OperatorNone,
						},
						{
							Children: nil,
							Term: &searchtypes.SearchTerm{
								Identifier: searchtypes.IdentifierBaselineStatus,
								Value:      "widely",
							},
							Operator: searchtypes.OperatorNone,
						},
					},
				},
			},
		},
	}

	availableOnBaselineStatusWithNegation = TestTree{
		Query: "-available_on:chrome AND baseline_status:widely",
		InputTree: &searchtypes.SearchNode{
			Operator: searchtypes.OperatorRoot,
			Term:     nil,
			Children: []*searchtypes.SearchNode{
				{
					Operator: searchtypes.OperatorAND,
					Term:     nil,
					Children: []*searchtypes.SearchNode{
						{
							Children: nil,
							Term: &searchtypes.SearchTerm{
								Identifier: searchtypes.IdentifierAvailableOn,
								Value:      "chrome",
							},
							Operator: searchtypes.OperatorNegation,
						},
						{
							Children: nil,
							Term: &searchtypes.SearchTerm{
								Identifier: searchtypes.IdentifierBaselineStatus,
								Value:      "widely",
							},
							Operator: searchtypes.OperatorNone,
						},
					},
				},
			},
		},
	}

	complexQuery = TestTree{
		Query: "available_on:chrome (baseline_status:widely OR name:avif) OR name:grid",
		InputTree: &searchtypes.SearchNode{
			Operator: searchtypes.OperatorRoot,
			Term:     nil,
			Children: []*searchtypes.SearchNode{
				{
					Operator: searchtypes.OperatorOR,
					Term:     nil,
					Children: []*searchtypes.SearchNode{
						{
							Operator: searchtypes.OperatorAND,
							Term:     nil,
							Children: []*searchtypes.SearchNode{
								{
									Operator: searchtypes.OperatorNone,
									Children: nil,
									Term: &searchtypes.SearchTerm{
										Identifier: searchtypes.IdentifierAvailableOn, Value: "chrome"},
								},
								{
									Operator: searchtypes.OperatorOR,
									Term:     nil,
									Children: []*searchtypes.SearchNode{
										{
											Operator: searchtypes.OperatorNone,
											Children: nil,
											Term: &searchtypes.SearchTerm{
												Identifier: searchtypes.IdentifierBaselineStatus, Value: "widely"},
										},
										{
											Operator: searchtypes.OperatorNone,
											Children: nil,
											Term: &searchtypes.SearchTerm{
												Identifier: searchtypes.IdentifierName, Value: "avif"},
										},
									},
								},
							},
						},
						{
							Operator: searchtypes.OperatorNone,
							Children: nil,
							Term:     &searchtypes.SearchTerm{Identifier: searchtypes.IdentifierName, Value: "grid"},
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
			expectedClauses: []string{`(wf.ID IN (SELECT WebFeatureID FROM BrowserFeatureAvailabilities
WHERE BrowserName = @param0))`},
			expectedParams: map[string]interface{}{
				"param0": "chrome",
			},
		},
		{
			inputTestTree:   simpleNameQuery,
			expectedClauses: []string{`((wf.Name_Lowercase LIKE @param0 OR wf.FeatureKey_Lowercase LIKE @param0))`},
			expectedParams: map[string]interface{}{
				"param0": "%" + "css grid" + "%",
			},
		},
		{
			inputTestTree:   simpleNameByIDQuery,
			expectedClauses: []string{`((wf.Name_Lowercase LIKE @param0 OR wf.FeatureKey_Lowercase LIKE @param0))`},
			expectedParams: map[string]interface{}{
				"param0": "%" + "grid" + "%",
			},
		},
		{
			inputTestTree: availableOnBaselineStatus,
			expectedClauses: []string{`((wf.ID IN (SELECT WebFeatureID FROM BrowserFeatureAvailabilities
WHERE BrowserName = @param0)) AND (fbs.Status = @param1))`},
			expectedParams: map[string]interface{}{
				"param0": "chrome",
				"param1": "high",
			},
		},
		{
			inputTestTree: availableOnBaselineStatusWithNegation,
			expectedClauses: []string{`((NOT (wf.ID IN (SELECT WebFeatureID FROM BrowserFeatureAvailabilities
WHERE BrowserName = @param0))) AND (fbs.Status = @param1))`},
			expectedParams: map[string]interface{}{
				"param0": "chrome",
				"param1": "high",
			},
		},
		{
			inputTestTree: complexQuery,
			expectedClauses: []string{`(((wf.ID IN (SELECT WebFeatureID FROM BrowserFeatureAvailabilities
WHERE BrowserName = @param0)) AND ((fbs.Status = @param1) OR ((wf.Name_Lowercase LIKE @param2 OR wf.FeatureKey_Lowercase LIKE @param2)))) OR ((wf.Name_Lowercase LIKE @param3 OR wf.FeatureKey_Lowercase LIKE @param3)))`},
			expectedParams: map[string]interface{}{
				"param0": "chrome",
				"param1": "high",
				"param2": "%" + "avif" + "%",
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
