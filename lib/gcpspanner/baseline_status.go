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
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
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

// spannerFeatureBaselineStatus is a wrapper for the baseline status that is actually
// stored in spanner.
type spannerFeatureBaselineStatus struct {
	WebFeatureID   string  `spanner:"WebFeatureID"`
	InternalStatus *string `spanner:"Status"`
	FeatureBaselineStatus
}

// Implements the entityMapper interface for FeatureBaselineStatus and SpannerFeatureBaselineStatus.
type baselineStatusMapper struct{}

func (m baselineStatusMapper) Table() string {
	return featureBaselineStatusTable
}

func (m baselineStatusMapper) SelectOne(id string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		WebFeatureID, Status, LowDate, HighDate
	FROM %s
	WHERE WebFeatureID = @webFeatureID
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"webFeatureID": id,
	}
	stmt.Params = parameters

	return stmt
}

func (m baselineStatusMapper) Merge(in spannerFeatureBaselineStatus,
	existing spannerFeatureBaselineStatus) spannerFeatureBaselineStatus {
	// Only allow overriding of the status, low date and high date.
	return spannerFeatureBaselineStatus{
		WebFeatureID:   existing.WebFeatureID,
		InternalStatus: in.InternalStatus,
		FeatureBaselineStatus: FeatureBaselineStatus{
			LowDate:  in.LowDate,
			HighDate: in.HighDate,
			// Status does not need to be set.
			Status: nil,
		},
	}
}

func (m baselineStatusMapper) GetKeyFromExternal(in spannerFeatureBaselineStatus) string {
	return in.WebFeatureID
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
	// Create status based on the table model.
	status := spannerFeatureBaselineStatus{
		WebFeatureID:          *id,
		InternalStatus:        (*string)(input.Status),
		FeatureBaselineStatus: input,
	}

	return newEntityWriter[baselineStatusMapper](c).upsert(ctx, status)
}
