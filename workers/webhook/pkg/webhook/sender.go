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
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
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

var (
	// ErrTransientWebhook is a transient failure that should be retried.
	ErrTransientWebhook = errors.New("transient webhook failure")
	// ErrPermanentWebhook is a permanent failure that should not be retried.
	ErrPermanentWebhook = errors.New("permanent webhook failure")
)

type webhookSender interface {
	Send(ctx context.Context) error
}

// Manager wraps the type-specific webhook logic.
type Manager struct {
	sender webhookSender
}

func (s *Sender) getManager(_ context.Context, job workertypes.IncomingWebhookDeliveryJob) (*Manager, error) {
	switch job.WebhookType {
	case workertypes.WebhookTypeSlack:
		slack, err := newSlackSender(s.frontendBaseURL, s.httpClient, job)
		if err != nil {
			return nil, err
		}

		return &Manager{sender: slack}, nil
	default:
		return nil, fmt.Errorf("%w: unsupported type %v", ErrPermanentWebhook, job.WebhookType)
	}
}

func (s *Sender) SendWebhook(ctx context.Context, job workertypes.IncomingWebhookDeliveryJob) error {
	slog.InfoContext(ctx, "sending webhook", "channelID", job.ChannelID)

	mgr, err := s.getManager(ctx, job)
	if err != nil {
		// If we fail here, it's permanent when trying to get the manager.
		s.recordFailure(ctx, job, err, true)

		return fmt.Errorf("failed to prepare webhook: %w", err)
	}

	if err := mgr.sender.Send(ctx); err != nil {
		isTransient := errors.Is(err, ErrTransientWebhook)
		s.recordFailure(ctx, job, err, !isTransient)

		if isTransient {
			return errors.Join(event.ErrTransientFailure, err)
		}

		return fmt.Errorf("failed to send webhook: %w", err)
	}

	if err := s.stateManager.RecordSuccess(ctx, job.ChannelID, time.Now(), job.WebhookEventID); err != nil {
		slog.WarnContext(ctx, "failed to record success", "error", err)
	}

	return nil
}

func (s *Sender) recordFailure(ctx context.Context, job workertypes.IncomingWebhookDeliveryJob,
	err error, permanent bool) {
	if dbErr := s.stateManager.RecordFailure(ctx, job.ChannelID, err, time.Now(),
		permanent, job.WebhookEventID); dbErr != nil {
		slog.ErrorContext(ctx, "failed to record failure", "error", dbErr)
	}
}
