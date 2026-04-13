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
	// SystemGlobalScope indicates that this is a globally shared bookmark defined by the system.
	SystemGlobalScope SavedSearchScope = "SYSTEM_GLOBAL"
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

// SavedSearchIDOnly is a specialized struct to fetch only IDs from Spanner.
type SavedSearchIDOnly struct {
	ID string `spanner:"ID"`
}

type referencingSavedSearchMapper struct{}

func (m referencingSavedSearchMapper) Table() string {
	return savedSearchesTable
}

func (m referencingSavedSearchMapper) SelectList(req referencingSavedSearchRequest) spanner.Statement {
	savedPattern := "%saved:" + req.ID + "%"
	hotlistPattern := "%hotlist:" + req.ID + "%"
	stmt := spanner.NewStatement(`
		SELECT ID FROM SavedSearches
		WHERE LOWER(Query) LIKE LOWER(@savedPattern)
		   OR LOWER(Query) LIKE LOWER(@hotlistPattern)`)
	stmt.Params["savedPattern"] = savedPattern
	stmt.Params["hotlistPattern"] = hotlistPattern

	return stmt
}

func (m referencingSavedSearchMapper) EncodePageToken(_ SavedSearchIDOnly) string {
	return "" // Not used
}

type referencingSavedSearchRequest struct {
	ID string
}

func (r referencingSavedSearchRequest) GetPageSize() int {
	return 0 // No pagination
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

// GetReferencingSavedSearchIDs finds all saved searches that reference the given ID in their query.
//
// WARNING: This function performs a full table scan on the SavedSearches table because it uses
// a leading wildcard in the LIKE query (e.g., '%saved:ID%').
// As the number of saved searches grows, the performance of this operation will degrade.
//
// Alternatives considered:
//  1. Exact Matching (WHERE Query = @pattern): Feasible ONLY if queries cannot be combined.
//     Since users can combine terms (e.g., 'saved:XYZ AND available_on:chrome'), this will miss references.
//  2. Spanner Search Index: Ideal for fast substring searching. However, Search Indexes
//     are not supported by the Spanner emulator, which would break the local development workflow.
//  3. Junction Table: An explicit table mapping references (e.g., SavedSearchReferences).
//     This is the most scalable approach for strict referential integrity but requires
//     schema changes and maintenance logic.
func (c *Client) GetReferencingSavedSearchIDs(ctx context.Context, id string) ([]string, error) {
	req := referencingSavedSearchRequest{ID: id}
	lister := newEntityLister[referencingSavedSearchMapper, SavedSearchIDOnly, referencingSavedSearchRequest](c)
	results, _, err := lister.list(ctx, req)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}

	return ids, nil
}
