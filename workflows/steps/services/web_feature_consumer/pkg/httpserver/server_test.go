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

package httpserver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/workflows/steps/web_feature_consumer"
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
	_ context.Context, data map[string]web_platform_dx__web_features.FeatureValue) (map[string]string, error) {
	if !reflect.DeepEqual(data, m.mockInsertWebFeaturesCfg.expectedData) {
		m.t.Error("unexpected data")
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

const (
	testRepoOwner = "owner"
	testRepoName  = "name"
	testFileName  = "file.txt"
)

func TestPostV1WebFeatures(t *testing.T) {
	testCases := []struct {
		name                             string
		mockDownloadFileFromReleaseCfg   mockDownloadFileFromReleaseConfig
		mockParseCfg                     mockParseConfig
		mockInsertWebFeaturesCfg         mockInsertWebFeaturesConfig
		mockInsertWebFeaturesMetadataCfg mockInsertWebFeaturesMetadataConfig
		expectedResponse                 web_feature_consumer.PostV1WebFeaturesResponseObject
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
					Features: map[string]web_platform_dx__web_features.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Spec:           nil,
							Status: web_platform_dx__web_features.Status{
								Baseline:         nil,
								BaselineHighDate: nil,
								BaselineLowDate:  nil,
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
						Spec:           nil,
						Status: web_platform_dx__web_features.Status{
							Baseline:         nil,
							BaselineHighDate: nil,
							BaselineLowDate:  nil,
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
						Spec:           nil,
						Status: web_platform_dx__web_features.Status{
							Baseline:         nil,
							BaselineHighDate: nil,
							BaselineLowDate:  nil,
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
			expectedResponse: web_feature_consumer.PostV1WebFeatures200Response{},
		},
		{
			name: "fail to get asset data",
			mockDownloadFileFromReleaseCfg: mockDownloadFileFromReleaseConfig{
				expectedOwner:    testRepoOwner,
				expectedRepo:     testRepoName,
				expectedFileName: testFileName,
				returnReadCloser: io.NopCloser(strings.NewReader("hi features")),
				returnError:      errors.New("fail to get asset"),
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
			expectedResponse: web_feature_consumer.PostV1WebFeatures500JSONResponse{
				Code:    500,
				Message: "unable to get asset",
			},
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
				returnError:          errors.New("cannot parse data"),
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
			expectedResponse: web_feature_consumer.PostV1WebFeatures500JSONResponse{
				Code:    500,
				Message: "unable to parse data",
			},
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
					Features: map[string]web_platform_dx__web_features.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Spec:           nil,
							Status: web_platform_dx__web_features.Status{
								Baseline:         nil,
								BaselineHighDate: nil,
								BaselineLowDate:  nil,
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
						Spec:           nil,
						Status: web_platform_dx__web_features.Status{
							Baseline:         nil,
							BaselineHighDate: nil,
							BaselineLowDate:  nil,
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
				returnError: errors.New("uh-oh"),
			},
			mockInsertWebFeaturesMetadataCfg: mockInsertWebFeaturesMetadataConfig{
				expectedData:    nil,
				expectedMapping: nil,
				returnError:     nil,
			},
			expectedResponse: web_feature_consumer.PostV1WebFeatures500JSONResponse{
				Code:    500,
				Message: "unable to store data",
			},
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
					Features: map[string]web_platform_dx__web_features.FeatureValue{
						"feature1": {
							Name:           "Feature 1",
							Caniuse:        nil,
							CompatFeatures: nil,
							Spec:           nil,
							Status: web_platform_dx__web_features.Status{
								Baseline:         nil,
								BaselineHighDate: nil,
								BaselineLowDate:  nil,
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
						Spec:           nil,
						Status: web_platform_dx__web_features.Status{
							Baseline:         nil,
							BaselineHighDate: nil,
							BaselineLowDate:  nil,
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
						Spec:           nil,
						Status: web_platform_dx__web_features.Status{
							Baseline:         nil,
							BaselineHighDate: nil,
							BaselineLowDate:  nil,
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
				returnError: errors.New("uh-oh"),
			},
			expectedResponse: web_feature_consumer.PostV1WebFeatures500JSONResponse{
				Code:    500,
				Message: "unable to store metadata",
			},
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

			server := &Server{
				assetGetter:           mockGetter,
				storer:                mockStorer,
				metadataStorer:        mockMetadataStorer,
				webFeaturesDataParser: mockParser,
				defaultAssetName:      testFileName,
				defaultRepoOwner:      testRepoOwner,
				defaultRepoName:       testRepoName,
			}

			req := web_feature_consumer.PostV1WebFeaturesRequestObject{}

			response, err := server.PostV1WebFeatures(context.TODO(), req)
			if err != nil {
				t.Errorf("error should not be set")
			}
			if !reflect.DeepEqual(tc.expectedResponse, response) {
				t.Error("unexpected response")
			}
		})
	}
}
