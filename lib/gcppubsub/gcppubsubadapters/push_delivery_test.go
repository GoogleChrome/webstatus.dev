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

package gcppubsubadapters

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync"
	"testing"
	"time"

	featurediffv1 "github.com/GoogleChrome/webstatus.dev/lib/event/featurediff/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/google/go-cmp/cmp"
)

// --- Mocks ---

type mockPushDeliveryPublisher struct {
	publishedData  []byte
	publishedTopic string
	err            error
	mu             sync.Mutex // Added mutex for concurrent access
}

func (m *mockPushDeliveryPublisher) Publish(_ context.Context, topicID string, data []byte) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishedData = data
	m.publishedTopic = topicID

	return "msg-id", m.err
}

type mockDispatcher struct {
	calls []processEventCall
	mu    sync.Mutex
	err   error
}

type processEventCall struct {
	Metadata workertypes.DispatchEventMetadata
	Summary  []byte
}

func (m *mockDispatcher) ProcessEvent(_ context.Context,
	metadata workertypes.DispatchEventMetadata, summary []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, processEventCall{Metadata: metadata, Summary: summary})

	return m.err
}

type mockPushDeliverySubscriber struct {
	handlers map[string]func(context.Context, string, []byte) error
	mu       sync.Mutex
	// block allows us to simulate a long-running Subscribe call so RunGroup doesn't exit immediately
	block chan struct{}
}

func (m *mockPushDeliverySubscriber) Subscribe(ctx context.Context, subID string,
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

// --- Tests ---

func TestPushDeliveryPublisher_PublishEmailJob(t *testing.T) {
	mockPub := new(mockPushDeliveryPublisher)
	publisher := NewPushDeliveryPublisher(mockPub, "email-topic")

	job := workertypes.EmailDeliveryJob{
		SubscriptionID: "sub-1",
		RecipientEmail: "test@example.com",
		SummaryRaw:     []byte(`{"text": "Test Body"}`),
		Metadata: workertypes.DeliveryMetadata{
			EventID:     "event-1",
			SearchID:    "search-1",
			Query:       "query-string",
			Frequency:   workertypes.FrequencyMonthly,
			GeneratedAt: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	err := publisher.PublishEmailJob(context.Background(), job)
	if err != nil {
		t.Fatalf("PublishEmailJob failed: %v", err)
	}

	if mockPub.publishedTopic != "email-topic" {
		t.Errorf("Topic mismatch: got %s, want email-topic", mockPub.publishedTopic)
	}

	var actualEnvelope map[string]interface{}
	if err := json.Unmarshal(mockPub.publishedData, &actualEnvelope); err != nil {
		t.Fatalf("Failed to unmarshal published data: %v", err)
	}

	expectedEnvelope := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "EmailJobEvent",
		"data": map[string]interface{}{
			"subscription_id": "sub-1",
			"recipient_email": "test@example.com",
			"summary_raw":     base64.StdEncoding.EncodeToString([]byte(`{"text": "Test Body"}`)),
			"metadata": map[string]interface{}{
				"event_id":     "event-1",
				"search_id":    "search-1",
				"query":        "query-string",
				"frequency":    "MONTHLY",
				"generated_at": "2025-01-01T12:00:00Z",
			},
		},
	}

	if diff := cmp.Diff(expectedEnvelope, actualEnvelope); diff != "" {
		t.Errorf("Email job mismatch (-want +got):\n%s", diff)
	}
}

type pushDeliveryTestEnv struct {
	dispatcher    *mockDispatcher
	subscriber    *mockPushDeliverySubscriber
	adapter       *PushDeliverySubscriberAdapter
	featureDiffFn func(context.Context, string, []byte) error
	stop          func()
}

func setupPushDeliveryTestAdapter(t *testing.T) *pushDeliveryTestEnv {
	t.Helper()
	dispatcher := new(mockDispatcher)
	subscriber := &mockPushDeliverySubscriber{block: make(chan struct{}), mu: sync.Mutex{}, handlers: nil}
	subscriptionID := "feature-diff-sub"

	adapter := NewPushDeliverySubscriberAdapter(dispatcher, subscriber, subscriptionID)

	ctx, cancel := context.WithCancel(context.Background())

	errChan := make(chan error, 1) // Buffered channel to prevent goroutine leak on t.Fatal
	go func() {
		errChan <- adapter.Subscribe(ctx)
	}()

	// Wait briefly for Subscribe to start and handler to be registered
	time.Sleep(50 * time.Millisecond)

	subscriber.mu.Lock()
	featureDiffFn := subscriber.handlers[subscriptionID]
	subscriber.mu.Unlock()

	if featureDiffFn == nil {
		cancel()
		close(subscriber.block)
		<-errChan
		t.Fatal("Subscribe did not register handler for subscription")
	}

	return &pushDeliveryTestEnv{
		dispatcher:    dispatcher,
		subscriber:    subscriber,
		adapter:       adapter,
		featureDiffFn: featureDiffFn,
		stop: func() {
			close(subscriber.block) // Unblock the subscriber
			cancel()                // Cancel the context
			<-errChan               // Wait for adapter.Subscribe to return
		},
	}
}

func TestPushDeliverySubscriber_RoutesFeatureDiffEvent(t *testing.T) {
	env := setupPushDeliveryTestAdapter(t)
	defer env.stop()

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	featureDiffEvent := featurediffv1.FeatureDiffEvent{
		EventID:       "evt-1",
		SearchID:      "s1",
		Query:         "q1",
		Summary:       []byte(`{"added": 1}`),
		StateID:       "state-id-1",
		StateBlobPath: "gs://bucket/state-blob",
		DiffID:        "diff-id-1",
		DiffBlobPath:  "gs://bucket/diff-blob",
		GeneratedAt:   now,
		Frequency:     featurediffv1.FrequencyMonthly,
		Reasons:       []featurediffv1.Reason{featurediffv1.ReasonDataUpdated},
	}
	ceWrapper := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "FeatureDiffEvent",
		"data":       featureDiffEvent,
	}
	ceBytes, _ := json.Marshal(ceWrapper)

	if err := env.featureDiffFn(context.Background(), "msg-1", ceBytes); err != nil {
		t.Errorf("featureDiffFn failed: %v", err)
	}

	if len(env.dispatcher.calls) != 1 {
		t.Fatalf("Expected 1 dispatcher call, got %d", len(env.dispatcher.calls))
	}

	expectedMetadata := workertypes.DispatchEventMetadata{
		EventID:     "evt-1",
		SearchID:    "s1",
		Query:       "q1",
		Frequency:   workertypes.FrequencyMonthly,
		GeneratedAt: now,
	}

	// Compare summary as string since cmp.Diff might struggle with []byte directly within interface{}
	actualSummaryStr := string(env.dispatcher.calls[0].Summary)
	expectedSummaryStr := string(featureDiffEvent.Summary)

	if diff := cmp.Diff(expectedMetadata, env.dispatcher.calls[0].Metadata); diff != "" {
		t.Errorf("Dispatcher metadata mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(expectedSummaryStr, actualSummaryStr); diff != "" {
		t.Errorf("Dispatcher summary mismatch (-want +got):\n%s", diff)
	}
}
