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
	"math/big"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

type webFeatureForeignKeyTestHelpers struct {
	featureID *string
}

func (w webFeatureForeignKeyTestHelpers) testWebFeature() WebFeature {
	return WebFeature{
		Name:            "Foreign Key Test",
		FeatureKey:      "fk-test",
		Description:     "fk description",
		DescriptionHTML: "Feature <b>FK</b> description",
	}
}

func (w *webFeatureForeignKeyTestHelpers) insertEntities(ctx context.Context, t *testing.T) {
	w.insertFeature(ctx, t)
	w.insertWPTTest(ctx, t)
	w.insertBaselineStatus(ctx, t)
	w.insertBrowserFeatureAvailability(ctx, t)
	w.insertFeatureSpec(ctx, t)
	w.insertWebFeatureGroupLookup(ctx, t)
	w.insertWebFeatureSnapshot(ctx, t)
	w.insertWebFeatureChromiumHistogramData(ctx, t)
	w.insertBrowserFeatureSupportEvent(ctx, t)
	w.insertFeatureDiscouragedDetails(ctx, t)
}

func (w *webFeatureForeignKeyTestHelpers) insertFeature(ctx context.Context, t *testing.T) {
	id, err := spannerClient.UpsertWebFeature(ctx, w.testWebFeature())
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}
	w.featureID = id
}

func (w webFeatureForeignKeyTestHelpers) insertWPTTest(ctx context.Context, t *testing.T) {
	testRunID := int64(10)
	err := spannerClient.InsertWPTRun(ctx, WPTRun{
		RunID:            testRunID,
		TimeStart:        time.Date(2020, time.January, 2, 0, 0, 0, 0, time.UTC),
		TimeEnd:          time.Date(2020, time.January, 2, 1, 0, 0, 0, time.UTC),
		BrowserName:      "fooBrowser",
		BrowserVersion:   "0.0.0",
		Channel:          shared.StableLabel,
		OSName:           "os",
		OSVersion:        "0.0.0",
		FullRevisionHash: "abcdef0123456789",
	})
	if err != nil {
		t.Errorf("unable to insert wpt test")
	}

	err = spannerClient.UpsertWPTRunFeatureMetrics(ctx, testRunID, map[string]WPTRunFeatureMetric{
		w.testWebFeature().FeatureKey: {
			TotalTests: valuePtr[int64](20),
			TestPass:   valuePtr[int64](10),
			// TODO: Put value when asserting subtest metrics and feature run details
			TotalSubtests:     nil,
			SubtestPass:       nil,
			FeatureRunDetails: nil,
		},
	})

	if err != nil {
		t.Errorf("unable to insert wpt test metrics")
	}
}

func (w *webFeatureForeignKeyTestHelpers) insertBaselineStatus(ctx context.Context, t *testing.T) {
	err := spannerClient.UpsertFeatureBaselineStatus(ctx, w.testWebFeature().FeatureKey, FeatureBaselineStatus{
		Status:   valuePtr(BaselineStatusHigh),
		HighDate: nil,
		LowDate:  nil,
	})
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}
}

func (w *webFeatureForeignKeyTestHelpers) insertBrowserFeatureAvailability(ctx context.Context, t *testing.T) {
	// Insert BrowserRelease first
	err := spannerClient.InsertBrowserRelease(ctx, BrowserRelease{
		BrowserName:    "fooBrowser",
		BrowserVersion: "0.0.0",
		ReleaseDate:    time.Date(2020, time.January, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}

	// Insert BrowserFeatureAvailability
	err = spannerClient.UpsertBrowserFeatureAvailability(ctx, w.testWebFeature().FeatureKey, BrowserFeatureAvailability{
		BrowserName:    "fooBrowser",
		BrowserVersion: "0.0.0",
	})
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}
}

func (w *webFeatureForeignKeyTestHelpers) insertFeatureSpec(ctx context.Context, t *testing.T) {
	err := spannerClient.UpsertFeatureSpec(ctx, w.testWebFeature().FeatureKey, FeatureSpec{
		Links: nil,
	})
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}
}

func (w *webFeatureForeignKeyTestHelpers) insertWebFeatureGroupLookup(ctx context.Context, t *testing.T) {
	// Insert the group at first
	_, err := spannerClient.UpsertGroup(ctx, Group{
		GroupKey: "parent1",
		Name:     "Parent 1",
	})
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}

	err = spannerClient.UpsertFeatureGroupLookups(
		ctx,
		map[string][]string{
			w.testWebFeature().FeatureKey: {"parent1"},
		},
		nil,
	)

	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}
}

func (w *webFeatureForeignKeyTestHelpers) insertWebFeatureSnapshot(ctx context.Context, t *testing.T) {
	err := spannerClient.UpsertWebFeatureSnapshot(ctx, WebFeatureSnapshot{
		WebFeatureID: *w.featureID,
		SnapshotIDs:  nil,
	})
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}
}

