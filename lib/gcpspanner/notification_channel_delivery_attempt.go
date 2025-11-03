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

const notificationChannelDeliveryAttemptTable = "NotificationChannelDeliveryAttempts"
const maxDeliveryAttemptsToKeep = 10

// NotificationChannelDeliveryAttempt represents a row in the NotificationChannelDeliveryAttempt table.
type NotificationChannelDeliveryAttempt struct {
	ID               string                                   `spanner:"ID"`
	ChannelID        string                                   `spanner:"ChannelID"`
	AttemptTimestamp time.Time                                `spanner:"AttemptTimestamp"`
	Status           NotificationChannelDeliveryAttemptStatus `spanner:"Status"`
	Details          spanner.NullJSON                         `spanner:"Details"`
}

type NotificationChannelDeliveryAttemptStatus string

const (
	// DeliveryAttemptStatusSuccess indicates that the delivery attempt was successful.
	DeliveryAttemptStatusSuccess NotificationChannelDeliveryAttemptStatus = "SUCCESS"
	// DeliveryAttemptStatusFailure indicates that the delivery attempt failed.
	DeliveryAttemptStatusFailure NotificationChannelDeliveryAttemptStatus = "FAILURE"
)

// CreateNotificationChannelDeliveryAttemptRequest is the request to create a delivery attempt.
type CreateNotificationChannelDeliveryAttemptRequest struct {
	ChannelID        string
	AttemptTimestamp time.Time
	Status           NotificationChannelDeliveryAttemptStatus
	Details          spanner.NullJSON
}

// ListNotificationChannelDeliveryAttemptsRequest is the request struct for listing delivery attempts.
type ListNotificationChannelDeliveryAttemptsRequest struct {
	ChannelID string
	PageSize  int
	PageToken *string
}

// GetPageSize returns the page size for the request.
func (r ListNotificationChannelDeliveryAttemptsRequest) GetPageSize() int {
	return r.PageSize
}

type notificationChannelDeliveryAttemptMapper struct{}

func (m notificationChannelDeliveryAttemptMapper) Table() string {
	return notificationChannelDeliveryAttemptTable
}

func (m notificationChannelDeliveryAttemptMapper) NewEntity(
	id string,
	req CreateNotificationChannelDeliveryAttemptRequest) (NotificationChannelDeliveryAttempt, error) {
	return NotificationChannelDeliveryAttempt{
		ID:               id,
		ChannelID:        req.ChannelID,
		AttemptTimestamp: req.AttemptTimestamp,
		Status:           req.Status,
		Details:          req.Details,
	}, nil
}

type deliveryAttemptKey struct {
	ID        string
	ChannelID string
}

func (m notificationChannelDeliveryAttemptMapper) SelectOne(key deliveryAttemptKey) spanner.Statement {
	stmt := spanner.NewStatement(`
		SELECT ID, ChannelID, AttemptTimestamp, Status, Details
		FROM NotificationChannelDeliveryAttempts
		WHERE ID = @id AND ChannelID = @channelID`)
	stmt.Params = map[string]interface{}{
		"id":        key.ID,
		"channelID": key.ChannelID,
	}

	return stmt
}

func (m notificationChannelDeliveryAttemptMapper) SelectList(
	req ListNotificationChannelDeliveryAttemptsRequest) spanner.Statement {
	var pageFilter string
	params := map[string]interface{}{
		"channelID": req.ChannelID,
		"pageSize":  req.PageSize,
	}
	if req.PageToken != nil {
		cursor, err := decodeCursor[notificationChannelDeliveryAttemptCursor](*req.PageToken)
		if err == nil {
			params["lastID"] = cursor.LastID
			pageFilter = " AND ID > @lastID"
		}
	}
	stmt := spanner.NewStatement(fmt.Sprintf(`
		SELECT ID, ChannelID, AttemptTimestamp, Status, Details
		FROM NotificationChannelDeliveryAttempts
		WHERE ChannelID = @channelID %s
		ORDER BY AttemptTimestamp, ID
		LIMIT @pageSize`, pageFilter))
	stmt.Params = params

	return stmt
}

type notificationChannelDeliveryAttemptCursor struct {
	LastID string `json:"last_id"`
}

// EncodePageToken returns the ID of the delivery attempt as a page token.
func (m notificationChannelDeliveryAttemptMapper) EncodePageToken(item NotificationChannelDeliveryAttempt) string {
	return encodeCursor(notificationChannelDeliveryAttemptCursor{
		LastID: item.ID,
	})
}

// CreateNotificationChannelDeliveryAttempt creates a new delivery attempt log, prunes old ones, and returns its ID.
func (c *Client) CreateNotificationChannelDeliveryAttempt(
	ctx context.Context, req CreateNotificationChannelDeliveryAttemptRequest) (*string, error) {
	var newID *string
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Create the new attempt
		id, err := newEntityCreator[notificationChannelDeliveryAttemptMapper](c).createWithTransaction(ctx, txn, req)
		if err != nil {
			return err
		}
		newID = id

		// 2. Count existing attempts for the channel. Note: This count does not include the new attempt just buffered.
		countStmt := spanner.NewStatement(`
			SELECT COUNT(*)
			FROM NotificationChannelDeliveryAttempts
			WHERE ChannelID = @channelID`)
		countStmt.Params["channelID"] = req.ChannelID
		var count int64
		err = txn.Query(ctx, countStmt).Do(func(r *spanner.Row) error {
			return r.Column(0, &count)
		})
		if err != nil {
			return err
		}

		// 3. If the pre-insert count is at the limit, fetch the oldest attempts to delete.
		if count >= maxDeliveryAttemptsToKeep {
			// We need to delete enough to make room for the one we are adding.
			deleteCount := count - maxDeliveryAttemptsToKeep + 1
			deleteStmt := spanner.NewStatement(`
				SELECT ID
				FROM NotificationChannelDeliveryAttempts
				WHERE ChannelID = @channelID
				ORDER BY AttemptTimestamp ASC
				LIMIT @deleteCount`)
			deleteStmt.Params["channelID"] = req.ChannelID
			deleteStmt.Params["deleteCount"] = deleteCount

			var mutations []*spanner.Mutation
			err := txn.Query(ctx, deleteStmt).Do(func(r *spanner.Row) error {
				var attemptID string
				if err := r.Column(0, &attemptID); err != nil {
					return err
				}
				mutations = append(mutations,
					spanner.Delete(notificationChannelDeliveryAttemptTable,
						spanner.Key{attemptID, req.ChannelID}))

				return nil
			})
			if err != nil {
				return err
			}

			// 4. Buffer delete mutations
			if len(mutations) > 0 {
				return txn.BufferWrite(mutations)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return newID, nil
}

// GetNotificationChannelDeliveryAttempt retrieves a single delivery attempt.
func (c *Client) GetNotificationChannelDeliveryAttempt(
	ctx context.Context, attemptID string, channelID string) (*NotificationChannelDeliveryAttempt, error) {
	key := deliveryAttemptKey{ID: attemptID, ChannelID: channelID}

	return newEntityReader[notificationChannelDeliveryAttemptMapper,
		NotificationChannelDeliveryAttempt, deliveryAttemptKey](c).readRowByKey(ctx, key)
}

// ListNotificationChannelDeliveryAttempts lists all delivery attempts for a channel.
func (c *Client) ListNotificationChannelDeliveryAttempts(
	ctx context.Context,
	req ListNotificationChannelDeliveryAttemptsRequest,
) ([]NotificationChannelDeliveryAttempt, *string, error) {
	return newEntityLister[notificationChannelDeliveryAttemptMapper](c).list(ctx, req)
}
