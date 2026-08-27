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
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/webhookverifiertypes"
)

type mockWebhookVerifier struct {
	validSignature string
}

func (m *mockWebhookVerifier) VerifySignature(_ []byte, signature string) error {
	if m.validSignature == "" {
		return webhookverifiertypes.ErrSecretNotConfigured
	}
	if signature == "" {
		return webhookverifiertypes.ErrMissingSignature
	}
	if signature != m.validSignature {
		return webhookverifiertypes.ErrInvalidSignature
	}

	return nil
}

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
	publishedTask *codescantaskv1.CodeScanTaskEvent
	publishErr    error
}

func (m *mockWebhookPublisher) PublishCodeScanTask(
	_ context.Context,
	task codescantaskv1.CodeScanTaskEvent,
) error {
	m.publishedTask = &task

	return m.publishErr
}

type webhookTestCase struct {
	name             string
	provider         string
	body             string
	sigHeader        *string
	deliveryID       *string
	eventHeader      *string
	recordResult     bool
	recordErr        error
	publishErr       error
	expectedResponse *http.Response
	expectTask       bool
}

//nolint:unparam,exhaustruct // Test helper parameterized for status codes
func testEmptyResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

func executeWebhookTest(t *testing.T, tc webhookTestCase, verifier WebhookVerifier) {
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
		publishErr:    tc.publishErr,
	}

	server := setupTestServer(t,
		withCustomStorer(storer),
		withCustomEventPublisher(publisher),
		withWebhookVerifier(verifier),
	)

	req := httptest.NewRequestWithContext(t.Context(),
		http.MethodPost, "/v1/webhooks/"+tc.provider, strings.NewReader(tc.body))
	req.Header.Set("Content-Type", "application/json")
	if tc.sigHeader != nil {
		req.Header.Set("X-Hub-Signature-256", *tc.sigHeader)
	}
	if tc.deliveryID != nil {
		req.Header.Set("X-GitHub-Delivery", *tc.deliveryID)
	}
	if tc.eventHeader != nil {
		req.Header.Set("X-GitHub-Event", *tc.eventHeader)
	}

	assertTestServerRequest(t, server, req, tc.expectedResponse)

	if tc.expectTask && publisher.publishedTask == nil {
		t.Errorf("expected task to be published, but got nil")
	}
	if !tc.expectTask && publisher.publishedTask != nil {
		t.Errorf("expected no task to be published, but got %+v", publisher.publishedTask)
	}
}

func TestHandleVCSWebhook(t *testing.T) {
	validSignature := "sha256=valid-signature-1234"
	invalidSignature := "sha256=invalid-signature-0000"
	verifier := &mockWebhookVerifier{validSignature: validSignature}

	pushBody := `{
		"ref": "refs/heads/main",
		"before": "0000000000000000000000000000000000000000",
		"after": "1111222233334444555566667777888899990000",
		"repository": {
			"id": 123456,
			"name": "webstatus.dev",
			"full_name": "GoogleChrome/webstatus.dev",
			"default_branch": "main",
			"owner": {
				"login": "GoogleChrome"
			}
		},
		"installation": {
			"id": 98765
		}
	}`

	deliveryGUID := "test-guid-1234"
	eventType := "push"

	// nolint:exhaustruct // Table-driven test cases
	testCases := []webhookTestCase{
		{
			name:             "Success Valid Signature Push Event",
			provider:         "github",
			body:             pushBody,
			sigHeader:        &validSignature,
			deliveryID:       &deliveryGUID,
			eventHeader:      &eventType,
			recordResult:     true,
			recordErr:        nil,
			expectedResponse: testEmptyResponse(http.StatusAccepted),
			expectTask:       true,
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
			expectedResponse: testJSONResponse(http.StatusUnauthorized, `{
				"code": 401,
				"message": "invalid webhook signature"
			}`),
			expectTask: false,
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
			expectedResponse: testJSONResponse(http.StatusUnauthorized, `{
				"code": 401,
				"message": "invalid webhook signature"
			}`),
			expectTask: false,
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
			expectedResponse: testJSONResponse(http.StatusBadRequest, `{
				"code": 400,
				"message": "unsupported VCS provider: unsupported-vcs"
			}`),
			expectTask: false,
		},
		{
			name:             "Duplicate Webhook Delivery (Replay Ignored)",
			provider:         "github",
			body:             pushBody,
			sigHeader:        &validSignature,
			deliveryID:       &deliveryGUID,
			eventHeader:      &eventType,
			recordResult:     false,
			recordErr:        nil,
			expectedResponse: testEmptyResponse(http.StatusAccepted),
			expectTask:       false,
		},
		{
			name:             "Non-Push Ping Event (Accepted Without Scan Task)",
			provider:         "github",
			body:             `{"zen": "Favor focus over features."}`,
			sigHeader:        &validSignature,
			deliveryID:       &deliveryGUID,
			eventHeader:      func(s string) *string { return &s }("ping"),
			recordResult:     true,
			recordErr:        nil,
			expectedResponse: testEmptyResponse(http.StatusAccepted),
			expectTask:       false,
		},
		{
			name:             "Non-Push Pull Request Event (Accepted Without Scan Task)",
			provider:         "github",
			body:             pushBody,
			sigHeader:        &validSignature,
			deliveryID:       &deliveryGUID,
			eventHeader:      func(s string) *string { return &s }("pull_request"),
			recordResult:     true,
			recordErr:        nil,
			expectedResponse: testEmptyResponse(http.StatusAccepted),
			expectTask:       false,
		},
		{
			name:         "Missing Delivery ID Header",
			provider:     "github",
			body:         pushBody,
			sigHeader:    &validSignature,
			deliveryID:   nil,
			eventHeader:  &eventType,
			recordResult: true,
			recordErr:    nil,
			expectedResponse: testJSONResponse(http.StatusBadRequest, `{
				"code": 400,
				"message": "missing X-GitHub-Delivery header"
			}`),
			expectTask: false,
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
			publishErr:   nil,
			expectedResponse: testJSONResponse(http.StatusInternalServerError, `{
				"code": 500,
				"message": "failed to record webhook delivery"
			}`),
			expectTask: false,
		},
		{
			name:         "PubSub Publish Error (500)",
			provider:     "github",
			body:         pushBody,
			sigHeader:    &validSignature,
			deliveryID:   &deliveryGUID,
			eventHeader:  &eventType,
			recordResult: true,
			recordErr:    nil,
			publishErr:   errors.New("pubsub publish error"),
			expectedResponse: testJSONResponse(http.StatusInternalServerError, `{
				"code": 500,
				"message": "failed to publish scan task"
			}`),
			expectTask: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			executeWebhookTest(t, tc, verifier)
		})
	}
}
