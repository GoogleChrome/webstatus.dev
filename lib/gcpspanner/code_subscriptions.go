// Copyright 2026 Google LLC
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
	"slices"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

var (
	// ErrDeliveryAlreadyDelivered indicates the delivery task is already completed.
	ErrDeliveryAlreadyDelivered = errors.New("delivery already delivered")
	// ErrDeliveryAlreadyLocked indicates the delivery task is currently leased by another worker.
	ErrDeliveryAlreadyLocked = errors.New("delivery already locked by another worker")
	// ErrWebhookAlreadyDelivered indicates the webhook delivery GUID was already processed.
	ErrWebhookAlreadyDelivered = errors.New("webhook delivery already recorded")
)

func readExistingRepositorySubscriptions(
	ctx context.Context,
	tx *spanner.ReadWriteTransaction,
	vcsProvider VCSProvider,
	repoID string,
) (map[string]CodeSubscription, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ID, VCSProvider, VCSInstallationID, VCSRepositoryID, RepositoryFullName,
			TargetQuery, Triggers, Status, Occurrences, CreatedAt, UpdatedAt
		FROM CodeSubscriptions
		WHERE VCSProvider = @vcsProvider AND VCSRepositoryID = @repoID`,
		Params: map[string]any{
			"vcsProvider": string(vcsProvider),
			"repoID":      repoID,
		},
	}

	iter := tx.Query(ctx, stmt)
	defer iter.Stop()

	existingMap := make(map[string]CodeSubscription)
	for {
		row, iterErr := iter.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, fmt.Errorf("failed to read existing code subscriptions: %w", iterErr)
		}

		var scs spannerCodeSubscription
		if err := row.ToStruct(&scs); err != nil {
			return nil, fmt.Errorf("failed to decode code subscription row: %w", err)
		}
		sub, err := scs.toCodeSubscription()
		if err != nil {
			return nil, fmt.Errorf("failed to parse code subscription: %w", err)
		}
		existingMap[sub.TargetQuery] = *sub
	}

	return existingMap, nil
}

func computeSyncMutations(
	incoming []CodeSubscription,
	existingMap map[string]CodeSubscription,
	now time.Time,
) ([]*spanner.Mutation, error) {
	var mutations []*spanner.Mutation
	incomingMatched := make(map[string]bool)

	for _, in := range incoming {
		incomingMatched[in.TargetQuery] = true
		if existing, found := existingMap[in.TargetQuery]; found {
			in.ID = existing.ID
			in.CreatedAt = existing.CreatedAt
			in.Status = SubscriptionActive
		} else {
			in.CreatedAt = now
			in.Status = SubscriptionActive
		}
		in.UpdatedAt = now

		scs := fromCodeSubscription(&in)
		mut, mutErr := spanner.InsertOrUpdateStruct(codeSubscriptionTable, scs)
		if mutErr != nil {
			return nil, fmt.Errorf("failed to create code subscription mutation: %w", mutErr)
		}
		mutations = append(mutations, mut)
	}

	for targetQuery, existing := range existingMap {
		if !incomingMatched[targetQuery] && existing.Status == SubscriptionActive {
			existing.Status = SubscriptionObsolete
			existing.UpdatedAt = now
			scs := fromCodeSubscription(&existing)
			mut, mutErr := spanner.InsertOrUpdateStruct(codeSubscriptionTable, scs)
			if mutErr != nil {
				return nil, fmt.Errorf("failed to create obsolete mutation: %w", mutErr)
			}
			mutations = append(mutations, mut)
		}
	}

	return mutations, nil
}

// SynchronizeRepositoryCodeSubscriptions transactionally updates the active code subscriptions
// for a specific repository, inserting new subscriptions, updating modified occurrences, and
// marking missing ones as obsolete.
func (c *Client) SynchronizeRepositoryCodeSubscriptions(
	ctx context.Context,
	vcsProvider VCSProvider,
	repoID string,
	incoming []CodeSubscription,
) error {
	now := time.Now().UTC()

	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		existingMap, err := readExistingRepositorySubscriptions(ctx, tx, vcsProvider, repoID)
		if err != nil {
			return err
		}

		mutations, err := computeSyncMutations(incoming, existingMap, now)
		if err != nil {
			return err
		}

		if len(mutations) > 0 {
			if err := tx.BufferWrite(mutations); err != nil {
				return fmt.Errorf("failed to buffer write mutations: %w", err)
			}
		}

		return nil
	})

	return err
}

// ListCodeSubscriptionsByRepository returns active code subscriptions for a repository.
func (c *Client) ListCodeSubscriptionsByRepository(
	ctx context.Context,
	vcsProvider VCSProvider,
	repoID string,
) ([]CodeSubscription, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ID, VCSProvider, VCSInstallationID, VCSRepositoryID, RepositoryFullName,
			TargetQuery, Triggers, Status, Occurrences, CreatedAt, UpdatedAt
		FROM CodeSubscriptions
		WHERE VCSProvider = @vcsProvider AND VCSRepositoryID = @repoID AND Status = 'ACTIVE'
		ORDER BY CreatedAt DESC, ID DESC`,
		Params: map[string]any{
			"vcsProvider": string(vcsProvider),
			"repoID":      repoID,
		},
	}

	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []CodeSubscription
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query code subscriptions: %w", err)
		}

		var scs spannerCodeSubscription
		if err := row.ToStruct(&scs); err != nil {
			return nil, fmt.Errorf("failed to decode code subscription: %w", err)
		}
		sub, err := scs.toCodeSubscription()
		if err != nil {
			return nil, fmt.Errorf("failed to parse code subscription: %w", err)
		}
		results = append(results, *sub)
	}

	return results, nil
}

