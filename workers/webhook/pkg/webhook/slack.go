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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/httputils"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type SlackPayload struct {
	Text string `json:"text"`
}

type slackManager struct {
	frontendBaseURL string
	httpClient      HTTPClient
	stateManager    ChannelStateManager
	job             workertypes.IncomingWebhookDeliveryJob
}

func (s *slackManager) Send(ctx context.Context) error {
	var summary workertypes.EventSummary
	if err := json.Unmarshal(s.job.SummaryRaw, &summary); err != nil {
		return fmt.Errorf("failed to unmarshal summary: %w", err)
	}

	resultsURL := fmt.Sprintf("%s/features?q=%s", s.frontendBaseURL, url.QueryEscape(s.job.Metadata.Query))

	payload := SlackPayload{
		Text: fmt.Sprintf("WebStatus.dev Notification: %s\nQuery: %s\nView Results: %s",
			summary.Text, s.job.Metadata.Query, resultsURL),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.job.WebhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		_ = s.stateManager.RecordFailure(ctx, s.job.ChannelID, err, time.Now(), false, s.job.WebhookEventID)

		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_ = s.stateManager.RecordSuccess(ctx, s.job.ChannelID, time.Now(), s.job.WebhookEventID)

		return nil
	}

	errorMsg := fmt.Sprintf("webhook returned status code %d", resp.StatusCode)
	webhookErr := errors.New(errorMsg)
	isPermanent := resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone ||
		resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden

	_ = s.stateManager.RecordFailure(ctx, s.job.ChannelID, webhookErr, time.Now(), isPermanent, s.job.WebhookEventID)

	return fmt.Errorf("webhook failed: %s", errorMsg)
}

type slackPreparer struct {
	frontendBaseURL string
	httpClient      HTTPClient
	stateManager    ChannelStateManager
}

//nolint:ireturn // Strategy pattern requires returning the interface
func (s *slackPreparer) Prepare(job workertypes.IncomingWebhookDeliveryJob) (Manager, error) {
	if err := httputils.ValidateSlackWebhookURL(job.WebhookURL); err != nil {
		// Preparation failures (like invalid payload or URL format) are typically permanent
		_ = s.stateManager.RecordFailure(context.Background(), job.ChannelID, err, time.Now(), true, job.WebhookEventID)

		return nil, fmt.Errorf("invalid webhook URL: %w", err)
	}

	return &slackManager{
		frontendBaseURL: s.frontendBaseURL,
		httpClient:      s.httpClient,
		stateManager:    s.stateManager,
		job:             job,
	}, nil
}
