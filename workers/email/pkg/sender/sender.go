// Copyright 2025 Google LLC
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
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type EmailSender interface {
	Send(ctx context.Context, to string, subject string, htmlBody string) error
}

type ChannelStateManager interface {
	RecordSuccess(ctx context.Context, channelID string) error
	RecordFailure(ctx context.Context, channelID string, err error) error
}

type TemplateRenderer interface {
	RenderDigest(job workertypes.EmailDeliveryJob) (string, string, error)
}

type Sender struct {
	sender       EmailSender
	stateManager ChannelStateManager
	renderer     TemplateRenderer
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
	}
}

func (s *Sender) ProcessMessage(ctx context.Context, job workertypes.EmailDeliveryJob) error {
	// 1. Render (Parsing happens inside RenderDigest implementation)
	subject, body, err := s.renderer.RenderDigest(job)
	if err != nil {
		slog.ErrorContext(ctx, "failed to render email", "subscription_id", job.SubscriptionID, "error", err)
		if err := s.stateManager.RecordFailure(ctx, job.ChannelID, err); err != nil {
			slog.ErrorContext(ctx, "failed to record channel failure", "channel_id", job.ChannelID, "error", err)
		}
		// Rendering errors might be transient or permanent. Assuming permanent for template bugs.
		return nil
	}

	// 2. Send
	if err := s.sender.Send(ctx, job.RecipientEmail, subject, body); err != nil {
		slog.ErrorContext(ctx, "failed to send email", "recipient", job.RecipientEmail, "error", err)
		// Record failure in DB
		if dbErr := s.stateManager.RecordFailure(ctx, job.ChannelID, err); dbErr != nil {
			slog.ErrorContext(ctx, "failed to record channel failure", "channel_id", job.ChannelID, "error", dbErr)
		}

		// Return error to NACK the message and retry sending?
		// Sending failures (network, rate limit) are often transient.
		return err
	}

	// 3. Success
	if err := s.stateManager.RecordSuccess(ctx, job.ChannelID); err != nil {
		// Non-critical error, but good to log
		slog.WarnContext(ctx, "failed to record channel success", "channel_id", job.ChannelID, "error", err)
	}

	return nil
}
