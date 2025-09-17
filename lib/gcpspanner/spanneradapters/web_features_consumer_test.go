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
	"sort"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/webdxfeaturetypes"
	"github.com/google/go-cmp/cmp"
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
		input    webdxfeaturetypes.Status
		expected *gcpspanner.BaselineStatus
	}{
		{
			name: "undefined status",
			input: webdxfeaturetypes.Status{
				Support: webdxfeaturetypes.StatusSupport{
					Chrome:         nil,
					ChromeAndroid:  nil,
					Edge:           nil,
					Firefox:        nil,
					FirefoxAndroid: nil,
					Safari:         nil,
					SafariIos:      nil,
				},
				Baseline:         nil,
				BaselineHighDate: nil,
				BaselineLowDate:  nil,
				ByCompatKey:      nil,
			},
			expected: nil,
		},
		{
			name: "undefined baseline",
			input: webdxfeaturetypes.Status{
				BaselineHighDate: nil,
				BaselineLowDate:  nil,
				ByCompatKey:      nil,
				Support: webdxfeaturetypes.StatusSupport{
					Chrome:         nil,
					ChromeAndroid:  nil,
					Edge:           nil,
					Firefox:        nil,
					FirefoxAndroid: nil,
					Safari:         nil,
					SafariIos:      nil,
				},
				Baseline: nil,
			},
			expected: nil,
		},
		{
			name: "enum: High",
			input: webdxfeaturetypes.Status{
				BaselineHighDate: nil,
				BaselineLowDate:  nil,
				ByCompatKey:      nil,
				Support: webdxfeaturetypes.StatusSupport{
					Chrome:         nil,
					ChromeAndroid:  nil,
					Edge:           nil,
					Firefox:        nil,
					FirefoxAndroid: nil,
					Safari:         nil,
					SafariIos:      nil,
				},
				Baseline: &webdxfeaturetypes.BaselineUnion{
					Enum: valuePtr(webdxfeaturetypes.High),
					Bool: nil,
				},
			},
			expected: valuePtr(gcpspanner.BaselineStatusHigh),
		},
		{
			name: "enum: Low",
			input: webdxfeaturetypes.Status{
				BaselineHighDate: nil,
				BaselineLowDate:  nil,
				ByCompatKey:      nil,
				Support: webdxfeaturetypes.StatusSupport{
					Chrome:         nil,
					ChromeAndroid:  nil,
					Edge:           nil,
					Firefox:        nil,
					FirefoxAndroid: nil,
					Safari:         nil,
					SafariIos:      nil,
				},
				Baseline: &webdxfeaturetypes.BaselineUnion{
					Enum: valuePtr(webdxfeaturetypes.Low),
					Bool: nil,
				},
			},
			expected: valuePtr(gcpspanner.BaselineStatusLow),
		},
		{
			name: "bool: False",
			input: webdxfeaturetypes.Status{
				BaselineHighDate: nil,
				BaselineLowDate:  nil,
				ByCompatKey:      nil,
				Support: webdxfeaturetypes.StatusSupport{
					Chrome:         nil,
					ChromeAndroid:  nil,
					Edge:           nil,
					Firefox:        nil,
					FirefoxAndroid: nil,
					Safari:         nil,
					SafariIos:      nil,
				},
				Baseline: &webdxfeaturetypes.BaselineUnion{
					Bool: valuePtr(false),
					Enum: nil,
				},
			},
			expected: valuePtr(gcpspanner.BaselineStatusNone),
		},
		{
			name: "bool: True (should never happen)",
			input: webdxfeaturetypes.Status{
				BaselineHighDate: nil,
				BaselineLowDate:  nil,
				ByCompatKey:      nil,
				Support: webdxfeaturetypes.StatusSupport{
					Chrome:         nil,
					ChromeAndroid:  nil,
					Edge:           nil,
					Firefox:        nil,
					FirefoxAndroid: nil,
					Safari:         nil,
					SafariIos:      nil,
				},
				Baseline: &webdxfeaturetypes.BaselineUnion{
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

type mockSyncWebFeaturesConfig struct {
	expectedInput   []gcpspanner.WebFeature
	expectedOptions []gcpspanner.SyncWebFeaturesOption
	err             error
	expectedCount   int
}

type mockFetchIDsAndKeysConfig struct {
	output        []gcpspanner.SpannerFeatureIDAndKey
	err           error
	expectedCount int
}

type mockUpsertFeatureBaselineStatusConfig struct {
	expectedInputs map[string]gcpspanner.FeatureBaselineStatus
	outputs        map[string]error
	expectedCount  int
}

type mockUpsertBrowserFeatureAvailabilityConfig struct {
	expectedInputs          map[string][]gcpspanner.BrowserFeatureAvailability
	outputs                 map[string][]error
	expectedCountPerFeature map[string]int
}

type mockUpsertFeatureSpecConfig struct {
	expectedInputs map[string]gcpspanner.FeatureSpec
	outputs        map[string]error
	expectedCount  int
}

type mockPrecalculateBrowserFeatureSupportEventsConfig struct {
	expectedCount int
	err           error
}

type mockUpsertFeatureDiscouragedDetailsConfig struct {
	expectedInputs map[string]gcpspanner.FeatureDiscouragedDetails
	outputs        map[string]error
	expectedCount  int
}

type mockSyncMovedWebFeaturesConfig struct {
	expectedFeatures []gcpspanner.MovedWebFeature
	err              error
	expectedCount    int
}

type mockSyncSplitWebFeaturesConfig struct {
	expectedFeatures []gcpspanner.SplitWebFeature
	err              error
	expectedCount    int
}

type mockWebFeatureSpannerClient struct {
	t                                               *testing.T
	syncWebFeaturesCount                            int
	mockSyncWebFeaturesCfg                          *mockSyncWebFeaturesConfig
	fetchIDsAndKeysCount                            int
	mockFetchIDsAndKeysCfg                          *mockFetchIDsAndKeysConfig
	upsertFeatureBaselineStatusCount                int
	mockUpsertFeatureBaselineStatusCfg              *mockUpsertFeatureBaselineStatusConfig
	insertBrowserFeatureAvailabilityCountPerFeature map[string]int
	mockUpsertBrowserFeatureAvailabilityCfg         *mockUpsertBrowserFeatureAvailabilityConfig
	mockUpsertFeatureSpecCfg                        *mockUpsertFeatureSpecConfig
	upsertFeatureSpecCount                          int
	mockPrecalculateBrowserFeatureSupportEventsCfg  *mockPrecalculateBrowserFeatureSupportEventsConfig
	precalculateBrowserFeatureSupportEventsCount    int
	mockUpsertFeatureDiscouragedDetailsCfg          *mockUpsertFeatureDiscouragedDetailsConfig
	upsertFeatureDiscouragedDetailsCount            int
	mockSyncMovedWebFeaturesCfg                     *mockSyncMovedWebFeaturesConfig
	syncMovedWebFeaturesCount                       int
	mockSyncSplitWebFeaturesCfg                     *mockSyncSplitWebFeaturesConfig
	syncSplitWebFeaturesCount                       int
}

// SyncMovedWebFeatures implements WebFeatureSpannerClient.
func (c *mockWebFeatureSpannerClient) SyncMovedWebFeatures(
	_ context.Context, features []gcpspanner.MovedWebFeature) error {
	c.syncMovedWebFeaturesCount++
	if diff := cmp.Diff(c.mockSyncMovedWebFeaturesCfg.expectedFeatures, features); diff != "" {
		c.t.Errorf("SyncMovedWebFeatures unexpected input (-want +got):\n%s", diff)
	}

	return c.mockSyncMovedWebFeaturesCfg.err
}

// SyncSplitWebFeatures implements WebFeatureSpannerClient.
func (c *mockWebFeatureSpannerClient) SyncSplitWebFeatures(
	_ context.Context, features []gcpspanner.SplitWebFeature) error {
	c.syncSplitWebFeaturesCount++
	if diff := cmp.Diff(c.mockSyncSplitWebFeaturesCfg.expectedFeatures, features); diff != "" {
		c.t.Errorf("SyncSplitWebFeatures unexpected input (-want +got):\n%s", diff)
	}

	return c.mockSyncSplitWebFeaturesCfg.err
}

func (c *mockWebFeatureSpannerClient) SyncWebFeatures(
	_ context.Context, features []gcpspanner.WebFeature, opts ...gcpspanner.SyncWebFeaturesOption) error {
	// Sort both slices for stable comparison
	sort.Slice(features, func(i, j int) bool {
		return features[i].FeatureKey < features[j].FeatureKey
	})
	sort.Slice(c.mockSyncWebFeaturesCfg.expectedInput, func(i, j int) bool {
		return c.mockSyncWebFeaturesCfg.expectedInput[i].FeatureKey < c.mockSyncWebFeaturesCfg.expectedInput[j].FeatureKey
	})

	if diff := cmp.Diff(c.mockSyncWebFeaturesCfg.expectedInput, features); diff != "" {
		c.t.Errorf("SyncWebFeatures unexpected input (-want +got):\n%s", diff)
	}

	if !reflect.DeepEqual(c.mockSyncWebFeaturesCfg.expectedOptions, opts) {
		c.t.Errorf("SyncWebFeatures unexpected options expected %v received %v",
			c.mockSyncWebFeaturesCfg.expectedOptions, opts)
	}

	c.syncWebFeaturesCount++

	return c.mockSyncWebFeaturesCfg.err
}

func (c *mockWebFeatureSpannerClient) FetchAllWebFeatureIDsAndKeys(
	_ context.Context) ([]gcpspanner.SpannerFeatureIDAndKey, error) {
	c.fetchIDsAndKeysCount++

	return c.mockFetchIDsAndKeysCfg.output, c.mockFetchIDsAndKeysCfg.err
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

func (c *mockWebFeatureSpannerClient) UpsertBrowserFeatureAvailability(
	_ context.Context, featureID string, featureAvailability gcpspanner.BrowserFeatureAvailability) error {
	expectedCountForFeature := c.insertBrowserFeatureAvailabilityCountPerFeature[featureID]
	if len(c.mockUpsertBrowserFeatureAvailabilityCfg.expectedInputs[featureID]) <=
		expectedCountForFeature {
		c.t.Fatal("no more expected input for UpsertBrowserFeatureAvailability")
	}
	if len(c.mockUpsertBrowserFeatureAvailabilityCfg.outputs[featureID]) <=
		expectedCountForFeature {
		c.t.Fatal("no more configured outputs for UpsertBrowserFeatureAvailability")
	}

	idx := expectedCountForFeature

	expectedInputs, found := c.mockUpsertBrowserFeatureAvailabilityCfg.expectedInputs[featureID]
	if !found {
		c.t.Errorf("unexpected input %v", featureAvailability)
	}

	expectedInput := expectedInputs[idx]

	if !reflect.DeepEqual(expectedInput, featureAvailability) {
		c.t.Errorf("unexpected input expected %s received %s", expectedInput, featureAvailability)
	}
	c.insertBrowserFeatureAvailabilityCountPerFeature[featureID]++

	return c.mockUpsertBrowserFeatureAvailabilityCfg.outputs[featureID][idx]
}

func (c *mockWebFeatureSpannerClient) PrecalculateBrowserFeatureSupportEvents(_ context.Context,
	startAt, endAt time.Time) error {
	c.precalculateBrowserFeatureSupportEventsCount++
	if !startAt.Equal(testInsertWebFeaturesStartAt) {
		c.t.Errorf("unexpected startAt time %s", startAt)
	}
	if !endAt.Equal(testInsertWebFeaturesEndAt) {
		c.t.Errorf("unexpected endAt time %s", endAt)
	}

	return c.mockPrecalculateBrowserFeatureSupportEventsCfg.err
}

func (c *mockWebFeatureSpannerClient) UpsertFeatureDiscouragedDetails(
	_ context.Context, featureID string, in gcpspanner.FeatureDiscouragedDetails) error {
	if len(c.mockUpsertFeatureDiscouragedDetailsCfg.expectedInputs) <= c.upsertFeatureDiscouragedDetailsCount {
		c.t.Fatal("no more expected input for UpsertFeatureDiscouragedDetails")
	}
	if len(c.mockUpsertFeatureDiscouragedDetailsCfg.outputs) <= c.upsertFeatureDiscouragedDetailsCount {
		c.t.Fatal("no more configured outputs for UpsertFeatureDiscouragedDetails")
	}
	expectedInput, found := c.mockUpsertFeatureDiscouragedDetailsCfg.expectedInputs[featureID]
	if !found {
		c.t.Errorf("unexpected input %v", in)
	}
	if !reflect.DeepEqual(expectedInput, in) {
		c.t.Errorf("unexpected input expected %v received %v", expectedInput, in)
	}
	c.upsertFeatureDiscouragedDetailsCount++

	return c.mockUpsertFeatureDiscouragedDetailsCfg.outputs[featureID]
}

func newMockmockWebFeatureSpannerClient(
	t *testing.T,
	mockSyncWebFeaturesCfg *mockSyncWebFeaturesConfig,
	mockFetchIDsAndKeysCfg *mockFetchIDsAndKeysConfig,
	mockUpsertFeatureBaselineStatusCfg *mockUpsertFeatureBaselineStatusConfig,
	mockUpsertBrowserFeatureAvailabilityCfg *mockUpsertBrowserFeatureAvailabilityConfig,
	mockUpsertFeatureSpecCfg *mockUpsertFeatureSpecConfig,
	mocmockPrecalculateBrowserFeatureSupportEventsCfg *mockPrecalculateBrowserFeatureSupportEventsConfig,
	mockUpsertFeatureDiscouragedDetailsCfg *mockUpsertFeatureDiscouragedDetailsConfig,
	mockSyncMovedWebFeaturesCfg *mockSyncMovedWebFeaturesConfig,
	mockSyncSplitWebFeaturesCfg *mockSyncSplitWebFeaturesConfig,
) *mockWebFeatureSpannerClient {
	return &mockWebFeatureSpannerClient{
		t:                                       t,
		mockSyncWebFeaturesCfg:                  mockSyncWebFeaturesCfg,
		mockFetchIDsAndKeysCfg:                  mockFetchIDsAndKeysCfg,
		mockUpsertFeatureBaselineStatusCfg:      mockUpsertFeatureBaselineStatusCfg,
		mockUpsertBrowserFeatureAvailabilityCfg: mockUpsertBrowserFeatureAvailabilityCfg,
		mockUpsertFeatureSpecCfg:                mockUpsertFeatureSpecCfg,
		syncWebFeaturesCount:                    0,
		fetchIDsAndKeysCount:                    0,
		upsertFeatureBaselineStatusCount:        0,
		upsertFeatureSpecCount:                  0,
		insertBrowserFeatureAvailabilityCountPerFeature: map[string]int{},
		mockPrecalculateBrowserFeatureSupportEventsCfg:  mocmockPrecalculateBrowserFeatureSupportEventsCfg,
		precalculateBrowserFeatureSupportEventsCount:    0,
		mockUpsertFeatureDiscouragedDetailsCfg:          mockUpsertFeatureDiscouragedDetailsCfg,
		upsertFeatureDiscouragedDetailsCount:            0,
		mockSyncMovedWebFeaturesCfg:                     mockSyncMovedWebFeaturesCfg,
		syncMovedWebFeaturesCount:                       0,
		mockSyncSplitWebFeaturesCfg:                     mockSyncSplitWebFeaturesCfg,
		syncSplitWebFeaturesCount:                       0,
	}
}

var ErrSyncWebFeaturesTest = errors.New("sync web features test error")
var ErrFetchIDsAndKeysTest = errors.New("fetch IDs and keys test error")
var ErrBaselineStatusTest = errors.New("baseline status test error")
var ErrBrowserFeatureAvailabilityTest = errors.New("browser feature availability test error")
var ErrFeatureSpecTest = errors.New("feature spec test error")
var ErrPrecalculateBrowserFeatureSupportEventsTest = errors.New("precalculate support events error")
var ErrFeatureDiscouragedDetailsTest = errors.New("feature discouraged details test error")
var ErrSyncMovedWebFeaturesTest = errors.New("sync moved web features test error")
var ErrSyncSplitWebFeaturesTest = errors.New("sync split web features test error")

// nolint:gochecknoglobals
var (
	testInsertWebFeaturesStartAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	testInsertWebFeaturesEndAt   = time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC)
	fakeBrowsersData             = webdxfeaturetypes.Browsers{
		Chrome: webdxfeaturetypes.BrowserData{
			Name:     "chrome",
			Releases: nil,
		},
		ChromeAndroid: webdxfeaturetypes.BrowserData{
			Name:     "chrome_android",
			Releases: nil,
		},
		Edge: webdxfeaturetypes.BrowserData{
			Name:     "edge",
			Releases: nil,
		},
		Firefox: webdxfeaturetypes.BrowserData{
			Name:     "firefox",
			Releases: nil,
		},
		FirefoxAndroid: webdxfeaturetypes.BrowserData{
			Name:     "firefox_android",
			Releases: nil,
		},
		Safari: webdxfeaturetypes.BrowserData{
			Name:     "safari",
			Releases: nil,
		},
		SafariIos: webdxfeaturetypes.BrowserData{
			Name:     "safari_ios",
			Releases: nil,
		},
	}
)

func TestInsertWebFeatures(t *testing.T) {
	// nolint: dupl // WONTFIX - some of the test cases are similar. It is better to be explicit for each case.
	testCases := []struct {
		name                                           string
		mockSyncWebFeaturesCfg                         mockSyncWebFeaturesConfig
		mockFetchIDsAndKeysCfg                         mockFetchIDsAndKeysConfig
		mockUpsertFeatureBaselineStatusCfg             mockUpsertFeatureBaselineStatusConfig
		mockUpsertBrowserFeatureAvailabilityCfg        mockUpsertBrowserFeatureAvailabilityConfig
		mockUpsertFeatureSpecCfg                       mockUpsertFeatureSpecConfig
		mockPrecalculateBrowserFeatureSupportEventsCfg mockPrecalculateBrowserFeatureSupportEventsConfig
		mockUpsertFeatureDiscouragedDetailsCfg         mockUpsertFeatureDiscouragedDetailsConfig
		input                                          *webdxfeaturetypes.ProcessedWebFeaturesData
		expectedError                                  error // Expected error from InsertWebFeatures
	}{
		{
			name: "success",
			mockSyncWebFeaturesCfg: mockSyncWebFeaturesConfig{
				expectedInput: []gcpspanner.WebFeature{
					{
						FeatureKey:      "feature1",
						Name:            "Feature 1",
						Description:     "text",
						DescriptionHTML: "<html>",
					},
					{
						FeatureKey:      "feature2",
						Name:            "Feature 2",
						Description:     "text",
						DescriptionHTML: "<html>",
					},
				},
				expectedOptions: []gcpspanner.SyncWebFeaturesOption{},
				err:             nil,
				expectedCount:   1,
			},
			mockFetchIDsAndKeysCfg: mockFetchIDsAndKeysConfig{
				output: []gcpspanner.SpannerFeatureIDAndKey{
					{ID: "id-1", FeatureKey: "feature1"},
					{ID: "id-2", FeatureKey: "feature2"},
				},
				err:           nil,
				expectedCount: 1,
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
			mockUpsertBrowserFeatureAvailabilityCfg: mockUpsertBrowserFeatureAvailabilityConfig{
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
						{
							BrowserName:    "chrome_android",
							BrowserVersion: "104",
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
						{
							BrowserName:    "safari_ios",
							BrowserVersion: "106",
						},
					},
				},
				outputs: map[string][]error{
					"feature1": {nil, nil, nil, nil, nil},
					"feature2": {nil, nil, nil},
				},
				expectedCountPerFeature: map[string]int{
					"feature1": 5,
					"feature2": 3,
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
			input: &webdxfeaturetypes.ProcessedWebFeaturesData{
				Snapshots: nil,
				Browsers:  fakeBrowsersData,
				Groups:    nil,
				Features: &webdxfeaturetypes.FeatureKinds{
					Data: map[string]webdxfeaturetypes.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged: &webdxfeaturetypes.Discouraged{
								AccordingTo:  []string{"according-to-1", "according-to-2"},
								Alternatives: []string{"alternative-1", "alternative-2"},
							},
							Spec: []string{"feature1-link1", "feature1-link2"},
							Status: webdxfeaturetypes.Status{
								BaselineHighDate: nil,
								BaselineLowDate:  nil,
								ByCompatKey:      nil,
								Support: webdxfeaturetypes.StatusSupport{
									Chrome:         valuePtr("100"),
									ChromeAndroid:  valuePtr("104"),
									Edge:           valuePtr("101"),
									Firefox:        valuePtr("102"),
									FirefoxAndroid: nil,
									Safari:         valuePtr("103"),
									SafariIos:      nil,
								},
								Baseline: &webdxfeaturetypes.BaselineUnion{
									Enum: valuePtr(webdxfeaturetypes.High),
									Bool: nil,
								},
							},
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
						},
						"feature2": {
							Name:           "Feature 2",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged:    nil,
							Spec: []string{
								"feature2-link",
							},
							Status: webdxfeaturetypes.Status{
								BaselineHighDate: nil,
								BaselineLowDate:  nil,
								ByCompatKey:      nil,
								Support: webdxfeaturetypes.StatusSupport{
									Chrome:         nil,
									ChromeAndroid:  nil,
									Edge:           nil,
									Firefox:        valuePtr("202"),
									FirefoxAndroid: nil,
									Safari:         valuePtr("203"),
									SafariIos:      valuePtr("106"),
								},
								Baseline: &webdxfeaturetypes.BaselineUnion{
									Enum: valuePtr(webdxfeaturetypes.Low),
									Bool: nil,
								},
							},
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
						},
					},
					Moved: nil,
					Split: nil,
				},
			},
			mockPrecalculateBrowserFeatureSupportEventsCfg: mockPrecalculateBrowserFeatureSupportEventsConfig{
				expectedCount: 1,
				err:           nil,
			},
			mockUpsertFeatureDiscouragedDetailsCfg: mockUpsertFeatureDiscouragedDetailsConfig{
				expectedInputs: map[string]gcpspanner.FeatureDiscouragedDetails{
					"feature1": {
						AccordingTo:  []string{"according-to-1", "according-to-2"},
						Alternatives: []string{"alternative-1", "alternative-2"},
					},
				},
				outputs:       map[string]error{"feature1": nil},
				expectedCount: 1,
			},
			expectedError: nil,
		},
		{
			name: "SyncWebFeatures error",
			mockSyncWebFeaturesCfg: mockSyncWebFeaturesConfig{
				expectedInput: []gcpspanner.WebFeature{
					{FeatureKey: "feature1", Name: "Feature 1", Description: "text", DescriptionHTML: "<html>"},
				},
				expectedOptions: []gcpspanner.SyncWebFeaturesOption{},
				err:             ErrSyncWebFeaturesTest,
				expectedCount:   1,
			},
			mockFetchIDsAndKeysCfg: mockFetchIDsAndKeysConfig{
				output:        nil,
				err:           nil,
				expectedCount: 0,
			},
			mockUpsertFeatureBaselineStatusCfg: mockUpsertFeatureBaselineStatusConfig{
				expectedInputs: nil,
				outputs:        nil,
				expectedCount:  0,
			},
			mockUpsertBrowserFeatureAvailabilityCfg: mockUpsertBrowserFeatureAvailabilityConfig{
				expectedInputs:          map[string][]gcpspanner.BrowserFeatureAvailability{},
				outputs:                 map[string][]error{},
				expectedCountPerFeature: map[string]int{},
			},
			mockUpsertFeatureSpecCfg: mockUpsertFeatureSpecConfig{
				expectedInputs: map[string]gcpspanner.FeatureSpec{},
				outputs:        map[string]error{},
				expectedCount:  0,
			},
			mockPrecalculateBrowserFeatureSupportEventsCfg: mockPrecalculateBrowserFeatureSupportEventsConfig{
				expectedCount: 0,
				err:           nil,
			},
			mockUpsertFeatureDiscouragedDetailsCfg: mockUpsertFeatureDiscouragedDetailsConfig{
				expectedInputs: map[string]gcpspanner.FeatureDiscouragedDetails{},
				outputs:        map[string]error{},
				expectedCount:  0,
			},
			input: &webdxfeaturetypes.ProcessedWebFeaturesData{
				Snapshots: nil,
				Browsers:  fakeBrowsersData,
				Groups:    nil,
				Features: &webdxfeaturetypes.FeatureKinds{
					Data: map[string]webdxfeaturetypes.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged:    nil,
							Spec:           nil,
							Status: webdxfeaturetypes.Status{
								BaselineHighDate: nil,
								BaselineLowDate:  nil,
								ByCompatKey:      nil,
								Support: webdxfeaturetypes.StatusSupport{
									Chrome:         nil,
									ChromeAndroid:  nil,
									Edge:           nil,
									Firefox:        nil,
									FirefoxAndroid: nil,
									Safari:         nil,
									SafariIos:      nil,
								},
								Baseline: &webdxfeaturetypes.BaselineUnion{
									Enum: valuePtr(webdxfeaturetypes.High),
									Bool: nil,
								},
							},
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
						},
					},
					Moved: nil,
					Split: nil,
				},
			},
			expectedError: ErrSyncWebFeaturesTest,
		},
		{
			name: "UpsertFeatureBaselineStatus error",
			mockSyncWebFeaturesCfg: mockSyncWebFeaturesConfig{
				expectedInput: []gcpspanner.WebFeature{
					{
						FeatureKey:      "feature1",
						Name:            "Feature 1",
						Description:     "text",
						DescriptionHTML: "<html>",
					},
				},
				expectedOptions: []gcpspanner.SyncWebFeaturesOption{},
				err:             nil,
				expectedCount:   1,
			},
			mockFetchIDsAndKeysCfg: mockFetchIDsAndKeysConfig{
				output:        nil,
				err:           nil,
				expectedCount: 0,
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
			mockUpsertBrowserFeatureAvailabilityCfg: mockUpsertBrowserFeatureAvailabilityConfig{
				expectedInputs:          map[string][]gcpspanner.BrowserFeatureAvailability{},
				outputs:                 map[string][]error{},
				expectedCountPerFeature: map[string]int{},
			},
			mockUpsertFeatureSpecCfg: mockUpsertFeatureSpecConfig{
				expectedInputs: map[string]gcpspanner.FeatureSpec{},
				outputs:        map[string]error{},
				expectedCount:  0,
			},
			input: &webdxfeaturetypes.ProcessedWebFeaturesData{
				Snapshots: nil,
				Browsers:  fakeBrowsersData,
				Groups:    nil,
				Features: &webdxfeaturetypes.FeatureKinds{
					Data: map[string]webdxfeaturetypes.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged:    nil,
							Spec:           nil,
							Status: webdxfeaturetypes.Status{
								BaselineHighDate: nil,
								BaselineLowDate:  nil,
								ByCompatKey:      nil,
								Support: webdxfeaturetypes.StatusSupport{
									Chrome:         nil,
									ChromeAndroid:  nil,
									Edge:           nil,
									Firefox:        nil,
									FirefoxAndroid: nil,
									Safari:         nil,
									SafariIos:      nil,
								},
								Baseline: &webdxfeaturetypes.BaselineUnion{
									Enum: valuePtr(webdxfeaturetypes.High),
									Bool: nil,
								},
							},
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
						},
					},
					Moved: nil,
					Split: nil,
				},
			},
			mockPrecalculateBrowserFeatureSupportEventsCfg: mockPrecalculateBrowserFeatureSupportEventsConfig{
				expectedCount: 0,
				err:           nil,
			},
			mockUpsertFeatureDiscouragedDetailsCfg: mockUpsertFeatureDiscouragedDetailsConfig{
				expectedInputs: map[string]gcpspanner.FeatureDiscouragedDetails{},
				outputs:        map[string]error{},
				expectedCount:  0,
			},
			expectedError: ErrBaselineStatusTest,
		},
		{
			name: "UpsertBrowserFeatureAvailability error",
			mockSyncWebFeaturesCfg: mockSyncWebFeaturesConfig{
				expectedInput: []gcpspanner.WebFeature{
					{
						FeatureKey:      "feature1",
						Name:            "Feature 1",
						Description:     "text",
						DescriptionHTML: "<html>",
					},
				},
				expectedOptions: []gcpspanner.SyncWebFeaturesOption{},
				err:             nil,
				expectedCount:   1,
			},
			mockFetchIDsAndKeysCfg: mockFetchIDsAndKeysConfig{
				output:        nil,
				err:           nil,
				expectedCount: 0,
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
			mockUpsertBrowserFeatureAvailabilityCfg: mockUpsertBrowserFeatureAvailabilityConfig{
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
			input: &webdxfeaturetypes.ProcessedWebFeaturesData{
				Snapshots: nil,
				Browsers:  fakeBrowsersData,
				Groups:    nil,
				Features: &webdxfeaturetypes.FeatureKinds{
					Data: map[string]webdxfeaturetypes.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged:    nil,
							Spec:           nil,
							Status: webdxfeaturetypes.Status{
								BaselineHighDate: nil,
								BaselineLowDate:  nil,
								ByCompatKey:      nil,
								Support: webdxfeaturetypes.StatusSupport{
									Chrome:         valuePtr("100"),
									ChromeAndroid:  nil,
									Edge:           valuePtr("101"),
									Firefox:        valuePtr("102"),
									FirefoxAndroid: nil,
									Safari:         valuePtr("103"),
									SafariIos:      nil,
								},
								Baseline: &webdxfeaturetypes.BaselineUnion{
									Enum: valuePtr(webdxfeaturetypes.High),
									Bool: nil,
								},
							},
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
						},
					},
					Moved: nil,
					Split: nil,
				},
			},
			mockPrecalculateBrowserFeatureSupportEventsCfg: mockPrecalculateBrowserFeatureSupportEventsConfig{
				expectedCount: 0,
				err:           nil,
			},
			mockUpsertFeatureDiscouragedDetailsCfg: mockUpsertFeatureDiscouragedDetailsConfig{
				expectedInputs: map[string]gcpspanner.FeatureDiscouragedDetails{},
				outputs:        map[string]error{},
				expectedCount:  0,
			},
			expectedError: ErrBrowserFeatureAvailabilityTest,
		},
		{
			name: "upsert feature spec failure",
			mockSyncWebFeaturesCfg: mockSyncWebFeaturesConfig{
				expectedInput: []gcpspanner.WebFeature{
					{
						FeatureKey:      "feature1",
						Name:            "Feature 1",
						Description:     "text",
						DescriptionHTML: "<html>",
					},
				},
				expectedOptions: []gcpspanner.SyncWebFeaturesOption{},
				err:             nil,
				expectedCount:   1,
			},
			mockFetchIDsAndKeysCfg: mockFetchIDsAndKeysConfig{
				output:        nil,
				err:           nil,
				expectedCount: 0,
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
			mockUpsertBrowserFeatureAvailabilityCfg: mockUpsertBrowserFeatureAvailabilityConfig{
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
						{
							BrowserName:    "chrome_android",
							BrowserVersion: "104",
						},
						{
							BrowserName:    "firefox_android",
							BrowserVersion: "105",
						},
						{
							BrowserName:    "safari_ios",
							BrowserVersion: "106",
						},
					},
				},
				outputs: map[string][]error{
					"feature1": {nil, nil, nil, nil, nil, nil, nil},
				},
				expectedCountPerFeature: map[string]int{
					"feature1": 7,
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
			input: &webdxfeaturetypes.ProcessedWebFeaturesData{
				Snapshots: nil,
				Browsers:  fakeBrowsersData,
				Groups:    nil,
				Features: &webdxfeaturetypes.FeatureKinds{
					Data: map[string]webdxfeaturetypes.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged:    nil,
							Spec:           []string{"feature1-link1", "feature1-link2"},
							Status: webdxfeaturetypes.Status{
								BaselineHighDate: nil,
								BaselineLowDate:  nil,
								ByCompatKey:      nil,
								Support: webdxfeaturetypes.StatusSupport{
									Chrome:         valuePtr("100"),
									ChromeAndroid:  valuePtr("104"),
									Edge:           valuePtr("101"),
									Firefox:        valuePtr("102"),
									FirefoxAndroid: valuePtr("105"),
									Safari:         valuePtr("103"),
									SafariIos:      valuePtr("106"),
								},
								Baseline: &webdxfeaturetypes.BaselineUnion{
									Enum: valuePtr(webdxfeaturetypes.High),
									Bool: nil,
								},
							},
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
						},
					},
					Moved: nil,
					Split: nil,
				},
			},
			mockPrecalculateBrowserFeatureSupportEventsCfg: mockPrecalculateBrowserFeatureSupportEventsConfig{
				expectedCount: 0,
				err:           nil,
			},
			mockUpsertFeatureDiscouragedDetailsCfg: mockUpsertFeatureDiscouragedDetailsConfig{
				expectedInputs: map[string]gcpspanner.FeatureDiscouragedDetails{},
				outputs:        map[string]error{},
				expectedCount:  0,
			},
			expectedError: ErrFeatureSpecTest,
		},
		{
			name: "PrecalculateBrowserFeatureSupportEvents failure",
			mockSyncWebFeaturesCfg: mockSyncWebFeaturesConfig{
				expectedInput: []gcpspanner.WebFeature{
					{
						FeatureKey:      "feature1",
						Name:            "Feature 1",
						Description:     "text",
						DescriptionHTML: "<html>",
					},
					{
						FeatureKey:      "feature2",
						Name:            "Feature 2",
						Description:     "text",
						DescriptionHTML: "<html>",
					},
				},
				expectedOptions: []gcpspanner.SyncWebFeaturesOption{},
				err:             nil,
				expectedCount:   1,
			},
			mockFetchIDsAndKeysCfg: mockFetchIDsAndKeysConfig{
				output:        nil,
				err:           nil,
				expectedCount: 0,
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
			mockUpsertBrowserFeatureAvailabilityCfg: mockUpsertBrowserFeatureAvailabilityConfig{
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
						{
							BrowserName:    "chrome_android",
							BrowserVersion: "104",
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
						{
							BrowserName:    "safari_ios",
							BrowserVersion: "106",
						},
					},
				},
				outputs: map[string][]error{
					"feature1": {nil, nil, nil, nil, nil},
					"feature2": {nil, nil, nil},
				},
				expectedCountPerFeature: map[string]int{
					"feature1": 5,
					"feature2": 3,
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
			input: &webdxfeaturetypes.ProcessedWebFeaturesData{
				Snapshots: nil,
				Browsers:  fakeBrowsersData,
				Groups:    nil,
				Features: &webdxfeaturetypes.FeatureKinds{
					Data: map[string]webdxfeaturetypes.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged:    nil,
							Spec:           []string{"feature1-link1", "feature1-link2"},
							Status: webdxfeaturetypes.Status{
								BaselineHighDate: nil,
								BaselineLowDate:  nil,
								ByCompatKey:      nil,
								Support: webdxfeaturetypes.StatusSupport{
									Chrome:         valuePtr("100"),
									ChromeAndroid:  valuePtr("104"),
									Edge:           valuePtr("101"),
									Firefox:        valuePtr("102"),
									FirefoxAndroid: nil,
									Safari:         valuePtr("103"),
									SafariIos:      nil,
								},
								Baseline: &webdxfeaturetypes.BaselineUnion{
									Enum: valuePtr(webdxfeaturetypes.High),
									Bool: nil,
								},
							},
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
						},
						"feature2": {
							Name:           "Feature 2",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged:    nil,
							Spec:           []string{"feature2-link"},
							Status: webdxfeaturetypes.Status{
								BaselineHighDate: nil,
								BaselineLowDate:  nil,
								ByCompatKey:      nil,
								Support: webdxfeaturetypes.StatusSupport{
									Chrome:         nil,
									ChromeAndroid:  nil,
									Edge:           nil,
									Firefox:        valuePtr("202"),
									FirefoxAndroid: nil,
									Safari:         valuePtr("203"),
									SafariIos:      valuePtr("106"),
								},
								Baseline: &webdxfeaturetypes.BaselineUnion{
									Enum: valuePtr(webdxfeaturetypes.Low),
									Bool: nil,
								},
							},
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
						},
					},
					Moved: nil,
					Split: nil,
				},
			},
			mockPrecalculateBrowserFeatureSupportEventsCfg: mockPrecalculateBrowserFeatureSupportEventsConfig{
				expectedCount: 1,
				err:           ErrPrecalculateBrowserFeatureSupportEventsTest,
			},
			mockUpsertFeatureDiscouragedDetailsCfg: mockUpsertFeatureDiscouragedDetailsConfig{
				expectedInputs: map[string]gcpspanner.FeatureDiscouragedDetails{},
				outputs:        map[string]error{},
				expectedCount:  0,
			},
			expectedError: ErrPrecalculateBrowserFeatureSupportEventsTest,
		},
		{
			name: "FetchAllWebFeatureIDsAndKeys error",
			mockSyncWebFeaturesCfg: mockSyncWebFeaturesConfig{
				expectedInput: []gcpspanner.WebFeature{
					{
						FeatureKey:      "feature1",
						Name:            "Feature 1",
						Description:     "text",
						DescriptionHTML: "<html>",
					},
					{
						FeatureKey:      "feature2",
						Name:            "Feature 2",
						Description:     "text",
						DescriptionHTML: "<html>",
					},
				},
				expectedOptions: []gcpspanner.SyncWebFeaturesOption{},
				err:             nil,
				expectedCount:   1,
			},
			mockFetchIDsAndKeysCfg: mockFetchIDsAndKeysConfig{
				output:        nil,
				err:           ErrFetchIDsAndKeysTest,
				expectedCount: 1,
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
			mockUpsertBrowserFeatureAvailabilityCfg: mockUpsertBrowserFeatureAvailabilityConfig{
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
						{
							BrowserName:    "chrome_android",
							BrowserVersion: "104",
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
						{
							BrowserName:    "safari_ios",
							BrowserVersion: "106",
						},
					},
				},
				outputs: map[string][]error{
					"feature1": {nil, nil, nil, nil, nil},
					"feature2": {nil, nil, nil},
				},
				expectedCountPerFeature: map[string]int{
					"feature1": 5,
					"feature2": 3,
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
			input: &webdxfeaturetypes.ProcessedWebFeaturesData{
				Snapshots: nil,
				Browsers:  fakeBrowsersData,
				Groups:    nil,
				Features: &webdxfeaturetypes.FeatureKinds{
					Data: map[string]webdxfeaturetypes.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged: &webdxfeaturetypes.Discouraged{
								AccordingTo:  []string{"according-to-1", "according-to-2"},
								Alternatives: []string{"alternative-1", "alternative-2"},
							},
							Spec: []string{"feature1-link1", "feature1-link2"},
							Status: webdxfeaturetypes.Status{
								BaselineHighDate: nil,
								BaselineLowDate:  nil,
								ByCompatKey:      nil,
								Support: webdxfeaturetypes.StatusSupport{
									Chrome:         valuePtr("100"),
									ChromeAndroid:  valuePtr("104"),
									Edge:           valuePtr("101"),
									Firefox:        valuePtr("102"),
									FirefoxAndroid: nil,
									Safari:         valuePtr("103"),
									SafariIos:      nil,
								},
								Baseline: &webdxfeaturetypes.BaselineUnion{
									Enum: valuePtr(webdxfeaturetypes.High),
									Bool: nil,
								},
							},
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
						},
						"feature2": {
							Name:           "Feature 2",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged:    nil,
							Spec:           []string{"feature2-link"},
							Status: webdxfeaturetypes.Status{
								BaselineHighDate: nil,
								BaselineLowDate:  nil,
								ByCompatKey:      nil,
								Support: webdxfeaturetypes.StatusSupport{
									Chrome:         nil,
									ChromeAndroid:  nil,
									Edge:           nil,
									Firefox:        valuePtr("202"),
									FirefoxAndroid: nil,
									Safari:         valuePtr("203"),
									SafariIos:      valuePtr("106"),
								},
								Baseline: &webdxfeaturetypes.BaselineUnion{
									Enum: valuePtr(webdxfeaturetypes.Low),
									Bool: nil,
								},
							},
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
						},
					},
					Moved: nil,
					Split: nil,
				},
			},
			mockPrecalculateBrowserFeatureSupportEventsCfg: mockPrecalculateBrowserFeatureSupportEventsConfig{
				expectedCount: 1,
				err:           nil,
			},
			mockUpsertFeatureDiscouragedDetailsCfg: mockUpsertFeatureDiscouragedDetailsConfig{
				expectedInputs: map[string]gcpspanner.FeatureDiscouragedDetails{
					"feature1": {
						AccordingTo:  []string{"according-to-1", "according-to-2"},
						Alternatives: []string{"alternative-1", "alternative-2"},
					},
				},
				outputs:       map[string]error{"feature1": nil},
				expectedCount: 1,
			},
			expectedError: ErrFetchIDsAndKeysTest,
		},
		{
			name: "UpsertFeatureDiscouragedDetails error",
			mockSyncWebFeaturesCfg: mockSyncWebFeaturesConfig{
				expectedInput: []gcpspanner.WebFeature{
					{
						FeatureKey:      "feature1",
						Name:            "Feature 1",
						Description:     "text",
						DescriptionHTML: "<html>",
					},
				},
				expectedOptions: []gcpspanner.SyncWebFeaturesOption{},
				err:             nil,
				expectedCount:   1,
			},
			mockFetchIDsAndKeysCfg: mockFetchIDsAndKeysConfig{
				output:        nil,
				err:           nil,
				expectedCount: 0,
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
			mockUpsertBrowserFeatureAvailabilityCfg: mockUpsertBrowserFeatureAvailabilityConfig{
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
						{
							BrowserName:    "chrome_android",
							BrowserVersion: "104",
						},
					},
				},
				outputs: map[string][]error{
					"feature1": {nil, nil, nil, nil, nil},
				},
				expectedCountPerFeature: map[string]int{
					"feature1": 5,
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
					"feature1": nil,
				},
				expectedCount: 1,
			},
			input: &webdxfeaturetypes.ProcessedWebFeaturesData{
				Snapshots: nil,
				Browsers:  fakeBrowsersData,
				Groups:    nil,
				Features: &webdxfeaturetypes.FeatureKinds{
					Data: map[string]webdxfeaturetypes.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged: &webdxfeaturetypes.Discouraged{
								AccordingTo:  []string{"according-to-1", "according-to-2"},
								Alternatives: []string{"alternative-1", "alternative-2"},
							},
							Spec: []string{"feature1-link1", "feature1-link2"},
							Status: webdxfeaturetypes.Status{
								BaselineHighDate: nil,
								BaselineLowDate:  nil,
								ByCompatKey:      nil,
								Support: webdxfeaturetypes.StatusSupport{
									Chrome:         valuePtr("100"),
									ChromeAndroid:  valuePtr("104"),
									Edge:           valuePtr("101"),
									Firefox:        valuePtr("102"),
									FirefoxAndroid: nil,
									Safari:         valuePtr("103"),
									SafariIos:      nil,
								},
								Baseline: &webdxfeaturetypes.BaselineUnion{
									Enum: valuePtr(webdxfeaturetypes.High),
									Bool: nil,
								},
							},
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
						},
					},
					Moved: nil,
					Split: nil,
				},
			},
			mockPrecalculateBrowserFeatureSupportEventsCfg: mockPrecalculateBrowserFeatureSupportEventsConfig{
				expectedCount: 0,
				err:           nil,
			},
			mockUpsertFeatureDiscouragedDetailsCfg: mockUpsertFeatureDiscouragedDetailsConfig{
				expectedInputs: map[string]gcpspanner.FeatureDiscouragedDetails{
					"feature1": {
						AccordingTo:  []string{"according-to-1", "according-to-2"},
						Alternatives: []string{"alternative-1", "alternative-2"},
					},
				},
				outputs:       map[string]error{"feature1": ErrFeatureDiscouragedDetailsTest},
				expectedCount: 1,
			},
			expectedError: ErrFeatureDiscouragedDetailsTest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := newMockmockWebFeatureSpannerClient(
				t,
				&tc.mockSyncWebFeaturesCfg,
				&tc.mockFetchIDsAndKeysCfg,
				&tc.mockUpsertFeatureBaselineStatusCfg,
				&tc.mockUpsertBrowserFeatureAvailabilityCfg,
				&tc.mockUpsertFeatureSpecCfg,
				&tc.mockPrecalculateBrowserFeatureSupportEventsCfg,
				&tc.mockUpsertFeatureDiscouragedDetailsCfg,
				nil,
				nil,
			)
			consumer := NewWebFeaturesConsumer(mockClient)

			_, err := consumer.InsertWebFeatures(context.TODO(), tc.input,
				testInsertWebFeaturesStartAt, testInsertWebFeaturesEndAt)

			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error: got %v, want %v", err, tc.expectedError)
			}

			if mockClient.syncWebFeaturesCount != mockClient.mockSyncWebFeaturesCfg.expectedCount {
				t.Errorf("expected %d calls to SyncWebFeatures, got %d",
					mockClient.mockSyncWebFeaturesCfg.expectedCount,
					mockClient.syncWebFeaturesCount)
			}

			if mockClient.fetchIDsAndKeysCount != mockClient.mockFetchIDsAndKeysCfg.expectedCount {
				t.Errorf("expected %d calls to FetchAllWebFeatureIDsAndKeys, got %d",
					mockClient.mockFetchIDsAndKeysCfg.expectedCount,
					mockClient.fetchIDsAndKeysCount)
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
				tc.mockUpsertBrowserFeatureAvailabilityCfg.expectedCountPerFeature) {
				t.Errorf("Unexpected call counts for UpsertBrowserFeatureAvailability. Expected: %v, Got: %v",
					tc.mockUpsertBrowserFeatureAvailabilityCfg.expectedCountPerFeature,
					mockClient.insertBrowserFeatureAvailabilityCountPerFeature)
			}

			if mockClient.precalculateBrowserFeatureSupportEventsCount !=
				mockClient.mockPrecalculateBrowserFeatureSupportEventsCfg.expectedCount {
				t.Errorf("expected %d calls to PrecalculateBrowserFeatureSupportEvents, got %d",
					mockClient.mockPrecalculateBrowserFeatureSupportEventsCfg.expectedCount,
					mockClient.precalculateBrowserFeatureSupportEventsCount)
			}

			if mockClient.upsertFeatureDiscouragedDetailsCount !=
				mockClient.mockUpsertFeatureDiscouragedDetailsCfg.expectedCount {
				t.Errorf("expected %d calls to UpsertFeatureDiscouragedDetails, got %d",
					mockClient.mockUpsertFeatureDiscouragedDetailsCfg.expectedCount,
					mockClient.upsertFeatureDiscouragedDetailsCount)
			}
		})
	}
}

