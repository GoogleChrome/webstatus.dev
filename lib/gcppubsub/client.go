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

package gcppubsub

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"github.com/GoogleChrome/webstatus.dev/lib/event"
)

type Client struct {
	client          *pubsub.Client
	receiveSettings receiveSettingsOverrides
}

// receiveSettingsOverrides holds optional subscriber settings. Zero values keep the
// pubsub library defaults.
type receiveSettingsOverrides struct {
	maxExtension           time.Duration
	maxOutstandingMessages int
}

// ClientOption configures optional Client behavior.
type ClientOption func(*Client)

// WithMaxExtension caps how long the subscriber keeps extending a message's lease while its
// handler runs (library default: 60 minutes). The clock starts when the message is received.
// When it expires the handler is not canceled, but Pub/Sub may redeliver the message.
func WithMaxExtension(d time.Duration) ClientOption {
	return func(c *Client) {
		c.receiveSettings.maxExtension = d
	}
}

// WithMaxOutstandingMessages caps how many messages are processed concurrently
// (library default: 1000).
func WithMaxOutstandingMessages(n int) ClientOption {
	return func(c *Client) {
		c.receiveSettings.maxOutstandingMessages = n
	}
}

// NewClient creates a new Pub/Sub client.
// It automatically respects PUBSUB_EMULATOR_HOST env var.
func NewClient(ctx context.Context, projectID string, opts ...ClientOption) (*Client, error) {
	c, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, errors.Join(ErrFailedToEstablishClient, err)
	}

	client := &Client{
		client: c,
		receiveSettings: receiveSettingsOverrides{
			maxExtension:           0,
			maxOutstandingMessages: 0,
		},
	}
	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

func (c *Client) applyReceiveSettings(settings *pubsub.ReceiveSettings) {
	if c.receiveSettings.maxExtension > 0 {
		settings.MaxExtension = c.receiveSettings.maxExtension
	}
	if c.receiveSettings.maxOutstandingMessages > 0 {
		settings.MaxOutstandingMessages = c.receiveSettings.maxOutstandingMessages
	}
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) Publish(ctx context.Context, topicID string, data []byte) (string, error) {
	p := c.client.Publisher(topicID)

	// Publish returns a Result which we must wait on.
	// nolint:exhaustruct // WONTFIX: external struct
	result := p.Publish(ctx, &pubsub.Message{
		Data: data,
	})

	// Block until the result is returned (synchronous publish for safety)
	id, err := result.Get(ctx)
	if err != nil {
		return "", errors.Join(ErrFailedToPublishMessage, err)
	}

	return id, nil
}

func (c *Client) Subscribe(ctx context.Context, subID string,
	handler func(ctx context.Context, msgID string, data []byte) error) error {
	sub := c.client.Subscriber(subID)
	c.applyReceiveSettings(&sub.ReceiveSettings)

	// Receive blocks until ctx is cancelled.
	err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		// Execute the worker's handler
		workErr := handler(ctx, msg.ID, msg.Data)
		if workErr == nil {
			// ACK: Success
			msg.Ack()
		} else if errors.Is(workErr, event.ErrTransientFailure) {
			// NACK: Retry later
			slog.WarnContext(ctx, "transient failure, will retry", "error", workErr)
			msg.Nack()
		} else {
			// ACK: Permanent failure or unknown error, do not retry
			slog.ErrorContext(ctx, "permanent failure", "error", workErr)
			msg.Ack()
		}
	})

	if err != nil {
		return errors.Join(ErrFailedToReceiveMessage, err)
	}

	return nil
}
