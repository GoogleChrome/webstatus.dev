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

func TestBuildGroupDescendants(t *testing.T) {
	testCases := []struct {
		name                  string
		groupData             map[string]web_platform_dx__web_features.GroupData
		groupKeyToInternalID  map[string]string
		expectedDescendantMap map[string]gcpspanner.GroupDescendantInfo
	}{
		{
			name: "Simple Hierarchy",
			groupData: map[string]web_platform_dx__web_features.GroupData{
				"group1": {Parent: nil, Name: "Group 1"},
				"group2": {Name: "Group 2", Parent: valuePtr("group1")},
				"group3": {Name: "Group 3", Parent: valuePtr("group1")},
			},
			groupKeyToInternalID: map[string]string{
				"group1": "uuid1",
				"group2": "uuid2",
				"group3": "uuid3",
			},
			expectedDescendantMap: map[string]gcpspanner.GroupDescendantInfo{
				"group1": {DescendantGroupIDs: []string{"uuid2", "uuid3"}},
				"group2": {DescendantGroupIDs: nil},
				"group3": {DescendantGroupIDs: nil},
			},
		},
		{
			name: "Deeper Hierarchy",
			groupData: map[string]web_platform_dx__web_features.GroupData{
				"group1": {Parent: nil, Name: "Group 1"},
				"group2": {Name: "Group 2", Parent: valuePtr("group1")},
				"group3": {Name: "Group 3", Parent: valuePtr("group2")},
				"group4": {Name: "Group 4", Parent: valuePtr("group2")},
			},
			groupKeyToInternalID: map[string]string{
				"group1": "uuid1",
				"group2": "uuid2",
				"group3": "uuid3",
				"group4": "uuid4",
			},
			expectedDescendantMap: map[string]gcpspanner.GroupDescendantInfo{
				"group1": {DescendantGroupIDs: []string{"uuid2", "uuid3", "uuid4"}},
				"group2": {DescendantGroupIDs: []string{"uuid3", "uuid4"}},
				"group3": {DescendantGroupIDs: nil},
				"group4": {DescendantGroupIDs: nil},
			},
		},
		{
			name: "Multiple Roots",
			groupData: map[string]web_platform_dx__web_features.GroupData{
				"group1": {Parent: nil, Name: "Group 1"},
				"group2": {Parent: nil, Name: "Group 2"},
				"group3": {Name: "Group 3", Parent: valuePtr("group1")},
			},
			groupKeyToInternalID: map[string]string{
				"group1": "uuid1",
				"group2": "uuid2",
				"group3": "uuid3",
			},
			expectedDescendantMap: map[string]gcpspanner.GroupDescendantInfo{
				"group1": {DescendantGroupIDs: []string{"uuid3"}},
				"group2": {DescendantGroupIDs: nil},
				"group3": {DescendantGroupIDs: nil},
			},
		},
		{
			name: "No Children",
			groupData: map[string]web_platform_dx__web_features.GroupData{
				"group1": {Name: "Group 1", Parent: nil},
			},
			groupKeyToInternalID: map[string]string{
				"group1": "uuid1",
			},
			expectedDescendantMap: map[string]gcpspanner.GroupDescendantInfo{
				"group1": {DescendantGroupIDs: nil},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			consumer := &WebFeatureGroupConsumer{client: nil}
			actualDescendantMap := consumer.buildGroupDescendants(tc.groupData, tc.groupKeyToInternalID)

			for _, info := range actualDescendantMap {
				slices.Sort(info.DescendantGroupIDs)
			}

			if !reflect.DeepEqual(actualDescendantMap, tc.expectedDescendantMap) {
				t.Errorf("Descendant maps not equal.\nExpected: %v\nReceived: %v", tc.expectedDescendantMap, actualDescendantMap)
			}
		})
	}
}

type mockUpsertGroupConfig struct {
	expectedInputs map[string]gcpspanner.Group
	outputIDs      map[string]string
	outputs        map[string]error
	expectedCount  int
}

type mockUpsertGroupDescendantInfoConfig struct {
	expectedInputs map[string]gcpspanner.GroupDescendantInfo
	outputs        map[string]error
	expectedCount  int
}

type mockUpsertWebFeatureGroupConfig struct {
	expectedInputs map[string]gcpspanner.WebFeatureGroup
	outputs        map[string]error
	expectedCount  int
}

