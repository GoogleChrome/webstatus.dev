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

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const (
	listUserSavedSearchesBaseRawTemplate = `
SELECT
	ID,
	Name,
	Description,
	Query,
	Scope,
	AuthorID,
	CreatedAt,
	UpdatedAt,
	r.UserRole AS Role,
	CASE
		WHEN b.UserID IS NOT NULL THEN TRUE
		ELSE FALSE
	END AS IsBookmarked
FROM SavedSearches s
LEFT JOIN
	SavedSearchUserRoles r ON s.ID = r.SavedSearchID AND r.UserID = @userID
JOIN
	UserSavedSearchBookmarks b ON s.ID = b.SavedSearchID AND b.UserID = @userID
WHERE
	s.Scope = 'USER_PUBLIC'
{{ if .PageFilter }}
	{{ .PageFilter }}
{{ end }}
ORDER BY Name ASC, ID ASC LIMIT @pageSize
`

	// Because the name might not be unique, we must allow tie breaking by ID.
	commonListUserSavedSearchesPaginationRawTemplate = `
AND (s.Name > @lastName OR (s.Name = @lastName AND s.ID > @lastID))`
)

// nolint: gochecknoglobals // WONTFIX. Compile the template once at startup. Startup fails if invalid.
var (
	// listUserSavedSearchesBaseTemplate is the compiled version of listUserSavedSearchesBaseRawTemplate.
	listUserSavedSearchesBaseTemplate BaseQueryTemplate
)

func init() {
	listUserSavedSearchesBaseTemplate = NewQueryTemplate(listUserSavedSearchesBaseRawTemplate)
}

// listUserSavedSearchTemplateData contains the variables for the listUserSavedSearchesBaseRawTemplate.
type listUserSavedSearchTemplateData struct {
	PageFilter string
}

// UserSavedSearchesPage contains the details for a page of UserSavedSearches.
type UserSavedSearchesPage struct {
	NextPageToken *string
	Searches      []UserSavedSearch
}

// UserSavedSearchesCursor represents a point for resuming queries based on
// the last date. Useful for pagination.
type UserSavedSearchesCursor struct {
	LastID   string `json:"last_id"`
	LastName string `json:"last_name"`
}

// decodeUserSavedSearchesCursor provides a wrapper around the generic decodeCursor for UserSavedSearchesCursor.
func decodeUserSavedSearchesCursor(
	cursor string) (*UserSavedSearchesCursor, error) {
	return decodeCursor[UserSavedSearchesCursor](cursor)
}

// encodeUserSavedSearchesCursor providers a wrapper around the generic encodeCursor for UserSavedSearchesCursor.
func encodeUserSavedSearchesCursor(id string, name string) string {
	return encodeCursor(UserSavedSearchesCursor{
		LastID:   id,
		LastName: name,
	})
}

func (c *Client) ListUserSavedSearches(
	ctx context.Context,
	userID string,
	pageSize int,
	pageToken *string) (*UserSavedSearchesPage, error) {
	params := map[string]interface{}{
		"userID":   userID,
		"pageSize": pageSize,
	}

	tmplData := listUserSavedSearchTemplateData{
		PageFilter: "",
	}

	if pageToken != nil {
		cursor, err := decodeUserSavedSearchesCursor(*pageToken)
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		params["lastName"] = cursor.LastName
		params["lastID"] = cursor.LastID
		tmplData.PageFilter = commonListUserSavedSearchesPaginationRawTemplate
	}

	tmpl := listUserSavedSearchesBaseTemplate.Execute(tmplData)
	stmt := spanner.NewStatement(tmpl)
	stmt.Params = params

	txn := c.Single()
	defer txn.Close()
	it := txn.Query(ctx, stmt)
	defer it.Stop()

	var results []UserSavedSearch

	for {
		row, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var result UserSavedSearch
		if err := row.ToStruct(&result); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	page := &UserSavedSearchesPage{
		Searches:      results,
		NextPageToken: nil,
	}

	if len(results) == pageSize {
		lastResult := results[len(results)-1]
		token := encodeUserSavedSearchesCursor(lastResult.ID, lastResult.Name)
		page.NextPageToken = &token
	}

	return page, nil
}
