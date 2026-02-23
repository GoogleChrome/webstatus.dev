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
	"time"

	"cloud.google.com/go/spanner"
)

// notificationChannelWithState is a struct to hold the result of the JOIN query
// between NotificationChannels and NotificationChannelStates.
type notificationChannelWithState struct {
	ID                  string
	UserID              string
	Name                string
	Type                NotificationChannelType
	Config              spanner.NullJSON
	CreatedAt           time.Time
	UpdatedAt           time.Time
	IsDisabledBySystem  spanner.NullBool
	ConsecutiveFailures spanner.NullInt64
	StateCreatedAt      spanner.NullTime
	StateUpdatedAt      spanner.NullTime
}

func getExistingEmailChannels(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	userID string,
) (map[string]NotificationChannel, map[string]NotificationChannelState, error) {
	existingChannels := make(map[string]NotificationChannel)
	existingChannelStates := make(map[string]NotificationChannelState)

	stmt := spanner.Statement{
		SQL: `SELECT nc.ID, nc.UserID, nc.Name, nc.Type, nc.Config, nc.CreatedAt, nc.UpdatedAt,
                                ncs.IsDisabledBySystem, ncs.ConsecutiveFailures,
                                ncs.CreatedAt AS StateCreatedAt, ncs.UpdatedAt AS StateUpdatedAt
                        FROM NotificationChannels AS nc
                        LEFT JOIN NotificationChannelStates AS ncs ON nc.ID = ncs.ChannelID
                        WHERE nc.UserID = @userID AND nc.Type = "email"`,
		Params: map[string]interface{}{
			"userID": userID,
		},
	}
	it := txn.Query(ctx, stmt)
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
			existingChannels[channel.EmailConfig.Address] = *channel
			existingChannelStates[channel.ID] = state
		}

		return nil
	})
	if err != nil {
		return nil, nil, errors.Join(ErrInternalQueryFailure, err)
	}

	return existingChannels, existingChannelStates, nil
}

func generateChannelMutations(
	ctx context.Context,
	c *Client,
	txn *spanner.ReadWriteTransaction,
	userProfile UserProfile,
	existingChannels map[string]NotificationChannel,
	existingChannelStates map[string]NotificationChannelState,
) (map[string]struct{}, error) {
	newEmailsMap := make(map[string]struct{})
	channelCreator := newEntityCreator[notificationChannelMapper](c)
	stateWriter := newEntityWriter[notificationChannelStateMapper](c)

	for _, email := range userProfile.Emails {
		newEmailsMap[email] = struct{}{}

		if _, exists := existingChannels[email]; exists {
			// Existing email: check if state needs to be enabled
			// In the future, we should leave it disabled if we marked it as disabled.
			// TODO: https://github.com/GoogleChrome/webstatus.dev/issues/2021
			existingState := existingChannelStates[existingChannels[email].ID]
			if existingState.IsDisabledBySystem {
				existingState.IsDisabledBySystem = false
				existingState.UpdatedAt = spanner.CommitTimestamp
				_, err := stateWriter.updateWithTransaction(ctx, txn, existingState)
				if err != nil {
					return nil, err
				}
			}

			continue
		}

		// New email: create NotificationChannel and NotificationChannelState
		req := CreateNotificationChannelRequest{
			UserID:        userProfile.UserID,
			Name:          email,
			Type:          NotificationChannelTypeEmail,
			EmailConfig:   &EmailConfig{Address: email, IsVerified: true, VerificationToken: nil},
			WebhookConfig: nil,
		}

		channelID, err := channelCreator.createWithTransaction(ctx, txn, req)

		if err != nil {
			return nil, err
		}

		newChannelState := NotificationChannelState{
			ChannelID:           *channelID,
			IsDisabledBySystem:  false,
			ConsecutiveFailures: 0,
			CreatedAt:           spanner.CommitTimestamp,
			UpdatedAt:           spanner.CommitTimestamp,
		}
		_, err = stateWriter.upsertWithTransaction(ctx, txn, newChannelState)
		if err != nil {
			return nil, err
		}
	}

	return newEmailsMap, nil
}

func generateDisableChannelMutations(
	ctx context.Context,
	c *Client,
	txn *spanner.ReadWriteTransaction,
	existingChannels map[string]NotificationChannel,
	existingChannelStates map[string]NotificationChannelState,
	newEmailsMap map[string]struct{},
) error {
	stateWriter := newEntityWriter[notificationChannelStateMapper](c)

	for email, existingChannel := range existingChannels {
		if _, existsInNewProfile := newEmailsMap[email]; !existsInNewProfile {
			existingState := existingChannelStates[existingChannel.ID]
			if !existingState.IsDisabledBySystem {
				existingState.IsDisabledBySystem = true
				existingState.UpdatedAt = spanner.CommitTimestamp
				_, err := stateWriter.updateWithTransaction(ctx, txn, existingState)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// UserProfile represents a user's profile information.
type UserProfile struct {
	UserID       string
	GitHubUserID int64
	Emails       []string
}

func (c *Client) SyncUserProfileInfo(
	ctx context.Context, userProfile UserProfile) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		existingChannels, existingChannelStates, err := getExistingEmailChannels(ctx, txn, userProfile.UserID)
		if err != nil {
			return err
		}

		newEmailsMap, err := generateChannelMutations(ctx, c, txn, userProfile, existingChannels, existingChannelStates)
		if err != nil {
			return err
		}

		return generateDisableChannelMutations(ctx, c, txn, existingChannels, existingChannelStates, newEmailsMap)
	})

	return err
}
