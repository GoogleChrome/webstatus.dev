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

package spanneradapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/google/go-cmp/cmp"
)

type mockPushDeliverySpannerClient struct {
	findAllCalledWith      *findAllCalledWith
	findAllReturns         findAllReturns
	listCodeSubsCalledWith *listCodeSubsCalledWith
	listCodeSubsReturns    listCodeSubsReturns
}

type findAllCalledWith struct {
	SearchID  string
	Frequency gcpspanner.SavedSearchSnapshotType
}

type findAllReturns struct {
	dests []gcpspanner.SubscriberDestination
	err   error
}

type listCodeSubsCalledWith struct {
	TargetQuery string
	Trigger     gcpspanner.SubscriptionTrigger
}

type listCodeSubsReturns struct {
	subs []gcpspanner.CodeSubscription
	err  error
}

func (m *mockPushDeliverySpannerClient) FindAllActivePushSubscriptions(
	_ context.Context,
	savedSearchID string,
	frequency gcpspanner.SavedSearchSnapshotType,
) ([]gcpspanner.SubscriberDestination, error) {
	m.findAllCalledWith = &findAllCalledWith{
		SearchID:  savedSearchID,
		Frequency: frequency,
	}

	return m.findAllReturns.dests, m.findAllReturns.err
}

func (m *mockPushDeliverySpannerClient) ListCodeSubscriptionsByTargetQuery(
	_ context.Context,
	targetQuery string,
	trigger gcpspanner.SubscriptionTrigger,
) ([]gcpspanner.CodeSubscription, error) {
	m.listCodeSubsCalledWith = &listCodeSubsCalledWith{
		TargetQuery: targetQuery,
		Trigger:     trigger,
	}

	return m.listCodeSubsReturns.subs, m.listCodeSubsReturns.err
}

