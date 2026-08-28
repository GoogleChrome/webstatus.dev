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
	"time"

	"cloud.google.com/go/spanner"
)

var (
	// ErrLockExpired indicates that the worker's lock lease duration has elapsed.
	ErrLockExpired = errors.New("worker lock lease has expired")
)

// LeasedLockState provides unified validation and acquisition semantics for distributed worker leases in Spanner.
type LeasedLockState struct {
	WorkerID  spanner.NullString
	ExpiresAt spanner.NullTime
}

// NewLeasedLockState creates a LeasedLockState from spanner nullable types.
func NewLeasedLockState(workerID spanner.NullString, expiresAt spanner.NullTime) LeasedLockState {
	return LeasedLockState{
		WorkerID:  workerID,
		ExpiresAt: expiresAt,
	}
}

// CanAcquire checks if callerWorkerID can acquire or re-acquire (extend) the lock at time 'now'.
// A lock is acquirable if:
// 1. It is expired (ExpiresAt <= now), OR
// 2. It has no assigned owner (!WorkerID.Valid), OR
// 3. It is already held by the same worker (re-entrant lease renewal).
func (l LeasedLockState) CanAcquire(callerWorkerID string, now time.Time) error {
	isActive := l.ExpiresAt.Valid && l.ExpiresAt.Time.After(now)
	isOtherWorker := l.WorkerID.Valid && l.WorkerID.StringVal != callerWorkerID

	if isActive && isOtherWorker {
		return ErrAlreadyLocked
	}

	return nil
}

// ValidateOwnership verifies worker fencing: the caller must actively hold an unexpired lease.
// Both completion recording (e.g. RecordDeliverySuccess) and lock release (e.g. ReleaseDeliveryLock)
// MUST use this exact check to prevent fencing token violations.
func (l LeasedLockState) ValidateOwnership(callerWorkerID string, now time.Time) error {
	if !l.ExpiresAt.Valid || !l.ExpiresAt.Time.After(now) {
		return ErrLockExpired
	}

	if !l.WorkerID.Valid || l.WorkerID.StringVal != callerWorkerID {
		return ErrLockNotOwned
	}

	return nil
}