func TestInsertMovedWebFeatures(t *testing.T) {
	testCases := []struct {
		name                        string
		movedFeatures               map[string]webdxfeaturetypes.FeatureMovedData
		mockSyncMovedWebFeaturesCfg *mockSyncMovedWebFeaturesConfig
		expectedErr                 error
	}{
		{
			name: "success",
			movedFeatures: map[string]webdxfeaturetypes.FeatureMovedData{
				"feature1": {
					Kind:           webdxfeaturetypes.Moved,
					RedirectTarget: "targetA",
				},
			},
			mockSyncMovedWebFeaturesCfg: &mockSyncMovedWebFeaturesConfig{
				expectedFeatures: []gcpspanner.MovedWebFeature{
					{
						OriginalFeatureKey: "feature1",
						NewFeatureKey:      "targetA",
					},
				},
				err:           nil,
				expectedCount: 1,
			},
			expectedErr: nil,
		},
		{
			name:          "error",
			movedFeatures: nil,
			mockSyncMovedWebFeaturesCfg: &mockSyncMovedWebFeaturesConfig{
				expectedFeatures: []gcpspanner.MovedWebFeature{},
				err:              ErrSyncMovedWebFeaturesTest,
				expectedCount:    1,
			},
			expectedErr: ErrSyncMovedWebFeaturesTest,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := newMockmockWebFeatureSpannerClient(
				t,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				tc.mockSyncMovedWebFeaturesCfg,
				nil,
			)
			consumer := NewWebFeaturesConsumer(mockClient)
			err := consumer.InsertMovedWebFeatures(context.TODO(), tc.movedFeatures)
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("unexpected error: got %v, want %v", err, tc.expectedErr)
			}

			if mockClient.syncMovedWebFeaturesCount != tc.mockSyncMovedWebFeaturesCfg.expectedCount {
				t.Errorf("expected %d calls to SyncMovedWebFeatures, got %d",
					tc.mockSyncMovedWebFeaturesCfg.expectedCount,
					mockClient.syncMovedWebFeaturesCount)
			}
		})
	}
}