func (w *webFeatureForeignKeyTestHelpers) insertWebFeatureChromiumHistogramData(
	ctx context.Context, t *testing.T) {
	// Insert ChromiumHistogramEnum, ChromiumHistogramEnumValue, WebFeatureChromiumHistogramEnumValue,
	// and DailyChromiumHistogramMetrics first
	histogramName := "test"
	id, err := spannerClient.UpsertChromiumHistogramEnum(ctx, ChromiumHistogramEnum{
		HistogramName: histogramName,
	})
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}

	bucketID := int64(1)

	enumID, err := spannerClient.UpsertChromiumHistogramEnumValue(ctx, ChromiumHistogramEnumValue{
		ChromiumHistogramEnumID: *id,
		BucketID:                bucketID,
		Label:                   "test label",
	})
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}

	err = spannerClient.UpsertWebFeatureChromiumHistogramEnumValue(ctx, WebFeatureChromiumHistogramEnumValue{
		WebFeatureID:                 *w.featureID,
		ChromiumHistogramEnumValueID: *enumID,
	})
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}

	err = spannerClient.UpsertDailyChromiumHistogramMetric(ctx,
		metricdatatypes.HistogramName(histogramName), bucketID, DailyChromiumHistogramMetric{
			Day: civil.Date{
				Year:  2000,
				Day:   1,
				Month: time.January,
			},
			Rate: *big.NewRat(91, 100),
		})
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}

	// Insert WebFeatureChromiumHistogramEnumValue
	err = spannerClient.UpsertWebFeatureChromiumHistogramEnumValue(ctx, WebFeatureChromiumHistogramEnumValue{
		WebFeatureID:                 *w.featureID,
		ChromiumHistogramEnumValueID: *enumID,
	})
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}
}

