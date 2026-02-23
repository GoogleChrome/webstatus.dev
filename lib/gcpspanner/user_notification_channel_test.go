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
		var rowData notificationChannelWithState
		if err := r.ToStruct(&rowData); err != nil {
			return err
		}

		spannerChannel := spannerNotificationChannel{
			ID:        rowData.ID,
			UserID:    rowData.UserID,
			Name:      rowData.Name,
			Type:      string(rowData.Type),
			Config:    rowData.Config,
			CreatedAt: rowData.CreatedAt,
			UpdatedAt: rowData.UpdatedAt,
		}
		state := NotificationChannelState{
			ChannelID:           rowData.ID,
			IsDisabledBySystem:  rowData.IsDisabledBySystem.Bool,
			ConsecutiveFailures: rowData.ConsecutiveFailures.Int64,
			CreatedAt:           rowData.StateCreatedAt.Time,
			UpdatedAt:           rowData.StateUpdatedAt.Time,
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

func verifyUpdatedAtModified(t *testing.T, client *Client, channelID string, initialUpdatedAt time.Time) {
	t.Helper()
	var state NotificationChannelState
	row, err := client.Single().ReadRow(context.Background(), "NotificationChannelStates",
		spanner.Key{channelID}, []string{"UpdatedAt"})
	if err != nil {
		t.Fatalf("Failed to read state for channel %s: %v", channelID, err)
	}
	if err := row.ToStruct(&state); err != nil {
		t.Fatalf("Failed to convert row to struct for channel %s state: %v", channelID, err)
	}
	if state.UpdatedAt.Equal(initialUpdatedAt) {
		t.Errorf("UpdatedAt for channel %s state was not modified after state change", channelID)
	}
}

func verifyCreatedAtRecent(t *testing.T, client *Client, channelID string) {
	t.Helper()
	var channel NotificationChannel
	row, err := client.Single().ReadRow(context.Background(), "NotificationChannels",
		spanner.Key{channelID}, []string{"CreatedAt"})
	if err != nil {
		t.Fatalf("Failed to read channel for %s: %v", channelID, err)
	}
	if err := row.ToStruct(&channel); err != nil {
		t.Fatalf("Failed to convert row to struct for %s channel: %v", channelID, err)
	}
	if time.Since(channel.CreatedAt) > 5*time.Second {
		t.Errorf("CreatedAt for new channel %s is not recent", channelID)
	}

	var state NotificationChannelState
	row, err = client.Single().ReadRow(context.Background(), "NotificationChannelStates",
		spanner.Key{channelID}, []string{"CreatedAt"})
	if err != nil {
		t.Fatalf("Failed to read state for %s: %v", channelID, err)
	}
	if err := row.ToStruct(&state); err != nil {
		t.Fatalf("Failed to convert row to struct for %s state: %v", channelID, err)
	}
	if time.Since(state.CreatedAt) > 5*time.Second {
		t.Errorf("CreatedAt for new %s state is not recent", channelID)
	}
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
	channelID1 := uuid.NewString()
	channelID2 := uuid.NewString()
	now := time.Now().UTC()

	initialChannels := []NotificationChannel{
		{
			ID: channelID1, UserID: userID, Name: email1, Type: "email",
			EmailConfig:   &EmailConfig{Address: email1, IsVerified: true, VerificationToken: nil},
			WebhookConfig: nil,
			CreatedAt:     now, UpdatedAt: now,
		},
		{
			ID: channelID2, UserID: userID, Name: email2, Type: "email",
			EmailConfig:   &EmailConfig{Address: email2, IsVerified: true, VerificationToken: nil},
			WebhookConfig: nil,
			CreatedAt:     now, UpdatedAt: now,
		},
	}
	initialChannelStates := []NotificationChannelState{
		{ChannelID: channelID1, IsDisabledBySystem: false, ConsecutiveFailures: 0, CreatedAt: now, UpdatedAt: now},
		{ChannelID: channelID2, IsDisabledBySystem: true, ConsecutiveFailures: 0, CreatedAt: now, UpdatedAt: now},
	}

	// Pre-allocate mutations slice to avoid reallocations.
	mutations := make([]*spanner.Mutation, 0, len(initialChannels)*2)
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

	userProfile := UserProfile{
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
		email1: {ID: "", UserID: userID, Email: email1, IsDisabledBySystem: true},  // Should be disabled
		email2: {ID: "", UserID: userID, Email: email2, IsDisabledBySystem: false}, // Should be enabled
		email3: {ID: "", UserID: userID, Email: email3, IsDisabledBySystem: false}, // New, should be enabled
	}

	opts := cmpopts.IgnoreFields(TestNotificationChannel{ID: "", UserID: "", Email: "", IsDisabledBySystem: false}, "ID")
	if diff := cmp.Diff(expectedState, currentState, opts); diff != "" {
		t.Errorf("Mismatch in user notification channels (-want +got):\n%s", diff)
	}

	// Verify UpdatedAt was modified for changed records
	if currentState[email1].IsDisabledBySystem && currentState[email1].ID != "" {
		verifyUpdatedAtModified(t, spannerClient, currentState[email1].ID, initialState1UpdatedAt)
	}

	// For email2 (enabled from disabled): NotificationChannelState.UpdatedAt should change
	if !currentState[email2].IsDisabledBySystem && currentState[email2].ID != "" {
		verifyUpdatedAtModified(t, spannerClient, currentState[email2].ID, initialState2UpdatedAt)
	}

	// For email3 (new): NotificationChannel.CreatedAt and NotificationChannelState.CreatedAt should be recent
	if currentState[email3].ID != "" {
		verifyCreatedAtRecent(t, spannerClient, currentState[email3].ID)
	}
}
