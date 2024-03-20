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
	"maps"
	"testing"
	"time"
)

func TestGCPBuildChannelMetricsFilter(t *testing.T) {
	testCases := []struct {
		name           string
		inputChannel   string
		inputResults   []LatestRunResult
		expectedFilter string
		expectedParams map[string]interface{}
	}{
		{
			name:           "no results",
			inputChannel:   "stable",
			inputResults:   []LatestRunResult{},
			expectedFilter: "",
			expectedParams: nil,
		},
		{
			name:         "one result",
			inputChannel: "stable",
			inputResults: []LatestRunResult{
				{
					Channel:     "stable",
					BrowserName: "fooBrowser",
					TimeStart:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			expectedFilter: " AND ((metrics.BrowserName = @stablebrowser0 AND metrics.TimeStart = @stabletime0))",
			expectedParams: map[string]interface{}{
				"stablebrowser0": "fooBrowser",
				"stabletime0":    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:         "multiple results",
			inputChannel: "stable",
			inputResults: []LatestRunResult{
				{
					Channel:     "stable",
					BrowserName: "fooBrowser",
					TimeStart:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Channel:     "stable",
					BrowserName: "barBrowser",
					TimeStart:   time.Date(2023, time.December, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			// nolint: lll // WONTFIX. Expected string will be long.
			expectedFilter: " AND ((metrics.BrowserName = @stablebrowser0 AND metrics.TimeStart = @stabletime0) OR (metrics.BrowserName = @stablebrowser1 AND metrics.TimeStart = @stabletime1))",
			expectedParams: map[string]interface{}{
				"stablebrowser0": "fooBrowser",
				"stabletime0":    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				"stablebrowser1": "barBrowser",
				"stabletime1":    time.Date(2023, time.December, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			q := GCPFeatureSearchBaseQuery{}
			filterStr, params := q.buildChannelMetricsFilter(tc.inputChannel, tc.inputResults)
			if filterStr != tc.expectedFilter {
				t.Errorf("unexpected filter.\nexpected |%s|\nactual |%s|", tc.expectedFilter, filterStr)
			}
			if !maps.Equal(tc.expectedParams, params) {
				t.Errorf("unexpected params.\nexpected |%s|\nactual |%s|", tc.expectedParams, params)
			}
		})
	}
}