func (w *webFeatureForeignKeyTestHelpers) insertBrowserFeatureSupportEvent(ctx context.Context, t *testing.T) {
	// Insert the BrowserRelease first
	err := spannerClient.InsertBrowserRelease(ctx, BrowserRelease{
		BrowserName:    "barBrowser",
		BrowserVersion: "0.0.0",
		ReleaseDate:    time.Date(2022, time.January, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}

	// Insert BrowserFeatureSupportEvent
	err = spannerClient.PrecalculateBrowserFeatureSupportEvents(ctx,
		time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, time.January, 3, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}
}

func (w *webFeatureForeignKeyTestHelpers) insertFeatureDiscouragedDetails(ctx context.Context, t *testing.T) {
	err := spannerClient.UpsertFeatureDiscouragedDetails(ctx, w.testWebFeature().FeatureKey, FeatureDiscouragedDetails{
		AccordingTo:  []string{"test1"},
		Alternatives: []string{"foobar"},
	})
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}
}

func (w *webFeatureForeignKeyTestHelpers) assertWebFeatureCount(ctx context.Context, t *testing.T, want int) {
	features, err := spannerClient.ReadAllWebFeatures(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	if len(features) != want {
		t.Errorf("unexpected number of features. want: %d, got %d", want, len(features))
	}
}

func (w *webFeatureForeignKeyTestHelpers) assertWPTMetricCount(ctx context.Context, t *testing.T, want int) {
	metrics, err := spannerClient.ReadAllWPTRunFeatureMetrics(ctx)
	if err != nil {
		t.Errorf("unexpected error during read all of metrics. %s", err.Error())
	}
	if len(metrics) != want {
		t.Errorf("unexpected number of metrics. want: %d, got %d", want, len(metrics))
	}
}

func (w *webFeatureForeignKeyTestHelpers) assertBaselineStatusCount(ctx context.Context, t *testing.T, want int) {
	statuses, err := spannerClient.ReadAllBaselineStatuses(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all of baseline statuses. %s", err.Error())
	}
	if len(statuses) != want {
		t.Errorf("unexpected number of baseline statuses. want: %d, got %d", want, len(statuses))
	}
}

func (w webFeatureForeignKeyTestHelpers) assertBrowserFeatureAvailabilityCount(
	ctx context.Context, t *testing.T, want int) {
	availabilities, err := spannerClient.ReadAllAvailabilities(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all of availabilities. %s", err.Error())
	}

	if len(availabilities) != want {
		t.Errorf("unexpected number of availabilities. want: %d, got %d", want, len(availabilities))
	}
}

func (w webFeatureForeignKeyTestHelpers) assertFeatureSpecCount(ctx context.Context, t *testing.T, want int) {
	specs, err := spannerClient.ReadAllFeatureSpecs(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all of specs. %s", err.Error())
	}
	if len(specs) != want {
		t.Errorf("unexpected number of specs. want: %d, got %d", want, len(specs))
	}
}

func (w webFeatureForeignKeyTestHelpers) assertFeatureGroupLookupCount(ctx context.Context, t *testing.T, want int) {
	groups := spannerClient.readAllFeatureGroupKeysLookups(ctx, t)
	if len(groups) != want {
		t.Errorf("unexpected number of groups. want: %d, got %d", want, len(groups))

	}
}

func (w webFeatureForeignKeyTestHelpers) assertWebFeatureSnapshotCount(ctx context.Context, t *testing.T, want int) {
	snapshots, err := spannerClient.ReadAllWebFeatureSnapshots(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all of snapshots. %s", err.Error())
	}
	if len(snapshots) != want {
		t.Errorf("unexpected number of snapshots. want: %d, got %d", want, len(snapshots))
	}
}

func (w webFeatureForeignKeyTestHelpers) assertWebFeatureChromiumHistogramEnumValueCount(
	ctx context.Context, t *testing.T, want int) {
	values, err := spannerClient.readAllWebFeatureChromiumHistogramEnums(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all of enum values. %s", err.Error())
	}

	if len(values) != want {
		t.Errorf("unexpected number of enum values. want: %d, got %d", want, len(values))
	}
}

func (w webFeatureForeignKeyTestHelpers) assertLatestDailyChromiumHistogramMetricsCount(
	ctx context.Context, t *testing.T, want int) {
	metrics, err := spannerClient.readAllLatestDailyChromiumHistogramMetrics(ctx)
	if err != nil {
		t.Errorf("unexpected error during read all of latest daily histogram metrics. %s", err.Error())
	}

	if len(metrics) != want {
		t.Errorf("unexpected number of latest daily histogram metrics. want: %d, got %d", want, len(metrics))
	}
}

func (w webFeatureForeignKeyTestHelpers) assertBrowserFeatureSupportEventCount(
	ctx context.Context, t *testing.T, want int) {
	events := spannerClient.readAllBrowserFeatureSupportEvents(ctx, t)
	if len(events) != want {
		t.Errorf("unexpected number of events. want: %d, got %d", want, len(events))
	}
}

func (w webFeatureForeignKeyTestHelpers) assertFeatureDiscouragedDetailsCount(
	ctx context.Context, t *testing.T, want int) {
	details, err := spannerClient.readAllFeatureDiscouragedDetails(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all of details. %s", err.Error())
	}
	if len(details) != want {
		t.Errorf("unexpected number of details. want: %d, got %d", want, len(details))
	}
}

// This is to test https://github.com/GoogleChrome/webstatus.dev/issues/513
// and to prevent it in the future.
// An easy way to find all of these would be to examine all the migrations with the
// following text:
// `REFERENCES WebFeatures(ID)` (lacks the `ON DELETE CASCADE`)
//
// Also, these constraints should also be named going forward to prevent spanner
// from auto generating a name.
func TestWebFeatureForeignKey(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	helpers := webFeatureForeignKeyTestHelpers{featureID: nil}
	helpers.insertEntities(ctx, t)

	// There should be 1 of each entity
	helpers.assertWebFeatureCount(ctx, t, 1)
	helpers.assertWPTMetricCount(ctx, t, 1)
	helpers.assertBaselineStatusCount(ctx, t, 1)
	helpers.assertBrowserFeatureAvailabilityCount(ctx, t, 1)
	helpers.assertFeatureSpecCount(ctx, t, 1)
	helpers.assertFeatureGroupLookupCount(ctx, t, 1)
	helpers.assertWebFeatureSnapshotCount(ctx, t, 1)
	helpers.assertWebFeatureChromiumHistogramEnumValueCount(ctx, t, 1)
	helpers.assertBrowserFeatureSupportEventCount(ctx, t, 1)
	helpers.assertLatestDailyChromiumHistogramMetricsCount(ctx, t, 1)
	helpers.assertFeatureDiscouragedDetailsCount(ctx, t, 1)

	// Delete the web feature
	err := spannerClient.DeleteWebFeature(ctx, *helpers.featureID)
	if err != nil {
		t.Errorf("unable to delete web feature %s", err)
	}

	// There should be no entities now
	helpers.assertWebFeatureCount(ctx, t, 0)
	helpers.assertWPTMetricCount(ctx, t, 0)
	helpers.assertBaselineStatusCount(ctx, t, 0)
	helpers.assertBrowserFeatureAvailabilityCount(ctx, t, 0)
	helpers.assertFeatureSpecCount(ctx, t, 0)
	helpers.assertFeatureGroupLookupCount(ctx, t, 0)
	helpers.assertWebFeatureSnapshotCount(ctx, t, 0)
	helpers.assertWebFeatureChromiumHistogramEnumValueCount(ctx, t, 0)
	helpers.assertBrowserFeatureSupportEventCount(ctx, t, 0)
	helpers.assertLatestDailyChromiumHistogramMetricsCount(ctx, t, 0)
	helpers.assertFeatureDiscouragedDetailsCount(ctx, t, 0)
}
