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
// WITHOUT WARRANTIES, OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcpspanner

import (
	"context"
	"errors"
	"math/big"
	"slices"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/iterator"
)

func getSampleFeatures() []WebFeature {
	return []WebFeature{
		{
			Name:            "Feature 1",
			FeatureKey:      "feature1",
			Description:     "Wow what a feature description",
			DescriptionHTML: "Feature <b>1</b> description",
		},
		{
			Name:            "Feature 2",
			FeatureKey:      "feature2",
			Description:     "Feature 2 description",
			DescriptionHTML: "Feature <b>2</b> description",
		},
		{
			Name:            "Feature 3",
			FeatureKey:      "feature3",
			Description:     "Feature 3 description",
			DescriptionHTML: "Feature <b>3</b> description",
		},
		{
			Name:            "Feature 4",
			FeatureKey:      "feature4",
			Description:     "Feature 4 description",
			DescriptionHTML: "Feature <b>4</b> description",
		},
	}
}

// Helper method to get all the features in a stable order.
func (c *Client) ReadAllWebFeatures(ctx context.Context, t *testing.T) ([]WebFeature, error) {
	stmt := spanner.NewStatement(`SELECT
		ID, FeatureKey, Name, Description, DescriptionHtml
	FROM WebFeatures ORDER BY FeatureKey ASC`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []WebFeature
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var feature SpannerWebFeature
		if err := row.ToStruct(&feature); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		if feature.ID == "" {
			t.Error("retrieved feature ID is empty")
		}
		ret = append(ret, feature.WebFeature)
	}

	return ret, nil
}

func (c *Client) DeleteWebFeature(ctx context.Context, internalID string) error {
	_, err := c.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		mutation := spanner.Delete(webFeaturesTable, spanner.Key{internalID})

		return txn.BufferWrite([]*spanner.Mutation{mutation})
	})
	if err != nil {
		// TODO wrap the error and return it

		return err
	}

	return nil
}

func TestUpsertWebFeature(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	sampleFeatures := getSampleFeatures()
	for _, feature := range sampleFeatures {
		_, err := spannerClient.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert. %s", err.Error())
		}
	}
	features, err := spannerClient.ReadAllWebFeatures(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	if !slices.Equal(sampleFeatures, features) {
		t.Errorf("unequal features. expected %+v actual %+v", sampleFeatures, features)
	}

	_, err = spannerClient.UpsertWebFeature(ctx, WebFeature{
		Name:            "Feature 1!!",
		FeatureKey:      "feature1",
		Description:     "Feature 1 description!",
		DescriptionHTML: "Feature <i>1</i> description!",
	})
	if err != nil {
		t.Errorf("unexpected error during update. %s", err.Error())
	}

	features, err = spannerClient.ReadAllWebFeatures(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}

	expectedPageAfterUpdate := []WebFeature{
		{
			Name:            "Feature 1!!", // Updated field
			FeatureKey:      "feature1",
			Description:     "Feature 1 description!", // Updated field
			DescriptionHTML: "Feature <i>1</i> description!",
		},
		{
			Name:            "Feature 2",
			FeatureKey:      "feature2",
			Description:     "Feature 2 description",
			DescriptionHTML: "Feature <b>2</b> description",
		},
		{
			Name:            "Feature 3",
			FeatureKey:      "feature3",
			Description:     "Feature 3 description",
			DescriptionHTML: "Feature <b>3</b> description",
		},
		{
			Name:            "Feature 4",
			FeatureKey:      "feature4",
			Description:     "Feature 4 description",
			DescriptionHTML: "Feature <b>4</b> description",
		},
	}
	if !slices.Equal[[]WebFeature](expectedPageAfterUpdate, features) {
		t.Errorf("unequal features after update. expected %+v actual %+v", sampleFeatures, features)
	}

	expectedKeys := []string{
		"feature1",
		"feature2",
		"feature3",
		"feature4",
	}
	keys, err := spannerClient.FetchAllFeatureKeys(ctx)
	if err != nil {
		t.Errorf("unexpected error fetching all keys")
	}
	slices.Sort(keys)
	if !slices.Equal(keys, expectedKeys) {
		t.Errorf("unequal keys. expected %+v actual %+v", expectedKeys, keys)
	}
}

