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
	"fmt"
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
		UserID:        userID,
		Name:          "Test",
		Type:          NotificationChannelTypeEmail,
		EmailConfig:   &EmailConfig{Address: "test@example.com", IsVerified: true, VerificationToken: nil},
		WebhookConfig: nil,
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
		ChannelType:   nil,
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
	expected := &SavedSearchSubscriptionView{
		SavedSearchSubscription: SavedSearchSubscription{
			ID:            subID,
			ChannelID:     createReq.ChannelID,
			SavedSearchID: createReq.SavedSearchID,
			Triggers:      createReq.Triggers,
			Frequency:     createReq.Frequency,
			CreatedAt:     time.Time{},
			UpdatedAt:     time.Time{},
		},
		SavedSearchName: "Test Search",
	}
	if diff := cmp.Diff(expected, retrieved,
		cmpopts.IgnoreFields(SavedSearchSubscriptionView{
			SavedSearchSubscription: SavedSearchSubscription{
				ID:            "",
				ChannelID:     "",
				SavedSearchID: "",
				Triggers:      nil,
				Frequency:     "",
				CreatedAt:     time.Time{},
				UpdatedAt:     time.Time{},
			},
			SavedSearchName: "",
		},
			"SavedSearchSubscription.CreatedAt", "SavedSearchSubscription.UpdatedAt")); diff != "" {
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
		UserID:        userID,
		Name:          "Test",
		Type:          NotificationChannelTypeEmail,
		EmailConfig:   &EmailConfig{Address: "test@example.com", IsVerified: true, VerificationToken: nil},
		WebhookConfig: nil,
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
		ChannelType:   nil,
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
		UserID:        userID,
		Name:          "Test",
		Type:          NotificationChannelTypeEmail,
		EmailConfig:   &EmailConfig{Address: "test@example.com", IsVerified: true, VerificationToken: nil},
		WebhookConfig: nil,
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
		ChannelType:   nil,
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
	if retrieved.SavedSearchName != "Test Search" {
		t.Errorf("expected SavedSearchName to be 'Test Search', got '%s'", retrieved.SavedSearchName)
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
		UserID:        userID,
		Name:          "Test",
		Type:          NotificationChannelTypeEmail,
		EmailConfig:   &EmailConfig{Address: "test@example.com", IsVerified: true, VerificationToken: nil},
		WebhookConfig: nil,
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
		ChannelType:   nil,
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
		UserID:        userID,
		Name:          "Test",
		Type:          NotificationChannelTypeEmail,
		EmailConfig:   &EmailConfig{Address: "test@example.com", IsVerified: true, VerificationToken: nil},
		WebhookConfig: nil,
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
		ChannelType:   nil,
	}

	// Create a few subscriptions to list
	for i := range 3 {
		savedSearchIDPtr, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
			Name:        "Test Search",
			Query:       fmt.Sprintf("is:widely_%d", i),
			OwnerUserID: userID,
			Description: nil,
		})
		if err != nil {
			t.Fatalf("failed to create saved search %d: %v", i, err)
		}
		req := baseCreateReq
		req.SavedSearchID = *savedSearchIDPtr
		_, err = spannerClient.CreateSavedSearchSubscription(ctx, req)
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
	for _, sub := range results1 {
		if sub.SavedSearchName != "Test Search" {
			t.Errorf("expected SavedSearchName to be 'Test Search', got '%s'", sub.SavedSearchName)
		}
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

	savedSearchID2Ptr, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name: "Test Search 2", Query: "is:test2", OwnerUserID: userID, Description: nil,
	})
	if err != nil {
		t.Fatalf("failed to create saved search 2: %v", err)
	}
	savedSearchID2 := *savedSearchID2Ptr

	// Channel 1: Valid, active EMAIL channel
	emailChannelReq := CreateNotificationChannelRequest{
		UserID: userID, Name: "Email", Type: NotificationChannelTypeEmail,
		EmailConfig:   &EmailConfig{Address: "active@example.com", IsVerified: true, VerificationToken: nil},
		WebhookConfig: nil,
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
	rssChannelReq := CreateNotificationChannelRequest{
		UserID: userID, Name: "RSS", Type: "RSS",
		EmailConfig: nil, WebhookConfig: nil,
	}
	rssChannelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, rssChannelReq)
	if err != nil {
		t.Fatalf("failed to create rss channel: %v", err)
	}
	rssChannelID := *rssChannelIDPtr

	// Channel 4: Disabled EMAIL channel (should be ignored)
	disabledEmailReq := CreateNotificationChannelRequest{
		UserID: userID, Name: "Disabled", Type: NotificationChannelTypeEmail,
		EmailConfig:   &EmailConfig{Address: "disabled@example.com", IsVerified: true, VerificationToken: nil},
		WebhookConfig: nil,
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
		ChannelType: nil,
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
		UserID: userID, ChannelID: emailChannelID, SavedSearchID: savedSearchID2, Frequency: "WEEKLY", Triggers: nil,
		ChannelType: nil,
	})
	if err != nil {
		t.Fatalf("failed to create sub 3: %v", err)
	}

	// Subscription 4: Non-push channel (RSS)
	_, err = spannerClient.CreateSavedSearchSubscription(ctx, CreateSavedSearchSubscriptionRequest{
		UserID: userID, ChannelID: rssChannelID, SavedSearchID: savedSearchID,
		Frequency: SavedSearchSnapshotTypeImmediate, Triggers: nil,
		ChannelType: nil,
	})
	if err != nil {
		t.Fatalf("failed to create sub 4: %v", err)
	}

	// Subscription 5: Disabled channel
	_, err = spannerClient.CreateSavedSearchSubscription(ctx, CreateSavedSearchSubscriptionRequest{
		UserID: userID, ChannelID: disabledChannelID, SavedSearchID: savedSearchID,
		Frequency:   SavedSearchSnapshotTypeImmediate,
		Triggers:    nil,
		ChannelType: nil,
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

	// Create 5 channels to avoid hitting channel limit.
	channelIDs := make([]string, 0, 5)
	for i := range 5 {
		channelReq := CreateNotificationChannelRequest{
			UserID: userID,
			Name:   fmt.Sprintf("Channel %d", i),
			Type:   NotificationChannelTypeEmail,
			EmailConfig: &EmailConfig{
				Address:           fmt.Sprintf("test%d@example.com", i),
				IsVerified:        true,
				VerificationToken: nil,
			},
			WebhookConfig: nil,
		}
		channelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, channelReq)
		if err != nil {
			t.Fatalf("failed to create channel %d: %v", i, err)
		}
		channelIDs = append(channelIDs, *channelIDPtr)
	}

	// Create 5 saved searches to avoid hitting saved search limit.
	savedSearchIDs := make([]string, 0, 5)
	for i := range 5 {
		savedSearchIDPtr, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
			Name:        fmt.Sprintf("Search %d", i),
			Query:       fmt.Sprintf("is:widely_%d", i),
			OwnerUserID: userID,
			Description: nil,
		})
		if err != nil {
			t.Fatalf("failed to create saved search %d: %v", i, err)
		}
		savedSearchIDs = append(savedSearchIDs, *savedSearchIDPtr)
	}

	// Create subscriptions up to the limit (25).
	count := 0
	for _, cid := range channelIDs {
		for _, ssid := range savedSearchIDs {
			if count >= 25 {
				break
			}
			_, err := spannerClient.CreateSavedSearchSubscription(ctx, CreateSavedSearchSubscriptionRequest{
				UserID:        userID,
				ChannelID:     cid,
				SavedSearchID: ssid,
				Triggers:      []SubscriptionTrigger{SubscriptionTriggerFeatureBaselineRegressionToLimited},
				Frequency:     SavedSearchSnapshotTypeImmediate,
				ChannelType:   nil,
			})
			if err != nil {
				t.Fatalf("failed to create subscription %d: %v", count, err)
			}
			count++
		}
	}

	// Try to create one more.
	// Create a 6th saved search to make it unique.
	savedSearchIDPtr, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "Search Limit",
		Query:       "is:widely_limit",
		OwnerUserID: userID,
		Description: nil,
	})
	if err != nil {
		t.Fatalf("failed to create saved search limit: %v", err)
	}
	_, err = spannerClient.CreateSavedSearchSubscription(ctx, CreateSavedSearchSubscriptionRequest{
		UserID:        userID,
		ChannelID:     channelIDs[0],
		SavedSearchID: *savedSearchIDPtr,
		Triggers:      []SubscriptionTrigger{SubscriptionTriggerFeatureBaselineRegressionToLimited},
		Frequency:     SavedSearchSnapshotTypeImmediate,
		ChannelType:   nil,
	})
	if !errors.Is(err, ErrSubscriptionLimitExceeded) {
		t.Errorf("expected ErrSubscriptionLimitExceeded, got %v", err)
	}
}

