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

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

type mockChannelStateManager struct {
	successCalls []successCall
	failureCalls []failureCall
}

type successCall struct {
	channelID string
	timestamp time.Time
	eventID   string
}

type failureCall struct {
	channelID   string
	err         error
	timestamp   time.Time
	isPermanent bool
	eventID     string
}

func (m *mockChannelStateManager) RecordSuccess(_ context.Context, channelID string,
	timestamp time.Time, eventID string) error {
	m.successCalls = append(m.successCalls, successCall{channelID, timestamp, eventID})

	return nil
}

func (m *mockChannelStateManager) RecordFailure(_ context.Context, channelID string,
	err error, timestamp time.Time, isPermanent bool, eventID string) error {
	m.failureCalls = append(m.failureCalls, failureCall{channelID, err, timestamp, isPermanent, eventID})

	return nil
}

func TestSender_SendWebhook_Success(t *testing.T) {
	mockHTTP := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://hooks.slack.com/services/123" {
				t.Errorf("unexpected URL: %s", req.URL.String())
			}
			if req.Method != http.MethodPost {
				t.Errorf("unexpected method: %s", req.Method)
			}
			body, _ := io.ReadAll(req.Body)
			var payload SlackPayload
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Errorf("failed to unmarshal payload: %v", err)
			}
			if !strings.Contains(payload.Text, "Test Body") {
				t.Errorf("payload does not contain expected text: %s", payload.Text)
			}
			expectedLink := "View Results: https://webstatus.dev/features?q=group%3Acss"
			if !strings.Contains(payload.Text, expectedLink) {
				t.Errorf("payload missing expected link. Got: %s", payload.Text)
			}

			return &http.Response{
				StatusCode:       http.StatusOK,
				Body:             io.NopCloser(strings.NewReader("ok")),
				Status:           "200 OK",
				Proto:            "HTTP/1.1",
				ProtoMajor:       1,
				ProtoMinor:       1,
				Header:           make(http.Header),
				ContentLength:    2,
				TransferEncoding: nil,
				Close:            false,
				Uncompressed:     false,
				Trailer:          nil,
				Request:          nil,
				TLS:              nil,
			}, nil
		},
	}

	mockState := &mockChannelStateManager{
		successCalls: nil,
		failureCalls: nil,
	}
	frontendBaseURL := "https://webstatus.dev"
	sender := NewSender(mockHTTP, mockState, frontendBaseURL)

	summary := workertypes.EventSummary{
		Text:          "Test Body",
		SchemaVersion: "v1",
		Categories: workertypes.SummaryCategories{
			QueryChanged:    0,
			Added:           0,
			Removed:         0,
			Deleted:         0,
			Moved:           0,
			Split:           0,
			Updated:         0,
			UpdatedImpl:     0,
			UpdatedRename:   0,
			UpdatedBaseline: 0,
		},
		Truncated:  false,
		Highlights: nil,
	}
	summaryRaw, _ := json.Marshal(summary)

	job := workertypes.IncomingWebhookDeliveryJob{
		WebhookDeliveryJob: workertypes.WebhookDeliveryJob{
			ChannelID:      "chan-1",
			WebhookURL:     "https://hooks.slack.com/services/123",
			WebhookType:    workertypes.WebhookTypeSlack,
			SummaryRaw:     summaryRaw,
			SubscriptionID: "sub-1",
			Triggers:       nil,
			Metadata: workertypes.DeliveryMetadata{
				EventID:     "evt-1",
				SearchID:    "search-1",
				SearchName:  "",
				Query:       "group:css",
				Frequency:   workertypes.FrequencyImmediate,
				GeneratedAt: time.Time{},
			},
		},
		WebhookEventID: "evt-1",
	}

	err := sender.SendWebhook(context.Background(), job)
	if err != nil {
		t.Fatalf("SendWebhook failed: %v", err)
	}

	if len(mockState.successCalls) != 1 {
		t.Errorf("expected 1 success call, got %d", len(mockState.successCalls))
	}
	if mockState.successCalls[0].channelID != "chan-1" {
		t.Errorf("unexpected channel ID: %s", mockState.successCalls[0].channelID)
	}
}

