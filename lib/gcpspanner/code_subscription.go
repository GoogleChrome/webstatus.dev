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
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
)

const codeSubscriptionTable = "CodeSubscriptions"

var (
	// ErrCodeSubscriptionNotFound indicates that the code subscription was not found.
	ErrCodeSubscriptionNotFound = errors.New("code subscription not found")
	// ErrUnknownSubscriptionStatus indicates an unrecognized subscription status.
	ErrUnknownSubscriptionStatus = errors.New("unknown subscription status")
)

// SubscriptionStatus represents the lifecycle state of a code subscription.
type SubscriptionStatus string

const (
	SubscriptionActive    SubscriptionStatus = "ACTIVE"
	SubscriptionTriggered SubscriptionStatus = "TRIGGERED"
	SubscriptionDelivered SubscriptionStatus = "DELIVERED"
	SubscriptionResolved  SubscriptionStatus = "RESOLVED"
	SubscriptionObsolete  SubscriptionStatus = "OBSOLETE"
	SubscriptionDeleted   SubscriptionStatus = "DELETED"
	SubscriptionError     SubscriptionStatus = "ERROR"
)

// SubscriptionOccurrence represents a code location where a directive is used.
type SubscriptionOccurrence struct {
	FilePath       string `json:"file_path"`
	LineNumber     int64  `json:"line_number"`
	CommentSnippet string `json:"comment_snippet"`
}

// CodeSubscriptionInput represents an AST scan result to be synchronized in Spanner.
type CodeSubscriptionInput struct {
	VCSProvider        VCSProvider
	VCSInstallationID  string
	VCSRepositoryID    string
	RepositoryFullName string
	TargetQuery        string
	Triggers           []SubscriptionTrigger
	Occurrences        []SubscriptionOccurrence
}

// CodeSubscription represents a subscription anchored to a code repository AST pattern.
type CodeSubscription struct {
	ID                 string
	VCSProvider        VCSProvider
	VCSInstallationID  string
	VCSRepositoryID    string
	RepositoryFullName string
	TargetQuery        string
	Triggers           []SubscriptionTrigger
	Status             SubscriptionStatus
	Occurrences        []SubscriptionOccurrence
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// spannerCodeSubscription is the internal struct for Spanner column mapping.
type spannerCodeSubscription struct {
	ID                 string                `spanner:"ID"`
	VCSProvider        string                `spanner:"VCSProvider"`
	VCSInstallationID  string                `spanner:"VCSInstallationID"`
	VCSRepositoryID    string                `spanner:"VCSRepositoryID"`
	RepositoryFullName string                `spanner:"RepositoryFullName"`
	TargetQuery        string                `spanner:"TargetQuery"`
	Triggers           []SubscriptionTrigger `spanner:"Triggers"`
	Status             string                `spanner:"Status"`
	Occurrences        spanner.NullJSON      `spanner:"Occurrences"`
	CreatedAt          time.Time             `spanner:"CreatedAt"`
	UpdatedAt          time.Time             `spanner:"UpdatedAt"`
}

type codeSubscriptionMapper struct{}

func (m codeSubscriptionMapper) Table() string { return codeSubscriptionTable }

func (m codeSubscriptionMapper) SelectOne(id string) spanner.Statement {
	return spanner.Statement{
		SQL: `SELECT ID, VCSProvider, VCSInstallationID, VCSRepositoryID, RepositoryFullName,
			TargetQuery, Triggers, Status, Occurrences, CreatedAt, UpdatedAt
		FROM CodeSubscriptions
		WHERE ID = @id`,
		Params: map[string]any{"id": id},
	}
}

// ParseSubscriptionStatus parses and validates a raw subscription status string into a typed SubscriptionStatus.
func ParseSubscriptionStatus(val string) (SubscriptionStatus, error) {
	status := SubscriptionStatus(val)
	switch status {
	case SubscriptionActive,
		SubscriptionTriggered,
		SubscriptionDelivered,
		SubscriptionResolved,
		SubscriptionObsolete,
		SubscriptionDeleted,
		SubscriptionError:
		return status, nil
	}

	return "", fmt.Errorf("%w: %q", ErrUnknownSubscriptionStatus, val)
}

func (s *spannerCodeSubscription) toCodeSubscription() (*CodeSubscription, error) {
	provider, err := ParseVCSProvider(s.VCSProvider)
	if err != nil {
		return nil, err
	}

	status, err := ParseSubscriptionStatus(s.Status)
	if err != nil {
		return nil, err
	}

	var occurrences []SubscriptionOccurrence
	if s.Occurrences.Valid && s.Occurrences.Value != nil {
		bytes, err := json.Marshal(s.Occurrences.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal occurrences: %w", err)
		}
		if err := json.Unmarshal(bytes, &occurrences); err != nil {
			return nil, fmt.Errorf("failed to unmarshal occurrences: %w", err)
		}
	} else {
		occurrences = []SubscriptionOccurrence{}
	}

	return &CodeSubscription{
		ID:                 s.ID,
		VCSProvider:        provider,
		VCSInstallationID:  s.VCSInstallationID,
		VCSRepositoryID:    s.VCSRepositoryID,
		RepositoryFullName: s.RepositoryFullName,
		TargetQuery:        s.TargetQuery,
		Triggers:           s.Triggers,
		Status:             status,
		Occurrences:        occurrences,
		CreatedAt:          s.CreatedAt,
		UpdatedAt:          s.UpdatedAt,
	}, nil
}

func fromCodeSubscription(sub *CodeSubscription) *spannerCodeSubscription {
	occurrences := sub.Occurrences
	if occurrences == nil {
		occurrences = []SubscriptionOccurrence{}
	}

	return &spannerCodeSubscription{
		ID:                 sub.ID,
		VCSProvider:        string(sub.VCSProvider),
		VCSInstallationID:  sub.VCSInstallationID,
		VCSRepositoryID:    sub.VCSRepositoryID,
		RepositoryFullName: sub.RepositoryFullName,
		TargetQuery:        sub.TargetQuery,
		Triggers:           sub.Triggers,
		Status:             string(sub.Status),
		Occurrences: spanner.NullJSON{
			Value: occurrences,
			Valid: true,
		},
		CreatedAt: sub.CreatedAt,
		UpdatedAt: sub.UpdatedAt,
	}
}

// GetCodeSubscription retrieves a code subscription by ID.
func (c *Client) GetCodeSubscription(ctx context.Context, id string) (*CodeSubscription, error) {
	r := newEntityReader[
		codeSubscriptionMapper, spannerCodeSubscription, CodeSubscription, string,
	](c)
	spannerSub, err := r.readRowByKey(ctx, id)
	if err != nil {
		if errors.Is(err, ErrQueryReturnedNoResults) {
			return nil, ErrCodeSubscriptionNotFound
		}

		return nil, err
	}

	return spannerSub.toCodeSubscription()
}
