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

// NOTE: This file is based on lib/gcppubsub/gcppubsubadapters/email_worker.go.
// TODO: Consider consolidating shared subscriber/router boilerplate into a generic adapter
// if additional notification types are added.

package gcppubsubadapters

import (
	"context"
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	v1 "github.com/GoogleChrome/webstatus.dev/lib/event/webhookjob/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

// WebhookSender defines the interface for sending webhooks.
type WebhookSender interface {
	SendWebhook(ctx context.Context, job workertypes.IncomingWebhookDeliveryJob) error
}

type WebhookWorkerSubscriberAdapter struct {
	sender          WebhookSender
	eventSubscriber EventSubscriber
	subscriptionID  string
	router          *event.Router
}

func NewWebhookWorkerSubscriberAdapter(
	sender WebhookSender,
	eventSubscriber EventSubscriber,
	subscriptionID string,
) *WebhookWorkerSubscriberAdapter {
	router := event.NewRouter()

	ret := &WebhookWorkerSubscriberAdapter{
		sender:          sender,
		eventSubscriber: eventSubscriber,
		subscriptionID:  subscriptionID,
		router:          router,
	}

	event.Register(router, ret.processWebhookJobEvent)

	return ret
}

func (a *WebhookWorkerSubscriberAdapter) Subscribe(ctx context.Context) error {
	return a.eventSubscriber.Subscribe(ctx, a.subscriptionID, func(ctx context.Context,
		msgID string, data []byte) error {
		return a.router.HandleMessage(ctx, msgID, data)
	})
}

func (a *WebhookWorkerSubscriberAdapter) processWebhookJobEvent(ctx context.Context,
	eventID string, event v1.WebhookJobEvent) error {
	slog.InfoContext(ctx, "received webhook job event", "eventID", eventID)

	job := workertypes.IncomingWebhookDeliveryJob{
		WebhookDeliveryJob: workertypes.WebhookDeliveryJob{
			SubscriptionID: event.SubscriptionID,
			WebhookType:    event.WebhookType.ToWorkerTypeWebhookType(),
			WebhookURL:     event.WebhookURL,
			ChannelID:      event.ChannelID,
			Triggers:       event.ToWorkerTypeJobTriggers(),
			SummaryRaw:     event.SummaryRaw,
			Metadata: workertypes.DeliveryMetadata{
				EventID:     event.Metadata.EventID,
				SearchID:    event.Metadata.SearchID,
				SearchName:  "",
				Query:       event.Metadata.Query,
				Frequency:   event.Metadata.Frequency.ToWorkerTypeJobFrequency(),
				GeneratedAt: event.Metadata.GeneratedAt,
			},
		},
		WebhookEventID: eventID,
	}

	return a.sender.SendWebhook(ctx, job)
}
