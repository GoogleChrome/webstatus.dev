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

	"github.com/google/go-cmp/cmp"
)

const worker1 = "worker-1"
const worker2 = "worker-2"
const snapshotType = "compat-stats"

func noopSavedSearchStateHelper(t *testing.T, _ string) {
	t.Helper()
}

func createSavedSearchForSavedSearchStateTests(ctx context.Context, t *testing.T) string {
	t.Helper()
	id, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "test search",
		Query:       "group:test",
		OwnerUserID: "owner-1",
		Description: nil,
	})
	if err != nil {
		t.Fatalf("CreateNewUserSavedSearch() returned unexpected error: %v", err)
	}

	return *id
}

// Asserts for TestTryAcquireSavedSearchStateWorkerLock.
func assertAbleToAcquireLock(ctx context.Context, t *testing.T, savedSearchID string,
	snapshotType SavedSearchSnapshotType, workerID string,
	initialTime time.Time, ttl time.Duration) {
	state, err := spannerClient.GetSavedSearchState(ctx, savedSearchID, snapshotType)
	if err != nil {
		t.Fatalf("GetSavedSearchState() got unexpected error = %v", err)
	}
	if state == nil {
		t.Fatal("GetSavedSearchState() state was nil")
	}
	if *state.WorkerLockID != workerID {
		t.Errorf("WorkerLockID mismatch: got %s, want %s", *state.WorkerLockID, workerID)
	}
	expectedExpiration := initialTime.Add(ttl)
	if !expectedExpiration.Equal(*state.WorkerLockExpiresAt) {
		t.Errorf("WorkerLockExpiresAt mismatch: got %v, want %v", *state.WorkerLockExpiresAt,
			expectedExpiration)
	}
}

func TestTryAcquireSavedSearchStateWorkerLock(t *testing.T) {
	ctx := context.Background()

	// Fixed time for deterministic tests
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	ttl := 10 * time.Minute

	testCases := []struct {
		name              string
		setup             func(t *testing.T, savedSearchID string)
		snapshotType      SavedSearchSnapshotType
		workerID          string
		ttl               time.Duration
		expectedSuccess   bool
		expectedErr       error
		assertAfterAction func(t *testing.T, savedSearchID string)
	}{
		{
			name:            "acquire lock when none exists",
			snapshotType:    snapshotType,
			workerID:        worker1,
			ttl:             ttl,
			expectedSuccess: true,
			expectedErr:     nil,
			setup:           noopSavedSearchStateHelper,
			assertAfterAction: func(t *testing.T, savedSearchID string) {
				t.Helper()
				assertAbleToAcquireLock(ctx, t, savedSearchID, snapshotType, worker1, fixedTime, ttl)
			},
		},
		{
			name:         "re-acquire existing lock",
			snapshotType: snapshotType,
			workerID:     worker1,
			ttl:          20 * time.Minute, // Extend TTL
			setup: func(t *testing.T, savedSearchID string) {
				t.Helper()
				// Pre-acquire the lock
				_, err := spannerClient.TryAcquireSavedSearchStateWorkerLock(ctx, savedSearchID, snapshotType,
					worker1, ttl)
				if err != nil {
					t.Fatalf("setup: TryAcquireSavedSearchStateWorkerLock failed: %v", err)
				}
			},
			expectedSuccess: true,
			expectedErr:     nil,
			assertAfterAction: func(t *testing.T, savedSearchID string) {
				t.Helper()
				assertAbleToAcquireLock(ctx, t, savedSearchID, snapshotType, worker1, fixedTime,
					// New TTL
					20*time.Minute)
			},
		},
		{
			name:         "fail to acquire lock held by another active worker",
			snapshotType: snapshotType,
			workerID:     worker2,
			ttl:          ttl,
			setup: func(t *testing.T, savedSearchID string) {
				t.Helper()
				// worker1 acquires the lock
				_, err := spannerClient.TryAcquireSavedSearchStateWorkerLock(ctx, savedSearchID, snapshotType, worker1,
					ttl)
				if err != nil {
					t.Fatalf("setup: TryAcquireSavedSearchStateWorkerLock failed: %v", err)
				}
			},
			expectedSuccess: false,
			expectedErr:     ErrAlreadyLocked,
			assertAfterAction: func(t *testing.T, savedSearchID string) {
				t.Helper()
				// State should be unchanged
				state, err := spannerClient.GetSavedSearchState(ctx, savedSearchID, snapshotType)
				if err != nil {
					t.Fatalf("GetSavedSearchState() got unexpected error = %v", err)
				}
				if state == nil {
					t.Fatal("GetSavedSearchState() state was nil")
				}
				if *state.WorkerLockID != worker1 {
					t.Errorf("WorkerLockID mismatch: got %s, want %s", *state.WorkerLockID, worker1)
				}
			},
		},
		{
			name:         "acquire lock held by another worker but expired",
			snapshotType: snapshotType,
			workerID:     worker2,
			ttl:          ttl,
			setup: func(t *testing.T, savedSearchID string) {
				t.Helper()
				// worker1 acquires the lock
				_, err := spannerClient.TryAcquireSavedSearchStateWorkerLock(ctx, savedSearchID, snapshotType, worker1,
					ttl)
				if err != nil {
					t.Fatalf("setup: TryAcquireSavedSearchStateWorkerLock failed: %v", err)
				}
				// Time moves forward, making the lock expire
				spannerClient.setTimeNowForTesting(func() time.Time { return fixedTime.Add(15 * time.Minute) })
			},
			expectedSuccess: true,
			expectedErr:     nil,
			assertAfterAction: func(t *testing.T, savedSearchID string) {
				t.Helper()
				assertAbleToAcquireLock(ctx, t, savedSearchID, snapshotType, worker2,
					// Current time is 15 minutes later. The new lock should reflect that.
					fixedTime.Add(15*time.Minute),
					// TTL is the same.
					ttl)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restartDatabaseContainer(t)
			savedSearchID := createSavedSearchForSavedSearchStateTests(ctx, t)
			spannerClient.setTimeNowForTesting(func() time.Time { return fixedTime })

			tc.setup(t, savedSearchID)

			success, err := spannerClient.TryAcquireSavedSearchStateWorkerLock(
				ctx, savedSearchID, tc.snapshotType, tc.workerID, tc.ttl)

			if success != tc.expectedSuccess {
				t.Errorf("TryAcquireSavedSearchStateWorkerLock() success = %v, want %v", success, tc.expectedSuccess)
			}
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("TryAcquireSavedSearchStateWorkerLock() error = %v, want %v", err, tc.expectedErr)
			}

			tc.assertAfterAction(t, savedSearchID)
		})
	}
}

