// Copyright 2025 Google LLC
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
	"github.com/GoogleChrome/webstatus.dev/lib/generic"
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
			name:             "Daily maps to Immediate (as per implementation)",
			freq:             workertypes.FrequencyDaily,
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
			mockPublishResp: generic.ValuePtr("new-event-id"),
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
				t.Errorf("PublishSavedSearchNotificationEvent called = %v, expected %v", mock.publishEventCalled, tc.expectCall)
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
		wantErr          bool
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
			wantErr: false,
		},
		{
			name:             "Spanner error",
			freq:             workertypes.FrequencyDaily,
			wantSnapshotType: gcpspanner.SavedSearchSnapshotTypeImmediate,
			mockResp:         nil,
			mockErr:          errors.New("db error"),
			wantInfo:         nil,
			wantErr:          true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := new(mockSpannerClient)
			mock.getLatestEventResp = tc.mockResp
			mock.getLatestEventErr = tc.mockErr

			adapter := NewEventProducer(mock)

			info, err := adapter.GetLatestEvent(context.Background(), tc.freq, "search-1")

			if (err != nil) != tc.wantErr {
				t.Errorf("GetLatestEvent() error = %v, wantErr %v", err, tc.wantErr)
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

func TestEventProducerDiffer_FetchFeatures(t *testing.T) {
	mock := new(mockBackendAdapterForEventProducer)
	feature1 := new(backend.Feature)
	feature1.FeatureId = "f1"
	feature1.Name = "Feature 1"
	feature2 := new(backend.Feature)
	feature2.FeatureId = "f2"
	feature2.Name = "Feature 2"
	mock.featuresSearchPages = []*backend.FeaturePage{
		// First page
		{
			Data: []backend.Feature{
				*feature1,
			},
			Metadata: backend.PageMetadataWithTotal{
				NextPageToken: generic.ValuePtr("token-1"),
				Total:         1000,
			},
		},
		// Second page
		{
			Data: []backend.Feature{
				*feature2,
			},
			Metadata: backend.PageMetadataWithTotal{
				Total:         1000,
				NextPageToken: nil, // End of iteration

			},
		},
	}

	adapter := NewEventProducerDiffer(mock)

	// A simple query that parses successfully
	query := "name:foo"
	features, err := adapter.FetchFeatures(context.Background(), query)
	if err != nil {
		t.Fatalf("FetchFeatures() unexpected error: %v", err)
	}

	if len(features) != 2 {
		t.Errorf("Expected 2 features (across 2 pages), got %d", len(features))
	}
	if features[0].FeatureId != "f1" || features[1].FeatureId != "f2" {
		t.Error("Feature ID mismatch in results")
	}

	if len(mock.FeaturesSearchReqs) != 2 {
		t.Errorf("Expected 2 calls to FeaturesSearch, got %d", len(mock.FeaturesSearchReqs))
	}
	// First call should have nil token
	if mock.FeaturesSearchReqs[0].PageToken != nil {
		t.Error("First page token should be nil")
	}
	// Second call should have token-1
	if mock.FeaturesSearchReqs[1].PageToken == nil || *mock.FeaturesSearchReqs[1].PageToken != "token-1" {
		t.Error("Second page token mismatch")
	}
}
