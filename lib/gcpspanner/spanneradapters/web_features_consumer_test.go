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

package spanneradapters

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
)

func TestConvertStringToDate(t *testing.T) {
	testCases := []struct {
		name     string
		input    *string
		expected *time.Time
	}{
		{
			name:     "valid date",
			input:    valuePtr("2024-03-22"),
			expected: valuePtr(time.Date(2024, time.March, 22, 0, 0, 0, 0, time.UTC)),
		},
		{name: "invalid date", input: valuePtr("invalid"), expected: nil},
		{name: "nil input", input: nil, expected: nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertStringToDate(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("different times. expected %v received %v", tc.expected, result)
			}
		})
	}
}

func TestGetBaselineStatusEnum(t *testing.T) {
	testCases := []struct {
		name     string
		input    *web_platform_dx__web_features.Status
		expected *gcpspanner.BaselineStatus
	}{
		{
			name:     "undefined status",
			input:    nil,
			expected: nil,
		},
		{
			name: "undefined baseline",
			input: &web_platform_dx__web_features.Status{
				BaselineHighDate: nil,
				BaselineLowDate:  nil,
				Support:          nil,
				Baseline:         nil,
			},
			expected: nil,
		},
		{
			name: "enum: High",
			input: &web_platform_dx__web_features.Status{
				BaselineHighDate: nil,
				BaselineLowDate:  nil,
				Support:          nil,
				Baseline: &web_platform_dx__web_features.BaselineUnion{
					Enum: valuePtr(web_platform_dx__web_features.High),
					Bool: nil,
				},
			},
			expected: valuePtr(gcpspanner.BaselineStatusHigh),
		},
		{
			name: "enum: Low",
			input: &web_platform_dx__web_features.Status{
				BaselineHighDate: nil,
				BaselineLowDate:  nil,
				Support:          nil,
				Baseline: &web_platform_dx__web_features.BaselineUnion{
					Enum: valuePtr(web_platform_dx__web_features.Low),
					Bool: nil,
				},
			},
			expected: valuePtr(gcpspanner.BaselineStatusLow),
		},
		{
			name: "bool: False",
			input: &web_platform_dx__web_features.Status{
				BaselineHighDate: nil,
				BaselineLowDate:  nil,
				Support:          nil,
				Baseline: &web_platform_dx__web_features.BaselineUnion{
					Bool: valuePtr(false),
					Enum: nil,
				},
			},
			expected: valuePtr(gcpspanner.BaselineStatusNone),
		},
		{
			name: "bool: True (should never happen)",
			input: &web_platform_dx__web_features.Status{
				BaselineHighDate: nil,
				BaselineLowDate:  nil,
				Support:          nil,
				Baseline: &web_platform_dx__web_features.BaselineUnion{
					Bool: valuePtr(true),
					Enum: nil,
				},
			},
			expected: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := getBaselineStatusEnum(tc.input)
			if !reflect.DeepEqual(tc.expected, output) {
				t.Errorf("unexpected output enum expected %v received %v", tc.expected, output)
			}
		})
	}
}

type mockUpsertWebFeatureConfig struct {
	expectedInputs map[string]gcpspanner.WebFeature
	outputIDs      map[string]*string
	outputs        map[string]error
	expectedCount  int
}

type mockUpsertFeatureBaselineStatusConfig struct {
	expectedInputs map[string]gcpspanner.FeatureBaselineStatus
	outputs        map[string]error
	expectedCount  int
}

type mockInsertBrowserFeatureAvailabilityConfig struct {
	expectedInputs          map[string][]gcpspanner.BrowserFeatureAvailability
	outputs                 map[string][]error
	expectedCountPerFeature map[string]int
}

type mockUpsertFeatureSpecConfig struct {
	expectedInputs map[string]gcpspanner.FeatureSpec
	outputs        map[string]error
	expectedCount  int
}

