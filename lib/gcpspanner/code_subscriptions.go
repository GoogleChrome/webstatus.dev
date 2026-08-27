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
	// ErrDeliveryLockMismatch indicates worker does not own active lease.
	ErrDeliveryLockMismatch = errors.New("delivery lock mismatch: worker does not own active lease")
	// ErrDeliveryLockExpired indicates the delivery lock lease has expired.
	ErrDeliveryLockExpired = errors.New("delivery lock expired")
	// ErrWebhookAlreadyDelivered indicates the webhook delivery GUID was already processed.
	ErrWebhookAlreadyDelivered = errors.New("webhook delivery already recorded")
	// ErrDuplicateTargetQuery indicates multiple incoming subscriptions have the same TargetQuery.
	ErrDuplicateTargetQuery = errors.New("duplicate target query in incoming subscriptions")
)

type codeSubscriptionAllByRepoKey struct {
	VCSProvider VCSProvider
	RepoID      string
}

type codeSubscriptionAllByRepoMapper struct{}

func (m codeSubscriptionAllByRepoMapper) Table() string { return codeSubscriptionTable }

func (m codeSubscriptionAllByRepoMapper) SelectAllByKeys(key codeSubscriptionAllByRepoKey) spanner.Statement {
	return spanner.Statement{
		SQL: `SELECT ID, VCSProvider, VCSInstallationID, VCSRepositoryID, RepositoryFullName,
			TargetQuery, Triggers, Status, Occurrences, CreatedAt, UpdatedAt
		FROM CodeSubscriptions
		WHERE VCSProvider = @vcsProvider AND VCSRepositoryID = @repoID`,
		Params: map[string]any{
			"vcsProvider": string(key.VCSProvider),
			"repoID":      key.RepoID,
		},
	}
}

// ListCodeSubscriptionsRequest is a request to list code subscriptions for a repository with pagination.
type ListCodeSubscriptionsRequest struct {
	VCSProvider VCSProvider
	RepoID      string
	PageSize    int
	PageToken   *string
}

func (r ListCodeSubscriptionsRequest) GetPageSize() int {
	return r.PageSize
}

func (r ListCodeSubscriptionsRequest) GetPageToken() *string {
	return r.PageToken
}

type codeSubscriptionCursor struct {
	LastID        string    `json:"last_id"`
	LastCreatedAt time.Time `json:"last_created_at"`
}

type listCodeSubscriptionsMapper struct{ codeSubscriptionMapper }

func (m listCodeSubscriptionsMapper) EncodePageToken(item spannerCodeSubscription) string {
	return encodeCursor(codeSubscriptionCursor{
		LastID:        item.ID,
		LastCreatedAt: item.CreatedAt,
	})
}

func (m listCodeSubscriptionsMapper) SelectList(req ListCodeSubscriptionsRequest) spanner.Statement {
	var pageFilter string
	params := map[string]any{
		"vcsProvider": string(req.VCSProvider),
		"repoID":      req.RepoID,
		"pageSize":    req.PageSize,
	}
	if req.PageToken != nil {
		cursor, err := decodeCursor[codeSubscriptionCursor](*req.PageToken)
		if err == nil {
			params["lastID"] = cursor.LastID
			params["lastCreatedAt"] = cursor.LastCreatedAt
			pageFilter = " AND (CreatedAt < @lastCreatedAt OR (CreatedAt = @lastCreatedAt AND ID > @lastID))"
		}
	}
	query := fmt.Sprintf(`SELECT
		ID, VCSProvider, VCSInstallationID, VCSRepositoryID, RepositoryFullName,
		TargetQuery, Triggers, Status, Occurrences, CreatedAt, UpdatedAt
	FROM CodeSubscriptions
	WHERE VCSProvider = @vcsProvider AND VCSRepositoryID = @repoID AND Status = 'ACTIVE' %s
	ORDER BY CreatedAt DESC, ID ASC
	LIMIT @pageSize`, pageFilter)

	stmt := spanner.NewStatement(query)
	stmt.Params = params

	return stmt
}

type codeSubscriptionsByTargetQueryMapper struct{}

func (m codeSubscriptionsByTargetQueryMapper) Table() string { return codeSubscriptionTable }

func (m codeSubscriptionsByTargetQueryMapper) SelectAllByKeys(targetQuery string) spanner.Statement {
	return spanner.Statement{
		SQL: `SELECT ID, VCSProvider, VCSInstallationID, VCSRepositoryID, RepositoryFullName,
			TargetQuery, Triggers, Status, Occurrences, CreatedAt, UpdatedAt
		FROM CodeSubscriptions
		WHERE TargetQuery = @targetQuery AND Status = 'ACTIVE'`,
		Params: map[string]any{
			"targetQuery": targetQuery,
		},
	}
}

