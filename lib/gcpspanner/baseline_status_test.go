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
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func getSampleBaselineStatuses() []struct {
	featureKey string
	status     FeatureBaselineStatus
} {
	return []struct {
		featureKey string
		status     FeatureBaselineStatus
	}{
		{
			featureKey: "feature1",
			status: FeatureBaselineStatus{
				Status:   nil,
				LowDate:  nil,
				HighDate: nil,
			},
		},
		{
			featureKey: "feature2",
			status: FeatureBaselineStatus{
				Status:   valuePtr(BaselineStatusHigh),
				LowDate:  valuePtr[time.Time](time.Date(2000, time.January, 15, 0, 0, 0, 0, time.UTC)),
				HighDate: valuePtr[time.Time](time.Date(2000, time.January, 31, 0, 0, 0, 0, time.UTC)),
			},
		},
	}
}

func setupRequiredTablesForBaselineStatus(ctx context.Context,
	client *Client, t *testing.T) {
	sampleFeatures := getSampleFeatures()
	for _, feature := range sampleFeatures {
		_, err := client.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert of features. %s", err.Error())
		}
	}
}

// Helper method to get all the statuses in a stable order.
func (c *Client) ReadAllBaselineStatuses(ctx context.Context, _ *testing.T) ([]FeatureBaselineStatus, error) {
	stmt := spanner.NewStatement("SELECT * FROM FeatureBaselineStatus ORDER BY HighDate ASC")
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []FeatureBaselineStatus
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var status spannerFeatureBaselineStatus
		if err := row.ToStruct(&status); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}

		status.Status = (*BaselineStatus)(status.InternalStatus)
		ret = append(ret, status.FeatureBaselineStatus)
	}

	return ret, nil
}
func statusEquality(left, right FeatureBaselineStatus) bool {
	return reflect.DeepEqual(left.Status, right.Status) &&
		((left.LowDate != nil && right.LowDate != nil && left.LowDate.Equal(*right.LowDate)) ||
			left.LowDate == right.LowDate) &&
		((left.HighDate != nil && right.HighDate != nil && left.HighDate.Equal(*right.HighDate)) ||
			left.LowDate == right.LowDate)
}

func TestUpsertFeatureBaselineStatus(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	setupRequiredTablesForBaselineStatus(ctx, spannerClient, t)
	sampleStatuses := getSampleBaselineStatuses()

	expectedStatuses := make([]FeatureBaselineStatus, 0, len(sampleStatuses))
	for _, status := range sampleStatuses {
		expectedStatuses = append(expectedStatuses, status.status)
		err := spannerClient.UpsertFeatureBaselineStatus(ctx, status.featureKey, status.status)
		if err != nil {
			t.Errorf("unexpected error during insert. %s", err.Error())
		}
	}

	statuses, err := spannerClient.ReadAllBaselineStatuses(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	if !slices.EqualFunc[[]FeatureBaselineStatus](
		expectedStatuses,
		statuses, statusEquality) {
		t.Errorf("unequal status.\nexpected %+v\nreceived %+v", expectedStatuses, statuses)
	}

	err = spannerClient.UpsertFeatureBaselineStatus(ctx, "feature1", FeatureBaselineStatus{
		Status:   valuePtr(BaselineStatusHigh),
		LowDate:  valuePtr[time.Time](time.Date(2000, time.February, 15, 0, 0, 0, 0, time.UTC)),
		HighDate: valuePtr[time.Time](time.Date(2000, time.February, 28, 0, 0, 0, 0, time.UTC)),
	})
	if err != nil {
		t.Errorf("unexpected error during update. %s", err.Error())
	}

	expectedPageAfterUpdate := []FeatureBaselineStatus{
		{
			Status:   valuePtr(BaselineStatusHigh),
			LowDate:  valuePtr[time.Time](time.Date(2000, time.January, 15, 0, 0, 0, 0, time.UTC)),
			HighDate: valuePtr[time.Time](time.Date(2000, time.January, 31, 0, 0, 0, 0, time.UTC)),
		},
		{
			Status:   valuePtr(BaselineStatusHigh),
			LowDate:  valuePtr[time.Time](time.Date(2000, time.February, 15, 0, 0, 0, 0, time.UTC)),
			HighDate: valuePtr[time.Time](time.Date(2000, time.February, 28, 0, 0, 0, 0, time.UTC)),
		},
	}

	statuses, err = spannerClient.ReadAllBaselineStatuses(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all after update. %s", err.Error())
	}
	if !slices.EqualFunc[[]FeatureBaselineStatus](
		expectedPageAfterUpdate,
		statuses, statusEquality) {
		t.Errorf("unequal status.\nexpected %+v\nreceived %+v", expectedPageAfterUpdate, statuses)
	}
}
