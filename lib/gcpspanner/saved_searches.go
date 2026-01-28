// Copyright 2024 Google LLC
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
	"time"

	"cloud.google.com/go/spanner"
)

// SavedSearchScope represents the scope of a saved search.
type SavedSearchScope string

const (
	// UserPublicScope indicates that this is user created saved search meant to be publicly accessible.
	UserPublicScope SavedSearchScope = "USER_PUBLIC"
	// SystemManagedScope indicates that this is a system managed saved search for a feature.
	SystemManagedScope SavedSearchScope = "SYSTEM_MANAGED"
)

const savedSearchesTable = "SavedSearches"

// SavedSearch represents a saved search row in the SavedSearches table.
type SavedSearch struct {
	ID          string           `spanner:"ID"`
	Name        string           `spanner:"Name"`
	Description *string          `spanner:"Description"`
	Query       string           `spanner:"Query"`
	Scope       SavedSearchScope `spanner:"Scope"`
	AuthorID    string           `spanner:"AuthorID"`
	CreatedAt   time.Time        `spanner:"CreatedAt"`
	UpdatedAt   time.Time        `spanner:"UpdatedAt"`
}

// savedSearchMapper implements the necessary interfaces for the generic helpers.
type savedSearchMapper struct{}

func (m savedSearchMapper) Table() string {
	return savedSearchesTable
}

// SelectOne returns a statement to select a single saved search.
func (m savedSearchMapper) SelectOne(id string) spanner.Statement {
	stmt := spanner.NewStatement(
		`SELECT ID, Name, Description, Query, Scope, AuthorID, CreatedAt, UpdatedAt
		 FROM SavedSearches WHERE ID = @id`,
	)
	stmt.Params["id"] = id

	return stmt
}

// GetSavedSearch retrieves a saved search by its ID.
func (c *Client) GetSavedSearch(ctx context.Context, id string) (*SavedSearch, error) {
	return newEntityReader[savedSearchMapper, SavedSearch, string](c).readRowByKey(ctx, id)
}