func readExistingRepositorySubscriptions(
	ctx context.Context,
	tx *spanner.ReadWriteTransaction,
	vcsProvider VCSProvider,
	repoID string,
) (map[string]CodeSubscription, error) {
	key := codeSubscriptionAllByRepoKey{
		VCSProvider: vcsProvider,
		RepoID:      repoID,
	}
	spannerSubs, err := newAllByKeysEntityReader[
		codeSubscriptionAllByRepoMapper, codeSubscriptionAllByRepoKey, spannerCodeSubscription,
	](nil).readAllByKeysWithTransaction(ctx, key, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing code subscriptions: %w", err)
	}

	existingMap := make(map[string]CodeSubscription, len(spannerSubs))
	for _, scs := range spannerSubs {
		sub, err := scs.toCodeSubscription()
		if err != nil {
			return nil, fmt.Errorf("failed to parse code subscription: %w", err)
		}
		existingMap[sub.TargetQuery] = *sub
	}

	return existingMap, nil
}

func computeSyncMutations(
	incoming []CodeSubscriptionInput,
	existingMap map[string]CodeSubscription,
	now time.Time,
) ([]*spanner.Mutation, error) {
	// Validate uniqueness of TargetQuery across incoming subscriptions.
	seenTargets := make(map[string]struct{}, len(incoming))
	for _, in := range incoming {
		if _, exists := seenTargets[in.TargetQuery]; exists {
			return nil, fmt.Errorf("%w: %q", ErrDuplicateTargetQuery, in.TargetQuery)
		}
		seenTargets[in.TargetQuery] = struct{}{}
	}

	var mutations []*spanner.Mutation
	incomingMatched := make(map[string]bool, len(incoming))

	for _, in := range incoming {
		incomingMatched[in.TargetQuery] = true
		var sub CodeSubscription
		if existing, found := existingMap[in.TargetQuery]; found {
			// Existing subscription or re-added subscription:
			// Preserve original ID and CreatedAt, update occurrences/triggers, and ensure Status = ACTIVE (revival).
			sub = CodeSubscription{
				ID:                 existing.ID,
				VCSProvider:        in.VCSProvider,
				VCSInstallationID:  in.VCSInstallationID,
				VCSRepositoryID:    in.VCSRepositoryID,
				RepositoryFullName: in.RepositoryFullName,
				TargetQuery:        in.TargetQuery,
				Triggers:           in.Triggers,
				Status:             SubscriptionActive,
				Occurrences:        in.Occurrences,
				CreatedAt:          existing.CreatedAt,
				UpdatedAt:          now,
			}
		} else {
			// New subscription: assign fresh UUID only upon database insertion
			sub = CodeSubscription{
				ID:                 uuid.NewString(),
				VCSProvider:        in.VCSProvider,
				VCSInstallationID:  in.VCSInstallationID,
				VCSRepositoryID:    in.VCSRepositoryID,
				RepositoryFullName: in.RepositoryFullName,
				TargetQuery:        in.TargetQuery,
				Triggers:           in.Triggers,
				Status:             SubscriptionActive,
				Occurrences:        in.Occurrences,
				CreatedAt:          now,
				UpdatedAt:          now,
			}
		}

		scs := fromCodeSubscription(&sub)
		mut, mutErr := spanner.InsertOrUpdateStruct(codeSubscriptionTable, scs)
		if mutErr != nil {
			return nil, fmt.Errorf("failed to create code subscription mutation: %w", mutErr)
		}
		mutations = append(mutations, mut)
	}

	for targetQuery, existing := range existingMap {
		// If an active subscription is missing from the incoming code scan, transition to OBSOLETE.
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
	incoming []CodeSubscriptionInput,
) error {
	now := c.timeNow()

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

// VCSRepository represents a distinct version-controlled repository configured in webstatus.dev.
type VCSRepository struct {
	ID                 string
	VCSProvider        VCSProvider
	VCSInstallationID  string
	VCSRepositoryID    string
	RepositoryFullName string
}

// ListVCSRepositoriesByProvider returns distinct repositories tracked in CodeSubscriptions for a given VCS provider.
func (c *Client) ListVCSRepositoriesByProvider(
	ctx context.Context,
	provider VCSProvider,
) ([]VCSRepository, error) {
	stmt := spanner.Statement{
		SQL: `SELECT DISTINCT VCSProvider, VCSInstallationID, VCSRepositoryID, RepositoryFullName
		FROM CodeSubscriptions
		WHERE VCSProvider = @vcsProvider
		ORDER BY RepositoryFullName ASC`,
		Params: map[string]any{
			"vcsProvider": string(provider),
		},
	}
	txn := c.Single()
	defer txn.Close()
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	var repos []VCSRepository
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query repositories: %w", err)
		}
		var s struct {
			VCSProvider        string `spanner:"VCSProvider"`
			VCSInstallationID  string `spanner:"VCSInstallationID"`
			VCSRepositoryID    string `spanner:"VCSRepositoryID"`
			RepositoryFullName string `spanner:"RepositoryFullName"`
		}
		if err := row.ToStruct(&s); err != nil {
			return nil, fmt.Errorf("failed to parse repository row: %w", err)
		}
		p, err := ParseVCSProvider(s.VCSProvider)
		if err != nil {
			return nil, err
		}
		repos = append(repos, VCSRepository{
			ID:                 s.VCSRepositoryID,
			VCSProvider:        p,
			VCSInstallationID:  s.VCSInstallationID,
			VCSRepositoryID:    s.VCSRepositoryID,
			RepositoryFullName: s.RepositoryFullName,
		})
	}

	return repos, nil
}

