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

package producer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/GoogleChrome/webstatus.dev/workers/event_producer/pkg/differ"
)

// FeatureDiffer encapsulates the core logic for comparing the live state.
type FeatureDiffer interface {
	Run(ctx context.Context, searchID, query, eventID string, previousStateBytes []byte) (*differ.DiffResult, error)
}

// BlobStorage handles the persistence of opaque data blobs (State Snapshots and Diff Reports).
type BlobStorage interface {
	Store(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
}

// EventMetadataStore handles the publishing and retrieval of event metadata.
type EventMetadataStore interface {
	PublishEvent(ctx context.Context, req workertypes.PublishEventRequest) error
	// GetLatestEvent retrieves the last known event for a search to establish continuity.
	GetLatestEvent(ctx context.Context, searchID string) (*workertypes.LatestEventInfo, error)
}

// EventPublisher handles broadcasting the event to the rest of the system (e.g. via Pub/Sub).
type EventPublisher interface {
	Publish(ctx context.Context, req workertypes.PublishEventRequest) error
}

// EventProducer orchestrates the diffing and publishing pipeline.
type EventProducer struct {
	differ    FeatureDiffer
	blobStore BlobStorage
	metaStore EventMetadataStore
	publisher EventPublisher
}

func NewEventProducer(d FeatureDiffer, b BlobStorage, m EventMetadataStore, p EventPublisher) *EventProducer {
	return &EventProducer{
		differ:    d,
		blobStore: b,
		metaStore: m,
		publisher: p,
	}
}

// ProcessSearch is the main entry point triggered when a search query needs to be checked.
// triggerID is the unique ID for this execution (e.g., from a Pub/Sub message).
func (p *EventProducer) ProcessSearch(ctx context.Context, searchID string, query string, triggerID string) error {
	// 1. Fetch Previous State
	// We need the last known state to compute the diff.
	lastEvent, err := p.metaStore.GetLatestEvent(ctx, searchID)
	if err != nil {
		return fmt.Errorf("failed to get latest event info: %w", err)
	}

	var previousStateBytes []byte
	if lastEvent != nil && lastEvent.StateID != "" {
		// If we have history, fetch the actual bytes from "Cold Storage"
		previousStateBytes, err = p.blobStore.Get(ctx, lastEvent.StateID)
		if err != nil {
			return fmt.Errorf("failed to fetch previous state blob: %w", err)
		}
	}

	// 2. Run the Differ
	// This performs the logic: Fetch Live -> Compare(Old, New) -> Generate Artifacts
	result, err := p.differ.Run(ctx, searchID, query, triggerID, previousStateBytes)
	if err != nil {
		if errors.Is(err, differ.ErrNoChangesDetected) {
			slog.InfoContext(ctx, "no changes detected", "search_id", searchID)

			return nil
		}

		return fmt.Errorf("differ execution failed: %w", err)
	}

	// 3. Store Artifacts (Blob Storage)
	// We have to save both the Full State and the Diff.
	// Note: We are trusting the Differ to have generated valid IDs in the result.
	if err := p.blobStore.Store(ctx, result.State.ID, result.State.Bytes); err != nil {
		return fmt.Errorf("failed to store state blob: %w", err)
	}

	if len(result.Diff.Bytes) > 0 {
		if err := p.blobStore.Store(ctx, result.Diff.ID, result.Diff.Bytes); err != nil {
			return fmt.Errorf("failed to store diff blob: %w", err)
		}
	}

	// 4. Publish Metadata (Hot Storage)
	req := workertypes.PublishEventRequest{
		EventID:  triggerID,
		SearchID: searchID,
		StateID:  result.State.ID,
		DiffID:   result.Diff.ID,
		Summary:  result.Summary,
		Reasons:  result.Reasons,
	}

	if err := p.metaStore.PublishEvent(ctx, req); err != nil {
		return fmt.Errorf("failed to publish event metadata: %w", err)
	}

	// 6. Publish Notification (Topic)
	// Notify downstream workers (e.g. Twitter Bot, RSS) that a new event is available.
	if err := p.publisher.Publish(ctx, req); err != nil {
		return fmt.Errorf("failed to publish event notification: %w", err)
	}

	slog.InfoContext(ctx, "event published successfully", "event_id", triggerID, "reasons", result.Reasons)

	return nil
}
