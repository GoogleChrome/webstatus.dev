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
	"cmp"
	"context"
	"errors"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const featureBaselineStatusTable = "FeatureBaselineStatus"

// Options come from
// https://github.com/web-platform-dx/web-features/blob/3d4d066c47c9f07514bf743b3955572a6073ff1e/packages/web-features/README.md
// nolint: lll
type BaselineStatus string

const (
	BaselineStatusNone BaselineStatus = "none"
	BaselineStatusLow  BaselineStatus = "low"
	BaselineStatusHigh BaselineStatus = "high"
)

// SpannerFeatureBaselineStatus is a wrapper for the baseline status that is actually
// stored in spanner.
type SpannerFeatureBaselineStatus struct {
	WebFeatureID   string  `spanner:"WebFeatureID"`
	InternalStatus *string `spanner:"Status"`
	FeatureBaselineStatus
}

// FeatureBaselineStatus contains information about the current baseline status
// of a feature.
type FeatureBaselineStatus struct {
	Status   *BaselineStatus `spanner:"-"` // Spanner can not handle pointer to custom type. So ignore it.
	LowDate  *time.Time      `spanner:"LowDate"`
	HighDate *time.Time      `spanner:"HighDate"`
}

// UpsertWebFeature will update the given baseline status.
// If the status, does not exist, it will insert a new status.
// If the status exists, it will allow updates to the status, low date and high date.
func (c *Client) UpsertFeatureBaselineStatus(ctx context.Context,
	featureKey string, input FeatureBaselineStatus) error {
	id, err := c.GetIDFromFeatureKey(ctx, NewFeatureKeyFilter(featureKey))
	if err != nil {
		return err
	}
	if id == nil {
		return ErrInternalQueryFailure
	}
	_, err = c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.NewStatement(`
		SELECT
			WebFeatureID, Status, LowDate, HighDate
		FROM FeatureBaselineStatus
		WHERE WebFeatureID = @webFeatureID
		LIMIT 1`)
		parameters := map[string]interface{}{
			"webFeatureID": *id,
		}
		stmt.Params = parameters

		// Create status based on the table model.
		status := SpannerFeatureBaselineStatus{
			WebFeatureID:          *id,
			InternalStatus:        (*string)(input.Status),
			FeatureBaselineStatus: input,
		}

		// Attempt to query for the row.
		it := txn.Query(ctx, stmt)
		defer it.Stop()
		var m *spanner.Mutation

		row, err := it.Next()
		// nolint: nestif // TODO: fix in the future.
		if err != nil {
			if errors.Is(err, iterator.Done) {
				// No rows returned. Act as if this is an insertion.
				var err error
				m, err = spanner.InsertOrUpdateStruct(featureBaselineStatusTable, status)
				if err != nil {
					return errors.Join(ErrInternalQueryFailure, err)
				}
			} else {
				// An unexpected error occurred.

				return errors.Join(ErrInternalQueryFailure, err)
			}
		} else {
			// Read the existing status and merge the values.
			var existingStatus SpannerFeatureBaselineStatus
			err = row.ToStruct(&existingStatus)
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
			// Only allow overriding of the status, low date and high date.
			existingStatus.InternalStatus = cmp.Or[*string](status.InternalStatus, existingStatus.InternalStatus)
			existingStatus.LowDate = cmp.Or[*time.Time](status.LowDate, existingStatus.LowDate)
			existingStatus.HighDate = cmp.Or[*time.Time](status.HighDate, existingStatus.HighDate)
			m, err = spanner.InsertOrUpdateStruct(featureBaselineStatusTable, existingStatus)
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
		}
		// Buffer the mutation to be committed.
		err = txn.BufferWrite([]*spanner.Mutation{m})
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}

		return nil
	})
	if err != nil {
		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}
