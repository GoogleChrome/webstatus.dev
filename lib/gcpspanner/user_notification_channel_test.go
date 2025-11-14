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
	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
)

// TestNotificationChannel is a helper struct for test assertions.
type TestNotificationChannel struct {
	ID                 string
	UserID             string
	Email              string
	IsDisabledBySystem bool
}

func getAllUserNotificationChannels(
	ctx context.Context,
	t *testing.T,
	client *Client,
	userID string,
) map[string]TestNotificationChannel {
	t.Helper()
	channels := make(map[string]TestNotificationChannel)
	stmt := spanner.Statement{
		SQL: `SELECT nc.ID, nc.UserID, nc.Name, nc.Type, nc.Config, nc.CreatedAt, nc.UpdatedAt,
						ncs.IsDisabledBySystem, ncs.ConsecutiveFailures, ncs.CreatedAt AS StateCreatedAt, ncs.UpdatedAt AS StateUpdatedAt
				FROM NotificationChannels AS nc
				LEFT JOIN NotificationChannelStates AS ncs ON nc.ID = ncs.ChannelID
				WHERE nc.UserID = @userID AND nc.Type = "email"`,
		Params: map[string]interface{}{
			"userID": userID,
		},
	}
	it := client.Single().Query(ctx, stmt)
	defer it.Stop()

	err := it.Do(func(r *spanner.Row) error {
		var spannerChannel spannerNotificationChannel
		var state NotificationChannelState
		var stateUpdatedAt spanner.NullTime

		// Explicitly scan NotificationChannel fields
		if err := r.ColumnByName("ID", &spannerChannel.ID); err != nil {
			return err
		}
		if err := r.ColumnByName("UserID", &spannerChannel.UserID); err != nil {
			return err
		}
		if err := r.ColumnByName("Name", &spannerChannel.Name); err != nil {
			return err
		}
		if err := r.ColumnByName("Type", &spannerChannel.Type); err != nil {
			return err
		}
		if err := r.ColumnByName("Config", &spannerChannel.Config); err != nil {
			return err
		}
		if err := r.ColumnByName("CreatedAt", &spannerChannel.CreatedAt); err != nil {
			return err
		}
		if err := r.ColumnByName("UpdatedAt", &spannerChannel.UpdatedAt); err != nil {
			return err
		}

		// Manually scan NotificationChannelState fields (which might be NULL from LEFT JOIN)
		var isDisabledBySystem spanner.NullBool
		var consecutiveFailures spanner.NullInt64
		var stateCreatedAt spanner.NullTime

		if err := r.ColumnByName("IsDisabledBySystem", &isDisabledBySystem); err != nil {
			return err
		}
		if err := r.ColumnByName("ConsecutiveFailures", &consecutiveFailures); err != nil {
			return err
		}
		if err := r.ColumnByName("CreatedAt", &stateCreatedAt); err != nil {
			return err
		} // This is ncs.CreatedAt
		if err := r.ColumnByName("StateUpdatedAt", &stateUpdatedAt); err != nil {
			return err
		}

		state.ChannelID = spannerChannel.ID
		state.IsDisabledBySystem = isDisabledBySystem.Bool
		state.ConsecutiveFailures = consecutiveFailures.Int64
		state.CreatedAt = stateCreatedAt.Time
		if stateUpdatedAt.Valid {
			state.UpdatedAt = stateUpdatedAt.Time
		}

		channel, err := spannerChannel.toPublic()
		if err != nil {
			return err
		}

		if channel.EmailConfig != nil && channel.EmailConfig.Address != "" {
			channels[channel.EmailConfig.Address] = TestNotificationChannel{
				ID:                 channel.ID,
				UserID:             channel.UserID,
				Email:              channel.EmailConfig.Address,
				IsDisabledBySystem: state.IsDisabledBySystem,
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to get user notification channels: %v", err)
	}
	return channels
}

func TestSyncUserProfileInfo(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	userID := "test-user-1"
	email1 := "test1@example.com"
	email2 := "test2@example.com"
	email3 := "test3@example.com"

	// Set up initial state for this test case
	// email1: enabled
	// email2: disabled
	channelID1 := uuid.New().String()
	channelID2 := uuid.New().String()
	now := time.Now().UTC()

	initialChannels := []NotificationChannel{
		{ID: channelID1, UserID: userID, Name: email1, Type: "email", EmailConfig: &EmailConfig{Address: email1, IsVerified: true}, CreatedAt: now, UpdatedAt: now},
		{ID: channelID2, UserID: userID, Name: email2, Type: "email", EmailConfig: &EmailConfig{Address: email2, IsVerified: true}, CreatedAt: now, UpdatedAt: now},
	}
	initialChannelStates := []NotificationChannelState{
		{ChannelID: channelID1, IsDisabledBySystem: false, ConsecutiveFailures: 0, CreatedAt: now, UpdatedAt: now},
		{ChannelID: channelID2, IsDisabledBySystem: true, ConsecutiveFailures: 0, CreatedAt: now, UpdatedAt: now},
	}

	var mutations []*spanner.Mutation
	for _, ch := range initialChannels {
		m, err := spanner.InsertStruct("NotificationChannels", ch.toSpanner())
		if err != nil {
			t.Fatalf("Failed to create initial channel mutation: %v", err)
		}
		mutations = append(mutations, m)
	}
	for _, cs := range initialChannelStates {
		m, err := spanner.InsertStruct("NotificationChannelStates", cs)
		if err != nil {
			t.Fatalf("Failed to create initial channel state mutation: %v", err)
		}
		mutations = append(mutations, m)
	}

	_, err := spannerClient.Apply(ctx, mutations)
	if err != nil {
		t.Fatalf("Failed to apply initial mutations: %v", err)
	}

	// Store initial UpdatedAt for comparison
	initialState1UpdatedAt := initialChannelStates[0].UpdatedAt
	initialState2UpdatedAt := initialChannelStates[1].UpdatedAt

	userProfile := backendtypes.UserProfile{
		UserID:       userID,
		GitHubUserID: 0,
		Emails:       []string{email2, email3}, // email1 is removed, email2 should be enabled, email3 is new
	}

	err = spannerClient.SyncUserProfileInfo(ctx, userProfile)
	if err != nil {
		t.Fatalf("SyncUserProfileInfo failed: %v", err)
	}

	currentState := getAllUserNotificationChannels(ctx, t, spannerClient, userID)

	expectedState := map[string]TestNotificationChannel{
		email1: {UserID: userID, Email: email1, IsDisabledBySystem: true},  // Should be disabled
		email2: {UserID: userID, Email: email2, IsDisabledBySystem: false}, // Should be enabled
		email3: {UserID: userID, Email: email3, IsDisabledBySystem: false}, // New, should be enabled
	}

	opts := cmpopts.IgnoreFields(TestNotificationChannel{}, "ID")
	if diff := cmp.Diff(expectedState, currentState, opts); diff != "" {
		t.Errorf("Mismatch in user notification channels (-want +got):\n%s", diff)
	}

	// Verify UpdatedAt was modified for changed records
	// For email1 (disabled from enabled): NotificationChannelState.UpdatedAt should change
	if currentState[email1].IsDisabledBySystem && currentState[email1].ID != "" {
		var state NotificationChannelState
		row, err := spannerClient.Single().ReadRow(ctx, "NotificationChannelStates", spanner.Key{currentState[email1].ID}, []string{"UpdatedAt"})
		if err != nil {
			t.Fatalf("Failed to read state for email1: %v", err)
		}
		if err := row.ToStruct(&state); err != nil {
			t.Fatalf("Failed to convert row to struct for email1 state: %v", err)
		}
		if state.UpdatedAt.Equal(initialState1UpdatedAt) {
			t.Errorf("UpdatedAt for email1 state was not modified after state change")
		}
	}

	// For email2 (enabled from disabled): NotificationChannelState.UpdatedAt should change
	if !currentState[email2].IsDisabledBySystem && currentState[email2].ID != "" {
		var state NotificationChannelState
		row, err := spannerClient.Single().ReadRow(ctx, "NotificationChannelStates", spanner.Key{currentState[email2].ID}, []string{"UpdatedAt"})
		if err != nil {
			t.Fatalf("Failed to read state for email2: %v", err)
		}
		if err := row.ToStruct(&state); err != nil {
			t.Fatalf("Failed to convert row to struct for email2 state: %v", err)
		}
		if state.UpdatedAt.Equal(initialState2UpdatedAt) {
			t.Errorf("UpdatedAt for email2 state was not modified after state change")
		}
	}

	// For email3 (new): NotificationChannel.CreatedAt and NotificationChannelState.CreatedAt should be recent
	if currentState[email3].ID != "" {
		var channel NotificationChannel
		row, err := spannerClient.Single().ReadRow(ctx, "NotificationChannels", spanner.Key{currentState[email3].ID}, []string{"CreatedAt"})
		if err != nil {
			t.Fatalf("Failed to read channel for email3: %v", err)
		}
		if err := row.ToStruct(&channel); err != nil {
			t.Fatalf("Failed to convert row to struct for email3 channel: %v", err)
		}
		if time.Since(channel.CreatedAt) > 5*time.Second {
			t.Errorf("CreatedAt for new email3 channel is not recent")
		}

		var state NotificationChannelState
		row, err = spannerClient.Single().ReadRow(ctx, "NotificationChannelStates", spanner.Key{currentState[email3].ID}, []string{"CreatedAt"})
		if err != nil {
			t.Fatalf("Failed to read state for email3: %v", err)
		}
		if err := row.ToStruct(&state); err != nil {
			t.Fatalf("Failed to convert row to struct for email3 state: %v", err)
		}
		if time.Since(state.CreatedAt) > 5*time.Second {
			t.Errorf("CreatedAt for new email3 state is not recent")
		}
	}
}
