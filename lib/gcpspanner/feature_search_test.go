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
	"math/big"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
	"github.com/GoogleChrome/webstatus.dev/lib/webfeaturesmappingtypes"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

func getDefaultTestBrowserList() []string {
	return []string{
		"fooBrowser",
		"barBrowser",
	}
}

// nolint:gocognit // TODO: break this into smaller methods.
func setupRequiredTablesForFeaturesSearch(ctx context.Context,
	client *Client, t *testing.T) {
	webFeatureKeyToInternalFeatureID := map[string]string{}
	//nolint: dupl // Okay to duplicate for tests
	sampleFeatures := []WebFeature{
		{
			Name:            "Feature 1",
			FeatureKey:      "feature1",
			Description:     "@container",
			DescriptionHTML: "<html>",
		},
		{
			Name:            "Feature 2",
			FeatureKey:      "feature2",
			Description:     "feature 2 description",
			DescriptionHTML: "<b>feature 2</b>",
		},
		{
			Name:            "Feature 3",
			FeatureKey:      "feature3",
			Description:     "feature 3 description",
			DescriptionHTML: "<b>feature 3</b>",
		},
		{
			Name:            "Feature 4",
			FeatureKey:      "feature4",
			Description:     "feature 4 description",
			DescriptionHTML: "<b>feature 4</b>",
		},
		{
			Name:            "Feature 5",
			FeatureKey:      "feature5",
			Description:     "feature 5 description",
			DescriptionHTML: "<b>feature 5</b>",
		},
	}
	for _, feature := range sampleFeatures {
		id, err := client.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert of features. %s", err.Error())
		}
		webFeatureKeyToInternalFeatureID[feature.FeatureKey] = *id
	}

	// Insert excluded feature 5
	err := client.InsertExcludedFeatureKey(ctx, "feature5")
	if err != nil {
		t.Errorf("unexpected error during insert of excluded keys. %s", err.Error())
	}

	sampleVendorPositions := []WebFeaturesMappingData{
		{
			WebFeatureID: "feature1",
			VendorPositions: spanner.NullJSON{
				Value: []webfeaturesmappingtypes.StandardsPosition{
					{
						Vendor:   "chrome",
						Position: "positive",
						URL:      "https://example.com/feature1",
						Concerns: nil,
					},
				},
				Valid: true,
			},
		},
		{
			WebFeatureID: "feature2",
			VendorPositions: spanner.NullJSON{
				Value: []webfeaturesmappingtypes.StandardsPosition{
					{
						Vendor:   "safari",
						Position: "negative",
						URL:      "https://example.com/feature2",
						Concerns: nil,
					},
				},
				Valid: true,
			},
		},
	}

	err = client.SyncWebFeaturesMappingData(ctx, sampleVendorPositions)
	if err != nil {
		t.Errorf("unexpected error during sync of vendor positions. %s", err.Error())
	}

	// nolint: dupl // Okay to duplicate for tests
	sampleReleases := []BrowserRelease{
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "0.0.0",
			ReleaseDate:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "0.0.0",
			ReleaseDate:    time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "1.0.0",
			ReleaseDate:    time.Date(2000, time.February, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "1.0.0",
			ReleaseDate:    time.Date(2000, time.February, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "2.0.0",
			ReleaseDate:    time.Date(2000, time.March, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "2.0.0",
			ReleaseDate:    time.Date(2000, time.March, 2, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, release := range sampleReleases {
		err := client.InsertBrowserRelease(ctx, release)
		if err != nil {
			t.Errorf("unexpected error during insert of releases. %s", err.Error())
		}
	}

	//nolint: dupl // Okay to duplicate for tests
	sampleFeatureBrowserAvailabilities := map[string][]BrowserFeatureAvailability{
		"feature1": {
			{
				BrowserName:    "fooBrowser",
				BrowserVersion: "0.0.0",
			},
			{
				BrowserName:    "barBrowser",
				BrowserVersion: "1.0.0",
			},
		},
		"feature2": {
			{
				BrowserName:    "barBrowser",
				BrowserVersion: "2.0.0",
			},
		},
		"feature3": {
			{
				BrowserName:    "fooBrowser",
				BrowserVersion: "1.0.0",
			},
		},
	}

	err = client.SyncBrowserFeatureAvailabilities(ctx, sampleFeatureBrowserAvailabilities)
	if err != nil {
		t.Errorf("unexpected error during insert of availabilities. %s", err.Error())
	}

	//nolint: dupl // Okay to duplicate for tests
	sampleBaselineStatuses := []struct {
		featureKey string
		status     FeatureBaselineStatus
	}{
		{
			featureKey: "feature1",
			status: FeatureBaselineStatus{
				Status:   valuePtr(BaselineStatusLow),
				LowDate:  valuePtr[time.Time](time.Date(2000, time.January, 5, 0, 0, 0, 0, time.UTC)),
				HighDate: nil,
			},
		},
		{
			featureKey: "feature2",
			status: FeatureBaselineStatus{
				Status:   valuePtr(BaselineStatusHigh),
				LowDate:  valuePtr[time.Time](time.Date(2000, time.January, 4, 0, 0, 0, 0, time.UTC)),
				HighDate: valuePtr[time.Time](time.Date(2000, time.January, 31, 0, 0, 0, 0, time.UTC)),
			},
		},
		{
			featureKey: "feature3",
			status: FeatureBaselineStatus{
				Status:   valuePtr(BaselineStatusNone),
				LowDate:  nil,
				HighDate: nil,
			},
		},
		// feature4 will default to nil.
	}
	for _, status := range sampleBaselineStatuses {
		err := client.UpsertFeatureBaselineStatus(ctx, status.featureKey, status.status)
		if err != nil {
			t.Errorf("unexpected error during insert of statuses. %s", err.Error())
		}
	}

	addSampleChromiumUsageMetricsData(ctx, client, t, webFeatureKeyToInternalFeatureID)

	// nolint: dupl // Okay to duplicate for tests
	sampleRuns := []WPTRun{
		{
			RunID:            0,
			TimeStart:        time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:      "fooBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.StableLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            1,
			TimeStart:        time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:      "fooBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.ExperimentalLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            2,
			TimeStart:        time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:      "barBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.StableLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            3,
			TimeStart:        time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC),
			BrowserName:      "barBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.ExperimentalLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            6,
			TimeStart:        time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:      "fooBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.StableLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            7,
			TimeStart:        time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:      "fooBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.ExperimentalLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            8,
			TimeStart:        time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:      "barBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.StableLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
		{
			RunID:            9,
			TimeStart:        time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
			TimeEnd:          time.Date(2000, time.January, 2, 1, 0, 0, 0, time.UTC),
			BrowserName:      "barBrowser",
			BrowserVersion:   "0.0.0",
			Channel:          shared.ExperimentalLabel,
			OSName:           "os",
			OSVersion:        "0.0.0",
			FullRevisionHash: "abcdef0123456789",
		},
	}

	for _, run := range sampleRuns {
		err := client.InsertWPTRun(ctx, run)
		if err != nil {
			t.Errorf("unexpected error during insert of runs. %s", err.Error())
		}
	}

	// nolint: dupl // Okay to duplicate for tests
	sampleRunMetrics := []struct {
		ExternalRunID int64
		Metrics       map[string]WPTRunFeatureMetric
	}{
		// Run 0 metrics - fooBrowser - stable
		{
			ExternalRunID: 0,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:    valuePtr[int64](20),
					TestPass:      valuePtr[int64](10),
					TotalSubtests: valuePtr[int64](220),
					SubtestPass:   valuePtr[int64](110),
					FeatureRunDetails: map[string]interface{}{
						"test": "stale-foo-stable",
					},
				},
				"feature2": {
					TotalTests:        valuePtr[int64](5),
					TestPass:          valuePtr[int64](0),
					TotalSubtests:     valuePtr[int64](55),
					SubtestPass:       valuePtr[int64](11),
					FeatureRunDetails: nil,
				},
				"feature3": {
					TotalTests:        valuePtr[int64](50),
					TestPass:          valuePtr[int64](5),
					TotalSubtests:     valuePtr[int64](5000),
					SubtestPass:       valuePtr[int64](150),
					FeatureRunDetails: nil,
				},
			},
		},
		// Run 1 metrics - fooBrowser - experimental
		{
			ExternalRunID: 1,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:        valuePtr[int64](20),
					TestPass:          valuePtr[int64](20),
					TotalSubtests:     valuePtr[int64](200),
					SubtestPass:       valuePtr[int64](200),
					FeatureRunDetails: nil,
				},
			},
		},
		// Run 2 metrics - barBrowser - stable
		{
			ExternalRunID: 2,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:        valuePtr[int64](20),
					TestPass:          valuePtr[int64](10),
					TotalSubtests:     valuePtr[int64](200),
					SubtestPass:       valuePtr[int64](15),
					FeatureRunDetails: nil,
				},
			},
		},
		// Run 3 metrics - barBrowser - experimental
		{
			ExternalRunID: 3,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:        valuePtr[int64](20),
					TestPass:          valuePtr[int64](10),
					TotalSubtests:     valuePtr[int64](700),
					SubtestPass:       valuePtr[int64](250),
					FeatureRunDetails: nil,
				},
			},
		},
		// Run 6 metrics - fooBrowser - stable
		{
			ExternalRunID: 6,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:    valuePtr[int64](20),
					TestPass:      valuePtr[int64](20),
					TotalSubtests: valuePtr[int64](1000),
					SubtestPass:   valuePtr[int64](1000),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest1-foo-stable",
					},
				},
				"feature2": {
					TotalTests:    valuePtr[int64](10),
					TestPass:      valuePtr[int64](0),
					TotalSubtests: valuePtr[int64](100),
					SubtestPass:   valuePtr[int64](15),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest2-foo-stable",
					},
				},
				"feature3": {
					TotalTests:    valuePtr[int64](50),
					TestPass:      valuePtr[int64](35),
					TotalSubtests: valuePtr[int64](9000),
					SubtestPass:   valuePtr[int64](4000),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest3-foo-stable",
					},
				},
			},
		},
		// Run 7 metrics - fooBrowser - experimental
		{
			ExternalRunID: 7,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:    valuePtr[int64](11),
					TestPass:      valuePtr[int64](11),
					TotalSubtests: valuePtr[int64](11),
					SubtestPass:   valuePtr[int64](11),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest1-foo-exp",
					},
				},
				"feature2": {
					TotalTests:    valuePtr[int64](12),
					TestPass:      valuePtr[int64](12),
					TotalSubtests: valuePtr[int64](12),
					SubtestPass:   valuePtr[int64](12),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest2-foo-exp",
					},
				},
			},
		},
		// Run 8 metrics - barBrowser - stable
		{
			ExternalRunID: 8,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:    valuePtr[int64](33),
					TestPass:      valuePtr[int64](33),
					TotalSubtests: valuePtr[int64](333),
					SubtestPass:   valuePtr[int64](333),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest1-bar-stable",
					},
				},
				"feature2": {
					TotalTests:    valuePtr[int64](10),
					TestPass:      valuePtr[int64](10),
					TotalSubtests: valuePtr[int64](100),
					SubtestPass:   valuePtr[int64](100),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest2-bar-stable",
					},
				},
			},
		},
		// Run 9 metrics - barBrowser - experimental
		{
			ExternalRunID: 9,
			Metrics: map[string]WPTRunFeatureMetric{
				"feature1": {
					TotalTests:        valuePtr[int64](220),
					TestPass:          valuePtr[int64](220),
					TotalSubtests:     valuePtr[int64](2220),
					SubtestPass:       valuePtr[int64](2220),
					FeatureRunDetails: nil,
				},
				"feature2": {
					TotalTests:    valuePtr[int64](120),
					TestPass:      valuePtr[int64](120),
					TotalSubtests: valuePtr[int64](1220),
					SubtestPass:   valuePtr[int64](1220),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest2-bar-exp",
					},
				},
			},
		},
	}
	for _, metric := range sampleRunMetrics {
		err := client.UpsertWPTRunFeatureMetrics(
			ctx, metric.ExternalRunID, metric.Metrics)
		if err != nil {
			t.Errorf("unexpected error during insert of metrics. %s", err.Error())
		}
	}

	sampleSpecs := []struct {
		featureKey string
		spec       FeatureSpec
	}{
		{
			featureKey: "feature1",
			spec: FeatureSpec{
				Links: []string{
					"http://example1.com",
					"http://example2.com",
				},
			},
		},
		{
			featureKey: "feature3",
			spec: FeatureSpec{
				Links: []string{
					"http://example3.com",
					"http://example4.com",
				},
			},
		},
	}
	for _, spec := range sampleSpecs {
		err := client.UpsertFeatureSpec(
			ctx, spec.featureKey, spec.spec)
		if err != nil {
			t.Errorf("unexpected error during insert of spec. %s", err.Error())
		}
	}
	// Insert Group information
	groupKeyToInternalID := map[string]string{}
	groups := []Group{
		{
			GroupKey: "parent1",
			Name:     "Parent 1",
		},
		{
			GroupKey: "parent2",
			Name:     "Parent 2",
		},
		{
			GroupKey: "child3",
			Name:     "Child 3",
		},
	}
	for _, group := range groups {
		id, err := client.UpsertGroup(ctx, group)
		if err != nil {
			t.Fatalf("failed to insert group. err: %s group: %v\n", err, group)
		}
		groupKeyToInternalID[group.GroupKey] = *id
	}
	featureKeyToGroupsMapping := map[string][]string{
		"feature1": {"parent1"},
		"feature2": {"parent2"},
		"feature3": {"child3"},
	}
	childGroupKeyToParentGroupKey := map[string]string{
		"child3": "parent1",
	}
	err = client.UpsertFeatureGroupLookups(ctx, featureKeyToGroupsMapping, childGroupKeyToParentGroupKey)
	if err != nil {
		t.Fatalf("unable to insert group feature lookups err %s", err)
	}
	// Insert Snapshot information
	snapshotKeyToInternalID := map[string]string{}
	snapshots := []Snapshot{
		{
			SnapshotKey: "snapshot1",
			Name:        "Snapshot 1",
		},
		{
			SnapshotKey: "snapshot2",
			Name:        "Snapshot 2",
		},
	}
	for _, snapshot := range snapshots {
		id, err := client.UpsertSnapshot(ctx, snapshot)
		if err != nil {
			t.Fatalf("failed to insert snapshot. err: %s snapshot: %v\n", err, snapshot)
		}
		snapshotKeyToInternalID[snapshot.SnapshotKey] = *id
	}
	webFeatureSnapshots := []WebFeatureSnapshot{
		{
			WebFeatureID: webFeatureKeyToInternalFeatureID["feature1"],
			SnapshotIDs: []string{
				snapshotKeyToInternalID["snapshot1"],
			},
		},
		{
			WebFeatureID: webFeatureKeyToInternalFeatureID["feature2"],
			SnapshotIDs: []string{
				snapshotKeyToInternalID["snapshot2"],
			},
		},
	}
	for _, webFeatureSnapshot := range webFeatureSnapshots {
		err = client.UpsertWebFeatureSnapshot(ctx, webFeatureSnapshot)
		if err != nil {
			t.Fatalf("failed to insert web feature snapshot. err: %s", err)
		}
	}

	// Sync Developer Signals.
	err = client.SyncLatestFeatureDeveloperSignals(ctx, []FeatureDeveloperSignal{
		{
			WebFeatureKey: "feature1",
			Upvotes:       1,
			Link:          "https://example.com",
		},
		{
			WebFeatureKey: "feature2",
			Upvotes:       9,
			Link:          "https://example2.com",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error during sync. %s", err.Error())
	}

	// Insert Discouraged Features
	discouragedDetails := map[string]FeatureDiscouragedDetails{
		"feature3": {
			AccordingTo: []string{
				"https://example.com",
				"https://example2.com",
			},
			Alternatives: []string{
				"feature1",
			},
		},
		"feature4": {
			AccordingTo: []string{
				"https://example.com",
			},
			// Alternatives can be null
			Alternatives: nil,
		},
	}
	for featureKey, details := range discouragedDetails {
		err := client.UpsertFeatureDiscouragedDetails(ctx, featureKey, details)
		if err != nil {
			t.Fatalf("unexpected error during insert of discouraged details. %s", err.Error())
		}
	}
}

func addSampleChromiumUsageMetricsData(ctx context.Context,
	client *Client, t *testing.T, webFeatureKeyToInternalFeatureID map[string]string) {
	sampleChromiumHistogramEnums := []ChromiumHistogramEnum{
		{
			HistogramName: "AnotherHistogram",
		},
		{
			HistogramName: "WebDXFeatureObserver",
		},
	}
	chromiumHistogramEnumIDMap := insertTestChromiumHistogramEnums(ctx, client, t, sampleChromiumHistogramEnums)

	sampleChromiumHistogramEnumValues := []ChromiumHistogramEnumValue{
		{
			ChromiumHistogramEnumID: chromiumHistogramEnumIDMap["AnotherHistogram"],
			BucketID:                1,
			Label:                   "AnotherLabel",
		},
		{
			ChromiumHistogramEnumID: chromiumHistogramEnumIDMap["WebDXFeatureObserver"],
			BucketID:                1,
			Label:                   "feature1",
		},
		{
			ChromiumHistogramEnumID: chromiumHistogramEnumIDMap["WebDXFeatureObserver"],
			BucketID:                2,
			Label:                   "feature2",
		},
	}
	chromiumHistogramEnumValueToIDMap := make(map[string]string)
	insertTestChromiumHistogramEnumValues(
		ctx, client, t, sampleChromiumHistogramEnumValues)
	for _, enumValue := range sampleChromiumHistogramEnumValues {
		id, err := client.GetIDFromChromiumHistogramEnumValueKey(
			ctx, enumValue.ChromiumHistogramEnumID, enumValue.BucketID)
		if err != nil {
			t.Fatalf("unexpected error getting enum value id. %s", err.Error())
		}
		chromiumHistogramEnumValueToIDMap[enumValue.Label] = *id
	}

	sampleWebFeatureChromiumHistogramEnumValues := []WebFeatureChromiumHistogramEnumValue{
		{
			WebFeatureID:                 webFeatureKeyToInternalFeatureID["feature1"],
			ChromiumHistogramEnumValueID: chromiumHistogramEnumValueToIDMap["feature1"],
		},
		{
			WebFeatureID:                 webFeatureKeyToInternalFeatureID["feature2"],
			ChromiumHistogramEnumValueID: chromiumHistogramEnumValueToIDMap["feature2"],
		},
	}
	insertTestWebFeatureChromiumHistogramEnumValues(
		ctx, client, t, sampleWebFeatureChromiumHistogramEnumValues)

	sampleDailyChromiumHistogramMetrics := []dailyChromiumHistogramMetricToInsert{
		// feature1
		{
			histogramName: metricdatatypes.WebDXFeatureEnum,
			bucketID:      1,
			DailyChromiumHistogramMetric: DailyChromiumHistogramMetric{
				Day: civil.Date{
					Year:  2000,
					Month: time.January,
					Day:   1,
				},
				Rate: *big.NewRat(7, 100),
			},
		},
		{
			histogramName: metricdatatypes.WebDXFeatureEnum,
			bucketID:      1,
			DailyChromiumHistogramMetric: DailyChromiumHistogramMetric{
				Day: civil.Date{
					Year:  2000,
					Month: time.January,
					Day:   2,
				},
				Rate: *big.NewRat(8, 100),
			},
		},
		// feature2
		{
			histogramName: metricdatatypes.WebDXFeatureEnum,
			bucketID:      2,
			DailyChromiumHistogramMetric: DailyChromiumHistogramMetric{
				Day: civil.Date{
					Year:  2000,
					Month: time.January,
					Day:   1,
				},
				Rate: *big.NewRat(89, 100),
			},
		},
		{
			histogramName: metricdatatypes.WebDXFeatureEnum,
			bucketID:      2,
			DailyChromiumHistogramMetric: DailyChromiumHistogramMetric{
				Day: civil.Date{
					Year:  2000,
					Month: time.January,
					Day:   2,
				},
				Rate: *big.NewRat(90, 100),
			},
		},
		{
			histogramName: metricdatatypes.WebDXFeatureEnum,
			bucketID:      2,
			DailyChromiumHistogramMetric: DailyChromiumHistogramMetric{
				Day: civil.Date{
					Year:  2000,
					Month: time.January,
					Day:   15,
				},
				Rate: *big.NewRat(91, 100),
			},
		},
	}
	insertTestDailyChromiumHistogramMetrics(ctx, client, t, sampleDailyChromiumHistogramMetrics)

	err := client.SyncLatestDailyChromiumHistogramMetrics(ctx)
	if err != nil {
		t.Fatalf("unexpected error during sync. %s", err.Error())
	}
}

func defaultSorting() Sortable {
	return NewFeatureNameSort(true)
}

func defaultWPTMetricView() WPTMetricView {
	// TODO. For now, default to the view mode. Switch to the subtest later.
	return WPTTestView
}

func sortImplementationStatusesByBrowserName(statuses []*ImplementationStatus) {
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].BrowserName < statuses[j].BrowserName
	})
}

