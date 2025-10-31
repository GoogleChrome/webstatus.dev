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

func (m notificationChannelStateMapper) Table() string {
	return notificationChannelStateTable
}

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
