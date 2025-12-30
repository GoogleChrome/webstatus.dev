// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may not use this file except in compliance with the License.
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
	"encoding/json"
	"sync"
	"testing"
	"time"

	v1 "github.com/GoogleChrome/webstatus.dev/lib/event/emailjob/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/google/go-cmp/cmp"
)

// --- Mocks ---

type mockEmailWorkerMessageHandler struct {
	calls []workertypes.IncomingEmailDeliveryJob
	mu    sync.Mutex
	err   error
}

func (m *mockEmailWorkerMessageHandler) ProcessMessage(
	_ context.Context, job workertypes.IncomingEmailDeliveryJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, job)

	return m.err
}

// --- Tests ---

type emailTestEnv struct {
	sender   *mockEmailWorkerMessageHandler
	adapter  *EmailWorkerSubscriberAdapter
	handleFn func(context.Context, string, []byte) error
	stop     func()
}

func setupEmailTestAdapter(t *testing.T) *emailTestEnv {
	t.Helper()
	sender := new(mockEmailWorkerMessageHandler)
	subscriber := &mockSubscriber{block: make(chan struct{}), mu: sync.Mutex{}, handlers: nil}
	subscriptionID := "email-sub"

	adapter := NewEmailWorkerSubscriberAdapter(sender, subscriber, subscriptionID)

	ctx, cancel := context.WithCancel(context.Background())

	errChan := make(chan error)
	go func() {
		errChan <- adapter.Subscribe(ctx)
	}()

	// Wait briefly for Subscribe to start and register the handler
	time.Sleep(50 * time.Millisecond)

	subscriber.mu.Lock()
	handleFn := subscriber.handlers[subscriptionID]
	subscriber.mu.Unlock()

	if handleFn == nil {
		cancel()
		t.Fatalf("Subscribe did not register a handler for subscription %s", subscriptionID)
	}

	return &emailTestEnv{
		sender:  sender,
		adapter: adapter,
		handleFn: func(ctx context.Context, msgID string, data []byte) error {
			return handleFn(ctx, msgID, data)
		},
		stop: func() {
			close(subscriber.block)
			cancel()
			<-errChan
		},
	}
}

func TestEmailWorkerSubscriberAdapter_RoutesEmailJobEvent(t *testing.T) {
	env := setupEmailTestAdapter(t)
	defer env.stop()

	now := time.Now()
	emailJobEvent := v1.EmailJobEvent{
		SubscriptionID: "sub-123",
		RecipientEmail: "test@example.com",
		ChannelID:      "chan-456",
		SummaryRaw:     []byte(`{"key":"value"}`),
		Metadata: v1.EmailJobEventMetadata{
			EventID:     "event-789",
			SearchID:    "search-abc",
			Query:       "is:open",
			Frequency:   v1.FrequencyMonthly,
			GeneratedAt: now,
		},
		Triggers: []v1.JobTrigger{
			v1.BrowserImplementationAnyComplete,
		},
	}

	ceWrapper := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "EmailJobEvent",
		"data":       emailJobEvent,
	}
	ceBytes, err := json.Marshal(ceWrapper)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	msgID := "msg-xyz"
	if err := env.handleFn(context.Background(), msgID, ceBytes); err != nil {
		t.Errorf("handleFn failed: %v", err)
	}

	if len(env.sender.calls) != 1 {
		t.Fatalf("Expected 1 call to ProcessMessage, got %d", len(env.sender.calls))
	}

	expectedJob := workertypes.IncomingEmailDeliveryJob{
		EmailDeliveryJob: workertypes.EmailDeliveryJob{
			SubscriptionID: "sub-123",
			RecipientEmail: "test@example.com",
			ChannelID:      "chan-456",
			SummaryRaw:     []byte(`{"key":"value"}`),
			Metadata: workertypes.DeliveryMetadata{
				EventID:     "event-789",
				SearchID:    "search-abc",
				Query:       "is:open",
				Frequency:   workertypes.FrequencyMonthly,
				GeneratedAt: now,
			},
			Triggers: []workertypes.JobTrigger{
				workertypes.BrowserImplementationAnyComplete,
			},
		},
		EmailEventID: msgID,
	}

	if diff := cmp.Diff(expectedJob, env.sender.calls[0]); diff != "" {
		t.Errorf("ProcessMessage call mismatch (-want +got):\n%s", diff)
	}
}

func TestEmailWorkerSubscriberAdapter_ReturnsErrorOnUnknownEvent(t *testing.T) {
	env := setupEmailTestAdapter(t)
	defer env.stop()

	ceWrapper := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "UnknownEvent",
		"data":       map[string]string{"foo": "bar"},
	}
	ceBytes, _ := json.Marshal(ceWrapper)

	err := env.handleFn(context.Background(), "msg-1", ceBytes)
	if err == nil {
		t.Error("Expected an error for an unknown event, but got nil")
	}
}
