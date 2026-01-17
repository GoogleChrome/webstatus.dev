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
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/GoogleChrome/webstatus.dev/workers/event_producer/pkg/differ"
)

// FeatureDiffer encapsulates the core logic for comparing the live state.
type FeatureDiffer interface {
	Run(ctx context.Context, searchID, query, eventID string, previousStateBytes []byte) (*differ.DiffResult, error)
}

// BlobStorage handles the persistence of opaque data blobs (State Snapshots and Diff Reports).
type BlobStorage interface {
	Store(ctx context.Context, dirs []string, key string, data []byte) (string, error)
	Get(ctx context.Context, fullpath string) ([]byte, error)
}

// EventMetadataStore handles the publishing and retrieval of event metadata.
type EventMetadataStore interface {
	AcquireLock(ctx context.Context, searchID string, frequency workertypes.JobFrequency,
		workerID string, lockTTL time.Duration) error
	ReleaseLock(ctx context.Context, searchID string, frequency workertypes.JobFrequency,
		workerID string) error
	PublishEvent(ctx context.Context, req workertypes.PublishEventRequest) error
	// GetLatestEvent retrieves the last known event for a search to establish continuity.
	GetLatestEvent(ctx context.Context,
		frequency workertypes.JobFrequency, searchID string) (*workertypes.LatestEventInfo, error)
}

// EventPublisher handles broadcasting the event to the rest of the system (e.g. via Pub/Sub).
type EventPublisher interface {
	Publish(ctx context.Context, req workertypes.PublishEventRequest) (string, error)
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

const (
	StateDir = "state"
	DiffDir  = "diff"
)

func baseblobname(id string) string {
	return fmt.Sprintf("%s.json", id)
}

func getDefaultLockTTL() time.Duration {
	return 2 * time.Minute
}

// ProcessSearch is the main entry point triggered when a search query needs to be checked.
// triggerID is the unique ID for this execution (e.g., from a Pub/Sub message).
func (p *EventProducer) ProcessSearch(ctx context.Context, searchID string, query string,
	frequency workertypes.JobFrequency, triggerID string) error {
	// 0. Acquire Lock
	// TODO: For now, use the triggerID as the worker ID.
	// https://github.com/GoogleChrome/webstatus.dev/issues/2123
	workerID := triggerID
	if err := p.metaStore.AcquireLock(ctx, searchID, frequency, workerID, getDefaultLockTTL()); err != nil {
		slog.ErrorContext(ctx, "failed to acquire lock", "search_id", searchID, "worker_id", workerID, "error", err)

		return fmt.Errorf("%w: failed to acquire lock: %w", event.ErrTransientFailure, err)
	}
	defer func() {
		if err := p.metaStore.ReleaseLock(ctx, searchID, frequency, workerID); err != nil {
			slog.ErrorContext(ctx, "failed to release lock", "search_id", searchID, "worker_id", workerID, "error", err)
		}
	}()
	// 1. Fetch Previous State
	// We need the last known state to compute the diff.
	lastEvent, err := p.metaStore.GetLatestEvent(ctx, frequency, searchID)
	if err != nil && !errors.Is(err, workertypes.ErrLatestEventNotFound) {
		return fmt.Errorf("failed to get latest event info: %w", err)
	}

	var previousStateBytes []byte
	if lastEvent != nil && lastEvent.StateBlobPath != "" {
		// If we have history, fetch the actual bytes from "Cold Storage"
		previousStateBytes, err = p.blobStore.Get(ctx, lastEvent.StateBlobPath)
		if err != nil {
			slog.ErrorContext(ctx, "unable to fetch previous state blob", "error", err, "search_id", searchID,
				"state_blob_path", lastEvent.StateBlobPath)

			return fmt.Errorf("failed to fetch previous state blob: %w", err)
		}
	} else {
		slog.InfoContext(ctx, "no prior history found", "search_id", searchID)
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
	statePath, err := p.blobStore.Store(ctx, []string{StateDir}, baseblobname(result.State.ID), result.State.Bytes)
	if err != nil {
		return fmt.Errorf("failed to store state blob: %w", err)
	}

	var diffPath string
	if len(result.Diff.Bytes) > 0 {
		if diffPath, err = p.blobStore.Store(ctx, []string{DiffDir},
			baseblobname(result.Diff.ID), result.Diff.Bytes); err != nil {
			return fmt.Errorf("failed to store diff blob: %w", err)
		}
	}

	// 4. Publish Metadata (Hot Storage)
	req := workertypes.PublishEventRequest{
		EventID:       triggerID,
		SearchID:      searchID,
		StateID:       result.State.ID,
		StateBlobPath: statePath,
		DiffID:        result.Diff.ID,
		DiffBlobPath:  diffPath,
		Summary:       result.Summary,
		Reasons:       result.Reasons,
		Frequency:     frequency,
		Query:         query,
		GeneratedAt:   result.GeneratedAt,
	}

	if err := p.metaStore.PublishEvent(ctx, req); err != nil {
		return fmt.Errorf("failed to publish event metadata: %w", err)
	}

	// 6. Publish Notification (Topic)
	// Notify downstream workers that a new event is available.
	eventID, err := p.publisher.Publish(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to publish event notification: %w", err)
	}

	slog.InfoContext(ctx, "event published successfully", "event_id", triggerID, "reasons", result.Reasons,
		"downstream_event_id", eventID)

	return nil
}
