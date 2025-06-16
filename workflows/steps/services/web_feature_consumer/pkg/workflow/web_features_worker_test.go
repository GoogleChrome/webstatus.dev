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

package workflow

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
)

var (
	errTestFailToGetAsset       = errors.New("fail to get asset")
	errTestCannotParseData      = errors.New("cannot parse data")
	errTestFailToStoreData      = errors.New("fail to store data")
	errTestFailToStoreMetadata  = errors.New("fail to store metadata")
	errTestFailToStoreGroups    = errors.New("fail to store groups")
	errTestFailToStoreSnapshots = errors.New("fail to store snapshots")
)

type mockAssetGetter struct {
	t                              *testing.T
	mockDownloadFileFromReleaseCfg mockDownloadFileFromReleaseConfig
}

type mockDownloadFileFromReleaseConfig struct {
	expectedFileName string
	expectedOwner    string
	expectedRepo     string
	returnReadCloser io.ReadCloser
	returnError      error
}

func (m *mockAssetGetter) DownloadFileFromRelease(
	_ context.Context, owner, repo string, _ *http.Client, filePattern string) (io.ReadCloser, error) {
	if filePattern != m.mockDownloadFileFromReleaseCfg.expectedFileName ||
		owner != m.mockDownloadFileFromReleaseCfg.expectedOwner ||
		repo != m.mockDownloadFileFromReleaseCfg.expectedRepo {
		m.t.Error("unexpected input to DownloadFileFromRelease")
	}

	return m.mockDownloadFileFromReleaseCfg.returnReadCloser, m.mockDownloadFileFromReleaseCfg.returnError
}

type mockAssetParser struct {
	t            *testing.T
	mockParseCfg mockParseConfig
}

type mockParseConfig struct {
	expectedFileContents string
	returnData           *web_platform_dx__web_features.FeatureData
	returnError          error
}

func (m *mockAssetParser) Parse(file io.ReadCloser) (*web_platform_dx__web_features.FeatureData, error) {
	defer file.Close()
	fileContents, err := io.ReadAll(file)
	if err != nil {
		m.t.Errorf("unable to read file")
	}
	if string(fileContents) != m.mockParseCfg.expectedFileContents {
		m.t.Error("unexpected file contents")
	}

	return m.mockParseCfg.returnData, m.mockParseCfg.returnError
}

// nolint:gochecknoglobals
var (
	testInsertWebFeaturesStartAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	testInsertWebFeaturesEndAt   = time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC)
	fakeBrowsersData             = web_platform_dx__web_features.Browsers{
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
)

type mockInsertWebFeaturesConfig struct {
	expectedData    map[string]web_platform_dx__web_features.FeatureValue
	returnedMapping map[string]string
	returnError     error
}

type mockWebFeatureStorer struct {
	t                        *testing.T
	mockInsertWebFeaturesCfg mockInsertWebFeaturesConfig
}

func (m *mockWebFeatureStorer) InsertWebFeatures(
	_ context.Context, data map[string]web_platform_dx__web_features.FeatureValue,
	startAt, endAt time.Time) (map[string]string, error) {
	if !reflect.DeepEqual(data, m.mockInsertWebFeaturesCfg.expectedData) {
		m.t.Error("unexpected data")
	}
	if !startAt.Equal(testInsertWebFeaturesStartAt) {
		m.t.Errorf("unexpected startAt time %s", startAt)
	}
	if !endAt.Equal(testInsertWebFeaturesEndAt) {
		m.t.Errorf("unexpected endAt time %s", endAt)
	}

	return m.mockInsertWebFeaturesCfg.returnedMapping, m.mockInsertWebFeaturesCfg.returnError
}

type mockInsertWebFeaturesMetadataConfig struct {
	expectedData    map[string]web_platform_dx__web_features.FeatureValue
	expectedMapping map[string]string
	returnError     error
}

type mockWebFeatureMetadataStorer struct {
	t                                *testing.T
	mockInsertWebFeaturesMetadataCfg mockInsertWebFeaturesMetadataConfig
}

