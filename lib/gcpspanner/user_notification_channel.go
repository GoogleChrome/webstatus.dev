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
	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/google/uuid"
)
func (c *Client) SyncUserProfileInfo(
	ctx context.Context, userProfile backendtypes.UserProfile) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Get existing notification channels and their states for the user.
		existingChannels := make(map[string]NotificationChannel) // Key: email address
		existingChannelStates := make(map[string]NotificationChannelState) // Key: ChannelID

		stmt := spanner.Statement{
			SQL: `SELECT nc.ID, nc.UserID, nc.Name, nc.Type, nc.Config, nc.CreatedAt, nc.UpdatedAt,
					ncs.IsDisabledBySystem, ncs.ConsecutiveFailures, ncs.CreatedAt AS StateCreatedAt, ncs.UpdatedAt AS StateUpdatedAt
				FROM NotificationChannels AS nc
				LEFT JOIN NotificationChannelStates AS ncs ON nc.ID = ncs.ChannelID
				WHERE nc.UserID = @userID AND nc.Type = "email"`,
			Params: map[string]interface{}{
				"userID": userProfile.UserID,
			},
		}
		it := txn.Query(ctx, stmt)
		defer it.Stop()

		err := it.Do(func(r *spanner.Row) error {
			var spannerChannel spannerNotificationChannel
			var state NotificationChannelState
			var stateUpdatedAt spanner.NullTime

			// Explicitly scan NotificationChannel fields
			if err := r.ColumnByName("ID", &spannerChannel.ID); err != nil { return err }
			if err := r.ColumnByName("UserID", &spannerChannel.UserID); err != nil { return err }
			if err := r.ColumnByName("Name", &spannerChannel.Name); err != nil { return err }
			if err := r.ColumnByName("Type", &spannerChannel.Type); err != nil { return err }
			if err := r.ColumnByName("Config", &spannerChannel.Config); err != nil { return err }
			if err := r.ColumnByName("CreatedAt", &spannerChannel.CreatedAt); err != nil { return err }
			if err := r.ColumnByName("UpdatedAt", &spannerChannel.UpdatedAt); err != nil { return err }

			// Manually scan NotificationChannelState fields (which might be NULL from LEFT JOIN)
			var isDisabledBySystem spanner.NullBool
			var consecutiveFailures spanner.NullInt64
			var stateCreatedAt spanner.NullTime

			if err := r.ColumnByName("IsDisabledBySystem", &isDisabledBySystem); err != nil { return err }
			if err := r.ColumnByName("ConsecutiveFailures", &consecutiveFailures); err != nil { return err }
			if err := r.ColumnByName("CreatedAt", &stateCreatedAt); err != nil { return err } // This is ncs.CreatedAt
			if err := r.ColumnByName("StateUpdatedAt", &stateUpdatedAt); err != nil { return err }

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
				existingChannels[channel.EmailConfig.Address] = *channel
				existingChannelStates[channel.ID] = state
			}
			return nil
		})
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}

		var mutations []*spanner.Mutation
		now := time.Now().UTC()

		newEmailsMap := make(map[string]struct{})
		for _, email := range userProfile.Emails {
			newEmailsMap[email] = struct{}{}

			if existingChannel, exists := existingChannels[email]; !exists {
				// New email: create NotificationChannel and NotificationChannelState
				channelID := uuid.New().String()
				newChannel := NotificationChannel{
					ID:        channelID,
					UserID:    userProfile.UserID,
					Name:      email, // Using email as name for simplicity
					Type:      "email",
					EmailConfig: &EmailConfig{Address: email, IsVerified: true}, // Assume verified for now
					CreatedAt: now,
					UpdatedAt: now,
				}
				m, err := spanner.InsertStruct("NotificationChannels", newChannel.toSpanner())
				if err != nil {
					return errors.Join(ErrInternalMutationFailure, err)
				}
				mutations = append(mutations, m)

				newChannelState := NotificationChannelState{
					ChannelID:           channelID,
					IsDisabledBySystem:  false,
					ConsecutiveFailures: 0,
					CreatedAt:           now,
					UpdatedAt:           now,
				}
				m, err = spanner.InsertStruct("NotificationChannelStates", newChannelState)
				if err != nil {
					return errors.Join(ErrInternalMutationFailure, err)
				}
				mutations = append(mutations, m)
			} else {
				// Existing email: check if state needs to be enabled
				existingState := existingChannelStates[existingChannel.ID]
				if existingState.IsDisabledBySystem {
					existingState.IsDisabledBySystem = false
					existingState.UpdatedAt = now
					m, err := spanner.UpdateStruct("NotificationChannelStates", existingState)
					if err != nil {
						return errors.Join(ErrInternalMutationFailure, err)
					}
					mutations = append(mutations, m)
				}
			}
		}

		// Disable channels that are no longer in the user profile
		for email, existingChannel := range existingChannels {
			if _, existsInNewProfile := newEmailsMap[email]; !existsInNewProfile {
				existingState := existingChannelStates[existingChannel.ID]
				if !existingState.IsDisabledBySystem {
					existingState.IsDisabledBySystem = true
					existingState.UpdatedAt = now
					m, err := spanner.UpdateStruct("NotificationChannelStates", existingState)
					if err != nil {
						return errors.Join(ErrInternalMutationFailure, err)
					}
					mutations = append(mutations, m)
				}
			}
		}

		if len(mutations) > 0 {
			return txn.BufferWrite(mutations)
		}

		return nil
	})

	return err
}