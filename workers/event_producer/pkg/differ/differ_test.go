// Copyright 2025 Google LLC
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

package differ

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/blobtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/oapi-codegen/runtime/types"
)

// mockFetcher implements FeatureFetcher for testing.
type mockFetcher struct {
	// query -> features
	queryResults map[string][]backend.Feature
	// featureID -> result
	featureDetails map[string]*backendtypes.GetFeatureResult
	// featureID -> error (for GetFeature)
	featureErrors map[string]error
	// specific error for FetchFeatures
	fetchError error
}

func (m *mockFetcher) FetchFeatures(_ context.Context, query string) ([]backend.Feature, error) {
	if m.fetchError != nil {
		return nil, m.fetchError
	}
	// If query key is missing in the map, simulate an error if needed,
	// or return empty slice.
	// For testing "Flush Failed", we want to simulate an error for a specific query.
	// We can use a convention: if query starts with "error:", return error.
	if query == "error:old" {
		return nil, errors.New("simulated fetch error")
	}

	return m.queryResults[query], nil
}

func (m *mockFetcher) GetFeature(_ context.Context, id string) (*backendtypes.GetFeatureResult, error) {
	if err := m.featureErrors[id]; err != nil {
		return nil, err
	}
	if res, ok := m.featureDetails[id]; ok {
		return res, nil
	}
	// Default to exists (Regular) if not specified, to prevent accidental "Deleted" detection in integration tests
	return backendtypes.NewGetFeatureResult(
		backendtypes.NewRegularFeatureResult(&backend.Feature{
			FeatureId:              id,
			Name:                   "",
			Spec:                   nil,
			Baseline:               nil,
			BrowserImplementations: nil,
			Discouraged:            nil,
			Usage:                  nil,
			Wpt:                    nil,
			VendorPositions:        nil,
			DeveloperSignals:       nil,
		}),
	), nil
}

// Helper to construct a backend.Feature with minimal fields.
func makeFeature(id, name, status string) backend.Feature {
	s := backend.BaselineInfoStatus(status)

	return backend.Feature{
		FeatureId:              id,
		Name:                   name,
		Spec:                   nil,
		Discouraged:            nil,
		Usage:                  nil,
		Wpt:                    nil,
		VendorPositions:        nil,
		DeveloperSignals:       nil,
		BrowserImplementations: nil,
		Baseline: &backend.BaselineInfo{
			Status:   &s,
			LowDate:  nil,
			HighDate: nil,
		},
	}
}

// Helper to create a previous state blob using the real serialization logic.
func makeStateBlob(t *testing.T, searchID, query string, features []backend.Feature) []byte {
	snapshot := toSnapshot(features)
	payload := FeatureListSnapshot{
		Metadata: StateMetadata{
			GeneratedAt:    time.Now(),
			SearchID:       searchID,
			QuerySignature: query,
		},
		Data: FeatureListData{
			Features: snapshot,
		},
	}
	b, err := blobtypes.NewBlob(payload)
	if err != nil {
		t.Fatalf("failed to create state blob: %v", err)
	}

	return b
}

