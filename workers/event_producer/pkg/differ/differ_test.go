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
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/generic"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes/comparables"
	"github.com/google/go-cmp/cmp"
	"github.com/oapi-codegen/runtime/types"
)

// Helper to construct a backend.Feature with minimal fields.
func makeFeature(id, name, status string) backend.Feature {
	s := backend.BaselineInfoStatus(status)

	return backend.Feature{
		FeatureId:                  id,
		Name:                       name,
		Spec:                       nil,
		Discouraged:                nil,
		Usage:                      nil,
		Wpt:                        nil,
		VendorPositions:            nil,
		DeveloperSignals:           nil,
		SystemManagedSavedSearchId: nil,
		BrowserImplementations:     nil,
		Baseline: &backend.BaselineInfo{
			Status:   &s,
			LowDate:  nil,
			HighDate: nil,
		},
	}
}

//
// Mocks for testing the generic FeatureDiffer
//

// testDiff is a concrete type for the generic parameter D in our tests.
type testDiff struct {
	Content string
}

type mockStateAdapter struct {
	loadReturns struct {
		snapshot  map[string]comparables.Feature
		id        string
		signature string
		isEmpty   bool
		err       error
	}
	serializeCalledWith struct {
		id        string
		searchID  string
		eventID   string
		query     string
		timestamp time.Time
		snapshot  map[string]comparables.Feature
	}
	serializeReturns struct {
		bytes []byte
		err   error
	}
}

func (m *mockStateAdapter) Load(_ []byte) (map[string]comparables.Feature, string, string, bool, error) {
	return m.loadReturns.snapshot, m.loadReturns.id, m.loadReturns.signature, m.loadReturns.isEmpty, m.loadReturns.err
}

func (m *mockStateAdapter) Serialize(id, searchID, eventID, query string,
	timestamp time.Time, snapshot map[string]comparables.Feature) ([]byte, error) {
	m.serializeCalledWith.id = id
	m.serializeCalledWith.searchID = searchID
	m.serializeCalledWith.eventID = eventID
	m.serializeCalledWith.query = query
	m.serializeCalledWith.timestamp = timestamp
	m.serializeCalledWith.snapshot = snapshot

	return m.serializeReturns.bytes, m.serializeReturns.err
}

type mockDiffSerializer[D any] struct {
	serializeCalledWith struct {
		id              string
		searchID        string
		eventID         string
		newStateID      string
		previousStateID string
		diff            *D
		timestamp       time.Time
	}
	serializeReturns struct {
		bytes []byte
		err   error
	}
}

func (m *mockDiffSerializer[D]) Serialize(id, searchID, eventID, newStateID,
	previousStateID string, diff *D, timestamp time.Time) ([]byte, error) {
	m.serializeCalledWith.id = id
	m.serializeCalledWith.searchID = searchID
	m.serializeCalledWith.eventID = eventID
	m.serializeCalledWith.newStateID = newStateID
	m.serializeCalledWith.previousStateID = previousStateID
	m.serializeCalledWith.diff = diff
	m.serializeCalledWith.timestamp = timestamp

	return m.serializeReturns.bytes, m.serializeReturns.err
}

type mockWorkflow[D any] struct {
	calculateDiffCalled      bool
	reconcileHistoryCalled   bool
	setQueryChangedCalled    bool
	hasRemovedFeaturesResult bool
	hasChangesResult         bool
	hasDataChangesResult     bool
	getDiffResult            *D
	summaryResult            []byte
	summaryError             error
}