func TestReleaseSavedSearchStateWorkerLock(t *testing.T) {
	ctx := context.Background()

	ttl := 10 * time.Minute

	testCases := []struct {
		name               string
		setup              func(t *testing.T, savedSearchID string)
		otherSavedSearchID *string
		snapshotType       SavedSearchSnapshotType
		workerID           string
		expectedErr        error
		assertAfterAction  func(t *testing.T, savedSearchID string)
	}{
		{
			name:         "successfully release an owned lock",
			snapshotType: snapshotType,
			workerID:     worker1,
			setup: func(t *testing.T, savedSearchID string) {
				t.Helper()
				_, err := spannerClient.TryAcquireSavedSearchStateWorkerLock(ctx, savedSearchID, snapshotType, worker1,
					ttl)
				if err != nil {
					t.Fatalf("setup: TryAcquireSavedSearchStateWorkerLock failed: %v", err)
				}
			},
			expectedErr:        nil,
			otherSavedSearchID: nil,
			assertAfterAction: func(t *testing.T, savedSearchID string) {
				t.Helper()
				state, err := spannerClient.GetSavedSearchState(ctx, savedSearchID, snapshotType)
				if err != nil {
					t.Fatalf("GetSavedSearchState() got unexpected error = %v", err)
				}
				if state == nil {
					t.Fatal("GetSavedSearchState() state was nil")
				}
				if state.WorkerLockID != nil {
					t.Errorf("WorkerLockID should be nil, got %s", *state.WorkerLockID)
				}
				if state.WorkerLockExpiresAt != nil {
					t.Errorf("WorkerLockExpiresAt should be nil, got %v", *state.WorkerLockExpiresAt)
				}
			},
		},
		{
			name:         "fail to release a lock owned by another worker",
			snapshotType: snapshotType,
			workerID:     worker2,
			setup: func(t *testing.T, savedSearchID string) {
				t.Helper()
				_, err := spannerClient.TryAcquireSavedSearchStateWorkerLock(ctx, savedSearchID, snapshotType, worker1,
					ttl)
				if err != nil {
					t.Fatalf("setup: TryAcquireSavedSearchStateWorkerLock failed: %v", err)
				}
			},
			expectedErr:        ErrLockNotOwned,
			otherSavedSearchID: nil,
			assertAfterAction: func(t *testing.T, savedSearchID string) {
				t.Helper()
				// State should be unchanged
				state, err := spannerClient.GetSavedSearchState(ctx, savedSearchID, snapshotType)
				if err != nil {
					t.Fatalf("GetSavedSearchState() got unexpected error = %v", err)
				}
				if state == nil {
					t.Fatal("GetSavedSearchState() state was nil")
				}
				if *state.WorkerLockID != worker1 {
					t.Errorf("WorkerLockID mismatch: got %s, want %s", *state.WorkerLockID, worker1)
				}
			},
		},
		{
			name:               "attempt to release a lock that does not exist",
			otherSavedSearchID: valuePtr("non-existent-search"),
			snapshotType:       snapshotType,
			workerID:           worker1,
			expectedErr:        nil, // Should be a no-op
			assertAfterAction:  noopSavedSearchStateHelper,
			setup:              noopSavedSearchStateHelper,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restartDatabaseContainer(t)
			savedSearchID := createSavedSearchForSavedSearchStateTests(ctx, t)
			tc.setup(t, savedSearchID)

			if tc.otherSavedSearchID != nil {
				savedSearchID = *tc.otherSavedSearchID
			}

			err := spannerClient.ReleaseSavedSearchStateWorkerLock(ctx, savedSearchID, tc.snapshotType, tc.workerID)

			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("ReleaseSavedSearchStateWorkerLock() error = %v, want %v", err, tc.expectedErr)
			}

			tc.assertAfterAction(t, savedSearchID)
		})
	}
}

