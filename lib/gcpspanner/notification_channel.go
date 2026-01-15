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
	"log/slog"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
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
	ID          string                  `spanner:"ID"`
	UserID      string                  `spanner:"UserID"`
	Name        string                  `spanner:"Name"`
	Type        NotificationChannelType `spanner:"Type"`
	EmailConfig *EmailConfig            `spanner:"-"`
	CreatedAt   time.Time               `spanner:"CreatedAt"`
	UpdatedAt   time.Time               `spanner:"UpdatedAt"`
}

// spannerNotificationChannel is the internal struct for Spanner mapping.
type spannerNotificationChannel struct {
	NotificationChannel
	Config spanner.NullJSON `spanner:"Config"`
}

type NotificationChannelType string

const (
	NotificationChannelTypeEmail NotificationChannelType = "email"
)

func getAllNotificationTypes() []NotificationChannelType {
	// Use a map so that exhaustive linter will pick up new ones.
	// Then convert the keys to a slice.
	types := map[NotificationChannelType]any{
		NotificationChannelTypeEmail: nil,
	}

	ret := make([]NotificationChannelType, 0, len(types))
	for t := range types {
		ret = append(ret, t)
	}

	return ret

}

// CreateNotificationChannelRequest is the request to create a channel.
type CreateNotificationChannelRequest struct {
	UserID      string
	Name        string
	Type        NotificationChannelType
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

func (m notificationChannelMapper) Table() string { return notificationChannelTable }

type updateNotificationChannelMapper struct{ notificationChannelMapper }

func (m updateNotificationChannelMapper) GetKeyFromExternal(in UpdateNotificationChannelRequest) string {
	return in.ID
}

func (m notificationChannelMapper) SelectOne(id string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID, UserID, Name, Type, Config, CreatedAt, UpdatedAt
	FROM %s
	WHERE ID = @id
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"id": id,
	}
	stmt.Params = parameters

	return stmt
}

func (m updateNotificationChannelMapper) Merge(
	req UpdateNotificationChannelRequest, existing spannerNotificationChannel) spannerNotificationChannel {
	if req.Name.IsSet {
		existing.Name = req.Name.Value
	}
	if req.EmailConfig.IsSet && req.EmailConfig.Value != nil {
		existing.Config = spanner.NullJSON{Value: *req.EmailConfig.Value, Valid: true}
	}

	return existing
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
		CreatedAt:   spanner.CommitTimestamp,
		UpdatedAt:   spanner.CommitTimestamp,
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
	subscriptionConfigs, err := loadSubscriptionConfigs(sc.Type, sc.Config)
	if err != nil {
		return nil, err
	}
	ret.EmailConfig = subscriptionConfigs.EmailConfig

	return ret, nil
}

type subscriptionConfigs struct {
	EmailConfig *EmailConfig
}

func loadSubscriptionConfigs(_ NotificationChannelType, config spanner.NullJSON) (subscriptionConfigs, error) {
	var ret subscriptionConfigs
	if !config.Valid {
		return ret, nil
	}
	// For now, only email config is supported.
	bytes, err := json.Marshal(config.Value)
	if err != nil {
		return ret, err
	}
	var emailConfig EmailConfig
	if err := json.Unmarshal(bytes, &emailConfig); err != nil {
		return ret, err
	}
	ret.EmailConfig = &emailConfig

	return ret, nil
}

