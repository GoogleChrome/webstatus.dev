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

package wptfyi

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/wptconsumertypes"
)

func TestGetRunsIntegration(t *testing.T) {
	specialNoResultsExpectedValue := -1
	type testCase struct {
		// expectedMinimumResultSize specifies the minimum number of runs expected for a given browser
		// and channel over the tested time period. This is used to verify that the client
		// correctly fetches all pages of results and not just the first page (which maxes out at 100).
		// A value of -1 indicates that no results are expected yet
		expectedMinimumResultSize int
		willFail                  bool
	}
	client := NewHTTPClient("wpt.fyi")
	// longTermResultSize means the product has been around for awhile and should get at least 100 results
	longTermResultSize := 100
	browsers := map[wptconsumertypes.BrowserName]testCase{
		// Desktop browsers
		wptconsumertypes.Chrome: {
			expectedMinimumResultSize: longTermResultSize,
			willFail:                  false,
		},
		wptconsumertypes.Edge: {
			expectedMinimumResultSize: longTermResultSize,
			willFail:                  false,
		},
		wptconsumertypes.Firefox: {
			expectedMinimumResultSize: longTermResultSize,
			willFail:                  false,
		},
		wptconsumertypes.Safari: {
			expectedMinimumResultSize: longTermResultSize,
			willFail:                  false,
		},

		// Mobile browsers
		// Stable results just started coming for ChromeAndroid
		wptconsumertypes.ChromeAndroid: {
			expectedMinimumResultSize: 1,
			willFail:                  false,
		},
		// No stable results for FirefoxAndroid yet
		wptconsumertypes.FirefoxAndroid: {
			expectedMinimumResultSize: specialNoResultsExpectedValue,
			willFail:                  false,
		},

		// Bad browser name
		wptconsumertypes.BrowserName("badname"): {
			expectedMinimumResultSize: 0,
			willFail:                  true,
		},
	}
	for browser, testCase := range browsers {
		// For now, we only care about stable channel.
		runs, err := client.GetRuns(context.TODO(), time.Now().AddDate(0, 0, -365).UTC(),
			longTermResultSize, string(browser), "stable")
		if err != nil && !testCase.willFail {
			t.Errorf("unexpected error getting runs: %s\n", err.Error())
		} else if err == nil && testCase.willFail {
			t.Error("expected an error but received none")
		}

		// Looking back a year, we should have more than 100 runs given there is a one run per day
		// This test is only to make sure we get more than the pageSize of results because currently
		// the external client will fetch the first pageSize of results but there may be actually more.
		// Our code ensures we get all the pages, not just the first page.
		if err == nil {
			// Special handling for cases where no results are expected.
			// In such cases, a successful call with 0 results is acceptable.
			if testCase.expectedMinimumResultSize == specialNoResultsExpectedValue && len(runs) == 0 {
				// No stable results expected and none received, test passes for this condition.
				return
			} else if len(runs) < testCase.expectedMinimumResultSize {
				t.Errorf("unexpected number of runs for %s. Expected at least %d runs, but got %d.",
					browser, testCase.expectedMinimumResultSize, len(runs))
			}
		}
	}
}
