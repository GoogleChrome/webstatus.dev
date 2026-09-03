// Copyright 2026 Google LLC
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
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
)

func TestLeasedLockState_CanAcquire(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	future := now.Add(30 * time.Second)
	past := now.Add(-10 * time.Second)

	testCases := []struct {
		name        string
		state       LeasedLockState
		caller      string
		expectedErr error
	}{
		{
			name: "Unassigned Lock (NULL worker and NULL expiry) Acquirable",
			state: LeasedLockState{
				WorkerID:  spanner.NullString{StringVal: "", Valid: false},
				ExpiresAt: spanner.NullTime{Time: time.Time{}, Valid: false},
			},
			caller:      "worker-1",
			expectedErr: nil,
		},
		{
			name: "Expired Lock Held by Another Worker Acquirable",
			state: LeasedLockState{
				WorkerID:  spanner.NullString{StringVal: "worker-2", Valid: true},
				ExpiresAt: spanner.NullTime{Time: past, Valid: true},
			},
			caller:      "worker-1",
			expectedErr: nil,
		},
		{
			name: "Active Lock Held by Same Worker Re-acquirable (Lease Extension)",
			state: LeasedLockState{
				WorkerID:  spanner.NullString{StringVal: "worker-1", Valid: true},
				ExpiresAt: spanner.NullTime{Time: future, Valid: true},
			},
			caller:      "worker-1",
			expectedErr: nil,
		},
		{
			name: "Active Lock Held by Another Worker Rejects Acquisition",
			state: LeasedLockState{
				WorkerID:  spanner.NullString{StringVal: "worker-2", Valid: true},
				ExpiresAt: spanner.NullTime{Time: future, Valid: true},
			},
			caller:      "worker-1",
			expectedErr: ErrAlreadyLocked,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.state.CanAcquire(tc.caller, now)
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("CanAcquire() err = %v, want %v", err, tc.expectedErr)
			}
		})
	}
}

func TestLeasedLockState_ValidateOwnership(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	future := now.Add(30 * time.Second)
	past := now.Add(-10 * time.Second)

	testCases := []struct {
		name        string
		state       LeasedLockState
		caller      string
		expectedErr error
	}{
		{
			name: "Valid Lease Owned by Caller Passes Validation",
			state: LeasedLockState{
				WorkerID:  spanner.NullString{StringVal: "worker-1", Valid: true},
				ExpiresAt: spanner.NullTime{Time: future, Valid: true},
			},
			caller:      "worker-1",
			expectedErr: nil,
		},
		{
			name: "Expired Lease Rejects with ErrLockExpired",
			state: LeasedLockState{
				WorkerID:  spanner.NullString{StringVal: "worker-1", Valid: true},
				ExpiresAt: spanner.NullTime{Time: past, Valid: true},
			},
			caller:      "worker-1",
			expectedErr: ErrLockExpired,
		},
		{
			name: "NULL Expiry Rejects with ErrLockExpired",
			state: LeasedLockState{
				WorkerID:  spanner.NullString{StringVal: "worker-1", Valid: true},
				ExpiresAt: spanner.NullTime{Time: time.Time{}, Valid: false},
			},
			caller:      "worker-1",
			expectedErr: ErrLockExpired,
		},
		{
			name: "Active Lease Owned by Other Worker Rejects with ErrLockNotOwned",
			state: LeasedLockState{
				WorkerID:  spanner.NullString{StringVal: "worker-2", Valid: true},
				ExpiresAt: spanner.NullTime{Time: future, Valid: true},
			},
			caller:      "worker-1",
			expectedErr: ErrLockNotOwned,
		},
		{
			name: "Active Lease with NULL WorkerID Rejects with ErrLockNotOwned",
			state: LeasedLockState{
				WorkerID:  spanner.NullString{StringVal: "", Valid: false},
				ExpiresAt: spanner.NullTime{Time: future, Valid: true},
			},
			caller:      "worker-1",
			expectedErr: ErrLockNotOwned,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.state.ValidateOwnership(tc.caller, now)
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("ValidateOwnership() err = %v, want %v", err, tc.expectedErr)
			}
		})
	}
}
