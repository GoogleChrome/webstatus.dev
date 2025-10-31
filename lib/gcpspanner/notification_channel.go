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
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
)

const notificationChannelTable = "NotificationChannels"

// EmailConfig represents the JSON structure for an EMAIL notification channel.
type EmailConfig struct {
	Address           string  `json:"address,omitempty"`
	IsVerified        bool    `json:"is_verified,omitempty"`
	VerificationToken *string `json:"verification_token,omitempty"`
}

// NotificationChannel represents a user-facing notification channel.
type NotificationChannel struct {
	ID          string       `spanner:"ID"`
	UserID      string       `spanner:"UserID"`
	Name        string       `spanner:"Name"`
	Type        string       `spanner:"Type"`
	EmailConfig *EmailConfig `spanner:"-"`
	CreatedAt   time.Time    `spanner:"CreatedAt"`
	UpdatedAt   time.Time    `spanner:"UpdatedAt"`
}

// spannerNotificationChannel is the internal struct for Spanner mapping.
type spannerNotificationChannel struct {
	NotificationChannel
	Config spanner.NullJSON `spanner:"Config"`
}

// CreateNotificationChannelRequest is the request to create a channel.
type CreateNotificationChannelRequest struct {
	UserID      string
	Name        string
	Type        string
	EmailConfig *EmailConfig
}

// UpdateNotificationChannelRequest is a request to update a notification channel.
type UpdateNotificationChannelRequest struct {
	ID          string
	UserID      string
	Name        OptionallySet[string]
	EmailConfig OptionallySet[*EmailConfig]
}

// notificationChannelMapper implements the necessary interfaces for the generic helpers.
type notificationChannelMapper struct{}

type notificationChannelKey struct {
	ID     string
	UserID string
}

func (m notificationChannelMapper) Table() string {
	return notificationChannelTable
}

func (m notificationChannelMapper) GetKeyFromExternal(
	in UpdateNotificationChannelRequest) notificationChannelKey {
	return notificationChannelKey{
		ID:     in.ID,
		UserID: in.UserID,
	}
}

func (m notificationChannelMapper) SelectOne(key notificationChannelKey) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID, UserID, Name, Type, Config, CreatedAt, UpdatedAt
	FROM %s
	WHERE ID = @id AND UserID = @userId
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"id":     key.ID,
		"userId": key.UserID,
	}
	stmt.Params = parameters

	return stmt
}

func (m notificationChannelMapper) Merge(
	req UpdateNotificationChannelRequest, existing spannerNotificationChannel) (spannerNotificationChannel, error) {
	if req.Name.IsSet {
		existing.Name = req.Name.Value
	}
	if req.EmailConfig.IsSet && req.EmailConfig.Value != nil {
		existing.Config = spanner.NullJSON{Value: *req.EmailConfig.Value, Valid: true}
	}

	return existing, nil
}

func (m notificationChannelMapper) NewEntity(
	id string,
	req CreateNotificationChannelRequest) (*spannerNotificationChannel, error) {
	channel := NotificationChannel{
		ID:          id,
		UserID:      req.UserID,
		Name:        req.Name,
		Type:        req.Type,
		EmailConfig: req.EmailConfig,
		CreatedAt:   time.Time{},
		UpdatedAt:   time.Time{},
	}

	return channel.toSpanner(), nil
}

// toSpanner converts the public NotificationChannel to the internal spannerNotificationChannel for writing.
func (c *NotificationChannel) toSpanner() *spannerNotificationChannel {
	var configData interface{}
	// This can be extended with a switch on c.Type for other configs.
	if c.EmailConfig != nil {
		configData = c.EmailConfig
	}

	var config spanner.NullJSON
	if configData != nil {
		config = spanner.NullJSON{Value: configData, Valid: true}
	}

	return &spannerNotificationChannel{
		NotificationChannel: *c,
		Config:              config,
	}
}

// toPublic converts the internal spannerNotificationChannel to the public NotificationChannel for reading.
func (sc *spannerNotificationChannel) toPublic() (*NotificationChannel, error) {
	ret := &sc.NotificationChannel
	if sc.Config.Valid {
		bytes, err := json.Marshal(sc.Config.Value)
		if err != nil {
			return nil, err
		}
		var emailConfig EmailConfig
		if err := json.Unmarshal(bytes, &emailConfig); err != nil {
			return nil, err
		}
		ret.EmailConfig = &emailConfig
	}

	return ret, nil
}

// CreateNotificationChannel creates a new notification channel.
func (c *Client) CreateNotificationChannel(
	ctx context.Context,
	req CreateNotificationChannelRequest,
) (*string, error) {
	return newEntityCreator[
		notificationChannelMapper, CreateNotificationChannelRequest, *spannerNotificationChannel](c).create(ctx, req)
}

// GetNotificationChannel retrieves a notification channel if it belongs to the specified user.
func (c *Client) GetNotificationChannel(
	ctx context.Context, channelID string, userID string) (*NotificationChannel, error) {
	key := notificationChannelKey{ID: channelID, UserID: userID}
	spannerChannel, err := newEntityReader[notificationChannelMapper,
		spannerNotificationChannel, notificationChannelKey](c).readRowByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	return spannerChannel.toPublic()
}

// UpdateNotificationChannel updates a notification channel if it belongs to the specified user.
func (c *Client) UpdateNotificationChannel(
	ctx context.Context, req UpdateNotificationChannelRequest) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		key := notificationChannelKey{ID: req.ID, UserID: req.UserID}
		spannerChannel, err := newEntityReader[notificationChannelMapper,
			spannerNotificationChannel, notificationChannelKey](c).readRowByKeyWithTransaction(ctx, key, txn)
		if err != nil {
			return err
		}

		merged, err := notificationChannelMapper{}.Merge(req, *spannerChannel)
		if err != nil {
			return err
		}

		m, err := spanner.UpdateStruct(notificationChannelTable, merged)
		if err != nil {
			return err
		}

		return txn.BufferWrite([]*spanner.Mutation{m})
	})

	return err
}

// DeleteNotificationChannel deletes a notification channel if it belongs to the specified user.
func (c *Client) DeleteNotificationChannel(
	ctx context.Context, channelID string, userID string) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		key := spanner.Key{channelID}
		row, err := txn.ReadRow(ctx, notificationChannelTable, key, []string{"UserID"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrQueryReturnedNoResults
			}

			return errors.Join(ErrInternalQueryFailure, err)
		}
		var ownerID string
		if err := row.Column(0, &ownerID); err != nil {
			return err
		}
		if ownerID != userID {
			return ErrQueryReturnedNoResults
		}

		return txn.BufferWrite([]*spanner.Mutation{spanner.Delete(notificationChannelTable, key)})
	})

	return err
}
