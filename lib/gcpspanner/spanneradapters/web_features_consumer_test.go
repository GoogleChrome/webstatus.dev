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
		expected gcpspanner.BaselineStatus
	}{
		{
			name:     "undefined status",
			input:    nil,
			expected: gcpspanner.BaselineStatusUndefined,
		},
		{
			name: "undefined baseline",
			input: &web_platform_dx__web_features.Status{
				BaselineHighDate: nil,
				BaselineLowDate:  nil,
				Support:          nil,
				Baseline:         nil,
			},
			expected: gcpspanner.BaselineStatusUndefined,
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
			expected: gcpspanner.BaselineStatusHigh,
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
			expected: gcpspanner.BaselineStatusLow,
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
			expected: gcpspanner.BaselineStatusNone,
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
			expected: gcpspanner.BaselineStatusUndefined,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := getBaselineStatusEnum(tc.input)
			if tc.expected != output {
				t.Errorf("unexpected output enum expected %v received %v", tc.expected, output)
			}
		})
	}
}

type mockUpsertWebFeatureConfig struct {
	expectedInputs map[string]gcpspanner.WebFeature
	outputs        map[string]error
	expectedCount  int
}

type mockUpsertFeatureBaselineStatusConfig struct {
	expectedInputs map[string]gcpspanner.FeatureBaselineStatus
	outputs        map[string]error
	expectedCount  int
}

type mockWebFeatureSpannerClient struct {
	t                                  *testing.T
	upsertWebFeatureCount              int
	mockUpsertWebFeatureCfg            mockUpsertWebFeatureConfig
	upsertFeatureBaselineStatusCount   int
	mockUpsertFeatureBaselineStatusCfg mockUpsertFeatureBaselineStatusConfig
}

func (c *mockWebFeatureSpannerClient) UpsertWebFeature(
	_ context.Context, feature gcpspanner.WebFeature) error {
	if len(c.mockUpsertWebFeatureCfg.expectedInputs) <= c.upsertWebFeatureCount {
		c.t.Fatal("no more expected input for UpsertWebFeature")
	}
	if len(c.mockUpsertWebFeatureCfg.outputs) <= c.upsertWebFeatureCount {
		c.t.Fatal("no more configured outputs for UpsertWebFeature")
	}
	expectedInput, found := c.mockUpsertWebFeatureCfg.expectedInputs[feature.FeatureID]
	if !found {
		c.t.Errorf("unexpected input %v", feature)
	}
	if !reflect.DeepEqual(expectedInput, feature) {
		c.t.Errorf("unexpected input expected %s received %s", expectedInput, feature)
	}
	c.upsertWebFeatureCount++

	return c.mockUpsertWebFeatureCfg.outputs[feature.FeatureID]
}

func (c *mockWebFeatureSpannerClient) UpsertFeatureBaselineStatus(
	_ context.Context, status gcpspanner.FeatureBaselineStatus) error {
	if len(c.mockUpsertFeatureBaselineStatusCfg.expectedInputs) <= c.upsertFeatureBaselineStatusCount {
		c.t.Fatal("no more expected input for UpsertFeatureBaselineStatus")
	}
	if len(c.mockUpsertFeatureBaselineStatusCfg.outputs) <= c.upsertFeatureBaselineStatusCount {
		c.t.Fatal("no more configured outputs for UpsertFeatureBaselineStatus")
	}
	expectedInput, found := c.mockUpsertFeatureBaselineStatusCfg.expectedInputs[status.FeatureID]
	if !found {
		c.t.Errorf("unexpected input %s", status)
	}
	if !reflect.DeepEqual(expectedInput, status) {
		c.t.Errorf("unexpected input expected %s received %s", expectedInput, status)
	}
	c.upsertFeatureBaselineStatusCount++

	return c.mockUpsertFeatureBaselineStatusCfg.outputs[status.FeatureID]
}

func newMockmockWebFeatureSpannerClient(
	t *testing.T,
	mockUpsertWebFeatureCfg mockUpsertWebFeatureConfig,
	mockUpsertFeatureBaselineStatusCfg mockUpsertFeatureBaselineStatusConfig,
) *mockWebFeatureSpannerClient {
	return &mockWebFeatureSpannerClient{
		t:                                  t,
		mockUpsertWebFeatureCfg:            mockUpsertWebFeatureCfg,
		mockUpsertFeatureBaselineStatusCfg: mockUpsertFeatureBaselineStatusCfg,
		upsertWebFeatureCount:              0,
		upsertFeatureBaselineStatusCount:   0,
	}
}

