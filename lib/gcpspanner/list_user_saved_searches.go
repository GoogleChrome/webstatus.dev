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

	"cloud.google.com/go/spanner"
)

// userSavedSearchListerMapper implements the necessary interfaces for the generic helpers.
type userSavedSearchListerMapper struct{}

func (m userSavedSearchListerMapper) Table() string {
	return savedSearchesTable
}

func (m userSavedSearchListerMapper) SelectList(req ListUserSavedSearchesRequest) spanner.Statement {
	params := map[string]interface{}{
		"userID":   req.UserID,
		"pageSize": req.PageSize,
	}

	tmplData := listUserSavedSearchTemplateData{
		PageFilter: "",
	}

	if req.PageToken != nil {
		cursor, err := decodeUserSavedSearchesCursor(*req.PageToken)
		if err == nil {
			params["lastName"] = cursor.LastName
			params["lastID"] = cursor.LastID
			tmplData.PageFilter = commonListUserSavedSearchesPaginationRawTemplate
		}
	}

	tmpl := listUserSavedSearchesBaseTemplate.Execute(tmplData)
	stmt := spanner.NewStatement(tmpl)
	stmt.Params = params

	return stmt
}

func (m userSavedSearchListerMapper) EncodePageToken(item UserSavedSearch) string {
	return encodeUserSavedSearchesCursor(item.ID, item.Name)
}

// ListUserSavedSearchesRequest is a request to list user saved searches.
type ListUserSavedSearchesRequest struct {
	UserID    string
	PageSize  int
	PageToken *string
}

// GetPageSize returns the page size for the request.
func (r ListUserSavedSearchesRequest) GetPageSize() int {
	return r.PageSize
}

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

// TODO: Change signature to take ListUserSavedSearchesRequest.
func (c *Client) ListUserSavedSearches(
	ctx context.Context,
	userID string,
	pageSize int,
	pageToken *string) (*UserSavedSearchesPage, error) {
	req := ListUserSavedSearchesRequest{
		UserID:    userID,
		PageSize:  pageSize,
		PageToken: pageToken,
	}
	lister := newEntityLister[userSavedSearchListerMapper, UserSavedSearch, ListUserSavedSearchesRequest](c)
	results, nextPageToken, err := lister.list(ctx, req)
	if err != nil {
		return nil, err
	}

	return &UserSavedSearchesPage{
		Searches:      results,
		NextPageToken: nextPageToken,
	}, nil
}

type savedSearchIDContainer struct {
	ID string `spanner:"ID"`
}

func (m userSavedSearchListerMapper) SelectAll() spanner.Statement {
	return spanner.Statement{
		SQL:    "SELECT ID FROM SavedSearches",
		Params: nil,
	}
}

// Used by the Cloud Scheduler batch job to find all entities to process.
func (c *Client) ListAllSavedSearchIDs(
	ctx context.Context) ([]string, error) {
	ret, err := newAllEntityReader[userSavedSearchListerMapper, savedSearchIDContainer](c).readAll(ctx)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(ret))
	for _, r := range ret {
		ids = append(ids, r.ID)
	}

	return ids, nil
}
