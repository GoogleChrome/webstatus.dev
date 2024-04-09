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

package spanneradapters

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/bcdconsumertypes"
)

type MockSpannerClient struct {
	callHistory []gcpspanner.BrowserRelease
}

func getFailureRelease() bcdconsumertypes.BrowserRelease {
	return bcdconsumertypes.BrowserRelease{
		BrowserName:    "failureBrowser",
		BrowserVersion: "failureVersion",
		ReleaseDate:    time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
	}
}

func (m *MockSpannerClient) InsertBrowserRelease(_ context.Context, release gcpspanner.BrowserRelease) error {
	m.callHistory = append(m.callHistory, release)
	// Check if the release matches the failure condition
	if strings.Contains(release.BrowserName, "fail") {
		return errors.New("Simulated Spanner error")
	}

	return nil
}

func TestInsertBrowserReleases(t *testing.T) {
	testCases := []struct {
		name          string
		releases      []bcdconsumertypes.BrowserRelease
		expectedError error
	}{
		{
			name: "Success",
			releases: []bcdconsumertypes.BrowserRelease{
				{
					BrowserName:    "Chrome",
					BrowserVersion: "100",
					ReleaseDate:    time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				},
			},
			expectedError: nil,
		},
		{
			name: "Unable to store release",
			releases: []bcdconsumertypes.BrowserRelease{
				{
					BrowserName:    "Chrome",
					BrowserVersion: "100",
					ReleaseDate:    time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				},
				getFailureRelease(),
			},
			expectedError: bcdconsumertypes.ErrUnableToStoreBrowserRelease,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockSpannerClient{
				callHistory: []gcpspanner.BrowserRelease{},
			}
			consumer := NewBCDWorkflowConsumer(mockClient)

			err := consumer.InsertBrowserReleases(context.Background(), tc.releases)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(mockClient.callHistory) != len(tc.releases) {
				t.Errorf("Call count mismatch. Expected: %d, got: %d", len(tc.releases), len(mockClient.callHistory))
			} else {
				for i, release := range tc.releases {
					if !compareReleases(release, mockClient.callHistory[i]) {
						t.Errorf("Call argument mismatch at index %d", i)
					}
				}
			}
		})
	}
}

func compareReleases(r1 bcdconsumertypes.BrowserRelease, r2 gcpspanner.BrowserRelease) bool {
	return string(r1.BrowserName) == r2.BrowserName &&
		r1.BrowserVersion == r2.BrowserVersion &&
		r1.ReleaseDate.Equal(r2.ReleaseDate)
}
