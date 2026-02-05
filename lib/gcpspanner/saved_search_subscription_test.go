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
	"slices"
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
		Type:        NotificationChannelTypeEmail,
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
		Triggers:      []SubscriptionTrigger{SubscriptionTriggerFeatureBaselineRegressionToLimited},
		Frequency:     SavedSearchSnapshotTypeImmediate,
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
		Type:        NotificationChannelTypeEmail,
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
		Triggers:      []SubscriptionTrigger{SubscriptionTriggerFeatureBaselineRegressionToLimited},
		Frequency:     SavedSearchSnapshotTypeImmediate,
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
		Type:        NotificationChannelTypeEmail,
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
		Triggers:      []SubscriptionTrigger{SubscriptionTriggerFeatureBaselinePromoteToNewly},
		Frequency:     SavedSearchSnapshotTypeImmediate,
	}

	subToUpdateIDPtr, err := spannerClient.CreateSavedSearchSubscription(ctx, baseCreateReq)
	if err != nil {
		t.Fatalf("failed to pre-populate subscription for update tests: %v", err)
	}
	subToUpdateID := *subToUpdateIDPtr

	updateReq := UpdateSavedSearchSubscriptionRequest{
		ID:     subToUpdateID,
		UserID: userID,
		Triggers: OptionallySet[[]SubscriptionTrigger]{Value: []SubscriptionTrigger{
			SubscriptionTriggerBrowserImplementationAnyComplete}, IsSet: true},
		Frequency: OptionallySet[SavedSearchSnapshotType]{Value: SavedSearchSnapshotTypeWeekly, IsSet: true},
	}
	err = spannerClient.UpdateSavedSearchSubscription(ctx, updateReq)
	if err != nil {
		t.Fatalf("UpdateSavedSearchSubscription failed: %v", err)
	}

	retrieved, err := spannerClient.GetSavedSearchSubscription(ctx, subToUpdateID, userID)
	if err != nil {
		t.Fatalf("GetSavedSearchSubscription after update failed: %v", err)
	}
	if retrieved.Frequency != SavedSearchSnapshotTypeWeekly {
		t.Errorf("expected updated frequency, got %s", retrieved.Frequency)
	}
	expectedTriggers := []SubscriptionTrigger{SubscriptionTriggerBrowserImplementationAnyComplete}
	if diff := cmp.Diff(expectedTriggers, retrieved.Triggers); diff != "" {
		t.Errorf("updated triggers mismatch (-want +got):\n%s", diff)
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
		Type:        NotificationChannelTypeEmail,
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
		Triggers:      []SubscriptionTrigger{SubscriptionTriggerFeatureBaselinePromoteToNewly},
		Frequency:     SavedSearchSnapshotTypeImmediate,
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
		Type:        NotificationChannelTypeEmail,
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
		Triggers:      []SubscriptionTrigger{SubscriptionTriggerFeatureBaselinePromoteToNewly},
		Frequency:     SavedSearchSnapshotTypeImmediate,
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

func TestFindAllActivePushSubscriptions(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	// User and Search setup
	userID := uuid.NewString()
	savedSearchIDPtr, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name: "Test Search", Query: "is:test", OwnerUserID: userID, Description: nil,
	})
	if err != nil {
		t.Fatalf("failed to create saved search: %v", err)
	}
	savedSearchID := *savedSearchIDPtr

	// Channel 1: Valid, active EMAIL channel
	emailChannelReq := CreateNotificationChannelRequest{
		UserID: userID, Name: "Email", Type: NotificationChannelTypeEmail,
		EmailConfig: &EmailConfig{Address: "active@example.com", IsVerified: true, VerificationToken: nil},
	}
	emailChannelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, emailChannelReq)
	if err != nil {
		t.Fatalf("failed to create email channel: %v", err)
	}
	emailChannelID := *emailChannelIDPtr

	// Channel 2: Valid, active WEBHOOK channel
	// TODO: Enable webhook channel tests once webhooks are supported.
	// webhookChannelReq := CreateNotificationChannelRequest{
	// 	UserID: userID, Name: "Webhook", Type: "WEBHOOK",
	// 	WebhookConfig: &WebhookConfig{URL: "https://example.com/webhook", IsVerified: true},
	// }
	// webhookChannelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, webhookChannelReq)
	// if err != nil {
	// 	t.Fatalf("failed to create webhook channel: %v", err)
	// }
	// webhookChannelID := *webhookChannelIDPtr

	// Channel 3: RSS channel (should be ignored)
	rssChannelReq := CreateNotificationChannelRequest{UserID: userID, Name: "RSS", Type: "RSS", EmailConfig: nil}
	rssChannelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, rssChannelReq)
	if err != nil {
		t.Fatalf("failed to create rss channel: %v", err)
	}
	rssChannelID := *rssChannelIDPtr

	// Channel 4: Disabled EMAIL channel (should be ignored)
	disabledEmailReq := CreateNotificationChannelRequest{
		UserID: userID, Name: "Disabled", Type: NotificationChannelTypeEmail,
		EmailConfig: &EmailConfig{Address: "disabled@example.com", IsVerified: true, VerificationToken: nil},
	}
	disabledChannelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, disabledEmailReq)
	if err != nil {
		t.Fatalf("failed to create disabled email channel: %v", err)
	}
	disabledChannelID := *disabledChannelIDPtr
	// Manually disable it
	err = spannerClient.UpsertNotificationChannelState(ctx, NotificationChannelState{
		ChannelID:           disabledChannelID,
		ConsecutiveFailures: 3,
		UpdatedAt:           time.Now(),
		CreatedAt:           time.Now(),
		IsDisabledBySystem:  true,
	})
	if err != nil {
		t.Fatalf("failed to disable channel: %v", err)
	}

	// Subscription 1: Correct, on active EMAIL channel
	_, err = spannerClient.CreateSavedSearchSubscription(ctx, CreateSavedSearchSubscriptionRequest{
		UserID: userID, ChannelID: emailChannelID, SavedSearchID: savedSearchID,
		Frequency: SavedSearchSnapshotTypeImmediate, Triggers: []SubscriptionTrigger{
			SubscriptionTriggerFeatureBaselineRegressionToLimited,
			SubscriptionTriggerBrowserImplementationAnyComplete,
			SubscriptionTriggerFeatureBaselinePromoteToNewly,
			SubscriptionTriggerFeatureBaselinePromoteToWidely,
		},
	})
	if err != nil {
		t.Fatalf("failed to create sub 1: %v", err)
	}

	// Subscription 2: Correct, on active WEBHOOK channel
	// TODO: Enable webhook channel tests once webhooks are supported.
	// _, err = spannerClient.CreateSavedSearchSubscription(ctx, CreateSavedSearchSubscriptionRequest{
	// 	UserID: userID, ChannelID: webhookChannelID, SavedSearchID: savedSearchID,
	// 		Frequency: SavedSearchSnapshotTypeImmediate,
	// })
	// if err != nil {
	// 	t.Fatalf("failed to create sub 2: %v", err)
	// }

	// Subscription 3: Wrong frequency
	_, err = spannerClient.CreateSavedSearchSubscription(ctx, CreateSavedSearchSubscriptionRequest{
		UserID: userID, ChannelID: emailChannelID, SavedSearchID: savedSearchID, Frequency: "WEEKLY", Triggers: nil,
	})
	if err != nil {
		t.Fatalf("failed to create sub 3: %v", err)
	}

	// Subscription 4: Non-push channel (RSS)
	_, err = spannerClient.CreateSavedSearchSubscription(ctx, CreateSavedSearchSubscriptionRequest{
		UserID: userID, ChannelID: rssChannelID, SavedSearchID: savedSearchID,
		Frequency: SavedSearchSnapshotTypeImmediate, Triggers: nil,
	})
	if err != nil {
		t.Fatalf("failed to create sub 4: %v", err)
	}

	// Subscription 5: Disabled channel
	_, err = spannerClient.CreateSavedSearchSubscription(ctx, CreateSavedSearchSubscriptionRequest{
		UserID: userID, ChannelID: disabledChannelID, SavedSearchID: savedSearchID,
		Frequency: SavedSearchSnapshotTypeImmediate,
		Triggers:  nil,
	})
	if err != nil {
		t.Fatalf("failed to create sub 5: %v", err)
	}

	// Find subscribers
	subscribers, err := spannerClient.FindAllActivePushSubscriptions(ctx, savedSearchID,
		SavedSearchSnapshotTypeImmediate)
	if err != nil {
		t.Fatalf("FindAllActivePushSubscriptions failed: %v", err)
	}

	// Assertions
	if len(subscribers) != 1 {
		t.Fatalf("expected 1 subscribers, got %d", len(subscribers))
	}

	foundEmail := false

	// Check for webhook once enabled.
	// foundWebhook := false
	for _, sub := range subscribers {
		if sub.ChannelID == emailChannelID {
			// Do the comparison for the email subscriber details
			if sub.Type != NotificationChannelTypeEmail {
				t.Errorf("expected EMAIL type for email channel, got %s", sub.Type)

				continue
			}
			if sub.EmailConfig == nil {
				t.Error("expected EmailConfig to be set for email subscriber, got nil")

				continue
			}
			if sub.EmailConfig.Address != "active@example.com" {
				t.Errorf("expected address to be active@example.com, got %s", sub.EmailConfig.Address)

				continue
			}
			expectedTriggers := []SubscriptionTrigger{
				SubscriptionTriggerFeatureBaselineRegressionToLimited,
				SubscriptionTriggerBrowserImplementationAnyComplete,
				SubscriptionTriggerFeatureBaselinePromoteToNewly,
				SubscriptionTriggerFeatureBaselinePromoteToWidely,
			}
			if !slices.Equal(expectedTriggers, sub.Triggers) {
				t.Errorf("expected triggers %v, got %v", expectedTriggers, sub.Triggers)

				continue
			}

			foundEmail = true
		}
		// if sub.ChannelID == webhookChannelID {
		// 	foundWebhook = true
		// }
	}

	if !foundEmail {
		t.Error("did not find the expected EMAIL subscriber")
	}
	// if !foundWebhook {
	// 	t.Error("did not find the expected WEBHOOK subscriber")
	// }
}

