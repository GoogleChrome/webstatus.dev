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
	"errors"
	"time"

	"cloud.google.com/go/spanner"
)

const savedSearchStateTableName = "SavedSearchState"

type savedSearchStateMapper struct{}

type savedSearchStateKey struct {
	SavedSearchID string
	SnapshotType  SavedSearchSnapshotType
}

type SavedSearchSnapshotType string

const (
	SavedSearchSnapshotTypeImmediate SavedSearchSnapshotType = "IMMEDIATE"
	SavedSearchSnapshotTypeWeekly    SavedSearchSnapshotType = "WEEKLY"
	SavedSearchSnapshotTypeMonthly   SavedSearchSnapshotType = "MONTHLY"
	SavedSearchSnapshotTypeUnknown   SavedSearchSnapshotType = "UNKNOWN"
)

type SavedSearchState struct {
	SavedSearchID          string                  `spanner:"SavedSearchId"`
	SnapshotType           SavedSearchSnapshotType `spanner:"SnapshotType"`
	LastKnownStateBlobPath *string                 `spanner:"LastKnownStateBlobPath"`
	WorkerLockID           *string                 `spanner:"WorkerLockId"`
	WorkerLockExpiresAt    *time.Time              `spanner:"WorkerLockExpiresAt"`
}

func (m savedSearchStateMapper) SelectOne(key savedSearchStateKey) spanner.Statement {
	return spanner.Statement{
		SQL:    "SELECT * FROM SavedSearchState WHERE SavedSearchId = @SavedSearchId AND SnapshotType = @SnapshotType",
		Params: map[string]any{"SavedSearchId": key.SavedSearchID, "SnapshotType": key.SnapshotType},
	}
}

var (
	ErrAlreadyLocked = errors.New("resource already locked by another worker")
	ErrLockNotOwned  = errors.New("cannot release lock not owned by worker")
)

// TryAcquireSavedSearchStateWorkerLock attempts to acquire a worker lock for the given saved search and snapshot type.
// If the lock is already held by another worker and is still active, ErrAlreadyLocked is returned.
// A caller can re-acquire a lock it already holds it (thereby extending the expiration).
func (c *Client) TryAcquireSavedSearchStateWorkerLock(
	ctx context.Context,
	savedSearchID string,
	snapshotType SavedSearchSnapshotType,
	workerID string,
	ttl time.Duration) (bool, error) {
	writer := newEntityMutator[savedSearchStateMapper, SavedSearchState](c)
	key := savedSearchStateKey{SavedSearchID: savedSearchID, SnapshotType: snapshotType}

	err := writer.readInspectMutate(ctx, key,
		func(_ context.Context, existing *SavedSearchState) (*spanner.Mutation, error) {
			now := c.timeNow()

			// If row exists, is it locked by someone else not the caller?
			if existing != nil {
				isLocked := existing.WorkerLockID != nil && *existing.WorkerLockID != workerID
				isActive := existing.WorkerLockExpiresAt != nil && existing.WorkerLockExpiresAt.After(now)

				if isLocked && isActive {
					return nil, ErrAlreadyLocked
				}
			}

			expiration := now.Add(ttl)

			// We can take the lock.
			newState := SavedSearchState{
				SavedSearchID:          savedSearchID,
				SnapshotType:           snapshotType,
				WorkerLockID:           &workerID,
				WorkerLockExpiresAt:    &expiration,
				LastKnownStateBlobPath: nil,
			}
			if existing != nil {
				newState.LastKnownStateBlobPath = existing.LastKnownStateBlobPath
			}

			return spanner.InsertOrUpdateStruct(savedSearchStateTableName, newState)
		})

	if err != nil {
		return false, err
	}

	return true, nil
}

// ReleaseSavedSearchStateWorkerLock releases the worker lock for the given saved search and snapshot type.
// The caller must own the lock. If not, ErrLockNotOwned is returned.
func (c *Client) ReleaseSavedSearchStateWorkerLock(
	ctx context.Context,
	savedSearchID string,
	snapshotType SavedSearchSnapshotType,
	workerID string) error {
	mutator := newEntityMutator[savedSearchStateMapper, SavedSearchState](c)
	key := savedSearchStateKey{SavedSearchID: savedSearchID, SnapshotType: snapshotType}

	return mutator.readInspectMutate(ctx, key,
		func(_ context.Context, existing *SavedSearchState) (*spanner.Mutation, error) {
			// If row is gone, nothing to release
			if existing == nil {
				return nil, nil
			}

			// Verify the caller owns this lock
			if existing.WorkerLockID == nil || *existing.WorkerLockID != workerID {
				return nil, ErrLockNotOwned
			}

			newState := SavedSearchState{
				SavedSearchID: savedSearchID,
				SnapshotType:  snapshotType,
				// Release the lock
				WorkerLockID:           nil,
				WorkerLockExpiresAt:    nil,
				LastKnownStateBlobPath: nil,
			}

			// Preserve the existing blob path
			newState.LastKnownStateBlobPath = existing.LastKnownStateBlobPath

			return spanner.InsertOrUpdateStruct(savedSearchStateTableName, newState)
		})
}

// GetSavedSearchState retrieves the SavedSearchState for the given saved search and snapshot type.
// If no such row exists, ErrQueryReturnedNoResults is returned.
func (c *Client) GetSavedSearchState(
	ctx context.Context,
	savedSearchID string,
	snapshotType SavedSearchSnapshotType) (*SavedSearchState, error) {
	r := newEntityReader[savedSearchStateMapper, SavedSearchState, savedSearchStateKey](c)
	key := savedSearchStateKey{SavedSearchID: savedSearchID, SnapshotType: snapshotType}

	return r.readRowByKey(ctx, key)
}

// UpdateSavedSearchStateLastKnownStateBlobPath updates the LastKnownStateBlobPath
// for the given saved search and snapshot type.
// The row must already exist. Else, ErrQueryReturnedNoResults is returned.
func (c *Client) UpdateSavedSearchStateLastKnownStateBlobPath(
	ctx context.Context,
	savedSearchID string,
	snapshotType SavedSearchSnapshotType,
	blobPath string) error {
	mutator := newEntityMutator[savedSearchStateMapper, SavedSearchState](c)
	key := savedSearchStateKey{SavedSearchID: savedSearchID, SnapshotType: snapshotType}

	return mutator.readInspectMutate(ctx, key,
		func(_ context.Context, existing *SavedSearchState) (*spanner.Mutation, error) {
			if existing == nil {
				return nil, ErrQueryReturnedNoResults
			}
			// Update existing row
			existing.LastKnownStateBlobPath = &blobPath

			return spanner.UpdateStruct(savedSearchStateTableName, *existing)
		})
}
