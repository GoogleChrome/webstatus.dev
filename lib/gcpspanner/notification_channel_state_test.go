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

	"github.com/google/go-cmp/cmp"
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
	channelID, err := spannerClient.CreateNotificationChannel(ctx, createReq)
	if err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}

	t.Run("Create and Get", func(t *testing.T) {
		state := &NotificationChannelState{
			ChannelID:           channelID,
			IsDisabledBySystem:  false,
			ConsecutiveFailures: 0,
		}

		err := spannerClient.UpsertNotificationChannelState(ctx, *state)
		if err != nil {
			t.Fatalf("UpsertNotificationChannelState (create) failed: %v", err)
		}

		retrieved, err := spannerClient.GetNotificationChannelState(ctx, channelID)
		if err != nil {
			t.Fatalf("GetNotificationChannelState failed: %v", err)
		}
		if diff := cmp.Diff(state, retrieved); diff != "" {
			t.Errorf("GetNotificationChannelState mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("Update", func(t *testing.T) {
		// First, ensure a known state exists.
		initialState := &NotificationChannelState{
			ChannelID:           channelID,
			IsDisabledBySystem:  false,
			ConsecutiveFailures: 1,
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
		if diff := cmp.Diff(updatedState, retrieved); diff != "" {
			t.Errorf("GetNotificationChannelState after update mismatch (-want +got):\n%s", diff)
		}
	})
}
