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
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/bcdconsumertypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/mdn__browser_compat_data"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/bcd_consumer/pkg/data"
)

func getAllSupportedFilters() []string {
	return []string{
		"chrome",
		"edge",
		"firefox",
		"safari",
	}
}

func TestCheckBrowserFilters(t *testing.T) {
	testCases := []struct {
		name          string
		input         []string
		expectedError error
	}{
		{
			name:          "all supported filters",
			input:         getAllSupportedFilters(),
			expectedError: nil,
		},
		{
			name:          "all supported filters and one bad filter",
			input:         append(getAllSupportedFilters(), "bad"),
			expectedError: ErrUnknownBrowserFilter,
		},
		{
			name:          "no filters provided",
			input:         nil,
			expectedError: ErrNoBrowserFiltersPresent,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter := BCDDataFilter{}
			err := filter.checkBrowserFilters(tc.input)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Unexpected error:\nGot: %v\nWant: %v", err, tc.expectedError)
			}
		})
	}
}

func TestFilterData(t *testing.T) {
	testCases := []struct {
		name             string
		inputBCD         *data.BCDData
		filteredBrowsers []string
		expectedResult   []bcdconsumertypes.BrowserRelease
		expectedError    error
	}{
		{
			name: "Valid Input",
			inputBCD: &data.BCDData{
				// nolint: exhaustruct // WONTFIX. External struct.
				BrowserData: mdn__browser_compat_data.BrowserData{
					Browsers: map[string]mdn__browser_compat_data.BrowserStatement{
						"chrome": {
							Releases: map[string]mdn__browser_compat_data.ReleaseStatement{
								"100.0.0": {ReleaseDate: valuePtr("2024-04-09")},
								"101.0.0": {ReleaseDate: valuePtr("2024-05-12")},
							},
						},
						"firefox": {
							Releases: map[string]mdn__browser_compat_data.ReleaseStatement{
								"110.0": {ReleaseDate: valuePtr("2024-04-20")},
							},
						},
					},
				},
			},
			filteredBrowsers: []string{"chrome", "firefox"},
			expectedResult: []bcdconsumertypes.BrowserRelease{
				{BrowserName: "chrome", BrowserVersion: "100.0.0", ReleaseDate: time.Date(2024, 4, 9, 0, 0, 0, 0, time.UTC)},
				{BrowserName: "chrome", BrowserVersion: "101.0.0", ReleaseDate: time.Date(2024, 5, 12, 0, 0, 0, 0, time.UTC)},
				{BrowserName: "firefox", BrowserVersion: "110.0", ReleaseDate: time.Date(2024, 4, 20, 0, 0, 0, 0, time.UTC)},
			},
			expectedError: nil,
		},
		{
			name: "Non-existent Browser",
			inputBCD: &data.BCDData{
				// nolint: exhaustruct // WONTFIX. External struct.
				BrowserData: mdn__browser_compat_data.BrowserData{
					Browsers: map[string]mdn__browser_compat_data.BrowserStatement{
						"firefox": {
							Releases: map[string]mdn__browser_compat_data.ReleaseStatement{
								"100.0.0": {ReleaseDate: valuePtr("2024-04-09")},
							},
						},
					},
				},
			},
			filteredBrowsers: []string{"firefox", "chrome"}, // 'chrome' not found
			expectedResult:   nil,
			expectedError:    ErrMissingBrowser,
		},
		{
			name: "Invalid Release Date Format",
			inputBCD: &data.BCDData{
				// nolint: exhaustruct // WONTFIX. External struct.
				BrowserData: mdn__browser_compat_data.BrowserData{
					Browsers: map[string]mdn__browser_compat_data.BrowserStatement{
						"firefox": {
							Releases: map[string]mdn__browser_compat_data.ReleaseStatement{
								"110.0": {ReleaseDate: valuePtr("2024-04-20")},
								"111.0": {ReleaseDate: valuePtr("invalid-date")}, // Incorrect format
							},
						},
					},
				},
			},
			filteredBrowsers: []string{"firefox"},
			expectedResult:   nil,
			expectedError:    ErrMalformedReleaseDate,
		},
		{
			name: "Release with No Date",
			inputBCD: &data.BCDData{
				// nolint: exhaustruct // WONTFIX. External struct.
				BrowserData: mdn__browser_compat_data.BrowserData{
					Browsers: map[string]mdn__browser_compat_data.BrowserStatement{
						"firefox": {
							Releases: map[string]mdn__browser_compat_data.ReleaseStatement{
								"100.0.0": {ReleaseDate: valuePtr("2024-04-09")},
								"101.0.0": {ReleaseDate: nil}, // Missing date
							},
						},
					},
				},
			},
			filteredBrowsers: []string{"firefox"},
			expectedResult: []bcdconsumertypes.BrowserRelease{ // Release 101.0.0 should be excluded
				{BrowserName: "firefox", BrowserVersion: "100.0.0", ReleaseDate: time.Date(2024, 4, 9, 0, 0, 0, 0, time.UTC)},
			},
			expectedError: nil,
		},
		{
			name:             "Nil BCDData Input",
			inputBCD:         nil,
			filteredBrowsers: []string{"firefox"}, // Irrelevant with nil input
			expectedResult:   nil,
			expectedError:    nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter := BCDDataFilter{}
			result, err := filter.FilterData(tc.inputBCD, tc.filteredBrowsers)

			// Sort the items to have stable output.
			sort.Slice(result, func(i, j int) bool {
				if result[i].BrowserName == result[j].BrowserName {
					return result[i].ReleaseDate.Before(result[j].ReleaseDate)
				}

				return result[i].BrowserName < result[j].BrowserName
			})

			if !reflect.DeepEqual(result, tc.expectedResult) {
				t.Errorf("Unexpected result:\nGot: %v\nWant: %v", result, tc.expectedResult)
			}

			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Unexpected error:\nGot: %v\nWant: %v", err, tc.expectedError)
			}
		})
	}
}
