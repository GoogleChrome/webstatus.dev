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
	"time"

	"cloud.google.com/go/spanner"
)

const codeSubscriptionDeliveryTable = "CodeSubscriptionDeliveries"

var (
	// ErrCodeSubscriptionDeliveryNotFound indicates that the delivery record was not found.
	ErrCodeSubscriptionDeliveryNotFound = errors.New("code subscription delivery not found")
	// ErrUnknownDeliveryStatus indicates an unrecognized delivery status.
	ErrUnknownDeliveryStatus = errors.New("unknown delivery status")
	// ErrUnknownDeliveryChannel indicates an unrecognized delivery channel.
	ErrUnknownDeliveryChannel = errors.New("unknown delivery channel")
)

// DeliveryStatus represents the state of a notification delivery attempt.
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "PENDING"
	DeliveryStatusDelivered DeliveryStatus = "DELIVERED"
	DeliveryStatusFailed    DeliveryStatus = "FAILED"
)

// DeliveryChannel represents the issue/notification delivery channel.
type DeliveryChannel string

const (
	DeliveryChannelGitHubIssue DeliveryChannel = "github_issue"
)

// CodeSubscriptionDelivery tracks an issue notification delivery for a subscription.
type CodeSubscriptionDelivery struct {
	ID               string
	SubscriptionID   string
	DeliveryStatus   DeliveryStatus
	DeliveryChannel  DeliveryChannel
	LockExpiresAt    *time.Time
	WorkerLockID     *string
	DeliveredAt      *time.Time
	ExternalIssueID  *string
	ExternalIssueURL *string
	ErrorMessage     *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// spannerCodeSubscriptionDelivery is the internal struct for Spanner column mapping.
type spannerCodeSubscriptionDelivery struct {
	ID               string             `spanner:"ID"`
	SubscriptionID   string             `spanner:"SubscriptionID"`
	DeliveryStatus   string             `spanner:"DeliveryStatus"`
	DeliveryChannel  string             `spanner:"DeliveryChannel"`
	LockExpiresAt    spanner.NullTime   `spanner:"LockExpiresAt"`
	WorkerLockID     spanner.NullString `spanner:"WorkerLockID"`
	DeliveredAt      spanner.NullTime   `spanner:"DeliveredAt"`
	ExternalIssueID  spanner.NullString `spanner:"ExternalIssueID"`
	ExternalIssueURL spanner.NullString `spanner:"ExternalIssueURL"`
	ErrorMessage     spanner.NullString `spanner:"ErrorMessage"`
	CreatedAt        time.Time          `spanner:"CreatedAt"`
	UpdatedAt        time.Time          `spanner:"UpdatedAt"`
}

type codeSubscriptionDeliveryMapper struct{}

func (m codeSubscriptionDeliveryMapper) Table() string { return codeSubscriptionDeliveryTable }

func (m codeSubscriptionDeliveryMapper) SelectOne(id string) spanner.Statement {
	return spanner.Statement{
		SQL: `SELECT ID, SubscriptionID, DeliveryStatus, DeliveryChannel, LockExpiresAt,
			WorkerLockID, DeliveredAt, ExternalIssueID, ExternalIssueURL, ErrorMessage,
			CreatedAt, UpdatedAt
		FROM CodeSubscriptionDeliveries
		WHERE ID = @id`,
		Params: map[string]any{"id": id},
	}
}

// ParseDeliveryStatus parses and validates a raw delivery status string into a typed DeliveryStatus.
func ParseDeliveryStatus(val string) (DeliveryStatus, error) {
	status := DeliveryStatus(val)
	switch status {
	case DeliveryStatusPending,
		DeliveryStatusDelivered,
		DeliveryStatusFailed:
		return status, nil
	}

	return "", fmt.Errorf("%w: %q", ErrUnknownDeliveryStatus, val)
}

// ParseDeliveryChannel parses and validates a raw delivery channel string into a typed DeliveryChannel.
func ParseDeliveryChannel(val string) (DeliveryChannel, error) {
	channel := DeliveryChannel(val)
	switch channel {
	case DeliveryChannelGitHubIssue:
		return channel, nil
	}

	return "", fmt.Errorf("%w: %q", ErrUnknownDeliveryChannel, val)
}

func (s *spannerCodeSubscriptionDelivery) toCodeSubscriptionDelivery() (*CodeSubscriptionDelivery, error) {
	status, err := ParseDeliveryStatus(s.DeliveryStatus)
	if err != nil {
		return nil, err
	}
	channel, err := ParseDeliveryChannel(s.DeliveryChannel)
	if err != nil {
		return nil, err
	}

	var lockExpiresAt *time.Time
	if s.LockExpiresAt.Valid {
		lockExpiresAt = &s.LockExpiresAt.Time
	}

	var workerLockID *string
	if s.WorkerLockID.Valid {
		workerLockID = &s.WorkerLockID.StringVal
	}

	var deliveredAt *time.Time
	if s.DeliveredAt.Valid {
		deliveredAt = &s.DeliveredAt.Time
	}

	var extIssueID *string
	if s.ExternalIssueID.Valid {
		extIssueID = &s.ExternalIssueID.StringVal
	}

	var extIssueURL *string
	if s.ExternalIssueURL.Valid {
		extIssueURL = &s.ExternalIssueURL.StringVal
	}

	var errMsg *string
	if s.ErrorMessage.Valid {
		errMsg = &s.ErrorMessage.StringVal
	}

	return &CodeSubscriptionDelivery{
		ID:               s.ID,
		SubscriptionID:   s.SubscriptionID,
		DeliveryStatus:   status,
		DeliveryChannel:  channel,
		LockExpiresAt:    lockExpiresAt,
		WorkerLockID:     workerLockID,
		DeliveredAt:      deliveredAt,
		ExternalIssueID:  extIssueID,
		ExternalIssueURL: extIssueURL,
		ErrorMessage:     errMsg,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}, nil
}

func fromCodeSubscriptionDelivery(d *CodeSubscriptionDelivery) *spannerCodeSubscriptionDelivery {
	var lockExpiresAt spanner.NullTime
	if d.LockExpiresAt != nil {
		lockExpiresAt = spanner.NullTime{Time: *d.LockExpiresAt, Valid: true}
	}

	var workerLockID spanner.NullString
	if d.WorkerLockID != nil {
		workerLockID = spanner.NullString{StringVal: *d.WorkerLockID, Valid: true}
	}

	var deliveredAt spanner.NullTime
	if d.DeliveredAt != nil {
		deliveredAt = spanner.NullTime{Time: *d.DeliveredAt, Valid: true}
	}

	var extIssueID spanner.NullString
	if d.ExternalIssueID != nil {
		extIssueID = spanner.NullString{StringVal: *d.ExternalIssueID, Valid: true}
	}

	var extIssueURL spanner.NullString
	if d.ExternalIssueURL != nil {
		extIssueURL = spanner.NullString{StringVal: *d.ExternalIssueURL, Valid: true}
	}

	var errMsg spanner.NullString
	if d.ErrorMessage != nil {
		errMsg = spanner.NullString{StringVal: *d.ErrorMessage, Valid: true}
	}

	return &spannerCodeSubscriptionDelivery{
		ID:               d.ID,
		SubscriptionID:   d.SubscriptionID,
		DeliveryStatus:   string(d.DeliveryStatus),
		DeliveryChannel:  string(d.DeliveryChannel),
		LockExpiresAt:    lockExpiresAt,
		WorkerLockID:     workerLockID,
		DeliveredAt:      deliveredAt,
		ExternalIssueID:  extIssueID,
		ExternalIssueURL: extIssueURL,
		ErrorMessage:     errMsg,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
}

// GetCodeSubscriptionDelivery retrieves a delivery record by ID.
func (c *Client) GetCodeSubscriptionDelivery(
	ctx context.Context,
	id string,
) (*CodeSubscriptionDelivery, error) {
	r := newEntityReader[
		codeSubscriptionDeliveryMapper, spannerCodeSubscriptionDelivery, CodeSubscriptionDelivery, string,
	](c)
	spannerDel, err := r.readRowByKey(ctx, id)
	if err != nil {
		if errors.Is(err, ErrQueryReturnedNoResults) {
			return nil, ErrCodeSubscriptionDeliveryNotFound
		}

		return nil, err
	}

	return spannerDel.toCodeSubscriptionDelivery()
}
