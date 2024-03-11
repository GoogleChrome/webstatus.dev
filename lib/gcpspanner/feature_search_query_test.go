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
		Query: "available_on:chrome AND baseline_status:high",
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
								Value:      "high",
							},
							Operator: searchtypes.OperatorNone,
						},
					},
				},
			},
		},
	}

	availableOnBaselineStatusWithNegation = TestTree{
		Query: "-available_on:chrome AND baseline_status:high",
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
								Value:      "high",
							},
							Operator: searchtypes.OperatorNone,
						},
					},
				},
			},
		},
	}

	complexQuery = TestTree{
		Query: "available_on:chrome (baseline_status:high OR name:avif) OR name:grid",
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
												Identifier: searchtypes.IdentifierBaselineStatus, Value: "high"},
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
		inputTestTree  TestTree
		expectedClause string
		expectedParams map[string]interface{}
	}{
		{
			inputTestTree: simpleAvailableOnQuery,
			expectedClause: `(wf.FeatureID IN (SELECT FeatureID FROM BrowserFeatureAvailabilities
WHERE BrowserName = @param0))`,
			expectedParams: map[string]interface{}{
				"param0": "chrome",
			},
		},
		{
			inputTestTree:  simpleNameQuery,
			expectedClause: `((wf.Name_Lowercase LIKE @param0 OR wf.FeatureID_Lowercase LIKE @param0))`,
			expectedParams: map[string]interface{}{
				"param0": "%" + "css grid" + "%",
			},
		},
		{
			inputTestTree:  simpleNameByIDQuery,
			expectedClause: `((wf.Name_Lowercase LIKE @param0 OR wf.FeatureID_Lowercase LIKE @param0))`,
			expectedParams: map[string]interface{}{
				"param0": "%" + "grid" + "%",
			},
		},
		{
			inputTestTree: availableOnBaselineStatus,
			expectedClause: `((wf.FeatureID IN (SELECT FeatureID FROM BrowserFeatureAvailabilities
WHERE BrowserName = @param0)) AND (fbs.Status = @param1))`,
			expectedParams: map[string]interface{}{
				"param0": "chrome",
				"param1": "high",
			},
		},
		{
			inputTestTree: availableOnBaselineStatusWithNegation,
			expectedClause: `((NOT (wf.FeatureID IN (SELECT FeatureID FROM BrowserFeatureAvailabilities
WHERE BrowserName = @param0))) AND (fbs.Status = @param1))`,
			expectedParams: map[string]interface{}{
				"param0": "chrome",
				"param1": "high",
			},
		},
		{
			inputTestTree: complexQuery,
			expectedClause: `(((wf.FeatureID IN (SELECT FeatureID FROM BrowserFeatureAvailabilities
WHERE BrowserName = @param0)) AND ((fbs.Status = @param1) OR ((wf.Name_Lowercase LIKE @param2 OR wf.FeatureID_Lowercase LIKE @param2)))) OR ((wf.Name_Lowercase LIKE @param3 OR wf.FeatureID_Lowercase LIKE @param3)))`,
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
			if filter.Clause() != tc.expectedClause {
				t.Errorf("\nexpected clause [%s]\n  actual clause [%s]", tc.expectedClause, filter.Clause())
			}
			if !reflect.DeepEqual(tc.expectedParams, filter.Params()) {
				t.Errorf("expected params (%+v) actual params (%+v)", tc.expectedParams, filter.Params())
			}
		})
	}
}
