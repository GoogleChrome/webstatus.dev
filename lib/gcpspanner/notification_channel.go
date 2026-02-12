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

// EmailConfig represents the JSON structure for an email notification channel.
type EmailConfig struct {
	Address           string  `json:"address,omitempty"`
	IsVerified        bool    `json:"is_verified,omitempty"`
	VerificationToken *string `json:"verification_token,omitempty"`
}

// WebhookConfig represents the JSON structure for a webhook notification channel.
type WebhookConfig struct {
	URL string `json:"url"`
}

// NotificationChannel represents a user-facing notification channel.
type NotificationChannel struct {
	ID            string
	UserID        string
	Name          string
	Type          NotificationChannelType
	EmailConfig   *EmailConfig
	WebhookConfig *WebhookConfig
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// spannerNotificationChannel is the internal struct for Spanner mapping.
type spannerNotificationChannel struct {
	ID        string           `spanner:"ID"`
	UserID    string           `spanner:"UserID"`
	Name      string           `spanner:"Name"`
	Type      string           `spanner:"Type"`
	Config    spanner.NullJSON `spanner:"Config"`
	CreatedAt time.Time        `spanner:"CreatedAt"`
	UpdatedAt time.Time        `spanner:"UpdatedAt"`
}

type NotificationChannelType string

const (
	NotificationChannelTypeEmail   NotificationChannelType = "email"
	NotificationChannelTypeWebhook NotificationChannelType = "webhook"
)

func getAllNotificationTypes() []NotificationChannelType {
	// Use a map so that exhaustive linter will pick up new ones.
	// Then convert the keys to a slice.
	types := map[NotificationChannelType]any{
		NotificationChannelTypeEmail:   nil,
		NotificationChannelTypeWebhook: nil,
	}

	ret := make([]NotificationChannelType, 0, len(types))
	for t := range types {
		ret = append(ret, t)
	}

	return ret

}

// CreateNotificationChannelRequest is the request to create a channel.
type CreateNotificationChannelRequest struct {
	UserID        string
	Name          string
	Type          NotificationChannelType
	EmailConfig   *EmailConfig
	WebhookConfig *WebhookConfig
}

// UpdateNotificationChannelRequest is a request to update a notification channel.
type UpdateNotificationChannelRequest struct {
	ID            string
	UserID        string
	Name          OptionallySet[string]
	Type          OptionallySet[NotificationChannelType]
	EmailConfig   OptionallySet[*EmailConfig]
	WebhookConfig OptionallySet[*WebhookConfig]
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
	if req.Type.IsSet {
		existing.Type = string(req.Type.Value)
	}
	if req.EmailConfig.IsSet && req.EmailConfig.Value != nil {
		existing.Config = spanner.NullJSON{Value: *req.EmailConfig.Value, Valid: true}
	}
	if req.WebhookConfig.IsSet && req.WebhookConfig.Value != nil {
		existing.Config = spanner.NullJSON{Value: *req.WebhookConfig.Value, Valid: true}
	}

	return existing
}

func (m notificationChannelMapper) NewEntity(
	id string,
	req CreateNotificationChannelRequest) (spannerNotificationChannel, error) {
	channel := NotificationChannel{
		ID:            id,
		UserID:        req.UserID,
		Name:          req.Name,
		Type:          req.Type,
		EmailConfig:   req.EmailConfig,
		WebhookConfig: req.WebhookConfig,
		CreatedAt:     spanner.CommitTimestamp,
		UpdatedAt:     spanner.CommitTimestamp,
	}

	return *channel.toSpanner(), nil
}

// toSpanner converts the public NotificationChannel to the internal spannerNotificationChannel for writing.
func (c *NotificationChannel) toSpanner() *spannerNotificationChannel {
	var configData interface{}
	switch c.Type {
	case NotificationChannelTypeEmail:
		configData = c.EmailConfig
	case NotificationChannelTypeWebhook:
		configData = c.WebhookConfig
	}

	var config spanner.NullJSON
	if configData != nil {
		config = spanner.NullJSON{Value: configData, Valid: true}
	}

	return &spannerNotificationChannel{
		ID:        c.ID,
		UserID:    c.UserID,
		Name:      c.Name,
		Type:      string(c.Type),
		Config:    config,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// toPublic converts the internal spannerNotificationChannel to the public NotificationChannel for reading.
func (sc *spannerNotificationChannel) toPublic() (*NotificationChannel, error) {
	var channelType NotificationChannelType
	switch sc.Type {
	case string(NotificationChannelTypeEmail):
		channelType = NotificationChannelTypeEmail
	case string(NotificationChannelTypeWebhook):
		channelType = NotificationChannelTypeWebhook
	default:
		return nil, fmt.Errorf("unknown notification channel type '%s'", sc.Type)
	}

	ret := &NotificationChannel{
		ID:            sc.ID,
		UserID:        sc.UserID,
		Name:          sc.Name,
		Type:          channelType,
		EmailConfig:   nil,
		WebhookConfig: nil,
		CreatedAt:     sc.CreatedAt,
		UpdatedAt:     sc.UpdatedAt,
	}
	subscriptionConfigs, err := loadSubscriptionConfigs(ret.Type, sc.Config)
	if err != nil {
		return nil, err
	}
	ret.EmailConfig = subscriptionConfigs.EmailConfig
	ret.WebhookConfig = subscriptionConfigs.WebhookConfig

	return ret, nil
}

type subscriptionConfigs struct {
	EmailConfig   *EmailConfig
	WebhookConfig *WebhookConfig
}

func loadSubscriptionConfigs(
	channelType NotificationChannelType,
	config spanner.NullJSON) (subscriptionConfigs, error) {
	var ret subscriptionConfigs
	if !config.Valid {
		return ret, nil
	}

	bytes, err := json.Marshal(config.Value)
	if err != nil {
		return ret, err
	}

	switch channelType {
	case NotificationChannelTypeEmail:
		var emailConfig EmailConfig
		if err := json.Unmarshal(bytes, &emailConfig); err != nil {
			return ret, err
		}
		ret.EmailConfig = &emailConfig
		ret.WebhookConfig = nil
	case NotificationChannelTypeWebhook:
		var webhookConfig WebhookConfig
		if err := json.Unmarshal(bytes, &webhookConfig); err != nil {
			return ret, err
		}
		ret.WebhookConfig = &webhookConfig
		ret.EmailConfig = nil
	}

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
	var id *string
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		count, err := c.countNotificationChannels(ctx, req.UserID, txn)
		if err != nil {
			return err
		}
		if count >= int64(c.notificationCfg.maxChannelsPerUser) {
			return ErrOwnerNotificationChannelLimitExceeded
		}

		newID, err := newEntityCreator[notificationChannelMapper](c).createWithTransaction(ctx, txn, req)
		if err != nil {
			return err
		}
		id = newID

		// Also create the initial state for the channel.
		_, err = newEntityCreator[notificationChannelStateMapper](c).createWithTransaction(ctx, txn,
			NotificationChannelState{
				ChannelID:           *id,
				IsDisabledBySystem:  false,
				ConsecutiveFailures: 0,
				CreatedAt:           spanner.CommitTimestamp,
				UpdatedAt:           spanner.CommitTimestamp,
			}, WithID(*id))

		return err
	})

	if err != nil {
		return nil, err
	}

	return id, nil
}

func (c *Client) countNotificationChannels(
	ctx context.Context, userID string, txn *spanner.ReadWriteTransaction) (int64, error) {
	stmt := spanner.Statement{
		SQL: fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE UserID = @userID", notificationChannelTable),
		Params: map[string]interface{}{
			"userID": userID,
		},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, err
	}
	var count int64
	err = row.Column(0, &count)

	return count, err
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

		_, err = newEntityWriter[updateNotificationChannelMapper](c).updateWithTransaction(ctx, txn, req)

		return err
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
