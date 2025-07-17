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

type mockUpsertGroupConfig struct {
	expectedInputs map[string]gcpspanner.Group
	outputIDs      map[string]string
	outputs        map[string]error
	expectedCount  int
}

type mockUpsertFeatureGroupLookupInput struct {
	featureKeyToGroupsMapping     map[string][]string
	childGroupKeyToParentGroupKey map[string]string
}

type mockUpsertFeatureGroupLookupsConfig struct {
	expectedInput mockUpsertFeatureGroupLookupInput
	output        error
	expectedCount int
}

func TestInsertWebFeatureGroups(t *testing.T) {
	testCases := []struct {
		name                             string
		mockUpsertGroupCfg               mockUpsertGroupConfig
		mockUpsertFeatureGroupLookupsCfg mockUpsertFeatureGroupLookupsConfig
		featureKeyToID                   map[string]string
		featureData                      webdxfeaturetypes.FeatureKinds
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
				expectedInput: mockUpsertFeatureGroupLookupInput{
					featureKeyToGroupsMapping: map[string][]string{
						"feature1": {"group1", "group2"},
						"feature2": {"group2"},
					},
					childGroupKeyToParentGroupKey: map[string]string{
						"child3": "group1",
					},
				},
				output:        nil,
				expectedCount: 1,
			},
			featureKeyToID: map[string]string{
				"feature1": "featureID1",
				"feature2": "featureID2",
			},
			featureData: webdxfeaturetypes.FeatureKinds{
				"feature1": {
					Group: &web_platform_dx__web_features.StringOrStrings{
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
					Status: web_platform_dx__web_features.StatusHeadlineClass{
						Baseline:         nil,
						BaselineHighDate: nil,
						BaselineLowDate:  nil,
						ByCompatKey:      nil,
						Support: web_platform_dx__web_features.ByCompatKeySupport{
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
					Group: &web_platform_dx__web_features.StringOrStrings{
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
					Status: web_platform_dx__web_features.StatusHeadlineClass{
						Baseline:         nil,
						BaselineHighDate: nil,
						BaselineLowDate:  nil,
						ByCompatKey:      nil,
						Support: web_platform_dx__web_features.ByCompatKeySupport{
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

			err := consumer.InsertWebFeatureGroups(context.TODO(), tc.featureData, tc.groupData)

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
	_ context.Context, featureKeyToGroupsMapping map[string][]string,
	childGroupKeyToParentGroupKey map[string]string) error {
	expectedInput := c.mockUpsertFeatureGroupLookupsCfg.expectedInput
	if !reflect.DeepEqual(expectedInput.featureKeyToGroupsMapping, featureKeyToGroupsMapping) ||
		!reflect.DeepEqual(expectedInput.childGroupKeyToParentGroupKey, childGroupKeyToParentGroupKey) {
		c.t.Errorf("unexpected input for UpsertFeatureGroupLookups\nexpected (%v %v)\nreceived (%v %v)",
			expectedInput.featureKeyToGroupsMapping,
			expectedInput.childGroupKeyToParentGroupKey,
			featureKeyToGroupsMapping,
			childGroupKeyToParentGroupKey)
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
