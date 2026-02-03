// Copyright 2026 Google LLC
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

package gcppubsubadapters

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync"
	"testing"
	"time"

	batchrefreshv1 "github.com/GoogleChrome/webstatus.dev/lib/event/batchrefreshtrigger/v1"
	refreshv1 "github.com/GoogleChrome/webstatus.dev/lib/event/refreshsearchcommand/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/google/go-cmp/cmp"
)

// --- Mocks ---

type mockSearchHandler struct {
	calls []searchCall
	mu    sync.Mutex
	err   error
}

type searchCall struct {
	SearchID  string
	Query     string
	Frequency workertypes.JobFrequency
	TriggerID string
}

func (m *mockSearchHandler) ProcessSearch(_ context.Context, searchID, query string,
	freq workertypes.JobFrequency, triggerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, searchCall{searchID, query, freq, triggerID})

	return m.err
}

type mockBatchHandler struct {
	calls []batchCall
	mu    sync.Mutex
	err   error
}

type batchCall struct {
	TriggerID string
	Frequency workertypes.JobFrequency
}

func (m *mockBatchHandler) ProcessBatchUpdate(_ context.Context, triggerID string,
	freq workertypes.JobFrequency) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, batchCall{triggerID, freq})

	return m.err
}

type mockSubscriber struct {
	handlers map[string]func(context.Context, string, []byte) error
	mu       sync.Mutex
	// block allows us to simulate a long-running Subscribe call so RunGroup doesn't exit immediately
	block chan struct{}
}

func (m *mockSubscriber) Subscribe(ctx context.Context, subID string,
	handler func(context.Context, string, []byte) error) error {
	m.mu.Lock()
	if m.handlers == nil {
		m.handlers = make(map[string]func(context.Context, string, []byte) error)
	}
	m.handlers[subID] = handler
	m.mu.Unlock()

	// Simulate blocking behavior of a real subscriber logic
	if m.block != nil {
		select {
		case <-m.block:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

type mockPublisher struct {
	publishedData  []byte
	publishedTopic string
	err            error
}

func (m *mockPublisher) Publish(_ context.Context, topicID string, data []byte) (string, error) {
	m.publishedData = data
	m.publishedTopic = topicID

	return "msg-id", m.err
}

// --- Tests ---

type testEnv struct {
	searchHandler *mockSearchHandler
	batchHandler  *mockBatchHandler
	subscriber    *mockSubscriber
	adapter       *EventProducerSubscriberAdapter
	searchFn      func(context.Context, string, []byte) error
	batchFn       func(context.Context, string, []byte) error
	stop          func()
}

func setupTestAdapter(t *testing.T) *testEnv {
	t.Helper()
	searchHandler := new(mockSearchHandler)
	batchHandler := new(mockBatchHandler)
	subscriber := &mockSubscriber{block: make(chan struct{}), mu: sync.Mutex{}, handlers: nil}
	config := SubscriberConfig{
		SearchSubscriptionID:      "search-sub",
		BatchUpdateSubscriptionID: "batch-sub",
	}

	adapter := NewEventProducerSubscriberAdapter(searchHandler, batchHandler, subscriber, config)

	// Run Subscribe in a goroutine because it blocks
	ctx, cancel := context.WithCancel(context.Background())

	errChan := make(chan error)
	go func() {
		errChan <- adapter.Subscribe(ctx)
	}()

	// Wait briefly for RunGroup to start and handlers to be registered
	time.Sleep(50 * time.Millisecond)

	subscriber.mu.Lock()
	searchFn := subscriber.handlers["search-sub"]
	batchFn := subscriber.handlers["batch-sub"]
	subscriber.mu.Unlock()

	if searchFn == nil || batchFn == nil {
		cancel()
		t.Fatal("Subscribe did not register handlers for both subscriptions")
	}

	return &testEnv{
		searchHandler: searchHandler,
		batchHandler:  batchHandler,
		subscriber:    subscriber,
		adapter:       adapter,
		searchFn:      searchFn,
		batchFn:       batchFn,
		stop: func() {
			close(subscriber.block) // Unblock the subscriber
			cancel()                // Cancel the context
			<-errChan               // Wait for adapter.Subscribe to return
		},
	}
}

func TestSubscribe_RoutesRefreshSearchCommand(t *testing.T) {
	env := setupTestAdapter(t)
	defer env.stop()

	refreshCmd := refreshv1.RefreshSearchCommand{
		SearchID:  "s1",
		Query:     "q1",
		Frequency: "IMMEDIATE",
		Timestamp: time.Time{},
	}
	ceWrapper := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "RefreshSearchCommand",
		"data":       refreshCmd,
	}
	ceBytes, _ := json.Marshal(ceWrapper)

	if err := env.searchFn(context.Background(), "msg-1", ceBytes); err != nil {
		t.Errorf("searchFn failed: %v", err)
	}

	if len(env.searchHandler.calls) != 1 {
		t.Fatalf("Expected 1 search call, got %d", len(env.searchHandler.calls))
	}

	expectedCall := searchCall{
		SearchID:  "s1",
		Query:     "q1",
		Frequency: workertypes.FrequencyImmediate,
		TriggerID: "msg-1",
	}

	if diff := cmp.Diff(expectedCall, env.searchHandler.calls[0]); diff != "" {
		t.Errorf("Search call mismatch (-want +got):\n%s", diff)
	}
}

func TestSubscribe_RoutesBatchUpdate(t *testing.T) {
	env := setupTestAdapter(t)
	defer env.stop()

	batchTrig := batchrefreshv1.BatchRefreshTrigger{
		Frequency: "WEEKLY",
	}
	ceWrapperBatch := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "BatchRefreshTrigger",
		"data":       batchTrig,
	}
	ceBytesBatch, _ := json.Marshal(ceWrapperBatch)

	if err := env.batchFn(context.Background(), "msg-2", ceBytesBatch); err != nil {
		t.Errorf("batchFn failed: %v", err)
	}

	if len(env.batchHandler.calls) != 1 {
		t.Fatalf("Expected 1 batch call, got %d", len(env.batchHandler.calls))
	}

	expectedCall := batchCall{
		TriggerID: "msg-2",
		Frequency: workertypes.FrequencyWeekly,
	}

	if diff := cmp.Diff(expectedCall, env.batchHandler.calls[0]); diff != "" {
		t.Errorf("Batch call mismatch (-want +got):\n%s", diff)
	}
}

func TestSubscribe_RoutesSearchConfigurationChanged(t *testing.T) {
	env := setupTestAdapter(t)
	defer env.stop()

	// We construct the payload manually for the test execution
	configEventPayload := map[string]interface{}{
		"search_id":   "s2",
		"query":       "q2",
		"user_id":     "user-1",
		"timestamp":   "0001-01-01T00:00:00Z",
		"is_creation": false,
		"frequency":   "IMMEDIATE",
	}

	ceWrapperConfig := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "SearchConfigurationChangedEvent",
		"data":       configEventPayload,
	}
	ceBytesConfig, _ := json.Marshal(ceWrapperConfig)

	if err := env.searchFn(context.Background(), "msg-3", ceBytesConfig); err != nil {
		t.Errorf("searchFn (config event) failed: %v", err)
	}

	if len(env.searchHandler.calls) != 1 {
		t.Fatalf("Expected 1 search call, got %d", len(env.searchHandler.calls))
	}

	expectedCall := searchCall{
		SearchID:  "s2",
		Query:     "q2",
		Frequency: workertypes.FrequencyImmediate,
		TriggerID: "msg-3",
	}

	if diff := cmp.Diff(expectedCall, env.searchHandler.calls[0]); diff != "" {
		t.Errorf("Search call mismatch (-want +got):\n%s", diff)
	}
}