func TestCreateSavedSearchSubscriptionRSSResolution(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	userID := uuid.NewString()

	savedSearchIDPtr, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "Test Search 1",
		Query:       "is:widely",
		OwnerUserID: userID,
		Description: nil,
	})
	if err != nil {
		t.Fatalf("failed to create saved search 1: %v", err)
	}
	savedSearchID1 := *savedSearchIDPtr

	savedSearchID2Ptr, err := spannerClient.CreateNewUserSavedSearch(ctx, CreateUserSavedSearchRequest{
		Name:        "Test Search 2",
		Query:       "is:widely2",
		OwnerUserID: userID,
		Description: nil,
	})
	if err != nil {
		t.Fatalf("failed to create saved search 2: %v", err)
	}
	savedSearchID2 := *savedSearchID2Ptr

	// Create subscription with implicit RSS channel.
	rssType := NotificationChannelTypeRSS
	req1 := CreateSavedSearchSubscriptionRequest{
		UserID:        userID,
		ChannelID:     "",
		ChannelType:   &rssType,
		SavedSearchID: savedSearchID1,
		Triggers:      []SubscriptionTrigger{SubscriptionTriggerFeatureBaselineRegressionToLimited},
		Frequency:     SavedSearchSnapshotTypeImmediate,
	}

	subID1Ptr, err := spannerClient.CreateSavedSearchSubscription(ctx, req1)
	if err != nil {
		t.Fatalf("CreateSavedSearchSubscription failed for RSS: %v", err)
	}
	subID1 := *subID1Ptr

	// Verify that an RSS channel was created.
	sub1, err := spannerClient.GetSavedSearchSubscription(ctx, subID1, userID)
	if err != nil {
		t.Fatalf("GetSavedSearchSubscription failed: %v", err)
	}
	rssChannelID := sub1.ChannelID

	// Verify it is indeed an RSS channel.
	channel, err := spannerClient.GetNotificationChannel(ctx, rssChannelID, userID)
	if err != nil {
		t.Fatalf("GetNotificationChannel failed: %v", err)
	}
	if channel.Type != NotificationChannelTypeRSS {
		t.Errorf("expected channel type to be RSS, got %s", channel.Type)
	}

	// Create another subscription with implicit RSS channel.
	req2 := CreateSavedSearchSubscriptionRequest{
		UserID:        userID,
		ChannelID:     "",
		ChannelType:   &rssType,
		SavedSearchID: savedSearchID2,
		Triggers:      []SubscriptionTrigger{SubscriptionTriggerFeatureBaselineRegressionToLimited},
		Frequency:     SavedSearchSnapshotTypeImmediate,
	}

	subID2Ptr, err := spannerClient.CreateSavedSearchSubscription(ctx, req2)
	if err != nil {
		t.Fatalf("CreateSavedSearchSubscription failed for second RSS sub: %v", err)
	}
	subID2 := *subID2Ptr

	sub2, err := spannerClient.GetSavedSearchSubscription(ctx, subID2, userID)
	if err != nil {
		t.Fatalf("GetSavedSearchSubscription failed: %v", err)
	}

	if sub2.ChannelID != rssChannelID {
		t.Errorf("expected reused channel ID %s, got %s", rssChannelID, sub2.ChannelID)
	}
}

