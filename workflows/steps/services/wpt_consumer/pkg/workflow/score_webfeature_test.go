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

package workflow

import (
	"context"
	"reflect"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/wptconsumertypes"
	"github.com/web-platform-tests/wpt.fyi/api/query"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

func getSimpleWebFeaturesData() shared.WebFeaturesData {
	return shared.WebFeaturesData{
		"test1.html": {
			"feature1": nil,
		},
	}
}

func getSimpleSummary() ResultsSummaryFile {
	return ResultsSummaryFile{
		"test1.html": query.SummaryResult{
			Status: string(WPTStatusPass),
			Counts: []int{1, 1},
		},
	}
}

func getComplexWebFeaturesData() shared.WebFeaturesData {
	return shared.WebFeaturesData{
		"test1.html": {
			"feature1": nil,
			"feature2": nil,
		},
		"test2-not-passing.html": {
			"feature2": nil,
			"feature3": nil,
			"feature4": nil,
		},
		"test3.html": {
			"feature1": nil,
			"feature2": nil,
			"feature3": nil,
			"feature5": nil,
		},
		"malformed-counts-test.html": {
			"feature5": nil,
		},
	}
}

func getComplexSummary() ResultsSummaryFile {
	return ResultsSummaryFile{
		"test1.html": query.SummaryResult{
			Status: string(WPTStatusPass),
			Counts: []int{1, 1},
		},
		"test2-not-passing.html": query.SummaryResult{
			Status: string(WPTStatusFail),
			Counts: []int{1, 11},
		},
		"test3.html": query.SummaryResult{
			Status: string(WPTStatusPass),
			Counts: []int{100, 100},
		},
		"no-webfeatures-mapping-test.html": query.SummaryResult{
			Status: string(WPTStatusPass),
			Counts: []int{1000, 1000},
		},
		// Mapped in side web features, it should no contribute the count if the summary data is bad.
		"malformed-counts-test.html": query.SummaryResult{
			Status: string(WPTStatusPass),
			Counts: []int{1000},
		},
		"passing-but-test-not-mapped-in-webfeatures-test.html": query.SummaryResult{
			Status: string(WPTStatusPass),
			Counts: []int{10, 10},
		},
	}
}

func valuePtr[T any](in T) *T { return &in }

func TestScore(t *testing.T) {
	testCases := []struct {
		name              string
		summary           ResultsSummaryFile
		testToWebFeatures shared.WebFeaturesData
		expectedOutput    map[string]wptconsumertypes.WPTFeatureMetric
	}{
		{
			name:              "simple",
			testToWebFeatures: getSimpleWebFeaturesData(),
			summary:           getSimpleSummary(),
			expectedOutput: map[string]wptconsumertypes.WPTFeatureMetric{
				"feature1": {
					TotalTests: valuePtr[int64](1),
					TestPass:   valuePtr[int64](1),
				},
			},
		},
		{
			name:              "complex",
			testToWebFeatures: getComplexWebFeaturesData(),
			summary:           getComplexSummary(),
			expectedOutput: map[string]wptconsumertypes.WPTFeatureMetric{
				"feature1": {
					TotalTests: valuePtr[int64](2),
					TestPass:   valuePtr[int64](2),
				},
				"feature2": {
					TotalTests: valuePtr[int64](3),
					TestPass:   valuePtr[int64](2),
				},
				"feature3": {
					TotalTests: valuePtr[int64](2),
					TestPass:   valuePtr[int64](1),
				},
				"feature4": {
					TotalTests: valuePtr[int64](1),
					TestPass:   valuePtr[int64](0),
				},
				"feature5": {
					TotalTests: valuePtr[int64](1),
					TestPass:   valuePtr[int64](1),
				},
			},
		},
	}
	for idx, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scorer := WPTScorerForWebFeatures{}
			output := scorer.Score(
				context.Background(),
				tc.summary,
				&testCases[idx].testToWebFeatures,
			)
			if !reflect.DeepEqual(tc.expectedOutput, output) {
				t.Errorf("unexpected score\nexpected %v\nreceived %v", tc.expectedOutput, output)
			}
		})
	}
}
