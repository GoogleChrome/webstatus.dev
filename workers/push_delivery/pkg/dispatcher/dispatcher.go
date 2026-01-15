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

package dispatcher

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type SubscriptionFinder interface {
	// FindSubscribers retrieves all active subscriptions for the given search and frequency.
	// The adapter is responsible for unmarshalling channel configs and sorting them into the set.
	FindSubscribers(ctx context.Context, searchID string,
		frequency workertypes.JobFrequency) (*workertypes.SubscriberSet, error)
}

type DeliveryPublisher interface {
	PublishEmailJob(ctx context.Context, job workertypes.EmailDeliveryJob) error
}

// SummaryParser abstracts the logic for parsing the event summary blob.
type SummaryParser func(data []byte, v workertypes.SummaryVisitor) error

type Dispatcher struct {
	finder    SubscriptionFinder
	publisher DeliveryPublisher
	parser    SummaryParser
}

func NewDispatcher(finder SubscriptionFinder, publisher DeliveryPublisher) *Dispatcher {
	return &Dispatcher{
		finder:    finder,
		publisher: publisher,
		parser:    workertypes.ParseEventSummary,
	}
}

// ProcessEvent is the main entry point for the worker.
// It handles the "Fan-Out" logic: One Event -> Many Delivery Jobs.
func (d *Dispatcher) ProcessEvent(ctx context.Context,
	metadata workertypes.DispatchEventMetadata, summary []byte) error {
	slog.InfoContext(ctx, "processing event", "event_id", metadata.EventID, "search_id", metadata.SearchID)

	// 1. Generate Delivery Jobs from Event Summary
	gen := &deliveryJobGenerator{
		finder:   d.finder,
		metadata: metadata,
		// We pass the raw summary bytes down so it can be attached to the jobs
		// without needing to re-marshal the struct.
		rawSummary: summary,
		emailJobs:  nil,
	}

	if err := d.parser(gen.rawSummary, gen); err != nil {
		return fmt.Errorf("failed to parse event summary: %w", err)
	}

	totalJobs := gen.JobCount()
	if totalJobs == 0 {
		slog.InfoContext(ctx, "no delivery jobs generated", "event_id", metadata.EventID)

		return nil
	}

	slog.InfoContext(ctx, "dispatching jobs", "count", totalJobs)

	// 2. Publish Delivery Jobs
	successCount := 0
	failCount := 0

	// Publish Email Jobs
	for _, job := range gen.emailJobs {
		if err := d.publisher.PublishEmailJob(ctx, job); err != nil {
			slog.ErrorContext(ctx, "failed to publish email job",
				"subscription_id", job.SubscriptionID, "error", err)
			failCount++
		} else {
			successCount++
		}
	}

	// TODO: Webhook jobs would be published here similarly
	// https://github.com/GoogleChrome/webstatus.dev/issues/1859

	slog.InfoContext(ctx, "dispatch complete",
		"event_id", metadata.EventID,
		"sent", successCount,
		"failed", failCount,
		"total_candidates", totalJobs)

	if failCount > 0 {
		return fmt.Errorf("partial failure: %d/%d jobs failed to publish", failCount, totalJobs)
	}

	return nil
}

// deliveryJobGenerator implements workertypes.SummaryVisitor to generate jobs from V1 summaries.
type deliveryJobGenerator struct {
	finder     SubscriptionFinder
	metadata   workertypes.DispatchEventMetadata
	rawSummary []byte
	emailJobs  []workertypes.EmailDeliveryJob
}

func (g *deliveryJobGenerator) VisitV1(s workertypes.EventSummary) error {
	// 1. Find Subscribers
	subscribers, err := g.finder.FindSubscribers(
		// TODO: modify Visitor to pass context down
		// https://github.com/GoogleChrome/webstatus.dev/issues/2132
		context.TODO(),
		g.metadata.SearchID,
		g.metadata.Frequency)
	if err != nil {
		return fmt.Errorf("failed to find subscribers: %w", err)
	}

	if subscribers == nil {
		return nil
	}

	deliveryMetadata := workertypes.DeliveryMetadata{
		EventID:     g.metadata.EventID,
		SearchID:    g.metadata.SearchID,
		Query:       g.metadata.Query,
		Frequency:   g.metadata.Frequency,
		GeneratedAt: g.metadata.GeneratedAt,
	}

	// 2. Filter & Create Jobs
	// Iterate Emails
	for _, sub := range subscribers.Emails {
		if !shouldNotifyV1(sub.Triggers, s) {
			continue
		}
		g.emailJobs = append(g.emailJobs, workertypes.EmailDeliveryJob{
			SubscriptionID: sub.SubscriptionID,
			RecipientEmail: sub.EmailAddress,
			SummaryRaw:     g.rawSummary,
			Metadata:       deliveryMetadata,
			ChannelID:      sub.ChannelID,
			Triggers:       sub.Triggers,
		})
	}

	// TODO: Iterate Webhooks when supported.
	// https://github.com/GoogleChrome/webstatus.dev/issues/1859

	return nil
}

// JobCount returns the total number of delivery jobs generated.
func (g *deliveryJobGenerator) JobCount() int {
	// TODO: When we add Webhook jobs, sum them here too.

	return len(g.emailJobs)
}

// shouldNotifyV1 determines if the V1 event summary matches any of the user's triggers.
func shouldNotifyV1(triggers []workertypes.JobTrigger, summary workertypes.EventSummary) bool {
	// 1. Determine if summary has changes.
	hasChanges := summary.Categories.Added > 0 ||
		summary.Categories.Removed > 0 ||
		summary.Categories.Updated > 0 ||
		summary.Categories.Moved > 0 ||
		summary.Categories.Split > 0 ||
		summary.Categories.QueryChanged > 0

	if !hasChanges {
		return false
	}

	// 2. Iterate triggers and check highlights.
	for _, t := range triggers {
		if matchesTrigger(t, summary) {
			return true
		}
	}

	return false
}

func matchesTrigger(t workertypes.JobTrigger, summary workertypes.EventSummary) bool {
	for _, h := range summary.Highlights {
		if h.MatchesTrigger(t) {
			return true
		}
	}

	return false
}