func TestInsertWebFeatureGroups(t *testing.T) {
	testCases := []struct {
		name                         string
		mockUpsertGroupCfg           mockUpsertGroupConfig
		mockUpsertGroupDescendantCfg mockUpsertGroupDescendantInfoConfig
		mockUpsertWebFeatureGroupCfg mockUpsertWebFeatureGroupConfig
		featureKeyToID               map[string]string
		featureData                  map[string]web_platform_dx__web_features.FeatureValue
		groupData                    map[string]web_platform_dx__web_features.GroupData
		expectedError                error
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
			mockUpsertGroupDescendantCfg: mockUpsertGroupDescendantInfoConfig{
				expectedInputs: map[string]gcpspanner.GroupDescendantInfo{
					"group1": {DescendantGroupIDs: []string{"uuid3"}},
					"group2": {DescendantGroupIDs: nil},
					"child3": {DescendantGroupIDs: nil},
				},
				outputs: map[string]error{
					"group1": nil,
					"group2": nil,
					"child3": nil,
				},
				expectedCount: 3,
			},
			mockUpsertWebFeatureGroupCfg: mockUpsertWebFeatureGroupConfig{
				expectedInputs: map[string]gcpspanner.WebFeatureGroup{
					"featureID1": {WebFeatureID: "featureID1", GroupIDs: []string{"uuid1", "uuid3"}},
					"featureID2": {WebFeatureID: "featureID2", GroupIDs: []string{"uuid2"}},
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
			featureData: map[string]web_platform_dx__web_features.FeatureValue{
				"feature1": {
					Group: &web_platform_dx__web_features.StringOrStringArray{
						StringArray: []string{"group1", "child3"},
						String:      nil,
					},
					Caniuse:         nil,
					CompatFeatures:  nil,
					Description:     "",
					DescriptionHTML: "",
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
					DescriptionHTML: "",
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
				t, tc.mockUpsertGroupCfg, tc.mockUpsertGroupDescendantCfg, tc.mockUpsertWebFeatureGroupCfg)
			consumer := NewWebFeatureGroupsConsumer(mockClient)

			err := consumer.InsertWebFeatureGroups(context.TODO(), tc.featureKeyToID, tc.featureData, tc.groupData)

			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error: got %v, want %v", err, tc.expectedError)
			}

			if mockClient.upsertGroupCount != tc.mockUpsertGroupCfg.expectedCount {
				t.Errorf("expected %d calls to UpsertGroup, got %d",
					tc.mockUpsertGroupCfg.expectedCount, mockClient.upsertGroupCount)
			}

			if mockClient.upsertGroupDescendantInfoCount != tc.mockUpsertGroupDescendantCfg.expectedCount {
				t.Errorf("expected %d calls to UpsertGroupDescendantInfo, got %d",
					tc.mockUpsertGroupDescendantCfg.expectedCount, mockClient.upsertGroupDescendantInfoCount)
			}

			if mockClient.upsertWebFeatureGroupCount != tc.mockUpsertWebFeatureGroupCfg.expectedCount {
				t.Errorf("expected %d calls to UpsertWebFeatureGroup, got %d",
					tc.mockUpsertWebFeatureGroupCfg.expectedCount, mockClient.upsertWebFeatureGroupCount)
			}
		})
	}
}

type mockWebFeatureGroupsClient struct {
	t *testing.T

	mockUpsertGroupCfg           mockUpsertGroupConfig
	mockUpsertGroupDescendantCfg mockUpsertGroupDescendantInfoConfig
	mockUpsertWebFeatureGroupCfg mockUpsertWebFeatureGroupConfig

	upsertGroupCount               int
	upsertGroupDescendantInfoCount int
	upsertWebFeatureGroupCount     int
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

func (c *mockWebFeatureGroupsClient) UpsertGroupDescendantInfo(
	_ context.Context, groupKey string, descendantInfo gcpspanner.GroupDescendantInfo) error {
	if len(c.mockUpsertGroupDescendantCfg.expectedInputs) <= c.upsertGroupDescendantInfoCount {
		c.t.Fatal("no more expected input for UpsertGroupDescendantInfo")
	}
	if len(c.mockUpsertGroupDescendantCfg.outputs) <= c.upsertGroupDescendantInfoCount {
		c.t.Fatal("no more configured outputs for UpsertGroupDescendantInfo")
	}

	expectedInput, found := c.mockUpsertGroupDescendantCfg.expectedInputs[groupKey]
	if !found {
		c.t.Errorf("unexpected input %v for groupKey %s", descendantInfo, groupKey)
	}
	if !reflect.DeepEqual(expectedInput, descendantInfo) {
		c.t.Errorf("unexpected input for groupKey %s expected %v received %v", groupKey, expectedInput, descendantInfo)
	}
	c.upsertGroupDescendantInfoCount++

	return c.mockUpsertGroupDescendantCfg.outputs[groupKey]
}

func (c *mockWebFeatureGroupsClient) UpsertWebFeatureGroup(_ context.Context, group gcpspanner.WebFeatureGroup) error {
	if len(c.mockUpsertWebFeatureGroupCfg.expectedInputs) <= c.upsertWebFeatureGroupCount {
		c.t.Fatal("no more expected input for UpsertWebFeatureGroup")
	}
	if len(c.mockUpsertWebFeatureGroupCfg.outputs) <= c.upsertWebFeatureGroupCount {
		c.t.Fatal("no more configured outputs for UpsertWebFeatureGroup")
	}

	expectedInput, found := c.mockUpsertWebFeatureGroupCfg.expectedInputs[group.WebFeatureID]
	if !found {
		c.t.Errorf("unexpected input %v", group)
	}
	if !reflect.DeepEqual(expectedInput, group) {
		c.t.Errorf("unexpected input expected %v received %v", expectedInput, group)
	}
	c.upsertWebFeatureGroupCount++

	return c.mockUpsertWebFeatureGroupCfg.outputs[group.WebFeatureID]
}

func newMockWebFeatureGroupsClient(t *testing.T,
	upsertGroupCfg mockUpsertGroupConfig,
	upsertGroupDescendantCfg mockUpsertGroupDescendantInfoConfig,
	upsertWebFeatureGroupCfg mockUpsertWebFeatureGroupConfig) *mockWebFeatureGroupsClient {

	return &mockWebFeatureGroupsClient{
		t:                              t,
		mockUpsertGroupCfg:             upsertGroupCfg,
		mockUpsertGroupDescendantCfg:   upsertGroupDescendantCfg,
		mockUpsertWebFeatureGroupCfg:   upsertWebFeatureGroupCfg,
		upsertGroupCount:               0,
		upsertGroupDescendantInfoCount: 0,
		upsertWebFeatureGroupCount:     0,
	}
}
