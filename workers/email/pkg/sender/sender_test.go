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

package sender

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/google/go-cmp/cmp"
)

// --- Mocks ---

type mockEmailSender struct {
	sentCalls []sentCall
	sendErr   error
}

type sentCall struct {
	to      string
	subject string
	body    string
}

func (m *mockEmailSender) Send(_ context.Context, to, subject, body string) error {
	m.sentCalls = append(m.sentCalls, sentCall{to, subject, body})

	return m.sendErr
}

type mockChannelStateManager struct {
	successCalls []string // channelIDs
	failureCalls []failureCall
	recordErr    error
}

type failureCall struct {
	channelID string
	err       error
}

func (m *mockChannelStateManager) RecordSuccess(_ context.Context, channelID string) error {
	m.successCalls = append(m.successCalls, channelID)

	return m.recordErr
}

func (m *mockChannelStateManager) RecordFailure(_ context.Context, channelID string, err error) error {
	m.failureCalls = append(m.failureCalls, failureCall{channelID, err})

	return m.recordErr
}

type mockTemplateRenderer struct {
	renderSubject string
	renderBody    string
	renderErr     error
	renderInput   workertypes.EmailDeliveryJob
}

func (m *mockTemplateRenderer) RenderDigest(job workertypes.EmailDeliveryJob) (string, string, error) {
	m.renderInput = job

	return m.renderSubject, m.renderBody, m.renderErr
}

func testGeneratedAt() time.Time {
	return time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
}

const testChannelID = "chan-1"

func testMetadata() workertypes.DeliveryMetadata {
	return workertypes.DeliveryMetadata{
		EventID:     "event-1",
		SearchID:    "search-1",
		Query:       "query-string",
		Frequency:   workertypes.FrequencyMonthly,
		GeneratedAt: testGeneratedAt(),
	}
}

// --- Tests ---

func TestProcessMessage_Success(t *testing.T) {
	ctx := context.Background()
	job := workertypes.EmailDeliveryJob{
		SubscriptionID: "sub-1",
		Metadata:       testMetadata(),
		RecipientEmail: "user@example.com",
		SummaryRaw:     []byte("{}"),
		ChannelID:      "chan-1",
	}

	sender := new(mockEmailSender)
	stateManager := new(mockChannelStateManager)
	renderer := new(mockTemplateRenderer)
	renderer.renderSubject = "Subject"
	renderer.renderBody = "Body"

	h := NewSender(sender, stateManager, renderer)

	err := h.ProcessMessage(ctx, job)
	if err != nil {
		t.Fatalf("ProcessMessage failed: %v", err)
	}

	// Verify Renderer Input
	if diff := cmp.Diff(job, renderer.renderInput); diff != "" {
		t.Errorf("Renderer input mismatch (-want +got):\n%s", diff)
	}

	// Verify Send
	if len(sender.sentCalls) != 1 {
		t.Fatalf("Expected 1 email sent, got %d", len(sender.sentCalls))
	}
	if sender.sentCalls[0].to != "user@example.com" {
		t.Errorf("Recipient mismatch: %s", sender.sentCalls[0].to)
	}

	// Verify State
	if len(stateManager.successCalls) != 1 {
		t.Errorf("Expected 1 success record, got %d", len(stateManager.successCalls))
	}
	if stateManager.successCalls[0] != testChannelID {
		t.Errorf("Success recorded for wrong channel: %s", stateManager.successCalls[0])
	}
}

func TestProcessMessage_RenderError(t *testing.T) {
	ctx := context.Background()
	job := workertypes.EmailDeliveryJob{
		SubscriptionID: "sub-1",
		Metadata:       testMetadata(),
		RecipientEmail: "user@example.com",
		SummaryRaw:     []byte("{}"),
		ChannelID:      "chan-1",
	}

	sender := new(mockEmailSender)
	stateManager := new(mockChannelStateManager)
	renderer := new(mockTemplateRenderer)
	renderer.renderErr = errors.New("template error")

	h := NewSender(sender, stateManager, renderer)

	// Should return nil (ACK) for rendering error (assuming permanent for now)
	if err := h.ProcessMessage(ctx, job); err != nil {
		t.Errorf("Expected nil error for render failure, got %v", err)
	}

	// Should record failure
	if len(stateManager.failureCalls) != 1 {
		t.Fatal("Expected failure recording")
	}

	// Should NOT send
	if len(sender.sentCalls) > 0 {
		t.Error("Should not send email on render error")
	}
}

func TestProcessMessage_SendError(t *testing.T) {
	ctx := context.Background()
	job := workertypes.EmailDeliveryJob{
		SubscriptionID: "sub-1",
		Metadata:       testMetadata(),
		RecipientEmail: "user@example.com",
		SummaryRaw:     []byte("{}"),
		ChannelID:      "chan-1",
	}

	sendErr := errors.New("smtp timeout")
	sender := &mockEmailSender{sendErr: sendErr, sentCalls: nil}
	stateManager := new(mockChannelStateManager)
	renderer := new(mockTemplateRenderer)
	renderer.renderSubject = "S"
	renderer.renderBody = "B"

	h := NewSender(sender, stateManager, renderer)

	// Should return error (NACK) for send failure to allow retry
	err := h.ProcessMessage(ctx, job)
	if !errors.Is(err, sendErr) {
		t.Errorf("Expected send error to propagate, got %v", err)
	}

	// Should record failure in DB as well
	if len(stateManager.failureCalls) != 1 {
		t.Fatal("Expected failure recording")
	}
	if stateManager.failureCalls[0].channelID != testChannelID {
		t.Errorf("Recorded failure for wrong channel")
	}
}