func TestInsertSplitWebFeatures(t *testing.T) {
	testCases := []struct {
		name                        string
		splitFeatures               map[string]webdxfeaturetypes.FeatureSplitData
		mockSyncSplitWebFeaturesCfg *mockSyncSplitWebFeaturesConfig
		expectedError               error
	}{
		{
			name: "success",
			splitFeatures: map[string]webdxfeaturetypes.FeatureSplitData{
				"feature1": {
					Kind: webdxfeaturetypes.Split,
					RedirectTargets: []string{
						"target1",
						"target2",
					},
				},
			},
			mockSyncSplitWebFeaturesCfg: &mockSyncSplitWebFeaturesConfig{
				expectedFeatures: []gcpspanner.SplitWebFeature{
					{
						OriginalFeatureKey: "feature1",
						TargetFeatureKeys: []string{
							"target1",
							"target2",
						},
					},
				},
				expectedCount: 1,
				err:           nil,
			},
			expectedError: nil,
		},
		{
			name:          "error",
			splitFeatures: nil,
			mockSyncSplitWebFeaturesCfg: &mockSyncSplitWebFeaturesConfig{
				expectedFeatures: []gcpspanner.SplitWebFeature{},
				expectedCount:    1,
				err:              ErrSyncSplitWebFeaturesTest,
			},
			expectedError: ErrSyncSplitWebFeaturesTest,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := newMockmockWebFeatureSpannerClient(
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				tc.mockSyncSplitWebFeaturesCfg,
			)
			consumer := NewWebFeaturesConsumer(mockClient)
			err := consumer.InsertSplitWebFeatures(context.TODO(), tc.splitFeatures)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error: got %v, want %v", err, tc.expectedError)
			}

			if mockClient.syncSplitWebFeaturesCount != tc.mockSyncSplitWebFeaturesCfg.expectedCount {
				t.Errorf("expected %d calls to SyncSplitWebFeatures, got %d",
					tc.mockSyncSplitWebFeaturesCfg.expectedCount,
					mockClient.syncSplitWebFeaturesCount)
			}
		})
	}
}