func TestSender_SendWebhook_HTTPFailure(t *testing.T) {
	mockHTTP := &mockHTTPClient{
		doFunc: func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:       http.StatusNotFound,
				Body:             io.NopCloser(strings.NewReader("not found")),
				Status:           "404 Not Found",
				Proto:            "HTTP/1.1",
				ProtoMajor:       1,
				ProtoMinor:       1,
				Header:           make(http.Header),
				ContentLength:    9,
				TransferEncoding: nil,
				Close:            false,
				Uncompressed:     false,
				Trailer:          nil,
				Request:          nil,
				TLS:              nil,
			}, nil
		},
	}

	mockState := &mockChannelStateManager{
		successCalls: nil,
		failureCalls: nil,
	}
	sender := NewSender(mockHTTP, mockState, "https://webstatus.dev")

	summary := workertypes.EventSummary{
		Text:          "Test Body",
		SchemaVersion: "v1",
		Categories: workertypes.SummaryCategories{
			QueryChanged:    0,
			Added:           0,
			Removed:         0,
			Deleted:         0,
			Moved:           0,
			Split:           0,
			Updated:         0,
			UpdatedImpl:     0,
			UpdatedRename:   0,
			UpdatedBaseline: 0,
		},
		Truncated:  false,
		Highlights: nil,
	}
	summaryRaw, _ := json.Marshal(summary)

	job := workertypes.IncomingWebhookDeliveryJob{
		WebhookDeliveryJob: workertypes.WebhookDeliveryJob{
			ChannelID:      "chan-1",
			WebhookURL:     "https://hooks.slack.com/services/123",
			WebhookType:    workertypes.WebhookTypeSlack,
			SummaryRaw:     summaryRaw,
			SubscriptionID: "sub-1",
			Triggers:       nil,
			Metadata: workertypes.DeliveryMetadata{
				EventID:     "evt-1",
				SearchID:    "search-1",
				SearchName:  "",
				Query:       "group:css",
				Frequency:   workertypes.FrequencyImmediate,
				GeneratedAt: time.Time{},
			},
		},
		WebhookEventID: "evt-1",
	}

	err := sender.SendWebhook(context.Background(), job)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if len(mockState.failureCalls) != 1 {
		t.Errorf("expected 1 failure call, got %d", len(mockState.failureCalls))
	}
	if !mockState.failureCalls[0].isPermanent {
		t.Error("expected permanent failure for 404")
	}
}

func TestSender_SendWebhook_UnsupportedType(t *testing.T) {
	mockState := &mockChannelStateManager{
		successCalls: nil,
		failureCalls: nil,
	}
	sender := NewSender(nil, mockState, "https://webstatus.dev")

	job := workertypes.IncomingWebhookDeliveryJob{
		WebhookDeliveryJob: workertypes.WebhookDeliveryJob{
			ChannelID:      "chan-1",
			WebhookURL:     "https://example.com/webhook",
			WebhookType:    "unknown",
			SummaryRaw:     nil,
			SubscriptionID: "sub-1",
			Triggers:       nil,
			Metadata: workertypes.DeliveryMetadata{
				EventID:     "evt-1",
				SearchID:    "search-1",
				SearchName:  "",
				Query:       "group:css",
				Frequency:   workertypes.FrequencyImmediate,
				GeneratedAt: time.Time{},
			},
		},
		WebhookEventID: "evt-1",
	}

	err := sender.SendWebhook(context.Background(), job)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported webhook type") {
		t.Errorf("unexpected error message: %v", err)
	}

	if len(mockState.failureCalls) != 1 {
		t.Errorf("expected 1 failure call, got %d", len(mockState.failureCalls))
	}
	if !mockState.failureCalls[0].isPermanent {
		t.Error("expected permanent failure for unsupported type")
	}
}