func TestPublisher_Publish(t *testing.T) {
	publisher := new(mockPublisher)
	adapter := NewEventProducerPublisherAdapter(publisher, "topic-1")
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	req := workertypes.PublishEventRequest{
		EventID:       "evt-1",
		SearchID:      "search-1",
		Query:         "query-1",
		Frequency:     workertypes.FrequencyImmediate,
		Reasons:       []workertypes.Reason{workertypes.ReasonDataUpdated},
		Summary:       []byte(`{"added": 1}`),
		StateID:       "state-id-1",
		DiffID:        "diff-id-1",
		StateBlobPath: "gs://bucket/state-blob",
		DiffBlobPath:  "gs://bucket/diff-blob",
		GeneratedAt:   now,
	}

	_, err := adapter.Publish(context.Background(), req)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	if publisher.publishedTopic != "topic-1" {
		t.Errorf("Topic mismatch: got %s, want topic-1", publisher.publishedTopic)
	}

	var actualEnvelope map[string]interface{}
	if err := json.Unmarshal(publisher.publishedData, &actualEnvelope); err != nil {
		t.Fatalf("Failed to unmarshal published data: %v", err)
	}

	expectedEnvelope := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "FeatureDiffEvent",
		"data": map[string]interface{}{
			"event_id":  "evt-1",
			"search_id": "search-1",
			"query":     "query-1",
			// go encodes/decodes []byte as base64 strings
			"summary":         base64.StdEncoding.EncodeToString([]byte(`{"added": 1}`)),
			"state_id":        "state-id-1",
			"diff_id":         "diff-id-1",
			"state_blob_path": "gs://bucket/state-blob",
			"diff_blob_path":  "gs://bucket/diff-blob",
			"reasons":         []interface{}{"DATA_UPDATED"},
			"generated_at":    now.Format(time.RFC3339),
			"frequency":       "IMMEDIATE",
		},
	}

	if diff := cmp.Diff(expectedEnvelope, actualEnvelope); diff != "" {
		t.Errorf("Payload mismatch (-want +got):\n%s", diff)
	}
}