func (m *mockWebFeatureMetadataStorer) InsertWebFeaturesMetadata(
	_ context.Context,
	featureKeyToID map[string]string,
	data map[string]web_platform_dx__web_features.FeatureValue) error {
	if !reflect.DeepEqual(data, m.mockInsertWebFeaturesMetadataCfg.expectedData) ||
		!reflect.DeepEqual(featureKeyToID, m.mockInsertWebFeaturesMetadataCfg.expectedMapping) {
		m.t.Error("unexpected input")
	}

	return m.mockInsertWebFeaturesMetadataCfg.returnError
}

type mockInsertWebFeatureGroupsConfig struct {
	expectedFeatureData map[string]web_platform_dx__web_features.FeatureValue
	expectedGroupData   map[string]web_platform_dx__web_features.GroupData
	expectedMapping     map[string]string
	returnError         error
}

type mockWebFeatureGroupStorer struct {
	t                             *testing.T
	mockInsertWebFeatureGroupsCfg mockInsertWebFeatureGroupsConfig
}

func (m *mockWebFeatureGroupStorer) InsertWebFeatureGroups(
	_ context.Context,
	featureKeyToID map[string]string,
	featureData map[string]web_platform_dx__web_features.FeatureValue,
	groupData map[string]web_platform_dx__web_features.GroupData) error {
	if !reflect.DeepEqual(featureData, m.mockInsertWebFeatureGroupsCfg.expectedFeatureData) ||
		!reflect.DeepEqual(groupData, m.mockInsertWebFeatureGroupsCfg.expectedGroupData) ||
		!reflect.DeepEqual(featureKeyToID, m.mockInsertWebFeatureGroupsCfg.expectedMapping) {
		m.t.Error("unexpected input")
	}

	return m.mockInsertWebFeatureGroupsCfg.returnError
}

type mockInsertWebFeatureSnapshotsConfig struct {
	expectedFeatureData  map[string]web_platform_dx__web_features.FeatureValue
	expectedSnapshotData map[string]web_platform_dx__web_features.SnapshotData
	expectedMapping      map[string]string
	returnError          error
}

type mockWebFeatureSnapshotStorer struct {
	t                               *testing.T
	mockInsertWebFeatureSnapshotCfg mockInsertWebFeatureSnapshotsConfig
}

func (m *mockWebFeatureSnapshotStorer) InsertWebFeatureSnapshots(
	_ context.Context,
	featureKeyToID map[string]string,
	featureData map[string]web_platform_dx__web_features.FeatureValue,
	snapshotData map[string]web_platform_dx__web_features.SnapshotData) error {
	if !reflect.DeepEqual(featureData, m.mockInsertWebFeatureSnapshotCfg.expectedFeatureData) ||
		!reflect.DeepEqual(snapshotData, m.mockInsertWebFeatureSnapshotCfg.expectedSnapshotData) ||
		!reflect.DeepEqual(featureKeyToID, m.mockInsertWebFeatureSnapshotCfg.expectedMapping) {
		m.t.Error("unexpected input")
	}

	return m.mockInsertWebFeatureSnapshotCfg.returnError
}

const (
	testRepoOwner = "owner"
	testRepoName  = "name"
	testFileName  = "file.txt"
)