// ListCodeSubscriptionsByTargetQuery returns all active subscriptions matching targetQuery and trigger.
func (c *Client) ListCodeSubscriptionsByTargetQuery(
	ctx context.Context,
	targetQuery, trigger string,
) ([]CodeSubscription, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ID, VCSProvider, VCSInstallationID, VCSRepositoryID, RepositoryFullName,
			TargetQuery, Triggers, Status, Occurrences, CreatedAt, UpdatedAt
		FROM CodeSubscriptions
		WHERE TargetQuery = @targetQuery AND Status = 'ACTIVE'`,
		Params: map[string]any{
			"targetQuery": targetQuery,
		},
	}

	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []CodeSubscription
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query code subscriptions by target: %w", err)
		}

		var scs spannerCodeSubscription
		if err := row.ToStruct(&scs); err != nil {
			return nil, fmt.Errorf("failed to decode code subscription: %w", err)
		}
		sub, err := scs.toCodeSubscription()
		if err != nil {
			return nil, fmt.Errorf("failed to parse code subscription: %w", err)
		}

		if slices.Contains(sub.Triggers, trigger) {
			results = append(results, *sub)
		}
	}

	return results, nil
}

// DeleteCodeSubscription marks a code subscription as deleted.
func (c *Client) DeleteCodeSubscription(ctx context.Context, id string) error {
	now := time.Now().UTC()
	mut := spanner.UpdateMap(codeSubscriptionTable, map[string]any{
		"ID":        id,
		"Status":    string(SubscriptionDeleted),
		"UpdatedAt": now,
	})

	_, err := c.Apply(ctx, []*spanner.Mutation{mut})

	return err
}

// RecordVCSWebhookDelivery checks and inserts a webhook delivery GUID for replay protection.
// Returns true if new delivery recorded, false if already processed.
func (c *Client) RecordVCSWebhookDelivery(
	ctx context.Context,
	delivery VCSWebhookDelivery,
) (bool, error) {
	mutator := newEntityMutator[vcsWebhookDeliveryMapper, spannerVCSWebhookDelivery, vcsWebhookDeliveryKey](c)

	key := vcsWebhookDeliveryKey{
		VCSProvider:  delivery.VCSProvider,
		DeliveryGUID: delivery.DeliveryGUID,
	}

	err := mutator.readInspectMutate(
		ctx,
		key,
		func(_ context.Context, existing *spannerVCSWebhookDelivery) (*spanner.Mutation, error) {
			if existing != nil {
				return nil, ErrWebhookAlreadyDelivered
			}

			spannerWebhook := fromVCSWebhookDelivery(&delivery)

			mut, mutErr := spanner.InsertOrUpdateStruct(vcsWebhookDeliveryTable, spannerWebhook)
			if mutErr != nil {
				return nil, fmt.Errorf("failed to create webhook delivery mutation: %w", mutErr)
			}

			return mut, nil
		},
	)

	if errors.Is(err, ErrWebhookAlreadyDelivered) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// InsertCodeSubscriptionScanLog stores an AST scan audit log entry using entityWriter.
func (c *Client) InsertCodeSubscriptionScanLog(
	ctx context.Context,
	scanLog CodeSubscriptionScanLog,
) error {
	if scanLog.ID == "" {
		scanLog.ID = uuid.NewString()
	}
	if scanLog.ScannedAt.IsZero() {
		scanLog.ScannedAt = c.timeNow()
	}

	return newEntityWriter[
		codeSubscriptionScanLogMapper,
		CodeSubscriptionScanLog,
		spannerCodeSubscriptionScanLog,
		string,
	](c).upsert(ctx, scanLog)
}

// AcquireDeliveryLock attempts to atomically lease a delivery task for a worker.
func (c *Client) AcquireDeliveryLock(
	ctx context.Context,
	subscriptionID, deliveryID, workerID string,
	ttl time.Duration,
) (bool, error) {
	mutator := newEntityMutator[codeSubscriptionDeliveryMapper, spannerCodeSubscriptionDelivery, string](c)

	err := mutator.readInspectMutate(
		ctx,
		deliveryID,
		func(_ context.Context, existing *spannerCodeSubscriptionDelivery) (*spanner.Mutation, error) {
			now := c.timeNow()
			lockExpires := now.Add(ttl)

			if existing == nil {
				// Delivery record does not exist yet; insert a new locked delivery
				return spanner.InsertOrUpdateMap(codeSubscriptionDeliveryTable, map[string]any{
					"ID":              deliveryID,
					"SubscriptionID":  subscriptionID,
					"DeliveryStatus":  string(DeliveryStatusPending),
					"DeliveryChannel": string(DeliveryChannelGitHubIssue),
					"LockExpiresAt":   lockExpires,
					"WorkerLockID":    workerID,
					"CreatedAt":       now,
					"UpdatedAt":       now,
				}), nil
			}

			if existing.DeliveryStatus == string(DeliveryStatusDelivered) {
				return nil, ErrDeliveryAlreadyDelivered
			}

			if existing.LockExpiresAt.Valid && existing.LockExpiresAt.Time.After(now) {
				// Lock still active by another worker
				return nil, ErrDeliveryAlreadyLocked
			}

			// Lock acquired / renewed
			return spanner.UpdateMap(codeSubscriptionDeliveryTable, map[string]any{
				"ID":             deliveryID,
				"DeliveryStatus": string(DeliveryStatusPending),
				"LockExpiresAt":  lockExpires,
				"WorkerLockID":   workerID,
				"UpdatedAt":      now,
			}), nil
		},
	)

	if errors.Is(err, ErrDeliveryAlreadyDelivered) || errors.Is(err, ErrDeliveryAlreadyLocked) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// RecordDeliverySuccess records successful issue delivery.
func (c *Client) RecordDeliverySuccess(
	ctx context.Context,
	deliveryID, issueID, issueURL string,
) error {
	mutator := newEntityMutator[codeSubscriptionDeliveryMapper, spannerCodeSubscriptionDelivery, string](c)

	return mutator.readInspectMutate(
		ctx,
		deliveryID,
		func(_ context.Context, existing *spannerCodeSubscriptionDelivery) (*spanner.Mutation, error) {
			if existing == nil {
				return nil, ErrCodeSubscriptionDeliveryNotFound
			}

			now := c.timeNow()

			return spanner.UpdateMap(codeSubscriptionDeliveryTable, map[string]any{
				"ID":               deliveryID,
				"DeliveryStatus":   string(DeliveryStatusDelivered),
				"DeliveredAt":      now,
				"ExternalIssueID":  issueID,
				"ExternalIssueURL": issueURL,
				"LockExpiresAt":    nil,
				"WorkerLockID":     nil,
				"UpdatedAt":        now,
			}), nil
		},
	)
}

// ReleaseDeliveryLock clears worker lock lease on transient errors.
func (c *Client) ReleaseDeliveryLock(ctx context.Context, deliveryID string) error {
	mutator := newEntityMutator[codeSubscriptionDeliveryMapper, spannerCodeSubscriptionDelivery, string](c)

	err := mutator.readInspectMutate(
		ctx,
		deliveryID,
		func(_ context.Context, existing *spannerCodeSubscriptionDelivery) (*spanner.Mutation, error) {
			if existing == nil {
				return nil, ErrCodeSubscriptionDeliveryNotFound
			}

			return spanner.UpdateMap(codeSubscriptionDeliveryTable, map[string]any{
				"ID":             deliveryID,
				"DeliveryStatus": string(DeliveryStatusPending),
				"LockExpiresAt":  nil,
				"WorkerLockID":   nil,
				"UpdatedAt":      c.timeNow(),
			}), nil
		},
	)
	if errors.Is(err, ErrCodeSubscriptionDeliveryNotFound) {
		return nil
	}

	return err
}
