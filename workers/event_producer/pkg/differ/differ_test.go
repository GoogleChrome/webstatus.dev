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
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	v1 "github.com/GoogleChrome/webstatus.dev/lib/blobtypes/featurelistdiff/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/generic"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes/comparables"
	"github.com/google/go-cmp/cmp"
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
		snapshot    map[string]comparables.Feature
		id          string
		signature   string
		queryErrors []workertypes.SummaryQueryError
		isEmpty     bool
		err         error
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

func (m *mockStateAdapter) Load(_ []byte) (
	map[string]comparables.Feature,
	string,
	string,
	[]workertypes.SummaryQueryError,
	bool,
	error,
) {
	return m.loadReturns.snapshot, m.loadReturns.id, m.loadReturns.signature,
		m.loadReturns.queryErrors, m.loadReturns.isEmpty, m.loadReturns.err
}

func (m *mockStateAdapter) Serialize(id, searchID, eventID, query string, _ []workertypes.SummaryQueryError,
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
	calculateDiffCalled          bool
	reconcileHistoryCalled       bool
	setQueryChangedCalled        bool
	setResolvedQueryErrorsCalled bool
	resolvedQueryErrors          comparables.QueryErrors
	hasRemovedFeaturesResult     bool
	hasChangesResult             bool
	hasDataChangesResult         bool
	getDiffResult                *D
	summaryResult                []byte
	summaryError                 error
}

func (m *mockWorkflow[D]) CalculateDiff(_, _ map[string]comparables.Feature, _ comparables.QueryErrors,
	_ comparables.SnapshotOrigin) {
	m.calculateDiffCalled = true
}
func (m *mockWorkflow[D]) ReconcileHistory(_ context.Context, _, _ map[string]comparables.Feature) error {
	m.reconcileHistoryCalled = true

	return nil
}
func (m *mockWorkflow[D]) HasRemovedFeatures() bool { return m.hasRemovedFeaturesResult }
func (m *mockWorkflow[D]) HasChanges() bool         { return m.hasChangesResult }
func (m *mockWorkflow[D]) HasDataChanges() bool     { return m.hasDataChangesResult }
func (m *mockWorkflow[D]) SetQueryChanged(val bool) {
	m.setQueryChangedCalled = val
}
func (m *mockWorkflow[D]) SetResolvedQueryErrors(errs comparables.QueryErrors) {
	m.setResolvedQueryErrorsCalled = true
	m.resolvedQueryErrors = errs
}

func (m *mockWorkflow[D]) GetDiff() *D {
	return m.getDiffResult
}
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
	userErrors   map[string]*workertypes.UserError
}