func TestProcess(t *testing.T) {
	// nolint: dupl
	testCases := []struct {
		name                             string
		mockDownloadFileFromReleaseCfg   mockDownloadFileFromReleaseConfig
		mockParseCfg                     mockParseConfig
		mockInsertWebFeaturesCfg         mockInsertWebFeaturesConfig
		mockInsertWebFeaturesMetadataCfg mockInsertWebFeaturesMetadataConfig
		mockInsertWebFeatureGroupsCfg    mockInsertWebFeatureGroupsConfig
		mockInsertWebFeatureSnapshotsCfg mockInsertWebFeatureSnapshotsConfig
		expectedError                    error
	}{
		{
			name: "success",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnReadCloser: io.NopCloser(strings.NewReader("hi features")),
				returnError:      nil,
			},
			mockParseCfg: mockParseConfig{
				expectedFileContents: "hi features",
				returnData: &web_platform_dx__web_features.FeatureData{
					Browsers: fakeBrowsersData,
					Features: map[string]web_platform_dx__web_features.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged:    nil,
							Spec:           nil,
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
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
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
							Spec: "",
						},
					},
				},
				returnError: nil,
			},
			mockInsertWebFeaturesCfg: mockInsertWebFeaturesConfig{
				expectedData: map[string]web_platform_dx__web_features.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
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
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				returnedMapping: map[string]string{
					"feature1": "id-1",
				},
				returnError: nil,
			},
			mockInsertWebFeaturesMetadataCfg: mockInsertWebFeaturesMetadataConfig{
				expectedData: map[string]web_platform_dx__web_features.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
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
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				expectedMapping: map[string]string{
					"feature1": "id-1",
				},
				returnError: nil,
			},
			mockInsertWebFeatureGroupsCfg: mockInsertWebFeatureGroupsConfig{
				expectedFeatureData: map[string]web_platform_dx__web_features.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
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
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				expectedMapping: map[string]string{
					"feature1": "id-1",
				},
				expectedGroupData: map[string]web_platform_dx__web_features.GroupData{
					"group1": {
						Name:   "Group 1",
						Parent: nil,
					},
				},
				returnError: nil,
			},
			mockInsertWebFeatureSnapshotsCfg: mockInsertWebFeatureSnapshotsConfig{
				expectedFeatureData: map[string]web_platform_dx__web_features.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
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
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				expectedMapping: map[string]string{
					"feature1": "id-1",
				},
				expectedSnapshotData: map[string]web_platform_dx__web_features.SnapshotData{
					"snapshot1": {
						Name: "Snapshot 1",
						Spec: "",
					},
				},
				returnError: nil,
			},
			expectedError: nil,
		},
		{
			name: "fail to get asset data",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnReadCloser: io.NopCloser(strings.NewReader("hi features")),
				returnError:      errTestFailToGetAsset,
			},
			mockParseCfg: mockParseConfig{
				expectedFileContents: "",
				returnData:           nil,
				returnError:          nil,
			},
			mockInsertWebFeaturesCfg: mockInsertWebFeaturesConfig{
				expectedData:    nil,
				returnedMapping: nil,
				returnError:     nil,
			},
			mockInsertWebFeaturesMetadataCfg: mockInsertWebFeaturesMetadataConfig{
				expectedData:    nil,
				expectedMapping: nil,
				returnError:     nil,
			},
			mockInsertWebFeatureGroupsCfg: mockInsertWebFeatureGroupsConfig{
				expectedFeatureData: nil,
				expectedGroupData:   nil,
				expectedMapping:     nil,
				returnError:         nil,
			},
			mockInsertWebFeatureSnapshotsCfg: mockInsertWebFeatureSnapshotsConfig{
				expectedFeatureData:  nil,
				expectedSnapshotData: nil,
				expectedMapping:      nil,
				returnError:          nil,
			},
			expectedError: errTestFailToGetAsset,
		},
		{
			name: "fail to parse data",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnReadCloser: io.NopCloser(strings.NewReader("hi features")),
				returnError:      nil,
			},
			mockParseCfg: mockParseConfig{
				expectedFileContents: "hi features",
				returnData:           nil,
				returnError:          errTestCannotParseData,
			},
			mockInsertWebFeaturesCfg: mockInsertWebFeaturesConfig{
				expectedData:    nil,
				returnedMapping: nil,
				returnError:     nil,
			},
			mockInsertWebFeaturesMetadataCfg: mockInsertWebFeaturesMetadataConfig{
				expectedData:    nil,
				expectedMapping: nil,
				returnError:     nil,
			},
			mockInsertWebFeatureGroupsCfg: mockInsertWebFeatureGroupsConfig{
				expectedFeatureData: nil,
				expectedGroupData:   nil,
				expectedMapping:     nil,
				returnError:         nil,
			},
			mockInsertWebFeatureSnapshotsCfg: mockInsertWebFeatureSnapshotsConfig{
				expectedFeatureData:  nil,
				expectedSnapshotData: nil,
				expectedMapping:      nil,
				returnError:          nil,
			},
			expectedError: errTestCannotParseData,
		},
		{
			name: "fail to store data",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnReadCloser: io.NopCloser(strings.NewReader("hi features")),
				returnError:      nil,
			},
			mockParseCfg: mockParseConfig{
				expectedFileContents: "hi features",
				returnData: &web_platform_dx__web_features.FeatureData{
					Browsers: fakeBrowsersData,
					Features: map[string]web_platform_dx__web_features.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged:    nil,
							Spec:           nil,
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
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
						},
					},
					Groups:    nil,
					Snapshots: nil,
				},
				returnError: nil,
			},
			mockInsertWebFeaturesCfg: mockInsertWebFeaturesConfig{
				expectedData: map[string]web_platform_dx__web_features.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
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
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				returnedMapping: map[string]string{
					"feature1": "id-1",
				},
				returnError: errTestFailToStoreData,
			},
			mockInsertWebFeaturesMetadataCfg: mockInsertWebFeaturesMetadataConfig{
				expectedData:    nil,
				expectedMapping: nil,
				returnError:     nil,
			},
			mockInsertWebFeatureGroupsCfg: mockInsertWebFeatureGroupsConfig{
				expectedFeatureData: nil,
				expectedGroupData:   nil,
				expectedMapping:     nil,
				returnError:         nil,
			},
			mockInsertWebFeatureSnapshotsCfg: mockInsertWebFeatureSnapshotsConfig{
				expectedFeatureData:  nil,
				expectedSnapshotData: nil,
				expectedMapping:      nil,
				returnError:          nil,
			},
			expectedError: errTestFailToStoreData,
		},
		{
			name: "fail to store metadata",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnReadCloser: io.NopCloser(strings.NewReader("hi features")),
				returnError:      nil,
			},
			mockParseCfg: mockParseConfig{
				expectedFileContents: "hi features",
				returnData: &web_platform_dx__web_features.FeatureData{
					Browsers: fakeBrowsersData,
					Features: map[string]web_platform_dx__web_features.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged:    nil,
							Spec:           nil,
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
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
						},
					},
					Groups:    nil,
					Snapshots: nil,
				},
				returnError: nil,
			},
			mockInsertWebFeaturesCfg: mockInsertWebFeaturesConfig{
				expectedData: map[string]web_platform_dx__web_features.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
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
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				returnedMapping: map[string]string{
					"feature1": "id-1",
				},
				returnError: nil,
			},
			mockInsertWebFeaturesMetadataCfg: mockInsertWebFeaturesMetadataConfig{
				expectedData: map[string]web_platform_dx__web_features.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
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
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				expectedMapping: map[string]string{
					"feature1": "id-1",
				},
				returnError: errTestFailToStoreMetadata,
			},
			mockInsertWebFeatureGroupsCfg: mockInsertWebFeatureGroupsConfig{
				expectedFeatureData: nil,
				expectedGroupData:   nil,
				expectedMapping:     nil,
				returnError:         nil,
			},
			mockInsertWebFeatureSnapshotsCfg: mockInsertWebFeatureSnapshotsConfig{
				expectedFeatureData:  nil,
				expectedSnapshotData: nil,
				expectedMapping:      nil,
				returnError:          nil,
			},
			expectedError: errTestFailToStoreMetadata,
		},
		{
			name: "fail to store groups",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnReadCloser: io.NopCloser(strings.NewReader("hi features")),
				returnError:      nil,
			},
			mockParseCfg: mockParseConfig{
				expectedFileContents: "hi features",
				returnData: &web_platform_dx__web_features.FeatureData{
					Browsers: fakeBrowsersData,
					Features: map[string]web_platform_dx__web_features.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged:    nil,
							Spec:           nil,
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
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
						},
					},
					Groups: map[string]web_platform_dx__web_features.GroupData{
						"group1": {
							Name:   "Group 1",
							Parent: nil,
						},
					},
					Snapshots: nil,
				},
				returnError: nil,
			},
			mockInsertWebFeaturesCfg: mockInsertWebFeaturesConfig{
				expectedData: map[string]web_platform_dx__web_features.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
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
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				returnedMapping: map[string]string{
					"feature1": "id-1",
				},
				returnError: nil,
			},
			mockInsertWebFeaturesMetadataCfg: mockInsertWebFeaturesMetadataConfig{
				expectedData: map[string]web_platform_dx__web_features.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
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
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				expectedMapping: map[string]string{
					"feature1": "id-1",
				},
				returnError: nil,
			},
			mockInsertWebFeatureGroupsCfg: mockInsertWebFeatureGroupsConfig{
				expectedFeatureData: map[string]web_platform_dx__web_features.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
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
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				expectedMapping: map[string]string{
					"feature1": "id-1",
				},
				expectedGroupData: map[string]web_platform_dx__web_features.GroupData{
					"group1": {
						Name:   "Group 1",
						Parent: nil,
					},
				},
				returnError: errTestFailToStoreGroups,
			},
			mockInsertWebFeatureSnapshotsCfg: mockInsertWebFeatureSnapshotsConfig{
				expectedFeatureData:  nil,
				expectedSnapshotData: nil,
				expectedMapping:      nil,
				returnError:          nil,
			},
			expectedError: errTestFailToStoreGroups,
		},
		{
			name: "fail to store snapshots",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnReadCloser: io.NopCloser(strings.NewReader("hi features")),
				returnError:      nil,
			},
			mockParseCfg: mockParseConfig{
				expectedFileContents: "hi features",
				returnData: &web_platform_dx__web_features.FeatureData{
					Browsers: fakeBrowsersData,
					Features: map[string]web_platform_dx__web_features.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Discouraged:    nil,
							Spec:           nil,
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
							Description:     "text",
							DescriptionHTML: "<html>",
							Group:           nil,
							Snapshot:        nil,
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
							Spec: "",
						},
					},
				},
				returnError: nil,
			},
			mockInsertWebFeaturesCfg: mockInsertWebFeaturesConfig{
				expectedData: map[string]web_platform_dx__web_features.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
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
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				returnedMapping: map[string]string{
					"feature1": "id-1",
				},
				returnError: nil,
			},
			mockInsertWebFeaturesMetadataCfg: mockInsertWebFeaturesMetadataConfig{
				expectedData: map[string]web_platform_dx__web_features.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
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
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				expectedMapping: map[string]string{
					"feature1": "id-1",
				},
				returnError: nil,
			},
			mockInsertWebFeatureGroupsCfg: mockInsertWebFeatureGroupsConfig{
				expectedFeatureData: map[string]web_platform_dx__web_features.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
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
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				expectedMapping: map[string]string{
					"feature1": "id-1",
				},
				expectedGroupData: map[string]web_platform_dx__web_features.GroupData{
					"group1": {
						Name:   "Group 1",
						Parent: nil,
					},
				},
				returnError: nil,
			},
			mockInsertWebFeatureSnapshotsCfg: mockInsertWebFeatureSnapshotsConfig{
				expectedFeatureData: map[string]web_platform_dx__web_features.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
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
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				expectedMapping: map[string]string{
					"feature1": "id-1",
				},
				expectedSnapshotData: map[string]web_platform_dx__web_features.SnapshotData{
					"snapshot1": {
						Name: "Snapshot 1",
						Spec: "",
					},
				},
				returnError: errTestFailToStoreSnapshots,
			},
			expectedError: errTestFailToStoreSnapshots,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockGetter := &mockAssetGetter{
				t:                              t,
				mockDownloadFileFromReleaseCfg: tc.mockDownloadFileFromReleaseCfg,
			}
			mockParser := &mockAssetParser{
				t:            t,
				mockParseCfg: tc.mockParseCfg,
			}
			mockStorer := &mockWebFeatureStorer{
				t:                        t,
				mockInsertWebFeaturesCfg: tc.mockInsertWebFeaturesCfg,
			}
			mockMetadataStorer := &mockWebFeatureMetadataStorer{
				t:                                t,
				mockInsertWebFeaturesMetadataCfg: tc.mockInsertWebFeaturesMetadataCfg,
			}

			mockGroupStorer := &mockWebFeatureGroupStorer{
				t:                             t,
				mockInsertWebFeatureGroupsCfg: tc.mockInsertWebFeatureGroupsCfg,
			}

			mockSnapshotStorer := &mockWebFeatureSnapshotStorer{
				t:                               t,
				mockInsertWebFeatureSnapshotCfg: tc.mockInsertWebFeatureSnapshotsCfg,
			}

			processor := NewWebFeaturesJobProcessor(
				mockGetter,
				mockStorer,
				mockMetadataStorer,
				mockGroupStorer,
				mockSnapshotStorer,
				mockParser,
			)

			err := processor.Process(context.TODO(), NewJobArguments(
				testFileName,
				testRepoOwner,
				testRepoName,
				testInsertWebFeaturesStartAt,
				testInsertWebFeaturesEndAt,
			))
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Expected error: %v, Got: %v", tc.expectedError, err)
			}
		})
	}
}
