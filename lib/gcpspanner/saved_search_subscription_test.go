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

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
)

func TestCreateAndGetSavedSearchSubscription(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	userID := uuid.NewString()

	// Pre-populate dependencies
	channelReq := CreateNotificationChannelRequest{
		UserID:      userID,
		Name:        "Test",
		Type:        "EMAIL",
		EmailConfig: &EmailConfig{Address: "test@example.com", IsVerified: true, VerificationToken: nil},
	}
	channelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, channelReq)
	if err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}
	channelID := *channelIDPtr

	savedSearchIDPtr, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "Test Search",
		Query:       "is:widely",
		OwnerUserID: userID,
		Description: nil,
	})
	if err != nil {
		t.Fatalf("failed to create saved search: %v", err)
	}
	savedSearchID := *savedSearchIDPtr

	createReq := CreateSavedSearchSubscriptionRequest{
		UserID:        userID,
		ChannelID:     channelID,
		SavedSearchID: savedSearchID,
		Triggers:      []string{"spec.links"},
		Frequency:     "DAILY",
	}
	subIDPtr, err := spannerClient.CreateSavedSearchSubscription(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateSavedSearchSubscription failed: %v", err)
	}
	subID := *subIDPtr

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
		CreatedAt:     time.Time{},
		UpdatedAt:     time.Time{},
	}
	if diff := cmp.Diff(expected, retrieved,
		cmpopts.IgnoreFields(SavedSearchSubscription{
			ID:            "",
			UserID:        "",
			ChannelID:     "",
			SavedSearchID: "",
			Triggers:      nil,
			Frequency:     "",
			CreatedAt:     time.Time{},
			UpdatedAt:     time.Time{},
		}, "CreatedAt", "UpdatedAt")); diff != "" {
		t.Errorf("GetSavedSearchSubscription mismatch (-want +got):\n%s", diff)
	}
}
func TestGetSavedSearchSubscriptionFailsForWrongUser(t *testing.T) {
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
	channelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, channelReq)
	if err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}
	channelID := *channelIDPtr

	savedSearchIDPtr, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "Test Search",
		Query:       "is:widely",
		OwnerUserID: userID,
		Description: nil,
	})
	if err != nil {
		t.Fatalf("failed to create saved search: %v", err)
	}
	savedSearchID := *savedSearchIDPtr

	baseCreateReq := CreateSavedSearchSubscriptionRequest{
		UserID:        userID,
		ChannelID:     channelID,
		SavedSearchID: savedSearchID,
		Triggers:      []string{"baseline.status"},
		Frequency:     "IMMEDIATE",
	}

	subToUpdateIDPtr, err := spannerClient.CreateSavedSearchSubscription(ctx, baseCreateReq)
	if err != nil {
		t.Fatalf("failed to pre-populate subscription for update tests: %v", err)
	}
	subToUpdateID := *subToUpdateIDPtr

	_, err = spannerClient.GetSavedSearchSubscription(ctx, subToUpdateID, otherUserID)
	if err == nil {
		t.Error("expected an error when getting subscription with wrong user ID, but got nil")
	}
}

func TestUpdateSavedSearchSubscription(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	userID := uuid.NewString()

	// Pre-populate dependencies
	channelReq := CreateNotificationChannelRequest{
		UserID:      userID,
		Name:        "Test",
		Type:        "EMAIL",
		EmailConfig: &EmailConfig{Address: "test@example.com", IsVerified: true, VerificationToken: nil},
	}
	channelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, channelReq)
	if err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}
	channelID := *channelIDPtr

	savedSearchIDPtr, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "Test Search",
		Query:       "is:widely",
		OwnerUserID: userID,
		Description: nil,
	})
	if err != nil {
		t.Fatalf("failed to create saved search: %v", err)
	}
	savedSearchID := *savedSearchIDPtr

	baseCreateReq := CreateSavedSearchSubscriptionRequest{
		UserID:        userID,
		ChannelID:     channelID,
		SavedSearchID: savedSearchID,
		Triggers:      []string{"baseline.status"},
		Frequency:     "IMMEDIATE",
	}

	subToUpdateIDPtr, err := spannerClient.CreateSavedSearchSubscription(ctx, baseCreateReq)
	if err != nil {
		t.Fatalf("failed to pre-populate subscription for update tests: %v", err)
	}
	subToUpdateID := *subToUpdateIDPtr

	updateReq := UpdateSavedSearchSubscriptionRequest{
		ID:        subToUpdateID,
		UserID:    userID,
		Triggers:  OptionallySet[[]string]{Value: []string{"developer_signals.upvotes"}, IsSet: true},
		Frequency: OptionallySet[string]{Value: "WEEKLY_DIGEST", IsSet: true},
	}
	err = spannerClient.UpdateSavedSearchSubscription(ctx, updateReq)
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
}

