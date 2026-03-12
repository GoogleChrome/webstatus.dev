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
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

func TestSender_SendWebhook_Success(t *testing.T) {
	mockHTTP := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			expectedLink := "View Results: https://webstatus.dev/?q=group%3Acss"
			if !strings.Contains(string(body), expectedLink) {
				t.Errorf("expected link %s not found in body", expectedLink)
			}

			return newTestResponse(http.StatusOK, "ok"), nil
		},
	}

	mockState := &mockChannelStateManager{
		successCalls: nil,
		failureCalls: nil,
		recordErr:    nil,
	}
	sender := NewSender(mockHTTP, mockState, "https://webstatus.dev")

	job := newTestIncomingWebhookDeliveryJob(
		"https://hooks.slack.com/services/123", workertypes.WebhookTypeSlack, "group:css", []byte(`{"text":"test"}`))

	err := sender.SendWebhook(context.Background(), job)
	if err != nil {
		t.Fatalf("SendWebhook failed: %v", err)
	}

	verifySuccess(t, mockState)
}

func TestSender_SendWebhook_TransientFailure(t *testing.T) {
	mockHTTP := &mockHTTPClient{
		doFunc: func(_ *http.Request) (*http.Response, error) {
			return nil, event.ErrTransientFailure
		},
	}
	mockState := &mockChannelStateManager{
		successCalls: nil,
		failureCalls: nil,
		recordErr:    nil,
	}
	sender := NewSender(mockHTTP, mockState, "https://webstatus.dev")

	job := newTestIncomingWebhookDeliveryJob(
		"https://hooks.slack.com/services/123", workertypes.WebhookTypeSlack, "group:css", []byte(`{"text":"test"}`))

	err := sender.SendWebhook(context.Background(), job)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, event.ErrTransientFailure) {
		t.Errorf("expected transient failure error, got %v", err)
	}

	if len(mockState.failureCalls) != 1 {
		t.Errorf("expected 1 failure call, got %d", len(mockState.failureCalls))
	}
	if mockState.failureCalls[0].isPermanent {
		t.Error("expected transient failure recorded")
	}
}

func TestSender_SendWebhook_FeatureDeepLink_Success(t *testing.T) {
	mockHTTP := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			expectedLink := "View Results: https://webstatus.dev/features/anchor-positioning"
			if !strings.Contains(string(body), expectedLink) {
				t.Errorf("expected link %s not found in body", expectedLink)
			}

			return newTestResponse(http.StatusOK, "ok"), nil
		},
	}

	mockState := &mockChannelStateManager{
		successCalls: nil,
		failureCalls: nil,
		recordErr:    nil,
	}
	sender := NewSender(mockHTTP, mockState, "https://webstatus.dev")

	job := newTestIncomingWebhookDeliveryJob(
		"https://hooks.slack.com/services/123", workertypes.WebhookTypeSlack,
		"id:\"anchor-positioning\"", []byte(`{"text":"Test Body"}`))

	err := sender.SendWebhook(context.Background(), job)
	if err != nil {
		t.Fatalf("SendWebhook failed: %v", err)
	}

	verifySuccess(t, mockState)
}

func TestSender_SendWebhook_HTTPFailure(t *testing.T) {
	mockHTTP := &mockHTTPClient{
		doFunc: func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}
	mockState := &mockChannelStateManager{
		successCalls: nil,
		failureCalls: nil,
		recordErr:    nil,
	}
	sender := NewSender(mockHTTP, mockState, "https://webstatus.dev")

	job := newTestIncomingWebhookDeliveryJob(
		"https://hooks.slack.com/services/123", workertypes.WebhookTypeSlack, "group:css", []byte(`{"text":"test"}`))

	err := sender.SendWebhook(context.Background(), job)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, event.ErrTransientFailure) {
		t.Errorf("expected transient failure error, got %v", err)
	}

	if len(mockState.failureCalls) != 1 {
		t.Errorf("expected 1 failure call, got %d", len(mockState.failureCalls))
	}
	if mockState.failureCalls[0].isPermanent {
		t.Error("expected transient failure recorded")
	}
}

func TestSender_SendWebhook_PermanentFailure(t *testing.T) {
	mockHTTP := &mockHTTPClient{
		doFunc: func(_ *http.Request) (*http.Response, error) {
			return newTestResponse(http.StatusNotFound, "not found"), nil
		},
	}
	mockState := &mockChannelStateManager{
		successCalls: nil,
		failureCalls: nil,
		recordErr:    nil,
	}
	sender := NewSender(mockHTTP, mockState, "https://webstatus.dev")

	job := newTestIncomingWebhookDeliveryJob(
		"https://hooks.slack.com/services/123", workertypes.WebhookTypeSlack, "group:css", []byte(`{"text":"test"}`))

	err := sender.SendWebhook(context.Background(), job)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if errors.Is(err, event.ErrTransientFailure) {
		t.Error("did not expect transient failure error")
	}

	if len(mockState.failureCalls) != 1 {
		t.Errorf("expected 1 failure call, got %d", len(mockState.failureCalls))
	}
	if !mockState.failureCalls[0].isPermanent {
		t.Error("expected permanent failure recorded")
	}
}

func TestSender_SendWebhook_UnsupportedType(t *testing.T) {
	mockState := &mockChannelStateManager{
		successCalls: nil,
		failureCalls: nil,
		recordErr:    nil,
	}
	sender := NewSender(nil, mockState, "https://webstatus.dev")

	job := newTestIncomingWebhookDeliveryJob(
		"https://example.com/webhook", "unknown", "group:css", nil)

	err := sender.SendWebhook(context.Background(), job)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported type") {
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
		recordErr:    nil,
	}
	sender := NewSender(nil, mockState, "https://webstatus.dev")

	job := newTestIncomingWebhookDeliveryJob(
		"https://not-slack.com/hook", workertypes.WebhookTypeSlack, "group:css", nil)

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

func TestSender_SendWebhook_InvalidSummary(t *testing.T) {
	mockState := &mockChannelStateManager{
		successCalls: nil,
		failureCalls: nil,
		recordErr:    nil,
	}
	sender := NewSender(nil, mockState, "https://webstatus.dev")

	job := newTestIncomingWebhookDeliveryJob(
		"https://hooks.slack.com/services/123", workertypes.WebhookTypeSlack, "group:css", nil)
	job.SummaryRaw = []byte(`invalid json`)

	err := sender.SendWebhook(context.Background(), job)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to unmarshal summary") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func verifySuccess(t *testing.T, mockState *mockChannelStateManager) {
	t.Helper()
	if len(mockState.successCalls) != 1 {
		t.Errorf("expected 1 success call, got %d", len(mockState.successCalls))
	} else if mockState.successCalls[0].channelID != "chan-1" {
		t.Errorf("unexpected channel ID: %s", mockState.successCalls[0].channelID)
	} else if mockState.successCalls[0].eventID != "evt-123" {
		t.Errorf("unexpected event ID: %s", mockState.successCalls[0].eventID)
	}
}
