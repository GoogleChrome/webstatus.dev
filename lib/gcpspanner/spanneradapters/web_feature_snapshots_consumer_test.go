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

// nolint: dupl // WONTFIX
package spanneradapters

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
	"github.com/GoogleChrome/webstatus.dev/lib/webdxfeaturetypes"
)

type mockUpsertSnapshotConfig struct {
	expectedInputs map[string]gcpspanner.Snapshot
	outputIDs      map[string]string
	outputs        map[string]error
	expectedCount  int
}

type mockUpsertWebFeatureSnapshotConfig struct {
	expectedInputs map[string]gcpspanner.WebFeatureSnapshot
	outputs        map[string]error
	expectedCount  int
}

type mockWebFeatureSnapshotsClient struct {
	t *testing.T

	mockUpsertSnapshotCfg           mockUpsertSnapshotConfig
	mockUpsertWebFeatureSnapshotCfg mockUpsertWebFeatureSnapshotConfig

	upsertSnapshotCount           int
	upsertWebFeatureSnapshotCount int
}

func newMockWebFeatureSnapshotsClient(t *testing.T,
	upsertSnapshotCfg mockUpsertSnapshotConfig,
	upsertWebFeatureSnapshotCfg mockUpsertWebFeatureSnapshotConfig) *mockWebFeatureSnapshotsClient {

	return &mockWebFeatureSnapshotsClient{
		t:                               t,
		mockUpsertSnapshotCfg:           upsertSnapshotCfg,
		mockUpsertWebFeatureSnapshotCfg: upsertWebFeatureSnapshotCfg,
		upsertSnapshotCount:             0,
		upsertWebFeatureSnapshotCount:   0,
	}
}

func (c *mockWebFeatureSnapshotsClient) UpsertSnapshot(
	_ context.Context, snapshot gcpspanner.Snapshot) (*string, error) {
	if len(c.mockUpsertSnapshotCfg.expectedInputs) <= c.upsertSnapshotCount {
		c.t.Fatal("no more expected input for UpsertSnapshot")
	}
	if len(c.mockUpsertSnapshotCfg.outputs) <= c.upsertSnapshotCount {
		c.t.Fatal("no more configured outputs for UpsertSnapshot")
	}

	expectedInput, found := c.mockUpsertSnapshotCfg.expectedInputs[snapshot.SnapshotKey]
	if !found {
		c.t.Errorf("unexpected input %v", snapshot)
	}
	if !reflect.DeepEqual(expectedInput, snapshot) {
		c.t.Errorf("unexpected input expected %v received %v", expectedInput, snapshot)
	}
	c.upsertSnapshotCount++
	ret := c.mockUpsertSnapshotCfg.outputIDs[snapshot.SnapshotKey]

	return &ret, c.mockUpsertSnapshotCfg.outputs[snapshot.SnapshotKey]
}

func (c *mockWebFeatureSnapshotsClient) UpsertWebFeatureSnapshot(
	_ context.Context, snapshot gcpspanner.WebFeatureSnapshot) error {
	if len(c.mockUpsertWebFeatureSnapshotCfg.expectedInputs) <= c.upsertWebFeatureSnapshotCount {
		c.t.Fatal("no more expected input for UpsertWebFeatureSnapshot")
	}
	if len(c.mockUpsertWebFeatureSnapshotCfg.outputs) <= c.upsertWebFeatureSnapshotCount {
		c.t.Fatal("no more configured outputs for UpsertWebFeatureSnapshot")
	}

	expectedInput, found := c.mockUpsertWebFeatureSnapshotCfg.expectedInputs[snapshot.WebFeatureID]
	if !found {
		c.t.Errorf("unexpected input %v", snapshot)
	}
	if !reflect.DeepEqual(expectedInput, snapshot) {
		c.t.Errorf("unexpected input expected %v received %v", expectedInput, snapshot)
	}
	c.upsertWebFeatureSnapshotCount++

	return c.mockUpsertWebFeatureSnapshotCfg.outputs[snapshot.WebFeatureID]
}