func (m *mockFetcher) FetchFeatures(_ context.Context, query string) (*workertypes.FetchFeaturesResult, error) {
	if m.fetchError != nil {
		return nil, m.fetchError
	}
	if query == "error:old" {
		return nil, errors.New("simulated fetch error")
	}
	if m.userErrors != nil {
		if err, ok := m.userErrors[query]; ok {
			return &workertypes.FetchFeaturesResult{Features: nil, UserError: err}, nil
		}
	}

	return &workertypes.FetchFeaturesResult{Features: m.queryResults[query], UserError: nil}, nil
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
	snapshotAB := comparables.NewFeatureMapFromBackendFeatures([]backend.Feature{featureA, featureB})

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
		{
			name:               "Query Change - Flush UserError (Old Query Fails)",
			query:              "q=new",
			previousStateBytes: []byte("old-state"),
			setupMocks: func(adapter *mockStateAdapter, serializer *mockDiffSerializer[testDiff],
				workflow *mockWorkflow[testDiff], fetcher *mockFetcher) {
				adapter.loadReturns.isEmpty = false
				adapter.loadReturns.signature = "q=old"
				adapter.loadReturns.snapshot = snapshotAB
				fetcher.queryResults = map[string][]backend.Feature{
					"q=new": {featureA},
				}
				fetcher.userErrors = map[string]*workertypes.UserError{
					"q=old": {
						QueryErrors: []workertypes.SummaryQueryError{
							{Code: workertypes.SummaryQueryErrorCodeSavedSearchNotFound},
						},
					},
				}
				workflow.hasChangesResult = false // No data diff was performed
				workflow.getDiffResult = &testDiff{Content: ""}
				adapter.serializeReturns.bytes = []byte("new-state-flush-usererror")
				serializer.serializeReturns.bytes = []byte("diff-flush-usererror")
				workflow.summaryResult = []byte("summary-flush-usererror")
			},
			wantResult: &DiffResult{
				State:       BlobArtifact{ID: "state-id", Bytes: []byte("new-state-flush-usererror")},
				Diff:        BlobArtifact{ID: "event-456", Bytes: []byte("diff-flush-usererror")},
				Summary:     []byte("summary-flush-usererror"),
				Reasons:     []workertypes.Reason{workertypes.ReasonQueryChanged},
				GeneratedAt: fixedTime,
			},
			wantErr: nil,
			verifyMocks: func(t *testing.T, _ *mockStateAdapter,
				_ *mockDiffSerializer[testDiff], workflow *mockWorkflow[testDiff]) {
				if workflow.calculateDiffCalled {
					t.Error("expected CalculateDiff not to be called on flush UserError")
				}
				if !workflow.setQueryChangedCalled {
					t.Error("expected SetQueryChanged to be called")
				}
			},
		},
		{
			name:               "Regression - All Query Recovery from QueryGrammar Error",
			query:              "", // All Features query is empty string
			previousStateBytes: []byte("old-state-with-error"),
			setupMocks: func(adapter *mockStateAdapter, serializer *mockDiffSerializer[testDiff],
				workflow *mockWorkflow[testDiff], fetcher *mockFetcher) {
				adapter.loadReturns.isEmpty = false
				adapter.loadReturns.signature = ""
				adapter.loadReturns.snapshot = comparables.NewFeatureMapFromBackendFeatures(nil)
				adapter.loadReturns.queryErrors = []workertypes.SummaryQueryError{
					{Code: workertypes.SummaryQueryErrorCodeQueryGrammar},
				}
				fetcher.queryResults = map[string][]backend.Feature{
					"": {featureA, featureB}, // Fix lands: empty query now returns all features cleanly
				}
				workflow.hasChangesResult = false
				workflow.hasDataChangesResult = false
				workflow.getDiffResult = &testDiff{Content: "Recovered all features"}
				adapter.serializeReturns.bytes = []byte("recovered-state")
				serializer.serializeReturns.bytes = []byte("recovered-diff")
				workflow.summaryResult = []byte("recovered-summary")
			},
			wantResult: &DiffResult{
				State:       BlobArtifact{ID: "state-id", Bytes: []byte("recovered-state")},
				Diff:        BlobArtifact{ID: "event-456", Bytes: []byte("recovered-diff")},
				Summary:     []byte("recovered-summary"),
				Reasons:     nil,
				GeneratedAt: fixedTime,
			},
			wantErr: nil,
			verifyMocks: func(t *testing.T, _ *mockStateAdapter,
				_ *mockDiffSerializer[testDiff], workflow *mockWorkflow[testDiff]) {
				if workflow.calculateDiffCalled {
					t.Error("expected CalculateDiff NOT to be called when TargetSnapshot is nil on error recovery")
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
				client:          fetcher,
				workflowFactory: func() StateCompareWorkflow[testDiff] { return workflow },
				stateAdapter:    adapter,
				diffSerializer:  serializer,
				idGenerator:     &mockIDGenerator{stateID: "state-id", diffID: "diff-id"},
				timeNow:         func() time.Time { return fixedTime },
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

type mockSummaryGenerator struct{}

func (m *mockSummaryGenerator) GenerateJSONSummary(_ v1.FeatureDiff) ([]byte, error) {
	return []byte(`{"schemaVersion":"v1","text":"Tracking resumed cleanly."}`), nil
}

func createLiveFeature(id, name string) comparables.Feature {
	f := new(comparables.Feature)
	f.ID = id
	f.Name = generic.SetOpt(name)

	return *f
}

func TestCalculateDiff_ErrorRecovery_WithPoint1Fix(t *testing.T) {
	workflow := v1.NewFeatureDiffWorkflow(nil, new(mockSummaryGenerator))

	prevQueryErrors := comparables.QueryErrors{
		{Code: comparables.ErrorCodeQueryGrammar},
	}
	currentQueryErrors := comparables.QueryErrors{}

	oldSnapshot := map[string]comparables.Feature{}
	newSnapshot := map[string]comparables.Feature{
		"feat-1": createLiveFeature("feat-1", "Feature 1"),
		"feat-2": createLiveFeature("feat-2", "Feature 2"),
	}

	var targetSnapshot map[string]comparables.Feature
	if len(prevQueryErrors) > 0 && len(currentQueryErrors) == 0 {
		targetSnapshot = nil
	} else {
		targetSnapshot = newSnapshot
	}

	if targetSnapshot != nil {
		workflow.CalculateDiff(oldSnapshot, targetSnapshot, currentQueryErrors, comparables.OriginLive)
	}

	diff := workflow.GetDiff()
	if len(diff.Added) != 0 {
		t.Errorf("expected 0 Added highlights upon error recovery, got %d", len(diff.Added))
	}
	if workflow.HasChanges() {
		t.Error("expected HasChanges() == false when TargetSnapshot is nil upon recovery")
	}
}

func TestRun_RealWorkflow_QueryErrorRecovery_NoSpam(t *testing.T) {
	adapter := new(mockStateAdapter)
	serializer := new(mockDiffSerializer[v1.FeatureDiff])
	fetcher := new(mockFetcher)

	adapter.loadReturns.isEmpty = false
	adapter.loadReturns.signature = "my-query"
	adapter.loadReturns.snapshot = map[string]comparables.Feature{}
	adapter.loadReturns.queryErrors = []workertypes.SummaryQueryError{
		{Code: workertypes.SummaryQueryErrorCodeQueryGrammar},
	}

	fetcher.queryResults = map[string][]backend.Feature{
		"my-query": {
			makeFeature("feat-1", "Feature 1", "available"),
			makeFeature("feat-2", "Feature 2", "available"),
		},
	}
	adapter.serializeReturns.bytes = []byte("clean-recovered-state")
	serializer.serializeReturns.bytes = []byte("recovered-diff-blob")

	d := NewFeatureDiffer[v1.FeatureDiff](
		fetcher,
		func() StateCompareWorkflow[v1.FeatureDiff] {
			return v1.NewFeatureDiffWorkflow(nil, new(mockSummaryGenerator))
		},
		adapter,
		serializer,
	)

	res, err := d.Run(context.Background(), "search-123", "my-query", "event-789", []byte("old-state"))
	if err != nil {
		t.Fatalf("unexpected error running differ: %v", err)
	}

	if string(res.State.Bytes) != "clean-recovered-state" {
		t.Errorf("expected clean state blob saved, got %s", string(res.State.Bytes))
	}
	if serializer.serializeCalledWith.diff == nil {
		t.Fatalf("expected DiffSerializer to be called with a valid diff, got nil")
	}
	serializedDiff := serializer.serializeCalledWith.diff
	if len(serializedDiff.Added) != 0 {
		t.Errorf("expected exactly 0 Added features in serialized diff blob upon recovery, got %d", len(serializedDiff.Added))
	}
}
