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

package httpserver

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

type mockWebhookStorer struct {
	*MockWPTMetricsStorer
	recordResult bool
	recordErr    error
}

func (m *mockWebhookStorer) RecordVCSWebhookDelivery(
	_ context.Context,
	_ gcpspanner.VCSWebhookDelivery,
) (bool, error) {
	return m.recordResult, m.recordErr
}

type mockWebhookPublisher struct {
	MockEventPublisher
	publishedTask *backendtypes.CodeScanTaskMessage
	publishErr    error
}

func (m *mockWebhookPublisher) PublishCodeScanTask(
	_ context.Context,
	task backendtypes.CodeScanTaskMessage,
) error {
	m.publishedTask = &task

	return m.publishErr
}

func signPayload(payload []byte, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)

	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

type webhookTestCase struct {
	name         string
	provider     string
	body         backend.HandleVCSWebhookJSONRequestBody
	sigHeader    *string
	deliveryID   *string
	eventHeader  *string
	recordResult bool
	recordErr    error
	expectedCode int
	expectTask   bool
}

func executeWebhookTest(t *testing.T, tc webhookTestCase, secret []byte) {
	t.Helper()

	storer := &mockWebhookStorer{
		MockWPTMetricsStorer: newDefaultMockWPTMetricsStorer(t),
		recordResult:         tc.recordResult,
		recordErr:            tc.recordErr,
	}
	publisher := &mockWebhookPublisher{
		MockEventPublisher: MockEventPublisher{
			t: t,
			callCountPublishSearchConfigurationChanged: 0,
			publishSearchConfigurationChangedCfg:       nil,
		},
		publishedTask: nil,
		publishErr:    nil,
	}

	server := setupTestServer(t,
		withCustomStorer(storer),
		withCustomEventPublisher(publisher),
		withWebhookSecret(secret),
	)

	req := backend.HandleVCSWebhookRequestObject{
		Provider: tc.provider,
		Params: backend.HandleVCSWebhookParams{
			XHubSignature256: tc.sigHeader,
			XGitHubDelivery:  tc.deliveryID,
			XGitHubEvent:     tc.eventHeader,
		},
		Body: &tc.body,
	}

	resp, err := server.HandleVCSWebhook(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertWebhookResponse(t, resp, tc.expectedCode)

	if tc.expectTask && publisher.publishedTask == nil {
		t.Errorf("expected task to be published, but got nil")
	}
	if !tc.expectTask && publisher.publishedTask != nil {
		t.Errorf("expected no task to be published, but got %+v", publisher.publishedTask)
	}
}

func assertWebhookResponse(t *testing.T, resp backend.HandleVCSWebhookResponseObject, expectedCode int) {
	t.Helper()

	rec := httptest.NewRecorder()
	if err := resp.VisitHandleVCSWebhookResponse(rec); err != nil {
		t.Fatalf("failed to visit webhook response: %v", err)
	}

	if rec.Code != expectedCode {
		t.Errorf("expected status %d, got %d", expectedCode, rec.Code)
	}
}

func TestHandleVCSWebhook(t *testing.T) {
	secret := []byte("a-very-long-secret-key-16-bytes-min")

	pushBody := backend.HandleVCSWebhookJSONRequestBody{
		"ref":    "refs/heads/main",
		"before": "0000000000000000000000000000000000000000",
		"after":  "1111222233334444555566667777888899990000",
		"repository": map[string]any{
			"id":             123456,
			"name":           "webstatus.dev",
			"full_name":      "GoogleChrome/webstatus.dev",
			"default_branch": "main",
			"owner": map[string]any{
				"login": "GoogleChrome",
			},
		},
		"installation": map[string]any{
			"id": 98765,
		},
	}
	bodyBytes, _ := json.Marshal(pushBody)
	validSignature := signPayload(bodyBytes, secret)
	invalidSignature := "sha256=0000000000000000000000000000000000000000000000000000000000000000"

	deliveryGUID := "test-guid-1234"
	eventType := "push"

	testCases := []webhookTestCase{
		{
			name:         "Success Valid Signature Push Event",
			provider:     "github",
			body:         pushBody,
			sigHeader:    &validSignature,
			deliveryID:   &deliveryGUID,
			eventHeader:  &eventType,
			recordResult: true,
			recordErr:    nil,
			expectedCode: http.StatusAccepted,
			expectTask:   true,
		},
		{
			name:         "Missing Signature Header",
			provider:     "github",
			body:         pushBody,
			sigHeader:    nil,
			deliveryID:   &deliveryGUID,
			eventHeader:  &eventType,
			recordResult: true,
			recordErr:    nil,
			expectedCode: http.StatusUnauthorized,
			expectTask:   false,
		},
		{
			name:         "Invalid Signature Header",
			provider:     "github",
			body:         pushBody,
			sigHeader:    &invalidSignature,
			deliveryID:   &deliveryGUID,
			eventHeader:  &eventType,
			recordResult: true,
			recordErr:    nil,
			expectedCode: http.StatusUnauthorized,
			expectTask:   false,
		},
		{
			name:         "Unsupported Provider",
			provider:     "unsupported-vcs",
			body:         pushBody,
			sigHeader:    &validSignature,
			deliveryID:   &deliveryGUID,
			eventHeader:  &eventType,
			recordResult: true,
			recordErr:    nil,
			expectedCode: http.StatusBadRequest,
			expectTask:   false,
		},
		{
			name:         "Duplicate Webhook Delivery (Replay Ignored)",
			provider:     "github",
			body:         pushBody,
			sigHeader:    &validSignature,
			deliveryID:   &deliveryGUID,
			eventHeader:  &eventType,
			recordResult: false,
			recordErr:    nil,
			expectedCode: http.StatusAccepted,
			expectTask:   false,
		},
		{
			name:         "Spanner Recording Error",
			provider:     "github",
			body:         pushBody,
			sigHeader:    &validSignature,
			deliveryID:   &deliveryGUID,
			eventHeader:  &eventType,
			recordResult: false,
			recordErr:    errors.New("spanner failure"),
			expectedCode: http.StatusInternalServerError,
			expectTask:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			executeWebhookTest(t, tc, secret)
		})
	}
}