func TestGetAndUpdateSavedSearchState(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	// Fixed time for deterministic tests
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	spannerClient.setTimeNowForTesting(func() time.Time { return fixedTime })

	savedSearchID := createSavedSearchForSavedSearchStateTests(ctx, t)

	workerID := "worker-1"
	ttl := 10 * time.Minute
	initialBlobPath := "path/to/blob/1"
	updatedBlobPath := "path/to/blob/2"

	// Setup: Create an initial state
	_, err := spannerClient.TryAcquireSavedSearchStateWorkerLock(ctx, savedSearchID, snapshotType, workerID, ttl)
	if err != nil {
		t.Fatalf("setup: TryAcquireSavedSearchStateWorkerLock failed: %v", err)
	}
	err = spannerClient.UpdateSavedSearchStateLastKnownStateBlobPath(ctx, savedSearchID, snapshotType, initialBlobPath)
	if err != nil {
		t.Fatalf("setup: UpdateSavedSearchStateLastKnownStateBlobPath failed: %v", err)
	}

	t.Run("GetSavedSearchState - success", func(t *testing.T) {
		state, err := spannerClient.GetSavedSearchState(ctx, savedSearchID, snapshotType)
		if err != nil {
			t.Fatalf("GetSavedSearchState() got unexpected error = %v", err)
		}
		if state == nil {
			t.Fatal("GetSavedSearchState() state was nil")
		}
		if state.SavedSearchID != savedSearchID {
			t.Errorf("SavedSearchID mismatch: got %s, want %s", state.SavedSearchID, savedSearchID)
		}
		if state.SnapshotType != snapshotType {
			t.Errorf("SnapshotType mismatch: got %s, want %s", state.SnapshotType, snapshotType)
		}
		if *state.WorkerLockID != workerID {
			t.Errorf("WorkerLockID mismatch: got %s, want %s", *state.WorkerLockID, workerID)
		}
		if *state.LastKnownStateBlobPath != initialBlobPath {
			t.Errorf("LastKnownStateBlobPath mismatch: got %s, want %s", *state.LastKnownStateBlobPath, initialBlobPath)
		}
	})

	t.Run("GetSavedSearchState - not found", func(t *testing.T) {
		_, err := spannerClient.GetSavedSearchState(ctx, "non-existent", snapshotType)
		if !errors.Is(err, ErrQueryReturnedNoResults) {
			t.Errorf("GetSavedSearchState() with non-existent key returned error = %v, want %v", err,
				ErrQueryReturnedNoResults)
		}
	})

	t.Run("UpdateSavedSearchStateLastKnownStateBlobPath - success", func(t *testing.T) {
		err := spannerClient.UpdateSavedSearchStateLastKnownStateBlobPath(ctx, savedSearchID,
			snapshotType, updatedBlobPath)
		if err != nil {
			t.Fatalf("UpdateSavedSearchStateLastKnownStateBlobPath() returned an error: %v", err)
		}

		// Verify update
		state, err := spannerClient.GetSavedSearchState(ctx, savedSearchID, snapshotType)
		if err != nil {
			t.Fatalf("GetSavedSearchState() after update returned an error: %v", err)
		}
		if state == nil {
			t.Fatal("GetSavedSearchState() after update state was nil")
		}
		expectedExpiration := fixedTime.Add(ttl)
		expectedState := SavedSearchState{
			SavedSearchID:          savedSearchID,
			SnapshotType:           snapshotType,
			WorkerLockID:           valuePtr(workerID),
			LastKnownStateBlobPath: valuePtr(updatedBlobPath),
			WorkerLockExpiresAt:    &expectedExpiration,
		}
		if diff := cmp.Diff(expectedState, *state); diff != "" {
			t.Errorf("GetSavedSearchState() after update mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("UpdateSavedSearchStateLastKnownStateBlobPath - not found", func(t *testing.T) {
		err := spannerClient.UpdateSavedSearchStateLastKnownStateBlobPath(ctx, "non-existent",
			snapshotType, updatedBlobPath)
		if !errors.Is(err, ErrQueryReturnedNoResults) {
			t.Errorf(
				"UpdateSavedSearchStateLastKnownStateBlobPath() with non-existent key returned error = %v, want %v",
				err, ErrQueryReturnedNoResults)
		}
	})
}
