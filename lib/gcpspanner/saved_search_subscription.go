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
	"log/slog"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const savedSearchSubscriptionTable = "SavedSearchSubscriptions"

// SavedSearchSubscription represents a row in the SavedSearchSubscription table.
type SavedSearchSubscription struct {
	ID            string                  `spanner:"ID"`
	ChannelID     string                  `spanner:"ChannelID"`
	SavedSearchID string                  `spanner:"SavedSearchID"`
	Triggers      []SubscriptionTrigger   `spanner:"Triggers"`
	Frequency     SavedSearchSnapshotType `spanner:"Frequency"`
	CreatedAt     time.Time               `spanner:"CreatedAt"`
	UpdatedAt     time.Time               `spanner:"UpdatedAt"`
}

type SubscriptionTrigger string

const (
	SubscriptionTriggerBrowserImplementationAnyComplete SubscriptionTrigger = "feature.browser_implementation." +
		"any_complete"
	SubscriptionTriggerFeatureBaselinePromoteToNewly      SubscriptionTrigger = "feature.baseline.promote_to_newly"
	SubscriptionTriggerFeatureBaselinePromoteToWidely     SubscriptionTrigger = "feature.baseline.promote_to_widely"
	SubscriptionTriggerFeatureBaselineRegressionToLimited SubscriptionTrigger = "feature.baseline.regression_to_limited"
	SubscriptionTriggerUnknown                            SubscriptionTrigger = "unknown"
)

var (
	// ErrSubscriptionLimitExceeded indicates that the user already has
	// reached the limit of subscriptions that a given user can own.
	ErrSubscriptionLimitExceeded = errors.New("subscription limit reached")
)

// CreateSavedSearchSubscriptionRequest is the request to create a subscription.
type CreateSavedSearchSubscriptionRequest struct {
	UserID        string
	ChannelID     string
	SavedSearchID string
	Triggers      []SubscriptionTrigger
	Frequency     SavedSearchSnapshotType
}

// UpdateSavedSearchSubscriptionRequest is a request to update a saved search subscription.
type UpdateSavedSearchSubscriptionRequest struct {
	ID        string
	UserID    string
	Triggers  OptionallySet[[]SubscriptionTrigger]
	Frequency OptionallySet[SavedSearchSnapshotType]
}

// ListSavedSearchSubscriptionsRequest is a request to list saved search subscriptions.
type ListSavedSearchSubscriptionsRequest struct {
	UserID    string
	PageSize  int
	PageToken *string
}

// GetPageToken returns the page token for the request.
func (r ListSavedSearchSubscriptionsRequest) GetPageToken() *string {
	return r.PageToken
}

// GetPageSize returns the page size for the request.
func (r ListSavedSearchSubscriptionsRequest) GetPageSize() int {
	return r.PageSize
}

type baseSavedSearchSubscriptionMapper struct{}

func (m baseSavedSearchSubscriptionMapper) Table() string {
	return savedSearchSubscriptionTable
}

// savedSearchSubscriptionMapper implements the necessary interfaces for the generic helpers.
type savedSearchSubscriptionMapper struct {
	baseSavedSearchSubscriptionMapper
}

func (m savedSearchSubscriptionMapper) GetKeyFromExternal(
	in UpdateSavedSearchSubscriptionRequest) string {
	return in.ID
}

func (m savedSearchSubscriptionMapper) SelectOne(key string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID, ChannelID, SavedSearchID, Triggers, Frequency, CreatedAt, UpdatedAt
	FROM %s
	WHERE ID = @id
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"id": key,
	}
	stmt.Params = parameters

	return stmt
}

func (m savedSearchSubscriptionMapper) SelectList(req ListSavedSearchSubscriptionsRequest) spanner.Statement {
	// Join with NotificationChannels to filter by UserID.
	var pageFilter string
	params := map[string]interface{}{
		"userID":   req.UserID,
		"pageSize": req.PageSize,
	}
	if req.PageToken != nil {
		cursor, err := decodeCursor[savedSearchSubscriptionCursor](*req.PageToken)
		if err == nil {
			params["lastID"] = cursor.LastID
			params["lastUpdatedAt"] = cursor.LastUpdatedAt
			pageFilter = " AND (sc.UpdatedAt < @lastUpdatedAt OR (sc.UpdatedAt = @lastUpdatedAt AND sc.ID > @lastID))"
		}
	}
	query := fmt.Sprintf(`SELECT
		sc.ID, sc.ChannelID, sc.SavedSearchID, sc.Triggers, sc.Frequency, sc.CreatedAt, sc.UpdatedAt
	FROM SavedSearchSubscriptions sc
	JOIN NotificationChannels nc ON sc.ChannelID = nc.ID
	WHERE nc.UserID = @userID %s
	ORDER BY sc.UpdatedAt DESC, sc.ID ASC LIMIT @pageSize`, pageFilter)

	stmt := spanner.NewStatement(query)
	stmt.Params = params

	return stmt
}