func (c *Client) checkNotificationChannelOwnership(
	ctx context.Context, channelID string, userID string, txn *spanner.ReadWriteTransaction,
) error {
	stmt := spanner.Statement{
		SQL: fmt.Sprintf(`SELECT ID FROM %s WHERE ID = @channelID AND UserID = @userID`, notificationChannelTable),
		Params: map[string]interface{}{
			"channelID": channelID,
			"userID":    userID,
		},
	}

	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	_, err := iter.Next()
	if err != nil {
		// No row found. User does not have a role.
		if errors.Is(err, iterator.Done) {
			return errors.Join(ErrMissingRequiredRole, err)
		}
		slog.ErrorContext(ctx, "failed to query user role", "error", err)

		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}

// CreateNotificationChannel creates a new notification channel.
func (c *Client) CreateNotificationChannel(
	ctx context.Context,
	req CreateNotificationChannelRequest,
) (*string, error) {
	return newEntityCreator[notificationChannelMapper](c).create(ctx, req)
}

// GetNotificationChannel retrieves a notification channel if it belongs to the specified user.
func (c *Client) GetNotificationChannel(
	ctx context.Context, channelID string, userID string) (*NotificationChannel, error) {
	var spannerChannel *spannerNotificationChannel
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		err := c.checkNotificationChannelOwnership(ctx, channelID, userID, txn)
		if err != nil {
			return err
		}
		spannerChannel, err = newEntityReader[notificationChannelMapper,
			spannerNotificationChannel, string](c).readRowByKeyWithTransaction(ctx, channelID, txn)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return spannerChannel.toPublic()
}

// ListNotificationChannelsRequest is a request to list notification channels.
type ListNotificationChannelsRequest struct {
	UserID    string
	PageSize  int
	PageToken *string
}

func (r ListNotificationChannelsRequest) GetPageSize() int {
	return r.PageSize
}

func (r ListNotificationChannelsRequest) GetPageToken() *string {
	return r.PageToken
}

type notificationChannelCursor struct {
	LastID        string    `json:"last_id"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
}

type listNotificationChannelsMapper struct{ notificationChannelMapper }

func (m listNotificationChannelsMapper) EncodePageToken(item spannerNotificationChannel) string {
	return encodeCursor(notificationChannelCursor{
		LastID:        item.ID,
		LastUpdatedAt: item.UpdatedAt,
	})
}

func (m listNotificationChannelsMapper) SelectList(req ListNotificationChannelsRequest) spanner.Statement {
	var pageFilter string
	params := map[string]interface{}{
		"userID":   req.UserID,
		"pageSize": req.PageSize,
	}
	if req.PageToken != nil {
		cursor, err := decodeCursor[notificationChannelCursor](*req.PageToken)
		if err == nil {
			params["lastID"] = cursor.LastID
			params["lastUpdatedAt"] = cursor.LastUpdatedAt
			pageFilter = " AND (UpdatedAt < @lastUpdatedAt OR (UpdatedAt = @lastUpdatedAt AND ID > @lastID))"
		}

	}
	query := fmt.Sprintf(`SELECT
		ID, UserID, Name, Type, Config, CreatedAt, UpdatedAt
	FROM NotificationChannels
	WHERE UserID = @userID %s
	ORDER BY UpdatedAt DESC, ID ASC
	LIMIT @pageSize`, pageFilter)
	stmt := spanner.NewStatement(query)
	stmt.Params = params

	return stmt
}

// ListNotificationChannels lists all notification channels for a user.
func (c *Client) ListNotificationChannels(
	ctx context.Context, req ListNotificationChannelsRequest) ([]NotificationChannel, *string, error) {
	items, token, err := newEntityLister[listNotificationChannelsMapper](c).list(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	channels := make([]NotificationChannel, 0, len(items))
	for _, item := range items {
		channel, err := item.toPublic()
		if err != nil {
			return nil, nil, err
		}
		channels = append(channels, *channel)
	}

	return channels, token, nil
}

// UpdateNotificationChannel updates a notification channel if it belongs to the specified user.
func (c *Client) UpdateNotificationChannel(
	ctx context.Context, req UpdateNotificationChannelRequest) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		err := c.checkNotificationChannelOwnership(ctx, req.ID, req.UserID, txn)
		if err != nil {
			return err
		}

		return newEntityWriter[updateNotificationChannelMapper](c).updateWithTransaction(ctx, txn, req)
	})

	return err
}

type removeNotificationChannelMapper struct{ notificationChannelMapper }

func (m removeNotificationChannelMapper) DeleteKey(in string) spanner.Key { return spanner.Key{in} }

func (m removeNotificationChannelMapper) GetKeyFromExternal(id string) string { return id }

// DeleteNotificationChannel deletes a notification channel if it belongs to the specified user.
func (c *Client) DeleteNotificationChannel(
	ctx context.Context, channelID string, userID string) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		err := c.checkNotificationChannelOwnership(ctx, channelID, userID, txn)
		if err != nil {
			return err
		}

		return newEntityRemover[removeNotificationChannelMapper, spannerNotificationChannel](c).removeWithTransaction(
			ctx, txn, channelID)
	})

	return err
}
