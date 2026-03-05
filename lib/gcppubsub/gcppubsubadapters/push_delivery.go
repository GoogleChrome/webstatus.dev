// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcppubsubadapters

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	emailjobv1 "github.com/GoogleChrome/webstatus.dev/lib/event/emailjob/v1"
	featurediffv1 "github.com/GoogleChrome/webstatus.dev/lib/event/featurediff/v1"
	webhookjobv1 "github.com/GoogleChrome/webstatus.dev/lib/event/webhookjob/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type PushDeliveryPublisher struct {
	client       EventPublisher
	emailTopic   string
	webhookTopic string
}

func NewPushDeliveryPublisher(client EventPublisher, emailTopic, webhookTopic string) *PushDeliveryPublisher {
	return &PushDeliveryPublisher{
		client:       client,
		emailTopic:   emailTopic,
		webhookTopic: webhookTopic,
	}
}

func (p *PushDeliveryPublisher) PublishEmailJob(ctx context.Context, job workertypes.EmailDeliveryJob) error {
	b, err := event.New(emailjobv1.EmailJobEvent{
		SubscriptionID: job.SubscriptionID,
		RecipientEmail: job.RecipientEmail,
		SummaryRaw:     job.SummaryRaw,
		Metadata: emailjobv1.EmailJobEventMetadata{
			EventID:     job.Metadata.EventID,
			SearchID:    job.Metadata.SearchID,
			Query:       job.Metadata.Query,
			Frequency:   emailjobv1.ToJobFrequency(job.Metadata.Frequency),
			GeneratedAt: job.Metadata.GeneratedAt,
		},
		Triggers:  emailjobv1.ToJobTriggers(job.Triggers),
		ChannelID: job.ChannelID,
	})
	if err != nil {
		return err
	}

	id, err := p.client.Publish(ctx, p.emailTopic, b)
	if err != nil {
		return fmt.Errorf("failed to publish email job: %w", err)
	}
	slog.InfoContext(ctx, "published email job", "id", id, "eventID", job.Metadata.EventID)

	return nil
}

func (p *PushDeliveryPublisher) PublishWebhookJob(ctx context.Context, job workertypes.WebhookDeliveryJob) error {
	b, err := event.New(webhookjobv1.WebhookJobEvent{
		SubscriptionID: job.SubscriptionID,
		WebhookURL:     job.WebhookURL,
		SummaryRaw:     job.SummaryRaw,
		Metadata: webhookjobv1.WebhookJobEventMetadata{
			EventID:     job.Metadata.EventID,
			SearchID:    job.Metadata.SearchID,
			Query:       job.Metadata.Query,
			Frequency:   webhookjobv1.ToJobFrequency(job.Metadata.Frequency),
			GeneratedAt: job.Metadata.GeneratedAt,
		},
		Triggers:  webhookjobv1.ToJobTriggers(job.Triggers),
		ChannelID: job.ChannelID,
	})
	if err != nil {
		return err
	}

	id, err := p.client.Publish(ctx, p.webhookTopic, b)
	if err != nil {
		return fmt.Errorf("failed to publish webhook job: %w", err)
	}
	slog.InfoContext(ctx, "published webhook job", "id", id, "eventID", job.Metadata.EventID)

	return nil
}

// PushDeliveryMessageHandler defines the interface for the Dispatcher logic.
type PushDeliveryMessageHandler interface {
	ProcessEvent(ctx context.Context, metadata workertypes.DispatchEventMetadata, summary []byte) error
}

type PushDeliverySubscriberAdapter struct {
	dispatcher      PushDeliveryMessageHandler
	eventSubscriber EventSubscriber
	subscriptionID  string
	router          *event.Router
}

func NewPushDeliverySubscriberAdapter(
	dispatcher PushDeliveryMessageHandler,
	eventSubscriber EventSubscriber,
	subscriptionID string,
) *PushDeliverySubscriberAdapter {
	router := event.NewRouter()

	ret := &PushDeliverySubscriberAdapter{
		dispatcher:      dispatcher,
		eventSubscriber: eventSubscriber,
		subscriptionID:  subscriptionID,
		router:          router,
	}

	event.Register(router, ret.processFeatureDiffEvent)

	return ret
}

func (a *PushDeliverySubscriberAdapter) Subscribe(ctx context.Context) error {
	return a.eventSubscriber.Subscribe(ctx, a.subscriptionID, func(ctx context.Context,
		msgID string, data []byte) error {
		return a.router.HandleMessage(ctx, msgID, data)
	})
}

func (a *PushDeliverySubscriberAdapter) processFeatureDiffEvent(ctx context.Context,
	eventID string, event featurediffv1.FeatureDiffEvent) error {
	slog.InfoContext(ctx, "received feature diff event", "eventID", eventID)

	metadata := workertypes.DispatchEventMetadata{
		EventID:     event.EventID,
		SearchID:    event.SearchID,
		Query:       event.Query,
		Frequency:   event.Frequency.ToWorkertypes(),
		GeneratedAt: event.GeneratedAt,
	}

	return a.dispatcher.ProcessEvent(ctx, metadata, event.Summary)
}
