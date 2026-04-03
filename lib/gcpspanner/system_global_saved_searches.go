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
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SystemGlobalSavedSearch represents a joined row from SystemGlobalSavedSearches and SavedSearches.
type SystemGlobalSavedSearch struct {
	ID           string           `spanner:"ID"`
	Name         string           `spanner:"Name"`
	Description  *string          `spanner:"Description"`
	Query        string           `spanner:"Query"`
	Scope        SavedSearchScope `spanner:"Scope"`
	AuthorID     string           `spanner:"AuthorID"`
	CreatedAt    time.Time        `spanner:"CreatedAt"`
	UpdatedAt    time.Time        `spanner:"UpdatedAt"`
	DisplayOrder int64            `spanner:"DisplayOrder"`
	Status       string           `spanner:"Status"`
}

// SystemGlobalSavedSearchWithSortOption combines SystemGlobalSavedSearch with custom sort order flag.
type SystemGlobalSavedSearchWithSortOption struct {
	SystemGlobalSavedSearch
	HasCustomSortOrder bool `spanner:"HasCustomSortOrder"`
}

type systemGlobalSavedSearchMapper struct{}

func (m systemGlobalSavedSearchMapper) SelectOne(id string) spanner.Statement {
	stmt := spanner.NewStatement(`
		SELECT
			s.ID, s.Name, s.Description, s.Query, s.Scope, s.AuthorID, s.CreatedAt, s.UpdatedAt,
			g.DisplayOrder, g.Status,
			EXISTS (SELECT 1 FROM SavedSearchFeatureSortOrder WHERE SavedSearchID = g.SavedSearchID) AS HasCustomSortOrder
		FROM SystemGlobalSavedSearches g
		JOIN SavedSearches s ON g.SavedSearchID = s.ID
		WHERE g.SavedSearchID = @id
	`)
	stmt.Params["id"] = id
	return stmt
}

// ListSystemGlobalSavedSearches returns the listed global saved searches with pagination total count.
func (c *Client) ListSystemGlobalSavedSearches(
	ctx context.Context,
	pageSize int,
	pageToken *string,
) ([]SystemGlobalSavedSearch, int64, *string, error) {
	offset := 0
	if pageToken != nil {
		offsetCursor, err := decodeInputGlobalSavedSearchCursor(*pageToken)
		if err != nil {
			return nil, 0, nil, err
		}
		offset = offsetCursor.Offset
	}
	// 1. Get total count
	countStmt := spanner.NewStatement(`
		SELECT COUNT(*)
		FROM SystemGlobalSavedSearches
		WHERE Status = 'LISTED'
	`)
	var total int64
	iterCount := c.Single().Query(ctx, countStmt)
	err := iterCount.Do(func(row *spanner.Row) error {
		return row.Column(0, &total)
	})
	iterCount.Stop()
	if err != nil {
		return nil, 0, nil, err
	}

	// 2. Get subset
	stmt := spanner.NewStatement(`
		SELECT
			s.ID, s.Name, s.Description, s.Query, s.Scope, s.AuthorID, s.CreatedAt, s.UpdatedAt,
			g.DisplayOrder, g.Status
		FROM SystemGlobalSavedSearches g
		JOIN SavedSearches s ON g.SavedSearchID = s.ID
		WHERE g.Status = 'LISTED'
		ORDER BY g.DisplayOrder DESC
		LIMIT @limit OFFSET @offset
	`)
	stmt.Params["limit"] = int64(pageSize)
	stmt.Params["offset"] = int64(offset)

	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []SystemGlobalSavedSearch
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, 0, nil, err
		}
		var r SystemGlobalSavedSearch
		if err := row.ToStruct(&r); err != nil {
			return nil, 0, nil, err
		}
		results = append(results, r)
	}

	var nextPageToken *string
	if len(results) == pageSize {
		token := encodeGlobalSavedSearchOffsetCursor(offset + pageSize)
		nextPageToken = &token
	}

	return results, total, nextPageToken, nil
}

// GetSystemGlobalSavedSearch retrieves a specific global saved search by id.
func (c *Client) GetSystemGlobalSavedSearch(
	ctx context.Context,
	id string,
) (*SystemGlobalSavedSearchWithSortOption, error) {
	reader := newEntityReader[systemGlobalSavedSearchMapper, SystemGlobalSavedSearchWithSortOption, string, string](c)
	return reader.readRowByKey(ctx, id)
}

