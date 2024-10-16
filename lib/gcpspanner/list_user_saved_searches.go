package gcpspanner

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
)

type UserSavedSearch struct {
	SavedSearch
	// The following fields will be nil if the user is not authenticated.
	Role         *SavedSearchUserRole `spanner:"Role"`
	IsBookmarked *bool                `spanner:"IsBookmarked"`
}

type baseUserSaveSearchMapper struct{}

func (m baseUserSaveSearchMapper) Table() string {
	return savedSearchesTable
}

func (m baseUserSaveSearchMapper) GetKey(in SavedSearch) string {
	return in.ID
}

type unauthenticatedUserSavedSearchMapper struct {
	baseUserSaveSearchMapper
}

func (m unauthenticatedUserSavedSearchMapper) SelectOne(
	key string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID,
		Name,
		Query,
		Scope,
		AuthorID,
		CreatedAt,
		UpdatedAt
	FROM %s
	WHERE ID = @id
	LIMIT 1`,
		m.Table()))
	parameters := map[string]interface{}{
		"id": key,
	}
	stmt.Params = parameters

	return stmt
}

type authenticatedUserSavedSearchMapperKey struct {
	ID     string
	UserID string
}

type authenticatedUserSavedSearchMapper struct {
	baseUserSaveSearchMapper
}

func (m authenticatedUserSavedSearchMapper) GetKey(in UserSavedSearch) authenticatedUserSavedSearchMapperKey {
	return authenticatedUserSavedSearchMapperKey{}
}

func (m authenticatedUserSavedSearchMapper) SelectOne(
	key authenticatedUserSavedSearchMapperKey) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID,
		Name,
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
	FROM %s s
	LEFT JOIN
    SavedSearchUserRoles r ON s.ID = r.SavedSearchID AND r.UserID = @userID
LEFT JOIN
    UserSavedSearchBookmarks b ON s.ID = b.SavedSearchID AND b.UserID = @userID
	WHERE s.ID = @id
	LIMIT 1`,
		m.Table()))
	parameters := map[string]interface{}{
		"id":     key.ID,
		"userID": key.UserID,
	}
	stmt.Params = parameters

	return stmt
}

func (c *Client) GetUserSavedSearch(ctx context.Context, savedSearchID string, authenticatedUserID *string) (*UserSavedSearch, error) {
	if authenticatedUserID == nil {
		row, err := newEntityReader[unauthenticatedUserSavedSearchMapper, SavedSearch](c).readRowByKey(ctx, savedSearchID)
		if err != nil {
			return nil, err
		}

		return &UserSavedSearch{
			SavedSearch:  *row,
			IsBookmarked: nil,
			Role:         nil,
		}, nil
	}
	newEntityReader[authenticatedUserSavedSearchMapper, UserSavedSearch, authenticatedUserSavedSearchMapperKey](c).readRowByKey(ctx, authenticatedUserSavedSearchMapperKey{
		UserID: *authenticatedUserID,
		ID:     savedSearchID,
	})
}
