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

package spanneradapters

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/developersignaltypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/google/go-cmp/cmp"
)

type SyncLatestFeatureDeveloperSignalsConfig struct {
	expectedData []gcpspanner.FeatureDeveloperSignal
	err          error
}

type MockDeveloperSignalsClient struct {
	// Config for GetAllMovedWebFeatures
	GetAllMovedWebFeaturesConfig *GetAllMovedWebFeaturesConfig

	// Config for SyncLatestFeatureDeveloperSignals
	SyncLatestFeatureDeveloperSignalsConfig *SyncLatestFeatureDeveloperSignalsConfig
	t                                       *testing.T
}

func (m *MockDeveloperSignalsClient) GetAllMovedWebFeatures(_ context.Context) ([]gcpspanner.MovedWebFeature, error) {
	return m.GetAllMovedWebFeaturesConfig.output, m.GetAllMovedWebFeaturesConfig.err
}

func (m *MockDeveloperSignalsClient) SyncLatestFeatureDeveloperSignals(
	_ context.Context,
	data []gcpspanner.FeatureDeveloperSignal,
) error {
	// Sort slices for deterministic comparison
	cmpFunc := func(i, j gcpspanner.FeatureDeveloperSignal) int {
		if i.WebFeatureKey < j.WebFeatureKey {
			return -1
		}
		if i.WebFeatureKey > j.WebFeatureKey {
			return 1
		}

		return 0
	}
	slices.SortFunc(data, cmpFunc)
	slices.SortFunc(m.SyncLatestFeatureDeveloperSignalsConfig.expectedData, cmpFunc)
	if diff := cmp.Diff(m.SyncLatestFeatureDeveloperSignalsConfig.expectedData, data); diff != "" {
		m.t.Errorf("unexpected data (-want +got): %s", diff)
	}

	return m.SyncLatestFeatureDeveloperSignalsConfig.err
}

func TestSyncLatestFeatureDeveloperSignals(t *testing.T) {
	var fakeGetAllMovedWebFeaturesError = errors.New("fake error")
	var fakeSyncError = errors.New("fake sync error")

	testCases := []struct {
		name                   string
		input                  *developersignaltypes.FeatureDeveloperSignals
		syncConfig             *SyncLatestFeatureDeveloperSignalsConfig
		allMovedFeaturesConfig *GetAllMovedWebFeaturesConfig
		expectedError          error
	}{
		{
			name: "Success",
			input: &developersignaltypes.FeatureDeveloperSignals{
				"feature1": {Upvotes: 100, Link: "link1"},
				"feature2": {Upvotes: 200, Link: "link2"},
			},
			syncConfig: &SyncLatestFeatureDeveloperSignalsConfig{
				expectedData: []gcpspanner.FeatureDeveloperSignal{
					{WebFeatureKey: "feature1", Upvotes: 100, Link: "link1"},
					{WebFeatureKey: "feature2", Upvotes: 200, Link: "link2"},
				},
				err: nil,
			},
			allMovedFeaturesConfig: &GetAllMovedWebFeaturesConfig{
				output: []gcpspanner.MovedWebFeature{},
				err:    nil,
			},
			expectedError: nil,
		},
		{
			name:                   "Empty input",
			input:                  &developersignaltypes.FeatureDeveloperSignals{},
			expectedError:          nil,
			syncConfig:             nil,
			allMovedFeaturesConfig: nil,
		},
		{
			name: "Spanner client error on sync",
			input: &developersignaltypes.FeatureDeveloperSignals{
				"feature1": {Upvotes: 100, Link: "link1"},
			},
			syncConfig: &SyncLatestFeatureDeveloperSignalsConfig{
				expectedData: []gcpspanner.FeatureDeveloperSignal{
					{WebFeatureKey: "feature1", Upvotes: 100, Link: "link1"},
				},
				err: fakeSyncError,
			},
			allMovedFeaturesConfig: &GetAllMovedWebFeaturesConfig{
				output: []gcpspanner.MovedWebFeature{},
				err:    nil,
			},
			expectedError: fakeSyncError,
		},
		{
			name: "Error getting moved features",
			input: &developersignaltypes.FeatureDeveloperSignals{
				"feature1": {Upvotes: 100, Link: "link1"},
			},
			allMovedFeaturesConfig: &GetAllMovedWebFeaturesConfig{
				output: nil,
				err:    fakeGetAllMovedWebFeaturesError,
			},
			syncConfig:    nil,
			expectedError: fakeGetAllMovedWebFeaturesError,
		},
		{
			name: "Conflict with moved feature",
			input: &developersignaltypes.FeatureDeveloperSignals{
				"feature1": {Upvotes: 100, Link: "link1"}, // This one is moved
				"feature2": {Upvotes: 200, Link: "link2"}, // This is the new one
			},
			allMovedFeaturesConfig: &GetAllMovedWebFeaturesConfig{
				output: []gcpspanner.MovedWebFeature{
					{OriginalFeatureKey: "feature1", NewFeatureKey: "feature2"},
				},
				err: nil,
			},
			expectedError: ErrConflictMigratingFeatureKey,
			syncConfig:    nil,
		},
		{
			name: "Success with moved feature",
			input: &developersignaltypes.FeatureDeveloperSignals{
				"feature1": {Upvotes: 100, Link: "link1"}, // This one is moved
				"feature3": {Upvotes: 300, Link: "link3"}, // This one is not moved
			},
			allMovedFeaturesConfig: &GetAllMovedWebFeaturesConfig{
				output: []gcpspanner.MovedWebFeature{
					{OriginalFeatureKey: "feature1", NewFeatureKey: "feature2"},
				},
				err: nil,
			},
			syncConfig: &SyncLatestFeatureDeveloperSignalsConfig{
				expectedData: []gcpspanner.FeatureDeveloperSignal{
					{WebFeatureKey: "feature2", Upvotes: 100, Link: "link1"},
					{WebFeatureKey: "feature3", Upvotes: 300, Link: "link3"},
				},
				err: nil,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			consumer := NewDeveloperSignalsConsumer(&MockDeveloperSignalsClient{
				GetAllMovedWebFeaturesConfig:            tc.allMovedFeaturesConfig,
				SyncLatestFeatureDeveloperSignalsConfig: tc.syncConfig,
				t:                                       t,
			})
			err := consumer.SyncLatestFeatureDeveloperSignals(context.Background(), tc.input)

			if !errors.Is(err, tc.expectedError) {
				t.Errorf("expected error %v, got %v", tc.expectedError, err)
			}
		})
	}
}
