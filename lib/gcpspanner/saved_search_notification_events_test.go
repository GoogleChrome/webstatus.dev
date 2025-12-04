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
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

// createSavedSearchForNotificationTests creates a saved search for testing notification events.
func createSavedSearchForNotificationTests(ctx context.Context, t *testing.T) string {
	t.Helper()
	id, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "test search for notifications",
		Query:       "group:notification-test",
		OwnerUserID: "owner-notification-1",
		Description: nil,
	})
	if err != nil {
		t.Fatalf("CreateNewUserSavedSearch() returned unexpected error: %v", err)
	}

	return *id
}

// setupLockAndInitialState acquires a lock and sets an initial state for testing.
func setupLockAndInitialState(
	ctx context.Context,
	t *testing.T,
	savedSearchID, snapshotType, workerID, initialBlobPath string,
	ttl time.Duration,
	fixedTime time.Time,
) {
	t.Helper()
	spannerClient.setTimeNowForTesting(func() time.Time { return fixedTime })

	_, err := spannerClient.TryAcquireSavedSearchStateWorkerLock(ctx, savedSearchID,
		SavedSearchSnapshotType(snapshotType), workerID, ttl)
	if err != nil {
		t.Fatalf("setup: TryAcquireSavedSearchStateWorkerLock failed: %v", err)
	}
	err = spannerClient.UpdateSavedSearchStateLastKnownStateBlobPath(ctx, savedSearchID,
		SavedSearchSnapshotType(snapshotType), initialBlobPath)
	if err != nil {
		t.Fatalf("setup: UpdateSavedSearchStateLastKnownStateBlobPath failed: %v", err)
	}
}

// assertPublishedEvent checks if the published event matches the expected event.
func assertPublishedEvent(
	ctx context.Context,
	t *testing.T,
	expectedEvent SavedSearchNotificationEvent,
) {
	t.Helper()
	retrievedEvent, err := spannerClient.GetSavedSearchNotificationEvent(ctx, expectedEvent.ID)
	if err != nil {
		t.Fatalf("GetSavedSearchNotificationEvent() failed: %v", err)
	}
	// Ignore timestamp as it's set by the server.
	if diff := cmp.Diff(expectedEvent, *retrievedEvent, cmp.FilterPath(func(p cmp.Path) bool {
		return p.String() == "Timestamp"
	}, cmp.Ignore())); diff != "" {
		t.Errorf("retrieved event mismatch (-want +got):\n%s", diff)
	}
}

// assertStateAndLockKept checks if the state was updated and the lock was preserved.
func assertStateAndLockKept(
	ctx context.Context,
	t *testing.T,
	savedSearchID, snapshotType, newStatePath, workerID string,
	expectedExpiration time.Time,
) {
	t.Helper()
	state, err := spannerClient.GetSavedSearchState(ctx, savedSearchID, SavedSearchSnapshotType(snapshotType))
	if err != nil {
		t.Fatalf("GetSavedSearchState() after publish failed: %v", err)
	}
	if state.LastKnownStateBlobPath == nil || *state.LastKnownStateBlobPath != newStatePath {
		t.Errorf("LastKnownStateBlobPath mismatch: got %v, want %s", state.LastKnownStateBlobPath, newStatePath)
	}
	if state.WorkerLockID == nil || *state.WorkerLockID != workerID {
		t.Errorf("WorkerLockID should have been kept, but mismatch: got %v, want %s", state.WorkerLockID, workerID)
	}
	if state.WorkerLockExpiresAt == nil || !state.WorkerLockExpiresAt.Equal(expectedExpiration) {
		t.Errorf("WorkerLockExpiresAt should have been kept, but mismatch: got %v, want %v",
			state.WorkerLockExpiresAt, expectedExpiration)
	}
}

// assertStateUnchanged checks if the state of SavedSearchState remained unchanged.
func assertStateUnchanged(
	ctx context.Context,
	t *testing.T,
	savedSearchID, snapshotType, expectedBlobPath, expectedWorkerID string,
	expectedExpiration time.Time,
) {
	t.Helper()
	state, err := spannerClient.GetSavedSearchState(ctx, savedSearchID, SavedSearchSnapshotType(snapshotType))
	if err != nil {
		t.Fatalf("GetSavedSearchState() failed: %v", err)
	}
	if state.LastKnownStateBlobPath == nil || *state.LastKnownStateBlobPath != expectedBlobPath {
		t.Errorf("LastKnownStateBlobPath changed unexpectedly: got %v, want %s",
			state.LastKnownStateBlobPath, expectedBlobPath)
	}
	if state.WorkerLockID == nil || *state.WorkerLockID != expectedWorkerID {
		t.Errorf("WorkerLockID changed unexpectedly: got %v, want %s", state.WorkerLockID, expectedWorkerID)
	}
	if state.WorkerLockExpiresAt == nil || !state.WorkerLockExpiresAt.Equal(expectedExpiration) {
		t.Errorf("WorkerLockExpiresAt changed unexpectedly: got %v, want %v",
			state.WorkerLockExpiresAt, expectedExpiration)
	}
}