func (m *mockWorkflow[D]) CalculateDiff(_, _ map[string]comparables.Feature) {
	m.calculateDiffCalled = true
}
func (m *mockWorkflow[D]) ReconcileHistory(_ context.Context) error {
	m.reconcileHistoryCalled = true

	return nil
}
func (m *mockWorkflow[D]) HasRemovedFeatures() bool { return m.hasRemovedFeaturesResult }
func (m *mockWorkflow[D]) HasChanges() bool         { return m.hasChangesResult }
func (m *mockWorkflow[D]) HasDataChanges() bool     { return m.hasDataChangesResult }
func (m *mockWorkflow[D]) SetQueryChanged(val bool) { m.setQueryChangedCalled = val }
func (m *mockWorkflow[D]) GetDiff() *D              { return m.getDiffResult }
func (m *mockWorkflow[D]) GenerateJSONSummary() ([]byte, error) {
	return m.summaryResult, m.summaryError
}

type mockIDGenerator struct {
	stateID string
	diffID  string
}

func (m *mockIDGenerator) NewStateID() string { return m.stateID }
func (m *mockIDGenerator) NewDiffID() string  { return m.diffID }

type mockFetcher struct {
	queryResults map[string][]backend.Feature
	fetchError   error
}

func (m *mockFetcher) FetchFeatures(_ context.Context, query string) ([]backend.Feature, error) {
	if m.fetchError != nil {
		return nil, m.fetchError
	}
	if query == "error:old" {
		return nil, errors.New("simulated fetch error")
	}

	return m.queryResults[query], nil
}

func (m *mockFetcher) GetFeature(_ context.Context, _ string) (*backendtypes.GetFeatureResult, error) {
	// Not needed for these test cases, but required by interface
	panic("not implemented")
}

func noopVerifyMocks(_ *testing.T, _ *mockStateAdapter, _ *mockDiffSerializer[testDiff], _ *mockWorkflow[testDiff]) {
}