func sortMetricsByBrowserName(metrics []*FeatureResultMetric) {
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].BrowserName < metrics[j].BrowserName
	})
}

func stabilizeFeatureResultPage(page *FeatureResultPage) {
	stabilizeFeatureResults(page.Features)
}

func stabilizeFeatureResults(results []FeatureResult) {
	for _, result := range results {
		stabilizeFeatureResult(result)
	}
}

func stabilizeFeatureResult(result FeatureResult) {
	sortMetricsByBrowserName(result.StableMetrics)
	sortMetricsByBrowserName(result.ExperimentalMetrics)
	sortImplementationStatusesByBrowserName(result.ImplementationStatuses)

}

// FeatureSearchTestFeatureID represents a unique identifier for a feature
// within the following files:
//   - lib/gcpspanner/feature_search_test.go
//   - lib/gcpspanner/get_feature_test.go
type FeatureSearchTestFeatureID int

const (
	FeatureSearchTestFId1 FeatureSearchTestFeatureID = 1
	FeatureSearchTestFId2 FeatureSearchTestFeatureID = 2
	FeatureSearchTestFId3 FeatureSearchTestFeatureID = 3
	FeatureSearchTestFId4 FeatureSearchTestFeatureID = 4
)

func getFeatureSearchTestFeature(testFeatureID FeatureSearchTestFeatureID) FeatureResult {
	var ret FeatureResult
	switch testFeatureID {
	case FeatureSearchTestFId1:
		ret = FeatureResult{
			FeatureKey: "feature1",
			Name:       "Feature 1",
			Status:     valuePtr(string(BaselineStatusLow)),
			LowDate:    valuePtr[time.Time](time.Date(2000, time.January, 5, 0, 0, 0, 0, time.UTC)),
			HighDate:   nil,
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(33, 33),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest1-bar-stable",
					},
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(20, 20),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest1-foo-stable",
					},
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName:       "barBrowser",
					PassRate:          big.NewRat(220, 220),
					FeatureRunDetails: nil,
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(11, 11),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest1-foo-exp",
					},
				},
			},
			ImplementationStatuses: []*ImplementationStatus{
				{
					BrowserName:           "barBrowser",
					ImplementationStatus:  Available,
					ImplementationDate:    valuePtr(time.Date(2000, time.February, 2, 0, 0, 0, 0, time.UTC)),
					ImplementationVersion: valuePtr("1.0.0"),
				},
				{
					BrowserName:           "fooBrowser",
					ImplementationStatus:  Available,
					ImplementationDate:    valuePtr(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)),
					ImplementationVersion: valuePtr("0.0.0"),
				},
			},
			SpecLinks: []string{
				"http://example1.com",
				"http://example2.com",
			},
			ChromiumUsage:          big.NewRat(8, 100),
			DeveloperSignalUpvotes: valuePtr(int64(1)),
			DeveloperSignalLink:    valuePtr("https://example.com"),
			AccordingTo:            nil,
			Alternatives:           nil,
			VendorPositions: spanner.NullJSON{
				Value: []interface{}{
					map[string]interface{}{
						"position": "positive",
						"url":      "https://example.com/feature1",
						"vendor":   "chrome",
					},
				},
				Valid: true,
			},
		}
	case FeatureSearchTestFId2:
		ret = FeatureResult{
			FeatureKey: "feature2",
			Name:       "Feature 2",
			Status:     valuePtr(string(BaselineStatusHigh)),
			LowDate:    valuePtr[time.Time](time.Date(2000, time.January, 4, 0, 0, 0, 0, time.UTC)),
			HighDate:   valuePtr[time.Time](time.Date(2000, time.January, 31, 0, 0, 0, 0, time.UTC)),
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(10, 10),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest2-bar-stable",
					},
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(0, 10),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest2-foo-stable",
					},
				},
			},
			ExperimentalMetrics: []*FeatureResultMetric{
				{
					BrowserName: "barBrowser",
					PassRate:    big.NewRat(120, 120),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest2-bar-exp",
					},
				},
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(12, 12),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest2-foo-exp",
					},
				},
			},
			ImplementationStatuses: []*ImplementationStatus{
				{
					BrowserName:           "barBrowser",
					ImplementationStatus:  Available,
					ImplementationDate:    valuePtr(time.Date(2000, time.March, 2, 0, 0, 0, 0, time.UTC)),
					ImplementationVersion: valuePtr("2.0.0"),
				},
			},
			SpecLinks:              nil,
			ChromiumUsage:          big.NewRat(91, 100),
			DeveloperSignalUpvotes: valuePtr(int64(9)),
			DeveloperSignalLink:    valuePtr("https://example2.com"),
			AccordingTo:            nil,
			Alternatives:           nil,
			VendorPositions: spanner.NullJSON{
				Value: []interface{}{
					map[string]interface{}{
						"position": "negative",
						"url":      "https://example.com/feature2",
						"vendor":   "safari",
					},
				},
				Valid: true,
			},
		}
	case FeatureSearchTestFId3:
		ret = FeatureResult{
			FeatureKey: "feature3",
			Name:       "Feature 3",
			Status:     valuePtr(string(BaselineStatusNone)),
			LowDate:    nil,
			HighDate:   nil,
			StableMetrics: []*FeatureResultMetric{
				{
					BrowserName: "fooBrowser",
					PassRate:    big.NewRat(35, 50),
					FeatureRunDetails: map[string]interface{}{
						"test": "latest3-foo-stable",
					},
				},
			},
			ExperimentalMetrics: nil,
			ImplementationStatuses: []*ImplementationStatus{
				{
					BrowserName:           "fooBrowser",
					ImplementationStatus:  Available,
					ImplementationDate:    valuePtr(time.Date(2000, time.February, 1, 0, 0, 0, 0, time.UTC)),
					ImplementationVersion: valuePtr("1.0.0"),
				},
			},
			SpecLinks: []string{
				"http://example3.com",
				"http://example4.com",
			},
			ChromiumUsage:          nil,
			DeveloperSignalUpvotes: nil,
			DeveloperSignalLink:    nil,
			AccordingTo: []string{
				"https://example.com",
				"https://example2.com",
			},
			Alternatives: []string{
				"feature1",
			},
			VendorPositions: spanner.NullJSON{Value: nil, Valid: false},
		}
	case FeatureSearchTestFId4:
		ret = FeatureResult{
			FeatureKey:             "feature4",
			Name:                   "Feature 4",
			Status:                 nil,
			LowDate:                nil,
			HighDate:               nil,
			StableMetrics:          nil,
			ExperimentalMetrics:    nil,
			ImplementationStatuses: nil,
			SpecLinks:              nil,
			ChromiumUsage:          nil,
			DeveloperSignalUpvotes: nil,
			DeveloperSignalLink:    nil,
			AccordingTo: []string{
				"https://example.com",
			},
			Alternatives:    nil,
			VendorPositions: spanner.NullJSON{Value: nil, Valid: false},
		}
	}

	return ret
}