func TestPublishSavedSearchNotificationEvent(t *testing.T) {
	ctx := context.Background()

	// Shared test data
	snapshotType := "compat-stats"
	workerID := "worker-1"
	ttl := 10 * time.Minute
	initialBlobPath := "path/initial"
	newStatePath := "path/new"

	// Fixed time for deterministic tests
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	initialExpiration := fixedTime.Add(ttl)

	testCases := []struct {
		name              string
		setup             func(t *testing.T, savedSearchID string)
		event             func(savedSearchID string) SavedSearchNotificationCreateRequest
		newStatePath      string
		workerID          string
		expectedErr       error
		assertAfterAction func(t *testing.T, savedSearchID string, eventID *string)
	}{
		{
			name: "success - publish event and update state, keeping lock",
			setup: func(t *testing.T, savedSearchID string) {
				t.Helper()
				setupLockAndInitialState(ctx, t, savedSearchID, snapshotType, workerID, initialBlobPath, ttl, fixedTime)
			},
			event: func(savedSearchID string) SavedSearchNotificationCreateRequest {
				return SavedSearchNotificationCreateRequest{
					SavedSearchID: savedSearchID,
					SnapshotType:  SavedSearchSnapshotType(snapshotType),
					Timestamp:     spanner.CommitTimestamp,
					EventType:     "IMMEDIATE_DIFF",
					Reason:        "DATA_UPDATED",
					BlobPath:      newStatePath,
					DataVersion:   "v1",
					Summary: spanner.NullJSON{
						Value: nil,
						Valid: false,
					},
					DiffKind: nil,
				}
			},
			newStatePath: newStatePath,
			workerID:     workerID,
			expectedErr:  nil,
			assertAfterAction: func(t *testing.T, savedSearchID string, eventID *string) {
				t.Helper()
				assertPublishedEvent(ctx, t, SavedSearchNotificationEvent{
					ID:            *eventID,
					SavedSearchID: savedSearchID,
					SnapshotType:  SavedSearchSnapshotType(snapshotType),
					Timestamp:     spanner.CommitTimestamp,
					EventType:     "IMMEDIATE_DIFF",
					Reason:        "DATA_UPDATED",
					BlobPath:      newStatePath,
					DataVersion:   "v1",
					Summary: spanner.NullJSON{
						Value: nil,
						Valid: false,
					},
					DiffKind: nil,
				})
				assertStateAndLockKept(ctx, t, savedSearchID, snapshotType, newStatePath, workerID, initialExpiration)
			},
		},
		{
			name: "fail - lock not owned by worker",
			setup: func(t *testing.T, savedSearchID string) {
				t.Helper()
				setupLockAndInitialState(ctx, t, savedSearchID, snapshotType, workerID, initialBlobPath, ttl, fixedTime)
			},
			event: func(savedSearchID string) SavedSearchNotificationCreateRequest {
				return SavedSearchNotificationCreateRequest{
					SavedSearchID: savedSearchID,
					SnapshotType:  SavedSearchSnapshotType(snapshotType),
					Timestamp:     spanner.CommitTimestamp,
					EventType:     "IMMEDIATE_DIFF",
					Reason:        "DATA_UPDATED",
					BlobPath:      newStatePath,
					DataVersion:   "v1",
					Summary: spanner.NullJSON{
						Value: nil,
						Valid: false,
					},
					DiffKind: nil,
				}
			},
			newStatePath: newStatePath,
			workerID:     "wrong-worker-id",
			expectedErr:  ErrAlreadyLocked,
			assertAfterAction: func(t *testing.T, savedSearchID string, _ *string) {
				t.Helper()
				assertStateUnchanged(ctx, t, savedSearchID, snapshotType, initialBlobPath, workerID, initialExpiration)
				// No event should be published
				_, err := spannerClient.GetSavedSearchNotificationEvent(ctx, uuid.NewString() /* arbitrary ID */)
				if !errors.Is(err, ErrQueryReturnedNoResults) {
					t.Errorf("expected no event to be published, but got %v", err)
				}
			},
		},
		{
			name:  "fail - saved search state does not exist",
			setup: noopSavedSearchStateHelper,
			event: func(_ string) SavedSearchNotificationCreateRequest {
				return SavedSearchNotificationCreateRequest{
					SavedSearchID: "non-existent-search",
					SnapshotType:  SavedSearchSnapshotType(snapshotType),
					Timestamp:     spanner.CommitTimestamp,
					EventType:     "IMMEDIATE_DIFF",
					Reason:        "DATA_UPDATED",
					BlobPath:      newStatePath,
					DataVersion:   "v1",
					Summary: spanner.NullJSON{
						Value: nil,
						Valid: false,
					},
					DiffKind: nil,
				}
			},
			newStatePath: newStatePath,
			workerID:     workerID,
			expectedErr:  ErrQueryReturnedNoResults,
			assertAfterAction: func(t *testing.T, _ string, _ *string) {
				t.Helper()
				// No event should be published
				_, err := spannerClient.GetSavedSearchNotificationEvent(ctx, uuid.NewString() /* arbitrary ID */)
				if !errors.Is(err, ErrQueryReturnedNoResults) {
					t.Errorf("expected no event to be published, but got %v", err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restartDatabaseContainer(t)
			savedSearchID := createSavedSearchForNotificationTests(ctx, t)

			tc.setup(t, savedSearchID)
			event := tc.event(savedSearchID)

			eventID, err := spannerClient.PublishSavedSearchNotificationEvent(ctx, event, tc.newStatePath, tc.workerID)

			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("PublishSavedSearchNotificationEvent() error = %v, want %v", err, tc.expectedErr)
			}

			tc.assertAfterAction(t, savedSearchID, eventID)
		})
	}
}

func TestGetSavedSearchNotificationEvent_NotFound(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	// Test case for when the event is not found
	_, err := spannerClient.GetSavedSearchNotificationEvent(ctx, "non-existent-event-id")
	if !errors.Is(err, ErrQueryReturnedNoResults) {
		t.Errorf("expected ErrQueryReturnedNoResults, got %v", err)
	}
}
