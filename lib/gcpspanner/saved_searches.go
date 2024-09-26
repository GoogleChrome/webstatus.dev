package gcpspanner

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

type NewSavedSearchRequest struct {
	Name        string
	Query       string
	OwnerUserID string
}

type SearchConfig struct {
	MaxOwnedSearchesPerUser uint32
}

var ErrOwnerSavedSearchLimitExceeded = errors.New("saved search limit reached")

type SavedSearch struct {
	ID        string    `spanner:"ID"`
	Name      string    `spanner:"Name"`
	Query     string    `spanner:"Query"`
	CreatedAt time.Time `spanner:"CreatedAt"`
	UpdatedAt time.Time `spanner:"UpdatedAt"`
}

func (c *Client) CreateNewSavedSearch(ctx context.Context, cfg SearchConfig, newSearch NewSavedSearchRequest) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Read the current count of owned searches
		var count int64
		stmt := spanner.Statement{
			SQL: `SELECT COUNT(*)
                  FROM SavedSearchRoles
                  WHERE UserID = @OwnerID AND UserRole = @Role`,
			Params: map[string]interface{}{
				"OwnerID": newSearch.OwnerUserID,
				"Role":    OwnerRole,
			},
		}
		row, err := txn.Query(ctx, stmt).Next()
		if err != nil {
			return err
		}
		if err := row.Columns(&count); err != nil {
			return err
		}

		// 2. Check against the limit
		if count >= int64(cfg.MaxOwnedSearchesPerUser) {
			return ErrOwnerSavedSearchLimitExceeded
		}

		uuid.NewString()
		// 3. Proceed with insertion if within limits
		stmt = spanner.Statement{
			// ... (your insert statement) ...
		}
		_, err = txn.Update(ctx, stmt)
		return err
	})

	return err
}
