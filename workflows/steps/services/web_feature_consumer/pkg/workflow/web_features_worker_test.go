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

	"github.com/GoogleChrome/webstatus.dev/lib/gh"
	"github.com/GoogleChrome/webstatus.dev/lib/webdxfeaturetypes"
	"github.com/google/go-cmp/cmp"
)

var (
	errTestFailToGetAsset           = errors.New("fail to get asset")
	errTestCannotParseData          = errors.New("cannot parse data")
	errTestFailToStoreData          = errors.New("fail to store data")
	errTestFailToStoreMetadata      = errors.New("fail to store metadata")
	errTestFailToStoreGroups        = errors.New("fail to store groups")
	errTestFailToStoreSnapshots     = errors.New("fail to store snapshots")
	errTestFailToStoreMovedFeatures = errors.New("fail to store moved features")
	errTestFailToStoreSplitFeatures = errors.New("fail to store split features")
)

type mockAssetGetter struct {
	t                              *testing.T
	mockDownloadFileFromReleaseCfg mockDownloadFileFromReleaseConfig
}

type mockDownloadFileFromReleaseConfig struct {
	expectedFileName string
	expectedOwner    string
	expectedRepo     string
	returnFile       *gh.ReleaseFile
	returnError      error
}

func (m *mockAssetGetter) DownloadFileFromRelease(
	_ context.Context, owner, repo string, _ *http.Client, filePattern string) (*gh.ReleaseFile, error) {
	if filePattern != m.mockDownloadFileFromReleaseCfg.expectedFileName ||
		owner != m.mockDownloadFileFromReleaseCfg.expectedOwner ||
		repo != m.mockDownloadFileFromReleaseCfg.expectedRepo {
		m.t.Error("unexpected input to DownloadFileFromRelease")
	}

	return m.mockDownloadFileFromReleaseCfg.returnFile, m.mockDownloadFileFromReleaseCfg.returnError
}

type mockAssetParser struct {
	t            *testing.T
	mockParseCfg mockParseConfig
	callCount    int
}

type mockParseConfig struct {
	expectedFileContents string
	returnData           *webdxfeaturetypes.ProcessedWebFeaturesData
	returnError          error
}

func (m *mockAssetParser) Parse(file io.ReadCloser) (*webdxfeaturetypes.ProcessedWebFeaturesData, error) {
	m.callCount++
	defer file.Close()
	fileContents, err := io.ReadAll(file)
	if err != nil {
		m.t.Errorf("unable to read file")
	}
	if string(fileContents) != m.mockParseCfg.expectedFileContents {
		m.t.Errorf("unexpected file contents want: %s, got: %s",
			m.mockParseCfg.expectedFileContents, string(fileContents))
	}

	return m.mockParseCfg.returnData, m.mockParseCfg.returnError
}

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

type mockInsertWebFeaturesConfig struct {
	expectedData    *webdxfeaturetypes.ProcessedWebFeaturesData
	returnedMapping map[string]string
	returnError     error
}

type mockInsertMovedFeaturesConfig struct {
	expectedData map[string]webdxfeaturetypes.FeatureMovedData
	returnError  error
}

type mockInsertSplitFeaturesConfig struct {
	expectedData map[string]webdxfeaturetypes.FeatureSplitData
	returnError  error
}

type mockWebFeatureStorer struct {
	t                             *testing.T
	mockInsertWebFeaturesCfg      mockInsertWebFeaturesConfig
	mockInsertMovedWebFeaturesCfg *mockInsertMovedFeaturesConfig
	mockInsertSplitWebFeaturesCfg *mockInsertSplitFeaturesConfig
}

func (m *mockWebFeatureStorer) InsertMovedWebFeatures(
	_ context.Context, data map[string]webdxfeaturetypes.FeatureMovedData) error {
	if !reflect.DeepEqual(data, m.mockInsertMovedWebFeaturesCfg.expectedData) {
		m.t.Error("unexpected data")
	}

	return m.mockInsertMovedWebFeaturesCfg.returnError
}

func (m *mockWebFeatureStorer) InsertSplitWebFeatures(
	_ context.Context, data map[string]webdxfeaturetypes.FeatureSplitData) error {
	if !reflect.DeepEqual(data, m.mockInsertSplitWebFeaturesCfg.expectedData) {
		m.t.Error("unexpected data")
	}

	return m.mockInsertSplitWebFeaturesCfg.returnError
}

