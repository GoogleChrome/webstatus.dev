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

package data

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
	"github.com/GoogleChrome/webstatus.dev/lib/webdxfeaturetypes"
	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name string
		path string
	}{
		{
			name: "data.json from https://github.com/web-platform-dx/web-features/releases/tag/v2.25.0",
			path: path.Join("testdata", "data.json"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			file, err := os.Open(tc.path)
			if err != nil {
				t.Fatalf("unable to read file err %s", err.Error())
			}
			result, err := Parser{}.Parse(file)
			if err != nil {
				t.Errorf("unable to parse file err %s", err.Error())
			}
			if len(result.Features.Data) == 0 {
				t.Error("unexpected empty map for features")
			}
			if len(result.Features.Moved) != 0 {
				t.Error("unexpected map for moved features")
			}
			if len(result.Features.Split) != 0 {
				t.Error("unexpected map for split features")
			}
			if len(result.Groups) == 0 {
				t.Error("unexpected empty map for groups")
			}
			if len(result.Snapshots) == 0 {
				t.Error("unexpected empty map for snapshots")
			}
		})
	}
}

func TestParseV3(t *testing.T) {
	testCases := []struct {
		name string
		path string
	}{
		{
			name: "data.json from https://github.com/web-platform-dx/web-features/releases/tag/v3.0.0",
			path: path.Join("testdata", "data.json"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Skip("3.0.0 does not exist yet")
			file, err := os.Open(tc.path)
			if err != nil {
				t.Fatalf("unable to read file err %s", err.Error())
			}
			result, err := Parser{}.Parse(file)
			if err != nil {
				t.Errorf("unable to parse file err %s", err.Error())
			}
			if len(result.Features.Data) == 0 {
				t.Error("unexpected empty map for features")
			}
			if len(result.Features.Moved) == 0 {
				t.Error("unexpected empty map for moved features")
			}
			if len(result.Features.Split) == 0 {
				t.Error("unexpected empty map for split features")
			}
			if len(result.Groups) == 0 {
				t.Error("unexpected empty map for groups")
			}
			if len(result.Snapshots) == 0 {
				t.Error("unexpected empty map for snapshots")
			}
		})
	}

}

type parser interface {
	Parse(io.ReadCloser) (*webdxfeaturetypes.ProcessedWebFeaturesData, error)
}

func TestParseError(t *testing.T) {
	testCases := []struct {
		name          string
		input         io.ReadCloser
		testParser    parser
		expectedError error
	}{
		{
			name:          "bad format",
			input:         io.NopCloser(strings.NewReader("Hello, world!")),
			expectedError: ErrUnexpectedFormat,
			testParser:    Parser{},
		},
		{
			name:          "bad format",
			input:         io.NopCloser(strings.NewReader("Hello, world!")),
			expectedError: ErrUnexpectedFormat,
			testParser:    V3Parser{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.testParser.Parse(tc.input)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error expected %v received %v", tc.expectedError, err)
			}
			if result != nil {
				t.Error("unexpected map")
			}
		})
	}
}

func valuePtr[T any](in T) *T { return &in }

func testBrowsers() web_platform_dx__web_features.Browsers {
	return web_platform_dx__web_features.Browsers{
		Chrome: web_platform_dx__web_features.BrowserData{
			Name:     "chrome",
			Releases: nil,
		},
		ChromeAndroid: web_platform_dx__web_features.BrowserData{
			Name:     "chrome_android",
			Releases: nil,
		},
		Edge: web_platform_dx__web_features.BrowserData{
			Name:     "edge",
			Releases: nil,
		},
		Firefox: web_platform_dx__web_features.BrowserData{
			Name:     "firefox",
			Releases: nil,
		},
		FirefoxAndroid: web_platform_dx__web_features.BrowserData{
			Name:     "firefox_android",
			Releases: nil,
		},
		Safari: web_platform_dx__web_features.BrowserData{
			Name:     "safari",
			Releases: nil,
		},
		SafariIos: web_platform_dx__web_features.BrowserData{
			Name:     "safari_ios",
			Releases: nil,
		},
	}
}

