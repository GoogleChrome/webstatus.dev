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
	"testing"
	"time"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

func valuePtr[T any](in T) *T {
	return &in
}

func getSampleRunMetrics() []struct {
	ExternalRunID int64
	WPTRunFeatureMetric
} {
	// nolint: dupl // Okay to duplicate for tests
	return []struct {
		ExternalRunID int64
		WPTRunFeatureMetric
	}{
		// Run 0 metrics
		{
			ExternalRunID: 0,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](20),
				TestPass:   valuePtr[int64](10),
			},
		},
		{
			ExternalRunID: 0,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature2",
				TotalTests: valuePtr[int64](5),
				TestPass:   valuePtr[int64](0),
			},
		},
		{
			ExternalRunID: 0,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature3",
				TotalTests: valuePtr[int64](50),
				TestPass:   valuePtr[int64](5),
			},
		},
		// Run 1 metrics
		{
			ExternalRunID: 1,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](20),
				TestPass:   valuePtr[int64](20),
			},
		},
		// Run 2 metrics
		{
			ExternalRunID: 2,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](20),
				TestPass:   valuePtr[int64](10),
			},
		},
		// Run 3 metrics
		{
			ExternalRunID: 3,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](20),
				TestPass:   valuePtr[int64](10),
			},
		},
		// Run 6 metrics
		{
			ExternalRunID: 6,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](20),
				TestPass:   valuePtr[int64](20),
			},
		},
		{
			ExternalRunID: 6,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature2",
				TotalTests: valuePtr[int64](10),
				TestPass:   valuePtr[int64](0),
			},
		},
		{
			ExternalRunID: 6,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature3",
				TotalTests: valuePtr[int64](50),
				TestPass:   valuePtr[int64](35),
			},
		},
		// Run 7 metrics
		{
			ExternalRunID: 7,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](20),
				TestPass:   valuePtr[int64](20),
			},
		},
		{
			ExternalRunID: 7,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature2",
				TotalTests: valuePtr[int64](10),
				TestPass:   valuePtr[int64](10),
			},
		},
		// Run 8 metrics
		{
			ExternalRunID: 8,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](20),
				TestPass:   valuePtr[int64](20),
			},
		},
		{
			ExternalRunID: 8,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature2",
				TotalTests: valuePtr[int64](10),
				TestPass:   valuePtr[int64](10),
			},
		},
		// Run 9 metrics
		{
			ExternalRunID: 9,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature1",
				TotalTests: valuePtr[int64](20),
				TestPass:   valuePtr[int64](20),
			},
		},
		{
			ExternalRunID: 9,
			WPTRunFeatureMetric: WPTRunFeatureMetric{
				FeatureID:  "feature2",
				TotalTests: valuePtr[int64](10),
				TestPass:   valuePtr[int64](10),
			},
		},
	}
}