func TestDeleteSavedSearchSubscription(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	userID := uuid.NewString()

	// Pre-populate dependencies
	channelReq := CreateNotificationChannelRequest{
		UserID:      userID,
		Name:        "Test",
		Type:        "EMAIL",
		EmailConfig: &EmailConfig{Address: "test@example.com", IsVerified: true, VerificationToken: nil},
	}
	channelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, channelReq)
	if err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}
	channelID := *channelIDPtr

	savedSearchIDPtr, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "Test Search",
		Query:       "is:widely",
		OwnerUserID: userID,
		Description: nil,
	})
	if err != nil {
		t.Fatalf("failed to create saved search: %v", err)
	}
	savedSearchID := *savedSearchIDPtr

	baseCreateReq := CreateSavedSearchSubscriptionRequest{
		UserID:        userID,
		ChannelID:     channelID,
		SavedSearchID: savedSearchID,
		Triggers:      []string{"baseline.status"},
		Frequency:     "IMMEDIATE",
	}

	subToDeleteIDPtr, err := spannerClient.CreateSavedSearchSubscription(ctx, baseCreateReq)
	if err != nil {
		t.Fatalf("failed to pre-populate subscription for delete tests: %v", err)
	}
	subToDeleteID := *subToDeleteIDPtr

	err = spannerClient.DeleteSavedSearchSubscription(ctx, subToDeleteID, userID)
	if err != nil {
		t.Fatalf("DeleteSavedSearchSubscription failed: %v", err)
	}

	_, err = spannerClient.GetSavedSearchSubscription(ctx, subToDeleteID, userID)
	if err == nil {
		t.Error("expected an error after getting a deleted subscription, but got nil")
	}
}

func TestListSavedSearchSubscriptions(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	userID := uuid.NewString()

	// Pre-populate dependencies
	channelReq := CreateNotificationChannelRequest{
		UserID:      userID,
		Name:        "Test",
		Type:        "EMAIL",
		EmailConfig: &EmailConfig{Address: "test@example.com", IsVerified: true, VerificationToken: nil},
	}
	channelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, channelReq)
	if err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}
	channelID := *channelIDPtr

	savedSearchIDPtr, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "Test Search",
		Query:       "is:widely",
		OwnerUserID: userID,
		Description: nil,
	})
	if err != nil {
		t.Fatalf("failed to create saved search: %v", err)
	}
	savedSearchID := *savedSearchIDPtr

	baseCreateReq := CreateSavedSearchSubscriptionRequest{
		UserID:        userID,
		ChannelID:     channelID,
		SavedSearchID: savedSearchID,
		Triggers:      []string{"baseline.status"},
		Frequency:     "IMMEDIATE",
	}

	// Create a few subscriptions to list
	for i := 0; i < 3; i++ {
		_, err := spannerClient.CreateSavedSearchSubscription(ctx, baseCreateReq)
		if err != nil {
			t.Fatalf("failed to create subscription for list test: %v", err)
		}
	}

	// List first page
	listReq1 := ListSavedSearchSubscriptionsRequest{
		UserID:    userID,
		PageSize:  2,
		PageToken: nil,
	}
	results1, nextPageToken1, err := spannerClient.ListSavedSearchSubscriptions(ctx, listReq1)
	if err != nil {
		t.Fatalf("ListSavedSearchSubscriptions page 1 failed: %v", err)
	}
	if len(results1) != 2 {
		t.Errorf("expected 2 results on page 1, got %d", len(results1))
	}
	if nextPageToken1 == nil {
		t.Fatal("expected a next page token on page 1, got nil")
	}

	// List second page
	listReq2 := ListSavedSearchSubscriptionsRequest{
		UserID:    userID,
		PageSize:  2,
		PageToken: nextPageToken1,
	}
	results2, _, err := spannerClient.ListSavedSearchSubscriptions(ctx, listReq2)
	if err != nil {
		t.Fatalf("ListSavedSearchSubscriptions page 2 failed: %v", err)
	}
	if len(results2) < 1 {
		t.Errorf("expected at least 1 result on page 2, got %d", len(results2))
	}
}
