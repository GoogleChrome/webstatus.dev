// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/google/go-cmp/cmp"
)

// mockSpannerClient implements EventProducerSpannerClient for testing.
type mockSpannerClient struct {
	tryAcquireLockCalled bool
	acquireLockReq       struct {
		SavedSearchID string
		SnapshotType  gcpspanner.SavedSearchSnapshotType
		WorkerID      string
		TTL           time.Duration
	}
	acquireLockResp bool
	acquireLockErr  error

	releaseLockCalled bool
	releaseLockReq    struct {
		SavedSearchID string
		SnapshotType  gcpspanner.SavedSearchSnapshotType
		WorkerID      string
	}
	releaseLockErr error

	publishEventCalled bool
	publishEventReq    struct {
		Event        gcpspanner.SavedSearchNotificationCreateRequest
		NewStatePath string
		WorkerID     string
	}
	publishEventResp *string
	publishEventErr  error

	getLatestEventCalled bool
	getLatestEventReq    struct {
		SavedSearchID string
		SnapshotType  gcpspanner.SavedSearchSnapshotType
	}
	getLatestEventResp *gcpspanner.SavedSearchNotificationEvent
	getLatestEventErr  error
}

func (m *mockSpannerClient) TryAcquireSavedSearchStateWorkerLock(
	_ context.Context,
	savedSearchID string,
	snapshotType gcpspanner.SavedSearchSnapshotType,
	workerID string,
	ttl time.Duration) (bool, error) {
	m.tryAcquireLockCalled = true
	m.acquireLockReq.SavedSearchID = savedSearchID
	m.acquireLockReq.SnapshotType = snapshotType
	m.acquireLockReq.WorkerID = workerID
	m.acquireLockReq.TTL = ttl

	return m.acquireLockResp, m.acquireLockErr
}

func (m *mockSpannerClient) PublishSavedSearchNotificationEvent(_ context.Context,
	event gcpspanner.SavedSearchNotificationCreateRequest, newStatePath, workerID string,
	_ ...gcpspanner.CreateOption) (*string, error) {
	m.publishEventCalled = true
	m.publishEventReq.Event = event
	m.publishEventReq.NewStatePath = newStatePath
	m.publishEventReq.WorkerID = workerID

	return m.publishEventResp, m.publishEventErr
}

func (m *mockSpannerClient) GetLatestSavedSearchNotificationEvent(
	_ context.Context,
	savedSearchID string,
	snapshotType gcpspanner.SavedSearchSnapshotType,
) (*gcpspanner.SavedSearchNotificationEvent, error) {
	m.getLatestEventCalled = true
	m.getLatestEventReq.SavedSearchID = savedSearchID
	m.getLatestEventReq.SnapshotType = snapshotType

	return m.getLatestEventResp, m.getLatestEventErr
}

func (m *mockSpannerClient) ReleaseSavedSearchStateWorkerLock(
	_ context.Context,
	savedSearchID string,
	snapshotType gcpspanner.SavedSearchSnapshotType,
	workerID string) error {
	m.releaseLockCalled = true
	m.releaseLockReq.SavedSearchID = savedSearchID
	m.releaseLockReq.SnapshotType = snapshotType
	m.releaseLockReq.WorkerID = workerID

	return m.releaseLockErr
}

