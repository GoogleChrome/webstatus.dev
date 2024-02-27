package gcpspanner

import (
	"context"
	"reflect"
	"testing"
)

func setupRequiredTablesForFeaturesSearch(ctx context.Context,
	client *Client, t *testing.T) {
	sampleFeatures := getSampleFeatures()
	for _, feature := range sampleFeatures {
		err := client.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert of features. %s", err.Error())
		}
	}
	for _, release := range getSampleBrowserReleases() {
		err := client.InsertBrowserRelease(ctx, release)
		if err != nil {
			t.Errorf("unexpected error during insert of releases. %s", err.Error())
		}
	}

	for _, availability := range getSampleBrowserAvailabilities() {
		err := client.InsertBrowserFeatureAvailability(ctx, availability)
		if err != nil {
			t.Errorf("unexpected error during insert of availabilities. %s", err.Error())
		}
	}

	for _, status := range getSampleBaselineStatuses() {
		err := client.UpsertFeatureBaselineStatus(ctx, status)
		if err != nil {
			t.Errorf("unexpected error during insert of statuses. %s", err.Error())
		}
	}

	for _, run := range getSampleRuns() {
		err := client.InsertWPTRun(ctx, run)
		if err != nil {
			t.Errorf("unexpected error during insert of runs. %s", err.Error())
		}
	}

	for _, metric := range getSampleRunMetrics() {
		err := client.UpsertWPTRunFeatureMetric(ctx, metric)
		if err != nil {
			t.Errorf("unexpected error during insert of metrics. %s", err.Error())
		}
	}
}

func TestFeaturesSearch(t *testing.T) {
	client := getTestDatabase(t)
	ctx := context.Background()
	setupRequiredTablesForFeaturesSearch(ctx, client, t)

	results, _, err := client.FeaturesSearch(ctx, nil, 100)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}

	expectedResults := []FeatureResult{}
	if !reflect.DeepEqual(expectedResults, results) {
		t.Errorf("unequal results. expected (%+v) received (%+v) ", expectedResults, results)
	}
}