type mockWebFeatureSpannerClient struct {
	t                                               *testing.T
	upsertWebFeatureCount                           int
	mockUpsertWebFeatureCfg                         mockUpsertWebFeatureConfig
	upsertFeatureBaselineStatusCount                int
	mockUpsertFeatureBaselineStatusCfg              mockUpsertFeatureBaselineStatusConfig
	insertBrowserFeatureAvailabilityCountPerFeature map[string]int
	mockInsertBrowserFeatureAvailabilityCfg         mockInsertBrowserFeatureAvailabilityConfig
	mockUpsertFeatureSpecCfg                        mockUpsertFeatureSpecConfig
	upsertFeatureSpecCount                          int
}

func (c *mockWebFeatureSpannerClient) UpsertWebFeature(
	_ context.Context, feature gcpspanner.WebFeature) (*string, error) {
	if len(c.mockUpsertWebFeatureCfg.expectedInputs) <= c.upsertWebFeatureCount {
		c.t.Fatal("no more expected input for UpsertWebFeature")
	}
	if len(c.mockUpsertWebFeatureCfg.outputs) <= c.upsertWebFeatureCount {
		c.t.Fatal("no more configured outputs for UpsertWebFeature")
	}
	expectedInput, found := c.mockUpsertWebFeatureCfg.expectedInputs[feature.FeatureKey]
	if !found {
		c.t.Errorf("unexpected input %v", feature)
	}
	if !reflect.DeepEqual(expectedInput, feature) {
		c.t.Errorf("unexpected input expected %s received %s", expectedInput, feature)
	}
	c.upsertWebFeatureCount++

	return c.mockUpsertWebFeatureCfg.outputIDs[feature.FeatureKey], c.mockUpsertWebFeatureCfg.outputs[feature.FeatureKey]
}

func (c *mockWebFeatureSpannerClient) UpsertFeatureBaselineStatus(
	_ context.Context, featureID string, status gcpspanner.FeatureBaselineStatus) error {
	if len(c.mockUpsertFeatureBaselineStatusCfg.expectedInputs) <= c.upsertFeatureBaselineStatusCount {
		c.t.Fatal("no more expected input for UpsertFeatureBaselineStatus")
	}
	if len(c.mockUpsertFeatureBaselineStatusCfg.outputs) <= c.upsertFeatureBaselineStatusCount {
		c.t.Fatal("no more configured outputs for UpsertFeatureBaselineStatus")
	}
	expectedInput, found := c.mockUpsertFeatureBaselineStatusCfg.expectedInputs[featureID]
	if !found {
		c.t.Errorf("unexpected input %v", status)
	}
	if !reflect.DeepEqual(expectedInput, status) {
		c.t.Errorf("unexpected input expected %v received %v", expectedInput, status)
	}
	c.upsertFeatureBaselineStatusCount++

	return c.mockUpsertFeatureBaselineStatusCfg.outputs[featureID]
}

func (c *mockWebFeatureSpannerClient) UpsertFeatureSpec(
	_ context.Context, featureID string, spec gcpspanner.FeatureSpec) error {
	if len(c.mockUpsertFeatureSpecCfg.expectedInputs) <= c.upsertFeatureSpecCount {
		c.t.Fatal("no more expected input for UpsertFeatureSpec")
	}
	if len(c.mockUpsertFeatureSpecCfg.outputs) <= c.upsertFeatureSpecCount {
		c.t.Fatal("no more configured outputs for UpsertFeatureSpec")
	}
	expectedInput, found := c.mockUpsertFeatureSpecCfg.expectedInputs[featureID]
	if !found {
		c.t.Errorf("unexpected input %v", spec)
	}
	if !reflect.DeepEqual(expectedInput, spec) {
		c.t.Errorf("unexpected input expected %v received %v", expectedInput, spec)
	}
	c.upsertFeatureSpecCount++

	return c.mockUpsertFeatureSpecCfg.outputs[featureID]
}

