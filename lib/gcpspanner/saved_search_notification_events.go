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

package gcpspanner

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
)

const savedSearchNotificationEventsTable = "SavedSearchNotificationEvents"

type SavedSearchNotificationEvent struct {
	ID            string                  `spanner:"EventId"`
	SavedSearchID string                  `spanner:"SavedSearchId"`
	SnapshotType  SavedSearchSnapshotType `spanner:"SnapshotType"`
	Timestamp     time.Time               `spanner:"Timestamp"`
	EventType     string                  `spanner:"EventType"`
	Reasons       []string                `spanner:"Reasons"`
	BlobPath      string                  `spanner:"BlobPath"`
	DiffBlobPath  string                  `spanner:"DiffBlobPath"`
	Summary       spanner.NullJSON        `spanner:"Summary"`
}

type SavedSearchNotificationCreateRequest struct {
	SavedSearchID string                  `spanner:"SavedSearchId"`
	SnapshotType  SavedSearchSnapshotType `spanner:"SnapshotType"`
	Timestamp     time.Time               `spanner:"Timestamp"`
	EventType     string                  `spanner:"EventType"`
	Reasons       []string                `spanner:"Reasons"`
	BlobPath      string                  `spanner:"BlobPath"`
	DiffBlobPath  string                  `spanner:"DiffBlobPath"`
	Summary       spanner.NullJSON        `spanner:"Summary"`
}

func (c *Client) GetSavedSearchNotificationEvent(
	ctx context.Context, eventID string) (*SavedSearchNotificationEvent, error) {
	r := newEntityReader[savedSearchNotificationEventMapper, SavedSearchNotificationEvent, string](c)

	return r.readRowByKey(ctx, eventID)
}

// savedSearchNotificationEventMapper implements the necessary interfaces for the generic helpers.
type savedSearchNotificationEventMapper struct{}

func (m savedSearchNotificationEventMapper) Table() string {
	return savedSearchNotificationEventsTable
}

func (m savedSearchNotificationEventMapper) SelectOne(EventID string) spanner.Statement {
	return spanner.Statement{
		SQL:    "SELECT * FROM SavedSearchNotificationEvents WHERE EventId = @EventId",
		Params: map[string]any{"EventId": EventID},
	}
}

func (m savedSearchNotificationEventMapper) NewEntity(id string, req SavedSearchNotificationCreateRequest) (
	SavedSearchNotificationEvent, error) {
	return SavedSearchNotificationEvent{
		ID:            id,
		SavedSearchID: req.SavedSearchID,
		SnapshotType:  req.SnapshotType,
		Timestamp:     req.Timestamp,
		EventType:     req.EventType,
		Reasons:       req.Reasons,
		BlobPath:      req.BlobPath,
		Summary:       req.Summary,
		DiffBlobPath:  req.DiffBlobPath,
	}, nil
}

// PublishSavedSearchNotificationEvent records a new saved search notification event.
// This saves the event and updates the state pointer, but explicitly KEEPS the lock.
// The worker is expected to call ReleaseLock via defer.
func (c *Client) PublishSavedSearchNotificationEvent(ctx context.Context,
	event SavedSearchNotificationCreateRequest, newStatePath, workerID string, opts ...CreateOption) (*string, error) {
	var id *string
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Check Lock & Update State (Using ReadInspectMutateWithTransaction)
		key := savedSearchStateKey{SavedSearchID: event.SavedSearchID,
			SnapshotType: SavedSearchSnapshotType(event.SnapshotType)}

		err := newEntityMutator[savedSearchStateMapper, SavedSearchState](c).readInspectMutateWithTransaction(ctx, key,
			func(_ context.Context, existing *SavedSearchState) (*spanner.Mutation, error) {
				if existing == nil {
					return nil, ErrQueryReturnedNoResults
				}
				// Fencing Check: Verify I still own the lock before committing
				if existing.WorkerLockID == nil || *existing.WorkerLockID != workerID {
					return nil, ErrAlreadyLocked
				}

				// Update Logic: Set New Path + KEEP Lock
				// We update the BlobPath but explicitly copy the lock identity from the existing row.
				newState := SavedSearchState{
					SavedSearchID:          event.SavedSearchID,
					SnapshotType:           event.SnapshotType,
					LastKnownStateBlobPath: &newStatePath,
					WorkerLockID:           existing.WorkerLockID,        // KEEP LOCK
					WorkerLockExpiresAt:    existing.WorkerLockExpiresAt, // KEEP EXPIRY
				}

				return spanner.InsertOrUpdateStruct(savedSearchStateTableName, newState)
			}, txn)

		if err != nil {
			return err
		}

		// Insert Event
		newID, err := newEntityCreator[savedSearchNotificationEventMapper](c).createWithTransaction(ctx, txn, event,
			opts...)
		if err != nil {
			return err
		}
		id = newID

		return nil
	})

	return id, err
}

type savedSearchNotificationEventBySearchAndSnapshotTypeKey struct {
	SavedSearchID string
	SnapshotType  SavedSearchSnapshotType
}

type savedSearchNotificationEventBySearchAndSnapshotTypeMapper struct{}

func (m savedSearchNotificationEventBySearchAndSnapshotTypeMapper) Table() string {
	return savedSearchNotificationEventsTable
}

func (m savedSearchNotificationEventBySearchAndSnapshotTypeMapper) SelectOne(
	key savedSearchNotificationEventBySearchAndSnapshotTypeKey) spanner.Statement {
	return spanner.Statement{
		SQL: `SELECT * FROM SavedSearchNotificationEvents
			  WHERE SavedSearchId = @SavedSearchId AND SnapshotType = @SnapshotType
			  ORDER BY Timestamp DESC
			  LIMIT 1`,
		Params: map[string]any{
			"SavedSearchId": key.SavedSearchID,
			"SnapshotType":  key.SnapshotType,
		},
	}
}

func (c *Client) GetLatestSavedSearchNotificationEvent(
	ctx context.Context, savedSearchID string,
	snapshotType SavedSearchSnapshotType) (*SavedSearchNotificationEvent, error) {
	r := newEntityReader[savedSearchNotificationEventBySearchAndSnapshotTypeMapper, SavedSearchNotificationEvent,
		savedSearchNotificationEventBySearchAndSnapshotTypeKey](c)

	key := savedSearchNotificationEventBySearchAndSnapshotTypeKey{
		SavedSearchID: savedSearchID,
		SnapshotType:  snapshotType,
	}

	return r.readRowByKey(ctx, key)
}
