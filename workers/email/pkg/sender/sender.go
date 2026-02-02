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

package sender

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type EmailSender interface {
	Send(ctx context.Context, id string, to string, subject string, htmlBody string) error
}

type ChannelStateManager interface {
	RecordSuccess(ctx context.Context, channelID string, timestamp time.Time, eventID string) error
	RecordFailure(ctx context.Context, channelID string, err error,
		timestamp time.Time, permanentUserFailure bool, emailEventID string) error
}

type TemplateRenderer interface {
	RenderDigest(job workertypes.IncomingEmailDeliveryJob) (string, string, error)
}

type Sender struct {
	sender       EmailSender
	stateManager ChannelStateManager
	renderer     TemplateRenderer
	now          func() time.Time
}

func NewSender(
	sender EmailSender,
	stateManager ChannelStateManager,
	renderer TemplateRenderer,
) *Sender {
	return &Sender{
		sender:       sender,
		stateManager: stateManager,
		renderer:     renderer,
		now:          time.Now,
	}
}

func (s *Sender) ProcessMessage(ctx context.Context, job workertypes.IncomingEmailDeliveryJob) error {
	// 1. Render (Parsing happens inside RenderDigest implementation)
	subject, body, err := s.renderer.RenderDigest(job)
	if err != nil {
		slog.ErrorContext(ctx, "failed to render email", "subscription_id", job.SubscriptionID, "error", err)
		if dbErr := s.stateManager.RecordFailure(ctx, job.ChannelID, err, s.now(), false, job.EmailEventID); dbErr != nil {
			slog.ErrorContext(ctx, "failed to record channel failure", "channel_id", job.ChannelID, "error", dbErr)
		}

		return err
	}

	// 2. Send
	if err := s.sender.Send(ctx, job.EmailEventID, job.RecipientEmail, subject, body); err != nil {
		isPermanentUserError := errors.Is(err, workertypes.ErrUnrecoverableUserFailureEmailSending)
		isPermanent := errors.Is(err, workertypes.ErrUnrecoverableSystemFailureEmailSending) ||
			isPermanentUserError
		slog.ErrorContext(ctx, "failed to send email", "recipient", job.RecipientEmail, "error", err)
		// Record failure in DB
		if dbErr := s.stateManager.RecordFailure(ctx, job.ChannelID, err, s.now(),
			isPermanentUserError, job.EmailEventID); dbErr != nil {
			slog.ErrorContext(ctx, "failed to record channel failure", "channel_id", job.ChannelID, "error", dbErr)
		}
		if isPermanent {
			return err
		}

		// If not permanent, wrap with ErrTransient to trigger NACK (which will retry)
		return errors.Join(event.ErrTransientFailure, err)
	}

	// 3. Success
	if err := s.stateManager.RecordSuccess(ctx, job.ChannelID, s.now(), job.EmailEventID); err != nil {
		// Non-critical error, but good to log
		slog.WarnContext(ctx, "failed to record channel success", "channel_id", job.ChannelID, "error", err)
	}

	return nil
}
