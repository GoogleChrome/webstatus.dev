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
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/blobtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
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

type testFeatureOption func(*backend.Feature)

func withHighBaselineStatus(lowDate, highDate time.Time) testFeatureOption {
	val := backend.Widely

	return func(f *backend.Feature) {
		f.Baseline = &backend.BaselineInfo{
			Status: &val,
			LowDate: &types.Date{
				Time: lowDate,
			},
			HighDate: &types.Date{
				Time: highDate,
			},
		}
	}
}

func withLimitedBaselineStatus() testFeatureOption {
	val := backend.Limited

	return func(f *backend.Feature) {
		f.Baseline = &backend.BaselineInfo{
			Status:   &val,
			LowDate:  nil,
			HighDate: nil,
		}
	}
}

// Helper to construct a backend.Feature with minimal fields.
func makeFeature(id, name string, opts ...testFeatureOption) backend.Feature {
	f := backend.Feature{
		FeatureId:              id,
		Name:                   name,
		Spec:                   nil,
		Discouraged:            nil,
		Usage:                  nil,
		Wpt:                    nil,
		VendorPositions:        nil,
		DeveloperSignals:       nil,
		BrowserImplementations: nil,
		Baseline:               nil,
	}
	for _, opt := range opts {
		opt(&f)
	}

	return f
}

