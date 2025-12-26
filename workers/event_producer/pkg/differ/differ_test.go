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
	v1 "github.com/GoogleChrome/webstatus.dev/lib/workertypes/featurediff/v1"
	snapshotV1 "github.com/GoogleChrome/webstatus.dev/lib/workertypes/featurelistsnapshot/v1"
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
	payload := snapshotV1.FeatureListSnapshotV1{
		Metadata: snapshotV1.StateMetadataV1{GeneratedAt: time.Now(),
			SearchID:       searchID,
			QuerySignature: query,
			ID:             "prev_state_123",
			EventID:        "",
		},
		Data: snapshotV1.FeatureListDataV1{
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
	return string(mustGenerateBlob(t, snapshotV1.FeatureListSnapshotV1{
		Metadata: snapshotV1.StateMetadataV1{
			GeneratedAt:    runTime,
			SearchID:       searchID,
			ID:             "test-state-id",
			EventID:        eventID,
			QuerySignature: query,
		},
		Data: snapshotV1.FeatureListDataV1{
			Features: toSnapshot(features),
		},
	}))
}

// expectDiffBlob generates the expected JSON string for the Diff Blob.
func expectDiffBlob(
	t *testing.T, runTime time.Time, searchID, prevStateID, eventID string, diff v1.FeatureDiffV1,
) string {
	// Need to Sort the diff before serializing for consistent ordering
	diff.Sort()

	return string(mustGenerateBlob(t, v1.FeatureDiffSnapshotV1{
		Metadata: v1.DiffMetadataV1{
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

// TestRun is for full integration testing of the differ.Run function.
// This test is critical for verifying the end-to-end behavior of the diffing engine,
// including cold starts, data updates, query changes, and historical reconciliation (moves/splits).
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
				DiffBlob: expectDiffBlob(t, runTime, searchID, "", eventID, v1.FeatureDiffV1{
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
				DiffBlob: expectDiffBlob(t, runTime, searchID, "prev_state_123", eventID, v1.FeatureDiffV1{
					QueryChanged: false,
					Added:        nil,
					Removed:      nil,
					Modified: []v1.FeatureModified{
						{
							ID:   "1",
							Name: "Grid",
							BaselineChange: &v1.Change[v1.BaselineState]{
								From: v1.BaselineState{
									Status:   backend.Limited,
									LowDate:  nil,
									HighDate: nil,
								},
								To: v1.BaselineState{
									Status:   backend.Widely,
									LowDate:  &lowDate,
									HighDate: &highDate,
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
				DiffBlob: expectDiffBlob(t, runTime, searchID, "prev_state_123", eventID, v1.FeatureDiffV1{
					QueryChanged: true,
					Added:        nil,
					Removed:      nil,
					Modified: []v1.FeatureModified{
						{
							ID:   "1",
							Name: "OldFeature",
							BaselineChange: &v1.Change[v1.BaselineState]{
								From: v1.BaselineState{
									Status:   backend.Limited,
									LowDate:  nil,
									HighDate: nil,
								},
								To: v1.BaselineState{
									Status:   backend.Widely,
									LowDate:  &lowDate,
									HighDate: &highDate,
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
				DiffBlob: expectDiffBlob(t, runTime, searchID, "prev_state_123", eventID, v1.FeatureDiffV1{
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
				DiffBlob: expectDiffBlob(t, runTime, searchID, "prev_state_123", eventID, v1.FeatureDiffV1{
					QueryChanged: false,
					Added:        nil,
					Removed:      nil,
					Modified:     nil,
					Moves: []v1.FeatureMoved{
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
			d := NewFeatureDiffer(tc.mock, v1.NewComparatorV1(tc.mock))
			d.idGen = &mockIDGenerator{}
			d.now = func() time.Time { return runTime }

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
