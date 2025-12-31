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

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/google/go-cmp/cmp"
)

// --- Mocks ---

type mockEmailSender struct {
	sentCalls []sentCall
	sendErr   error
}

type sentCall struct {
	id      string
	to      string
	subject string
	body    string
}

func (m *mockEmailSender) Send(_ context.Context, id, to, subject, body string) error {
	m.sentCalls = append(m.sentCalls, sentCall{id, to, subject, body})

	return m.sendErr
}

type successCall struct {
	channelID    string
	emailEventID string
	timestamp    time.Time
}

type mockChannelStateManager struct {
	successCalls []successCall
	failureCalls []failureCall
	recordErr    error
}

type failureCall struct {
	channelID            string
	emailEventID         string
	err                  error
	isPermanentUserError bool
	timestamp            time.Time
}

func (m *mockChannelStateManager) RecordSuccess(_ context.Context, channelID string,
	timestamp time.Time, emailEventID string) error {
	m.successCalls = append(m.successCalls, successCall{channelID, emailEventID, timestamp})

	return m.recordErr
}

func (m *mockChannelStateManager) RecordFailure(_ context.Context, channelID string, err error,
	timestamp time.Time, isPermanentUserError bool, emailEventID string,
) error {
	m.failureCalls = append(m.failureCalls, failureCall{channelID, emailEventID, err, isPermanentUserError, timestamp})

	return m.recordErr
}

type mockTemplateRenderer struct {
	renderSubject string
	renderBody    string
	renderErr     error
	renderInput   workertypes.IncomingEmailDeliveryJob
}

func (m *mockTemplateRenderer) RenderDigest(job workertypes.IncomingEmailDeliveryJob) (string, string, error) {
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

func fakeNow() time.Time {
	return time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
}

// --- Tests ---

const testEmailEventID = "job-id"

func TestProcessMessage_Success(t *testing.T) {
	ctx := context.Background()
	job := workertypes.IncomingEmailDeliveryJob{
		EmailDeliveryJob: workertypes.EmailDeliveryJob{
			SubscriptionID: "sub-1",
			Metadata:       testMetadata(),
			RecipientEmail: "user@example.com",
			SummaryRaw:     []byte("{}"),
			ChannelID:      "chan-1",
			Triggers: []workertypes.JobTrigger{
				workertypes.BrowserImplementationAnyComplete,
			},
		},
		EmailEventID: testEmailEventID,
	}

	sender := new(mockEmailSender)
	stateManager := new(mockChannelStateManager)
	renderer := new(mockTemplateRenderer)
	renderer.renderSubject = "Subject"
	renderer.renderBody = "Body"

	h := NewSender(sender, stateManager, renderer)
	h.now = fakeNow

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
	if sender.sentCalls[0].id != testEmailEventID {
		t.Errorf("Event ID mismatch: %s", sender.sentCalls[0].id)
	}
	if sender.sentCalls[0].subject != "Subject" {
		t.Errorf("Subject mismatch: %s", sender.sentCalls[0].subject)
	}
	if sender.sentCalls[0].body != "Body" {
		t.Errorf("Body mismatch: %s", sender.sentCalls[0].body)
	}

	// Verify State
	if len(stateManager.successCalls) != 1 {
		t.Errorf("Expected 1 success record, got %d", len(stateManager.successCalls))
	}
	if stateManager.successCalls[0].channelID != testChannelID {
		t.Errorf("Success recorded for wrong channel: %v", stateManager.successCalls[0])
	}
	if stateManager.successCalls[0].emailEventID != "job-id" {
		t.Errorf("Success recorded for wrong event: %v", stateManager.successCalls[0])
	}
	if !stateManager.successCalls[0].timestamp.Equal(fakeNow()) {
		t.Errorf("Success recorded with wrong timestamp: %v", stateManager.successCalls[0])
	}
}

func TestProcessMessage_RenderError(t *testing.T) {
	ctx := context.Background()
	job := workertypes.IncomingEmailDeliveryJob{
		EmailDeliveryJob: workertypes.EmailDeliveryJob{
			SubscriptionID: "sub-1",
			Metadata:       testMetadata(),
			RecipientEmail: "user@example.com",
			SummaryRaw:     []byte("{}"),
			ChannelID:      "chan-1",
			Triggers:       nil,
		},
		EmailEventID: "job-id",
	}

	sender := new(mockEmailSender)
	stateManager := new(mockChannelStateManager)
	renderer := new(mockTemplateRenderer)
	renderer.renderErr = errors.New("template error")

	h := NewSender(sender, stateManager, renderer)
	h.now = fakeNow

	// Should return non transient error (ACK) for rendering error
	err := h.ProcessMessage(ctx, job)
	if errors.Is(err, event.ErrTransientFailure) {
		t.Errorf("Expected non transient error for render failure, got %v", err)
	}

	if !errors.Is(err, renderer.renderErr) {
		t.Errorf("Expected configured renderer error, got %v", err)
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
	job := workertypes.IncomingEmailDeliveryJob{
		EmailDeliveryJob: workertypes.EmailDeliveryJob{
			SubscriptionID: "sub-1",
			Metadata:       testMetadata(),
			RecipientEmail: "user@example.com",
			SummaryRaw:     []byte("{}"),
			ChannelID:      "chan-1",
			Triggers:       nil,
		},
		EmailEventID: "job-id",
	}

	testCases := []struct {
		name                 string
		sendErr              error
		isPermanentUserError bool
		wantNack             bool
	}{
		{
			"regular error = NACK",
			errors.New("send error"),
			false,
			true,
		},
		{
			"user error = ACK",
			workertypes.ErrUnrecoverableUserFailureEmailSending,
			true,
			false,
		},
		{
			"system error = ACK",
			workertypes.ErrUnrecoverableSystemFailureEmailSending,
			false,
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sender := &mockEmailSender{sendErr: tc.sendErr, sentCalls: nil}
			stateManager := new(mockChannelStateManager)
			renderer := new(mockTemplateRenderer)
			renderer.renderSubject = "S"
			renderer.renderBody = "B"

			h := NewSender(sender, stateManager, renderer)
			h.now = fakeNow

			err := h.ProcessMessage(ctx, job)
			if !errors.Is(err, tc.sendErr) {
				t.Errorf("Expected send error %v, got %v", tc.sendErr, err)
			}
			// Should record failure in DB as well
			if len(stateManager.failureCalls) != 1 {
				t.Fatal("Expected failure recording")
			}
			if stateManager.failureCalls[0].channelID != testChannelID {
				t.Errorf("Recorded failure for wrong channel")
			}
			if stateManager.failureCalls[0].emailEventID != "job-id" {
				t.Errorf("Recorded failure for wrong event")
			}
			if tc.isPermanentUserError != stateManager.failureCalls[0].isPermanentUserError {
				t.Errorf("Recorded failure for wrong error type")
			}
			if !stateManager.failureCalls[0].timestamp.Equal(fakeNow()) {
				t.Errorf("Recorded failure for wrong timestamp")
			}

			if tc.wantNack {
				if !errors.Is(err, event.ErrTransientFailure) {
					t.Errorf("Expected transient failure for NACK, got %v", err)
				}
			}
		})
	}

}
