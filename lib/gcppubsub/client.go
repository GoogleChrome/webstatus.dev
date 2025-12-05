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

	"cloud.google.com/go/pubsub/v2"
	"github.com/GoogleChrome/webstatus.dev/lib/event"
)

type Client struct {
	client *pubsub.Client
}

// NewClient creates a new Pub/Sub client.
// It automatically respects PUBSUB_EMULATOR_HOST env var.
func NewClient(ctx context.Context, projectID string) (*Client, error) {
	c, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, errors.Join(ErrFailedToEstablishClient, err)
	}

	return &Client{client: c}, nil
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
	handler func(ctx context.Context, data []byte) error) error {
	sub := c.client.Subscriber(subID)

	// Receive blocks until ctx is cancelled.
	err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		// Execute the worker's handler
		workErr := handler(ctx, msg.Data)
		if workErr == nil {
			// ACK: Success
			msg.Ack()
		} else if errors.Is(workErr, event.ErrTransientFailure) {
			// NACK: Retry later
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
