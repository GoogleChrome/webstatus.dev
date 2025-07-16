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
	"slices"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
)

type mockUpsertGroupConfig struct {
	expectedInputs map[string]gcpspanner.Group
	outputIDs      map[string]string
	outputs        map[string]error
	expectedCount  int
}

type mockUpsertFeatureGroupLookupsConfig struct {
	expectedInput []gcpspanner.FeatureGroupIDsLookup
	output        error
	expectedCount int
}

func TestCalculateAllLookups(t *testing.T) {
	testCases := []struct {
		name             string
		featureKeyToID   map[string]string
		featureData      map[string]web_platform_dx__web_features.FeatureValue
		groupKeyToID     map[string]string
		childToParentMap map[string]string
		expectedLookups  []gcpspanner.FeatureGroupIDsLookup
	}{
		{
			name:           "Deep Hierarchy",
			featureKeyToID: map[string]string{"feat1": "feature_id_1"},
			featureData: map[string]web_platform_dx__web_features.FeatureValue{
				"feat1": {
					Caniuse:         nil,
					CompatFeatures:  nil,
					Description:     "feature 1",
					DescriptionHTML: "<html>",
					Discouraged:     nil,
					Name:            "Feature 1",
					Snapshot:        nil,
					Spec:            nil,
					Status: web_platform_dx__web_features.Status{
						Baseline:         nil,
						BaselineHighDate: nil,
						BaselineLowDate:  nil,
						ByCompatKey:      nil,
						Support: web_platform_dx__web_features.StatusSupport{
							Chrome:         nil,
							ChromeAndroid:  nil,
							Edge:           nil,
							Firefox:        nil,
							FirefoxAndroid: nil,
							Safari:         nil,
							SafariIos:      nil,
						},
					},
					Group: &web_platform_dx__web_features.StringOrStringArray{
						String: valuePtr("grandchild"), StringArray: nil},
				},
			},
			groupKeyToID: map[string]string{
				"root":       "uuid_root",
				"child":      "uuid_child",
				"grandchild": "uuid_grandchild",
			},
			childToParentMap: map[string]string{
				"child":      "root",
				"grandchild": "child",
			},
			expectedLookups: []gcpspanner.FeatureGroupIDsLookup{
				{ID: "uuid_grandchild", WebFeatureID: "feature_id_1", Depth: 0},
				{ID: "uuid_child", WebFeatureID: "feature_id_1", Depth: 1},
				{ID: "uuid_root", WebFeatureID: "feature_id_1", Depth: 2},
			},
		},
		{
			name:           "Multiple Direct Groups",
			featureKeyToID: map[string]string{"feat1": "feature_id_1"},
			featureData: map[string]web_platform_dx__web_features.FeatureValue{
				"feat1": {
					Caniuse:         nil,
					CompatFeatures:  nil,
					Description:     "feature 1",
					DescriptionHTML: "<html>",
					Discouraged:     nil,
					Name:            "Feature 1",
					Snapshot:        nil,
					Spec:            nil,
					Status: web_platform_dx__web_features.Status{
						Baseline:         nil,
						BaselineHighDate: nil,
						BaselineLowDate:  nil,
						ByCompatKey:      nil,
						Support: web_platform_dx__web_features.StatusSupport{
							Chrome:         nil,
							ChromeAndroid:  nil,
							Edge:           nil,
							Firefox:        nil,
							FirefoxAndroid: nil,
							Safari:         nil,
							SafariIos:      nil,
						},
					},
					Group: &web_platform_dx__web_features.StringOrStringArray{
						StringArray: []string{"child1", "child2"}, String: nil},
				},
			},
			groupKeyToID: map[string]string{
				"root":   "uuid_root",
				"child1": "uuid_child1",
				"child2": "uuid_child2",
			},
			childToParentMap: map[string]string{
				"child1": "root",
				"child2": "root",
			},
			expectedLookups: []gcpspanner.FeatureGroupIDsLookup{
				{ID: "uuid_child1", WebFeatureID: "feature_id_1", Depth: 0},
				{ID: "uuid_root", WebFeatureID: "feature_id_1", Depth: 1},
				{ID: "uuid_child2", WebFeatureID: "feature_id_1", Depth: 0},
				// Note: Duplicates are expected here and handled by the DB.
				{ID: "uuid_root", WebFeatureID: "feature_id_1", Depth: 1},
			},
		},
		{
			name:           "Feature with No Group",
			featureKeyToID: map[string]string{"feat1": "feature_id_1"},
			featureData: map[string]web_platform_dx__web_features.FeatureValue{
				"feat1": {
					Caniuse:         nil,
					CompatFeatures:  nil,
					Description:     "feature 1",
					DescriptionHTML: "<html>",
					Discouraged:     nil,
					Name:            "Feature 1",
					Snapshot:        nil,
					Spec:            nil,
					Status: web_platform_dx__web_features.Status{
						Baseline:         nil,
						BaselineHighDate: nil,
						BaselineLowDate:  nil,
						ByCompatKey:      nil,
						Support: web_platform_dx__web_features.StatusSupport{
							Chrome:         nil,
							ChromeAndroid:  nil,
							Edge:           nil,
							Firefox:        nil,
							FirefoxAndroid: nil,
							Safari:         nil,
							SafariIos:      nil,
						},
					},
					// No group associated
					Group: nil,
				},
			},
			groupKeyToID:     map[string]string{"group1": "uuid_1"},
			childToParentMap: map[string]string{},
			expectedLookups:  []gcpspanner.FeatureGroupIDsLookup{}, // Expect no lookups
		},
		{
			name:             "No Features",
			featureKeyToID:   map[string]string{},
			featureData:      map[string]web_platform_dx__web_features.FeatureValue{},
			groupKeyToID:     map[string]string{"group1": "uuid_1"},
			childToParentMap: map[string]string{},
			expectedLookups:  []gcpspanner.FeatureGroupIDsLookup{},
		},
	}

	consumer := NewWebFeatureGroupsConsumer(nil)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run the calculation
			actualLookups := consumer.calculateAllLookups(
				context.Background(), tc.featureKeyToID, tc.featureData, tc.groupKeyToID, tc.childToParentMap)

			sortLookups(actualLookups)
			sortLookups(tc.expectedLookups)

			if !slices.Equal(actualLookups, tc.expectedLookups) {
				t.Errorf("lookup slice mismatch.\ngot= %v\nwant=%v", actualLookups, tc.expectedLookups)
			}
		})
	}
}

