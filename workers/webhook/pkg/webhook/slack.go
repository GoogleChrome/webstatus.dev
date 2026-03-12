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

	"github.com/GoogleChrome/webstatus.dev/lib/httputils"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type SlackPayload struct {
	Text string `json:"text"`
}

type slackSender struct {
	frontendBaseURL string
	httpClient      HTTPClient
	job             workertypes.IncomingWebhookDeliveryJob
}

func newSlackSender(frontendBaseURL string, httpClient HTTPClient,
	job workertypes.IncomingWebhookDeliveryJob) (*slackSender, error) {
	if err := httputils.ValidateSlackWebhookURL(job.WebhookURL); err != nil {
		return nil, fmt.Errorf("%w: invalid webhook URL: %w", ErrPermanentWebhook, err)
	}

	return &slackSender{
		frontendBaseURL: frontendBaseURL,
		httpClient:      httpClient,
		job:             job,
	}, nil
}

func (s *slackSender) Send(ctx context.Context) error {
	var summary workertypes.EventSummary
	if err := json.Unmarshal(s.job.SummaryRaw, &summary); err != nil {
		return fmt.Errorf("%w: failed to unmarshal summary: %w", ErrPermanentWebhook, err)
	}

	query := s.job.Metadata.Query
	// Default search results page
	resultsURL := fmt.Sprintf("%s/features?q=%s", s.frontendBaseURL, url.QueryEscape(query))

	payload := SlackPayload{
		Text: fmt.Sprintf("WebStatus.dev Notification: %s\nQuery: %s\nView Results: %s",
			summary.Text, query, resultsURL),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal slack payload: %w", ErrPermanentWebhook, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.job.WebhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("%w: failed to create request: %w", ErrPermanentWebhook, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Join(ErrTransientWebhook, fmt.Errorf("network error: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	webhookErr := fmt.Errorf("webhook returned status code %d", resp.StatusCode)
	isPermanent := resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone ||
		resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden

	if !isPermanent {
		return errors.Join(ErrTransientWebhook, webhookErr)
	}

	return errors.Join(ErrPermanentWebhook, webhookErr)
}
