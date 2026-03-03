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
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type ChannelStateManager interface {
	RecordSuccess(ctx context.Context, channelID string, timestamp time.Time, eventID string) error
	RecordFailure(ctx context.Context, channelID string, err error, timestamp time.Time,
		isPermanent bool, eventID string) error
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Sender struct {
	httpClient      HTTPClient
	stateManager    ChannelStateManager
	frontendBaseURL string
}

func NewSender(httpClient HTTPClient, stateManager ChannelStateManager, frontendBaseURL string) *Sender {
	return &Sender{
		httpClient:      httpClient,
		stateManager:    stateManager,
		frontendBaseURL: frontendBaseURL,
	}
}

type SlackPayload struct {
	Text string `json:"text"`
}

type webhookPreparer interface {
	Prepare(ctx context.Context, job workertypes.IncomingWebhookDeliveryJob) (*http.Request, error)
}

type slackPreparer struct {
	frontendBaseURL string
}

func (s *slackPreparer) Prepare(
	ctx context.Context, job workertypes.IncomingWebhookDeliveryJob) (*http.Request, error) {
	parsedURL, err := url.Parse(job.WebhookURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Host != "hooks.slack.com" {
		// Record permanent failure due to invalid URL
		return nil, fmt.Errorf("invalid webhook URL: %s", job.WebhookURL)
	}

	var summary workertypes.EventSummary
	if err := json.Unmarshal(job.SummaryRaw, &summary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal summary: %w", err)
	}

	resultsURL := fmt.Sprintf("%s/features?q=%s", s.frontendBaseURL, url.QueryEscape(job.Metadata.Query))

	payload := SlackPayload{
		Text: fmt.Sprintf("WebStatus.dev Notification: %s\nQuery: %s\nView Results: %s",
			summary.Text, job.Metadata.Query, resultsURL),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, job.WebhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (s *Sender) SendWebhook(ctx context.Context, job workertypes.IncomingWebhookDeliveryJob) error {
	slog.InfoContext(ctx, "sending webhook", "channelID", job.ChannelID, "url", job.WebhookURL)

	var preparer webhookPreparer
	switch job.WebhookType {
	case workertypes.WebhookTypeSlack:
		preparer = &slackPreparer{frontendBaseURL: s.frontendBaseURL}
	default:
		err := fmt.Errorf("unsupported webhook type: %v", job.WebhookType)
		_ = s.stateManager.RecordFailure(ctx, job.ChannelID, err, time.Now(), true, job.WebhookEventID)

		return err
	}

	req, err := preparer.Prepare(ctx, job)
	if err != nil {
		// Preparation failures (like invalid payload or URL format) are typically permanent
		_ = s.stateManager.RecordFailure(ctx, job.ChannelID, err, time.Now(), true, job.WebhookEventID)

		return fmt.Errorf("failed to prepare webhook request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		// Transient error?
		_ = s.stateManager.RecordFailure(ctx, job.ChannelID, err, time.Now(), false, job.WebhookEventID)

		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Success
		_ = s.stateManager.RecordSuccess(ctx, job.ChannelID, time.Now(), job.WebhookEventID)

		return nil
	}

	// Failure
	errorMsg := fmt.Sprintf("webhook returned status code %d", resp.StatusCode)
	webhookErr := errors.New(errorMsg)
	isPermanent := resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone ||
		resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden

	_ = s.stateManager.RecordFailure(ctx, job.ChannelID, webhookErr, time.Now(), isPermanent, job.WebhookEventID)

	return fmt.Errorf("webhook failed: %s", errorMsg)
}