// sortLookups sorts the slice for deterministic testing.
func sortLookups(lookups []gcpspanner.FeatureGroupIDsLookup) {
	slices.SortFunc(lookups, func(a, b gcpspanner.FeatureGroupIDsLookup) int {
		if a.WebFeatureID != b.WebFeatureID {
			return slices.Compare([]string{a.WebFeatureID}, []string{b.WebFeatureID})
		}
		if a.ID != b.ID {
			return slices.Compare([]string{a.ID}, []string{b.ID})
		}
		if a.Depth != b.Depth {
			return int(a.Depth - b.Depth)
		}

		return 0
	})
}

func TestInsertWebFeatureGroups(t *testing.T) {
	testCases := []struct {
		name                             string
		mockUpsertGroupCfg               mockUpsertGroupConfig
		mockUpsertFeatureGroupLookupsCfg mockUpsertFeatureGroupLookupsConfig
		featureKeyToID                   map[string]string
		featureData                      map[string]web_platform_dx__web_features.FeatureValue
		groupData                        map[string]web_platform_dx__web_features.GroupData
		expectedError                    error
	}{
		{
			name: "Success with single and multiple groups per feature",
			mockUpsertGroupCfg: mockUpsertGroupConfig{
				expectedInputs: map[string]gcpspanner.Group{
					"group1": {GroupKey: "group1", Name: "Group 1"},
					"group2": {GroupKey: "group2", Name: "Group 2"},
					"child3": {GroupKey: "child3", Name: "Child 3"},
				},
				outputIDs: map[string]string{
					"group1": "uuid1",
					"group2": "uuid2",
					"child3": "uuid3",
				},
				outputs: map[string]error{
					"group1": nil,
					"group2": nil,
					"child3": nil,
				},
				expectedCount: 3,
			},
			mockUpsertFeatureGroupLookupsCfg: mockUpsertFeatureGroupLookupsConfig{
				expectedInput: []gcpspanner.FeatureGroupIDsLookup{
					{ID: "uuid1", WebFeatureID: "featureID1", Depth: 0},
					{ID: "uuid2", WebFeatureID: "featureID1", Depth: 0},
					{ID: "uuid2", WebFeatureID: "featureID2", Depth: 0},
				},
				output:        nil,
				expectedCount: 1,
			},
			featureKeyToID: map[string]string{
				"feature1": "featureID1",
				"feature2": "featureID2",
			},
			featureData: map[string]web_platform_dx__web_features.FeatureValue{
				"feature1": {
					Group: &web_platform_dx__web_features.StringOrStringArray{
						StringArray: []string{"group1", "group2"},
						String:      nil,
					},
					Caniuse:         nil,
					CompatFeatures:  nil,
					Description:     "",
					DescriptionHTML: "<html>",
					Discouraged:     nil,
					Name:            "",
					Snapshot:        nil,
					Spec:            nil,
					Status: web_platform_dx__web_features.Status{
						Baseline:         nil,
						BaselineHighDate: nil,
						BaselineLowDate:  nil,
						ByCompatKey:      nil,
						Support: web_platform_dx__web_features.StatusSupport{
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
					Group: &web_platform_dx__web_features.StringOrStringArray{
						String:      valuePtr("group2"),
						StringArray: nil,
					},
					Caniuse:         nil,
					CompatFeatures:  nil,
					Description:     "",
					DescriptionHTML: "<html>",
					Discouraged:     nil,
					Name:            "",
					Snapshot:        nil,
					Spec:            nil,
					Status: web_platform_dx__web_features.Status{
						Baseline:         nil,
						BaselineHighDate: nil,
						BaselineLowDate:  nil,
						ByCompatKey:      nil,
						Support: web_platform_dx__web_features.StatusSupport{
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
			groupData: map[string]web_platform_dx__web_features.GroupData{
				"group1": {Name: "Group 1", Parent: nil},
				"group2": {Name: "Group 2", Parent: nil},
				"child3": {Name: "Child 3", Parent: valuePtr("group1")},
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := newMockWebFeatureGroupsClient(
				t, tc.mockUpsertGroupCfg, tc.mockUpsertFeatureGroupLookupsCfg)
			consumer := NewWebFeatureGroupsConsumer(mockClient)

			err := consumer.InsertWebFeatureGroups(context.TODO(), tc.featureKeyToID, tc.featureData, tc.groupData)

			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error: got %v, want %v", err, tc.expectedError)
			}

			if mockClient.upsertGroupCount != tc.mockUpsertGroupCfg.expectedCount {
				t.Errorf("expected %d calls to UpsertGroup, got %d",
					tc.mockUpsertGroupCfg.expectedCount, mockClient.upsertGroupCount)
			}

			if mockClient.upsertFeatureGroupLookupsCount != tc.mockUpsertFeatureGroupLookupsCfg.expectedCount {
				t.Errorf("expected %d calls to UpsertFeatureGroupLookups, got %d",
					tc.mockUpsertFeatureGroupLookupsCfg.expectedCount, mockClient.upsertFeatureGroupLookupsCount)
			}
		})
	}
}

type mockWebFeatureGroupsClient struct {
	t *testing.T

	mockUpsertGroupCfg               mockUpsertGroupConfig
	mockUpsertFeatureGroupLookupsCfg mockUpsertFeatureGroupLookupsConfig

	upsertGroupCount               int
	upsertFeatureGroupLookupsCount int
}

func (c *mockWebFeatureGroupsClient) UpsertGroup(_ context.Context, group gcpspanner.Group) (*string, error) {
	if len(c.mockUpsertGroupCfg.expectedInputs) <= c.upsertGroupCount {
		c.t.Fatal("no more expected input for UpsertGroup")
	}
	if len(c.mockUpsertGroupCfg.outputs) <= c.upsertGroupCount {
		c.t.Fatal("no more configured outputs for UpsertGroup")
	}

	expectedInput, found := c.mockUpsertGroupCfg.expectedInputs[group.GroupKey]
	if !found {
		c.t.Errorf("unexpected input %v", group)
	}
	if !reflect.DeepEqual(expectedInput, group) {
		c.t.Errorf("unexpected input expected %v received %v", expectedInput, group)
	}
	c.upsertGroupCount++

	output := c.mockUpsertGroupCfg.outputIDs[group.GroupKey]

	return &output, c.mockUpsertGroupCfg.outputs[group.GroupKey]
}

func (c *mockWebFeatureGroupsClient) UpsertFeatureGroupLookups(
	_ context.Context, lookups []gcpspanner.FeatureGroupIDsLookup) error {
	expectedInput := c.mockUpsertFeatureGroupLookupsCfg.expectedInput
	sortLookups(expectedInput)
	sortLookups(lookups)
	if !slices.Equal(expectedInput, lookups) {
		c.t.Errorf("unexpected input for UpsertFeatureGroupLookups expected %v received %v", expectedInput, lookups)
	}
	c.upsertFeatureGroupLookupsCount++

	return c.mockUpsertFeatureGroupLookupsCfg.output
}

func newMockWebFeatureGroupsClient(t *testing.T,
	upsertGroupCfg mockUpsertGroupConfig,
	upsertFeatureGroupLookupsCfg mockUpsertFeatureGroupLookupsConfig,
) *mockWebFeatureGroupsClient {

	return &mockWebFeatureGroupsClient{
		t:                                t,
		mockUpsertGroupCfg:               upsertGroupCfg,
		mockUpsertFeatureGroupLookupsCfg: upsertFeatureGroupLookupsCfg,
		upsertGroupCount:                 0,
		upsertFeatureGroupLookupsCount:   0,
	}
}