func TestPostProcess(t *testing.T) {
	testCases := []struct {
		name          string
		featureData   *rawWebFeaturesJSONDataV2
		expectedValue *webdxfeaturetypes.ProcessedWebFeaturesData
	}{
		{
			name: "catch-all case",
			featureData: &rawWebFeaturesJSONDataV2{
				Browsers:  testBrowsers(),
				Groups:    nil,
				Snapshots: nil,
				Features: map[string]web_platform_dx__web_features.FeatureValue{
					"feature1": {
						CompatFeatures:  []string{"compat1", "compat2"},
						Description:     "description",
						DescriptionHTML: "description html",
						Discouraged: &web_platform_dx__web_features.Discouraged{
							AccordingTo: []string{
								"discouraged1",
								"discouraged2",
							},
							Alternatives: []string{
								"feature2",
								"feature3",
							},
						},
						Name: "feature 1 name",
						Caniuse: &web_platform_dx__web_features.StringOrStringArray{
							String: valuePtr("caniuse_data"),
							StringArray: []string{
								"caniuse1",
								"caniuse2",
							},
						},
						Group: &web_platform_dx__web_features.StringOrStringArray{
							String: valuePtr("group_name"),
							StringArray: []string{
								"group1",
								"group2",
							},
						},
						Snapshot: &web_platform_dx__web_features.StringOrStringArray{
							String: valuePtr("snapshot_data"),
							StringArray: []string{
								"snapshot1",
								"snapshot2",
							},
						},
						Spec: &web_platform_dx__web_features.StringOrStringArray{
							String: valuePtr("spec_link"),
							StringArray: []string{
								"spec1",
								"spec2",
							},
						},
						Status: web_platform_dx__web_features.Status{
							Baseline: &web_platform_dx__web_features.BaselineUnion{
								Bool: valuePtr(false),
								Enum: valuePtr(web_platform_dx__web_features.High),
							},
							BaselineHighDate: valuePtr("≤2023-01-01"),
							BaselineLowDate:  valuePtr("≤2022-12-01"),
							ByCompatKey:      nil,
							Support: web_platform_dx__web_features.StatusSupport{
								Chrome:         valuePtr("≤99"),
								ChromeAndroid:  valuePtr("≤98"),
								Firefox:        valuePtr("≤97"),
								FirefoxAndroid: valuePtr("≤96"),
								Edge:           valuePtr("≤95"),
								Safari:         valuePtr("≤94"),
								SafariIos:      valuePtr("≤93"),
							},
						},
					},
				},
			},
			expectedValue: &webdxfeaturetypes.ProcessedWebFeaturesData{
				Browsers: web_platform_dx__web_features.Browsers{
					Chrome: web_platform_dx__web_features.BrowserData{
						Name:     "chrome",
						Releases: nil,
					},
					ChromeAndroid: web_platform_dx__web_features.BrowserData{
						Name:     "chrome_android",
						Releases: nil,
					},
					Edge: web_platform_dx__web_features.BrowserData{
						Name:     "edge",
						Releases: nil,
					},
					Firefox: web_platform_dx__web_features.BrowserData{
						Name:     "firefox",
						Releases: nil,
					},
					FirefoxAndroid: web_platform_dx__web_features.BrowserData{
						Name:     "firefox_android",
						Releases: nil,
					},
					Safari: web_platform_dx__web_features.BrowserData{
						Name:     "safari",
						Releases: nil,
					},
					SafariIos: web_platform_dx__web_features.BrowserData{
						Name:     "safari_ios",
						Releases: nil,
					},
				},
				Features: &webdxfeaturetypes.FeatureKinds{
					Data: map[string]web_platform_dx__web_features.FeatureValue{
						"feature1": {
							CompatFeatures:  []string{"compat1", "compat2"},
							Description:     "description",
							DescriptionHTML: "description html",
							Discouraged: &web_platform_dx__web_features.Discouraged{
								AccordingTo: []string{
									"discouraged1",
									"discouraged2",
								},
								Alternatives: []string{
									"feature2",
									"feature3",
								},
							},
							Name: "feature 1 name",
							Caniuse: &web_platform_dx__web_features.StringOrStringArray{
								String: valuePtr("caniuse_data"),
								StringArray: []string{
									"caniuse1",
									"caniuse2",
								},
							},
							Group: &web_platform_dx__web_features.StringOrStringArray{
								String: valuePtr("group_name"),
								StringArray: []string{
									"group1",
									"group2",
								},
							},
							Snapshot: &web_platform_dx__web_features.StringOrStringArray{
								String: valuePtr("snapshot_data"),
								StringArray: []string{
									"snapshot1",
									"snapshot2",
								},
							},
							Spec: &web_platform_dx__web_features.StringOrStringArray{
								String: valuePtr("spec_link"),
								StringArray: []string{
									"spec1",
									"spec2",
								},
							},
							Status: web_platform_dx__web_features.Status{
								ByCompatKey: nil,
								Baseline: &web_platform_dx__web_features.BaselineUnion{
									Bool: valuePtr(false),
									Enum: valuePtr(web_platform_dx__web_features.High),
								},
								BaselineHighDate: valuePtr("2023-01-01"),
								BaselineLowDate:  valuePtr("2022-12-01"),
								Support: web_platform_dx__web_features.StatusSupport{
									Chrome:         valuePtr("99"),
									ChromeAndroid:  valuePtr("98"),
									Firefox:        valuePtr("97"),
									FirefoxAndroid: valuePtr("96"),
									Edge:           valuePtr("95"),
									Safari:         valuePtr("94"),
									SafariIos:      valuePtr("93"),
								},
							},
						},
					},
					Moved: nil,
					Split: nil,
				},
				Snapshots: nil,
				Groups:    nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			value := postProcess(tc.featureData)
			if diff := cmp.Diff(tc.expectedValue, value); diff != "" {
				t.Errorf("postProcess unexpected output (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPostProcessV3(t *testing.T) {
	testCases := []struct {
		name          string
		featureData   *rawWebFeaturesJSONDataV3
		expectedValue *webdxfeaturetypes.ProcessedWebFeaturesData
		expectedErr   error
	}{
		{
			name: "catch-all case",
			featureData: &rawWebFeaturesJSONDataV3{
				Browsers: testBrowsers(),
				Groups: map[string]web_platform_dx__web_features.GroupData{
					"group1": {
						Name:   "Group 1",
						Parent: nil,
					},
				},
				Snapshots: map[string]web_platform_dx__web_features.SnapshotData{
					"snapshot1": {
						Name: "Snapshot 1",
						Spec: "spec1",
					},
				},
				Features: json.RawMessage(`
{
	"feature1": {
		"kind": "feature",
		"compat_features": [
			"compat1",
			"compat2"
		],
		"description": "description",
		"description_html": "description html",
		"discouraged": {
			"according_to": [
				"discouraged1",
				"discouraged2"
			],
			"alternatives": [
				"feature2",
				"feature3"
			]
		},
		"name": "feature 1 name",
		"caniuse": "caniuse_data",
		"group": [
			"group1",
			"group2"
		],
		"snapshot": [
			"snapshot1",
			"snapshot2"
		],
		"spec": [
			"spec1",
			"spec2"
		],
		"status": {
			"baseline": "high",
			"baseline_high_date": "≤2023-01-01",
			"baseline_low_date": "≤2022-12-01",
			"support": {
				"chrome": "≤99",
				"chrome_android": "≤98",
				"firefox": "≤97",
				"firefox_android": "≤96",
				"edge": "≤95",
				"safari": "≤94",
				"safari_ios": "≤93"
			}
		}
	},
	"feature2": {
		"kind": "split",
		"redirected_created_date": "2000-01-01",
		"redirect_targets": [
			"feature1",
			"feature3"
		]
	},
	"feature3": {
		"kind": "moved",
		"redirect_target": "feature4",
		"redirect_created_date": "2001-01-01"
	}
}`),
			},
			expectedValue: &webdxfeaturetypes.ProcessedWebFeaturesData{
				Browsers: web_platform_dx__web_features.Browsers{
					Chrome: web_platform_dx__web_features.BrowserData{
						Name:     "chrome",
						Releases: nil,
					},
					ChromeAndroid: web_platform_dx__web_features.BrowserData{
						Name:     "chrome_android",
						Releases: nil,
					},
					Edge: web_platform_dx__web_features.BrowserData{
						Name:     "edge",
						Releases: nil,
					},
					Firefox: web_platform_dx__web_features.BrowserData{
						Name:     "firefox",
						Releases: nil,
					},
					FirefoxAndroid: web_platform_dx__web_features.BrowserData{
						Name:     "firefox_android",
						Releases: nil,
					},
					Safari: web_platform_dx__web_features.BrowserData{
						Name:     "safari",
						Releases: nil,
					},
					SafariIos: web_platform_dx__web_features.BrowserData{
						Name:     "safari_ios",
						Releases: nil,
					},
				},
				Features: &webdxfeaturetypes.FeatureKinds{
					Data: map[string]web_platform_dx__web_features.FeatureValue{
						"feature1": {
							CompatFeatures:  []string{"compat1", "compat2"},
							Description:     "description",
							DescriptionHTML: "description html",
							Discouraged: &web_platform_dx__web_features.Discouraged{
								AccordingTo: []string{
									"discouraged1",
									"discouraged2",
								},
								Alternatives: []string{
									"feature2",
									"feature3",
								},
							},
							Name: "feature 1 name",
							Caniuse: &web_platform_dx__web_features.StringOrStringArray{
								String:      valuePtr("caniuse_data"),
								StringArray: nil,
							},
							Group: &web_platform_dx__web_features.StringOrStringArray{
								String: nil,
								StringArray: []string{
									"group1",
									"group2",
								},
							},
							Snapshot: &web_platform_dx__web_features.StringOrStringArray{
								String: nil,
								StringArray: []string{
									"snapshot1",
									"snapshot2",
								},
							},
							Spec: &web_platform_dx__web_features.StringOrStringArray{
								String: nil,
								StringArray: []string{
									"spec1",
									"spec2",
								},
							},
							Status: web_platform_dx__web_features.Status{
								ByCompatKey: nil,
								Baseline: &web_platform_dx__web_features.BaselineUnion{
									Bool: nil,
									Enum: valuePtr(web_platform_dx__web_features.High),
								},
								BaselineHighDate: valuePtr("2023-01-01"),
								BaselineLowDate:  valuePtr("2022-12-01"),
								Support: web_platform_dx__web_features.StatusSupport{
									Chrome:         valuePtr("99"),
									ChromeAndroid:  valuePtr("98"),
									Firefox:        valuePtr("97"),
									FirefoxAndroid: valuePtr("96"),
									Edge:           valuePtr("95"),
									Safari:         valuePtr("94"),
									SafariIos:      valuePtr("93"),
								},
							},
						},
					},
					Moved: map[string]web_platform_dx__web_features.FeatureMovedData{
						"feature3": {
							Kind:           "moved",
							RedirectTarget: "feature4",
						},
					},
					Split: map[string]web_platform_dx__web_features.FeatureSplitData{
						"feature2": {
							Kind:            "split",
							RedirectTargets: []string{"feature1", "feature3"},
						},
					},
				},
				Groups: map[string]web_platform_dx__web_features.GroupData{
					"group1": {
						Name:   "Group 1",
						Parent: nil,
					},
				},
				Snapshots: map[string]web_platform_dx__web_features.SnapshotData{
					"snapshot1": {
						Name: "Snapshot 1",
						Spec: "spec1",
					},
				},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			value, err := postProcessV3(tc.featureData)
			if diff := cmp.Diff(tc.expectedValue, value); diff != "" {
				t.Errorf("postProcess unexpected output (-want +got):\n%s", diff)
			}
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("postProcessV3 unexpected error expected %v received %v", tc.expectedErr, err)
			}
		})
	}
}