func TestSender_SendWebhook_InvalidSlackURL(t *testing.T) {
	mockState := &mockChannelStateManager{
		successCalls: nil,
		failureCalls: nil,
	}
	sender := NewSender(nil, mockState, "https://webstatus.dev")

	job := workertypes.IncomingWebhookDeliveryJob{
		WebhookDeliveryJob: workertypes.WebhookDeliveryJob{
			ChannelID:      "chan-1",
			WebhookURL:     "https://not-slack.com/hook",
			WebhookType:    workertypes.WebhookTypeSlack,
			SummaryRaw:     nil,
			SubscriptionID: "sub-1",
			Triggers:       nil,
			Metadata: workertypes.DeliveryMetadata{
				EventID:     "evt-1",
				SearchID:    "search-1",
				SearchName:  "",
				Query:       "group:css",
				Frequency:   workertypes.FrequencyImmediate,
				GeneratedAt: time.Time{},
			},
		},
		WebhookEventID: "evt-1",
	}

	err := sender.SendWebhook(context.Background(), job)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if len(mockState.failureCalls) != 1 {
		t.Errorf("expected 1 failure call, got %d", len(mockState.failureCalls))
	}
	if !mockState.failureCalls[0].isPermanent {
		t.Error("expected permanent failure for invalid URL")
	}
}

func TestSender_SendWebhook_NetworkError(t *testing.T) {
	mockHTTP := &mockHTTPClient{
		doFunc: func(_ *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		},
	}
	mockState := &mockChannelStateManager{
		successCalls: nil,
		failureCalls: nil,
	}
	sender := NewSender(mockHTTP, mockState, "https://webstatus.dev")

	job := workertypes.IncomingWebhookDeliveryJob{
		WebhookDeliveryJob: workertypes.WebhookDeliveryJob{
			ChannelID:      "chan-1",
			WebhookURL:     "https://hooks.slack.com/services/123",
			WebhookType:    workertypes.WebhookTypeSlack,
			SummaryRaw:     []byte(`{"text":"test"}`),
			SubscriptionID: "sub-1",
			Triggers:       nil,
			Metadata: workertypes.DeliveryMetadata{
				EventID:     "evt-1",
				SearchID:    "search-1",
				SearchName:  "",
				Query:       "group:css",
				Frequency:   workertypes.FrequencyImmediate,
				GeneratedAt: time.Time{},
			},
		},
		WebhookEventID: "evt-1",
	}

	err := sender.SendWebhook(context.Background(), job)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if len(mockState.failureCalls) != 1 {
		t.Errorf("expected 1 failure call, got %d", len(mockState.failureCalls))
	}
	if mockState.failureCalls[0].isPermanent {
		t.Error("expected transient failure for network error")
	}
}

func TestSender_SendWebhook_InvalidSummary(t *testing.T) {
	mockState := &mockChannelStateManager{
		successCalls: nil,
		failureCalls: nil,
	}
	sender := NewSender(nil, mockState, "https://webstatus.dev")

	job := workertypes.IncomingWebhookDeliveryJob{
		WebhookDeliveryJob: workertypes.WebhookDeliveryJob{
			ChannelID:      "chan-1",
			WebhookURL:     "https://hooks.slack.com/services/123",
			WebhookType:    workertypes.WebhookTypeSlack,
			SummaryRaw:     []byte(`invalid json`),
			SubscriptionID: "sub-1",
			Triggers:       nil,
			Metadata: workertypes.DeliveryMetadata{
				EventID:     "evt-1",
				SearchID:    "search-1",
				SearchName:  "",
				Query:       "group:css",
				Frequency:   workertypes.FrequencyImmediate,
				GeneratedAt: time.Time{},
			},
		},
		WebhookEventID: "evt-1",
	}

	err := sender.SendWebhook(context.Background(), job)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to unmarshal summary") {
		t.Errorf("unexpected error message: %v", err)
	}
}