func TestRun(t *testing.T) {
	ctx := context.Background()
	searchID := "search-123"

	tests := []struct {
		name         string
		query        string
		oldStateBlob []byte
		mock         *mockFetcher
		wantDiff     *FeatureDiff
		wantWrite    bool
		wantErr      bool
	}{
		{
			name:         "Cold Start",
			query:        "group:css",
			oldStateBlob: nil,
			mock: &mockFetcher{
				queryResults: map[string][]backend.Feature{
					"group:css": {makeFeature("1", "Grid", "limited")},
				},
				featureDetails: nil,
				featureErrors:  nil,
				fetchError:     nil,
			},
			wantDiff: &FeatureDiff{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			wantWrite: true,
			wantErr:   false,
		}, {
			name:  "No Changes",
			query: "group:css",
			oldStateBlob: makeStateBlob(t, searchID, "group:css", []backend.Feature{
				makeFeature("1", "Grid", "limited"),
			}),
			mock: &mockFetcher{
				queryResults: map[string][]backend.Feature{
					"group:css": {makeFeature("1", "Grid", "limited")},
				},
				featureDetails: nil,
				featureErrors:  nil,
				fetchError:     nil,
			},
			wantDiff:  nil,
			wantWrite: false,
			wantErr:   false,
		},
		{
			name:  "Data Update",
			query: "group:css",
			oldStateBlob: makeStateBlob(t, searchID, "group:css", []backend.Feature{
				makeFeature("1", "Grid", "limited"),
			}),
			mock: &mockFetcher{
				queryResults: map[string][]backend.Feature{
					"group:css": {makeFeature("1", "Grid", "widely")},
				},
				featureDetails: nil,
				featureErrors:  nil,
				fetchError:     nil,
			},
			wantDiff: &FeatureDiff{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified: []FeatureModified{
					{
						ID:   "1",
						Name: "Grid",
						BaselineChange: &Change[backend.BaselineInfoStatus]{
							From: backend.Limited,
							To:   backend.Widely,
						},
						NameChange:     nil,
						BrowserChanges: nil,
					},
				},
				Moves:  nil,
				Splits: nil,
			},
			wantWrite: true,
			wantErr:   false,
		},
		{
			name:  "Query Change (Flush Success)",
			query: "group:new",
			oldStateBlob: makeStateBlob(t, searchID, "group:old", []backend.Feature{
				makeFeature("1", "OldFeature", "limited"),
			}),
			mock: &mockFetcher{
				queryResults: map[string][]backend.Feature{
					"group:new": {},
					// Old query shows update happened before switch
					"group:old": {makeFeature("1", "OldFeature", "widely")},
				},
				featureDetails: nil,
				featureErrors:  nil,
				fetchError:     nil,
			},
			wantDiff: &FeatureDiff{
				QueryChanged: true,
				Added:        nil,
				Removed:      nil,
				Modified: []FeatureModified{
					{
						ID:   "1",
						Name: "OldFeature",
						BaselineChange: &Change[backend.BaselineInfoStatus]{
							From: backend.Limited,
							To:   backend.Widely,
						},
						NameChange:     nil,
						BrowserChanges: nil,
					},
				},
				Moves:  nil,
				Splits: nil,
			},
			wantWrite: true,
			wantErr:   false,
		},
		{
			name:  "Query Change (Flush Failed)",
			query: "group:new",
			// Old state has a query "error:old" which will trigger mock error
			oldStateBlob: makeStateBlob(t, searchID, "error:old", []backend.Feature{
				makeFeature("1", "OldFeature", "limited"),
			}),
			mock: &mockFetcher{
				queryResults: map[string][]backend.Feature{
					"group:new": {makeFeature("2", "NewFeature", "limited")},
				},
				featureDetails: nil,
				featureErrors:  nil,
				fetchError:     nil,
			},
			// Expectation: Fallback logic kicks in. We skip diffing the data.
			// Result is just QueryChanged=true, with NO removals/adds reported.
			wantDiff: &FeatureDiff{
				QueryChanged: true,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			wantWrite: true,
			wantErr:   false,
		},
		{
			name:  "Reconciliation (Move)",
			query: "group:css",
			oldStateBlob: makeStateBlob(t, searchID, "group:css", []backend.Feature{
				makeFeature("old-id", "Old Name", "limited"),
			}),
			mock: &mockFetcher{
				queryResults: map[string][]backend.Feature{
					"group:css": {makeFeature("new-id", "New Name", "limited")},
				},
				featureDetails: map[string]*backendtypes.GetFeatureResult{
					"old-id": backendtypes.NewGetFeatureResult(
						backendtypes.NewMovedFeatureResult("new-id"),
					),
				},
				featureErrors: nil,
				fetchError:    nil,
			},
			wantDiff: &FeatureDiff{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves: []FeatureMoved{
					{FromID: "old-id", ToID: "new-id", FromName: "Old Name", ToName: "New Name"},
				},
				Splits: nil,
			},
			wantWrite: true,
			wantErr:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewFeatureDiffer(tc.mock)

			newState, diff, shouldWrite, err := d.Run(ctx, searchID, tc.query, tc.oldStateBlob)

			// Helper 1: Verify Error state
			if checkRunError(t, tc.wantErr, err) {
				return
			}

			// Helper 2: Verify Diff output
			checkRunDiff(t, tc.wantDiff, diff)

			// Verify Write logic
			if shouldWrite != tc.wantWrite {
				t.Errorf("shouldWrite = %v, want %v", shouldWrite, tc.wantWrite)
			}
			if tc.wantWrite && len(newState) == 0 {
				t.Error("Expected newState bytes, got empty")
			}
		})
	}
}

// checkRunError handles verifying if an error occurred vs if one was expected.
// Returns true if the test should stop (error mismatch found or error was expected and received).
func checkRunError(t *testing.T, wantErr bool, err error) bool {
	t.Helper()
	if wantErr {
		if err == nil {
			t.Fatal("Run() expected error, got nil")
		}

		return true
	}
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	return false
}

// checkRunDiff handles comparing the expected diff with the actual diff.
func checkRunDiff(t *testing.T, wantDiff, gotDiff *FeatureDiff) {
	t.Helper()
	if wantDiff != nil {
		if gotDiff == nil {
			t.Fatal("Expected diff, got nil")
		}
		// Sort for deterministic comparison
		gotDiff.Sort()
		wantDiff.Sort()

		if d := cmp.Diff(wantDiff, gotDiff, cmpopts.EquateEmpty()); d != "" {
			t.Errorf("Diff mismatch (-want +got):\n%s", d)
		}
	} else {
		if gotDiff != nil {
			t.Errorf("Expected nil diff, got: %+v", gotDiff)
		}
	}
}

func TestToComparable(t *testing.T) {
	avail := backend.Available
	unavail := backend.Unavailable
	status := backend.Widely
	date := types.Date{Time: time.Now()}

	tests := []struct {
		name string
		in   backend.Feature
		want ComparableFeature
	}{
		{
			name: "Fully Populated",
			in: backend.Feature{
				FeatureId:   "feat-1",
				Name:        "Feature One",
				Spec:        nil,
				Discouraged: nil,
				Usage:       nil,
				Wpt:         nil,
				Baseline: &backend.BaselineInfo{
					Status:   &status,
					LowDate:  nil,
					HighDate: nil,
				},
				BrowserImplementations: &map[string]backend.BrowserImplementation{
					"chrome":  {Status: &avail, Date: &date, Version: nil},
					"firefox": {Status: &unavail, Date: nil, Version: nil},
					"safari":  {Status: &avail, Date: nil, Version: nil},
					"unknown": {Status: &avail, Date: nil, Version: nil}, // Should be ignored
				},
				VendorPositions:  nil,
				DeveloperSignals: nil,
			},
			want: createExpectedFeature("feat-1", "Feature One", backend.Widely, map[backend.SupportedBrowsers]string{
				backend.Chrome:         "available",
				backend.ChromeAndroid:  "",
				backend.Firefox:        "unavailable",
				backend.FirefoxAndroid: "",
				backend.Safari:         "available",
				backend.SafariIos:      "",
				backend.Edge:           "",
			}),
		},
		{
			name: "Minimal (Nil Maps)",
			in: backend.Feature{
				FeatureId:              "feat-2",
				Name:                   "Minimal Feature",
				Baseline:               nil,
				Spec:                   nil,
				BrowserImplementations: nil,
				Discouraged:            nil,
				Usage:                  nil,
				Wpt:                    nil,
				VendorPositions:        nil,
				DeveloperSignals:       nil,
			},
			want: createExpectedFeature("feat-2", "Minimal Feature", backend.Limited, nil),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := toComparable(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("toComparable mismatch.\nGot:  %+v\nWant: %+v", got, tc.want)
			}
		})
	}
}