func testFeatureSearchAll(ctx context.Context, t *testing.T, client *Client) {
	// Simple test to get all the features without filters.
	expectedPage := FeatureResultPage{
		Features: []FeatureResult{
			getFeatureSearchTestFeature(FeatureSearchTestFId1),
			getFeatureSearchTestFeature(FeatureSearchTestFId2),
			getFeatureSearchTestFeature(FeatureSearchTestFId3),
			getFeatureSearchTestFeature(FeatureSearchTestFId4),
		},
		Total:         4,
		NextPageToken: nil,
	}
	// Test: Get all the results.
	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      nil,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)
}

func testFeatureSearchPagination(ctx context.Context, t *testing.T, client *Client) {
	type PaginationTestCase struct {
		name         string
		pageSize     int
		pageToken    *string // Optional
		expectedPage *FeatureResultPage
	}
	testCases := []PaginationTestCase{
		{
			name:      "page one",
			pageSize:  2,
			pageToken: nil, // First page does not need a page token.
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: valuePtr(encodeFeatureResultOffsetCursor(2)),
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
				},
			},
		},
		{
			name:     "page two",
			pageSize: 2,
			// The token should be made from the token of the previous page's last item
			pageToken: valuePtr(encodeFeatureResultOffsetCursor(2)),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: valuePtr(encodeFeatureResultOffsetCursor(4)),
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: tc.pageToken,
					pageSize:  tc.pageSize,
					node:      nil,
					sort:      defaultSorting(),
				},
				tc.expectedPage,
			)
		})
	}
}