// Helper to create a previous state blob using the real serialization logic.
func makeStateBlob(t *testing.T, searchID, query string, features []backend.Feature) []byte {
	snapshot := toSnapshot(features)
	payload := FeatureListSnapshot{
		Metadata: StateMetadata{
			GeneratedAt:    time.Now(),
			SearchID:       searchID,
			QuerySignature: query,
			ID:             "prev_state_123",
			EventID:        "",
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

func mustGenerateBlob[T blobtypes.Payload](t *testing.T, v T) []byte {
	b, err := blobtypes.NewBlob(v)
	if err != nil {
		t.Fatalf("failed to create blob: %v", err)
	}

	return b
}

// expectStateBlob generates the expected JSON string for the State Blob.
func expectStateBlob(t *testing.T, runTime time.Time, searchID, query, eventID string,
	features []backend.Feature) string {
	return string(mustGenerateBlob(t, FeatureListSnapshot{
		Metadata: StateMetadata{
			GeneratedAt:    runTime,
			SearchID:       searchID,
			ID:             "test-state-id",
			EventID:        eventID,
			QuerySignature: query,
		},
		Data: FeatureListData{
			Features: toSnapshot(features),
		},
	}))
}

// expectDiffBlob generates the expected JSON string for the Diff Blob.
func expectDiffBlob(t *testing.T, runTime time.Time, searchID, prevStateID, eventID string, diff FeatureDiffV1) string {
	// Need to Sort the diff before serializing for consistent ordering
	diff.Sort()

	return string(mustGenerateBlob(t, FeatureDiffSnapshotV1{
		Metadata: DiffMetadataV1{
			GeneratedAt:     runTime,
			EventID:         eventID,
			ID:              "test-diff-id",
			SearchID:        searchID,
			PreviousStateID: prevStateID,
			NewStateID:      "test-state-id",
		},
		Data: diff,
	}))
}

// ExpectedDiffResult defines the expected outcome of a Run.
// Blobs are represented as JSON strings to allow for flexible structure comparison.
type ExpectedDiffResult struct {
	HasChanges bool
	Reasons    []string
	Summary    workertypes.EventSummary
	StateID    string
	DiffID     string
	// JSON strings representing the expected blob content.
	// These will be compared against the actual bytes by decoding both into interface{}.
	StateBlob string
	DiffBlob  string
}

// mockIDGenerator for deterministic tests.
type mockIDGenerator struct{}

func (m *mockIDGenerator) NewStateID() string { return "test-state-id" }
func (m *mockIDGenerator) NewDiffID() string  { return "test-diff-id" }

func TestRun(t *testing.T) {
	ctx := context.Background()
	searchID := "search-123"
	lowDate := time.Date(2000, time.April, 1, 1, 1, 1, 1, time.UTC)
	highDate := time.Date(2000, time.April, 2, 1, 1, 1, 1, time.UTC)
	runTime := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	eventID := "eventID"

	tests := []struct {
		name         string
		query        string
		oldStateBlob []byte
		mock         *mockFetcher
		want         ExpectedDiffResult
		wantErr      bool
	}{
		{
			name:         "Cold Start",
			query:        "group:css",
			oldStateBlob: nil,
			mock: &mockFetcher{
				queryResults: map[string][]backend.Feature{
					"group:css": {makeFeature("1", "Grid", withLimitedBaselineStatus())},
				},
				featureDetails: nil,
				featureErrors:  nil,
				fetchError:     nil,
			},
			want: ExpectedDiffResult{
				HasChanges: true,
				Reasons:    nil,
				Summary: workertypes.EventSummary{
					SchemaVersion: workertypes.VersionEventSummaryV1,
					Text:          "No changes detected",
					Categories: workertypes.SummaryCategories{
						QueryChanged:    0,
						Added:           0,
						Removed:         0,
						Moved:           0,
						Split:           0,
						Updated:         0,
						UpdatedImpl:     0,
						UpdatedRename:   0,
						UpdatedBaseline: 0,
					},
				},
				StateID: "test-state-id",
				DiffID:  "test-diff-id",
				StateBlob: expectStateBlob(t, runTime, searchID, "group:css", eventID, []backend.Feature{
					makeFeature("1", "Grid", withLimitedBaselineStatus()),
				}),
				DiffBlob: expectDiffBlob(t, runTime, searchID, "", eventID, FeatureDiffV1{
					QueryChanged: false,
					Added:        nil,
					Removed:      nil,
					Modified:     nil,
					Moves:        nil,
					Splits:       nil,
				}),
			},
			wantErr: false,
		}, {
			name:  "No Changes",
			query: "group:css",
			oldStateBlob: makeStateBlob(t, searchID, "group:css", []backend.Feature{
				makeFeature("1", "Grid", withLimitedBaselineStatus()),
			}),
			mock: &mockFetcher{
				queryResults: map[string][]backend.Feature{
					"group:css": {makeFeature("1", "Grid", withLimitedBaselineStatus())},
				},
				featureDetails: nil,
				featureErrors:  nil,
				fetchError:     nil,
			},
			want: ExpectedDiffResult{
				HasChanges: false,
				Reasons:    nil,
				Summary: workertypes.EventSummary{
					SchemaVersion: workertypes.VersionEventSummaryV1,
					Text:          "",
					Categories: workertypes.SummaryCategories{
						QueryChanged:    0,
						Added:           0,
						Removed:         0,
						Moved:           0,
						Split:           0,
						Updated:         0,
						UpdatedImpl:     0,
						UpdatedRename:   0,
						UpdatedBaseline: 0,
					},
				},
				StateID:   "",
				DiffID:    "",
				StateBlob: "",
				DiffBlob:  "",
			},
			wantErr: false,
		},
		{
			name:  "Data Update",
			query: "group:css",
			oldStateBlob: makeStateBlob(t, searchID, "group:css", []backend.Feature{
				makeFeature("1", "Grid", withLimitedBaselineStatus()),
			}),
			mock: &mockFetcher{
				queryResults: map[string][]backend.Feature{
					"group:css": {makeFeature("1", "Grid", withHighBaselineStatus(lowDate, highDate))},
				},
				featureDetails: nil,
				featureErrors:  nil,
				fetchError:     nil,
			},
			want: ExpectedDiffResult{
				HasChanges: true,
				Reasons:    []string{"DATA_UPDATED"},
				StateID:    "test-state-id",
				DiffID:     "test-diff-id",
				Summary: workertypes.EventSummary{
					SchemaVersion: workertypes.VersionEventSummaryV1,
					Text:          "1 features updated",
					Categories: workertypes.SummaryCategories{
						QueryChanged:    0,
						Added:           0,
						Removed:         0,
						Moved:           0,
						Split:           0,
						Updated:         1,
						UpdatedImpl:     0,
						UpdatedRename:   0,
						UpdatedBaseline: 1,
					},
				},
				StateBlob: expectStateBlob(t, runTime, searchID, "group:css", eventID, []backend.Feature{
					makeFeature("1", "Grid", withHighBaselineStatus(lowDate, highDate)),
				}),
				DiffBlob: expectDiffBlob(t, runTime, searchID, "prev_state_123", eventID, FeatureDiffV1{
					QueryChanged: false,
					Added:        nil,
					Removed:      nil,
					Modified: []FeatureModified{
						{
							ID:   "1",
							Name: "Grid",
							BaselineChange: &Change[BaselineState]{
								From: BaselineState{
									Status:   ptrToSet(backend.Limited),
									LowDate:  ptrToSet[*time.Time](nil),
									HighDate: ptrToSet[*time.Time](nil),
								},
								To: BaselineState{
									Status:   ptrToSet(backend.Widely),
									LowDate:  ptrToSet(&lowDate),
									HighDate: ptrToSet(&highDate),
								},
							},
							NameChange:     nil,
							BrowserChanges: nil,
							Docs:           nil,
							DocsChange:     nil,
						},
					},
					Moves:  nil,
					Splits: nil,
				}),
			},
			wantErr: false,
		},
		{
			name:  "Query Change (Flush Success)",
			query: "group:new",
			oldStateBlob: makeStateBlob(t, searchID, "group:old", []backend.Feature{
				makeFeature("1", "OldFeature", withLimitedBaselineStatus()),
			}),
			mock: &mockFetcher{
				queryResults: map[string][]backend.Feature{
					"group:new": {},
					// Old query shows update happened before switch
					"group:old": {makeFeature("1", "OldFeature", withHighBaselineStatus(lowDate, highDate))},
				},
				featureDetails: nil,
				featureErrors:  nil,
				fetchError:     nil,
			},
			want: ExpectedDiffResult{
				HasChanges: true,
				Reasons:    []string{"DATA_UPDATED", "QUERY_EDITED"},
				StateID:    "test-state-id",
				DiffID:     "test-diff-id",
				Summary: workertypes.EventSummary{
					SchemaVersion: workertypes.VersionEventSummaryV1,
					Text:          "Search criteria updated, 1 features updated",
					Categories: workertypes.SummaryCategories{
						QueryChanged:    1,
						Added:           0,
						Removed:         0,
						Moved:           0,
						Split:           0,
						Updated:         1,
						UpdatedImpl:     0,
						UpdatedRename:   0,
						UpdatedBaseline: 1,
					},
				},
				StateBlob: expectStateBlob(t, runTime, searchID, "group:new", eventID, []backend.Feature{}),
				DiffBlob: expectDiffBlob(t, runTime, searchID, "prev_state_123", eventID, FeatureDiffV1{
					QueryChanged: true,
					Added:        nil,
					Removed:      nil,
					Modified: []FeatureModified{
						{
							ID:   "1",
							Name: "OldFeature",
							BaselineChange: &Change[BaselineState]{
								From: BaselineState{
									Status:   ptrToSet(backend.Limited),
									LowDate:  ptrToSet[*time.Time](nil),
									HighDate: ptrToSet[*time.Time](nil),
								},
								To: BaselineState{
									Status:   ptrToSet(backend.Widely),
									LowDate:  ptrToSet(&lowDate),
									HighDate: ptrToSet(&highDate),
								},
							},
							NameChange:     nil,
							BrowserChanges: nil,
							Docs:           nil,
							DocsChange:     nil,
						},
					},
					Moves:  nil,
					Splits: nil,
				}),
			},
			wantErr: false,
		},
		{
			name:  "Query Change (Flush Failed)",
			query: "group:new",
			// Old state has a query "error:old" which will trigger mock error
			oldStateBlob: makeStateBlob(t, searchID, "error:old", []backend.Feature{
				makeFeature("1", "OldFeature", withLimitedBaselineStatus()),
			}),
			mock: &mockFetcher{
				queryResults: map[string][]backend.Feature{
					"group:new": {makeFeature("2", "NewFeature", withLimitedBaselineStatus())},
				},
				featureDetails: nil,
				featureErrors:  nil,
				fetchError:     nil,
			},
			// Expectation: Fallback logic kicks in. We skip diffing the data.
			// Result is just QueryChanged=true, with NO removals/adds reported.
			want: ExpectedDiffResult{
				HasChanges: true,
				Reasons:    []string{"QUERY_EDITED"},
				StateID:    "test-state-id",
				DiffID:     "test-diff-id",
				Summary: workertypes.EventSummary{
					SchemaVersion: workertypes.VersionEventSummaryV1,
					Text:          "Search criteria updated",
					Categories: workertypes.SummaryCategories{
						QueryChanged:    1,
						Added:           0,
						Removed:         0,
						Moved:           0,
						Split:           0,
						Updated:         0,
						UpdatedImpl:     0,
						UpdatedRename:   0,
						UpdatedBaseline: 0,
					},
				},
				StateBlob: expectStateBlob(t, runTime, searchID, "group:new", eventID, []backend.Feature{
					makeFeature("2", "NewFeature", withLimitedBaselineStatus()),
				}),
				DiffBlob: expectDiffBlob(t, runTime, searchID, "prev_state_123", eventID, FeatureDiffV1{
					QueryChanged: true,
					Added:        nil,
					Removed:      nil,
					Modified:     nil,
					Moves:        nil,
					Splits:       nil,
				}),
			},
			wantErr: false,
		},
		{
			name:  "Reconciliation (Move)",
			query: "group:css",
			oldStateBlob: makeStateBlob(t, searchID, "group:css", []backend.Feature{
				makeFeature("old-id", "Old Name", withLimitedBaselineStatus()),
			}),
			mock: &mockFetcher{
				queryResults: map[string][]backend.Feature{
					"group:css": {makeFeature("new-id", "New Name", withLimitedBaselineStatus())},
				},
				featureDetails: map[string]*backendtypes.GetFeatureResult{
					"old-id": backendtypes.NewGetFeatureResult(
						backendtypes.NewMovedFeatureResult("new-id"),
					),
				},
				featureErrors: nil,
				fetchError:    nil,
			},
			want: ExpectedDiffResult{
				HasChanges: true,
				Reasons:    []string{"DATA_UPDATED"},
				StateID:    "test-state-id",
				DiffID:     "test-diff-id",
				Summary: workertypes.EventSummary{
					SchemaVersion: workertypes.VersionEventSummaryV1,
					Text:          "1 features moved/renamed",
					Categories: workertypes.SummaryCategories{
						QueryChanged:    0,
						Added:           0,
						Removed:         0,
						Moved:           1,
						Split:           0,
						Updated:         0,
						UpdatedImpl:     0,
						UpdatedRename:   0,
						UpdatedBaseline: 0,
					},
				},
				StateBlob: expectStateBlob(t, runTime, searchID, "group:css", eventID, []backend.Feature{
					makeFeature("new-id", "New Name", withLimitedBaselineStatus()),
				}),
				DiffBlob: expectDiffBlob(t, runTime, searchID, "prev_state_123", eventID, FeatureDiffV1{
					QueryChanged: false,
					Added:        nil,
					Removed:      nil,
					Modified:     nil,
					Moves: []FeatureMoved{
						{FromID: "old-id", ToID: "new-id", FromName: "Old Name", ToName: "New Name"},
					},
					Splits: nil,
				}),
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewFeatureDiffer(tc.mock)
			d.now = func() time.Time { return runTime }
			d.idGen = &mockIDGenerator{}

			result, err := d.Run(ctx, searchID, tc.query, eventID, tc.oldStateBlob)

			// Helper 1: Verify Error state
			if checkRunError(t, tc.wantErr, err) {
				return
			}

			if result.HasChanges != tc.want.HasChanges {
				t.Errorf("HasChanges = %v, want %v", result.HasChanges, tc.want.HasChanges)
			}

			if !tc.want.HasChanges {
				return
			}

			// Validate Metadata
			if result.StateID != tc.want.StateID {
				t.Errorf("StateID mismatch. Got %s, Want %s", result.StateID, tc.want.StateID)
			}
			if result.DiffID != tc.want.DiffID {
				t.Errorf("DiffID mismatch. Got %s, Want %s", result.DiffID, tc.want.DiffID)
			}

			// Reasons: sort before compare
			less := func(a, b string) bool { return a < b }
			if d := cmp.Diff(tc.want.Reasons, result.Reasons, cmpopts.SortSlices(less), cmpopts.EquateEmpty()); d != "" {
				t.Errorf("Reasons mismatch (-want +got):\n%s", d)
			}
			// Summary
			if d := cmp.Diff(tc.want.Summary, result.Summary, cmpopts.EquateEmpty()); d != "" {
				t.Errorf("Summary mismatch (-want +got):\n%s", d)
			}

			// Validate Blobs via flexible JSON comparison
			compareJSON(t, "StateBytes", tc.want.StateBlob, result.StateBytes)
			compareJSON(t, "DiffBytes", tc.want.DiffBlob, result.DiffBytes)
		})
	}
}

// compareJSON unmarshals both inputs into interface{} containers and compares them using cmp.Diff.
func compareJSON(t *testing.T, name string, wantStr string, gotBytes []byte) {
	t.Helper()
	var want, got interface{}

	if wantStr == "" {
		t.Fatalf("%s expectation is empty, but got bytes", name)
	}

	if err := json.Unmarshal([]byte(wantStr), &want); err != nil {
		t.Fatalf("%s: failed to unmarshal expected JSON string: %v", name, err)
	}
	if err := json.Unmarshal(gotBytes, &got); err != nil {
		t.Fatalf("%s: failed to unmarshal actual bytes: %v", name, err)
	}

	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("%s mismatch (-want +got):\n%s", name, d)
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
			want: createExpectedFeature("feat-1", "Feature One", backend.Widely,
				map[backend.SupportedBrowsers]OptionallySet[BrowserState]{
					backend.Chrome: {Value: BrowserState{
						Status:  OptionallySet[backend.BrowserImplementationStatus]{Value: "available", IsSet: true},
						Date:    OptionallySet[*time.Time]{Value: &date.Time, IsSet: true},
						Version: OptionallySet[*string]{Value: nil, IsSet: true},
					}, IsSet: true},
					backend.ChromeAndroid: unsetBrowserState(),
					backend.Firefox: {Value: BrowserState{
						Status: OptionallySet[backend.BrowserImplementationStatus]{Value: "unavailable", IsSet: true},
						Date:   OptionallySet[*time.Time]{Value: nil, IsSet: true},
						Version: OptionallySet[*string]{
							Value: nil, IsSet: true,
						},
					}, IsSet: true},
					backend.FirefoxAndroid: unsetBrowserState(),
					backend.Safari: {
						Value: BrowserState{Status: OptionallySet[backend.BrowserImplementationStatus]{Value: "available", IsSet: true},
							Date:    OptionallySet[*time.Time]{Value: nil, IsSet: true},
							Version: OptionallySet[*string]{Value: nil, IsSet: true},
						}, IsSet: true},
					backend.SafariIos: unsetBrowserState(),
					backend.Edge:      unsetBrowserState(),
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
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("toComparable mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// createExpectedFeature constructs a ComparableFeature with all OptionallySet fields initialized.
func createExpectedFeature(id, name string, baseline backend.BaselineInfoStatus,
	browsers map[backend.SupportedBrowsers]OptionallySet[BrowserState]) ComparableFeature {
	cf := ComparableFeature{
		ID:   id,
		Name: OptionallySet[string]{Value: name, IsSet: true},
		BaselineStatus: OptionallySet[BaselineState]{
			Value: BaselineState{
				Status: OptionallySet[backend.BaselineInfoStatus]{Value: baseline, IsSet: true},
				// Nil is a valid value for LowDate and HighDate
				LowDate:  OptionallySet[*time.Time]{IsSet: true, Value: nil},
				HighDate: OptionallySet[*time.Time]{IsSet: true, Value: nil},
			},
			IsSet: true,
		},
		Docs: docs(true),
		BrowserImpls: OptionallySet[BrowserImplementations]{
			Value: BrowserImplementations{
				Chrome:         unsetBrowserState(),
				ChromeAndroid:  unsetBrowserState(),
				Edge:           unsetBrowserState(),
				Firefox:        unsetBrowserState(),
				FirefoxAndroid: unsetBrowserState(),
				Safari:         unsetBrowserState(),
				SafariIos:      unsetBrowserState(),
			},
			IsSet: true,
		},
	}

	for browserKey := range browsers {
		if state, ok := browsers[browserKey]; ok {
			cf.BrowserImpls.Value.setBrowserState(browserKey, state)
		}
	}

	return cf
}

// TestSnapshotSerialization validates that our ComparableFeature struct marshals into the exact JSON
// format we expect for GCS storage.
//
// Importance:
// 1. Storage Efficiency: Verifies that 'omitzero' (via IsZero) correctly removes IsSet=false fields.
// 2. Data Integrity: Verifies that explicit nulls (IsSet=true, Value=nil) are preserved as "null".
//
// Developer Guide:
//   - **Adding a Field:** Add a test case here to verify the new field is omitted when IsSet=false
//     and serialized correctly when IsSet=true. This ensures the *current* write path is correct.
//   - **Breaking Change (Rename/Type Change):** Update these test expectations to match your new schema.
//     This test validates the *new* format you intend to write.
func TestSnapshotSerialization(t *testing.T) {
	tests := []struct {
		name     string
		input    ComparableFeature
		wantJSON string
	}{
		{
			name: "Explicit Nulls (IsSet=true, Value=nil)",
			input: ComparableFeature{
				ID:   "test-nulls",
				Name: OptionallySet[string]{Value: "", IsSet: false},
				BaselineStatus: OptionallySet[BaselineState]{
					IsSet: true,
					Value: BaselineState{
						Status:   OptionallySet[backend.BaselineInfoStatus]{IsSet: true, Value: backend.Limited},
						LowDate:  OptionallySet[*time.Time]{IsSet: true, Value: nil}, // Expect "lowDate": null
						HighDate: OptionallySet[*time.Time]{IsSet: true, Value: nil}, // Expect "highDate": null
					},
				},
				BrowserImpls: OptionallySet[BrowserImplementations]{Value: BrowserImplementations{
					Chrome:         unsetBrowserState(),
					ChromeAndroid:  unsetBrowserState(),
					Edge:           unsetBrowserState(),
					Firefox:        unsetBrowserState(),
					FirefoxAndroid: unsetBrowserState(),
					Safari:         unsetBrowserState(),
					SafariIos:      unsetBrowserState(),
				}, IsSet: false},
				Docs: docs(false),
			},
			// All other fields (Name, BrowserImpls, Docs) are IsSet=false (zero value), so omitted via omitzero.
			wantJSON: `{"id":"test-nulls","baselineStatus":{"status":"limited","lowDate":null,"highDate":null}}`,
		},
		{
			name: "Omitted Fields (IsSet=false)",
			input: ComparableFeature{
				ID:   "test-omitted",
				Name: OptionallySet[string]{Value: "", IsSet: false},
				BaselineStatus: OptionallySet[BaselineState]{
					IsSet: true,
					Value: BaselineState{
						Status:   OptionallySet[backend.BaselineInfoStatus]{IsSet: true, Value: backend.Limited},
						LowDate:  OptionallySet[*time.Time]{IsSet: false, Value: nil}, // Expect omitted
						HighDate: OptionallySet[*time.Time]{IsSet: false, Value: nil}, // Expect omitted
					},
				},
				BrowserImpls: OptionallySet[BrowserImplementations]{Value: BrowserImplementations{
					Chrome:         unsetBrowserState(),
					ChromeAndroid:  unsetBrowserState(),
					Edge:           unsetBrowserState(),
					Firefox:        unsetBrowserState(),
					FirefoxAndroid: unsetBrowserState(),
					Safari:         unsetBrowserState(),
					SafariIos:      unsetBrowserState(),
				}, IsSet: false},
				Docs: OptionallySet[Docs]{Value: Docs{MdnDocs: OptionallySet[[]MdnDoc]{Value: nil, IsSet: false}}, IsSet: false},
			},
			wantJSON: `{"id":"test-omitted","baselineStatus":{"status":"limited"}}`,
		},
		{
			name: "Top Level Omission",
			input: ComparableFeature{
				ID:   "test-top-omit",
				Name: OptionallySet[string]{Value: "", IsSet: false},
				BaselineStatus: OptionallySet[BaselineState]{IsSet: false, Value: BaselineState{
					Status:   OptionallySet[backend.BaselineInfoStatus]{Value: "", IsSet: false},
					LowDate:  OptionallySet[*time.Time]{Value: nil, IsSet: false},
					HighDate: OptionallySet[*time.Time]{Value: nil, IsSet: false},
				}},
				BrowserImpls: browserImplementations(false),
				Docs:         docs(false),
			},
			wantJSON: `{"id":"test-top-omit"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Marshal
			b, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// 2. Verify JSON String (Omitzero check)
			if got := string(b); got != tc.wantJSON {
				t.Errorf("JSON mismatch.\nGot:  %s\nWant: %s", got, tc.wantJSON)
			}

			// 3. Round Trip (Unmarshal back to struct)
			var restored ComparableFeature
			if err := json.Unmarshal(b, &restored); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// 4. Verify restored state matches input
			// This ensures that "null" in JSON becomes {IsSet:true, Value:nil}
			// and missing field becomes {IsSet:false}.
			if d := cmp.Diff(tc.input, restored); d != "" {
				t.Errorf("Round trip mismatch (-want +got):\n%s", d)
			}
		})
	}
}

func browserImplementations(isSet bool) OptionallySet[BrowserImplementations] {
	return OptionallySet[BrowserImplementations]{Value: BrowserImplementations{
		Chrome:         unsetBrowserState(),
		ChromeAndroid:  unsetBrowserState(),
		Edge:           unsetBrowserState(),
		Firefox:        unsetBrowserState(),
		FirefoxAndroid: unsetBrowserState(),
		Safari:         unsetBrowserState(),
		SafariIos:      unsetBrowserState(),
	}, IsSet: isSet}
}

func docs(isSet bool) OptionallySet[Docs] {
	return OptionallySet[Docs]{Value: Docs{MdnDocs: OptionallySet[[]MdnDoc]{Value: nil, IsSet: false}}, IsSet: isSet}
}

// TestDiffSerialization validates the JSON format of the event payload sent to downstream workers.
//
// Importance:
// 1. Payload Size: Verifies that empty lists (Added, Removed, etc) are omitted to keep messages small.
// 2. Contract: Ensures downstream consumers receive the expected structure.
//
// Developer Guide:
//   - **Adding a Field:** Add a test case to ensure the field is omitted when empty/nil.
//   - **Modifying Structure:** Update these tests to reflect the new contract with downstream workers.
func TestDiffSerialization(t *testing.T) {
	tests := []struct {
		name     string
		input    FeatureDiffV1
		wantJSON string
	}{
		{
			name: "Empty Diff (All Zero)",
			// All slices are nil, bool is false. omitempty handles this.
			input: FeatureDiffV1{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves:        nil,
				Splits:       nil,
			},
			wantJSON: `{}`,
		},
		{
			name: "Partial Diff",
			input: FeatureDiffV1{
				QueryChanged: true,
				Added: []FeatureAdded{
					// Docs is nil (pointer), so it should be omitted.
					{ID: "1", Name: "A", Reason: ReasonNewMatch, Docs: nil},
				},
				Removed:  nil,
				Modified: nil,
				Moves:    nil,
				Splits:   nil,
			},
			wantJSON: `{"queryChanged":true,"added":[{"id":"1","name":"A","reason":"new_match"}]}`,
		},
		{
			name: "Moves Only",
			input: FeatureDiffV1{
				QueryChanged: false,
				Added:        nil,
				Removed:      nil,
				Modified:     nil,
				Moves: []FeatureMoved{
					{FromID: "A", ToID: "B", FromName: "", ToName: ""},
				},
				Splits: nil,
			},
			wantJSON: `{"moves":[{"fromId":"A","toId":"B","fromName":"","toName":""}]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			if got := string(b); got != tc.wantJSON {
				t.Errorf("JSON mismatch.\nGot:  %s\nWant: %s", got, tc.wantJSON)
			}

			var restored FeatureDiffV1
			if err := json.Unmarshal(b, &restored); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if d := cmp.Diff(tc.input, restored); d != "" {
				t.Errorf("Round trip mismatch (-want +got):\n%s", d)
			}
		})
	}
}

// TestSchemaEvolution ensures backward compatibility with historical data.
// Unlike other tests that use helpers to generate input (which evolves with the code),
// this test uses HARDCODED RAW STRINGS representing data structures from the past (V1).
//
// Why this is important:
//  1. Safety: It prevents accidental breaking changes where we rename a Go struct field
//     but forget that terabytes of JSON in GCS still use the old key.
//  2. Evolution: It verifies that our code can gracefully handle missing fields from older blobs
//     (e.g. reading a V1 blob that lacks 'Docs').
//
// Developer Guide:
//   - **Do NOT update/delete existing cases** when changing the schema. These raw strings represent
//     immutable files currently stored in GCS.
//   - **Adding a Field (Non-Breaking):** Existing tests should pass (the new field will be IsSet=false).
//     You generally do not need to bump the EnvelopeVersion for additive changes.
//   - **Renaming/Removing a Field (Breaking):** Existing tests will likely FAIL.
//   - Do NOT fix the test by updating the JSON string (that would falsify history).
//   - FIX THE CODE: Add migration logic (e.g. custom UnmarshalJSON or a migration step in `loadPreviousContext`)
//     to map the old data format to the new struct.
//   - **New Versions:** If you create a fundamentally new storage format (EnvelopeVersion v2),
//     add a new test case here with a raw JSON string matching the v2 format. Keep v1 cases to ensure
//     we can still read old backups.
func TestSchemaEvolution(t *testing.T) {
	ctx := context.Background()
	searchID := "search-123"
	// Mock time for deterministic output
	runTime := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)

	// Defines what we care about for evolution tests:
	// Did we make the right decision based on the old data?
	type expectedEvolutionResult struct {
		HasChanges bool
		Reasons    []string
	}

	tests := []struct {
		name         string
		query        string
		oldStateJSON string // Raw JSON string representing legacy data
		mock         *mockFetcher
		want         expectedEvolutionResult
	}{
		{
			name:  "V1 Blob (Missing Docs) to Current (Has Docs)",
			query: "group:css",
			// V1 Blob: Has ID, Name, BaselineStatus. Missing 'Docs' and 'BrowserImpls'.
			// This simulates a blob written before we added 'Docs' to the struct.
			// NOTE: We must wrap this in the standard envelope (kind/apiVersion) because
			// the loader uses blobtypes.Apply() which expects this structure.
			oldStateJSON: `{
				"kind": "FeatureListSnapshot",
				"apiVersion": "v1",
				"metadata": {
					"generatedAt": "2024-01-01T00:00:00Z",
					"searchId": "search-123",
					"id": "old_state_v1",
					"querySignature": "group:css"
				},
				"data": {
					"features": {
						"feat-1": {
							"id": "feat-1",
							"name": "Grid",
							"baselineStatus": {
								"status": "limited"
							}
						}
					}
				}
			}`,
			mock: &mockFetcher{
				queryResults: map[string][]backend.Feature{
					// Live data has Docs populated (via toComparable)
					"group:css": {makeFeature("feat-1", "Grid", withLimitedBaselineStatus())},
				},
				featureDetails: nil,
				featureErrors:  nil,
				fetchError:     nil,
			},
			want: expectedEvolutionResult{
				HasChanges: false, // Quiet Rollout! New field 'Docs' ignored because missing in Old.
				Reasons:    nil,
			},
		},
		{
			name:  "V1 Blob (Limited) to Current (Widely) [Logic Check]",
			query: "group:css",
			// Same V1 structure (Missing Docs), but we change the Live status to Widely.
			// We expect the system to correctly read "Limited" from the old JSON and compare it.
			oldStateJSON: `{
				"kind": "FeatureListSnapshot",
				"apiVersion": "v1",
				"metadata": {
					"generatedAt": "2024-01-01T00:00:00Z",
					"searchId": "search-123",
					"id": "old_state_v1",
					"querySignature": "group:css"
				},
				"data": {
					"features": {
						"feat-1": {
							"id": "feat-1",
							"name": "Grid",
							"baselineStatus": {
								"status": "limited"
							}
						}
					}
				}
			}`,
			mock: &mockFetcher{
				queryResults: map[string][]backend.Feature{
					// Live data is now WIDELY available
					"group:css": {makeFeature("feat-1", "Grid", withHighBaselineStatus(runTime, runTime))},
				},
				featureDetails: nil,
				featureErrors:  nil,
				fetchError:     nil,
			},
			want: expectedEvolutionResult{
				HasChanges: true,
				Reasons:    []string{"DATA_UPDATED"},
				// We don't check full blobs here (TestRun does that), but checking Reasons
				// confirms the differ successfully read the old state and found the delta.
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewFeatureDiffer(tc.mock)
			d.idGen = &mockIDGenerator{}
			d.now = func() time.Time { return runTime }

			result, err := d.Run(ctx, searchID, tc.query, "eventID", []byte(tc.oldStateJSON))

			if err != nil {
				t.Fatalf("Run failed: %v", err)
			}

			if result.HasChanges != tc.want.HasChanges {
				t.Errorf("HasChanges = %v, want %v", result.HasChanges, tc.want.HasChanges)
			}

			if tc.want.HasChanges {
				// Verify we actually diffed against the loaded data
				if d := cmp.Diff(tc.want.Reasons, result.Reasons,
					cmpopts.SortSlices(func(a, b string) bool { return a < b })); d != "" {
					t.Errorf("Reasons mismatch (-want +got):\n%s", d)
				}
			}
		})
	}
}
