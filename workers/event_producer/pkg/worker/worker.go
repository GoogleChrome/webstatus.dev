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

package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/blobtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/event"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/GoogleChrome/webstatus.dev/workers/event_producer/pkg/differ"
	"github.com/google/uuid"
)

// Job represents the input message triggering the worker.
type Job struct {
	SearchID     string `json:"searchId"`
	SnapshotType string `json:"snapshotType"`
	WorkerID     string `json:"workerId"`
}

// Repository defines the transactional database operations.
type Repository interface {
	// Locking
	TryAcquireLock(ctx context.Context, searchID, snapshotType, workerID string, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, searchID, snapshotType, workerID string) error

	// Data Retrieval
	// GetSavedSearch returns the definition (Query) needed by the Differ.
	GetSavedSearch(ctx context.Context, searchID string) (*backend.SavedSearch, error)
	// GetSavedSearchState returns the location of the previous snapshot.
	GetSavedSearchState(ctx context.Context, searchID, snapshotType string) (*workertypes.SavedSearchState, error)

	// State Updates
	// UpdateStateOnly updates the 'LastKnownStateBlobPath' without creating an event.
	// Used for initialization or silent updates (e.g. query change without data change).
	UpdateStateOnly(ctx context.Context, searchID, snapshotType, workerID string,
		req workertypes.SavedSearchStateUpdateRequest) error

	// PublishEvent transactionally updates the State AND inserts the NotificationEvent.
	// This is the "Commit" step for a diff.
	PublishEvent(ctx context.Context, event workertypes.NotificationEventRequest) error
}

// BlobStore defines the contract for reading and writing GCS blobs.
type BlobStore interface {
	// ReadBlob returns the raw bytes. Returns (nil, nil) if not found (Cold Start).
	ReadBlob(ctx context.Context, path string) ([]byte, error)
	// WriteBlob uploads data.
	WriteBlob(ctx context.Context, path string, data []byte) error
}

// Publisher defines the contract for notifying the Delivery worker.
type Publisher interface {
	// Publish sends the Event ID to the delivery queue.
	Publish(ctx context.Context, topicID string, data []byte) (string, error)
}

// Differ defines the contract for detecting changes between states.
// This interface allows mocking the complex diff/reconciliation logic when testing the Worker.
type Differ interface {
	Run(ctx context.Context, searchID string, query string, previousStateBytes []byte) (
		[]byte, *differ.FeatureDiff, bool, error)
}

// Constants for DB Enums.
const (
	ReasonDataUpdated = "DATA_UPDATED"
	ReasonQueryEdited = "QUERY_EDITED"
)

type Worker struct {
	repo      Repository
	blobStore BlobStore
	publisher Publisher
	differ    *differ.FeatureDiffer
	// Config
	notificationTopicID string
	stateBucket         string // Used to construct GCS paths
}

func NewWorker(repo Repository, blobStore BlobStore, pub Publisher, diff *differ.FeatureDiffer,
	topicID, bucket string) *Worker {
	return &Worker{
		repo:                repo,
		blobStore:           blobStore,
		publisher:           pub,
		differ:              diff,
		notificationTopicID: topicID,
		stateBucket:         bucket,
	}
}