func testFeatureSearchFilters(ctx context.Context, t *testing.T, client *Client) {
	testFeatureAvailableSearchFilters(ctx, t, client)
	testFeatureNotAvailableSearchFilters(ctx, t, client)
	testFeatureCommonFilterCombos(ctx, t, client)
	testFeatureNameFilters(ctx, t, client)
	testFeatureDescriptionFilters(ctx, t, client)
	testFeatureBaselineStatusFilters(ctx, t, client)
	testFeatureBaselineStatusDateFilters(ctx, t, client)
	testFeatureAvailableBrowserDateFilters(ctx, t, client)
	testIDFilters(ctx, t, client)
	testGroupFilters(ctx, t, client)
	testSnapshotFilters(ctx, t, client)
}

func testFeatureCommonFilterCombos(ctx context.Context, t *testing.T, client *Client) {
	type FilterComboTestCase struct {
		name         string
		searchNode   *searchtypes.SearchNode
		expectedPage *FeatureResultPage
	}
	testCases := []FilterComboTestCase{
		{
			name: "Available and not available filters",
			// available on barBrowser AND not available on fooBrowser
			searchNode: &searchtypes.SearchNode{
				Keyword: searchtypes.KeywordRoot,
				Term:    nil,
				Children: []*searchtypes.SearchNode{
					{
						Keyword: searchtypes.KeywordAND,
						Term:    nil,
						Children: []*searchtypes.SearchNode{
							{
								Children: nil,
								Term: &searchtypes.SearchTerm{
									Identifier: searchtypes.IdentifierAvailableOn,
									Value:      "barBrowser",
									Operator:   searchtypes.OperatorEq,
								},
								Keyword: searchtypes.KeywordNone,
							},
							{
								Children: nil,
								Term: &searchtypes.SearchTerm{
									Identifier: searchtypes.IdentifierAvailableOn,
									Value:      "fooBrowser",
									Operator:   searchtypes.OperatorNeq,
								},
								Keyword: searchtypes.KeywordNone,
							},
						},
					},
				},
			},
			expectedPage: &FeatureResultPage{
				Total:         1,
				NextPageToken: nil,
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: nil,
					pageSize:  100,
					node:      tc.searchNode,
					sort:      defaultSorting(),
				},
				tc.expectedPage,
			)
		})
	}
}

func testFeatureNotAvailableSearchFilters(ctx context.Context, t *testing.T, client *Client) {
	type NotAvailableFilterTestCase struct {
		name         string
		searchNode   *searchtypes.SearchNode
		expectedPage *FeatureResultPage
	}
	testCases := []NotAvailableFilterTestCase{
		{
			name: "single browser: not available on fooBrowser",
			searchNode: &searchtypes.SearchNode{
				Keyword: searchtypes.KeywordRoot,
				Term:    nil,
				Children: []*searchtypes.SearchNode{
					{
						Children: nil,
						Term: &searchtypes.SearchTerm{
							Identifier: searchtypes.IdentifierAvailableOn,
							Value:      "fooBrowser",
							Operator:   searchtypes.OperatorNeq,
						},
						Keyword: searchtypes.KeywordNone,
					},
				},
			},
			expectedPage: &FeatureResultPage{
				Total:         2,
				NextPageToken: nil,
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: nil,
					pageSize:  100,
					node:      tc.searchNode,
					sort:      defaultSorting(),
				},
				tc.expectedPage,
			)
		})
	}
}
func testFeatureAvailableSearchFilters(ctx context.Context, t *testing.T, client *Client) {
	type AvailableFilterTestCase struct {
		name         string
		searchNode   *searchtypes.SearchNode
		expectedPage *FeatureResultPage
	}
	testCases := []AvailableFilterTestCase{
		{
			name: "single browser: available on barBrowser",
			// available on barBrowser
			searchNode: &searchtypes.SearchNode{
				Keyword: searchtypes.KeywordRoot,
				Term:    nil,
				Children: []*searchtypes.SearchNode{
					{
						Children: nil,
						Term: &searchtypes.SearchTerm{
							Identifier: searchtypes.IdentifierAvailableOn,
							Value:      "barBrowser",
							Operator:   searchtypes.OperatorEq,
						},
						Keyword: searchtypes.KeywordNone,
					},
				},
			},
			expectedPage: &FeatureResultPage{
				Total:         2,
				NextPageToken: nil,
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
				},
			},
		},
		{
			name: "multiple browsers: available on either barBrowser OR fooBrowser",
			// available on either barBrowser OR fooBrowser
			searchNode: &searchtypes.SearchNode{
				Keyword: searchtypes.KeywordRoot,
				Term:    nil,
				Children: []*searchtypes.SearchNode{
					{
						Keyword: searchtypes.KeywordOR,
						Term:    nil,
						Children: []*searchtypes.SearchNode{
							{
								Children: nil,
								Term: &searchtypes.SearchTerm{
									Identifier: searchtypes.IdentifierAvailableOn,
									Value:      "barBrowser",
									Operator:   searchtypes.OperatorEq,
								},
								Keyword: searchtypes.KeywordNone,
							},
							{
								Children: nil,
								Term: &searchtypes.SearchTerm{
									Identifier: searchtypes.IdentifierAvailableOn,
									Value:      "fooBrowser",
									Operator:   searchtypes.OperatorEq,
								},
								Keyword: searchtypes.KeywordNone,
							},
						},
					},
				},
			},
			expectedPage: &FeatureResultPage{
				Total:         3,
				NextPageToken: nil,
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: nil,
					pageSize:  100,
					node:      tc.searchNode,
					sort:      defaultSorting(),
				},
				tc.expectedPage,
			)
		})
	}
}

