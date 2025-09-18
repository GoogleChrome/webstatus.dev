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

	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features_v3"
	"github.com/GoogleChrome/webstatus.dev/lib/webdxfeaturetypes"
	"github.com/google/go-cmp/cmp"
)

func TestParseV3(t *testing.T) {
	testCases := []struct {
		name string
		path string
	}{
		{
			name: "data.json from https://github.com/web-platform-dx/web-features/releases/tag/v3.0.0",
			path: path.Join("testdata", "v3.data.json"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			file, err := os.Open(tc.path)
			if err != nil {
				t.Fatalf("unable to read file err %s", err.Error())
			}
			result, err := V3Parser{}.Parse(file)
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

func testBrowsersV3() web_platform_dx__web_features_v3.Browsers {
	return web_platform_dx__web_features_v3.Browsers{
		Chrome: web_platform_dx__web_features_v3.BrowserData{
			Name:     "chrome",
			Releases: nil,
		},
		ChromeAndroid: web_platform_dx__web_features_v3.BrowserData{
			Name:     "chrome_android",
			Releases: nil,
		},
		Edge: web_platform_dx__web_features_v3.BrowserData{
			Name:     "edge",
			Releases: nil,
		},
		Firefox: web_platform_dx__web_features_v3.BrowserData{
			Name:     "firefox",
			Releases: nil,
		},
		FirefoxAndroid: web_platform_dx__web_features_v3.BrowserData{
			Name:     "firefox_android",
			Releases: nil,
		},
		Safari: web_platform_dx__web_features_v3.BrowserData{
			Name:     "safari",
			Releases: nil,
		},
		SafariIos: web_platform_dx__web_features_v3.BrowserData{
			Name:     "safari_ios",
			Releases: nil,
		},
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
				Browsers: testBrowsersV3(),
				Groups: map[string]web_platform_dx__web_features_v3.GroupData{
					"group1": {
						Name:   "Group 1",
						Parent: nil,
					},
				},
				Snapshots: map[string]web_platform_dx__web_features_v3.SnapshotData{
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
		"caniuse": [
			"caniuse_data"
		],
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
				Browsers: webdxfeaturetypes.Browsers{
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
				},
				Features: &webdxfeaturetypes.FeatureKinds{
					Data: map[string]webdxfeaturetypes.FeatureValue{
						"feature1": {
							CompatFeatures:  []string{"compat1", "compat2"},
							Description:     "description",
							DescriptionHTML: "description html",
							Discouraged: &webdxfeaturetypes.Discouraged{
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
							Caniuse: []string{
								"caniuse_data",
							},
							Group: []string{
								"group1",
								"group2",
							},
							Snapshot: []string{
								"snapshot1",
								"snapshot2",
							},
							Spec: []string{
								"spec1",
								"spec2",
							},
							Status: webdxfeaturetypes.Status{
								ByCompatKey: nil,
								Baseline: &webdxfeaturetypes.BaselineUnion{
									Bool: nil,
									Enum: valuePtr(webdxfeaturetypes.High),
								},
								BaselineHighDate: valuePtr("2023-01-01"),
								BaselineLowDate:  valuePtr("2022-12-01"),
								Support: webdxfeaturetypes.StatusSupport{
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
					Moved: map[string]webdxfeaturetypes.FeatureMovedData{
						"feature3": {
							Kind:           "moved",
							RedirectTarget: "feature4",
						},
					},
					Split: map[string]webdxfeaturetypes.FeatureSplitData{
						"feature2": {
							Kind:            "split",
							RedirectTargets: []string{"feature1", "feature3"},
						},
					},
				},
				Groups: map[string]webdxfeaturetypes.GroupData{
					"group1": {
						Name:   "Group 1",
						Parent: nil,
					},
				},
				Snapshots: map[string]webdxfeaturetypes.SnapshotData{
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
