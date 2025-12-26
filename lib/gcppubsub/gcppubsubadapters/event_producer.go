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
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	batchrefreshv1 "github.com/GoogleChrome/webstatus.dev/lib/event/batchrefreshtrigger/v1"
	featurediffv1 "github.com/GoogleChrome/webstatus.dev/lib/event/featurediff/v1"
	refreshv1 "github.com/GoogleChrome/webstatus.dev/lib/event/refreshsearchcommand/v1"
	searchconfigv1 "github.com/GoogleChrome/webstatus.dev/lib/event/searchconfigurationchanged/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type EventProducerSearchMessageHandler interface {
	ProcessSearch(ctx context.Context, searchID string, query string,
		frequency workertypes.JobFrequency, triggerID string) error
}

type EventProducerBatchUpdateHandler interface {
	ProcessBatchUpdate(ctx context.Context, triggerID string, frequency workertypes.JobFrequency) error
}

type EventSubscriber interface {
	Subscribe(ctx context.Context, subID string,
		handler func(ctx context.Context, msgID string, data []byte) error) error
}

type SubscriberConfig struct {
	SearchSubscriptionID      string
	BatchUpdateSubscriptionID string
}

type EventProducerSubscriberAdapter struct {
	searchEventHandler EventProducerSearchMessageHandler
	batchUpdateHandler EventProducerBatchUpdateHandler
	eventSubscriber    EventSubscriber
	config             SubscriberConfig
	searchEventRouter  *event.Router
	batchUpdateRouter  *event.Router
}

func NewEventProducerSubscriberAdapter(
	searchMessageHandler EventProducerSearchMessageHandler,
	batchUpdateHandler EventProducerBatchUpdateHandler,
	eventSubscriber EventSubscriber,
	config SubscriberConfig,
) *EventProducerSubscriberAdapter {
	searchEventRouter := event.NewRouter()

	batchUpdateRouter := event.NewRouter()

	ret := &EventProducerSubscriberAdapter{
		searchEventHandler: searchMessageHandler,
		batchUpdateHandler: batchUpdateHandler,
		eventSubscriber:    eventSubscriber,
		config:             config,
		searchEventRouter:  searchEventRouter,
		batchUpdateRouter:  batchUpdateRouter,
	}

	event.Register(searchEventRouter, ret.processRefreshSearchCommand)
	event.Register(searchEventRouter, ret.processSearchConfigurationChangedEvent)

	event.Register(batchUpdateRouter, ret.processBatchUpdateCommand)

	return ret
}

func (a *EventProducerSubscriberAdapter) processRefreshSearchCommand(ctx context.Context,
	eventID string, event refreshv1.RefreshSearchCommand) error {
	slog.InfoContext(ctx, "received refresh search command", "eventID", eventID, "event", event)

	return a.searchEventHandler.ProcessSearch(ctx, event.SearchID, event.Query,
		event.Frequency.ToWorkerTypeJobFrequency(), eventID)
}

func (a *EventProducerSubscriberAdapter) processSearchConfigurationChangedEvent(ctx context.Context,
	eventID string, event searchconfigv1.SearchConfigurationChangedEvent) error {
	slog.InfoContext(ctx, "received search configuration changed event", "eventID", eventID, "event", event)

	return a.searchEventHandler.ProcessSearch(ctx, event.SearchID, event.Query,
		event.Frequency.ToWorkerTypeJobFrequency(), eventID)
}

func (a *EventProducerSubscriberAdapter) Subscribe(ctx context.Context) error {
	return RunGroup(ctx,
		// Handler 1: Search
		func(ctx context.Context) error {
			return a.eventSubscriber.Subscribe(ctx, a.config.SearchSubscriptionID,
				func(ctx context.Context, msgID string, data []byte) error {
					return a.searchEventRouter.HandleMessage(ctx, msgID, data)
				})
		},
		// Handler 2: Batch Update
		func(ctx context.Context) error {
			return a.eventSubscriber.Subscribe(ctx, a.config.BatchUpdateSubscriptionID,
				func(ctx context.Context, msgID string, data []byte) error {
					return a.batchUpdateRouter.HandleMessage(ctx, msgID, data)
				})
		},
	)
}

func (a *EventProducerSubscriberAdapter) processBatchUpdateCommand(ctx context.Context,
	eventID string, event batchrefreshv1.BatchRefreshTrigger) error {
	slog.InfoContext(ctx, "received batch update command", "eventID", eventID, "event", event)

	return a.batchUpdateHandler.ProcessBatchUpdate(ctx, eventID,
		event.Frequency.ToWorkerTypeJobFrequency())
}

type EventPublisher interface {
	Publish(ctx context.Context, topicID string, data []byte) (string, error)
}

type EventProducerPublisherAdapter struct {
	eventPublisher EventPublisher
	topicID        string
}

func NewEventProducerPublisherAdapter(eventPublisher EventPublisher, topicID string) *EventProducerPublisherAdapter {
	return &EventProducerPublisherAdapter{
		eventPublisher: eventPublisher,
		topicID:        topicID,
	}
}

func (a *EventProducerPublisherAdapter) Publish(ctx context.Context,
	req workertypes.PublishEventRequest) (string, error) {
	b, err := event.New(featurediffv1.FeatureDiffEvent{
		EventID:       req.EventID,
		SearchID:      req.SearchID,
		Query:         req.Query,
		Summary:       req.Summary,
		StateID:       req.StateID,
		StateBlobPath: req.StateBlobPath,
		DiffID:        req.DiffID,
		DiffBlobPath:  req.DiffBlobPath,
		GeneratedAt:   req.GeneratedAt,
		Frequency:     featurediffv1.ToJobFrequency(req.Frequency),
		Reasons:       featurediffv1.ToReasons(req.Reasons),
	})
	if err != nil {
		return "", err
	}

	return a.eventPublisher.Publish(ctx, a.topicID, b)
}
