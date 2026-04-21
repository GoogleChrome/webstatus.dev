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
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
)

const (
	SystemGlobalSavedSearchStatusListed   = "LISTED"
	SystemGlobalSavedSearchStatusUnlisted = "UNLISTED"
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

func (m systemGlobalSavedSearchMapper) Table() string {
	return "SystemGlobalSavedSearches"
}

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

// ListSystemGlobalSavedSearchesRequest is a request to list system global saved searches.
type ListSystemGlobalSavedSearchesRequest struct {
	PageSize  int
	PageToken *string
}

// GetPageSize returns the page size for the request.
func (r ListSystemGlobalSavedSearchesRequest) GetPageSize() int {
	return r.PageSize
}

type globalSavedSearchCursor struct {
	LastID           string `json:"last_id"`
	LastDisplayOrder int64  `json:"last_display_order"`
}

type listSystemGlobalSavedSearchesMapper struct{ systemGlobalSavedSearchMapper }

func (m listSystemGlobalSavedSearchesMapper) EncodePageToken(item SystemGlobalSavedSearch) string {
	return encodeCursor(globalSavedSearchCursor{
		LastID:           item.ID,
		LastDisplayOrder: item.DisplayOrder,
	})
}

func (m listSystemGlobalSavedSearchesMapper) SelectList(req ListSystemGlobalSavedSearchesRequest) spanner.Statement {
	var pageFilter string
	params := map[string]any{
		"pageSize": req.PageSize,
		"status":   SystemGlobalSavedSearchStatusListed,
	}
	if req.PageToken != nil {
		cursor, err := decodeCursor[globalSavedSearchCursor](*req.PageToken)
		if err == nil {
			params["lastID"] = cursor.LastID
			params["lastDisplayOrder"] = cursor.LastDisplayOrder
			pageFilter = " AND (g.DisplayOrder < @lastDisplayOrder OR " +
				"(g.DisplayOrder = @lastDisplayOrder AND g.SavedSearchID > @lastID))"
		}
	}
	query := fmt.Sprintf(`
		SELECT
			s.ID, s.Name, s.Description, s.Query, s.Scope, s.AuthorID, s.CreatedAt, s.UpdatedAt,
			g.DisplayOrder, g.Status
		FROM SystemGlobalSavedSearches g
		JOIN SavedSearches s ON g.SavedSearchID = s.ID
		WHERE g.Status = @status %s
		ORDER BY g.DisplayOrder DESC, g.SavedSearchID ASC
		LIMIT @pageSize`, pageFilter)
	stmt := spanner.NewStatement(query)
	stmt.Params = params

	return stmt
}

// ListSystemGlobalSavedSearches returns the listed global saved searches with pagination.
func (c *Client) ListSystemGlobalSavedSearches(
	ctx context.Context,
	pageSize int,
	pageToken *string,
) ([]SystemGlobalSavedSearch, *string, error) {
	req := ListSystemGlobalSavedSearchesRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
	}
	lister := newEntityLister[listSystemGlobalSavedSearchesMapper](c)
	results, nextPageToken, err := lister.list(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	return results, nextPageToken, nil
}

// GetSystemGlobalSavedSearch retrieves a specific global saved search by id.
func (c *Client) GetSystemGlobalSavedSearch(
	ctx context.Context,
	id string,
) (*SystemGlobalSavedSearchWithSortOption, error) {
	reader := newEntityReader[systemGlobalSavedSearchMapper, SystemGlobalSavedSearchWithSortOption, string, string](c)

	return reader.readRowByKey(ctx, id)
}