func testFeatureNameFilters(ctx context.Context, t *testing.T, client *Client) {
	// All lower case with partial "feature" name. Should return all.
	expectedResults := []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId1),
		getFeatureSearchTestFeature(FeatureSearchTestFId2),
		getFeatureSearchTestFeature(FeatureSearchTestFId3),
		getFeatureSearchTestFeature(FeatureSearchTestFId4),
	}
	node := &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Keyword: searchtypes.KeywordNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierName,
					Value:      "feature",
					Operator:   searchtypes.OperatorLike,
				},
				Children: nil,
			},
		},
	}

	expectedPage := FeatureResultPage{
		Total:         4,
		NextPageToken: nil,
		Features:      expectedResults,
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)

	// All upper case with partial "FEATURE" name. Should return same results (all).
	node = &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Keyword: searchtypes.KeywordNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierName,
					Value:      "FEATURE",
					Operator:   searchtypes.OperatorLike,
				},
				Children: nil,
			},
		},
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)

	// Search for name with "4" Should return only feature 4.
	expectedResults = []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId4),
	}
	expectedPage = FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node = &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Keyword: searchtypes.KeywordNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierName,
					Value:      "4",
					Operator:   searchtypes.OperatorLike,
				},
				Children: nil,
			},
		},
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)
}

func testFeatureDescriptionFilters(ctx context.Context, t *testing.T, client *Client) {
	// All lower case with description. Should return feature 1.
	expectedResults := []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId1),
	}
	node := &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Keyword: searchtypes.KeywordNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierDescription,
					Value:      "@container",
					Operator:   searchtypes.OperatorLike,
				},
				Children: nil,
			},
		},
	}

	expectedPage := FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)

	// All upper case with partial description. Should return same results (all).
	node = &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Keyword: searchtypes.KeywordNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierDescription,
					Value:      "@CON",
					Operator:   searchtypes.OperatorLike,
				},
				Children: nil,
			},
		},
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)
}

func testSnapshotFilters(ctx context.Context, t *testing.T, client *Client) {
	// snapshot:snapshot1
	// Should get feature1
	expectedResults := []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId1),
	}
	expectedPage := FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node := &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Children: nil,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierSnapshot,
					Value:      "snapshot1",
					Operator:   searchtypes.OperatorEq,
				},
				Keyword: searchtypes.KeywordNone,
			},
		},
	}
	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)
	// snapshot:snapshot1
	// Should get feature2
	expectedResults = []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId2),
	}
	expectedPage = FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node = &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Children: nil,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierSnapshot,
					Value:      "snapshot2",
					Operator:   searchtypes.OperatorEq,
				},
				Keyword: searchtypes.KeywordNone,
			},
		},
	}
	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)
}

func testIDFilters(ctx context.Context, t *testing.T, client *Client) {
	expectedResults := []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId1),
	}
	node := &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Keyword: searchtypes.KeywordNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierID,
					Value:      "feature1",
					Operator:   searchtypes.OperatorEq,
				},
				Children: nil,
			},
		},
	}

	expectedPage := FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)

	// All upper case with "FEATURE1" name. Should return same result(s).
	node = &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Keyword: searchtypes.KeywordNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierID,
					Value:      "FEATURE1",
					Operator:   searchtypes.OperatorEq,
				},
				Children: nil,
			},
		},
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)

	// Return empty when search for a partial featurekey name.
	node = &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Keyword: searchtypes.KeywordNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierID,
					Value:      "FEATU",
					Operator:   searchtypes.OperatorEq,
				},
				Children: nil,
			},
		},
	}

	emptyResult := FeatureResultPage{
		Total:         0,
		NextPageToken: nil,
		Features:      nil,
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&emptyResult,
	)
}

func testGroupFilters(ctx context.Context, t *testing.T, client *Client) {
	// group:parent1
	// Should get feature1 (mapped directly to parent1) and feature3 (mapped to child3 which is a child of parent1)
	expectedResults := []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId1),
		getFeatureSearchTestFeature(FeatureSearchTestFId3),
	}
	expectedPage := FeatureResultPage{
		Total:         2,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node := &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Children: nil,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierGroup,
					Value:      "parent1",
					Operator:   searchtypes.OperatorEq,
				},
				Keyword: searchtypes.KeywordNone,
			},
		},
	}
	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)
	// group:parent2
	// Should get feature2 (mapped directly to parent2)
	expectedResults = []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId2),
	}
	expectedPage = FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node = &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Children: nil,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierGroup,
					Value:      "parent2",
					Operator:   searchtypes.OperatorEq,
				},
				Keyword: searchtypes.KeywordNone,
			},
		},
	}
	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)

	// group:child3
	// Should get feature3 (mapped directly to child3)
	expectedResults = []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId3),
	}
	expectedPage = FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node = &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Children: nil,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierGroup,
					Value:      "child3",
					Operator:   searchtypes.OperatorEq,
				},
				Keyword: searchtypes.KeywordNone,
			},
		},
	}
	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)

	// group:child3 OR group:parent2
	// Should get feature2 and feature3
	expectedResults = []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId2),
		getFeatureSearchTestFeature(FeatureSearchTestFId3),
	}
	expectedPage = FeatureResultPage{
		Total:         2,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node = &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Keyword: searchtypes.KeywordOR,
				Term:    nil,
				Children: []*searchtypes.SearchNode{
					{
						Children: nil,
						Term: &searchtypes.SearchTerm{
							Identifier: searchtypes.IdentifierGroup,
							Value:      "child3",
							Operator:   searchtypes.OperatorEq,
						},
						Keyword: searchtypes.KeywordNone,
					},
					{
						Children: nil,
						Term: &searchtypes.SearchTerm{
							Identifier: searchtypes.IdentifierGroup,
							Value:      "parent2",
							Operator:   searchtypes.OperatorEq,
						},
						Keyword: searchtypes.KeywordNone,
					},
				},
			},
		},
	}
	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)
}

func testFeatureAvailableBrowserDateFilters(ctx context.Context, t *testing.T, client *Client) {
	// available_date:barBrowser:2000-01-01..2000-02-02
	// Only Feature 1 is available on barBrowser during that same time window.
	expectedResults := []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId1),
	}
	expectedPage := FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node := &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Keyword: searchtypes.KeywordAND,
				Term:    nil,
				Children: []*searchtypes.SearchNode{
					{
						Keyword: searchtypes.KeywordNone,
						Term: &searchtypes.SearchTerm{
							Identifier: searchtypes.IdentifierAvailableBrowserDate,
							Operator:   searchtypes.OperatorNone,
							Value:      "",
						},
						Children: []*searchtypes.SearchNode{
							{
								Keyword: searchtypes.KeywordNone,
								Term: &searchtypes.SearchTerm{
									Identifier: searchtypes.IdentifierAvailableOn,
									Value:      "barBrowser",
									Operator:   searchtypes.OperatorEq,
								},
								Children: nil,
							},
							{
								Keyword: searchtypes.KeywordNone,
								Term: &searchtypes.SearchTerm{
									Identifier: searchtypes.IdentifierAvailableDate,
									Value:      "2000-01-01",
									Operator:   searchtypes.OperatorGtEq,
								},
								Children: nil,
							},
						},
					},
					{
						Keyword: searchtypes.KeywordNone,
						Term: &searchtypes.SearchTerm{
							Identifier: searchtypes.IdentifierAvailableBrowserDate,
							Operator:   searchtypes.OperatorNone,
							Value:      "",
						},
						Children: []*searchtypes.SearchNode{
							{
								Keyword: searchtypes.KeywordNone,
								Term: &searchtypes.SearchTerm{
									Identifier: searchtypes.IdentifierAvailableOn,
									Value:      "barBrowser",
									Operator:   searchtypes.OperatorEq,
								},
								Children: nil,
							},
							{
								Keyword: searchtypes.KeywordNone,
								Term: &searchtypes.SearchTerm{
									Identifier: searchtypes.IdentifierAvailableDate,
									Value:      "2000-02-02",
									Operator:   searchtypes.OperatorLtEq,
								},
								Children: nil,
							},
						},
					},
				},
			},
		},
	}
	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)
}

