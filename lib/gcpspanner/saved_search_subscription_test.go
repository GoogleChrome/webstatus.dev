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

func TestSavedSearchSubscriptionOperations(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	userID := uuid.NewString()
	otherUserID := uuid.NewString()

	// Pre-populate dependencies
	channelReq := CreateNotificationChannelRequest{
		UserID:      userID,
		Name:        "Test",
		Type:        "EMAIL",
		EmailConfig: &EmailConfig{Address: "test@example.com", IsVerified: true, VerificationToken: nil},
	}
	channelID, err := spannerClient.CreateNotificationChannel(ctx, channelReq)
	if err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}

	savedSearchID, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "Test Search",
		Query:       "is:widely",
		OwnerUserID: userID,
		Description: nil,
	})
	if err != nil {
		t.Fatalf("failed to create saved search: %v", err)
	}

	baseCreateReq := CreateSavedSearchSubscriptionRequest{
		UserID:        userID,
		ChannelID:     channelID,
		SavedSearchID: *savedSearchID,
		Triggers:      []string{"baseline.status"},
		Frequency:     "IMMEDIATE",
	}

	// Pre-populate a subscription for update/delete tests
	subToUpdateID, err := spannerClient.CreateSavedSearchSubscription(ctx, baseCreateReq)
	if err != nil {
		t.Fatalf("failed to pre-populate subscription for update tests: %v", err)
	}
	subToDeleteID, err := spannerClient.CreateSavedSearchSubscription(ctx, baseCreateReq)
	if err != nil {
		t.Fatalf("failed to pre-populate subscription for delete tests: %v", err)
	}

	t.Run("Create and Get", func(t *testing.T) {
		createReq := CreateSavedSearchSubscriptionRequest{
			UserID:        userID,
			ChannelID:     channelID,
			SavedSearchID: *savedSearchID,
			Triggers:      []string{"spec.links"},
			Frequency:     "DAILY",
		}
		subID, err := spannerClient.CreateSavedSearchSubscription(ctx, createReq)
		if err != nil {
			t.Fatalf("CreateSavedSearchSubscription failed: %v", err)
		}

		retrieved, err := spannerClient.GetSavedSearchSubscription(ctx, subID, userID)
		if err != nil {
			t.Fatalf("GetSavedSearchSubscription failed: %v", err)
		}
		expected := &SavedSearchSubscription{
			ID:            subID,
			UserID:        createReq.UserID,
			ChannelID:     createReq.ChannelID,
			SavedSearchID: createReq.SavedSearchID,
			Triggers:      createReq.Triggers,
			Frequency:     createReq.Frequency,
		}
		if diff := cmp.Diff(expected, retrieved); diff != "" {
			t.Errorf("GetSavedSearchSubscription mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("Get fails for wrong user", func(t *testing.T) {
		_, err := spannerClient.GetSavedSearchSubscription(ctx, subToUpdateID, otherUserID)
		if err == nil {
			t.Error("expected an error when getting subscription with wrong user ID, but got nil")
		}
	})

	t.Run("Update", func(t *testing.T) {
		updateReq := UpdateSavedSearchSubscriptionRequest{
			ID:        subToUpdateID,
			UserID:    userID,
			Triggers:  OptionallySet[[]string]{Value: []string{"developer_signals.upvotes"}, IsSet: true},
			Frequency: OptionallySet[string]{Value: "WEEKLY_DIGEST", IsSet: true},
		}
		err := spannerClient.UpdateSavedSearchSubscription(ctx, updateReq)
		if err != nil {
			t.Fatalf("UpdateSavedSearchSubscription failed: %v", err)
		}

		retrieved, err := spannerClient.GetSavedSearchSubscription(ctx, subToUpdateID, userID)
		if err != nil {
			t.Fatalf("GetSavedSearchSubscription after update failed: %v", err)
		}
		if retrieved.Frequency != "WEEKLY_DIGEST" {
			t.Errorf("expected updated frequency, got %s", retrieved.Frequency)
		}
	})

	t.Run("Delete success", func(t *testing.T) {
		err := spannerClient.DeleteSavedSearchSubscription(ctx, subToDeleteID, userID)
		if err != nil {
			t.Fatalf("DeleteSavedSearchSubscription failed: %v", err)
		}

		_, err = spannerClient.GetSavedSearchSubscription(ctx, subToDeleteID, userID)
		if err == nil {
			t.Error("expected an error after getting a deleted subscription, but got nil")
		}
	})
}