func (m savedSearchSubscriptionMapper) Merge(
	req UpdateSavedSearchSubscriptionRequest, existing SavedSearchSubscription) SavedSearchSubscription {
	if req.Triggers.IsSet {
		existing.Triggers = req.Triggers.Value
	}
	if req.Frequency.IsSet {
		existing.Frequency = req.Frequency.Value
	}

	return existing
}

type savedSearchSubscriptionCursor struct {
	LastID        string    `json:"last_id"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
}

// EncodePageToken returns the ID of the subscription as a page token.
func (m savedSearchSubscriptionMapper) EncodePageToken(item SavedSearchSubscription) string {
	return encodeCursor(savedSearchSubscriptionCursor{
		LastID:        item.ID,
		LastUpdatedAt: item.UpdatedAt,
	})
}

func (m savedSearchSubscriptionMapper) NewEntity(
	id string,
	req CreateSavedSearchSubscriptionRequest) (SavedSearchSubscription, error) {
	return SavedSearchSubscription{
		ID:            id,
		ChannelID:     req.ChannelID,
		SavedSearchID: req.SavedSearchID,
		Triggers:      req.Triggers,
		Frequency:     req.Frequency,
		CreatedAt:     time.Time{},
		UpdatedAt:     time.Time{},
	}, nil
}

func (c *Client) checkNotificationChannelOwnershipBySubscriptionID(
	ctx context.Context, subscriptionID string, userID string, txn *spanner.ReadWriteTransaction,
) error {
	stmt := spanner.Statement{
		// Join the SavedSearchSubscriptions and NotificationChannels tables to verify ownership.
		SQL: `SELECT
			sc.ID
		FROM SavedSearchSubscriptions sc
		JOIN NotificationChannels nc ON sc.ChannelID = nc.ID
		WHERE sc.ID = @subscriptionID AND nc.UserID = @userID
		LIMIT 1`,
		Params: map[string]interface{}{
			"subscriptionID": subscriptionID,
			"userID":         userID,
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

// CreateSavedSearchSubscription creates a new saved search subscription.
func (c *Client) CreateSavedSearchSubscription(
	ctx context.Context,
	req CreateSavedSearchSubscriptionRequest,
) (*string, error) {
	return c.createSavedSearchSubscription(ctx, req)
}

// CreateSubscriptionWithUUID creates a new saved search subscription with a specified UUID.
func (c *Client) CreateSubscriptionWithUUID(
	ctx context.Context,
	req CreateSavedSearchSubscriptionRequest,
	uuid string,
) (*string, error) {
	return c.createSavedSearchSubscription(ctx, req, WithID(uuid))
}

func (c *Client) createSavedSearchSubscription(
	ctx context.Context,
	req CreateSavedSearchSubscriptionRequest,
	opts ...CreateOption,
) (*string, error) {
	var id *string
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Check limit
		var count int64
		stmt := spanner.Statement{
			SQL: `SELECT COUNT(*)
              FROM SavedSearchSubscriptions sc
              JOIN NotificationChannels nc ON sc.ChannelID = nc.ID
              WHERE nc.UserID = @userID`,
			Params: map[string]interface{}{
				"userID": req.UserID,
			},
		}
		row, err := txn.Query(ctx, stmt).Next()
		if err != nil {
			return err
		}
		if err := row.Columns(&count); err != nil {
			return err
		}

		if count >= int64(c.searchCfg.maxSubscriptionsPerUser) {
			return ErrSubscriptionLimitExceeded
		}

		err = c.checkNotificationChannelOwnership(ctx, req.ChannelID, req.UserID, txn)
		if err != nil {
			return err
		}
		newID, err := newEntityCreator[savedSearchSubscriptionMapper](c).createWithTransaction(ctx, txn, req, opts...)
		if err != nil {
			return err
		}
		id = newID

		return nil
	})
	if err != nil {
		return nil, err
	}

	return id, nil
}

// GetSavedSearchSubscription retrieves a subscription if it belongs to the specified user.
func (c *Client) GetSavedSearchSubscription(
	ctx context.Context, subscriptionID string, userID string) (*SavedSearchSubscription, error) {
	var ret *SavedSearchSubscription
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		err := c.checkNotificationChannelOwnershipBySubscriptionID(ctx, subscriptionID, userID, txn)
		if err != nil {
			return err
		}
		sub, err := newEntityReader[savedSearchSubscriptionMapper,
			SavedSearchSubscription, string](c).readRowByKeyWithTransaction(ctx, subscriptionID, txn)
		if err != nil {
			return err
		}
		ret = sub

		return nil
	})

	return ret, err
}

// UpdateSavedSearchSubscription updates a subscription if it belongs to the specified user.
func (c *Client) UpdateSavedSearchSubscription(
	ctx context.Context, req UpdateSavedSearchSubscriptionRequest) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		err := c.checkNotificationChannelOwnershipBySubscriptionID(ctx, req.ID, req.UserID, txn)
		if err != nil {
			return err
		}

		_, err = newEntityWriter[savedSearchSubscriptionMapper](c).updateWithTransaction(ctx, txn, req)

		return err
	})

	return err
}

// removeUserSavedSearchMapper implements removableEntityMapper.
type removeSavedSearchSubscriptionMapper struct {
	baseSavedSearchSubscriptionMapper
}

func (m removeSavedSearchSubscriptionMapper) DeleteKey(key string) spanner.Key {
	return spanner.Key{key}
}
func (m removeSavedSearchSubscriptionMapper) GetKeyFromExternal(in string) string { return in }

func (m removeSavedSearchSubscriptionMapper) SelectOne(key string) spanner.Statement {
	return savedSearchSubscriptionMapper{baseSavedSearchSubscriptionMapper: baseSavedSearchSubscriptionMapper{}}.
		SelectOne(key)
}

// DeleteSavedSearchSubscription deletes a subscription if it belongs to the specified user.
func (c *Client) DeleteSavedSearchSubscription(
	ctx context.Context, subscriptionID string, userID string) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		err := c.checkNotificationChannelOwnershipBySubscriptionID(ctx, subscriptionID, userID, txn)
		if err != nil {
			return err
		}

		return newEntityRemover[removeSavedSearchSubscriptionMapper, string](c).
			removeWithTransaction(ctx, txn, subscriptionID)
	})

	return err
}

// ListSavedSearchSubscriptions retrieves a list of subscriptions for a user with pagination.
func (c *Client) ListSavedSearchSubscriptions(
	ctx context.Context, req ListSavedSearchSubscriptionsRequest) ([]SavedSearchSubscription, *string, error) {
	return newEntityLister[savedSearchSubscriptionMapper](c).list(ctx, req)
}

type spannerSubscriberDestination struct {
	SubscriptionID string                  `spanner:"ID"`
	UserID         string                  `spanner:"UserID"`
	ChannelID      string                  `spanner:"ChannelID"`
	Type           NotificationChannelType `spanner:"Type"`
	Triggers       []SubscriptionTrigger   `spanner:"Triggers"`
	Config         spanner.NullJSON        `spanner:"Config"`
}

type SubscriberDestination struct {
	SubscriptionID string
	UserID         string
	ChannelID      string
	Type           NotificationChannelType
	Triggers       []SubscriptionTrigger
	// If type is EMAIL, EmailConfig is set.
	EmailConfig *EmailConfig
}

type readAllActivePushSubscriptionsMapper struct {
	baseSavedSearchSubscriptionMapper
}

type activePushSubscriptionKey struct {
	SavedSearchID string
	Frequency     SavedSearchSnapshotType
}

func (m readAllActivePushSubscriptionsMapper) SelectAllByKeys(key activePushSubscriptionKey) spanner.Statement {
	// We are looking for subscriptions that match the Event's criteria.
	// We only want PUSH channels (Email/Webhook), not RSS.
	// We LEFT JOIN NotificationChannelStates to check if the channel is healthy.
	return spanner.Statement{
		SQL: `SELECT
			sc.ID,
			nc.UserID,
			sc.ChannelID,
			sc.Triggers,
			nc.Type,
			nc.Config
		FROM SavedSearchSubscriptions sc
		JOIN NotificationChannels nc ON sc.ChannelID = nc.ID
		LEFT JOIN NotificationChannelStates AS cs ON nc.ID = cs.ChannelID
		WHERE
			sc.SavedSearchID = @savedSearchID
			AND sc.Frequency = @frequency
			AND nc.Type IN UNNEST(@notificationTypes)
			AND (cs.IsDisabledBySystem IS NULL OR cs.IsDisabledBySystem = FALSE)`,
		Params: map[string]interface{}{
			"savedSearchID":     key.SavedSearchID,
			"frequency":         key.Frequency,
			"notificationTypes": getAllNotificationTypes(),
		},
	}
}

// FindAllActivePushSubscriptions
// Finds all active subscriptions for the given Search + Frequency
// AND joins them with their NotificationChannel to get the delivery address.
// It also filters out channels that have been disabled by the system (Health Status).
func (c *Client) FindAllActivePushSubscriptions(
	ctx context.Context,
	savedSearchID string,
	frequency SavedSearchSnapshotType,
) ([]SubscriberDestination, error) {
	values, err := newAllByKeysEntityReader[
		readAllActivePushSubscriptionsMapper,
		activePushSubscriptionKey,
		spannerSubscriberDestination](c).readAllByKeys(
		ctx,
		activePushSubscriptionKey{
			SavedSearchID: savedSearchID,
			Frequency:     frequency,
		},
	)
	if err != nil {
		return nil, err
	}
	results := make([]SubscriberDestination, 0, len(values))
	for _, v := range values {
		dest := SubscriberDestination{
			SubscriptionID: v.SubscriptionID,
			UserID:         v.UserID,
			ChannelID:      v.ChannelID,
			Type:           v.Type,
			Triggers:       v.Triggers,
			EmailConfig:    nil,
		}
		subscriptionConfigs, err := loadSubscriptionConfigs(v.Type, v.Config)
		if err != nil {
			return nil, err
		}
		dest.EmailConfig = subscriptionConfigs.EmailConfig
		results = append(results, dest)
	}

	return results, nil
}