// ListVCSInstallations returns all VCS installations stored in Spanner.
func (c *Client) ListVCSInstallations(ctx context.Context) ([]VCSInstallation, error) {
	stmt := spanner.NewStatement(`SELECT ID, VCSProvider, VCSInstallationID, AccountLogin, AccountType,
			RepositorySelection, Permissions, CreatedAt, UpdatedAt
		FROM VCSInstallations`)
	txn := c.Single()
	defer txn.Close()
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	var installations []VCSInstallation
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query installations: %w", err)
		}
		var s spannerVCSInstallation
		if err := row.ToStruct(&s); err != nil {
			return nil, fmt.Errorf("failed to parse installation row: %w", err)
		}
		inst, err := s.toVCSInstallation()
		if err != nil {
			return nil, err
		}
		installations = append(installations, *inst)
	}

	return installations, nil
}

// ListVCSInstallationsByAccount returns active installations for a specific provider and account login.
func (c *Client) ListVCSInstallationsByAccount(
	ctx context.Context,
	provider VCSProvider,
	accountLogin string,
) ([]VCSInstallation, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ID, VCSProvider, VCSInstallationID, AccountLogin, AccountType,
			RepositorySelection, Permissions, CreatedAt, UpdatedAt
		FROM VCSInstallations
		WHERE VCSProvider = @vcsProvider AND AccountLogin = @accountLogin`,
		Params: map[string]any{
			"vcsProvider":  string(provider),
			"accountLogin": accountLogin,
		},
	}
	txn := c.Single()
	defer txn.Close()
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	var installations []VCSInstallation
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query installations: %w", err)
		}
		var s spannerVCSInstallation
		if err := row.ToStruct(&s); err != nil {
			return nil, fmt.Errorf("failed to parse installation row: %w", err)
		}
		inst, err := s.toVCSInstallation()
		if err != nil {
			return nil, err
		}
		installations = append(installations, *inst)
	}

	return installations, nil
}

// ListCodeSubscriptionsByRepository returns active code subscriptions for a repository with pagination.
func (c *Client) ListCodeSubscriptionsByRepository(
	ctx context.Context,
	req ListCodeSubscriptionsRequest,
) ([]CodeSubscription, *string, error) {
	if req.PageToken != nil {
		if _, err := decodeCursor[codeSubscriptionCursor](*req.PageToken); err != nil {
			return nil, nil, err
		}
	}

	items, token, err := newEntityLister[listCodeSubscriptionsMapper](c).list(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query code subscriptions: %w", err)
	}

	results := make([]CodeSubscription, 0, len(items))
	for _, scs := range items {
		sub, err := scs.toCodeSubscription()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse code subscription: %w", err)
		}
		results = append(results, *sub)
	}

	return results, token, nil
}

// ListCodeSubscriptionsByTargetQuery returns all active subscriptions matching targetQuery and trigger.
func (c *Client) ListCodeSubscriptionsByTargetQuery(
	ctx context.Context,
	targetQuery string,
	trigger SubscriptionTrigger,
) ([]CodeSubscription, error) {
	spannerSubs, err := newAllByKeysEntityReader[
		codeSubscriptionsByTargetQueryMapper, string, spannerCodeSubscription,
	](c).readAllByKeys(ctx, targetQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query code subscriptions by target: %w", err)
	}

	results := make([]CodeSubscription, 0, len(spannerSubs))
	for _, scs := range spannerSubs {
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
// Note: DeliveryChannel is currently initialized to DeliveryChannelGitHubIssue as GitHub Issues
// are the only supported code subscription delivery channel in this phase.
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
				newDelivery := spannerCodeSubscriptionDelivery{
					ID:               deliveryID,
					SubscriptionID:   subscriptionID,
					DeliveryStatus:   string(DeliveryStatusPending),
					DeliveryChannel:  string(DeliveryChannelGitHubIssue),
					DeliveredAt:      spanner.NullTime{Valid: false, Time: time.Time{}},
					ExternalIssueID:  spanner.NullString{Valid: false, StringVal: ""},
					ExternalIssueURL: spanner.NullString{Valid: false, StringVal: ""},
					ErrorMessage:     spanner.NullString{Valid: false, StringVal: ""},
					LockExpiresAt:    spanner.NullTime{Valid: true, Time: lockExpires},
					WorkerLockID:     spanner.NullString{Valid: true, StringVal: workerID},
					CreatedAt:        now,
					UpdatedAt:        now,
				}

				return spanner.InsertOrUpdateStruct(codeSubscriptionDeliveryTable, newDelivery)
			}

			if existing.DeliveryStatus == string(DeliveryStatusDelivered) {
				return nil, ErrDeliveryAlreadyDelivered
			}

			if existing.LockExpiresAt.Valid && existing.LockExpiresAt.Time.After(now) {
				// Lock still active by another worker
				return nil, ErrDeliveryAlreadyLocked
			}

			// Lock acquired / renewed
			existing.DeliveryStatus = string(DeliveryStatusPending)
			existing.LockExpiresAt = spanner.NullTime{Valid: true, Time: lockExpires}
			existing.WorkerLockID = spanner.NullString{Valid: true, StringVal: workerID}
			existing.UpdatedAt = now

			return spanner.UpdateStruct(codeSubscriptionDeliveryTable, *existing)
		},
	)

	if errors.Is(err, ErrDeliveryAlreadyDelivered) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// RecordDeliverySuccess records successful issue delivery.
// Fencing check: verifies that the workerID owns the active lock lease before recording success.
func (c *Client) RecordDeliverySuccess(
	ctx context.Context,
	deliveryID, workerID, issueID, issueURL string,
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

			// Lock fencing check: verify lock has not expired
			if !existing.LockExpiresAt.Valid || !existing.LockExpiresAt.Time.After(now) {
				return nil, ErrDeliveryLockExpired
			}

			// Lock fencing check: verify worker ownership
			if existing.WorkerLockID.Valid && existing.WorkerLockID.StringVal != workerID {
				return nil, ErrDeliveryLockMismatch
			}

			existing.DeliveryStatus = string(DeliveryStatusDelivered)
			existing.DeliveredAt = spanner.NullTime{Valid: true, Time: now}
			existing.ExternalIssueID = spanner.NullString{Valid: true, StringVal: issueID}
			existing.ExternalIssueURL = spanner.NullString{Valid: true, StringVal: issueURL}
			existing.LockExpiresAt = spanner.NullTime{Valid: false, Time: time.Time{}}
			existing.WorkerLockID = spanner.NullString{Valid: false, StringVal: ""}
			existing.UpdatedAt = now

			return spanner.UpdateStruct(codeSubscriptionDeliveryTable, *existing)
		},
	)
}

// ReleaseDeliveryLock clears worker lock lease on transient errors.
// Fencing check: only clears the lock if the specified workerID owns the active lock lease
// and the lease has not expired.
// If the caller does not own the lock, ErrDeliveryLockMismatch is returned.
// If the lock lease has expired, ErrDeliveryLockExpired is returned.
func (c *Client) ReleaseDeliveryLock(ctx context.Context, deliveryID, workerID string) error {
	mutator := newEntityMutator[codeSubscriptionDeliveryMapper, spannerCodeSubscriptionDelivery, string](c)

	err := mutator.readInspectMutate(
		ctx,
		deliveryID,
		func(_ context.Context, existing *spannerCodeSubscriptionDelivery) (*spanner.Mutation, error) {
			if existing == nil {
				return nil, ErrCodeSubscriptionDeliveryNotFound
			}

			now := c.timeNow()

			// Lock fencing: verify lock has not expired
			if !existing.LockExpiresAt.Valid || !existing.LockExpiresAt.Time.After(now) {
				return nil, ErrDeliveryLockExpired
			}

			// Lock fencing: verify worker ownership
			if !existing.WorkerLockID.Valid || existing.WorkerLockID.StringVal != workerID {
				return nil, ErrDeliveryLockMismatch
			}

			existing.DeliveryStatus = string(DeliveryStatusPending)
			existing.LockExpiresAt = spanner.NullTime{Valid: false, Time: time.Time{}}
			existing.WorkerLockID = spanner.NullString{Valid: false, StringVal: ""}
			existing.UpdatedAt = now

			return spanner.UpdateStruct(codeSubscriptionDeliveryTable, *existing)
		},
	)
	if errors.Is(err, ErrCodeSubscriptionDeliveryNotFound) {
		return nil
	}

	return err
}