func TestCreateSavedSearchSubscriptionLimitExceeded(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	userID := uuid.NewString()

	// Pre-populate dependencies
	channelReq := CreateNotificationChannelRequest{
		UserID:      userID,
		Name:        "Test",
		Type:        NotificationChannelTypeEmail,
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

	// Create subscriptions up to the limit
	limit := defaultMaxSubscriptionsPerUser
	for i := 0; i < limit; i++ {
		_, err := spannerClient.CreateSavedSearchSubscription(ctx, CreateSavedSearchSubscriptionRequest{
			UserID:        userID,
			ChannelID:     channelID,
			SavedSearchID: savedSearchID,
			Triggers:      []SubscriptionTrigger{SubscriptionTriggerFeatureBaselineRegressionToLimited},
			Frequency:     SavedSearchSnapshotTypeImmediate,
		})
		if err != nil {
			t.Fatalf("failed to create subscription %d: %v", i, err)
		}
	}

	// Try to create one more
	_, err = spannerClient.CreateSavedSearchSubscription(ctx, CreateSavedSearchSubscriptionRequest{
		UserID:        userID,
		ChannelID:     channelID,
		SavedSearchID: savedSearchID,
		Triggers:      []SubscriptionTrigger{SubscriptionTriggerFeatureBaselineRegressionToLimited},
		Frequency:     SavedSearchSnapshotTypeImmediate,
	})
	if !errors.Is(err, ErrSubscriptionLimitExceeded) {
		t.Errorf("expected ErrSubscriptionLimitExceeded, got %v", err)
	}
}
