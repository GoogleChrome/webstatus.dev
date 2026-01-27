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
	"time"

	"cloud.google.com/go/spanner"
)

const notificationChannelStateTable = "NotificationChannelStates"

// NotificationChannelState represents a row in the NotificationChannelState table.
type NotificationChannelState struct {
	ChannelID           string    `spanner:"ChannelID"`
	IsDisabledBySystem  bool      `spanner:"IsDisabledBySystem"`
	ConsecutiveFailures int64     `spanner:"ConsecutiveFailures"`
	CreatedAt           time.Time `spanner:"CreatedAt"`
	UpdatedAt           time.Time `spanner:"UpdatedAt"`
}

// notificationChannelStateMapper implements the necessary interfaces for the generic helpers.
type notificationChannelStateMapper struct{}

func (m notificationChannelStateMapper) Table() string { return notificationChannelStateTable }

func (m notificationChannelStateMapper) GetKeyFromExternal(in NotificationChannelState) string {
	return in.ChannelID
}

func (m notificationChannelStateMapper) SelectOne(key string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ChannelID, IsDisabledBySystem, ConsecutiveFailures, CreatedAt, UpdatedAt
	FROM %s
	WHERE ChannelID = @channelId
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"channelId": key,
	}
	stmt.Params = parameters

	return stmt
}

func (m notificationChannelStateMapper) Merge(
	in NotificationChannelState, existing NotificationChannelState) NotificationChannelState {
	existing.IsDisabledBySystem = in.IsDisabledBySystem
	existing.ConsecutiveFailures = in.ConsecutiveFailures
	existing.UpdatedAt = in.UpdatedAt

	return existing
}

// UpsertNotificationChannelState inserts or updates a notification channel state.
func (c *Client) UpsertNotificationChannelState(
	ctx context.Context, state NotificationChannelState) error {
	return newEntityWriter[notificationChannelStateMapper](c).upsert(ctx, state)
}

// GetNotificationChannelState retrieves a notification channel state by its ID.
func (c *Client) GetNotificationChannelState(
	ctx context.Context, channelID string) (*NotificationChannelState, error) {
	return newEntityReader[notificationChannelStateMapper,
		NotificationChannelState, string](c).readRowByKey(ctx, channelID)
}

// RecordNotificationChannelSuccess resets the consecutive failures count in the NotificationChannelStates table
// and logs a successful delivery attempt in the NotificationChannelDeliveryAttempts table.
func (c *Client) RecordNotificationChannelSuccess(
	ctx context.Context, channelID string, timestamp time.Time, eventID string) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Update NotificationChannelStates
		_, err := newEntityWriter[notificationChannelStateMapper](c).upsertWithTransaction(ctx, txn,
			NotificationChannelState{
				ChannelID:           channelID,
				IsDisabledBySystem:  false,
				ConsecutiveFailures: 0,
				CreatedAt:           timestamp,
				UpdatedAt:           timestamp,
			})
		if err != nil {
			return err
		}

		_, err = c.createNotificationChannelDeliveryAttemptWithTransaction(ctx, txn,
			CreateNotificationChannelDeliveryAttemptRequest{
				ChannelID:        channelID,
				AttemptTimestamp: timestamp,
				Status:           DeliveryAttemptStatusSuccess,
				Details: spanner.NullJSON{Value: AttemptDetails{
					EventID: eventID,
					Message: "delivered"}, Valid: true},
			})

		return err
	})

	return err

}

// RecordNotificationChannelFailure increments the consecutive failures count in the NotificationChannelStates table
// and logs a failure delivery attempt in the NotificationChannelDeliveryAttempts table.
// If isPermanent is true, it increments the failure count and potentially disables the channel.
// If isPermanent is false (transient), it logs the error but does not penalize the channel health.
func (c *Client) RecordNotificationChannelFailure(
	ctx context.Context, channelID string, errorMsg string, timestamp time.Time,
	isPermanent bool, eventID string) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Read current state
		state, err := newEntityReader[notificationChannelStateMapper, NotificationChannelState, string](c).
			readRowByKeyWithTransaction(ctx, channelID, txn)
		if err != nil && !errors.Is(err, ErrQueryReturnedNoResults) {
			return err
		} else if errors.Is(err, ErrQueryReturnedNoResults) {
			state = &NotificationChannelState{
				ChannelID:           channelID,
				CreatedAt:           timestamp,
				UpdatedAt:           timestamp,
				IsDisabledBySystem:  false,
				ConsecutiveFailures: 0,
			}
		}

		// Calculate new state
		if isPermanent {
			state.ConsecutiveFailures++
		}
		state.UpdatedAt = timestamp
		state.IsDisabledBySystem = state.ConsecutiveFailures >= int64(
			c.notificationCfg.maxConsecutiveFailuresPerChannel)

		// Update NotificationChannelStates
		_, err = newEntityWriter[notificationChannelStateMapper](c).upsertWithTransaction(ctx,
			txn, NotificationChannelState{
				ChannelID:           channelID,
				IsDisabledBySystem:  state.IsDisabledBySystem,
				ConsecutiveFailures: state.ConsecutiveFailures,
				CreatedAt:           state.CreatedAt,
				UpdatedAt:           state.UpdatedAt,
			})
		if err != nil {
			return err
		}

		// Log attempt
		_, err = c.createNotificationChannelDeliveryAttemptWithTransaction(ctx, txn,
			CreateNotificationChannelDeliveryAttemptRequest{
				ChannelID:        channelID,
				AttemptTimestamp: timestamp,
				Status:           DeliveryAttemptStatusFailure,
				Details: spanner.NullJSON{Value: AttemptDetails{
					EventID: eventID,
					Message: errorMsg}, Valid: true},
			})

		return err

	})

	return err
}

type AttemptDetails struct {
	Message string `json:"message"`
	EventID string `json:"event_id"`
}
