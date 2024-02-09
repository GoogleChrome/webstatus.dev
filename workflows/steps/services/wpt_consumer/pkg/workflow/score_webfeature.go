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
	"cmp"
	"context"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

type WebFeatureScore struct {
	numberOfTests        int
	numberOfPassingTests int
}

type WPTStatusAbbreviation string

// Abbreivations come from
// https://github.com/web-platform-tests/wpt.fyi/tree/main/api#results-summaries
const (
	WPTStatusOK                 WPTStatusAbbreviation = "O"
	WPTStatusPass               WPTStatusAbbreviation = "P"
	WPTStatusFail               WPTStatusAbbreviation = "F"
	WPTStatusSkip               WPTStatusAbbreviation = "S"
	WPTStatusError              WPTStatusAbbreviation = "E"
	WPTStatusNotRun             WPTStatusAbbreviation = "N"
	WPTStatusCrash              WPTStatusAbbreviation = "C"
	WPTStatusTimeout            WPTStatusAbbreviation = "T"
	WPTStatusPreconditionFailed WPTStatusAbbreviation = "PF"
)

type WPTScorerForWebFeatures struct{}

func (s WPTScorerForWebFeatures) Score(
	ctx context.Context,
	summary ResultsSummaryFile,
	testToWebFeatures shared.WebFeaturesData) {
	scoreMap := make(map[string]WebFeatureScore)
	for test, testSummary := range summary {
		if len(testSummary.Counts) < 2 {
			// Need at least the number of subtests passes and the number of subtests
			continue
		}
		s.scoreTest(ctx, test, scoreMap, testToWebFeatures, testSummary.Counts[0], testSummary.Counts[1])
	}
}

func (s WPTScorerForWebFeatures) scoreTest(
	_ context.Context,
	test string,
	webFeatureScoreMap map[string]WebFeatureScore,
	testToWebFeatures shared.WebFeaturesData,
	numberOfSubtestPassing int,
	numberofSubtests int,
) {
	var webFeatures map[string]interface{}
	var found bool
	if webFeatures, found = testToWebFeatures[test]; !found {
		// There are no web features associated with this test. Skip
		return
	}
	// Calculate the value early so we can re-use for multiple web features.
	countsAsPassing := numberOfSubtestPassing == numberofSubtests
	for webFeature := range webFeatures {
		webFeatureScore := cmp.Or(
			webFeatureScoreMap[webFeature],
			WebFeatureScore{numberOfTests: 0, numberOfPassingTests: 0})
		webFeatureScore.numberOfTests++
		// If all of the sub tests passed, only count it.
		if countsAsPassing {
			webFeatureScore.numberOfPassingTests++
		}
		webFeatureScoreMap[webFeature] = webFeatureScore
	}
}