func testFeatureBaselineStatusDateFilters(ctx context.Context, t *testing.T, client *Client) {
	// Baseline Date 2000-01-04..2000-01-05
	expectedResults := []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId1),
		getFeatureSearchTestFeature(FeatureSearchTestFId2),
	}
	expectedPage := FeatureResultPage{
		Total:         2,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node := &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Keyword: searchtypes.KeywordAND,
				Term:    nil,
				Children: []*searchtypes.SearchNode{
					{
						Keyword: searchtypes.KeywordNone,
						Term: &searchtypes.SearchTerm{
							Identifier: searchtypes.IdentifierBaselineDate,
							Value:      "2000-01-04",
							Operator:   searchtypes.OperatorGtEq,
						},
						Children: nil,
					},
					{
						Keyword: searchtypes.KeywordNone,
						Term: &searchtypes.SearchTerm{
							Identifier: searchtypes.IdentifierBaselineDate,
							Value:      "2000-01-05",
							Operator:   searchtypes.OperatorLtEq,
						},
						Children: nil,
					},
				},
			},
		},
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)

	// Baseline Date 2000-01-01..2000-01-04
	expectedResults = []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId2),
	}
	expectedPage = FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node = &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Keyword: searchtypes.KeywordAND,
				Term:    nil,
				Children: []*searchtypes.SearchNode{
					{
						Keyword: searchtypes.KeywordNone,
						Term: &searchtypes.SearchTerm{
							Identifier: searchtypes.IdentifierBaselineDate,
							Value:      "2000-01-01",
							Operator:   searchtypes.OperatorGtEq,
						},
						Children: nil,
					},
					{
						Keyword: searchtypes.KeywordNone,
						Term: &searchtypes.SearchTerm{
							Identifier: searchtypes.IdentifierBaselineDate,
							Value:      "2000-01-04",
							Operator:   searchtypes.OperatorLtEq,
						},
						Children: nil,
					},
				},
			},
		},
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)
}

func testFeatureBaselineStatusFilters(ctx context.Context, t *testing.T, client *Client) {
	// Baseline status low only
	expectedResults := []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId1),
	}
	expectedPage := FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node := &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Keyword: searchtypes.KeywordNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierBaselineStatus,
					Value:      "newly",
					Operator:   searchtypes.OperatorEq,
				},
				Children: nil,
			},
		},
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)

	// baseline_status high only
	expectedResults = []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId2),
	}
	expectedPage = FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node = &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Keyword: searchtypes.KeywordNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierBaselineStatus,
					Value:      "widely",
					Operator:   searchtypes.OperatorEq,
				},
				Children: nil,
			},
		},
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)

	// Baseline none only, should exclude feature 4 which is nil.
	expectedResults = []FeatureResult{
		getFeatureSearchTestFeature(FeatureSearchTestFId3),
	}
	expectedPage = FeatureResultPage{
		Total:         1,
		NextPageToken: nil,
		Features:      expectedResults,
	}
	node = &searchtypes.SearchNode{
		Keyword: searchtypes.KeywordRoot,
		Term:    nil,
		Children: []*searchtypes.SearchNode{
			{
				Keyword: searchtypes.KeywordNone,
				Term: &searchtypes.SearchTerm{
					Identifier: searchtypes.IdentifierBaselineStatus,
					Value:      "limited",
					Operator:   searchtypes.OperatorEq,
				},
				Children: nil,
			},
		},
	}

	assertFeatureSearch(ctx, t, client,
		featureSearchArgs{
			pageToken: nil,
			pageSize:  100,
			node:      node,
			sort:      defaultSorting(),
		},
		&expectedPage,
	)
}

func testFeatureSearchSortAndPagination(ctx context.Context, t *testing.T, client *Client) {
	type SortAndPaginationTestCase struct {
		name         string
		sortable     Sortable
		pageToken    *string
		expectedPage *FeatureResultPage
	}
	testCases := []SortAndPaginationTestCase{
		{
			name:      "BaselineStatus asc - page 1",
			sortable:  NewBaselineStatusSort(true),
			pageToken: nil,
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: valuePtr(encodeFeatureResultOffsetCursor(2)),
				Features: []FeatureResult{
					// nil status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// none status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
				},
			},
		},
		{
			name:     "BaselineStatus asc - page 2",
			sortable: NewBaselineStatusSort(true),
			// Same page token as the next page token from the previous page.
			pageToken: valuePtr(encodeFeatureResultOffsetCursor(2)),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: valuePtr(encodeFeatureResultOffsetCursor(4)),
				Features: []FeatureResult{
					// high status low date 2000-01-04 high date 2000-01-31
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// low status low date 2000-01-05
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
				},
			},
		},
		{
			name:      "BaselineStatus desc - page 1",
			sortable:  NewBaselineStatusSort(false),
			pageToken: nil,
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: valuePtr(encodeFeatureResultOffsetCursor(2)),
				Features: []FeatureResult{
					// low status low date 2000-01-05
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// high status low date 2000-01-04 high date 2000-01-31
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
				},
			},
		},
		{
			name:     "BaselineStatus desc - page 2",
			sortable: NewBaselineStatusSort(false),
			// Same page token as the next page token from the previous page.
			pageToken: valuePtr(encodeFeatureResultOffsetCursor(2)),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: valuePtr(encodeFeatureResultOffsetCursor(4)),
				Features: []FeatureResult{
					// none status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// nil status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: tc.pageToken,
					pageSize:  2,
					node:      nil,
					sort:      tc.sortable,
				},
				tc.expectedPage,
			)
		})
	}
}

func testFeatureSearchComplexQueries(ctx context.Context, t *testing.T, client *Client) {
	testFeatureSearchSortAndPagination(ctx, t, client)
}

func testFeatureSearchSort(ctx context.Context, t *testing.T, client *Client) {
	testFeatureSearchSortName(ctx, t, client)
	testFeatureSearchSortBaselineStatus(ctx, t, client)
	testFeatureSearchSortBrowserImpl(ctx, t, client)
	testFeatureSearchChromiumUsage(ctx, t, client)
	testFeatureSearchSortBrowserFeatureSupport(ctx, t, client)
	testFeatureSearchSortDeveloperSignalUpvotes(ctx, t, client)
}

// nolint: dupl // WONTFIX. Only duplicated because the feature filter test yields similar results.
func testFeatureSearchSortName(ctx context.Context, t *testing.T, client *Client) {
	type NameSortTestCase struct {
		name         string
		sortable     Sortable
		expectedPage *FeatureResultPage
	}
	testCases := []NameSortTestCase{
		{
			name:     "Name asc",
			sortable: NewFeatureNameSort(true),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
				},
			},
		},
		{
			name:     "Name desc",
			sortable: NewFeatureNameSort(false),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: nil,
					pageSize:  100,
					node:      nil,
					sort:      tc.sortable,
				},
				tc.expectedPage,
			)
		})
	}
}

// nolint: dupl // Okay to duplicate for tests
func testFeatureSearchSortBaselineStatus(ctx context.Context, t *testing.T, client *Client) {
	type BaselineStatusSortCase struct {
		name         string
		sortable     Sortable
		expectedPage *FeatureResultPage
	}
	testCases := []BaselineStatusSortCase{
		{
			name:     "BaselineStatus asc",
			sortable: NewBaselineStatusSort(true),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// nil status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// none status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// high status low date 2000-01-04 high date 2000-01-31
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// low status low date 2000-01-05
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
				},
			},
		},
		{
			name:     "BaselineStatus desc",
			sortable: NewBaselineStatusSort(false),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// low status low date 2000-01-05
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// high status low date 2000-01-04 high date 2000-01-31
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// none status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// nil status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: nil,
					pageSize:  100,
					node:      nil,
					sort:      tc.sortable,
				},
				tc.expectedPage,
			)
		})
	}
}