type syncWebFeaturesRedirectCase struct {
	sampleWPTRunOld WPTRun
	sampleWPTRunNew WPTRun
}

func (s syncWebFeaturesRedirectCase) postFirstSyncSetup(
	ctx context.Context, t *testing.T, featureKeyToIDMap map[string]string) {
	// Insert some data
	// Insert WPT Runs
	err := spannerClient.InsertWPTRun(ctx, s.sampleWPTRunOld)
	if err != nil {
		t.Fatalf("Failed to insert run: %v", err)
	}
	err = spannerClient.InsertWPTRun(ctx, s.sampleWPTRunNew)
	if err != nil {
		t.Fatalf("Failed to insert run: %v", err)
	}
	// Insert WPT Metrics for feature-a
	err = spannerClient.UpsertWPTRunFeatureMetrics(ctx, s.sampleWPTRunOld.RunID,
		map[string]WPTRunFeatureMetric{
			"feature-a": {
				TotalTests:        valuePtr(int64(123)),
				TestPass:          valuePtr(int64(45)),
				TotalSubtests:     valuePtr(int64(789)),
				SubtestPass:       valuePtr(int64(234)),
				FeatureRunDetails: nil,
			},
		})
	if err != nil {
		t.Fatalf("Failed to insert WPT metrics: %v", err)
	}
	err = spannerClient.UpsertWPTRunFeatureMetrics(ctx, s.sampleWPTRunNew.RunID,
		map[string]WPTRunFeatureMetric{
			"feature-a": {
				TotalTests:        valuePtr(int64(124)),
				TestPass:          valuePtr(int64(46)),
				TotalSubtests:     valuePtr(int64(790)),
				SubtestPass:       valuePtr(int64(235)),
				FeatureRunDetails: nil,
			},
		})
	if err != nil {
		t.Fatalf("Failed to insert WPT metrics: %v", err)
	}
	histogramName := metricdatatypes.HistogramName("testHistogram")
	// Insert chromium enum value
	enumID, err := spannerClient.UpsertChromiumHistogramEnum(ctx, ChromiumHistogramEnum{
		HistogramName: string(histogramName),
	})
	if err != nil {
		t.Fatalf("Failed to insert chromium histogram enum: %v", err)
	}

	bucketID := int64(100)
	featureEnumID, err := spannerClient.UpsertChromiumHistogramEnumValue(ctx, ChromiumHistogramEnumValue{
		ChromiumHistogramEnumID: *enumID,
		BucketID:                bucketID,
		Label:                   "FeatureAOrB",
	})
	if err != nil {
		t.Fatalf("Failed to insert chromium histogram enum value: %v", err)
	}

	// Insert chromium histogram metrics for feature-a.
	err = spannerClient.StoreDailyChromiumHistogramMetrics(ctx,
		histogramName,
		map[int64]DailyChromiumHistogramMetric{
			bucketID: {
				Day:  civil.Date{Year: 2000, Month: time.January, Day: 20},
				Rate: *big.NewRat(93, 100),
			},
		})
	if err != nil {
		t.Fatalf("Failed to store day 1 chromium histogram metrics: %v", err)
	}

	err = spannerClient.StoreDailyChromiumHistogramMetrics(ctx, histogramName,
		map[int64]DailyChromiumHistogramMetric{
			bucketID: {
				Day:  civil.Date{Year: 2000, Month: time.January, Day: 21},
				Rate: *big.NewRat(94, 100),
			},
		})
	if err != nil {
		t.Fatalf("Failed to store day 2 chromium histogram metrics: %v", err)
	}

	// Associate the enum to feature-a
	err = spannerClient.UpsertWebFeatureChromiumHistogramEnumValue(ctx,
		WebFeatureChromiumHistogramEnumValue{
			WebFeatureID:                 featureKeyToIDMap["feature-a"],
			ChromiumHistogramEnumValueID: *featureEnumID,
		})
	if err != nil {
		t.Fatalf("Failed to upsert web feature chromium histogram enum value: %v", err)
	}

	err = spannerClient.SyncLatestDailyChromiumHistogramMetrics(ctx)
	if err != nil {
		t.Fatalf("Failed to sync latest chromium histogram metrics: %v", err)
	}

	// Add Feature Developer Signals
	err = spannerClient.SyncLatestFeatureDeveloperSignals(ctx, []FeatureDeveloperSignal{
		{
			WebFeatureKey: "feature-a",
			Upvotes:       1,
			Link:          "https://example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to sync latest feature developer signals: %v", err)
	}
}

func (s syncWebFeaturesRedirectCase) postSecondSyncCheck(
	ctx context.Context, t *testing.T, featureKeyToIDMap map[string]string) {
	// Check that the data for feature-a is missing and that feature-b has the data
	// WPTRunMetricData
	metrics, err := spannerClient.getAllWPTRunFeatureMetricIDsByWebFeatureID(ctx,
		featureKeyToIDMap["feature-a"])
	if err != nil {
		t.Fatalf("unexpected error reading WPT metrics for feature-a. %s", err.Error())
	}
	if len(metrics) != 0 {
		t.Fatal("expected no WPT metrics for feature-a")
	}
	metrics, err = spannerClient.getAllWPTRunFeatureMetricIDsByWebFeatureID(ctx,
		featureKeyToIDMap["feature-b"])
	if err != nil {
		t.Fatalf("unexpected error reading WPT metrics for feature-b. %s", err.Error())
	}
	if len(metrics) != 2 {
		t.Fatal("expected 2 WPT metrics for feature-b")
	}
	metric, err := spannerClient.GetMetricByRunIDAndFeatureID(ctx, s.sampleWPTRunOld.RunID, "feature-b")
	if err != nil {
		t.Fatalf("unexpected error getting WPT metric. %s", err.Error())
	}
	expectedMetric := &WPTRunFeatureMetric{
		TotalTests:        valuePtr(int64(123)),
		TestPass:          valuePtr(int64(45)),
		TotalSubtests:     valuePtr(int64(789)),
		SubtestPass:       valuePtr(int64(234)),
		FeatureRunDetails: nil,
	}
	if diff := cmp.Diff(expectedMetric, metric); diff != "" {
		t.Errorf("WPT metrics mismatch (-want +got):\n%s", diff)
	}
	metric, err = spannerClient.GetMetricByRunIDAndFeatureID(ctx, s.sampleWPTRunNew.RunID, "feature-b")
	if err != nil {
		t.Fatalf("unexpected error getting WPT metric. %s", err.Error())
	}
	expectedMetric = &WPTRunFeatureMetric{
		TotalTests:        valuePtr(int64(124)),
		TestPass:          valuePtr(int64(46)),
		TotalSubtests:     valuePtr(int64(790)),
		SubtestPass:       valuePtr(int64(235)),
		FeatureRunDetails: nil,
	}
	if diff := cmp.Diff(expectedMetric, metric); diff != "" {
		t.Errorf("WPT metrics mismatch (-want +got):\n%s", diff)
	}
	latestWPTMetric, err := spannerClient.getAllSpannerLatestWPTRunFeatureMetricIDsByWebFeatureID(ctx,
		featureKeyToIDMap["feature-b"])
	if err != nil {
		t.Fatalf("unexpected error reading latest WPT metrics for feature-b. %s", err.Error())
	}
	if len(latestWPTMetric) != 1 {
		t.Fatal("expected 1 latest WPT metric for feature-b")
	}
	expectedLatestWPTMetric := SpannerLatestWPTRunFeatureMetric{
		RunMetricID:  metrics[0].ID,
		WebFeatureID: featureKeyToIDMap["feature-b"],
		BrowserName:  s.sampleWPTRunOld.BrowserName,
		Channel:      s.sampleWPTRunOld.Channel,
	}
	if diff := cmp.Diff(expectedLatestWPTMetric, latestWPTMetric[0]); diff != "" {
		t.Errorf("latest WPT metrics mismatch (-want +got):\n%s", diff)
	}

	// Check Chromium Enum metrics
	chromiumLatestMetrics, err := spannerClient.getAllLatestDailyChromiumHistogramMetricsByFeatureID(
		ctx, featureKeyToIDMap["feature-b"])
	if err != nil {
		t.Errorf("unexpected error reading latest chromium metrics for feature-b. %s", err.Error())
	}

	if len(chromiumLatestMetrics) != 1 {
		t.Errorf("expected 1 latest chromium metric for feature-b. Received %d", len(chromiumLatestMetrics))
	}

	webFeatureEnums, err := spannerClient.readAllWebFeatureChromiumHistogramEnums(ctx, t)
	if err != nil {
		t.Errorf("unexpected error reading web feature chromium enums. %s", err.Error())
	}
	if len(webFeatureEnums) != 1 {
		t.Errorf("expected 1 web feature chromium enum. Received %d", len(webFeatureEnums))
	}
	if webFeatureEnums[0].WebFeatureID != featureKeyToIDMap["feature-b"] {
		t.Error("expected web feature chromium enum to be for feature-b")
	}

	// Check Feature Developer Signals
	// Check that the signal information is missing for feature-a and now feature-b has the information.
	signals, err := spannerClient.getAllLatestFeatureDeveloperSignalsByWebFeatureID(ctx, featureKeyToIDMap["feature-a"])
	if err != nil {
		t.Fatalf("unexpected error reading feature developer signals for feature-a. %s", err.Error())
	}
	if len(signals) != 0 {
		t.Fatal("expected no feature developer signals for feature-a")
	}

	signals, err = spannerClient.getAllLatestFeatureDeveloperSignalsByWebFeatureID(ctx, featureKeyToIDMap["feature-b"])
	if err != nil {
		t.Fatalf("unexpected error reading feature developer signals for feature-b. %s", err.Error())
	}
	if len(signals) != 1 {
		t.Fatal("expected 1 feature developer signal for feature-b")
	}
}

func (s syncWebFeaturesRedirectCase) secondSyncOptions() []SyncWebFeaturesOption {
	return []SyncWebFeaturesOption{
		WithRedirectTargets(map[string]string{
			"feature-a": "feature-b",
		}),
	}
}

func TestSyncWebFeatures(t *testing.T) {
	ctx := context.Background()

	sampleWPTRunOld := getSampleRuns()[0]
	sampleWPTRunNew := getSampleRuns()[4]

	redirectCase := syncWebFeaturesRedirectCase{
		sampleWPTRunOld: sampleWPTRunOld,
		sampleWPTRunNew: sampleWPTRunNew,
	}

	type syncTestCase struct {
		name                string
		initialState        []WebFeature
		desiredState        []WebFeature
		postFirstSyncSetup  func(ctx context.Context, t *testing.T, featureKeyToIDMap map[string]string)
		secondSyncOptions   []SyncWebFeaturesOption
		expectedState       []WebFeature
		postSecondSyncCheck func(ctx context.Context, t *testing.T, featureKeyToIDMap map[string]string)
	}

	testCases := []syncTestCase{
		{
			name:                "Initial creation",
			initialState:        nil, // No initial state
			desiredState:        getSampleFeatures(),
			expectedState:       getSampleFeatures(),
			postFirstSyncSetup:  nil,
			postSecondSyncCheck: nil,
			secondSyncOptions:   nil,
		},
		{
			name:         "Deletes features not in desired state",
			initialState: getSampleFeatures(),
			desiredState: []WebFeature{
				getSampleFeatures()[0], // feature1
				getSampleFeatures()[2], // feature3
			},
			expectedState: []WebFeature{
				getSampleFeatures()[0],
				getSampleFeatures()[2],
			},
			postFirstSyncSetup:  nil,
			postSecondSyncCheck: nil,
			secondSyncOptions:   nil,
		},
		{
			name:         "Updates existing features",
			initialState: getSampleFeatures(),
			desiredState: func() []WebFeature {
				features := getSampleFeatures()
				features[1].Name = "UPDATED Feature 2"
				features[3].Description = "UPDATED Description 4"

				return features
			}(),
			expectedState: func() []WebFeature {
				features := getSampleFeatures()
				features[1].Name = "UPDATED Feature 2"
				features[3].Description = "UPDATED Description 4"

				return features
			}(),
			postFirstSyncSetup:  nil,
			postSecondSyncCheck: nil,
			secondSyncOptions:   nil,
		},
		{
			name:         "Performs mixed insert, update, and delete",
			initialState: getSampleFeatures(),
			desiredState: []WebFeature{
				{FeatureKey: "feature1", Name: "Updated Feature 1 Name", Description: "", DescriptionHTML: ""},
				getSampleFeatures()[2], // Keep feature3
				{FeatureKey: "feature5", Name: "New Feature 5", Description: "", DescriptionHTML: ""},
			},
			expectedState: []WebFeature{
				{
					FeatureKey:      "feature1",
					Name:            "Updated Feature 1 Name",
					Description:     "Wow what a feature description", // Preserved by merge logic
					DescriptionHTML: "Feature <b>1</b> description",   // Preserved by merge logic
				},
				getSampleFeatures()[2], // feature3 is unchanged
				{
					FeatureKey:      "feature5",
					Name:            "New Feature 5",
					Description:     "", // New fields are empty
					DescriptionHTML: "",
				},
			},
			postFirstSyncSetup:  nil,
			postSecondSyncCheck: nil,
			secondSyncOptions:   nil,
		},
		{
			name:                "No changes when desired state matches current state",
			initialState:        getSampleFeatures(),
			desiredState:        getSampleFeatures(),
			expectedState:       getSampleFeatures(),
			postFirstSyncSetup:  nil,
			postSecondSyncCheck: nil,
			secondSyncOptions:   nil,
		},
		{
			name:                "Deletes all features when desired state is empty",
			initialState:        getSampleFeatures(),
			desiredState:        []WebFeature{},
			expectedState:       nil,
			postFirstSyncSetup:  nil,
			postSecondSyncCheck: nil,
			secondSyncOptions:   nil,
		},
		{
			name: "Redirects feature and moves data",
			initialState: []WebFeature{
				{FeatureKey: "feature-a", Name: "Feature A", Description: "", DescriptionHTML: ""},
				{FeatureKey: "feature-b", Name: "Feature B", Description: "", DescriptionHTML: ""},
			},
			desiredState: []WebFeature{
				{FeatureKey: "feature-b", Name: "Feature B",
					Description: "", DescriptionHTML: ""}, // feature-a is removed
			},

			postFirstSyncSetup: redirectCase.postFirstSyncSetup,
			secondSyncOptions:  redirectCase.secondSyncOptions(),
			expectedState: []WebFeature{
				{FeatureKey: "feature-b", Name: "Feature B", Description: "", DescriptionHTML: ""},
			},
			postSecondSyncCheck: redirectCase.postSecondSyncCheck,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restartDatabaseContainer(t)

			// 1. Setup initial state if provided
			if tc.initialState != nil {
				if err := spannerClient.SyncWebFeatures(ctx, tc.initialState); err != nil {
					t.Fatalf("Failed to set up initial state: %v", err)
				}
			}

			pairs, err := spannerClient.FetchAllWebFeatureIDsAndKeys(ctx)
			if err != nil {
				t.Fatalf("Failed to fetch all web feature IDs and keys: %v", err)
			}
			featureKeyToIDMap := map[string]string{}
			for _, pair := range pairs {
				featureKeyToIDMap[pair.FeatureKey] = pair.ID
			}

			// 2. Add any additional information if needed after the initial features are added.
			if tc.postFirstSyncSetup != nil {
				tc.postFirstSyncSetup(ctx, t, featureKeyToIDMap)
			}

			// 3. Run the sync with the desired state
			if err := spannerClient.SyncWebFeatures(ctx, tc.desiredState, tc.secondSyncOptions...); err != nil {
				t.Fatalf("SyncWebFeatures failed: %v", err)
			}

			// 4. Verify the final state
			featuresInDB, err := spannerClient.ReadAllWebFeatures(ctx, t)
			if err != nil {
				t.Fatalf("ReadAllWebFeatures failed: %v", err)
			}

			if diff := cmp.Diff(tc.expectedState, featuresInDB); diff != "" {
				t.Errorf("features mismatch (-want +got):\n%s", diff)
			}

			// 5. Run post-sync check if provided
			if tc.postSecondSyncCheck != nil {
				tc.postSecondSyncCheck(ctx, t, featureKeyToIDMap)
			}
		})
	}
}