func (m *mockWebFeatureStorer) InsertWebFeatures(
	_ context.Context, data *webdxfeaturetypes.ProcessedWebFeaturesData,
	startAt, endAt time.Time) (map[string]string, error) {
	if diff := cmp.Diff(m.mockInsertWebFeaturesCfg.expectedData, data); diff != "" {
		m.t.Errorf("unexpected data (-want +got):\n%s", diff)
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
	expectedData    map[string]webdxfeaturetypes.FeatureValue
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
	data map[string]webdxfeaturetypes.FeatureValue) error {
	if !reflect.DeepEqual(data, m.mockInsertWebFeaturesMetadataCfg.expectedData) ||
		!reflect.DeepEqual(featureKeyToID, m.mockInsertWebFeaturesMetadataCfg.expectedMapping) {
		m.t.Error("unexpected input")
	}

	return m.mockInsertWebFeaturesMetadataCfg.returnError
}

type mockInsertWebFeatureGroupsConfig struct {
	expectedFeatureData map[string]webdxfeaturetypes.FeatureValue
	expectedGroupData   map[string]webdxfeaturetypes.GroupData
	returnError         error
}

type mockWebFeatureGroupStorer struct {
	t                             *testing.T
	mockInsertWebFeatureGroupsCfg mockInsertWebFeatureGroupsConfig
}

func (m *mockWebFeatureGroupStorer) InsertWebFeatureGroups(
	_ context.Context,
	featureData map[string]webdxfeaturetypes.FeatureValue,
	groupData map[string]webdxfeaturetypes.GroupData) error {
	if !reflect.DeepEqual(featureData, m.mockInsertWebFeatureGroupsCfg.expectedFeatureData) ||
		!reflect.DeepEqual(groupData, m.mockInsertWebFeatureGroupsCfg.expectedGroupData) {
		m.t.Error("unexpected input")
	}

	return m.mockInsertWebFeatureGroupsCfg.returnError
}

type mockInsertWebFeatureSnapshotsConfig struct {
	expectedFeatureData  map[string]webdxfeaturetypes.FeatureValue
	expectedSnapshotData map[string]webdxfeaturetypes.SnapshotData
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
	featureData map[string]webdxfeaturetypes.FeatureValue,
	snapshotData map[string]webdxfeaturetypes.SnapshotData) error {
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

func testFile(tag *string, contents string) *gh.ReleaseFile {
	return &gh.ReleaseFile{
		Contents: io.NopCloser(strings.NewReader(contents)),
		Info: gh.ReleaseInfo{
			Tag: tag,
		},
	}
}

func TestProcess(t *testing.T) {
	// According https://pkg.go.dev/golang.org/x/mod/semver, the version must start with v
	testFileFn := func() *gh.ReleaseFile {
		return testFile(valuePtr(v2), "hi features")
	}
	// nolint: dupl
	testCases := []struct {
		name                             string
		mockDownloadFileFromReleaseCfg   mockDownloadFileFromReleaseConfig
		mockParseCfg                     mockParseConfig
		mockInsertWebFeaturesCfg         mockInsertWebFeaturesConfig
		mockInsertWebFeaturesMetadataCfg mockInsertWebFeaturesMetadataConfig
		mockInsertWebFeatureGroupsCfg    mockInsertWebFeatureGroupsConfig
		mockInsertWebFeatureSnapshotsCfg mockInsertWebFeatureSnapshotsConfig
		mockInsertMovedFeaturesCfg       *mockInsertMovedFeaturesConfig
		mockInsertSplitFeaturesCfg       *mockInsertSplitFeaturesConfig
		expectedError                    error
	}{
		{
			name: "success",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnFile:       testFileFn(),
				returnError:      nil,
			},
			mockParseCfg: mockParseConfig{
				expectedFileContents: "hi features",
				returnData: &webdxfeaturetypes.ProcessedWebFeaturesData{
					Browsers: fakeBrowsersData,
					Features: &webdxfeaturetypes.FeatureKinds{
						Data: map[string]webdxfeaturetypes.FeatureValue{
							"feature1": {
								Name:           "Feature 1",
								Caniuse:        nil,
								CompatFeatures: nil,
								Discouraged:    nil,
								Spec:           nil,
								Status: webdxfeaturetypes.Status{
									Baseline:         nil,
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
								},
								Description:     "text",
								DescriptionHTML: "<html>",
								Group:           nil,
								Snapshot:        nil,
							},
						},
						Moved: map[string]webdxfeaturetypes.FeatureMovedData{
							"movedFeature": {
								Kind:           webdxfeaturetypes.Moved,
								RedirectTarget: "new-feature",
							},
						},
						Split: map[string]webdxfeaturetypes.FeatureSplitData{
							"splitFeature": {
								Kind: webdxfeaturetypes.Split,
								RedirectTargets: []string{
									"new-feature-1",
									"new-feature-2",
								},
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
							Spec: "",
						},
					},
				},
				returnError: nil,
			},
			mockInsertWebFeaturesCfg: mockInsertWebFeaturesConfig{
				expectedData: &webdxfeaturetypes.ProcessedWebFeaturesData{
					Browsers: fakeBrowsersData,
					Groups: map[string]webdxfeaturetypes.GroupData{
						"group1": {
							Name:   "Group 1",
							Parent: nil,
						},
					},
					Snapshots: map[string]webdxfeaturetypes.SnapshotData{
						"snapshot1": {
							Name: "Snapshot 1",
							Spec: "",
						},
					},
					Features: &webdxfeaturetypes.FeatureKinds{
						Data: map[string]webdxfeaturetypes.FeatureValue{
							"feature1": {
								Name:           "Feature 1",
								Caniuse:        nil,
								CompatFeatures: nil,
								Discouraged:    nil,
								Spec:           nil,
								Status: webdxfeaturetypes.Status{
									Baseline:         nil,
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
								},
								Description:     "text",
								DescriptionHTML: "<html>",
								Group:           nil,
								Snapshot:        nil,
							},
						},
						Moved: map[string]webdxfeaturetypes.FeatureMovedData{
							"movedFeature": {
								Kind:           webdxfeaturetypes.Moved,
								RedirectTarget: "new-feature",
							},
						},
						Split: map[string]webdxfeaturetypes.FeatureSplitData{
							"splitFeature": {
								Kind: webdxfeaturetypes.Split,
								RedirectTargets: []string{
									"new-feature-1",
									"new-feature-2",
								},
							},
						},
					},
				},
				returnedMapping: map[string]string{
					"feature1": "id-1",
				},
				returnError: nil,
			},
			mockInsertWebFeaturesMetadataCfg: mockInsertWebFeaturesMetadataConfig{
				expectedData: map[string]webdxfeaturetypes.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
						Status: webdxfeaturetypes.Status{
							Baseline:         nil,
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
				expectedFeatureData: map[string]webdxfeaturetypes.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
						Status: webdxfeaturetypes.Status{
							Baseline:         nil,
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
						},
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				expectedGroupData: map[string]webdxfeaturetypes.GroupData{
					"group1": {
						Name:   "Group 1",
						Parent: nil,
					},
				},
				returnError: nil,
			},
			mockInsertWebFeatureSnapshotsCfg: mockInsertWebFeatureSnapshotsConfig{
				expectedFeatureData: map[string]webdxfeaturetypes.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
						Status: webdxfeaturetypes.Status{
							Baseline:         nil,
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
				expectedSnapshotData: map[string]webdxfeaturetypes.SnapshotData{
					"snapshot1": {
						Name: "Snapshot 1",
						Spec: "",
					},
				},
				returnError: nil,
			},
			mockInsertMovedFeaturesCfg: &mockInsertMovedFeaturesConfig{
				expectedData: map[string]webdxfeaturetypes.FeatureMovedData{
					"movedFeature": {
						Kind:           webdxfeaturetypes.Moved,
						RedirectTarget: "new-feature",
					},
				},
				returnError: nil,
			},
			mockInsertSplitFeaturesCfg: &mockInsertSplitFeaturesConfig{
				expectedData: map[string]webdxfeaturetypes.FeatureSplitData{
					"splitFeature": {
						Kind: webdxfeaturetypes.Split,
						RedirectTargets: []string{
							"new-feature-1",
							"new-feature-2",
						},
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
				returnFile:       testFileFn(),
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
				returnError:         nil,
			},
			mockInsertWebFeatureSnapshotsCfg: mockInsertWebFeatureSnapshotsConfig{
				expectedFeatureData:  nil,
				expectedSnapshotData: nil,
				expectedMapping:      nil,
				returnError:          nil,
			},
			mockInsertMovedFeaturesCfg: nil,
			mockInsertSplitFeaturesCfg: nil,
			expectedError:              errTestFailToGetAsset,
		},
		{
			name: "fail to parse data",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnFile:       testFileFn(),
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
				returnError:         nil,
			},
			mockInsertWebFeatureSnapshotsCfg: mockInsertWebFeatureSnapshotsConfig{
				expectedFeatureData:  nil,
				expectedSnapshotData: nil,
				expectedMapping:      nil,
				returnError:          nil,
			},
			mockInsertMovedFeaturesCfg: nil,
			mockInsertSplitFeaturesCfg: nil,
			expectedError:              errTestCannotParseData,
		},
		{
			name: "fail to store data",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnFile:       testFileFn(),
				returnError:      nil,
			},
			mockParseCfg: mockParseConfig{
				expectedFileContents: "hi features",
				returnData: &webdxfeaturetypes.ProcessedWebFeaturesData{
					Browsers: fakeBrowsersData,
					Features: &webdxfeaturetypes.FeatureKinds{
						Data: map[string]webdxfeaturetypes.FeatureValue{
							"feature1": {
								Name:           "Feature 1",
								Caniuse:        nil,
								CompatFeatures: nil,
								Discouraged:    nil,
								Spec:           nil,
								Status: webdxfeaturetypes.Status{
									Baseline:         nil,
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
					Groups:    nil,
					Snapshots: nil,
				},
				returnError: nil,
			},
			mockInsertWebFeaturesCfg: mockInsertWebFeaturesConfig{
				expectedData: &webdxfeaturetypes.ProcessedWebFeaturesData{
					Browsers:  fakeBrowsersData,
					Groups:    nil,
					Snapshots: nil,
					Features: &webdxfeaturetypes.FeatureKinds{
						Data: map[string]webdxfeaturetypes.FeatureValue{
							"feature1": {
								Name:           "Feature 1",
								Caniuse:        nil,
								CompatFeatures: nil,
								Discouraged:    nil,
								Spec:           nil,
								Status: webdxfeaturetypes.Status{
									Baseline:         nil,
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
				returnError:         nil,
			},
			mockInsertWebFeatureSnapshotsCfg: mockInsertWebFeatureSnapshotsConfig{
				expectedFeatureData:  nil,
				expectedSnapshotData: nil,
				expectedMapping:      nil,
				returnError:          nil,
			},
			mockInsertMovedFeaturesCfg: nil,
			mockInsertSplitFeaturesCfg: nil,
			expectedError:              errTestFailToStoreData,
		},
		{
			name: "fail to store metadata",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnFile:       testFileFn(),
				returnError:      nil,
			},
			mockParseCfg: mockParseConfig{
				expectedFileContents: "hi features",
				returnData: &webdxfeaturetypes.ProcessedWebFeaturesData{
					Browsers: fakeBrowsersData,
					Features: &webdxfeaturetypes.FeatureKinds{
						Data: map[string]webdxfeaturetypes.FeatureValue{
							"feature1": {
								Name:           "Feature 1",
								Caniuse:        nil,
								CompatFeatures: nil,
								Discouraged:    nil,
								Spec:           nil,
								Status: webdxfeaturetypes.Status{
									Baseline:         nil,
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
					Groups:    nil,
					Snapshots: nil,
				},
				returnError: nil,
			},
			mockInsertWebFeaturesCfg: mockInsertWebFeaturesConfig{
				expectedData: &webdxfeaturetypes.ProcessedWebFeaturesData{
					Browsers:  fakeBrowsersData,
					Snapshots: nil,
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
									Baseline:         nil,
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
								},
								Description:     "text",
								DescriptionHTML: "<html>",
								Group:           nil,
								Snapshot:        nil,
							},
						},
						Split: nil,
						Moved: nil,
					},
				},
				returnedMapping: map[string]string{
					"feature1": "id-1",
				},
				returnError: nil,
			},
			mockInsertWebFeaturesMetadataCfg: mockInsertWebFeaturesMetadataConfig{
				expectedData: map[string]webdxfeaturetypes.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
						Status: webdxfeaturetypes.Status{
							Baseline:         nil,
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
				returnError:         nil,
			},
			mockInsertWebFeatureSnapshotsCfg: mockInsertWebFeatureSnapshotsConfig{
				expectedFeatureData:  nil,
				expectedSnapshotData: nil,
				expectedMapping:      nil,
				returnError:          nil,
			},
			mockInsertMovedFeaturesCfg: nil,
			mockInsertSplitFeaturesCfg: nil,
			expectedError:              errTestFailToStoreMetadata,
		},
		{
			name: "fail to store groups",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnFile:       testFileFn(),
				returnError:      nil,
			},
			mockParseCfg: mockParseConfig{
				expectedFileContents: "hi features",
				returnData: &webdxfeaturetypes.ProcessedWebFeaturesData{
					Browsers: fakeBrowsersData,
					Features: &webdxfeaturetypes.FeatureKinds{
						Data: map[string]webdxfeaturetypes.FeatureValue{
							"feature1": {
								Name:           "Feature 1",
								Caniuse:        nil,
								CompatFeatures: nil,
								Discouraged:    nil,
								Spec:           nil,
								Status: webdxfeaturetypes.Status{
									Baseline:         nil,
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
					Groups: map[string]webdxfeaturetypes.GroupData{
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
				expectedData: &webdxfeaturetypes.ProcessedWebFeaturesData{
					Browsers: fakeBrowsersData,
					Groups: map[string]webdxfeaturetypes.GroupData{
						"group1": {
							Name:   "Group 1",
							Parent: nil,
						},
					},
					Snapshots: nil,
					Features: &webdxfeaturetypes.FeatureKinds{
						Data: map[string]webdxfeaturetypes.FeatureValue{
							"feature1": {
								Name:           "Feature 1",
								Caniuse:        nil,
								CompatFeatures: nil,
								Discouraged:    nil,
								Spec:           nil,
								Status: webdxfeaturetypes.Status{
									Baseline:         nil,
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
				returnedMapping: map[string]string{
					"feature1": "id-1",
				},
				returnError: nil,
			},
			mockInsertWebFeaturesMetadataCfg: mockInsertWebFeaturesMetadataConfig{
				expectedData: map[string]webdxfeaturetypes.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
						Status: webdxfeaturetypes.Status{
							Baseline:         nil,
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
				expectedFeatureData: map[string]webdxfeaturetypes.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
						Status: webdxfeaturetypes.Status{
							Baseline:         nil,
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
						},
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				expectedGroupData: map[string]webdxfeaturetypes.GroupData{
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
			mockInsertMovedFeaturesCfg: nil,
			mockInsertSplitFeaturesCfg: nil,
			expectedError:              errTestFailToStoreGroups,
		},
		{
			name: "fail to store snapshots",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnFile:       testFileFn(),
				returnError:      nil,
			},
			mockParseCfg: mockParseConfig{
				expectedFileContents: "hi features",
				returnData: &webdxfeaturetypes.ProcessedWebFeaturesData{
					Browsers: fakeBrowsersData,
					Features: &webdxfeaturetypes.FeatureKinds{
						Data: map[string]webdxfeaturetypes.FeatureValue{
							"feature1": {
								Name:           "Feature 1",
								Caniuse:        nil,
								CompatFeatures: nil,
								Discouraged:    nil,
								Spec:           nil,
								Status: webdxfeaturetypes.Status{
									Baseline:         nil,
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
					Groups: map[string]webdxfeaturetypes.GroupData{
						"group1": {
							Name:   "Group 1",
							Parent: nil,
						},
					},
					Snapshots: map[string]webdxfeaturetypes.SnapshotData{
						"snapshot1": {
							Name: "Snapshot 1",
							Spec: "",
						},
					},
				},
				returnError: nil,
			},
			mockInsertWebFeaturesCfg: mockInsertWebFeaturesConfig{
				expectedData: &webdxfeaturetypes.ProcessedWebFeaturesData{
					Browsers: fakeBrowsersData,
					Groups: map[string]webdxfeaturetypes.GroupData{
						"group1": {
							Name:   "Group 1",
							Parent: nil,
						},
					},
					Snapshots: map[string]webdxfeaturetypes.SnapshotData{
						"snapshot1": {
							Name: "Snapshot 1",
							Spec: "",
						},
					},
					Features: &webdxfeaturetypes.FeatureKinds{
						Data: map[string]webdxfeaturetypes.FeatureValue{
							"feature1": {
								Name:           "Feature 1",
								Caniuse:        nil,
								CompatFeatures: nil,
								Discouraged:    nil,
								Spec:           nil,
								Status: webdxfeaturetypes.Status{
									Baseline:         nil,
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
				returnedMapping: map[string]string{
					"feature1": "id-1",
				},
				returnError: nil,
			},
			mockInsertWebFeaturesMetadataCfg: mockInsertWebFeaturesMetadataConfig{
				expectedData: map[string]webdxfeaturetypes.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
						Status: webdxfeaturetypes.Status{
							Baseline:         nil,
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
				expectedFeatureData: map[string]webdxfeaturetypes.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
						Status: webdxfeaturetypes.Status{
							Baseline:         nil,
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
						},
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				expectedGroupData: map[string]webdxfeaturetypes.GroupData{
					"group1": {
						Name:   "Group 1",
						Parent: nil,
					},
				},
				returnError: nil,
			},
			mockInsertWebFeatureSnapshotsCfg: mockInsertWebFeatureSnapshotsConfig{
				expectedFeatureData: map[string]webdxfeaturetypes.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
						Status: webdxfeaturetypes.Status{
							Baseline:         nil,
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
				expectedSnapshotData: map[string]webdxfeaturetypes.SnapshotData{
					"snapshot1": {
						Name: "Snapshot 1",
						Spec: "",
					},
				},
				returnError: errTestFailToStoreSnapshots,
			},
			mockInsertMovedFeaturesCfg: nil,
			mockInsertSplitFeaturesCfg: nil,
			expectedError:              errTestFailToStoreSnapshots,
		},
		{
			name: "fail to store moved features",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnFile:       testFileFn(),
				returnError:      nil,
			},
			mockParseCfg: mockParseConfig{
				expectedFileContents: "hi features",
				returnData: &webdxfeaturetypes.ProcessedWebFeaturesData{
					Browsers: fakeBrowsersData,
					Features: &webdxfeaturetypes.FeatureKinds{
						Data: map[string]webdxfeaturetypes.FeatureValue{
							"feature1": {
								Name:           "Feature 1",
								Caniuse:        nil,
								CompatFeatures: nil,
								Discouraged:    nil,
								Spec:           nil,
								Status: webdxfeaturetypes.Status{
									Baseline:         nil,
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
								},
								Description:     "text",
								DescriptionHTML: "<html>",
								Group:           nil,
								Snapshot:        nil,
							},
						},
						Moved: map[string]webdxfeaturetypes.FeatureMovedData{
							"movedFeature": {
								Kind:           webdxfeaturetypes.Moved,
								RedirectTarget: "new-feature",
							},
						},
						Split: map[string]webdxfeaturetypes.FeatureSplitData{
							"splitFeature": {
								Kind: webdxfeaturetypes.Split,
								RedirectTargets: []string{
									"new-feature-1",
									"new-feature-2",
								},
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
							Spec: "",
						},
					},
				},
				returnError: nil,
			},
			mockInsertWebFeaturesCfg: mockInsertWebFeaturesConfig{
				expectedData: &webdxfeaturetypes.ProcessedWebFeaturesData{
					Browsers: fakeBrowsersData,
					Groups: map[string]webdxfeaturetypes.GroupData{
						"group1": {
							Name:   "Group 1",
							Parent: nil,
						},
					},
					Snapshots: map[string]webdxfeaturetypes.SnapshotData{
						"snapshot1": {
							Name: "Snapshot 1",
							Spec: "",
						},
					},
					Features: &webdxfeaturetypes.FeatureKinds{
						Data: map[string]webdxfeaturetypes.FeatureValue{
							"feature1": {
								Name:           "Feature 1",
								Caniuse:        nil,
								CompatFeatures: nil,
								Discouraged:    nil,
								Spec:           nil,
								Status: webdxfeaturetypes.Status{
									Baseline:         nil,
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
								},
								Description:     "text",
								DescriptionHTML: "<html>",
								Group:           nil,
								Snapshot:        nil,
							},
						},
						Moved: map[string]webdxfeaturetypes.FeatureMovedData{
							"movedFeature": {
								Kind:           webdxfeaturetypes.Moved,
								RedirectTarget: "new-feature",
							},
						},
						Split: map[string]webdxfeaturetypes.FeatureSplitData{
							"splitFeature": {
								Kind: webdxfeaturetypes.Split,
								RedirectTargets: []string{
									"new-feature-1",
									"new-feature-2",
								},
							},
						},
					},
				},
				returnedMapping: map[string]string{
					"feature1": "id-1",
				},
				returnError: nil,
			},
			mockInsertWebFeaturesMetadataCfg: mockInsertWebFeaturesMetadataConfig{
				expectedData: map[string]webdxfeaturetypes.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
						Status: webdxfeaturetypes.Status{
							Baseline:         nil,
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
				expectedFeatureData: map[string]webdxfeaturetypes.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
						Status: webdxfeaturetypes.Status{
							Baseline:         nil,
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
						},
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				expectedGroupData: map[string]webdxfeaturetypes.GroupData{
					"group1": {
						Name:   "Group 1",
						Parent: nil,
					},
				},
				returnError: nil,
			},
			mockInsertWebFeatureSnapshotsCfg: mockInsertWebFeatureSnapshotsConfig{
				expectedFeatureData: map[string]webdxfeaturetypes.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
						Status: webdxfeaturetypes.Status{
							Baseline:         nil,
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
				expectedSnapshotData: map[string]webdxfeaturetypes.SnapshotData{
					"snapshot1": {
						Name: "Snapshot 1",
						Spec: "",
					},
				},
				returnError: nil,
			},
			mockInsertMovedFeaturesCfg: &mockInsertMovedFeaturesConfig{
				expectedData: map[string]webdxfeaturetypes.FeatureMovedData{
					"movedFeature": {
						Kind:           webdxfeaturetypes.Moved,
						RedirectTarget: "new-feature",
					},
				},
				returnError: errTestFailToStoreMovedFeatures,
			},
			mockInsertSplitFeaturesCfg: nil,
			expectedError:              errTestFailToStoreMovedFeatures,
		},
		{
			name: "fail to store split features",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnFile:       testFileFn(),
				returnError:      nil,
			},
			mockParseCfg: mockParseConfig{
				expectedFileContents: "hi features",
				returnData: &webdxfeaturetypes.ProcessedWebFeaturesData{
					Browsers: fakeBrowsersData,
					Features: &webdxfeaturetypes.FeatureKinds{
						Data: map[string]webdxfeaturetypes.FeatureValue{
							"feature1": {
								Name:           "Feature 1",
								Caniuse:        nil,
								CompatFeatures: nil,
								Discouraged:    nil,
								Spec:           nil,
								Status: webdxfeaturetypes.Status{
									Baseline:         nil,
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
								},
								Description:     "text",
								DescriptionHTML: "<html>",
								Group:           nil,
								Snapshot:        nil,
							},
						},
						Moved: map[string]webdxfeaturetypes.FeatureMovedData{
							"movedFeature": {
								Kind:           webdxfeaturetypes.Moved,
								RedirectTarget: "new-feature",
							},
						},
						Split: map[string]webdxfeaturetypes.FeatureSplitData{
							"splitFeature": {
								Kind: webdxfeaturetypes.Split,
								RedirectTargets: []string{
									"new-feature-1",
									"new-feature-2",
								},
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
							Spec: "",
						},
					},
				},
				returnError: nil,
			},
			mockInsertWebFeaturesCfg: mockInsertWebFeaturesConfig{
				expectedData: &webdxfeaturetypes.ProcessedWebFeaturesData{
					Browsers: fakeBrowsersData,
					Groups: map[string]webdxfeaturetypes.GroupData{
						"group1": {
							Name:   "Group 1",
							Parent: nil,
						},
					},
					Snapshots: map[string]webdxfeaturetypes.SnapshotData{
						"snapshot1": {
							Name: "Snapshot 1",
							Spec: "",
						},
					},
					Features: &webdxfeaturetypes.FeatureKinds{
						Data: map[string]webdxfeaturetypes.FeatureValue{
							"feature1": {
								Name:           "Feature 1",
								Caniuse:        nil,
								CompatFeatures: nil,
								Discouraged:    nil,
								Spec:           nil,
								Status: webdxfeaturetypes.Status{
									Baseline:         nil,
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
								},
								Description:     "text",
								DescriptionHTML: "<html>",
								Group:           nil,
								Snapshot:        nil,
							},
						},
						Moved: map[string]webdxfeaturetypes.FeatureMovedData{
							"movedFeature": {
								Kind:           webdxfeaturetypes.Moved,
								RedirectTarget: "new-feature",
							},
						},
						Split: map[string]webdxfeaturetypes.FeatureSplitData{
							"splitFeature": {
								Kind: webdxfeaturetypes.Split,
								RedirectTargets: []string{
									"new-feature-1",
									"new-feature-2",
								},
							},
						},
					},
				},
				returnedMapping: map[string]string{
					"feature1": "id-1",
				},
				returnError: nil,
			},
			mockInsertWebFeaturesMetadataCfg: mockInsertWebFeaturesMetadataConfig{
				expectedData: map[string]webdxfeaturetypes.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
						Status: webdxfeaturetypes.Status{
							Baseline:         nil,
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
				expectedFeatureData: map[string]webdxfeaturetypes.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
						Status: webdxfeaturetypes.Status{
							Baseline:         nil,
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
						},
						Description:     "text",
						DescriptionHTML: "<html>",
						Group:           nil,
						Snapshot:        nil,
					},
				},
				expectedGroupData: map[string]webdxfeaturetypes.GroupData{
					"group1": {
						Name:   "Group 1",
						Parent: nil,
					},
				},
				returnError: nil,
			},
			mockInsertWebFeatureSnapshotsCfg: mockInsertWebFeatureSnapshotsConfig{
				expectedFeatureData: map[string]webdxfeaturetypes.FeatureValue{
					"feature1": {
						Name:           "Feature 1",
						Caniuse:        nil,
						CompatFeatures: nil,
						Discouraged:    nil,
						Spec:           nil,
						Status: webdxfeaturetypes.Status{
							Baseline:         nil,
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
				expectedSnapshotData: map[string]webdxfeaturetypes.SnapshotData{
					"snapshot1": {
						Name: "Snapshot 1",
						Spec: "",
					},
				},
				returnError: nil,
			},
			mockInsertMovedFeaturesCfg: &mockInsertMovedFeaturesConfig{
				expectedData: map[string]webdxfeaturetypes.FeatureMovedData{
					"movedFeature": {
						Kind:           webdxfeaturetypes.Moved,
						RedirectTarget: "new-feature",
					},
				},
				returnError: nil,
			},
			mockInsertSplitFeaturesCfg: &mockInsertSplitFeaturesConfig{
				expectedData: map[string]webdxfeaturetypes.FeatureSplitData{
					"splitFeature": {
						Kind: webdxfeaturetypes.Split,
						RedirectTargets: []string{
							"new-feature-1",
							"new-feature-2",
						},
					},
				},
				returnError: errTestFailToStoreSplitFeatures,
			},
			expectedError: errTestFailToStoreSplitFeatures,
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
				callCount:    0,
			}
			mockParserV3 := &mockAssetParser{
				t:            t,
				mockParseCfg: tc.mockParseCfg,
				callCount:    0,
			}
			mockStorer := &mockWebFeatureStorer{
				t:                             t,
				mockInsertWebFeaturesCfg:      tc.mockInsertWebFeaturesCfg,
				mockInsertMovedWebFeaturesCfg: tc.mockInsertMovedFeaturesCfg,
				mockInsertSplitWebFeaturesCfg: tc.mockInsertSplitFeaturesCfg,
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
				mockParserV3,
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

func valuePtr[T any](in T) *T { return &in }

func TestParseByVersion(t *testing.T) {
	testCases := []struct {
		name                string
		file                *gh.ReleaseFile
		v2Parser            *mockAssetParser
		expectedV2CallCount int
		v3Parser            *mockAssetParser
		expectedV3CallCount int
		expectedError       error
	}{
		{
			name:                "missing tag",
			file:                testFile(nil, ""),
			v2Parser:            nil,
			v3Parser:            nil,
			expectedError:       ErrUnknownAssetVersion,
			expectedV2CallCount: 0,
			expectedV3CallCount: 0,
		},
		{
			name: "v2 parses successfully",
			file: testFile(valuePtr(v2), ""),
			v2Parser: &mockAssetParser{
				t: t,
				mockParseCfg: mockParseConfig{
					expectedFileContents: "",
					returnError:          nil,
					returnData:           nil,
				},
				callCount: 0,
			},
			v3Parser:            nil,
			expectedError:       nil,
			expectedV2CallCount: 1,
			expectedV3CallCount: 0,
		},
		{
			name: "v2 parses unsuccessfully",
			file: testFile(valuePtr(v2), ""),
			v2Parser: &mockAssetParser{
				t: t,
				mockParseCfg: mockParseConfig{
					expectedFileContents: "",
					returnError:          errTestCannotParseData,
					returnData:           nil,
				},
				callCount: 0,
			},
			v3Parser:            nil,
			expectedError:       errTestCannotParseData,
			expectedV2CallCount: 1,
			expectedV3CallCount: 0,
		},
		{
			name:     "v3 parses successfully",
			file:     testFile(valuePtr(v3), ""),
			v2Parser: nil,
			v3Parser: &mockAssetParser{
				t: t,
				mockParseCfg: mockParseConfig{
					expectedFileContents: "",
					returnError:          nil,
					returnData:           nil,
				},
				callCount: 0,
			},
			expectedError:       nil,
			expectedV2CallCount: 0,
			expectedV3CallCount: 1,
		},
		{
			name:     "v3 parses unsuccessfully",
			file:     testFile(valuePtr(v3), ""),
			v2Parser: nil,
			v3Parser: &mockAssetParser{
				t: t,
				mockParseCfg: mockParseConfig{
					expectedFileContents: "",
					returnError:          errTestCannotParseData,
					returnData:           nil,
				},
				callCount: 0,
			},
			expectedError:       errTestCannotParseData,
			expectedV2CallCount: 0,
			expectedV3CallCount: 1,
		},
		{
			name:                "unsupported tag",
			file:                testFile(valuePtr("v4.0.0"), ""),
			v2Parser:            nil,
			v3Parser:            nil,
			expectedError:       ErrUnsupportedAssetVersion,
			expectedV2CallCount: 0,
			expectedV3CallCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := WebFeaturesJobProcessor{
				assetGetter:             nil,
				webFeaturesDataV2Parser: tc.v2Parser,
				webFeaturesDataV3Parser: tc.v3Parser,
				storer:                  nil,
				metadataStorer:          nil,
				groupStorer:             nil,
				snapshotStorer:          nil,
			}
			_, err := p.parseByVersion(t.Context(), tc.file)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Expected error: %v, Got: %v", tc.expectedError, err)
			}
			if tc.v2Parser != nil && tc.v2Parser.callCount != tc.expectedV2CallCount {
				t.Errorf("Expected v2 call count: %d, Got: %d", tc.expectedV2CallCount, tc.v2Parser.callCount)
			}
			if tc.v2Parser == nil && tc.expectedV2CallCount > 0 {
				t.Error("Expected v2 parser to be called")
			}
			if tc.v3Parser != nil && tc.v3Parser.callCount != tc.expectedV3CallCount {
				t.Errorf("Expected v3 call count: %d, Got: %d", tc.expectedV3CallCount, tc.v3Parser.callCount)
			}
			if tc.v3Parser == nil && tc.expectedV3CallCount > 0 {
				t.Error("Expected v3 parser to be called")
			}
		})
	}
}