func testFeatureSearchSortBrowserImpl(ctx context.Context, t *testing.T, client *Client) {
	type BaselineStatusSortCase struct {
		name         string
		sortable     Sortable
		expectedPage *FeatureResultPage
	}
	testCases := []BaselineStatusSortCase{
		{
			name:     "BrowserImpl fooBrowser Stable asc",
			sortable: NewBrowserImplSort(true, "fooBrowser", true),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// null metric, null status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// 0 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// 0.7 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// 1.0 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
				},
			},
		},
		{
			name:     "BrowserImpl fooBrowser Stable desc",
			sortable: NewBrowserImplSort(false, "fooBrowser", true),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// 1.0 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// 0.7 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// 0 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// null metric, null status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
				},
			},
		},
		{
			name:     "BrowserImpl fooBrowser Experimental asc",
			sortable: NewBrowserImplSort(true, "fooBrowser", false),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// null metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// null metric, null status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// 1.0 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// 1.0 metric, null status
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
				},
			},
		},
		{
			name:     "BrowserImpl fooBrowser Experimental desc",
			sortable: NewBrowserImplSort(false, "fooBrowser", false),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// 1.0 metric, null status
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// 1.0 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// null metric, null status
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// null metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: nil,
					pageSize:  100,
					node:      nil,
					sort:      tc.sortable,
				},
				tc.expectedPage,
			)
		})
	}
}

func testFeatureSearchChromiumUsage(ctx context.Context, t *testing.T, client *Client) {
	type BaselineStatusSortCase struct {
		name         string
		sortable     Sortable
		expectedPage *FeatureResultPage
	}
	testCases := []BaselineStatusSortCase{
		{
			name:     "ChromiumUsage asc",
			sortable: NewChromiumUsageSort(true),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// null metric
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// null metric
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// 0.7 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// 1.0 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
				},
			},
		},
		{
			name:     "ChromiumUsage desc",
			sortable: NewChromiumUsageSort(false),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// 1.0 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// 0.7 metric, available status
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// null metric
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// null metric
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
				},
			},
		},
		{
			name:     "BrowserImpl fooBrowser Experimental asc",
			sortable: NewBrowserImplSort(true, "fooBrowser", false),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// null metric
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// null metric
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// 0.08 metric
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// 0.91 metric
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
				},
			},
		},
		{
			name:     "BrowserImpl fooBrowser Experimental desc",
			sortable: NewBrowserImplSort(false, "fooBrowser", false),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// 0.91 metric
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// 0.08 metric
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// null metric
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// null metric
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: nil,
					pageSize:  100,
					node:      nil,
					sort:      tc.sortable,
				},
				tc.expectedPage,
			)
		})
	}
}

func testFeatureSearchSortBrowserFeatureSupport(ctx context.Context, t *testing.T, client *Client) {
	type BrowserFeatureSupportCase struct {
		name         string
		sortable     Sortable
		expectedPage *FeatureResultPage
	}
	testCases := []BrowserFeatureSupportCase{
		{
			name:     "BrowserFeatureSupport fooBrowser asc",
			sortable: NewBrowserFeatureSupportSort(true, "fooBrowser"),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// null feature support
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// null feature support
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// Jan 1, 2000
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// Feb 1, 2000
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
				},
			},
		},
		{
			name:     "BrowserFeatureSupport fooBrowser desc",
			sortable: NewBrowserFeatureSupportSort(false, "fooBrowser"),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// Feb 1, 2000
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// Jan 1, 2000
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// null feature support
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// null feature support
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: nil,
					pageSize:  100,
					node:      nil,
					sort:      tc.sortable,
				},
				tc.expectedPage,
			)
		})
	}
}

func testFeatureSearchSortDeveloperSignalUpvotes(ctx context.Context, t *testing.T, client *Client) {
	type DeveloperSignalUpvoteCase struct {
		name         string
		sortable     Sortable
		expectedPage *FeatureResultPage
	}
	testCases := []DeveloperSignalUpvoteCase{
		{
			name:     "DeveloperSignalUpvotes asc",
			sortable: NewDeveloperSignalUpvotesSort(true),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// null developer signal, defaults to `feature key ASC`, feature3
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// null developer signal, defaults to `feature key ASC`, feature4
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
					// upvotes = 1, feature 1
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// upvotes = 9, feature 2
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
				},
			},
		},
		{
			name:     "DeveloperSignalUpvotes desc",
			sortable: NewDeveloperSignalUpvotesSort(false),
			expectedPage: &FeatureResultPage{
				Total:         4,
				NextPageToken: nil,
				Features: []FeatureResult{
					// upvotes = 9, feature 2
					getFeatureSearchTestFeature(FeatureSearchTestFId2),
					// upvotes = 1, feature 1
					getFeatureSearchTestFeature(FeatureSearchTestFId1),
					// null developer signal, defaults to `feature key ASC`, feature3
					getFeatureSearchTestFeature(FeatureSearchTestFId3),
					// null developer signal, defaults to `feature key ASC`, feature4
					getFeatureSearchTestFeature(FeatureSearchTestFId4),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertFeatureSearch(ctx, t, client,
				featureSearchArgs{
					pageToken: nil,
					pageSize:  100,
					node:      nil,
					sort:      tc.sortable,
				},
				tc.expectedPage,
			)
		})
	}
}

func TestFeaturesSearch(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	setupRequiredTablesForFeaturesSearch(ctx, spannerClient, t)

	// Try with default GCPSpannerBaseQuery
	t.Run("gcp spanner queries", func(t *testing.T) {
		testFeatureSearchAll(ctx, t, spannerClient)
		testFeatureSearchPagination(ctx, t, spannerClient)
		testFeatureSearchFilters(ctx, t, spannerClient)
		testFeatureSearchSort(ctx, t, spannerClient)
		testFeatureSearchComplexQueries(ctx, t, spannerClient)
	})

	// Try with LocalFeatureBaseQuery
	t.Run("local spanner queries", func(t *testing.T) {
		spannerClient.SetFeatureSearchBaseQuery(LocalFeatureBaseQuery{})
		testFeatureSearchAll(ctx, t, spannerClient)
		testFeatureSearchPagination(ctx, t, spannerClient)
		testFeatureSearchFilters(ctx, t, spannerClient)
		testFeatureSearchSort(ctx, t, spannerClient)
		testFeatureSearchComplexQueries(ctx, t, spannerClient)
	})
}

type featureSearchArgs struct {
	pageToken *string
	pageSize  int
	node      *searchtypes.SearchNode
	sort      Sortable
}

func assertFeatureSearch(
	ctx context.Context,
	t *testing.T,
	client *Client,
	args featureSearchArgs,
	expectedPage *FeatureResultPage) {
	page, err := client.FeaturesSearch(
		ctx,
		args.pageToken,
		args.pageSize,
		args.node,
		args.sort,
		// TODO. When the tests assert both views, remove this and allow the test
		// to pass this.
		defaultWPTMetricView(),
		getDefaultTestBrowserList(),
	)
	if err != nil {
		t.Errorf("unexpected error during search of features %s", err.Error())
	}
	stabilizeFeatureResultPage(page)
	if !AreFeatureResultPagesEqual(expectedPage, page) {
		t.Errorf("unequal results.\nexpected (%+v)\nreceived (%+v) ",
			PrettyPrintFeatureResultPage(expectedPage),
			PrettyPrintFeatureResultPage(page))
	}
}

func AreFeatureResultPagesEqual(a, b *FeatureResultPage) bool {
	return a.Total == b.Total &&
		((a.NextPageToken == nil && b.NextPageToken == nil) ||
			((a.NextPageToken != nil && b.NextPageToken != nil) && *a.NextPageToken == *b.NextPageToken)) &&
		AreFeatureResultsSlicesEqual(a.Features, b.Features)
}

func AreFeatureResultsSlicesEqual(a, b []FeatureResult) bool {
	return slices.EqualFunc[[]FeatureResult](a, b, AreFeatureResultsEqual)
}

