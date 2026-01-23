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
	"reflect"
	"testing"
	"time"
)

func TestListBaselineStatusCounts_LowDate(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	loadDataForListBaselineStatusCounts(ctx, t)

	startAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	pageSize := 10

	expected := &BaselineStatusCountResultPage{
		Metrics: []BaselineStatusCountMetric{
			{Date: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), StatusCount: 1},
			{Date: time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC), StatusCount: 2},
			{Date: time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC), StatusCount: 3},
			{Date: time.Date(2024, 4, 25, 0, 0, 0, 0, time.UTC), StatusCount: 5},
		},
		NextPageToken: nil,
	}

	result, err := spannerClient.ListBaselineStatusCounts(ctx, BaselineDateTypeLow, startAt, endAt, pageSize, nil)
	if err != nil {
		t.Fatalf("ListBaselineStatusCounts failed: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Unexpected result. Got: %+v, Want: %+v", result, expected)
	}
}

func TestListBaselineStatusCounts_Pagination(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	loadDataForListBaselineStatusCounts(ctx, t)

	startAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	pageSize := 3

	// First page
	result1, err := spannerClient.ListBaselineStatusCounts(ctx, BaselineDateTypeLow, startAt, endAt, pageSize, nil)
	if err != nil {
		t.Fatalf("ListBaselineStatusCounts failed: %v", err)
	}

	expected1 := &BaselineStatusCountResultPage{
		Metrics: []BaselineStatusCountMetric{
			{Date: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), StatusCount: 1},
			{Date: time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC), StatusCount: 2},
			{Date: time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC), StatusCount: 3},
		},
		NextPageToken: valuePtr(encodeBaselineStatusCountCursor(time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC), 3)),
	}

	if !reflect.DeepEqual(result1, expected1) {
		t.Errorf("Unexpected result for first page. Got: %+v, Want: %+v", result1, expected1)
	}

	// Second page
	result2, err := spannerClient.ListBaselineStatusCounts(
		ctx, BaselineDateTypeLow, startAt, endAt, pageSize, result1.NextPageToken)
	if err != nil {
		t.Fatalf("ListBaselineStatusCounts failed: %v", err)
	}

	expected2 := &BaselineStatusCountResultPage{
		Metrics: []BaselineStatusCountMetric{
			{Date: time.Date(2024, 4, 25, 0, 0, 0, 0, time.UTC), StatusCount: 5},
		},
		NextPageToken: nil, // No more pages
	}

	if !reflect.DeepEqual(result2, expected2) {
		t.Errorf("Unexpected result for second page. Got: %+v, Want: %+v", result2, expected2)
	}
}

func TestListBaselineStatusCounts_ExcludedFeatures(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	loadDataForListBaselineStatusCounts(ctx, t)

	// Exclude "FeatureB"
	err := spannerClient.InsertExcludedFeatureKey(ctx, "FeatureB")
	if err != nil {
		t.Fatalf("Failed to insert excluded feature key: %v", err)
	}

	// Discourage FeatureE
	err = spannerClient.UpsertFeatureDiscouragedDetails(ctx, "FeatureE", FeatureDiscouragedDetails{
		AccordingTo:  nil,
		Alternatives: nil,
	})
	if err != nil {
		t.Fatalf("UpsertFeatureDiscouragedDetails failed: %v", err)
	}

	startAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	pageSize := 10

	expected := &BaselineStatusCountResultPage{
		Metrics: []BaselineStatusCountMetric{
			{Date: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), StatusCount: 1},
			{Date: time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC), StatusCount: 2},
			{Date: time.Date(2024, 4, 25, 0, 0, 0, 0, time.UTC), StatusCount: 3},
		},
		NextPageToken: nil,
	}

	result, err := spannerClient.ListBaselineStatusCounts(ctx, BaselineDateTypeLow, startAt, endAt, pageSize, nil)
	if err != nil {
		t.Fatalf("ListBaselineStatusCounts failed: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Unexpected result. Got: %+v, Want: %+v", result, expected)
	}
}

func loadDataForListBaselineStatusCounts(ctx context.Context, t *testing.T) {
	// Insert web features
	webFeatures := []WebFeature{
		{FeatureKey: "FeatureA", Name: "Feature A", Description: "text", DescriptionHTML: "<html>"},
		{FeatureKey: "FeatureB", Name: "Feature B", Description: "text", DescriptionHTML: "<html>"},
		{FeatureKey: "FeatureC", Name: "Feature C", Description: "text", DescriptionHTML: "<html>"},
		{FeatureKey: "FeatureD", Name: "Feature D", Description: "text", DescriptionHTML: "<html>"},
		{FeatureKey: "FeatureE", Name: "Feature E", Description: "text", DescriptionHTML: "<html>"},
	}
	for _, wf := range webFeatures {
		_, err := spannerClient.upsertWebFeature(ctx, wf)
		if err != nil {
			t.Fatalf("UpsertWebFeature failed: %v", err)
		}
	}

	// Insert feature baseline statuses
	fbs := []struct {
		featureKey string
		status     BaselineStatus
		lowDate    time.Time
		highDate   *time.Time
	}{
		{"FeatureA", BaselineStatusLow, time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), nil},
		{"FeatureB", BaselineStatusLow, time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC), nil},
		{"FeatureC", BaselineStatusHigh, time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC),
			valuePtr(time.Date(2024, 5, 20, 0, 0, 0, 0, time.UTC))},
		{"FeatureD", BaselineStatusLow, time.Date(2024, 4, 25, 0, 0, 0, 0, time.UTC), nil},
		{"FeatureE", BaselineStatusLow, time.Date(2024, 4, 25, 0, 0, 0, 0, time.UTC), nil},
	}

	for _, s := range fbs {
		err := spannerClient.UpsertFeatureBaselineStatus(ctx, s.featureKey, FeatureBaselineStatus{
			Status:   &s.status,
			LowDate:  &s.lowDate,
			HighDate: s.highDate,
		})
		if err != nil {
			t.Fatalf("UpsertFeatureBaselineStatus failed: %v", err)
		}
	}
}