func TestUpsertWPTRunFeatureMetric(t *testing.T) {
	client := getTestDatabase(t)
	ctx := context.Background()

	sampleRunMetrics := getSampleRunMetrics()

	// Should fail without the runs and features being uploaded first
	for _, metric := range sampleRunMetrics {
		err := client.UpsertWPTRunFeatureMetric(ctx, metric.ExternalRunID, metric.WPTRunFeatureMetric)
		if err == nil {
			t.Errorf("expected error upon insert")
		}
	}

	// Now, let's insert the runs and features.
	for _, run := range getSampleRuns() {
		err := client.InsertWPTRun(ctx, run)
		if !errors.Is(err, nil) {
			t.Errorf("expected no error upon insert. received %s", err.Error())
		}
	}
	sampleFeatures := getSampleFeatures()
	for _, feature := range sampleFeatures {
		err := client.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert of features. %s", err.Error())
		}
	}

	// Now, let's insert the metrics
	for _, metric := range sampleRunMetrics {
		err := client.UpsertWPTRunFeatureMetric(ctx, metric.ExternalRunID, metric.WPTRunFeatureMetric)
		if !errors.Is(err, nil) {
			t.Errorf("expected no error upon insert. received %s", err.Error())
		}
	}

	metric, err := client.GetMetricByRunIDAndFeatureID(ctx, 0, "feature1")
	if !errors.Is(err, nil) {
		t.Errorf("expected no error when reading the metric. received %s", err.Error())
	}

	if metric == nil {
		t.Fatal("expected non null metric")
	}

	if !reflect.DeepEqual(sampleRunMetrics[0].WPTRunFeatureMetric, *metric) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", sampleRunMetrics[0], *metric)
	}

	// Test 1. Upsert a metric where the run only has one metric.
	// Upsert the metric
	updatedMetric1 := struct {
		ExternalRunID int64
		WPTRunFeatureMetric
	}{
		ExternalRunID: 0,
		WPTRunFeatureMetric: WPTRunFeatureMetric{
			FeatureID:  "feature1",
			TotalTests: valuePtr[int64](300), // Change this value
			TestPass:   valuePtr[int64](100), // Change this value
		},
	}

	err = client.UpsertWPTRunFeatureMetric(ctx, updatedMetric1.ExternalRunID, updatedMetric1.WPTRunFeatureMetric)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error upon insert. received %s", err.Error())
	}

	// Try to get the metric again and compare with the updated metric.
	metric, err = client.GetMetricByRunIDAndFeatureID(ctx, 0, "feature1")
	if !errors.Is(err, nil) {
		t.Errorf("expected no error when reading the metric. received %s", err.Error())
	}

	if metric == nil {
		t.Fatal("expected non null metric")
	}

	if !reflect.DeepEqual(updatedMetric1.WPTRunFeatureMetric, *metric) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", updatedMetric1.WPTRunFeatureMetric, *metric)
	}

	// Test 2. Upsert a metric where the run has multiple metrics.
	updatedMetric2 := struct {
		ExternalRunID int64
		WPTRunFeatureMetric
	}{
		ExternalRunID: 9,
		WPTRunFeatureMetric: WPTRunFeatureMetric{
			FeatureID:  "feature2",
			TotalTests: valuePtr[int64](300), // This value should be changed
			TestPass:   valuePtr[int64](100), // This value should be changed
		},
	}
	// Upsert the metric
	err = client.UpsertWPTRunFeatureMetric(ctx, updatedMetric2.ExternalRunID, updatedMetric2.WPTRunFeatureMetric)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error upon insert. received %s", err.Error())
	}

	// Try to get the metric again and compare with the updated metric.
	metric, err = client.GetMetricByRunIDAndFeatureID(ctx, 9, "feature2")
	if !errors.Is(err, nil) {
		t.Errorf("expected no error when reading the metric. received %s", err.Error())
	}

	if metric == nil {
		t.Fatal("expected non null metric")
	}

	if !reflect.DeepEqual(updatedMetric2.WPTRunFeatureMetric, *metric) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", updatedMetric2.WPTRunFeatureMetric, *metric)
	}

	// Get the other metric for that run which should be unaffected
	metric, err = client.GetMetricByRunIDAndFeatureID(ctx, 9, "feature1")
	if !errors.Is(err, nil) {
		t.Errorf("expected no error when reading the metric. received %s", err.Error())
	}

	if metric == nil {
		t.Fatal("expected non null metric")
	}

	otherMetric := struct {
		WPTRunFeatureMetric
		ExternalRunID int64
	}{
		ExternalRunID: 9,
		WPTRunFeatureMetric: WPTRunFeatureMetric{
			FeatureID:  "feature1",
			TotalTests: valuePtr[int64](20),
			TestPass:   valuePtr[int64](20),
		},
	}
	if !reflect.DeepEqual(otherMetric.WPTRunFeatureMetric, *metric) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", otherMetric.WPTRunFeatureMetric, *metric)
	}
}

func TestListMetricsForFeatureIDBrowserAndChannel(t *testing.T) {
	client := getTestDatabase(t)
	ctx := context.Background()
	// Load up runs, metrics and features.
	sampleFeatures := getSampleFeatures()
	for _, feature := range sampleFeatures {
		err := client.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert of features. %s", err.Error())
		}
	}
	// Now, let's insert the runs.
	for _, run := range getSampleRuns() {
		err := client.InsertWPTRun(ctx, run)
		if !errors.Is(err, nil) {
			t.Errorf("expected no error upon insert. received %s", err.Error())
		}
	}

	// Now, let's insert the metrics
	for _, metric := range getSampleRunMetrics() {
		err := client.UpsertWPTRunFeatureMetric(ctx, metric.ExternalRunID, metric.WPTRunFeatureMetric)
		if !errors.Is(err, nil) {
			t.Errorf("expected no error upon insert. received %s", err.Error())
		}
	}

	// Test 1. Get all the metrics. Should only be 2 for the browser, channel,
	// feature combination.
	metrics, token, err := client.ListMetricsForFeatureIDBrowserAndChannel(
		ctx,
		"feature1",
		"fooBrowser",
		shared.StableLabel,
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		10,
		nil,
	)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token != nil {
		t.Error("expected null token")
	}
	expectedMetrics := []WPTRunFeatureMetricWithTime{
		{
			TimeStart:  time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			RunID:      6,
			TotalTests: valuePtr[int64](20),
			TestPass:   valuePtr[int64](20),
		},
		{
			TimeStart:  time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			RunID:      0,
			TotalTests: valuePtr[int64](20),
			TestPass:   valuePtr[int64](10),
		},
	}
	if !reflect.DeepEqual(expectedMetrics, metrics) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", expectedMetrics, metrics)
	}

	// Test 2. Try pagination. Only return 1 per page.
	// Get page 1
	metrics, token, err = client.ListMetricsForFeatureIDBrowserAndChannel(
		ctx,
		"feature1",
		"fooBrowser",
		shared.StableLabel,
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		1,
		nil,
	)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token == nil {
		t.Error("expected token")
	}
	expectedMetricsPageOne := []WPTRunFeatureMetricWithTime{
		expectedMetrics[0],
	}
	if !reflect.DeepEqual(expectedMetricsPageOne, metrics) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", expectedMetricsPageOne, metrics)
	}
	// Get page 2.
	metrics, token, err = client.ListMetricsForFeatureIDBrowserAndChannel(
		ctx,
		"feature1",
		"fooBrowser",
		shared.StableLabel,
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		1,
		token,
	)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token == nil {
		t.Error("expected token")
	}
	expectedMetricsPageTwo := []WPTRunFeatureMetricWithTime{
		expectedMetrics[1],
	}
	if !reflect.DeepEqual(expectedMetricsPageTwo, metrics) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", expectedMetricsPageTwo, metrics)
	}
	// Get page 3
	metrics, token, err = client.ListMetricsForFeatureIDBrowserAndChannel(
		ctx,
		"feature1",
		"fooBrowser",
		shared.StableLabel,
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		1,
		token,
	)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token != nil {
		t.Error("expected no token")
	}
	var expectedMetricsPageThree []WPTRunFeatureMetricWithTime
	if !reflect.DeepEqual(expectedMetricsPageThree, metrics) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", expectedMetricsPageThree, metrics)
	}
}

