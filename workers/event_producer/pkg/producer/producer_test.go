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

package producer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/GoogleChrome/webstatus.dev/workers/event_producer/pkg/differ"
	"github.com/google/go-cmp/cmp"
)

//
// Mocks for testing the EventProducer
//

type mockFeatureDiffer struct {
	runCalledWith struct {
		searchID           string
		query              string
		eventID            string
		previousStateBytes []byte
	}
	runReturns struct {
		result *differ.DiffResult
		err    error
	}
}

func (m *mockFeatureDiffer) Run(_ context.Context, searchID, query, eventID string,
	previousStateBytes []byte) (*differ.DiffResult, error) {
	m.runCalledWith.searchID = searchID
	m.runCalledWith.query = query
	m.runCalledWith.eventID = eventID
	m.runCalledWith.previousStateBytes = previousStateBytes

	return m.runReturns.result, m.runReturns.err
}

type storeCall struct {
	key  string
	dirs []string
}

type mockBlobStorage struct {
	storeCalls  map[string]storeCall
	storeErrors map[string]error
	getResults  map[string][]byte
	getErrors   map[string]error
}

func (m *mockBlobStorage) Store(_ context.Context, dirs []string, key string, _ []byte) (string, error) {
	if err, ok := m.storeErrors[key]; ok {
		return "", err
	}
	if m.storeCalls == nil {
		m.storeCalls = make(map[string]storeCall)
	}
	m.storeCalls[key] = storeCall{
		key:  key,
		dirs: dirs,
	}

	return "full-path/" + key, nil
}

func (m *mockBlobStorage) Get(_ context.Context, key string) ([]byte, error) {
	if err, ok := m.getErrors[key]; ok {
		return nil, err
	}

	return m.getResults[key], nil
}

type mockEventMetadataStore struct {
	publishEventCalledWith workertypes.PublishEventRequest
	publishEventReturns    error
	getLatestEventReturns  struct {
		info *workertypes.LatestEventInfo
		err  error
	}
	acquireLockReturns error
	releaseLockReturns error
}

func (m *mockEventMetadataStore) AcquireLock(_ context.Context, _ string, _ workertypes.JobFrequency, _ string,
	_ time.Duration) error {
	return m.acquireLockReturns
}

func (m *mockEventMetadataStore) ReleaseLock(_ context.Context, _ string, _ workertypes.JobFrequency, _ string) error {
	return m.releaseLockReturns
}

func (m *mockEventMetadataStore) PublishEvent(_ context.Context, req workertypes.PublishEventRequest) error {
	m.publishEventCalledWith = req

	return m.publishEventReturns
}

func (m *mockEventMetadataStore) GetLatestEvent(_ context.Context,
	_ workertypes.JobFrequency, _ string) (*workertypes.LatestEventInfo, error) {
	return m.getLatestEventReturns.info, m.getLatestEventReturns.err
}

type mockEventPublisher struct {
	publishCalledWith   workertypes.PublishEventRequest
	publishReturnID     string
	publishReturnsError error
}

func (m *mockEventPublisher) Publish(_ context.Context, req workertypes.PublishEventRequest) (string, error) {
	m.publishCalledWith = req

	return m.publishReturnID, m.publishReturnsError
}