func TestFindSubscribers(t *testing.T) {
	tests := []struct {
		name          string
		dests         []gcpspanner.SubscriberDestination
		clientErr     error
		expectedCall  *findAllCalledWith
		expectedSet   *workertypes.SubscriberSet
		expectedError error
	}{
		{
			name: "Success with Email and Triggers",
			expectedCall: &findAllCalledWith{
				SearchID:  "search-1",
				Frequency: gcpspanner.SavedSearchSnapshotTypeImmediate,
			},
			dests: []gcpspanner.SubscriberDestination{
				{
					SubscriptionID: "sub-1",
					UserID:         "user-1",
					Type:           gcpspanner.NotificationChannelTypeEmail,
					ChannelID:      "chan-1",
					EmailConfig: &gcpspanner.EmailConfig{
						Address:           "test@example.com",
						IsVerified:        true,
						VerificationToken: nil,
					},
					WebhookConfig: nil,
					Triggers: []gcpspanner.SubscriptionTrigger{
						gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToNewly,
						gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToWidely,
					},
				},
			},
			expectedSet: &workertypes.SubscriberSet{
				Emails: []workertypes.EmailSubscriber{
					{
						SubscriptionID: "sub-1",
						UserID:         "user-1",
						EmailAddress:   "test@example.com",
						Triggers: []workertypes.JobTrigger{
							workertypes.FeaturePromotedToNewly,
							workertypes.FeaturePromotedToWidely,
						},
						ChannelID: "chan-1",
					},
				},
				Webhooks: []workertypes.WebhookSubscriber{},
			},
			clientErr:     nil,
			expectedError: nil,
		},
		{
			name: "Mixed types (Webhook ignored)",
			expectedCall: &findAllCalledWith{
				SearchID:  "search-1",
				Frequency: gcpspanner.SavedSearchSnapshotTypeImmediate,
			},
			clientErr: nil,
			dests: []gcpspanner.SubscriberDestination{
				{
					UserID:         "user-1",
					SubscriptionID: "sub-1",
					Type:           gcpspanner.NotificationChannelTypeEmail,
					ChannelID:      "chan-1",
					EmailConfig: &gcpspanner.EmailConfig{
						Address:           "test@example.com",
						IsVerified:        true,
						VerificationToken: nil,
					},
					WebhookConfig: nil,
					Triggers: []gcpspanner.SubscriptionTrigger{
						gcpspanner.SubscriptionTriggerBrowserImplementationAnyComplete,
					},
				},
				{
					SubscriptionID: "sub-2",
					Type:           "WEBHOOK",
					ChannelID:      "chan-2",
					EmailConfig:    nil, // Webhooks don't have EmailConfig
					WebhookConfig:  nil,
					Triggers:       nil,
					UserID:         "user-3",
				},
			},
			expectedSet: &workertypes.SubscriberSet{
				Emails: []workertypes.EmailSubscriber{
					{
						SubscriptionID: "sub-1",
						UserID:         "user-1",
						EmailAddress:   "test@example.com",
						Triggers: []workertypes.JobTrigger{
							workertypes.BrowserImplementationAnyComplete,
						},
						ChannelID: "chan-1",
					},
				},
				Webhooks: []workertypes.WebhookSubscriber{},
			},
			expectedError: nil,
		},
		{
			name: "Client Error",
			expectedCall: &findAllCalledWith{
				SearchID:  "search-1",
				Frequency: gcpspanner.SavedSearchSnapshotTypeImmediate,
			},
			expectedSet:   nil,
			dests:         nil,
			clientErr:     errTest,
			expectedError: errTest,
		},
		{
			name: "Nil Email Config (Should Skip)",
			expectedCall: &findAllCalledWith{
				SearchID:  "search-1",
				Frequency: gcpspanner.SavedSearchSnapshotTypeImmediate,
			},
			dests: []gcpspanner.SubscriberDestination{
				{
					UserID:         "user-1",
					SubscriptionID: "sub-1",
					Type:           gcpspanner.NotificationChannelTypeEmail,
					ChannelID:      "chan-1",
					Triggers:       nil,
					EmailConfig:    nil, // Missing config should be skipped
					WebhookConfig:  nil,
				},
			},
			clientErr: nil,
			expectedSet: &workertypes.SubscriberSet{
				Emails:   []workertypes.EmailSubscriber{},
				Webhooks: []workertypes.WebhookSubscriber{},
			},
			expectedError: nil,
		},
		{
			name: "Unknown Trigger (Should be logged/ignored/empty string)",
			expectedCall: &findAllCalledWith{
				SearchID:  "search-1",
				Frequency: gcpspanner.SavedSearchSnapshotTypeImmediate,
			},
			dests: []gcpspanner.SubscriberDestination{
				{
					UserID:         "user-1",
					SubscriptionID: "sub-1",
					Type:           gcpspanner.NotificationChannelTypeEmail,
					ChannelID:      "chan-1",
					EmailConfig: &gcpspanner.EmailConfig{
						Address:           "test@example.com",
						IsVerified:        true,
						VerificationToken: nil,
					},
					WebhookConfig: nil,
					Triggers: []gcpspanner.SubscriptionTrigger{
						"some_unknown_trigger",
					},
				},
			},
			clientErr: nil,
			expectedSet: &workertypes.SubscriberSet{
				Emails: []workertypes.EmailSubscriber{
					{
						UserID:         "user-1",
						SubscriptionID: "sub-1",
						EmailAddress:   "test@example.com",
						Triggers: []workertypes.JobTrigger{
							"", // Unknown triggers map to empty string/zero value in current implementation
						},
						ChannelID: "chan-1",
					},
				},
				Webhooks: []workertypes.WebhookSubscriber{},
			},
			expectedError: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := new(mockPushDeliverySpannerClient)
			mock.findAllReturns.dests = tc.dests
			mock.findAllReturns.err = tc.clientErr

			finder := NewPushDeliverySubscriberFinder(mock)
			set, err := finder.FindSubscribers(context.Background(), "search-1", workertypes.FrequencyImmediate)

			if !errors.Is(err, tc.expectedError) {
				t.Errorf("FindSubscribers error = %v, wantErr %v", err, tc.expectedError)
			}

			if diff := cmp.Diff(tc.expectedSet, set); diff != "" {
				t.Errorf("SubscriberSet mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.expectedCall, mock.findAllCalledWith); diff != "" {
				t.Errorf("findAllCalledWith mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindCodeSubscriptions(t *testing.T) {
	errTest := errors.New("spanner failure")
	expectedSubs := []gcpspanner.CodeSubscription{
		{
			ID:                 "sub-1",
			VCSProvider:        "github",
			VCSInstallationID:  "inst-1",
			VCSRepositoryID:    "repo-1",
			RepositoryFullName: "owner/repo",
			TargetQuery:        "id:popover",
			Triggers: []gcpspanner.SubscriptionTrigger{
				gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToWidely,
			},
			Status:      gcpspanner.SubscriptionActive,
			Occurrences: nil,
			CreatedAt:   time.Time{},
			UpdatedAt:   time.Time{},
		},
	}

	tests := []struct {
		name         string
		subs         []gcpspanner.CodeSubscription
		clientErr    error
		targetQuery  string
		trigger      gcpspanner.SubscriptionTrigger
		expectedSubs []gcpspanner.CodeSubscription
		expectedErr  error
	}{
		{
			name:         "success",
			subs:         expectedSubs,
			clientErr:    nil,
			targetQuery:  "id:popover",
			trigger:      gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToWidely,
			expectedSubs: expectedSubs,
			expectedErr:  nil,
		},
		{
			name:         "error propagation",
			subs:         nil,
			clientErr:    errTest,
			targetQuery:  "id:popover",
			trigger:      gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToWidely,
			expectedSubs: nil,
			expectedErr:  errTest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := new(mockPushDeliverySpannerClient)
			mock.listCodeSubsReturns.subs = tc.subs
			mock.listCodeSubsReturns.err = tc.clientErr

			finder := NewPushDeliverySubscriberFinder(mock)
			subs, err := finder.FindCodeSubscriptions(context.Background(), tc.targetQuery, tc.trigger)

			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("FindCodeSubscriptions error = %v, wantErr %v", err, tc.expectedErr)
			}

			if diff := cmp.Diff(tc.expectedSubs, subs); diff != "" {
				t.Errorf("subs mismatch (-want +got):\n%s", diff)
			}

			if mock.listCodeSubsCalledWith.TargetQuery != tc.targetQuery {
				t.Errorf("TargetQuery = %v, want %v", mock.listCodeSubsCalledWith.TargetQuery, tc.targetQuery)
			}
			if mock.listCodeSubsCalledWith.Trigger != tc.trigger {
				t.Errorf("Trigger = %v, want %v", mock.listCodeSubsCalledWith.Trigger, tc.trigger)
			}
		})
	}
}
