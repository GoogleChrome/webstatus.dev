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
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
)

func TestNotificationChannelStateOperations(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	// We need a channel to associate the state with.
	userID := uuid.NewString()
	createReq := CreateNotificationChannelRequest{
		UserID:      userID,
		Name:        "Test Channel",
		Type:        "EMAIL",
		EmailConfig: &EmailConfig{Address: "test@example.com", IsVerified: true, VerificationToken: nil},
	}
	channelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, createReq)
	if err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}
	channelID := *channelIDPtr

	t.Run("Create and Get", func(t *testing.T) {
		tstate := &NotificationChannelState{
			ChannelID:           channelID,
			IsDisabledBySystem:  false,
			ConsecutiveFailures: 0,
			CreatedAt:           spanner.CommitTimestamp,
			UpdatedAt:           spanner.CommitTimestamp,
		}

		err := spannerClient.UpsertNotificationChannelState(ctx, *tstate)
		if err != nil {
			t.Fatalf("UpsertNotificationChannelState (create) failed: %v", err)
		}

		retrieved, err := spannerClient.GetNotificationChannelState(ctx, channelID)
		if err != nil {
			t.Fatalf("GetNotificationChannelState failed: %v", err)
		}
		if diff := cmp.Diff(tstate, retrieved,
			cmpopts.IgnoreFields(NotificationChannelState{
				ChannelID:           "",
				IsDisabledBySystem:  false,
				ConsecutiveFailures: 0,
				CreatedAt:           spanner.CommitTimestamp,
				UpdatedAt:           spanner.CommitTimestamp,
			}, "CreatedAt", "UpdatedAt")); diff != "" {
			t.Errorf("GetNotificationChannelState mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("Update", func(t *testing.T) {
		// First, ensure a known state exists.
		initialState := &NotificationChannelState{
			ChannelID:           channelID,
			IsDisabledBySystem:  false,
			ConsecutiveFailures: 1,
			CreatedAt:           spanner.CommitTimestamp,
			UpdatedAt:           spanner.CommitTimestamp,
		}
		err := spannerClient.UpsertNotificationChannelState(ctx, *initialState)
		if err != nil {
			t.Fatalf("pre-test UpsertNotificationChannelState failed: %v", err)
		}

		// Now, update it.
		updatedState := &NotificationChannelState{
			ChannelID:           channelID,
			IsDisabledBySystem:  true,
			ConsecutiveFailures: 5,
			CreatedAt:           spanner.CommitTimestamp,
			UpdatedAt:           spanner.CommitTimestamp,
		}
		err = spannerClient.UpsertNotificationChannelState(ctx, *updatedState)
		if err != nil {
			t.Fatalf("UpsertNotificationChannelState (update) failed: %v", err)
		}

		// Verify the update.
		retrieved, err := spannerClient.GetNotificationChannelState(ctx, channelID)
		if err != nil {
			t.Fatalf("GetNotificationChannelState after update failed: %v", err)
		}
		if diff := cmp.Diff(updatedState, retrieved,
			cmpopts.IgnoreFields(NotificationChannelState{
				ChannelID:           "",
				IsDisabledBySystem:  false,
				ConsecutiveFailures: 0,
				CreatedAt:           spanner.CommitTimestamp,
				UpdatedAt:           spanner.CommitTimestamp,
			}, "CreatedAt", "UpdatedAt")); diff != "" {
			t.Errorf("GetNotificationChannelState after update mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("RecordNotificationChannelSuccess", func(t *testing.T) {
		testRecordNotificationChannelSuccess(t, channelID)
	})

	t.Run("RecordNotificationChannelFailure", func(t *testing.T) {
		testRecordNotificationChannelFailure(t, channelID)
	})
}

func testRecordNotificationChannelSuccess(t *testing.T, channelID string) {
	ctx := t.Context()
	// First, set up a channel state with some failures.
	initialState := &NotificationChannelState{
		ChannelID:           channelID,
		IsDisabledBySystem:  true,
		ConsecutiveFailures: 3,
		CreatedAt:           spanner.CommitTimestamp,
		UpdatedAt:           spanner.CommitTimestamp,
	}
	err := spannerClient.UpsertNotificationChannelState(ctx, *initialState)
	if err != nil {
		t.Fatalf("pre-test UpsertNotificationChannelState failed: %v", err)
	}

	testTime := time.Now()
	eventID := "evt-1"
	err = spannerClient.RecordNotificationChannelSuccess(ctx, channelID, testTime, eventID)
	if err != nil {
		t.Fatalf("RecordNotificationChannelSuccess failed: %v", err)
	}

	// Verify state update.
	retrievedState, err := spannerClient.GetNotificationChannelState(ctx, channelID)
	if err != nil {
		t.Fatalf("GetNotificationChannelState after success failed: %v", err)
	}
	if retrievedState.IsDisabledBySystem != false {
		t.Errorf("expected IsDisabledBySystem to be false, got %t", retrievedState.IsDisabledBySystem)
	}
	if retrievedState.ConsecutiveFailures != 0 {
		t.Errorf("expected ConsecutiveFailures to be 0, got %d", retrievedState.ConsecutiveFailures)
	}

	// Verify delivery attempt log.
	listAttemptsReq := ListNotificationChannelDeliveryAttemptsRequest{
		ChannelID: channelID,
		PageSize:  1,
		PageToken: nil,
	}
	attempts, _, err := spannerClient.ListNotificationChannelDeliveryAttempts(ctx, listAttemptsReq)
	if err != nil {
		t.Fatalf("ListNotificationChannelDeliveryAttempts after success failed: %v", err)
	}
	if len(attempts) != 1 {
		t.Fatalf("expected 1 delivery attempt, got %d", len(attempts))
	}
	if attempts[0].Status != DeliveryAttemptStatusSuccess {
		t.Errorf("expected status SUCCESS, got %s", attempts[0].Status)
	}
	if attempts[0].AttemptDetails == nil || attempts[0].AttemptDetails.Message != "delivered" ||
		attempts[0].AttemptDetails.EventID != "evt-1" {
		t.Errorf("expected details message 'delivered', got %v", attempts[0].AttemptDetails)
	}
}

func testRecordNotificationChannelFailure(t *testing.T, channelID string) {
	ctx := t.Context()
	// Reset state for new test
	initialState := &NotificationChannelState{
		ChannelID:           channelID,
		IsDisabledBySystem:  false,
		ConsecutiveFailures: 0,
		CreatedAt:           spanner.CommitTimestamp,
		UpdatedAt:           spanner.CommitTimestamp,
	}
	err := spannerClient.UpsertNotificationChannelState(ctx, *initialState)
	if err != nil {
		t.Fatalf("pre-test UpsertNotificationChannelState failed: %v", err)
	}

	t.Run("Permanent Failure", func(t *testing.T) {
		_ = spannerClient.UpsertNotificationChannelState(ctx, *initialState) // Ensure clean state
		testTime := time.Now()
		errorMsg := "permanent error"
		eventID := "evt-124"
		err = spannerClient.RecordNotificationChannelFailure(ctx, channelID, errorMsg, testTime, true, eventID)
		if err != nil {
			t.Fatalf("RecordNotificationChannelFailure (permanent) failed: %v", err)
		}

		verifyFailureAttemptAndState(t, channelID, 1, false, errorMsg, eventID)
	})

	t.Run("Transient Failure", func(t *testing.T) {
		_ = spannerClient.UpsertNotificationChannelState(ctx, *initialState) // Ensure clean state
		testTime := time.Now()
		errorMsg := "transient error"
		eventID := "evt-125"
		err = spannerClient.RecordNotificationChannelFailure(ctx, channelID, errorMsg, testTime, false, eventID)
		if err != nil {
			t.Fatalf("RecordNotificationChannelFailure (transient) failed: %v", err)
		}

		verifyFailureAttemptAndState(t, channelID, 0, false, errorMsg, eventID)
	})
}

// verifyFailureAttemptAndState is a helper function to verify the state and delivery attempt after a failure.
func verifyFailureAttemptAndState(t *testing.T, channelID string,
	expectedFailures int64, expectedIsDisabled bool, expectedAttemptMessage string, expectedEventID string) {
	t.Helper()
	ctx := t.Context()

	// Verify state update.
	retrievedState, err := spannerClient.GetNotificationChannelState(ctx, channelID)
	if err != nil {
		t.Fatalf("GetNotificationChannelState after failure failed: %v", err)
	}
	if retrievedState.ConsecutiveFailures != expectedFailures {
		t.Errorf("expected ConsecutiveFailures to be %d, got %d", expectedFailures, retrievedState.ConsecutiveFailures)
	}
	if retrievedState.IsDisabledBySystem != expectedIsDisabled {
		t.Errorf("expected IsDisabledBySystem to be %t, got %t", expectedIsDisabled, retrievedState.IsDisabledBySystem)
	}

	// Verify delivery attempt log.
	listAttemptsReq := ListNotificationChannelDeliveryAttemptsRequest{
		ChannelID: channelID,
		PageSize:  1,
		PageToken: nil,
	}
	attempts, _, err := spannerClient.ListNotificationChannelDeliveryAttempts(ctx, listAttemptsReq)
	if err != nil {
		t.Fatalf("ListNotificationChannelDeliveryAttempts after failure failed: %v", err)
	}
	if len(attempts) != 1 {
		t.Fatalf("expected 1 delivery attempt, got %d", len(attempts))
	}
	if attempts[0].Status != DeliveryAttemptStatusFailure {
		t.Errorf("expected status FAILURE, got %s", attempts[0].Status)
	}
	if attempts[0].AttemptDetails == nil || attempts[0].AttemptDetails.Message != expectedAttemptMessage ||
		attempts[0].AttemptDetails.EventID != expectedEventID {
		t.Errorf("expected details message '%s' eventID '%s', got %v", expectedAttemptMessage, expectedEventID,
			attempts[0].AttemptDetails)
	}
}
