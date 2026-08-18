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

// CodeSubscription represents a subscription anchored to a code repository AST pattern.
type CodeSubscription struct {
	ID                 string
	VCSProvider        string
	VCSInstallationID  string
	VCSRepositoryID    string
	RepositoryFullName string
	TargetQuery        string
	Triggers           []string
	Status             SubscriptionStatus
	Occurrences        []SubscriptionOccurrence
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// spannerCodeSubscription is the internal struct for Spanner column mapping.
type spannerCodeSubscription struct {
	ID                 string           `spanner:"ID"`
	VCSProvider        string           `spanner:"VCSProvider"`
	VCSInstallationID  string           `spanner:"VCSInstallationID"`
	VCSRepositoryID    string           `spanner:"VCSRepositoryID"`
	RepositoryFullName string           `spanner:"RepositoryFullName"`
	TargetQuery        string           `spanner:"TargetQuery"`
	Triggers           []string         `spanner:"Triggers"`
	Status             string           `spanner:"Status"`
	Occurrences        spanner.NullJSON `spanner:"Occurrences"`
	CreatedAt          time.Time        `spanner:"CreatedAt"`
	UpdatedAt          time.Time        `spanner:"UpdatedAt"`
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

func (s *spannerCodeSubscription) toCodeSubscription() (*CodeSubscription, error) {
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
		VCSProvider:        s.VCSProvider,
		VCSInstallationID:  s.VCSInstallationID,
		VCSRepositoryID:    s.VCSRepositoryID,
		RepositoryFullName: s.RepositoryFullName,
		TargetQuery:        s.TargetQuery,
		Triggers:           s.Triggers,
		Status:             SubscriptionStatus(s.Status),
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
		VCSProvider:        sub.VCSProvider,
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