func (c *mockWebFeatureSpannerClient) InsertBrowserFeatureAvailability(
	_ context.Context, featureID string, featureAvailability gcpspanner.BrowserFeatureAvailability) error {
	expectedCountForFeature := c.insertBrowserFeatureAvailabilityCountPerFeature[featureID]
	if len(c.mockInsertBrowserFeatureAvailabilityCfg.expectedInputs[featureID]) <=
		expectedCountForFeature {
		c.t.Fatal("no more expected input for InsertBrowserFeatureAvailability")
	}
	if len(c.mockInsertBrowserFeatureAvailabilityCfg.outputs[featureID]) <=
		expectedCountForFeature {
		c.t.Fatal("no more configured outputs for InsertBrowserFeatureAvailability")
	}

	idx := expectedCountForFeature

	expectedInputs, found := c.mockInsertBrowserFeatureAvailabilityCfg.expectedInputs[featureID]
	if !found {
		c.t.Errorf("unexpected input %v", featureAvailability)
	}

	expectedInput := expectedInputs[idx]

	if !reflect.DeepEqual(expectedInput, featureAvailability) {
		c.t.Errorf("unexpected input expected %s received %s", expectedInput, featureAvailability)
	}
	c.insertBrowserFeatureAvailabilityCountPerFeature[featureID]++

	return c.mockInsertBrowserFeatureAvailabilityCfg.outputs[featureID][idx]
}

func newMockmockWebFeatureSpannerClient(
	t *testing.T,
	mockUpsertWebFeatureCfg mockUpsertWebFeatureConfig,
	mockUpsertFeatureBaselineStatusCfg mockUpsertFeatureBaselineStatusConfig,
	mockInsertBrowserFeatureAvailabilityCfg mockInsertBrowserFeatureAvailabilityConfig,
	mockUpsertFeatureSpecCfg mockUpsertFeatureSpecConfig,
) *mockWebFeatureSpannerClient {
	return &mockWebFeatureSpannerClient{
		t:                                       t,
		mockUpsertWebFeatureCfg:                 mockUpsertWebFeatureCfg,
		mockUpsertFeatureBaselineStatusCfg:      mockUpsertFeatureBaselineStatusCfg,
		mockInsertBrowserFeatureAvailabilityCfg: mockInsertBrowserFeatureAvailabilityCfg,
		mockUpsertFeatureSpecCfg:                mockUpsertFeatureSpecCfg,
		upsertWebFeatureCount:                   0,
		upsertFeatureBaselineStatusCount:        0,
		upsertFeatureSpecCount:                  0,
		insertBrowserFeatureAvailabilityCountPerFeature: map[string]int{},
	}
}

var ErrWebFeatureTest = errors.New("web feature test error")
var ErrBaselineStatusTest = errors.New("baseline status test error")
var ErrBrowserFeatureAvailabilityTest = errors.New("browser feature availability test error")
var ErrFeatureSpecTest = errors.New("feature spec test error")