func testGetAllAggregatedMetrics(ctx context.Context, client *Client, t *testing.T) {
	// Test 1. Get aggregation metrics for all features.
	metrics, token, err := client.ListMetricsOverTimeWithAggregatedTotals(
		ctx,
		nil,
		"fooBrowser",
		shared.StableLabel,
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		10,
		nil,
	)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token != nil {
		t.Error("expected null token")
	}
	expectedMetrics := []WPTRunAggregationMetricWithTime{
		{
			WPTRunFeatureMetricWithTime: WPTRunFeatureMetricWithTime{
				TimeStart:  time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
				RunID:      6,
				TotalTests: valuePtr[int64](80),
				TestPass:   valuePtr[int64](55),
			},
		},
		{
			WPTRunFeatureMetricWithTime: WPTRunFeatureMetricWithTime{
				TimeStart:  time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				RunID:      0,
				TotalTests: valuePtr[int64](75),
				TestPass:   valuePtr[int64](15),
			},
		},
	}
	if !reflect.DeepEqual(expectedMetrics, metrics) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", expectedMetrics, metrics)
	}
}
func testGetAllAggregatedMetricsPages(ctx context.Context, client *Client, t *testing.T) {
	// Test 2. Get aggregation metrics for all features with pagination.
	// Get page 1.
	metrics, token, err := client.ListMetricsOverTimeWithAggregatedTotals(
		ctx,
		nil,
		"fooBrowser",
		shared.StableLabel,
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		1,
		nil,
	)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token == nil {
		t.Error("expected token")
	}
	expectedMetricsPageOne := []WPTRunAggregationMetricWithTime{
		{
			WPTRunFeatureMetricWithTime: WPTRunFeatureMetricWithTime{
				TimeStart:  time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
				RunID:      6,
				TotalTests: valuePtr[int64](80),
				TestPass:   valuePtr[int64](55),
			},
		},
	}
	if !reflect.DeepEqual(expectedMetricsPageOne, metrics) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", expectedMetricsPageOne, metrics)
	}

	// Get page 2.
	metrics, token, err = client.ListMetricsOverTimeWithAggregatedTotals(
		ctx,
		nil,
		"fooBrowser",
		shared.StableLabel,
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		1,
		token,
	)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token == nil {
		t.Error("expected token")
	}
	expectedMetricsPageTwo := []WPTRunAggregationMetricWithTime{
		{
			WPTRunFeatureMetricWithTime: WPTRunFeatureMetricWithTime{
				TimeStart:  time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				RunID:      0,
				TotalTests: valuePtr[int64](75),
				TestPass:   valuePtr[int64](15),
			},
		},
	}
	if !reflect.DeepEqual(expectedMetricsPageTwo, metrics) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", expectedMetricsPageTwo, metrics)
	}

	// Get page 3.
	metrics, token, err = client.ListMetricsOverTimeWithAggregatedTotals(
		ctx,
		nil,
		"fooBrowser",
		shared.StableLabel,
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		1,
		token,
	)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token != nil {
		t.Error("expected no token")
	}
	var expectedMetricsPageThree []WPTRunAggregationMetricWithTime
	if !reflect.DeepEqual(expectedMetricsPageThree, metrics) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", expectedMetricsPageThree, metrics)
	}
}
func testGetSubsetAggregatedMetrics(ctx context.Context, client *Client, t *testing.T) {
	// Test 3. Get aggregation metrics for subset of features.
	metrics, token, err := client.ListMetricsOverTimeWithAggregatedTotals(
		ctx,
		[]string{"feature2", "feature3"},
		"fooBrowser",
		shared.StableLabel,
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		10,
		nil,
	)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token != nil {
		t.Error("expected null token")
	}
	expectedMetrics := []WPTRunAggregationMetricWithTime{
		{
			WPTRunFeatureMetricWithTime: WPTRunFeatureMetricWithTime{
				TimeStart:  time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
				RunID:      6,
				TotalTests: valuePtr[int64](60),
				TestPass:   valuePtr[int64](35),
			},
		},
		{
			WPTRunFeatureMetricWithTime: WPTRunFeatureMetricWithTime{
				TimeStart:  time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				RunID:      0,
				TotalTests: valuePtr[int64](55),
				TestPass:   valuePtr[int64](5),
			},
		},
	}
	if !reflect.DeepEqual(expectedMetrics, metrics) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", expectedMetrics, metrics)
	}
}
func testGetSubsetAggregatedMetricsPages(ctx context.Context, client *Client, t *testing.T) {
	// Test 4. Get aggregation metrics for subset of features with pagination.
	// Get page 1.
	metrics, token, err := client.ListMetricsOverTimeWithAggregatedTotals(
		ctx,
		[]string{"feature2", "feature3"},
		"fooBrowser",
		shared.StableLabel,
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		1,
		nil,
	)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token == nil {
		t.Error("expected token")
	}
	expectedMetricsPageOne := []WPTRunAggregationMetricWithTime{
		{
			WPTRunFeatureMetricWithTime: WPTRunFeatureMetricWithTime{
				TimeStart:  time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
				RunID:      6,
				TotalTests: valuePtr[int64](60),
				TestPass:   valuePtr[int64](35),
			},
		},
	}
	if !reflect.DeepEqual(expectedMetricsPageOne, metrics) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", expectedMetricsPageOne, metrics)
	}

	// Get page 2.
	metrics, token, err = client.ListMetricsOverTimeWithAggregatedTotals(
		ctx,
		[]string{"feature2", "feature3"},
		"fooBrowser",
		shared.StableLabel,
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		1,
		token,
	)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token == nil {
		t.Error("expected token")
	}
	expectedMetricsPageTwo := []WPTRunAggregationMetricWithTime{
		{
			WPTRunFeatureMetricWithTime: WPTRunFeatureMetricWithTime{
				TimeStart:  time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
				RunID:      0,
				TotalTests: valuePtr[int64](55),
				TestPass:   valuePtr[int64](5),
			},
		},
	}
	if !reflect.DeepEqual(expectedMetricsPageTwo, metrics) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", expectedMetricsPageTwo, metrics)
	}

	// Get page 3.
	metrics, token, err = client.ListMetricsOverTimeWithAggregatedTotals(
		ctx,
		[]string{"feature2", "feature3"},
		"fooBrowser",
		shared.StableLabel,
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		1,
		token,
	)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token != nil {
		t.Error("expected no token")
	}

	var expectedMetricsPageThree []WPTRunAggregationMetricWithTime
	if !reflect.DeepEqual(expectedMetricsPageThree, metrics) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", expectedMetricsPageThree, metrics)
	}
}

func TestListMetricsOverTimeWithAggregatedTotals(t *testing.T) {
	client := getTestDatabase(t)
	ctx := context.Background()
	// Load up runs, metrics and features.
	sampleFeatures := getSampleFeatures()
	for _, feature := range sampleFeatures {
		err := client.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert of features. %s", err.Error())
		}
	}
	// Now, let's insert the runs.
	for _, run := range getSampleRuns() {
		err := client.InsertWPTRun(ctx, run)
		if !errors.Is(err, nil) {
			t.Errorf("expected no error upon insert. received %s", err.Error())
		}
	}

	// Now, let's insert the metrics
	for _, metric := range getSampleRunMetrics() {
		err := client.UpsertWPTRunFeatureMetric(ctx, metric.ExternalRunID, metric.WPTRunFeatureMetric)
		if !errors.Is(err, nil) {
			t.Errorf("expected no error upon insert. received %s", err.Error())
		}
	}
	testGetAllAggregatedMetrics(ctx, client, t)
	testGetAllAggregatedMetricsPages(ctx, client, t)
	testGetSubsetAggregatedMetrics(ctx, client, t)
	testGetSubsetAggregatedMetricsPages(ctx, client, t)
}
