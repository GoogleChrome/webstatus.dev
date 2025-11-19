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

	"cloud.google.com/go/spanner"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
)

func TestNotificationChannelRefactoredOperations(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	userID := uuid.NewString()
	otherUserID := uuid.NewString()

	baseCreateReq := CreateNotificationChannelRequest{
		UserID: userID,
		Name:   "Test Email",
		Type:   "EMAIL",
		EmailConfig: &EmailConfig{
			Address:           "test@example.com",
			IsVerified:        true,
			VerificationToken: nil,
		},
	}

	// Pre-populate a channel for update/delete tests
	channelToUpdateIDPtr, err := spannerClient.CreateNotificationChannel(ctx, baseCreateReq)
	if err != nil {
		t.Fatalf("failed to pre-populate channel for update/delete tests: %v", err)
	}
	channelToUpdateID := *channelToUpdateIDPtr

	channelToDeleteIDPtr, err := spannerClient.CreateNotificationChannel(ctx, baseCreateReq)
	if err != nil {
		t.Fatalf("failed to pre-populate channel for delete tests: %v", err)
	}
	channelToDeleteID := *channelToDeleteIDPtr

	t.Run("Create and Get", func(t *testing.T) {
		createReq := CreateNotificationChannelRequest{
			UserID: userID,
			Name:   "A new channel",
			Type:   "EMAIL",
			EmailConfig: &EmailConfig{
				Address:           "new@example.com",
				IsVerified:        false,
				VerificationToken: nil,
			},
		}
		channelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, createReq)
		if err != nil {
			t.Fatalf("CreateNotificationChannel failed: %v", err)
		}
		channelID := *channelIDPtr

		retrieved, err := spannerClient.GetNotificationChannel(ctx, channelID, userID)
		if err != nil {
			t.Fatalf("GetNotificationChannel failed: %v", err)
		}

		expected := &NotificationChannel{
			ID:          channelID,
			UserID:      createReq.UserID,
			Name:        createReq.Name,
			Type:        createReq.Type,
			EmailConfig: createReq.EmailConfig,
			CreatedAt:   spanner.CommitTimestamp,
			UpdatedAt:   spanner.CommitTimestamp,
		}

		if diff := cmp.Diff(expected, retrieved,
			cmpopts.IgnoreFields(NotificationChannel{
				ID:          "",
				UserID:      "",
				Name:        "",
				Type:        "",
				EmailConfig: nil,
				CreatedAt:   spanner.CommitTimestamp,
				UpdatedAt:   spanner.CommitTimestamp,
			}, "CreatedAt", "UpdatedAt")); diff != "" {
			t.Errorf("GetNotificationChannel mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("Get fails for wrong user", func(t *testing.T) {
		_, err := spannerClient.GetNotificationChannel(ctx, channelToUpdateID, otherUserID)
		if err == nil {
			t.Error("expected an error when getting channel with wrong user ID, but got nil")
		}
	})

	t.Run("Update", func(t *testing.T) {
		updateReq := UpdateNotificationChannelRequest{
			ID:     channelToUpdateID,
			UserID: userID,
			Name:   OptionallySet[string]{Value: "Updated Name", IsSet: true},
			EmailConfig: OptionallySet[*EmailConfig]{
				Value: &EmailConfig{Address: "updated@example.com", IsVerified: true, VerificationToken: nil},
				IsSet: true,
			},
		}
		err := spannerClient.UpdateNotificationChannel(ctx, updateReq)
		if err != nil {
			t.Fatalf("UpdateNotificationChannel failed: %v", err)
		}

		retrieved, err := spannerClient.GetNotificationChannel(ctx, channelToUpdateID, userID)
		if err != nil {
			t.Fatalf("GetNotificationChannel after update failed: %v", err)
		}
		if retrieved.Name != "Updated Name" {
			t.Errorf("expected updated name, got %s", retrieved.Name)
		}
		if retrieved.EmailConfig == nil || retrieved.EmailConfig.Address != "updated@example.com" {
			t.Errorf("expected updated email config, got %+v", retrieved.EmailConfig)
		}
	})

	t.Run("Delete success", func(t *testing.T) {
		err := spannerClient.DeleteNotificationChannel(ctx, channelToDeleteID, userID)
		if err != nil {
			t.Fatalf("DeleteNotificationChannel failed: %v", err)
		}

		_, err = spannerClient.GetNotificationChannel(ctx, channelToDeleteID, userID)
		if err == nil {
			t.Error("expected an error after getting a deleted channel, but got nil")
		}
	})
}

func TestListNotificationChannels(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	userID := uuid.NewString()

	baseCreateReq := CreateNotificationChannelRequest{
		UserID: userID,
		Name:   "Test Email",
		Type:   "EMAIL",
		EmailConfig: &EmailConfig{
			Address:           "test@example.com",
			IsVerified:        true,
			VerificationToken: nil,
		},
	}

	// Create a few channels to list
	for i := 0; i < 3; i++ {
		_, err := spannerClient.CreateNotificationChannel(ctx, baseCreateReq)
		if err != nil {
			t.Fatalf("failed to create notification channel for list test: %v", err)
		}
	}

	// List first page
	listReq1 := ListNotificationChannelsRequest{
		UserID:    userID,
		PageSize:  2,
		PageToken: nil,
	}
	results1, nextPageToken1, err := spannerClient.ListNotificationChannels(ctx, listReq1)
	if err != nil {
		t.Fatalf("ListNotificationChannels page 1 failed: %v", err)
	}
	if len(results1) != 2 {
		t.Errorf("expected 2 results on page 1, got %d", len(results1))
	}
	if nextPageToken1 == nil {
		t.Fatal("expected a next page token on page 1, got nil")
	}

	// List second page
	listReq2 := ListNotificationChannelsRequest{
		UserID:    userID,
		PageSize:  2,
		PageToken: nextPageToken1,
	}
	results2, _, err := spannerClient.ListNotificationChannels(ctx, listReq2)
	if err != nil {
		t.Fatalf("ListNotificationChannels page 2 failed: %v", err)
	}
	if len(results2) < 1 {
		t.Errorf("expected at least 1 result on page 2, got %d", len(results2))
	}
}
