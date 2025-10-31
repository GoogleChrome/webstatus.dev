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

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
)

const savedSearchSubscriptionTable = "SavedSearchSubscriptions"

// SavedSearchSubscription represents a row in the SavedSearchSubscription table.
type SavedSearchSubscription struct {
	ID            string   `spanner:"ID"`
	UserID        string   `spanner:"UserID"`
	ChannelID     string   `spanner:"ChannelID"`
	SavedSearchID string   `spanner:"SavedSearchID"`
	Triggers      []string `spanner:"Triggers"`
	Frequency     string   `spanner:"Frequency"`
}

// CreateSavedSearchSubscriptionRequest is the request to create a subscription.
type CreateSavedSearchSubscriptionRequest struct {
	UserID        string
	ChannelID     string
	SavedSearchID string
	Triggers      []string
	Frequency     string
}

// UpdateSavedSearchSubscriptionRequest is a request to update a saved search subscription.
type UpdateSavedSearchSubscriptionRequest struct {
	ID        string
	UserID    string
	Triggers  OptionallySet[[]string]
	Frequency OptionallySet[string]
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
		ID, UserID, ChannelID, SavedSearchID, Triggers, Frequency
	FROM %s
	WHERE ID = @id
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"id": key,
	}
	stmt.Params = parameters

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

func (m savedSearchSubscriptionMapper) NewEntity(
	id string,
	req CreateSavedSearchSubscriptionRequest) (SavedSearchSubscription, error) {
	return SavedSearchSubscription{
		ID:            id,
		UserID:        req.UserID,
		ChannelID:     req.ChannelID,
		SavedSearchID: req.SavedSearchID,
		Triggers:      req.Triggers,
		Frequency:     req.Frequency,
	}, nil
}

func (c *Client) checkForSavedSearchSubscriptionOwnership(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	userID string,
	subscriptionID string) error {
	key := spanner.Key{subscriptionID}
	row, err := txn.ReadRow(ctx, savedSearchSubscriptionTable, key, []string{"UserID"})
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
		return ErrMissingRequiredRole
	}

	return nil
}

// CreateSavedSearchSubscription creates a new saved search subscription for a user.
func (c *Client) CreateSavedSearchSubscription(
	ctx context.Context, req CreateSavedSearchSubscriptionRequest) (string, error) {
	var newID string
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Verify user owns the channel
		key := spanner.Key{req.ChannelID}
		row, err := txn.ReadRow(ctx, notificationChannelTable, key, []string{"UserID"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrQueryReturnedNoResults
			}

			return errors.Join(ErrInternalQueryFailure, err)
		}
		var channelOwnerID string
		if err := row.Column(0, &channelOwnerID); err != nil {
			return err
		}
		if channelOwnerID != req.UserID {
			return ErrMissingRequiredRole
		}

		// 2. Create the subscription
		id, err := newEntityCreator[savedSearchSubscriptionMapper](c).createWithTransaction(ctx, txn, req)
		if err != nil {
			return err
		}
		newID = id

		return nil
	})
	if err != nil {
		return "", err
	}

	return newID, nil
}

// GetSavedSearchSubscription retrieves a subscription if it belongs to the specified user.
func (c *Client) GetSavedSearchSubscription(
	ctx context.Context, subscriptionID string, userID string) (*SavedSearchSubscription, error) {
	var ret *SavedSearchSubscription
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		err := c.checkForSavedSearchSubscriptionOwnership(ctx, txn, userID, subscriptionID)
		if err != nil {
			return err
		}

		ret, err = newEntityReader[savedSearchSubscriptionMapper,
			SavedSearchSubscription, string](c).readRowByKeyWithTransaction(ctx, subscriptionID, txn)

		return err
	})

	return ret, err
}

// UpdateSavedSearchSubscription updates a subscription if it belongs to the specified user.
func (c *Client) UpdateSavedSearchSubscription(
	ctx context.Context, req UpdateSavedSearchSubscriptionRequest) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		err := c.checkForSavedSearchSubscriptionOwnership(ctx, txn, req.UserID, req.ID)
		if err != nil {
			return err
		}

		return newEntityWriter[savedSearchSubscriptionMapper](c).updateWithTransaction(ctx, txn, req)
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
		err := c.checkForSavedSearchSubscriptionOwnership(ctx, txn, userID, subscriptionID)
		if err != nil {
			return err
		}

		return newEntityRemover[removeSavedSearchSubscriptionMapper, SavedSearchSubscription](c).
			removeWithTransaction(ctx, txn, subscriptionID)
	})

	return err
}