func AreFeatureResultsEqual(a, b FeatureResult) bool {
	return a.FeatureKey == b.FeatureKey &&
		a.Name == b.Name &&
		reflect.DeepEqual(a.Status, b.Status) &&
		reflect.DeepEqual(a.LowDate, b.LowDate) &&
		reflect.DeepEqual(a.HighDate, b.HighDate) &&
		AreMetricsEqual(a.StableMetrics, b.StableMetrics) &&
		AreMetricsEqual(a.ExperimentalMetrics, b.ExperimentalMetrics) &&
		AreImplementationStatusesEqual(a.ImplementationStatuses, b.ImplementationStatuses) &&
		AreSpecLinksEqual(a.SpecLinks, b.SpecLinks) &&
		AreChromiumUsagesEqual(a.ChromiumUsage, b.ChromiumUsage) &&
		AreDeveloperSignalUpvotesEqual(a.DeveloperSignalUpvotes, b.DeveloperSignalUpvotes) &&
		AreDeveloperSignalLinksEqual(a.DeveloperSignalLink, b.DeveloperSignalLink) &&
		AreDiscouragedAccordingToEqual(a.AccordingTo, b.AccordingTo) &&
		AreDiscouragedAlternativesEqual(a.Alternatives, b.Alternatives) &&
		AreVendorPositionsEqual(a.VendorPositions, b.VendorPositions)
}

func AreVendorPositionsEqual(a, b spanner.NullJSON) bool {
	return reflect.DeepEqual(a, b)
}

func AreDeveloperSignalUpvotesEqual(a, b *int64) bool { return reflect.DeepEqual(a, b) }

func AreDeveloperSignalLinksEqual(a, b *string) bool { return reflect.DeepEqual(a, b) }

func AreDiscouragedAccordingToEqual(a, b []string) bool { return slices.Equal(a, b) }

func AreDiscouragedAlternativesEqual(a, b []string) bool { return slices.Equal(a, b) }

func AreSpecLinksEqual(a, b []string) bool {
	return slices.Equal(a, b)
}

func AreChromiumUsagesEqual(a, b *big.Rat) bool {
	if (a == nil && b != nil) || (a != nil && b == nil) {
		return false
	}

	return (a == nil && b == nil) || (a.Cmp(b) == 0)
}

func AreImplementationStatusesEqual(a, b []*ImplementationStatus) bool {
	return slices.EqualFunc[[]*ImplementationStatus](a, b, func(a, b *ImplementationStatus) bool {
		return a.BrowserName == b.BrowserName &&
			(a.ImplementationStatus == b.ImplementationStatus) &&
			((a.ImplementationDate == nil &&
				b.ImplementationDate == nil) ||
				(a.ImplementationDate != nil &&
					b.ImplementationDate != nil &&
					(*a.ImplementationDate).Equal(*b.ImplementationDate))) &&
			((a.ImplementationVersion == nil &&
				b.ImplementationVersion == nil) ||
				(a.ImplementationVersion != nil &&
					b.ImplementationVersion != nil &&
					(*a.ImplementationVersion) == (*b.ImplementationVersion)))
	})
}

func AreMetricsEqual(a, b []*FeatureResultMetric) bool {
	return slices.EqualFunc[[]*FeatureResultMetric](a, b, func(a, b *FeatureResultMetric) bool {
		if (a.PassRate == nil && b.PassRate != nil) || (a.PassRate != nil && b.PassRate == nil) {
			return false
		}

		return a.BrowserName == b.BrowserName &&
			((a.PassRate == nil && b.PassRate == nil) || (a.PassRate.Cmp(b.PassRate) == 0)) &&
			reflect.DeepEqual(a.FeatureRunDetails, b.FeatureRunDetails)
	})
}

func PrintNullableField[T any](in *T) string {
	if in == nil {
		return "NIL"
	}

	return fmt.Sprintf("%v", *in)
}

func PrettyPrintFeatureResult(result FeatureResult) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "\tFeatureID: %s\n", result.FeatureKey)
	fmt.Fprintf(&builder, "\tName: %s\n", result.Name)

	fmt.Fprintf(&builder, "\tStatus: %s\n", PrintNullableField(result.Status))
	fmt.Fprintf(&builder, "\tLowDate: %s\n", PrintNullableField(result.LowDate))
	fmt.Fprintf(&builder, "\tHighDate: %s\n", PrintNullableField(result.HighDate))
	fmt.Fprintf(&builder, "\tSpecLinks: %s\n", result.SpecLinks)
	fmt.Fprintf(&builder, "\tChromiumUsage: %s\n", PrintNullableField(result.ChromiumUsage))

	fmt.Fprintln(&builder, "\tStable Metrics:")
	for _, metric := range result.StableMetrics {
		fmt.Fprint(&builder, PrettyPrintMetric(metric))
	}

	fmt.Fprintln(&builder, "\tExperimental Metrics:")
	for _, metric := range result.ExperimentalMetrics {
		fmt.Fprint(&builder, PrettyPrintMetric(metric))
	}
	fmt.Fprintln(&builder, "\tImplementation Statuses:")
	for _, status := range result.ImplementationStatuses {
		fmt.Fprint(&builder, PrettyPrintImplementationStatus(status))
	}
	fmt.Fprintf(&builder, "\tDeveloperSignalUpvotes: %s\n", PrintNullableField(result.DeveloperSignalUpvotes))
	fmt.Fprintf(&builder, "\tDeveloperSignalLink %s\n", PrintNullableField(result.DeveloperSignalLink))

	fmt.Fprintf(&builder, "\tAccordingTo: %s\n", result.AccordingTo)
	fmt.Fprintf(&builder, "\tAlternatives: %s\n", result.Alternatives)
	fmt.Fprintf(&builder, "\tVendorPositions: %s\n", result.VendorPositions)
	fmt.Fprintln(&builder)

	return builder.String()
}

func PrettyPrintImplementationStatus(status *ImplementationStatus) string {
	var builder strings.Builder
	if status == nil {
		return "\t\tNIL STATUS\n"
	}
	fmt.Fprintf(&builder, "\t\tBrowserName: %s\n", status.BrowserName)
	fmt.Fprintf(&builder, "\t\tStatus: %s\n", status.ImplementationStatus)
	fmt.Fprintf(&builder, "\t\tDate: %s\n", PrintNullableField(status.ImplementationDate))
	fmt.Fprintf(&builder, "\t\tVersion: %s\n", PrintNullableField(status.ImplementationVersion))

	return builder.String()
}

func PrettyPrintMetric(metric *FeatureResultMetric) string {
	var builder strings.Builder
	if metric == nil {
		return "\t\tNIL\n"
	}
	fmt.Fprintf(&builder, "\t\tBrowserName: %s\n", metric.BrowserName)
	fmt.Fprintf(&builder, "\t\tFeatureRunDetails: %v\n", metric.FeatureRunDetails)
	fmt.Fprintf(&builder, "\t\tPassRate: %s\n", PrettyPrintPassRate(metric.PassRate))

	return builder.String()
}

func PrettyPrintPassRate(passRate *big.Rat) string {
	if passRate == nil {
		return "\t\tNIL\n"
	}

	return passRate.String() + "\n"
}

func PrettyPrintPageToken(token *string) string {
	if token == nil {
		return "NIL\n"
	}

	return *token + "\n"
}

func PrettyPrintFeatureResultPage(page *FeatureResultPage) string {
	if page == nil {
		return ""
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "Total: %d\n", page.Total)
	fmt.Fprintf(&builder, "NextPageToken: %s\n", PrettyPrintPageToken(page.NextPageToken))
	fmt.Fprint(&builder, PrettyPrintFeatureResults(page.Features))

	return builder.String()
}

// PrettyPrintFeatureResults returns a formatted string representation of a slice of FeatureResult structs.
func PrettyPrintFeatureResults(results []FeatureResult) string {
	var builder strings.Builder
	for _, result := range results {
		fmt.Fprint(&builder, PrettyPrintFeatureResult(result))
	}

	return builder.String()
}

// Test helper to populate ExcludedFeatureKeys table.
func (c *Client) InsertExcludedFeatureKey(ctx context.Context, featureKey string) error {
	_, err := c.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.InsertOrUpdateMap(
			"ExcludedFeatureKeys",
			map[string]interface{}{
				"FeatureKey": featureKey,
			})

		return txn.BufferWrite([]*spanner.Mutation{m})
	})

	return err
}

func (c *Client) ClearExcludedFeatureKeys(ctx context.Context) error {
	_, err := c.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		mutation := spanner.Delete("ExcludedFeatureKeys", spanner.AllKeys())

		return txn.BufferWrite([]*spanner.Mutation{mutation})
	})

	return err
}

func (c *Client) ClearFeatureDiscouragedDetails(ctx context.Context) error {
	_, err := c.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		mutation := spanner.Delete(featureDiscouragedDetailsTable, spanner.AllKeys())

		return txn.BufferWrite([]*spanner.Mutation{mutation})
	})

	return err
}