func TestEventProducer_AcquireLock(t *testing.T) {
	tests := []struct {
		name             string
		freq             workertypes.JobFrequency
		wantSnapshotType gcpspanner.SavedSearchSnapshotType
		mockResp         bool
		mockErr          error
		wantErr          bool
	}{
		{
			name:             "Immediate maps to Immediate",
			freq:             workertypes.FrequencyImmediate,
			wantSnapshotType: gcpspanner.SavedSearchSnapshotTypeImmediate,
			mockResp:         true,
			mockErr:          nil,
			wantErr:          false,
		},
		{
			name:             "Weekly maps to Weekly",
			freq:             workertypes.FrequencyWeekly,
			wantSnapshotType: gcpspanner.SavedSearchSnapshotTypeWeekly,
			mockResp:         true,
			mockErr:          nil,
			wantErr:          false,
		},
		{
			name:             "Error propagation",
			freq:             workertypes.FrequencyMonthly,
			wantSnapshotType: gcpspanner.SavedSearchSnapshotTypeMonthly,
			mockResp:         false,
			mockErr:          errors.New("spanner error"),
			wantErr:          true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := new(mockSpannerClient)
			mock.acquireLockResp = tc.mockResp
			mock.acquireLockErr = tc.mockErr

			adapter := NewEventProducer(mock)

			err := adapter.AcquireLock(context.Background(), "search-1", tc.freq, "worker-1", time.Minute)

			if (err != nil) != tc.wantErr {
				t.Errorf("AcquireLock() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !mock.tryAcquireLockCalled {
				t.Fatal("TryAcquireSavedSearchStateWorkerLock not called")
			}

			expectedReq := struct {
				SavedSearchID string
				SnapshotType  gcpspanner.SavedSearchSnapshotType
				WorkerID      string
				TTL           time.Duration
			}{
				SavedSearchID: "search-1",
				SnapshotType:  tc.wantSnapshotType,
				WorkerID:      "worker-1",
				TTL:           time.Minute,
			}

			if diff := cmp.Diff(expectedReq, mock.acquireLockReq); diff != "" {
				t.Errorf("TryAcquireSavedSearchStateWorkerLock request mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEventProducer_ReleaseLock(t *testing.T) {
	tests := []struct {
		name             string
		freq             workertypes.JobFrequency
		wantSnapshotType gcpspanner.SavedSearchSnapshotType
		mockErr          error
		wantErr          bool
	}{
		{
			name:             "Regular release",
			freq:             workertypes.FrequencyImmediate,
			wantSnapshotType: gcpspanner.SavedSearchSnapshotTypeImmediate,
			mockErr:          nil,
			wantErr:          false,
		},
		{
			name:             "Error propagation",
			freq:             workertypes.FrequencyWeekly,
			wantSnapshotType: gcpspanner.SavedSearchSnapshotTypeWeekly,
			mockErr:          errors.New("lock lost"),
			wantErr:          true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := new(mockSpannerClient)
			mock.releaseLockErr = tc.mockErr

			adapter := NewEventProducer(mock)

			err := adapter.ReleaseLock(context.Background(), "search-1", tc.freq, "worker-1")

			if (err != nil) != tc.wantErr {
				t.Errorf("ReleaseLock() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !mock.releaseLockCalled {
				t.Fatal("ReleaseSavedSearchStateWorkerLock not called")
			}

			expectedReq := struct {
				SavedSearchID string
				SnapshotType  gcpspanner.SavedSearchSnapshotType
				WorkerID      string
			}{
				SavedSearchID: "search-1",
				SnapshotType:  tc.wantSnapshotType,
				WorkerID:      "worker-1",
			}

			if diff := cmp.Diff(expectedReq, mock.releaseLockReq); diff != "" {
				t.Errorf("ReleaseSavedSearchStateWorkerLock request mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEventProducer_PublishEvent(t *testing.T) {
	generatedAt := time.Now()
	summaryJSON := `{"added": 1, "removed": 2}`

	// Defined named struct for better type safety and comparison in tests
	type expectedRequest struct {
		Event        gcpspanner.SavedSearchNotificationCreateRequest
		NewStatePath string
		WorkerID     string
	}

	tests := []struct {
		name            string
		req             workertypes.PublishEventRequest
		mockPublishErr  error
		mockPublishResp *string
		wantErr         bool
		expectCall      bool
		expectedReq     *expectedRequest
	}{
		{
			name: "success",
			req: workertypes.PublishEventRequest{
				EventID:       "event-1",
				SearchID:      "search-1",
				SearchName:    "Search 1",
				StateID:       "state-1",
				StateBlobPath: "gs://bucket/state",
				DiffID:        "diff-1",
				DiffBlobPath:  "gs://bucket/diff",
				Summary:       []byte(summaryJSON),
				Reasons:       []workertypes.Reason{workertypes.ReasonDataUpdated},
				Frequency:     workertypes.FrequencyWeekly,
				Query:         "query",
				GeneratedAt:   generatedAt,
			},
			mockPublishErr:  nil,
			mockPublishResp: new("new-event-id"),
			wantErr:         false,
			expectCall:      true,
			expectedReq: &expectedRequest{
				Event: gcpspanner.SavedSearchNotificationCreateRequest{
					SavedSearchID: "search-1",
					SnapshotType:  gcpspanner.SavedSearchSnapshotTypeWeekly,
					Timestamp:     generatedAt,
					EventType:     "", // Matches TODO in implementation
					Reasons:       []string{"DATA_UPDATED"},
					BlobPath:      "gs://bucket/state",
					DiffBlobPath:  "gs://bucket/diff",
					Summary: spanner.NullJSON{Value: map[string]any{
						"summary": map[string]any{
							"added":   float64(1),
							"removed": float64(2),
						},
					}, Valid: true},
				},
				NewStatePath: "gs://bucket/state",
				WorkerID:     "event-1",
			},
		},
		{
			name: "spanner publish error",
			req: workertypes.PublishEventRequest{
				EventID:       "event-1",
				SearchID:      "search-1",
				SearchName:    "Search 1",
				StateID:       "state-1",
				StateBlobPath: "gs://bucket/state",
				DiffID:        "diff-1",
				DiffBlobPath:  "gs://bucket/diff",
				Summary:       []byte(summaryJSON),
				Reasons:       []workertypes.Reason{workertypes.ReasonDataUpdated},
				Frequency:     workertypes.FrequencyWeekly,
				Query:         "query",
				GeneratedAt:   generatedAt,
			},
			mockPublishErr:  errors.New("spanner failure"),
			mockPublishResp: nil,
			wantErr:         true,
			expectCall:      true,
			expectedReq: &expectedRequest{
				Event: gcpspanner.SavedSearchNotificationCreateRequest{
					SavedSearchID: "search-1",
					SnapshotType:  gcpspanner.SavedSearchSnapshotTypeWeekly,
					Timestamp:     generatedAt,
					EventType:     "",
					Reasons:       []string{"DATA_UPDATED"},
					BlobPath:      "gs://bucket/state",
					DiffBlobPath:  "gs://bucket/diff",
					Summary: spanner.NullJSON{Value: map[string]any{
						"summary": map[string]any{
							"added":   float64(1),
							"removed": float64(2),
						},
					}, Valid: true},
				},
				NewStatePath: "gs://bucket/state",
				WorkerID:     "event-1",
			},
		},
		{
			name: "invalid json summary",
			req: workertypes.PublishEventRequest{
				EventID:       "event-1",
				SearchID:      "search-1",
				SearchName:    "Search 1",
				Summary:       []byte("invalid-json"),
				Frequency:     workertypes.FrequencyWeekly,
				StateID:       "",
				StateBlobPath: "",
				DiffID:        "",
				DiffBlobPath:  "",
				Reasons:       nil,
				Query:         "",
				GeneratedAt:   time.Time{},
			},
			mockPublishResp: nil,
			mockPublishErr:  nil,
			wantErr:         true,
			expectCall:      false,
			expectedReq:     nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := new(mockSpannerClient)
			mock.publishEventErr = tc.mockPublishErr
			mock.publishEventResp = tc.mockPublishResp

			adapter := NewEventProducer(mock)

			err := adapter.PublishEvent(context.Background(), tc.req)

			if (err != nil) != tc.wantErr {
				t.Errorf("PublishEvent() error = %v, wantErr %v", err, tc.wantErr)
			}

			if mock.publishEventCalled != tc.expectCall {
				t.Errorf(
					"PublishSavedSearchNotificationEvent called = %v, expected %v",
					mock.publishEventCalled,
					tc.expectCall,
				)
			}

			if tc.expectCall && tc.expectedReq != nil {
				// Verify mapping by converting mock data to our expected type for type-safe comparison
				actualReq := expectedRequest{
					Event:        mock.publishEventReq.Event,
					NewStatePath: mock.publishEventReq.NewStatePath,
					WorkerID:     mock.publishEventReq.WorkerID,
				}

				if diff := cmp.Diff(*tc.expectedReq, actualReq); diff != "" {
					t.Errorf("PublishSavedSearchNotificationEvent request mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestEventProducer_GetLatestEvent(t *testing.T) {
	testEvent := new(gcpspanner.SavedSearchNotificationEvent)
	testEvent.ID = "event-123"
	testEvent.BlobPath = "gs://bucket/blob"

	tests := []struct {
		name             string
		freq             workertypes.JobFrequency
		wantSnapshotType gcpspanner.SavedSearchSnapshotType
		mockResp         *gcpspanner.SavedSearchNotificationEvent
		mockErr          error
		wantInfo         *workertypes.LatestEventInfo
		expectedError    error
	}{
		{
			name:             "Found event",
			freq:             workertypes.FrequencyWeekly,
			wantSnapshotType: gcpspanner.SavedSearchSnapshotTypeWeekly,
			mockResp:         testEvent,
			mockErr:          nil,
			wantInfo: &workertypes.LatestEventInfo{
				EventID:       "event-123",
				StateBlobPath: "gs://bucket/blob",
			},
			expectedError: nil,
		},
		{
			name:             "No event found",
			freq:             workertypes.FrequencyImmediate,
			wantSnapshotType: gcpspanner.SavedSearchSnapshotTypeImmediate,
			mockResp:         nil,
			mockErr:          gcpspanner.ErrQueryReturnedNoResults,
			wantInfo:         nil,
			expectedError:    workertypes.ErrLatestEventNotFound,
		},
		{
			name:             "Spanner error",
			freq:             workertypes.FrequencyImmediate,
			wantSnapshotType: gcpspanner.SavedSearchSnapshotTypeImmediate,
			mockResp:         nil,
			mockErr:          errTest,
			wantInfo:         nil,
			expectedError:    errTest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := new(mockSpannerClient)
			mock.getLatestEventResp = tc.mockResp
			mock.getLatestEventErr = tc.mockErr

			adapter := NewEventProducer(mock)

			info, err := adapter.GetLatestEvent(context.Background(), tc.freq, "search-1")

			if !errors.Is(err, tc.expectedError) {
				t.Errorf("GetLatestEvent() error = %v, wantErr %v", err, tc.expectedError)
			}

			if !mock.getLatestEventCalled {
				t.Fatal("GetLatestSavedSearchNotificationEvent not called")
			}

			if mock.getLatestEventReq.SavedSearchID != "search-1" {
				t.Errorf("GetLatestSavedSearchNotificationEvent called with wrong ID: got %q, want %q",
					mock.getLatestEventReq.SavedSearchID, "search-1")
			}
			if mock.getLatestEventReq.SnapshotType != tc.wantSnapshotType {
				t.Errorf("GetLatestSavedSearchNotificationEvent called with wrong SnapshotType: got %v, want %v",
					mock.getLatestEventReq.SnapshotType, tc.wantSnapshotType)
			}

			if diff := cmp.Diff(tc.wantInfo, info); diff != "" {
				t.Errorf("GetLatestEvent() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

type mockBackendAdapterForEventProducer struct {
	BackendSpannerClient

	getFeatureCalled bool
	GetFeatureReq    struct {
		FeatureID string
	}
	getFeatureResp *backendtypes.GetFeatureResult
	getFeatureErr  error

	featuresSearchCalled bool
	FeaturesSearchReqs   []struct {
		PageToken *string
		PageSize  int
		QueryNode *searchtypes.SearchNode
	}
	// A sequence of pages to return.
	featuresSearchPages []*backend.FeaturePage
	featuresSearchErr   error
	// Call count for search to iterate through pages
	searchCallCount int
}

func (m *mockBackendAdapterForEventProducer) GetFeature(
	_ context.Context,
	featureID string,
	_ backend.WPTMetricView,
	_ []backend.BrowserPathParam,
) (*backendtypes.GetFeatureResult, error) {
	m.getFeatureCalled = true
	m.GetFeatureReq.FeatureID = featureID

	return m.getFeatureResp, m.getFeatureErr
}

func (m *mockBackendAdapterForEventProducer) FeaturesSearch(
	_ context.Context,
	pageToken *string,
	pageSize int,
	queryNode *searchtypes.SearchNode,
	_ *backend.ListFeaturesParamsSort,
	_ backend.WPTMetricView,
	_ []backend.BrowserPathParam,
) (*backend.FeaturePage, error) {
	m.featuresSearchCalled = true
	m.FeaturesSearchReqs = append(m.FeaturesSearchReqs, struct {
		PageToken *string
		PageSize  int
		QueryNode *searchtypes.SearchNode
	}{pageToken, pageSize, queryNode})

	if m.featuresSearchErr != nil {
		return nil, m.featuresSearchErr // Return nil for FeaturePage on error
	}

	if m.searchCallCount < len(m.featuresSearchPages) {
		page := m.featuresSearchPages[m.searchCallCount]
		m.searchCallCount++

		return page, nil
	}

	return new(backend.FeaturePage), nil // End of results
}

func TestEventProducerDiffer_GetFeature(t *testing.T) {
	mock := new(mockBackendAdapterForEventProducer)
	f := new(backend.Feature)
	f.FeatureId = "fx"
	f.Name = "Feature x"
	mock.getFeatureResp = backendtypes.NewGetFeatureResult(backendtypes.NewRegularFeatureResult(f))

	adapter := NewEventProducerDiffer(mock)

	res, err := adapter.GetFeature(context.Background(), "fx")
	if err != nil {
		t.Fatalf("GetFeature() unexpected error: %v", err)
	}

	if !mock.getFeatureCalled {
		t.Error("GetFeature on backend client not called")
	}
	if mock.GetFeatureReq.FeatureID != "fx" {
		t.Errorf("GetFeature called with %q, want %q", mock.GetFeatureReq.FeatureID, "fx")
	}
	visitor := new(simpleRegularFeatureVisitor)
	if err := res.Visit(context.Background(), visitor); err != nil {
		t.Fatalf("Visit failed: %v", err)
	}

	if visitor.feature == nil {
		t.Fatal("Visitor did not receive a regular feature")
	}
	if visitor.feature.FeatureId != "fx" {
		t.Errorf("Feature ID mismatch: got %q, want fx", visitor.feature.FeatureId)
	}
	if visitor.feature.Name != "Feature x" {
		t.Errorf("Feature Name mismatch: got %q, want 'Feature x'", visitor.feature.Name)
	}
}

type simpleRegularFeatureVisitor struct {
	feature *backend.Feature
}

func (v *simpleRegularFeatureVisitor) VisitRegularFeature(_ context.Context,
	res backendtypes.RegularFeatureResult) error {
	v.feature = res.Feature()

	return nil
}
func (v *simpleRegularFeatureVisitor) VisitMovedFeature(_ context.Context, _ backendtypes.MovedFeatureResult) error {
	return nil
}
func (v *simpleRegularFeatureVisitor) VisitSplitFeature(_ context.Context, _ backendtypes.SplitFeatureResult) error {
	return nil
}

func newTestBackendFeature(id, name string) backend.Feature {
	f := new(backend.Feature)
	f.FeatureId = id
	f.Name = name

	return *f
}

func newMockBackendAdapterForEventProducer(pages []*backend.FeaturePage) *mockBackendAdapterForEventProducer {
	mock := new(mockBackendAdapterForEventProducer)
	mock.featuresSearchPages = pages

	return mock
}

func TestEventProducerDiffer_FetchFeatures_Success(t *testing.T) {
	testCases := []struct {
		name             string
		query            string
		wantFeatureCount int
		wantNilQueryNode bool
	}{
		{
			name:             "Valid non-empty query parses and searches",
			query:            "name:foo",
			wantFeatureCount: 2,
			wantNilQueryNode: false,
		},
		{
			name:             "Empty query bypasses parseQuery and fetches all features",
			query:            "",
			wantFeatureCount: 2,
			wantNilQueryNode: true,
		},
		{
			name:             "Whitespace query bypasses parseQuery and fetches all features",
			query:            "   ",
			wantFeatureCount: 2,
			wantNilQueryNode: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMockBackendAdapterForEventProducer([]*backend.FeaturePage{
				{
					Data: []backend.Feature{
						newTestBackendFeature("f1", "Feature 1"),
					},
					Metadata: backend.PageMetadataWithTotal{
						Total:         1000,
						NextPageToken: new("token-1"),
					},
				},
				{
					Data: []backend.Feature{
						newTestBackendFeature("f2", "Feature 2"),
					},
					Metadata: backend.PageMetadataWithTotal{
						Total:         1000,
						NextPageToken: nil,
					},
				},
			})

			adapter := NewEventProducerDiffer(mock)
			result, err := adapter.FetchFeatures(context.Background(), tc.query)
			if err != nil {
				t.Fatalf("FetchFeatures(%q) unexpected error: %v", tc.query, err)
			}
			if result.UserError != nil {
				t.Fatalf("FetchFeatures(%q) unexpected user error: %v", tc.query, result.UserError)
			}
			if len(result.Features) != tc.wantFeatureCount {
				t.Errorf("FetchFeatures(%q) expected %d features, got %d",
					tc.query, tc.wantFeatureCount, len(result.Features))
			}
			if len(mock.FeaturesSearchReqs) == 0 {
				t.Fatalf("FetchFeatures(%q) expected adapter calls, got 0", tc.query)
			}
			if tc.wantNilQueryNode && mock.FeaturesSearchReqs[0].QueryNode != nil {
				t.Errorf("FetchFeatures(%q) expected nil QueryNode, got non-nil", tc.query)
			}
			if !tc.wantNilQueryNode && mock.FeaturesSearchReqs[0].QueryNode == nil {
				t.Errorf("FetchFeatures(%q) expected non-nil QueryNode, got nil", tc.query)
			}
		})
	}
}

func TestEventProducerDiffer_FetchFeatures_Malformed(t *testing.T) {
	mock := newMockBackendAdapterForEventProducer(nil)
	adapter := NewEventProducerDiffer(mock)
	query := "id:(unclosed"

	result, err := adapter.FetchFeatures(context.Background(), query)
	if err != nil {
		t.Fatalf("FetchFeatures(%q) unexpected error: %v", query, err)
	}
	if result.UserError == nil {
		t.Fatalf("FetchFeatures(%q) expected UserError, got nil", query)
	}
	if len(result.UserError.QueryErrors) == 0 {
		t.Fatalf("FetchFeatures(%q) expected QueryErrors in UserError, got empty", query)
	}
	if result.UserError.QueryErrors[0].Code != workertypes.SummaryQueryErrorCodeQueryGrammar {
		t.Errorf("FetchFeatures(%q) UserError code = %q, want %q",
			query, result.UserError.QueryErrors[0].Code, workertypes.SummaryQueryErrorCodeQueryGrammar)
	}
	if len(mock.FeaturesSearchReqs) != 0 {
		t.Errorf("FetchFeatures(%q) expected 0 calls to adapter on grammar error, got %d",
			query, len(mock.FeaturesSearchReqs))
	}
}

type mockBatchEventProducerSpannerClient struct {
	listAllSavedSearchesCalled bool
	listAllSavedSearchesResp   []gcpspanner.SavedSearchBriefDetails
	listAllSavedSearchesErr    error
}

func (m *mockBatchEventProducerSpannerClient) ListAllSavedSearches(
	_ context.Context) ([]gcpspanner.SavedSearchBriefDetails, error) {
	m.listAllSavedSearchesCalled = true

	return m.listAllSavedSearchesResp, m.listAllSavedSearchesErr
}

func TestBatchEventProducer_ListAllSavedSearches(t *testing.T) {
	tests := []struct {
		name           string
		mockResp       []gcpspanner.SavedSearchBriefDetails
		mockErr        error
		wantSearchJobs []workertypes.SearchJob
		wantErr        bool
	}{
		{
			name: "success with searches",
			mockResp: []gcpspanner.SavedSearchBriefDetails{
				{ID: "s1", Name: "Search 1", Query: "q1"},
				{ID: "s2", Name: "Search 2", Query: "q2"},
			},
			mockErr: nil,
			wantSearchJobs: []workertypes.SearchJob{
				{ID: "s1", Name: "Search 1", Query: "q1"},
				{ID: "s2", Name: "Search 2", Query: "q2"},
			},
			wantErr: false,
		},
		{
			name:           "success empty list",
			mockResp:       []gcpspanner.SavedSearchBriefDetails{},
			mockErr:        nil,
			wantSearchJobs: []workertypes.SearchJob{},
			wantErr:        false,
		},
		{
			name:           "lister error",
			mockResp:       nil,
			mockErr:        errors.New("db error"),
			wantSearchJobs: nil,
			wantErr:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := new(mockBatchEventProducerSpannerClient)
			mock.listAllSavedSearchesResp = tc.mockResp
			mock.listAllSavedSearchesErr = tc.mockErr

			adapter := NewBatchEventProducer(mock)

			searchJobs, err := adapter.ListAllSavedSearches(context.Background())

			if (err != nil) != tc.wantErr {
				t.Errorf("ListAllSavedSearchIDs() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !mock.listAllSavedSearchesCalled {
				t.Fatal("ListAllSavedSearchIDs not called")
			}

			if diff := cmp.Diff(tc.wantSearchJobs, searchJobs); diff != "" {
				t.Errorf("ListAllSavedSearchIDs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