func TestCreateSavedSearchSubscriptionIdempotency(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	userID := uuid.NewString()

	// Pre-populate dependencies.
	channelReq := CreateNotificationChannelRequest{
		UserID:        userID,
		Name:          "Test",
		Type:          NotificationChannelTypeEmail,
		EmailConfig:   &EmailConfig{Address: "test@example.com", IsVerified: true, VerificationToken: nil},
		WebhookConfig: nil,
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
		Triggers: []SubscriptionTrigger{
			SubscriptionTriggerFeatureBaselineRegressionToLimited,
			SubscriptionTriggerFeatureBaselinePromoteToNewly,
		},
		Frequency:   SavedSearchSnapshotTypeImmediate,
		ChannelType: nil,
	}

	subIDPtr, err := spannerClient.CreateSavedSearchSubscription(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateSavedSearchSubscription failed: %v", err)
	}
	subID := *subIDPtr

	// 1. Idempotent recreation (identical).
	subID2Ptr, err := spannerClient.CreateSavedSearchSubscription(ctx, createReq)
	if err != nil {
		t.Fatalf("Idempotent recreation failed: %v", err)
	}
	if *subID2Ptr != subID {
		t.Errorf("expected same ID %s, got %s", subID, *subID2Ptr)
	}

	// 2. Idempotent recreation (shuffled triggers).
	createReqShuffled := createReq
	createReqShuffled.Triggers = []SubscriptionTrigger{
		SubscriptionTriggerFeatureBaselinePromoteToNewly,
		SubscriptionTriggerFeatureBaselineRegressionToLimited,
	}
	subID3Ptr, err := spannerClient.CreateSavedSearchSubscription(ctx, createReqShuffled)
	if err != nil {
		t.Fatalf("Idempotent recreation with shuffled triggers failed: %v", err)
	}
	if *subID3Ptr != subID {
		t.Errorf("expected same ID %s for shuffled triggers, got %s", subID, *subID3Ptr)
	}

	// 3. Conflict (different frequency).
	createReqConflict := createReq
	createReqConflict.Frequency = SavedSearchSnapshotTypeWeekly
	_, err = spannerClient.CreateSavedSearchSubscription(ctx, createReqConflict)
	if !errors.Is(err, ErrSubscriptionConflict) {
		t.Errorf("expected ErrSubscriptionConflict, got %v", err)
	}
}
