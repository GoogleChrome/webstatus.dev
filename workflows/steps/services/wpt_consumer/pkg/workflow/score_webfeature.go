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
	"slices"
	"strings"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/wptconsumertypes"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

// WPTStatusAbbreviation is an enumeration of the abbreivations from
// https://github.com/web-platform-tests/wpt.fyi/tree/main/api#results-summaries
type WPTStatusAbbreviation string

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

// Score calculates web feature metrics from a V2 results summary file.
// It ensures the file is in the expected format and uses web features
// data for the scoring logic.
func (s ResultsSummaryFileV2) Score(
	ctx context.Context,
	testToWebFeatures *shared.WebFeaturesData) map[string]wptconsumertypes.WPTFeatureMetric {
	scoreMap := make(map[string]wptconsumertypes.WPTFeatureMetric)
	featuresToExcludeSubtests := make(map[string]any)
	for test, testSummary := range s {
		if len(testSummary.Counts) < 2 {
			// Need at least the number of subtests passes and the number of subtests
			continue
		}
		if isTestTentative(test) {
			continue
		}
		s.scoreTest(ctx, test, scoreMap, testToWebFeatures,
			testSummary.Counts[0], testSummary.Counts[1], testSummary.Status)
		s.scoreSubtests(
			ctx, test, scoreMap, featuresToExcludeSubtests, testToWebFeatures,
			testSummary.Counts[0], testSummary.Counts[1], testSummary.Status)
	}

	return scoreMap
}

func isTestTentative(test string) bool {
	// More info: https://web-platform-tests.org/writing-tests/file-names.html
	return strings.Contains(test, ".tentative") ||
		// nolint:lll // WONTFIX: comment contains long line for pinned functionality.
		// The results file uses "/" as the path separator.
		// Example: https://storage.googleapis.com/wptd/536297144c737f84096d1f448e790de0fb654956/chrome-124.0.6367.91-linux-20.04-ed25fd18da-summary_v2.json.gz
		slices.Contains(strings.Split(test, "/"), "tentative")
}

// scoreSubtests calculates the metrics for a test using the "default/subtests" methodology.
func (s ResultsSummaryFileV2) scoreSubtests(
	_ context.Context,
	test string,
	webFeatureScoreMap map[string]wptconsumertypes.WPTFeatureMetric,
	featuresToExcludeSubtests map[string]any,
	testToWebFeatures *shared.WebFeaturesData,
	numberOfSubtestPassing int,
	numberofSubtests int,
	testStatus string,
) {
	var webFeatures map[string]interface{}
	var found bool
	if webFeatures, found = (*testToWebFeatures)[test]; !found {
		return
	}
	// In the event of a crash, wpt records the incorrect number of subtests (but not tests).
	// Ignore subtest metrics for now. And mark the web feature as a whole as one to not update the subtest metrics
	if WPTStatusAbbreviation(testStatus) == WPTStatusCrash {
		for webFeature := range webFeatures {
			score := getScoreForFeature(webFeature, webFeatureScoreMap)
			// Reset the sub test metrics to nil.
			score.SubtestPass = nil
			score.TotalSubtests = nil
			score.FeatureRunDetails = map[string]interface{}{
				"status": string(WPTStatusCrash),
			}
			webFeatureScoreMap[webFeature] = score
			// Skip the feature for future sub tests calculations.
			featuresToExcludeSubtests[webFeature] = nil
		}

		return
	}
	if numberofSubtests == 0 {
		numberofSubtests = 1
		// Determine the appropriate logic based on the status, as done in JavaScript
		if WPTStatusAbbreviation(testStatus) == WPTStatusOK || WPTStatusAbbreviation(testStatus) == WPTStatusPass {
			numberOfSubtestPassing = 1 // Treat as passing single subtest if status is OK or Pass
		}
	}

	for webFeature := range webFeatures {
		if _, found := featuresToExcludeSubtests[webFeature]; found {
			// If this web feature is marked to be excluded, skip it.
			continue
		}
		webFeatureScore := getScoreForFeature(webFeature, webFeatureScoreMap)
		*webFeatureScore.TotalSubtests += int64(numberofSubtests)
		*webFeatureScore.SubtestPass += int64(numberOfSubtestPassing)
		webFeatureScoreMap[webFeature] = webFeatureScore
	}
}

func getScoreForFeature(
	webFeature string,
	webFeatureScoreMap map[string]wptconsumertypes.WPTFeatureMetric) wptconsumertypes.WPTFeatureMetric {
	score, found := webFeatureScoreMap[webFeature]
	if !found {
		var initialTestTotal, initialTestPass, initialSubtestTotal, initialSubtestPass int64 = 0, 0, 0, 0
		score = wptconsumertypes.WPTFeatureMetric{
			TotalTests:    &initialTestTotal,
			TestPass:      &initialTestPass,
			TotalSubtests: &initialSubtestTotal,
			SubtestPass:   &initialSubtestPass,
			// Up to the setter to initialize this since it is not commonly used.
			FeatureRunDetails: nil,
		}
	}

	return score
}

// scoreTest updates web feature metrics for a single test
// based on provided subtest results and web features data.
func (s ResultsSummaryFileV2) scoreTest(
	_ context.Context,
	test string,
	webFeatureScoreMap map[string]wptconsumertypes.WPTFeatureMetric,
	testToWebFeatures *shared.WebFeaturesData,
	numberOfSubtestPassing int,
	numberofSubtests int,
	testStatus string,
) {
	var webFeatures map[string]interface{}
	var found bool
	if webFeatures, found = (*testToWebFeatures)[test]; !found {
		// There are no web features associated with this test. Skip
		return
	}
	// Calculate the value early so we can re-use for multiple web features.
	// Logic for zero subtests
	var countsAsPassing bool
	if numberofSubtests == 0 {
		// Determine the appropriate logic based on the status, as done in JavaScript
		if WPTStatusAbbreviation(testStatus) == WPTStatusOK || WPTStatusAbbreviation(testStatus) == WPTStatusPass {
			countsAsPassing = true // Treat as passing if status is OK or Pass
		}
	} else {
		countsAsPassing = numberOfSubtestPassing == numberofSubtests
	}
	for webFeature := range webFeatures {
		webFeatureScore := getScoreForFeature(webFeature, webFeatureScoreMap)
		*webFeatureScore.TotalTests++
		// If all of the sub tests passed, only count it.
		if countsAsPassing {
			*webFeatureScore.TestPass++
		}
		webFeatureScoreMap[webFeature] = webFeatureScore
	}
}