func TestProcessSearch_Success(t *testing.T) {
	ctx := context.Background()
	searchID := "search-abc"
	triggerID := "trigger-123"
	query := "q=test"
	frequency := workertypes.JobFrequency("asap")
	expectedDate := time.Date(2000, time.April, 1, 1, 1, 1, 0, time.UTC)

	// 1. Define a helper struct for mocks to fix lll and function signature complexity
	type testMocks struct {
		differ *mockFeatureDiffer
		blob   *mockBlobStorage
		meta   *mockEventMetadataStore
		pub    *mockEventPublisher
	}

	type testCase struct {
		name        string
		setup       func(*testMocks)
		verify      func(*testing.T, *testMocks)
		expectedReq workertypes.PublishEventRequest
	}

	tests := []testCase{
		{
			name: "First run (cold start)",
			setup: func(m *testMocks) {
				m.meta.getLatestEventReturns.info = nil
				m.differ.runReturns.result = &differ.DiffResult{
					State:       differ.BlobArtifact{ID: "state-1", Bytes: []byte("state-data")},
					Diff:        differ.BlobArtifact{ID: "diff-1", Bytes: []byte("diff-data")},
					Summary:     []byte("summary"),
					Reasons:     []workertypes.Reason{workertypes.ReasonDataUpdated},
					GeneratedAt: expectedDate,
				}
			},
			expectedReq: workertypes.PublishEventRequest{
				EventID:       triggerID,
				SearchID:      searchID,
				StateID:       "state-1",
				StateBlobPath: "full-path/state-1.json",
				DiffID:        "diff-1",
				DiffBlobPath:  "full-path/diff-1.json",
				Query:         "q=test",
				Summary:       []byte("summary"),
				Reasons:       []workertypes.Reason{workertypes.ReasonDataUpdated},
				Frequency:     frequency,
				GeneratedAt:   expectedDate,
			},
			verify: func(t *testing.T, m *testMocks) {
				if m.differ.runCalledWith.previousStateBytes != nil {
					t.Error("expected previousStateBytes to be nil on first run")
				}
				if _, ok := m.blob.storeCalls["state-1.json"]; !ok {
					t.Error("expected state blob to be stored")
				}
				if _, ok := m.blob.storeCalls["diff-1.json"]; !ok {
					t.Error("expected diff blob to be stored")
				}
			},
		},
		{
			name: "Subsequent run with changes",
			setup: func(m *testMocks) {
				m.meta.getLatestEventReturns.info = &workertypes.LatestEventInfo{
					EventID:       "",
					StateBlobPath: "prev-state-0",
				}
				m.blob.getResults = map[string][]byte{
					"prev-state-0": []byte("old-state-data"),
				}
				m.differ.runReturns.result = &differ.DiffResult{
					State:       differ.BlobArtifact{ID: "state-2", Bytes: []byte("new-state-data")},
					Diff:        differ.BlobArtifact{ID: "diff-2", Bytes: []byte("new-diff-data")},
					Summary:     []byte("new-summary"),
					Reasons:     []workertypes.Reason{workertypes.ReasonQueryChanged},
					GeneratedAt: expectedDate,
				}
			},
			expectedReq: workertypes.PublishEventRequest{
				EventID:       triggerID,
				SearchID:      searchID,
				StateID:       "state-2",
				StateBlobPath: "full-path/state-2.json",
				DiffID:        "diff-2",
				DiffBlobPath:  "full-path/diff-2.json",
				Query:         "q=test",
				Summary:       []byte("new-summary"),
				Reasons:       []workertypes.Reason{workertypes.ReasonQueryChanged},
				Frequency:     "asap",
				GeneratedAt:   expectedDate,
			},
			verify: func(t *testing.T, m *testMocks) {
				if string(m.differ.runCalledWith.previousStateBytes) != "old-state-data" {
					t.Errorf("got prev state %s", m.differ.runCalledWith.previousStateBytes)
				}
				if _, ok := m.blob.storeCalls["state-2.json"]; !ok {
					t.Error("expected new state blob to be stored")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			mocks := &testMocks{
				differ: new(mockFeatureDiffer),
				blob:   new(mockBlobStorage),
				meta:   new(mockEventMetadataStore),
				pub:    new(mockEventPublisher),
			}
			tc.setup(mocks)

			// Execute
			producer := NewEventProducer(mocks.differ, mocks.blob, mocks.meta, mocks.pub)
			err := producer.ProcessSearch(ctx, searchID, query, frequency, triggerID)

			// Verify
			if err != nil {
				t.Fatalf("ProcessSearch() unexpected error: %v", err)
			}

			// Common verification for success cases
			if diff := cmp.Diff(tc.expectedReq, mocks.meta.publishEventCalledWith); diff != "" {
				t.Errorf("PublishEvent metadata mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.expectedReq, mocks.pub.publishCalledWith); diff != "" {
				t.Errorf("Publish notification mismatch (-want +got):\n%s", diff)
			}

			// Custom verification
			if tc.verify != nil {
				tc.verify(t, mocks)
			}
		})
	}
}

func TestProcessSearch_NoChanges(t *testing.T) {
	// Separated because the assertions are completely different
	// (checking that things did NOT happen).
	ctx := context.Background()
	differMock := new(mockFeatureDiffer)
	blobMock := new(mockBlobStorage)
	metaMock := new(mockEventMetadataStore)
	pubMock := new(mockEventPublisher)

	metaMock.getLatestEventReturns.info = nil
	metaMock.getLatestEventReturns.err = workertypes.ErrLatestEventNotFound
	differMock.runReturns.err = differ.ErrNoChangesDetected

	producer := NewEventProducer(differMock, blobMock, metaMock, pubMock)
	err := producer.ProcessSearch(ctx, "search-id", "q=test", "asap", "trigger-id")

	if err != nil {
		t.Fatalf("expected nil error for no changes, got %v", err)
	}
	if len(blobMock.storeCalls) > 0 {
		t.Error("expected no blobs to be stored")
	}
	if metaMock.publishEventCalledWith.EventID != "" {
		t.Error("expected no metadata published")
	}
	if pubMock.publishCalledWith.EventID != "" {
		t.Error("expected no notification published")
	}
}

func TestProcessSearch_Failures(t *testing.T) {
	ctx := context.Background()
	type testMocks struct {
		differ *mockFeatureDiffer
		blob   *mockBlobStorage
		meta   *mockEventMetadataStore
		pub    *mockEventPublisher
	}

	type testCase struct {
		name   string
		setup  func(*testMocks)
		verify func(*testing.T, *testMocks)
	}
	expectedDate := time.Date(2000, time.April, 1, 1, 1, 1, 0, time.UTC)

	tests := []testCase{
		{
			name: "GetLatestEvent fails",
			setup: func(m *testMocks) {
				m.meta.getLatestEventReturns.err = errors.New("db error")
			},
			verify: nil, // No extra verification needed
		},
		{
			name: "Get blob fails",
			setup: func(m *testMocks) {
				m.meta.getLatestEventReturns.info = &workertypes.LatestEventInfo{
					EventID:       "",
					StateBlobPath: "prev-state-x",
				}
				m.blob.getErrors = map[string]error{"prev-state-x": errors.New("gcs error")}
			},
			verify: nil,
		},
		{
			name: "Differ Run fails",
			setup: func(m *testMocks) {
				m.meta.getLatestEventReturns.info = nil
				m.differ.runReturns.err = errors.New("differ fatal error")
			},
			verify: nil,
		},
		{
			name: "Store state blob fails",
			setup: func(m *testMocks) {
				m.meta.getLatestEventReturns.info = nil
				m.differ.runReturns.result = &differ.DiffResult{
					State:       differ.BlobArtifact{ID: "state-fail", Bytes: []byte("state")},
					Diff:        differ.BlobArtifact{ID: "diff-fail", Bytes: []byte("diff")},
					Summary:     nil,
					Reasons:     nil,
					GeneratedAt: expectedDate,
				}
				m.blob.storeErrors = map[string]error{
					"state-fail.json": errors.New("storage error"),
				}
			},
			verify: func(t *testing.T, m *testMocks) {
				if _, ok := m.blob.storeCalls["diff-fail.json"]; ok {
					t.Error("should not have tried to store diff blob when state blob failed")
				}
				if m.meta.publishEventCalledWith.EventID != "" {
					t.Error("should not have published metadata")
				}
			},
		},
		{
			name: "Publish event metadata fails",
			setup: func(m *testMocks) {
				m.meta.getLatestEventReturns.info = nil
				m.differ.runReturns.result = &differ.DiffResult{
					State:       differ.BlobArtifact{ID: "state-1", Bytes: []byte("state-data")},
					Diff:        differ.BlobArtifact{ID: "diff-1", Bytes: []byte("diff-data")},
					Summary:     nil,
					Reasons:     nil,
					GeneratedAt: expectedDate,
				}
				m.meta.publishEventReturns = errors.New("metadata store error")
			},
			verify: func(t *testing.T, m *testMocks) {
				if _, ok := m.blob.storeCalls["state-1.json"]; !ok {
					t.Error("expected state blob to be stored even if metadata publish fails")
				}
			},
		},
		{
			name: "Publish event notification fails",
			setup: func(m *testMocks) {
				m.meta.getLatestEventReturns.info = nil
				m.differ.runReturns.result = &differ.DiffResult{
					State:       differ.BlobArtifact{ID: "state-1", Bytes: []byte("state-data")},
					Diff:        differ.BlobArtifact{ID: "diff-1", Bytes: []byte("diff-data")},
					Summary:     nil,
					Reasons:     nil,
					GeneratedAt: expectedDate,
				}
				m.pub.publishReturnsError = errors.New("pubsub error")
			},
			verify: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mocks := &testMocks{
				differ: new(mockFeatureDiffer),
				blob:   new(mockBlobStorage),
				meta:   new(mockEventMetadataStore),
				pub:    new(mockEventPublisher),
			}
			tc.setup(mocks)

			producer := NewEventProducer(mocks.differ, mocks.blob, mocks.meta, mocks.pub)
			err := producer.ProcessSearch(ctx, "search-abc", "q=test", "asap", "trigger-123")

			if err == nil {
				t.Error("ProcessSearch() expected error, got nil")
			}
			if tc.verify != nil {
				tc.verify(t, mocks)
			}
		})
	}
}