// createExpectedFeature constructs a ComparableFeature with all OptionallySet fields initialized.
// This is required to pass the exhaustruct linter in tests.
func createExpectedFeature(id, name string, baseline backend.BaselineInfoStatus,
	browsers map[backend.SupportedBrowsers]string) ComparableFeature {
	cf := ComparableFeature{
		ID:             id,
		Name:           OptionallySet[string]{Value: name, IsSet: true},
		BaselineStatus: OptionallySet[backend.BaselineInfoStatus]{Value: baseline, IsSet: true},
		BrowserImpls: BrowserImplementations{
			// Initialize all to IsSet=false by default
			Chrome:         OptionallySet[string]{IsSet: false, Value: ""},
			ChromeAndroid:  OptionallySet[string]{IsSet: false, Value: ""},
			Edge:           OptionallySet[string]{IsSet: false, Value: ""},
			Firefox:        OptionallySet[string]{IsSet: false, Value: ""},
			FirefoxAndroid: OptionallySet[string]{IsSet: false, Value: ""},
			Safari:         OptionallySet[string]{IsSet: false, Value: ""},
			SafariIos:      OptionallySet[string]{IsSet: false, Value: ""},
		},
	}

	// Override specific browsers if provided
	if browsers != nil {
		setIfPresent(browsers, "chrome", &cf.BrowserImpls.Chrome)
		setIfPresent(browsers, "chrome_android", &cf.BrowserImpls.ChromeAndroid)
		setIfPresent(browsers, "edge", &cf.BrowserImpls.Edge)
		setIfPresent(browsers, "firefox", &cf.BrowserImpls.Firefox)
		setIfPresent(browsers, "firefox_android", &cf.BrowserImpls.FirefoxAndroid)
		setIfPresent(browsers, "safari", &cf.BrowserImpls.Safari)
		setIfPresent(browsers, "safari_ios", &cf.BrowserImpls.SafariIos)
	}

	return cf
}

func setIfPresent[K comparable, V any](m map[K]V, key K, target *OptionallySet[V]) {
	var zero V
	if val, ok := m[key]; ok && !reflect.DeepEqual(zero, val) {
		target.IsSet = true
		target.Value = val
	}
}