func TestInsertWebFeatures(t *testing.T) {
	testCases := []struct {
		name                                    string
		mockUpsertWebFeatureCfg                 mockUpsertWebFeatureConfig
		mockUpsertFeatureBaselineStatusCfg      mockUpsertFeatureBaselineStatusConfig
		mockInsertBrowserFeatureAvailabilityCfg mockInsertBrowserFeatureAvailabilityConfig
		mockUpsertFeatureSpecCfg                mockUpsertFeatureSpecConfig
		input                                   map[string]web_platform_dx__web_features.FeatureData
		expectedError                           error // Expected error from InsertWebFeatures
	}{
		{
			name: "success",
			mockUpsertWebFeatureCfg: mockUpsertWebFeatureConfig{
				expectedInputs: map[string]gcpspanner.WebFeature{
					"feature1": {
						FeatureKey: "feature1",
						Name:       "Feature 1",
					},
					"feature2": {
						FeatureKey: "feature2",
						Name:       "Feature 2",
					},
				},
				outputIDs: map[string]*string{
					"feature1": valuePtr("id-1"),
					"feature2": valuePtr("id-2"),
				},
				outputs: map[string]error{
					"feature1": nil,
					"feature2": nil,
				},
				expectedCount: 2,
			},
			mockUpsertFeatureBaselineStatusCfg: mockUpsertFeatureBaselineStatusConfig{
				expectedInputs: map[string]gcpspanner.FeatureBaselineStatus{
					"feature1": {
						Status:   valuePtr(gcpspanner.BaselineStatusHigh),
						HighDate: nil,
						LowDate:  nil,
					},
					"feature2": {
						Status:   valuePtr(gcpspanner.BaselineStatusLow),
						HighDate: nil,
						LowDate:  nil,
					},
				},
				outputs: map[string]error{
					"feature1": nil,
					"feature2": nil,
				},
				expectedCount: 2,
			},
			mockInsertBrowserFeatureAvailabilityCfg: mockInsertBrowserFeatureAvailabilityConfig{
				expectedInputs: map[string][]gcpspanner.BrowserFeatureAvailability{
					"feature1": {
						{
							BrowserName:    "chrome",
							BrowserVersion: "100",
						},
						{
							BrowserName:    "edge",
							BrowserVersion: "101",
						},
						{
							BrowserName:    "firefox",
							BrowserVersion: "102",
						},
						{
							BrowserName:    "safari",
							BrowserVersion: "103",
						},
					},
					"feature2": {
						{
							BrowserName:    "firefox",
							BrowserVersion: "202",
						},
						{
							BrowserName:    "safari",
							BrowserVersion: "203",
						},
					},
				},
				outputs: map[string][]error{
					"feature1": {nil, nil, nil, nil},
					"feature2": {nil, nil},
				},
				expectedCountPerFeature: map[string]int{
					"feature1": 4,
					"feature2": 2,
				},
			},
			mockUpsertFeatureSpecCfg: mockUpsertFeatureSpecConfig{
				expectedInputs: map[string]gcpspanner.FeatureSpec{
					"feature1": {
						Links: []string{
							"feature1-link1",
							"feature1-link2",
						},
					},
					"feature2": {
						Links: []string{
							"feature2-link",
						},
					},
				},
				outputs: map[string]error{
					"feature1": nil,
					"feature2": nil,
				},
				expectedCount: 2,
			},
			input: map[string]web_platform_dx__web_features.FeatureData{
				"feature1": {
					Name:           "Feature 1",
					Alias:          nil,
					Caniuse:        nil,
					CompatFeatures: nil,
					Spec: &web_platform_dx__web_features.Alias{
						StringArray: []string{"feature1-link1", "feature1-link2"},
						String:      nil,
					},
					Status: &web_platform_dx__web_features.Status{
						BaselineHighDate: nil,
						BaselineLowDate:  nil,
						Support: &web_platform_dx__web_features.Support{
							Chrome:         valuePtr("100"),
							ChromeAndroid:  nil,
							Edge:           valuePtr("101"),
							Firefox:        valuePtr("102"),
							FirefoxAndroid: nil,
							Safari:         valuePtr("103"),
							SafariIos:      nil,
						},
						Baseline: &web_platform_dx__web_features.BaselineUnion{
							Enum: valuePtr(web_platform_dx__web_features.High),
							Bool: nil,
						},
					},
					Description:     "text",
					DescriptionHTML: "<html>",
					UsageStats:      nil,
				},
				"feature2": {
					Name:           "Feature 2",
					Alias:          nil,
					Caniuse:        nil,
					CompatFeatures: nil,
					Spec: &web_platform_dx__web_features.Alias{
						StringArray: nil,
						String:      valuePtr("feature2-link"),
					},
					Status: &web_platform_dx__web_features.Status{
						BaselineHighDate: nil,
						BaselineLowDate:  nil,
						Support: &web_platform_dx__web_features.Support{
							Chrome:         nil,
							ChromeAndroid:  nil,
							Edge:           nil,
							Firefox:        valuePtr("202"),
							FirefoxAndroid: nil,
							Safari:         valuePtr("203"),
							SafariIos:      nil,
						},
						Baseline: &web_platform_dx__web_features.BaselineUnion{
							Enum: valuePtr(web_platform_dx__web_features.Low),
							Bool: nil,
						},
					},
					Description:     "text",
					DescriptionHTML: "<html>",
					UsageStats:      nil,
				},
			},
			expectedError: nil,
		},
		{
			name: "UpsertWebFeature error",
			mockUpsertWebFeatureCfg: mockUpsertWebFeatureConfig{
				expectedInputs: map[string]gcpspanner.WebFeature{
					"feature1": {
						FeatureKey: "feature1",
						Name:       "Feature 1",
					},
				},
				outputs: map[string]error{
					"feature1": ErrWebFeatureTest,
				},
				outputIDs: map[string]*string{
					"feature1": valuePtr("id-1"),
				},
				expectedCount: 1,
			},
			mockUpsertFeatureBaselineStatusCfg: mockUpsertFeatureBaselineStatusConfig{
				expectedInputs: nil,
				outputs:        nil,
				expectedCount:  0,
			},
			mockInsertBrowserFeatureAvailabilityCfg: mockInsertBrowserFeatureAvailabilityConfig{
				expectedInputs:          map[string][]gcpspanner.BrowserFeatureAvailability{},
				outputs:                 map[string][]error{},
				expectedCountPerFeature: map[string]int{},
			},
			mockUpsertFeatureSpecCfg: mockUpsertFeatureSpecConfig{
				expectedInputs: map[string]gcpspanner.FeatureSpec{},
				outputs:        map[string]error{},
				expectedCount:  0,
			},
			input: map[string]web_platform_dx__web_features.FeatureData{
				"feature1": {
					Name:           "Feature 1",
					Alias:          nil,
					Caniuse:        nil,
					CompatFeatures: nil,
					Spec:           nil,
					Status: &web_platform_dx__web_features.Status{
						BaselineHighDate: nil,
						BaselineLowDate:  nil,
						Support:          nil,
						Baseline: &web_platform_dx__web_features.BaselineUnion{
							Enum: valuePtr(web_platform_dx__web_features.High),
							Bool: nil,
						},
					},
					Description:     "text",
					DescriptionHTML: "<html>",
					UsageStats:      nil,
				},
			},
			expectedError: ErrWebFeatureTest,
		},
		{
			name: "UpsertFeatureBaselineStatus error",
			mockUpsertWebFeatureCfg: mockUpsertWebFeatureConfig{
				expectedInputs: map[string]gcpspanner.WebFeature{
					"feature1": {
						FeatureKey: "feature1",
						Name:       "Feature 1",
					},
				},
				outputs: map[string]error{
					"feature1": nil,
				},
				outputIDs: map[string]*string{
					"feature1": valuePtr("id-1"),
				},
				expectedCount: 1,
			},
			mockUpsertFeatureBaselineStatusCfg: mockUpsertFeatureBaselineStatusConfig{
				expectedInputs: map[string]gcpspanner.FeatureBaselineStatus{
					"feature1": {
						Status:   valuePtr(gcpspanner.BaselineStatusHigh),
						HighDate: nil,
						LowDate:  nil,
					},
				},
				outputs: map[string]error{
					"feature1": ErrBaselineStatusTest,
				},
				expectedCount: 1,
			},
			mockInsertBrowserFeatureAvailabilityCfg: mockInsertBrowserFeatureAvailabilityConfig{
				expectedInputs:          map[string][]gcpspanner.BrowserFeatureAvailability{},
				outputs:                 map[string][]error{},
				expectedCountPerFeature: map[string]int{},
			},
			mockUpsertFeatureSpecCfg: mockUpsertFeatureSpecConfig{
				expectedInputs: map[string]gcpspanner.FeatureSpec{},
				outputs:        map[string]error{},
				expectedCount:  0,
			},
			input: map[string]web_platform_dx__web_features.FeatureData{
				"feature1": {
					Name:           "Feature 1",
					Alias:          nil,
					Caniuse:        nil,
					CompatFeatures: nil,
					Spec:           nil,
					Status: &web_platform_dx__web_features.Status{
						BaselineHighDate: nil,
						BaselineLowDate:  nil,
						Support:          nil,
						Baseline: &web_platform_dx__web_features.BaselineUnion{
							Enum: valuePtr(web_platform_dx__web_features.High),
							Bool: nil,
						},
					},
					Description:     "text",
					DescriptionHTML: "<html>",
					UsageStats:      nil,
				},
			},
			expectedError: ErrBaselineStatusTest,
		},
		{
			name: "InsertBrowserFeatureAvailability error",
			mockUpsertWebFeatureCfg: mockUpsertWebFeatureConfig{
				expectedInputs: map[string]gcpspanner.WebFeature{
					"feature1": {
						FeatureKey: "feature1",
						Name:       "Feature 1",
					},
				},
				outputs: map[string]error{
					"feature1": nil,
				},
				outputIDs: map[string]*string{
					"feature1": valuePtr("id-1"),
				},
				expectedCount: 1,
			},
			mockUpsertFeatureBaselineStatusCfg: mockUpsertFeatureBaselineStatusConfig{
				expectedInputs: map[string]gcpspanner.FeatureBaselineStatus{
					"feature1": {
						Status:   valuePtr(gcpspanner.BaselineStatusHigh),
						HighDate: nil,
						LowDate:  nil,
					},
				},
				outputs: map[string]error{
					"feature1": nil,
				},
				expectedCount: 1,
			},
			mockInsertBrowserFeatureAvailabilityCfg: mockInsertBrowserFeatureAvailabilityConfig{
				expectedInputs: map[string][]gcpspanner.BrowserFeatureAvailability{
					"feature1": {
						{
							BrowserName:    "chrome",
							BrowserVersion: "100",
						},
					},
				},
				outputs: map[string][]error{
					"feature1": {ErrBrowserFeatureAvailabilityTest},
				},
				expectedCountPerFeature: map[string]int{
					"feature1": 1,
				},
			},
			mockUpsertFeatureSpecCfg: mockUpsertFeatureSpecConfig{
				expectedInputs: map[string]gcpspanner.FeatureSpec{},
				outputs:        map[string]error{},
				expectedCount:  0,
			},
			input: map[string]web_platform_dx__web_features.FeatureData{
				"feature1": {
					Name:           "Feature 1",
					Alias:          nil,
					Caniuse:        nil,
					CompatFeatures: nil,
					Spec:           nil,
					Status: &web_platform_dx__web_features.Status{
						BaselineHighDate: nil,
						BaselineLowDate:  nil,
						Support: &web_platform_dx__web_features.Support{
							Chrome:         valuePtr("100"),
							ChromeAndroid:  nil,
							Edge:           valuePtr("101"),
							Firefox:        valuePtr("102"),
							FirefoxAndroid: nil,
							Safari:         valuePtr("103"),
							SafariIos:      nil,
						},
						Baseline: &web_platform_dx__web_features.BaselineUnion{
							Enum: valuePtr(web_platform_dx__web_features.High),
							Bool: nil,
						},
					},
					Description:     "text",
					DescriptionHTML: "<html>",
					UsageStats:      nil,
				},
			},
			expectedError: ErrBrowserFeatureAvailabilityTest,
		},
		{
			name: "upsert feature spec failure",
			mockUpsertWebFeatureCfg: mockUpsertWebFeatureConfig{
				expectedInputs: map[string]gcpspanner.WebFeature{
					"feature1": {
						FeatureKey: "feature1",
						Name:       "Feature 1",
					},
				},
				outputs: map[string]error{
					"feature1": nil,
				},
				outputIDs: map[string]*string{
					"feature1": valuePtr("id-1"),
				},
				expectedCount: 1,
			},
			mockUpsertFeatureBaselineStatusCfg: mockUpsertFeatureBaselineStatusConfig{
				expectedInputs: map[string]gcpspanner.FeatureBaselineStatus{
					"feature1": {
						Status:   valuePtr(gcpspanner.BaselineStatusHigh),
						HighDate: nil,
						LowDate:  nil,
					},
				},
				outputs: map[string]error{
					"feature1": nil,
				},
				expectedCount: 1,
			},
			mockInsertBrowserFeatureAvailabilityCfg: mockInsertBrowserFeatureAvailabilityConfig{
				expectedInputs: map[string][]gcpspanner.BrowserFeatureAvailability{
					"feature1": {
						{
							BrowserName:    "chrome",
							BrowserVersion: "100",
						},
						{
							BrowserName:    "edge",
							BrowserVersion: "101",
						},
						{
							BrowserName:    "firefox",
							BrowserVersion: "102",
						},
						{
							BrowserName:    "safari",
							BrowserVersion: "103",
						},
					},
				},
				outputs: map[string][]error{
					"feature1": {nil, nil, nil, nil},
				},
				expectedCountPerFeature: map[string]int{
					"feature1": 4,
				},
			},
			mockUpsertFeatureSpecCfg: mockUpsertFeatureSpecConfig{
				expectedInputs: map[string]gcpspanner.FeatureSpec{
					"feature1": {
						Links: []string{
							"feature1-link1",
							"feature1-link2",
						},
					},
				},
				outputs: map[string]error{
					"feature1": ErrFeatureSpecTest,
				},
				expectedCount: 1,
			},
			input: map[string]web_platform_dx__web_features.FeatureData{
				"feature1": {
					Name:           "Feature 1",
					Alias:          nil,
					Caniuse:        nil,
					CompatFeatures: nil,
					Spec: &web_platform_dx__web_features.Alias{
						StringArray: []string{"feature1-link1", "feature1-link2"},
						String:      nil,
					},
					Status: &web_platform_dx__web_features.Status{
						BaselineHighDate: nil,
						BaselineLowDate:  nil,
						Support: &web_platform_dx__web_features.Support{
							Chrome:         valuePtr("100"),
							ChromeAndroid:  nil,
							Edge:           valuePtr("101"),
							Firefox:        valuePtr("102"),
							FirefoxAndroid: nil,
							Safari:         valuePtr("103"),
							SafariIos:      nil,
						},
						Baseline: &web_platform_dx__web_features.BaselineUnion{
							Enum: valuePtr(web_platform_dx__web_features.High),
							Bool: nil,
						},
					},
					Description:     "text",
					DescriptionHTML: "<html>",
					UsageStats:      nil,
				},
			},
			expectedError: ErrFeatureSpecTest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := newMockmockWebFeatureSpannerClient(
				t,
				tc.mockUpsertWebFeatureCfg,
				tc.mockUpsertFeatureBaselineStatusCfg,
				tc.mockInsertBrowserFeatureAvailabilityCfg,
				tc.mockUpsertFeatureSpecCfg,
			)
			consumer := NewWebFeaturesConsumer(mockClient)

			_, err := consumer.InsertWebFeatures(context.TODO(), tc.input)

			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error: got %v, want %v", err, tc.expectedError)
			}

			if mockClient.upsertWebFeatureCount != mockClient.mockUpsertWebFeatureCfg.expectedCount {
				t.Errorf("expected %d calls to UpsertWebFeature, got %d",
					mockClient.mockUpsertWebFeatureCfg.expectedCount,
					mockClient.upsertWebFeatureCount)
			}

			if mockClient.upsertFeatureBaselineStatusCount !=
				mockClient.mockUpsertFeatureBaselineStatusCfg.expectedCount {
				t.Errorf("expected %d calls to UpsertFeatureBaselineStatus, got %d",
					mockClient.mockUpsertFeatureBaselineStatusCfg.expectedCount,
					mockClient.upsertFeatureBaselineStatusCount)
			}

			if mockClient.upsertFeatureSpecCount !=
				mockClient.mockUpsertFeatureSpecCfg.expectedCount {
				t.Errorf("expected %d calls to UpsertFeatureSpec, got %d",
					mockClient.mockUpsertFeatureSpecCfg.expectedCount,
					mockClient.upsertFeatureSpecCount)
			}

			if !reflect.DeepEqual(mockClient.insertBrowserFeatureAvailabilityCountPerFeature,
				tc.mockInsertBrowserFeatureAvailabilityCfg.expectedCountPerFeature) {
				t.Errorf("Unexpected call counts for InsertBrowserFeatureAvailability. Expected: %v, Got: %v",
					tc.mockInsertBrowserFeatureAvailabilityCfg.expectedCountPerFeature,
					mockClient.insertBrowserFeatureAvailabilityCountPerFeature)
			}
		})
	}
}