func TestInsertWebFeatureSnapshots(t *testing.T) {
	testCases := []struct {
		name                            string
		mockUpsertSnapshotCfg           mockUpsertSnapshotConfig
		mockUpsertWebFeatureSnapshotCfg mockUpsertWebFeatureSnapshotConfig
		featureKeyToID                  map[string]string
		featureData                     webdxfeaturetypes.FeatureKinds
		snapshotData                    map[string]web_platform_dx__web_features.SnapshotData
		expectedError                   error
	}{
		{
			name: "Success with single and multiple snapshots per feature",
			mockUpsertSnapshotCfg: mockUpsertSnapshotConfig{
				expectedInputs: map[string]gcpspanner.Snapshot{
					"snapshot1": {SnapshotKey: "snapshot1", Name: "Snapshot 1"},
					"snapshot2": {SnapshotKey: "snapshot2", Name: "Snapshot 2"},
				},
				outputIDs: map[string]string{
					"snapshot1": "uuid1",
					"snapshot2": "uuid2",
				},
				outputs: map[string]error{
					"snapshot1": nil,
					"snapshot2": nil,
				},
				expectedCount: 2,
			},
			mockUpsertWebFeatureSnapshotCfg: mockUpsertWebFeatureSnapshotConfig{
				expectedInputs: map[string]gcpspanner.WebFeatureSnapshot{
					"featureID1": {WebFeatureID: "featureID1", SnapshotIDs: []string{"uuid1", "uuid2"}}, // Multiple snapshots
					"featureID2": {WebFeatureID: "featureID2", SnapshotIDs: []string{"uuid1"}},          // Single snapshot
				},
				outputs: map[string]error{
					"featureID1": nil,
					"featureID2": nil,
				},
				expectedCount: 2,
			},
			featureKeyToID: map[string]string{
				"feature1": "featureID1",
				"feature2": "featureID2",
			},
			featureData: webdxfeaturetypes.FeatureKinds{
				Moved: nil,
				Split: nil,
				Data: map[string]web_platform_dx__web_features.FeatureData{
					"feature1": {
						Discouraged: nil,
						Snapshot: &web_platform_dx__web_features.StringOrStrings{
							StringArray: []string{"snapshot1", "snapshot2"},
							String:      nil,
						},
						Kind:            web_platform_dx__web_features.Feature,
						Caniuse:         nil,
						CompatFeatures:  nil,
						Description:     "",
						DescriptionHTML: "<html>",
						Name:            "",
						Group:           nil,
						Spec:            nil,
						Status: web_platform_dx__web_features.StatusHeadline{
							ByCompatKey:      nil,
							Baseline:         nil,
							BaselineHighDate: nil,
							BaselineLowDate:  nil,
							Support: web_platform_dx__web_features.Support{
								Chrome:         nil,
								ChromeAndroid:  nil,
								Edge:           nil,
								Firefox:        nil,
								FirefoxAndroid: nil,
								Safari:         nil,
								SafariIos:      nil,
							},
						},
					},
					"feature2": {
						Discouraged: nil,
						Snapshot: &web_platform_dx__web_features.StringOrStrings{
							String:      valuePtr("snapshot1"),
							StringArray: nil,
						},
						Kind:            web_platform_dx__web_features.Feature,
						Caniuse:         nil,
						CompatFeatures:  nil,
						Description:     "",
						DescriptionHTML: "<html>",
						Name:            "",
						Group:           nil,
						Spec:            nil,
						Status: web_platform_dx__web_features.StatusHeadline{
							Baseline:         nil,
							BaselineHighDate: nil,
							BaselineLowDate:  nil,
							ByCompatKey:      nil,
							Support: web_platform_dx__web_features.Support{
								Chrome:         nil,
								ChromeAndroid:  nil,
								Edge:           nil,
								Firefox:        nil,
								FirefoxAndroid: nil,
								Safari:         nil,
								SafariIos:      nil,
							},
						},
					},
				},
			},
			snapshotData: map[string]web_platform_dx__web_features.SnapshotData{
				"snapshot1": {Name: "Snapshot 1", Spec: ""},
				"snapshot2": {Name: "Snapshot 2", Spec: ""},
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := newMockWebFeatureSnapshotsClient(t, tc.mockUpsertSnapshotCfg, tc.mockUpsertWebFeatureSnapshotCfg)
			consumer := NewWebFeatureSnapshotsConsumer(mockClient)

			err := consumer.InsertWebFeatureSnapshots(context.TODO(), tc.featureKeyToID, tc.featureData, tc.snapshotData)

			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error: got %v, want %v", err, tc.expectedError)
			}

			if mockClient.upsertSnapshotCount != tc.mockUpsertSnapshotCfg.expectedCount {
				t.Errorf("expected %d calls to UpsertSnapshot, got %d",
					tc.mockUpsertSnapshotCfg.expectedCount, mockClient.upsertSnapshotCount)
			}

			if mockClient.upsertWebFeatureSnapshotCount != tc.mockUpsertWebFeatureSnapshotCfg.expectedCount {
				t.Errorf("expected %d calls to UpsertWebFeatureSnapshot, got %d",
					tc.mockUpsertWebFeatureSnapshotCfg.expectedCount, mockClient.upsertWebFeatureSnapshotCount)
			}
		})
	}
}