func (w *Worker) Process(ctx context.Context, job Job) error {
	logger := slog.With("search_id", job.SearchID, "worker_id", job.WorkerID)

	// 1. Acquire Lock (Fail-Fast)
	// We use a short TTL (e.g. 10 mins) to prevent zombies.
	locked, err := w.repo.TryAcquireLock(ctx, job.SearchID, job.SnapshotType, job.WorkerID, 10*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		logger.InfoContext(ctx, "skipping job: lock held by another worker")

		return nil
	}
	defer func() {
		if err := w.repo.ReleaseLock(ctx, job.SearchID, job.SnapshotType, job.WorkerID); err != nil {
			logger.WarnContext(ctx, "failed to release lock", "error", err)
		}
	}()

	// 2. Fetch Inputs (Definition + Previous State)
	searchDef, err := w.repo.GetSavedSearch(ctx, job.SearchID)
	if err != nil {
		return fmt.Errorf("failed to get saved search definition: %w", err)
	}

	searchState, err := w.repo.GetSavedSearchState(ctx, job.SearchID, job.SnapshotType)
	if err != nil {
		return fmt.Errorf("failed to get saved search state: %w", err)
	}

	// 3. Load Previous Blob (if exists)
	var oldStateBytes []byte
	if searchState != nil && searchState.StateBlobPath != nil {
		oldStateBytes, err = w.blobStore.ReadBlob(ctx, *searchState.StateBlobPath)
		if err != nil {
			// If blob is missing (but DB says it exists), we treat it as corruption -> Fatal?
			// Or we fall back to Cold Start. Let's log and treat as Cold Start for resilience.
			logger.ErrorContext(ctx, "state blob missing, forcing cold start", "path",
				*searchState.StateBlobPath, "error", err)
			oldStateBytes = nil
		}
	}

	// 4. Run the Differ (The Brain)
	newStateBytes, diff, shouldWrite, err := w.differ.Run(ctx, job.SearchID, searchDef.Query, oldStateBytes)
	if err != nil {
		if errors.Is(err, differ.ErrFatal) {
			// Fatal errors (e.g. corrupt state) should not retry.
			// In a real worker, we might return nil here to ACK the message and stop the loop,
			// alerting via logs.
			logger.ErrorContext(ctx, "fatal differ error", "error", err)

			return nil
		}
		// Transient errors (network) should retry (NACK).
		return err
	}

	if !shouldWrite {
		logger.InfoContext(ctx, "no changes detected")

		return nil
	}

	// 5. Persist New State (Snapshot)
	// We generate a deterministic path: searches/{id}/{type}/state_{timestamp}.json
	// Or simply overwrite a versioned path. Let's use unique paths for safety.
	newStatePath := fmt.Sprintf("searches/%s/%s/state_%d.json", job.SearchID, job.SnapshotType, time.Now().Unix())
	if err := w.blobStore.WriteBlob(ctx, newStatePath, newStateBytes); err != nil {
		return fmt.Errorf("failed to write new state blob: %w", err)
	}

	// 6. Handle Diff & Events
	if !diff.HasChanges() {
		// Silent Update (e.g. Cold Start or Query Change w/ Flush Failed).
		// We update the DB state pointer, but do NOT fire an event.
		logger.InfoContext(ctx, "updating state without event (silent update)")
		if err := w.repo.UpdateStateOnly(ctx, job.SearchID, job.SnapshotType, job.WorkerID,
			workertypes.SavedSearchStateUpdateRequest{
				StateBlobPath: &newStatePath,
				UpdateMask: []workertypes.SavedSearchStateUpdateRequestUpdateMask{
					workertypes.SavedSearchStateUpdateRequestStateBlobPath},
			}); err != nil {
			return fmt.Errorf("failed to update state: %w", err)
		}

		return nil
	}

	// 7. Persist Diff Blob
	// Diffs are immutable event data.
	eventID := uuid.New().String()
	diffPath := fmt.Sprintf("events/%s/%s.json", job.SearchID, eventID)

	diffPayload := differ.FeatureDiffSnapshot{
		Metadata: differ.DiffMetadata{
			GeneratedAt: time.Now(),
			EventID:     eventID,
			SearchID:    job.SearchID,
		},
		Data: *diff,
	}

	diffBytes, err := blobtypes.NewBlob(diffPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal diff envelope: %w", err)
	}

	if err := w.blobStore.WriteBlob(ctx, diffPath, diffBytes); err != nil {
		return fmt.Errorf("failed to write diff blob: %w", err)
	}

	// 8. Commit Transaction (Publish Event + Update State)
	reasons := []string{}
	if diff.QueryChanged {
		reasons = append(reasons, ReasonQueryEdited)
	}
	// If there are adds/removes/mods/moves/splits, it's a data update
	if len(diff.Added) > 0 || len(diff.Removed) > 0 || len(diff.Modified) > 0 ||
		len(diff.Moves) > 0 || len(diff.Splits) > 0 {
		reasons = append(reasons, ReasonDataUpdated)
	}

	summary := diff.Summarize()

	req := workertypes.NotificationEventRequest{
		EventID:      eventID,
		SearchID:     job.SearchID,
		SnapshotType: job.SnapshotType,
		Reasons:      reasons,
		DiffBlobPath: diffPath,
		Summary:      summary,
		NewStatePath: newStatePath,
		WorkerID:     job.WorkerID,
	}

	if err := w.repo.PublishEvent(ctx, req); err != nil {
		return fmt.Errorf("failed to publish event to db: %w", err)
	}

	eventBytes, err := event.New(workertypes.NotificationEventCreatedV1{ID: eventID})
	if err != nil {
		// Should not happen.
		return fmt.Errorf("failed to create event: %w", err)
	}

	// 9. Notify Delivery Worker
	// The Delivery worker will look it up in Spanner.
	if _, err := w.publisher.Publish(ctx, w.notificationTopicID, eventBytes); err != nil {
		// If this fails, the DB is already updated. This is an inconsistency risk.
		// However, since we return an error here, the Ingestion Job will retry.
		// The Ingestion Job is idempotent-ish:
		// - It will re-acquire lock (if released or expired)
		// - It will fetch new data (might have changed slightly)
		// - It will produce a NEW event.
		// Result: Duplicate events in DB, but at least we ensure delivery.

		return fmt.Errorf("failed to publish notification trigger: %w", err)
	}

	logger.InfoContext(ctx, "event published successfully", "event_id", eventID)

	return nil
}