var ErrWebFeatureTest = errors.New("web feature test error")
var ErrBaselineStatusTest = errors.New("baseline status test error")

func TestInsertWebFeatures(t *testing.T) {
	testCases := []struct {
		name                               string
		mockUpsertWebFeatureCfg            mockUpsertWebFeatureConfig
		mockUpsertFeatureBaselineStatusCfg mockUpsertFeatureBaselineStatusConfig
		input                              map[string]web_platform_dx__web_features.FeatureData
		expectedError                      error // Expected error from InsertWebFeatures
	}{
		{
			name: "success",
			mockUpsertWebFeatureCfg: mockUpsertWebFeatureConfig{
				expectedInputs: map[string]gcpspanner.WebFeature{
					"feature1": {
						FeatureID: "feature1",
						Name:      "Feature 1",
					},
					"feature2": {
						FeatureID: "feature2",
						Name:      "Feature 2",
					},
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
						FeatureID: "feature1",
						Status:    gcpspanner.BaselineStatusHigh,
						HighDate:  nil,
						LowDate:   nil,
					},
					"feature2": {
						FeatureID: "feature2",
						Status:    gcpspanner.BaselineStatusLow,
						HighDate:  nil,
						LowDate:   nil,
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
					UsageStats: nil,
				},
				"feature2": {
					Name:           "Feature 2",
					Alias:          nil,
					Caniuse:        nil,
					CompatFeatures: nil,
					Spec:           nil,
					Status: &web_platform_dx__web_features.Status{
						BaselineHighDate: nil,
						BaselineLowDate:  nil,
						Support:          nil,
						Baseline: &web_platform_dx__web_features.BaselineUnion{
							Enum: valuePtr(web_platform_dx__web_features.Low),
							Bool: nil,
						},
					},
					UsageStats: nil,
				},
			},
			expectedError: nil,
		},
		{
			name: "UpsertWebFeature error",
			mockUpsertWebFeatureCfg: mockUpsertWebFeatureConfig{
				expectedInputs: map[string]gcpspanner.WebFeature{
					"feature1": {
						FeatureID: "feature1",
						Name:      "Feature 1",
					},
				},
				outputs: map[string]error{
					"feature1": ErrWebFeatureTest,
				},
				expectedCount: 1,
			},
			mockUpsertFeatureBaselineStatusCfg: mockUpsertFeatureBaselineStatusConfig{
				expectedInputs: nil,
				outputs:        nil,
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
					UsageStats: nil,
				},
			},
			expectedError: ErrWebFeatureTest,
		},
		{
			name: "UpsertFeatureBaselineStatus error",
			mockUpsertWebFeatureCfg: mockUpsertWebFeatureConfig{
				expectedInputs: map[string]gcpspanner.WebFeature{
					"feature1": {
						FeatureID: "feature1",
						Name:      "Feature 1",
					},
				},
				outputs: map[string]error{
					"feature1": nil,
				},
				expectedCount: 1,
			},
			mockUpsertFeatureBaselineStatusCfg: mockUpsertFeatureBaselineStatusConfig{
				expectedInputs: map[string]gcpspanner.FeatureBaselineStatus{
					"feature1": {
						FeatureID: "feature1",
						Status:    gcpspanner.BaselineStatusHigh,
						HighDate:  nil,
						LowDate:   nil,
					},
				},
				outputs: map[string]error{
					"feature1": ErrBaselineStatusTest,
				},
				expectedCount: 1,
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
					UsageStats: nil,
				},
			},
			expectedError: ErrBaselineStatusTest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := newMockmockWebFeatureSpannerClient(
				t,
				tc.mockUpsertWebFeatureCfg,
				tc.mockUpsertFeatureBaselineStatusCfg,
			)
			consumer := NewWebFeaturesConsumer(mockClient)

			err := consumer.InsertWebFeatures(context.TODO(), tc.input)

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
					mockClient.mockUpsertWebFeatureCfg.expectedCount,
					mockClient.upsertFeatureBaselineStatusCount)
			}
		})
	}
}