func TestRun(t *testing.T) {
	ctx := context.Background()
	searchID := "search-123"
	eventID := "event-456"
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	featureA := makeFeature("a", "Feature A", "limited")
	featureB := makeFeature("b", "Feature B", "limited")
	featureBUpdated := makeFeature("b", "Feature B", "widely")

	// Pre-build snapshots for re-use
	snapshotAB := toSnapshot([]backend.Feature{featureA, featureB})
	// snapshotBUpdated := toSnapshot([]backend.Feature{featureBUpdated})

	tests := []struct {
		name               string
		query              string
		previousStateBytes []byte
		setupMocks         func(*mockStateAdapter, *mockDiffSerializer[testDiff], *mockWorkflow[testDiff], *mockFetcher)
		wantResult         *DiffResult
		wantErr            error
		verifyMocks        func(t *testing.T, adapter *mockStateAdapter,
			serializer *mockDiffSerializer[testDiff], workflow *mockWorkflow[testDiff])
	}{
		{
			name:               "Cold Start",
			query:              "q=new",
			previousStateBytes: nil,
			setupMocks: func(adapter *mockStateAdapter, serializer *mockDiffSerializer[testDiff],
				workflow *mockWorkflow[testDiff], fetcher *mockFetcher) {
				adapter.loadReturns.isEmpty = true
				fetcher.queryResults = map[string][]backend.Feature{"q=new": {featureA}}
				workflow.getDiffResult = new(testDiff)
				workflow.hasChangesResult = false // No data changes on cold start
				adapter.serializeReturns.bytes = []byte("new-state")
				serializer.serializeReturns.bytes = []byte("diff-output")
				workflow.summaryResult = []byte("summary")
			},
			wantResult: &DiffResult{
				State:       BlobArtifact{ID: "state-id", Bytes: []byte("new-state")},
				Diff:        BlobArtifact{ID: "event-456", Bytes: []byte("diff-output")},
				Summary:     []byte("summary"),
				Reasons:     nil,
				GeneratedAt: fixedTime,
			},
			wantErr: nil,
			verifyMocks: func(t *testing.T, adapter *mockStateAdapter,
				_ *mockDiffSerializer[testDiff], workflow *mockWorkflow[testDiff]) {
				if workflow.calculateDiffCalled {
					t.Error("expected CalculateDiff not to be called on cold start")
				}
				if adapter.serializeCalledWith.id != "state-id" {
					t.Errorf("adapter.Serialize id mismatch: got %s, want %s",
						adapter.serializeCalledWith.id, "state-id")
				}
			},
		},
		{
			name:               "No Changes",
			query:              "q=same",
			previousStateBytes: []byte("old-state"),
			setupMocks: func(adapter *mockStateAdapter,
				_ *mockDiffSerializer[testDiff], workflow *mockWorkflow[testDiff], fetcher *mockFetcher) {
				adapter.loadReturns.isEmpty = false
				adapter.loadReturns.signature = "q=same"
				adapter.loadReturns.snapshot = snapshotAB
				fetcher.queryResults = map[string][]backend.Feature{"q=same": {featureA, featureB}}
				workflow.hasChangesResult = false
			},
			wantResult:  nil,
			wantErr:     ErrNoChangesDetected,
			verifyMocks: noopVerifyMocks,
		},
		{
			name:               "Data Update",
			query:              "q=same",
			previousStateBytes: []byte("old-state"),
			setupMocks: func(adapter *mockStateAdapter, serializer *mockDiffSerializer[testDiff],
				workflow *mockWorkflow[testDiff], fetcher *mockFetcher) {
				adapter.loadReturns.isEmpty = false
				adapter.loadReturns.signature = "q=same"
				adapter.loadReturns.snapshot = snapshotAB
				fetcher.queryResults = map[string][]backend.Feature{"q=same": {featureA, featureBUpdated}}
				workflow.hasChangesResult = true
				workflow.hasDataChangesResult = true
				workflow.getDiffResult = &testDiff{Content: "B updated"}
				adapter.serializeReturns.bytes = []byte("new-state-updated")
				serializer.serializeReturns.bytes = []byte("diff-updated")
				workflow.summaryResult = []byte("summary-updated")
			},
			wantResult: &DiffResult{
				State:       BlobArtifact{ID: "state-id", Bytes: []byte("new-state-updated")},
				Diff:        BlobArtifact{ID: "event-456", Bytes: []byte("diff-updated")},
				Summary:     []byte("summary-updated"),
				Reasons:     []workertypes.Reason{workertypes.ReasonDataUpdated},
				GeneratedAt: fixedTime,
			},
			wantErr:     nil,
			verifyMocks: noopVerifyMocks,
		},
		{
			name:               "Query Change - Flush Success",
			query:              "q=new",
			previousStateBytes: []byte("old-state"),
			setupMocks: func(adapter *mockStateAdapter, serializer *mockDiffSerializer[testDiff],
				workflow *mockWorkflow[testDiff], fetcher *mockFetcher) {
				adapter.loadReturns.isEmpty = false
				adapter.loadReturns.signature = "q=old"
				adapter.loadReturns.snapshot = snapshotAB
				fetcher.queryResults = map[string][]backend.Feature{
					"q=old": {featureA, featureBUpdated}, // Data changed on old query
					"q=new": {featureA},                  // New query returns different set
				}
				workflow.hasChangesResult = true // Because of the flush diff
				workflow.getDiffResult = &testDiff{Content: "B updated"}
				adapter.serializeReturns.bytes = []byte("new-state-after-query-change")
				serializer.serializeReturns.bytes = []byte("diff-after-query-change")
				workflow.summaryResult = []byte("summary-query-change")
			},
			wantResult: &DiffResult{
				State:       BlobArtifact{ID: "state-id", Bytes: []byte("new-state-after-query-change")},
				Diff:        BlobArtifact{ID: "event-456", Bytes: []byte("diff-after-query-change")},
				Summary:     []byte("summary-query-change"),
				Reasons:     []workertypes.Reason{workertypes.ReasonQueryChanged},
				GeneratedAt: fixedTime,
			},
			wantErr: nil,
			verifyMocks: func(t *testing.T, _ *mockStateAdapter,
				_ *mockDiffSerializer[testDiff], workflow *mockWorkflow[testDiff]) {
				if !workflow.calculateDiffCalled {
					t.Error("expected CalculateDiff to be called on query change")
				}
				if !workflow.setQueryChangedCalled {
					t.Error("expected SetQueryChanged to be called")
				}
			},
		},
		{
			name:               "Query Change - Flush Failed",
			query:              "q=new",
			previousStateBytes: []byte("old-state"),
			setupMocks: func(adapter *mockStateAdapter, serializer *mockDiffSerializer[testDiff],
				workflow *mockWorkflow[testDiff], fetcher *mockFetcher) {
				adapter.loadReturns.isEmpty = false
				adapter.loadReturns.signature = "error:old" // This will trigger fetch error
				adapter.loadReturns.snapshot = snapshotAB
				fetcher.queryResults = map[string][]backend.Feature{
					"q=new": {featureA},
				}
				workflow.hasChangesResult = false // No data diff was performed
				workflow.getDiffResult = &testDiff{Content: ""}
				adapter.serializeReturns.bytes = []byte("new-state-flush-failed")
				serializer.serializeReturns.bytes = []byte("diff-flush-failed")
				workflow.summaryResult = []byte("summary-flush-failed")
			},
			wantResult: &DiffResult{
				State:       BlobArtifact{ID: "state-id", Bytes: []byte("new-state-flush-failed")},
				Diff:        BlobArtifact{ID: "event-456", Bytes: []byte("diff-flush-failed")},
				Summary:     []byte("summary-flush-failed"),
				Reasons:     []workertypes.Reason{workertypes.ReasonQueryChanged},
				GeneratedAt: fixedTime,
			},
			wantErr: nil,
			verifyMocks: func(t *testing.T, _ *mockStateAdapter,
				_ *mockDiffSerializer[testDiff], workflow *mockWorkflow[testDiff]) {
				if workflow.calculateDiffCalled {
					t.Error("expected CalculateDiff not to be called on flush failure")
				}
				if !workflow.setQueryChangedCalled {
					t.Error("expected SetQueryChanged to be called")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks for this test case
			adapter := new(mockStateAdapter)
			serializer := new(mockDiffSerializer[testDiff])
			workflow := new(mockWorkflow[testDiff])
			fetcher := new(mockFetcher)

			tc.setupMocks(adapter, serializer, workflow, fetcher)

			// Create the differ with mocked dependencies
			d := &FeatureDiffer[testDiff]{
				client:         fetcher,
				workflow:       workflow,
				stateAdapter:   adapter,
				diffSerializer: serializer,
				idGenerator:    &mockIDGenerator{stateID: "state-id", diffID: "diff-id"},
				timeNow:        func() time.Time { return fixedTime },
			}

			// Run the method under test
			result, err := d.Run(ctx, searchID, tc.query, eventID, tc.previousStateBytes)

			// Assert results
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Run() error = %v, wantErr %v", err, tc.wantErr)
			}

			if diff := cmp.Diff(tc.wantResult, result); diff != "" {
				t.Errorf("Run() result mismatch (-want +got):\n%s", diff)
			}

			tc.verifyMocks(t, adapter, serializer, workflow)
		})
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
		want comparables.Feature
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
					"chrome":  {Status: &avail, Date: &date, Version: generic.ValuePtr("version")},
					"firefox": {Status: &unavail, Date: nil, Version: nil},
					"safari":  {Status: &avail, Date: nil, Version: nil},
					"unknown": {Status: &avail, Date: nil, Version: nil}, // Should be ignored
				},
				VendorPositions:            nil,
				DeveloperSignals:           nil,
				SystemManagedSavedSearchId: nil,
			},
			want: createExpectedFeature("feat-1", "Feature One",
				backend.Widely, map[backend.SupportedBrowsers]comparables.BrowserState{
					backend.Chrome: {
						Status:  generic.OptionallySet[backend.BrowserImplementationStatus]{Value: backend.Available, IsSet: true},
						Date:    generic.OptionallySet[*time.Time]{Value: &date.Time, IsSet: true},
						Version: generic.OptionallySet[*string]{Value: generic.ValuePtr("version"), IsSet: true},
					},
					backend.ChromeAndroid: zero[comparables.BrowserState](),
					backend.Firefox: {
						Status:  generic.OptionallySet[backend.BrowserImplementationStatus]{Value: backend.Unavailable, IsSet: true},
						Date:    generic.UnsetOpt[*time.Time](),
						Version: generic.UnsetOpt[*string](),
					},
					backend.FirefoxAndroid: zero[comparables.BrowserState](),
					backend.Safari: {
						Status:  generic.OptionallySet[backend.BrowserImplementationStatus]{Value: backend.Available, IsSet: true},
						Date:    generic.UnsetOpt[*time.Time](),
						Version: generic.UnsetOpt[*string](),
					},
					backend.SafariIos: zero[comparables.BrowserState](),
					backend.Edge:      zero[comparables.BrowserState](),
				}),
		},
		{
			name: "Minimal (Nil Maps)",
			in: backend.Feature{
				FeatureId:                  "feat-2",
				Name:                       "Minimal Feature",
				Baseline:                   nil,
				Spec:                       nil,
				BrowserImplementations:     nil,
				Discouraged:                nil,
				Usage:                      nil,
				Wpt:                        nil,
				VendorPositions:            nil,
				DeveloperSignals:           nil,
				SystemManagedSavedSearchId: nil,
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

// createExpectedFeature constructs a comparables.Feature with all OptionallySet fields initialized.
// This is required to pass the exhaustruct linter in tests.
func createExpectedFeature(id, name string, baseline backend.BaselineInfoStatus,
	browsers map[backend.SupportedBrowsers]comparables.BrowserState) comparables.Feature {
	cf := comparables.Feature{
		ID:   id,
		Name: generic.OptionallySet[string]{Value: name, IsSet: true},
		BaselineStatus: generic.OptionallySet[comparables.BaselineState]{Value: comparables.BaselineState{
			Status:   generic.OptionallySet[backend.BaselineInfoStatus]{Value: baseline, IsSet: true},
			LowDate:  generic.OptionallySet[*time.Time]{Value: nil, IsSet: true},
			HighDate: generic.OptionallySet[*time.Time]{Value: nil, IsSet: true},
		}, IsSet: true},
		BrowserImpls: generic.UnsetOpt[comparables.BrowserImplementations](),
		Docs:         generic.UnsetOpt[comparables.Docs](),
	}

	// Override specific browsers if provided
	if browsers != nil {
		cf.BrowserImpls.IsSet = true
		setIfPresent(browsers, backend.Chrome, &cf.BrowserImpls.Value.Chrome)
		setIfPresent(browsers, backend.ChromeAndroid, &cf.BrowserImpls.Value.ChromeAndroid)
		setIfPresent(browsers, backend.Edge, &cf.BrowserImpls.Value.Edge)
		setIfPresent(browsers, backend.Firefox, &cf.BrowserImpls.Value.Firefox)
		setIfPresent(browsers, backend.FirefoxAndroid, &cf.BrowserImpls.Value.FirefoxAndroid)
		setIfPresent(browsers, backend.Safari, &cf.BrowserImpls.Value.Safari)
		setIfPresent(browsers, backend.SafariIos, &cf.BrowserImpls.Value.SafariIos)
	}

	return cf
}

// nolint:ireturn // WONTFIX: used for testing only.
func zero[T any]() T {
	return *new(T)
}

func setIfPresent[K comparable, V any](m map[K]V, key K, target *generic.OptionallySet[V]) {
	var zero V
	if val, ok := m[key]; ok && !reflect.DeepEqual(zero, val) {
		target.IsSet = true
		target.Value = val
	}
}
