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

package gcppubsubadapters

import (
	"context"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	v1 "github.com/GoogleChrome/webstatus.dev/lib/event/emailjob/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

// EmailWorkerMessageHandler defines the interface for the Sender logic.
type EmailWorkerMessageHandler interface {
	ProcessMessage(ctx context.Context, job workertypes.IncomingEmailDeliveryJob) error
}

type EmailWorkerSubscriberAdapter struct {
	sender          EmailWorkerMessageHandler
	eventSubscriber EventSubscriber
	subscriptionID  string
	router          *event.Router
}

func NewEmailWorkerSubscriberAdapter(
	sender EmailWorkerMessageHandler,
	eventSubscriber EventSubscriber,
	subscriptionID string,
) *EmailWorkerSubscriberAdapter {
	router := event.NewRouter()

	ret := &EmailWorkerSubscriberAdapter{
		sender:          sender,
		eventSubscriber: eventSubscriber,
		subscriptionID:  subscriptionID,
		router:          router,
	}

	event.Register(router, ret.handleEmailJobEvent)

	return ret
}

func (a *EmailWorkerSubscriberAdapter) Subscribe(ctx context.Context) error {
	return a.eventSubscriber.Subscribe(ctx, a.subscriptionID, func(ctx context.Context,
		msgID string, data []byte) error {
		return a.router.HandleMessage(ctx, msgID, data)
	})
}

func (a *EmailWorkerSubscriberAdapter) handleEmailJobEvent(
	ctx context.Context, msgID string, event v1.EmailJobEvent) error {

	incomingJob := workertypes.IncomingEmailDeliveryJob{
		EmailDeliveryJob: workertypes.EmailDeliveryJob{
			SubscriptionID: event.SubscriptionID,
			RecipientEmail: event.RecipientEmail,
			ChannelID:      event.ChannelID,
			SummaryRaw:     event.SummaryRaw,
			Metadata: workertypes.DeliveryMetadata{
				EventID:     event.Metadata.EventID,
				SearchID:    event.Metadata.SearchID,
				Query:       event.Metadata.Query,
				Frequency:   event.Metadata.Frequency.ToWorkerTypeJobFrequency(),
				GeneratedAt: event.Metadata.GeneratedAt,
			},
			Triggers: event.ToWorkerTypeJobTriggers(),
		},
		EmailEventID: msgID,
	}

	return a.sender.ProcessMessage(ctx, incomingJob)
}
