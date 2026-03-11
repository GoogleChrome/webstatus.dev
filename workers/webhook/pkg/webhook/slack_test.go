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
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/google/go-cmp/cmp"
)

func TestSlackSender_Send(t *testing.T) {
	tests := []slackTestCase{
		{
			name: "successful send with correct query-based payload",
			job: newTestIncomingWebhookDeliveryJob(
				"https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
				workertypes.WebhookTypeSlack,
				"group:css",
				[]byte(`{"text":"New feature landed"}`),
			),
			mockResponse: newTestResponse(http.StatusOK, "ok"),
			mockErr:      nil,
			expectedPayload: &SlackPayload{
				Text: "WebStatus.dev Notification: New feature landed\n" +
					"Query: group:css\n" +
					"View Results: https://webstatus.dev/features?q=group%3Acss",
			},
			expectedErr: nil,
		},
		{
			name: "successful send with direct feature link",
			job: newTestIncomingWebhookDeliveryJob(
				"https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
				workertypes.WebhookTypeSlack,
				"id:\"anchor-positioning\"",
				[]byte(`{"text":"Test Body"}`),
			),
			mockResponse: newTestResponse(http.StatusOK, "ok"),
			mockErr:      nil,
			expectedPayload: &SlackPayload{
				Text: "WebStatus.dev Notification: Test Body\n" +
					"Query: id:\"anchor-positioning\"\n" +
					"View Results: https://webstatus.dev/features?q=id%3A%22anchor-positioning%22",
			},
			expectedErr: nil,
		},
		{
			name: "network error",
			job: newTestIncomingWebhookDeliveryJob(
				"https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
				workertypes.WebhookTypeSlack,
				"",
				[]byte(`{"text":"retry"}`),
			),
			mockResponse:    nil,
			mockErr:         errors.New("network failure"),
			expectedPayload: nil,
			expectedErr:     ErrTransientWebhook,
		},
		{
			name: "permanent error (404)",
			job: newTestIncomingWebhookDeliveryJob(
				"https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
				workertypes.WebhookTypeSlack,
				"",
				[]byte(`{"text":"fail"}`),
			),
			mockResponse: newTestResponse(http.StatusNotFound, "not found"),
			mockErr:      nil,
			expectedPayload: &SlackPayload{
				Text: "WebStatus.dev Notification: fail\n" +
					"Query: \n" +
					"View Results: https://webstatus.dev/features?q=",
			},
			expectedErr: ErrPermanentWebhook,
		},
		{
			name: "transient error (500)",
			job: newTestIncomingWebhookDeliveryJob(
				"https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
				workertypes.WebhookTypeSlack,
				"",
				[]byte(`{"text":"retry"}`),
			),
			mockResponse: newTestResponse(http.StatusInternalServerError, "internal error"),
			mockErr:      nil,
			expectedPayload: &SlackPayload{
				Text: "WebStatus.dev Notification: retry\n" +
					"Query: \n" +
					"View Results: https://webstatus.dev/features?q=",
			},
			expectedErr: ErrTransientWebhook,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturedBody := tt.runTest(t)
			tt.verifyPayload(t, capturedBody)
		})
	}
}

func (tc *slackTestCase) runTest(t *testing.T) []byte {
	var capturedBody []byte
	mockHTTP := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			if tc.mockErr != nil {
				return nil, tc.mockErr
			}
			var err error
			capturedBody, err = io.ReadAll(req.Body)

			return tc.mockResponse, err
		},
	}

	sender, err := newSlackSender("https://webstatus.dev", mockHTTP, tc.job)
	if err != nil {
		if tc.expectedErr != nil && errors.Is(err, tc.expectedErr) {
			return nil
		}
		t.Fatalf("unexpected error creating sender: %v", err)
	}

	err = sender.Send(context.Background())
	if tc.expectedErr != nil {
		if !errors.Is(err, tc.expectedErr) {
			t.Errorf("Send() error = %v, expectedErr %v", err, tc.expectedErr)
		}
	} else if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	return capturedBody
}

func (tc *slackTestCase) verifyPayload(t *testing.T, capturedBody []byte) {
	var actualPayload *SlackPayload
	if len(capturedBody) > 0 {
		actualPayload = new(SlackPayload)
		if err := json.Unmarshal(capturedBody, actualPayload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
	}

	if diff := cmp.Diff(tc.expectedPayload, actualPayload); diff != "" {
		t.Errorf("payload mismatch (-want +got):\n%s", diff)
	}
}

type slackTestCase struct {
	name            string
	job             workertypes.IncomingWebhookDeliveryJob
	mockResponse    *http.Response
	mockErr         error
	expectedPayload *SlackPayload
	expectedErr     error
}
