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

package datastoreadapters

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/gds"
	"github.com/GoogleChrome/webstatus.dev/lib/webdxfeaturetypes"
)

// MockWebFeatureDatastoreClient allows us to control UpsertFeatureMetadata behavior.
type MockWebFeatureDatastoreClient struct {
	UpsertedData  map[string]gds.FeatureMetadata
	ErrorToReturn error
}

func (m *MockWebFeatureDatastoreClient) UpsertFeatureMetadata(_ context.Context, data gds.FeatureMetadata) error {
	if m.ErrorToReturn != nil {
		return m.ErrorToReturn
	}
	m.UpsertedData[data.WebFeatureID] = data

	return nil
}

func valuePtr[T any](in T) *T { return &in }

func TestInsertWebFeaturesMetadata(t *testing.T) {
	testCases := []struct {
		name                string
		featureKeyToID      map[string]string
		inputFeatureData    map[string]webdxfeaturetypes.FeatureValue
		expectedUpserts     map[string]gds.FeatureMetadata
		mockClientError     error
		expectedErrorReturn error
	}{
		{
			name:           "Success with single CanIUse ID",
			featureKeyToID: map[string]string{"feature1": "id1"},
			inputFeatureData: map[string]webdxfeaturetypes.FeatureValue{
				"feature1": {
					CompatFeatures:  nil,
					Name:            "feature 1",
					Description:     "Feature 1 description",
					Caniuse:         []string{"caniuse-id1"},
					DescriptionHTML: "<html>1",
					Discouraged:     nil,
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
					Spec:     nil,
					Group:    nil,
					Snapshot: nil,
				},
			},
			expectedUpserts: map[string]gds.FeatureMetadata{
				"id1": {WebFeatureID: "id1", Description: "Feature 1 description", CanIUseIDs: []string{"caniuse-id1"}},
			},
			mockClientError:     nil,
			expectedErrorReturn: nil,
		},
		{
			name:           "Success with multiple CanIUse IDs",
			featureKeyToID: map[string]string{"feature2": "id2"},
			inputFeatureData: map[string]webdxfeaturetypes.FeatureValue{
				"feature2": {
					CompatFeatures:  nil,
					Name:            "feature 2",
					Description:     "Feature 2 description",
					Discouraged:     nil,
					Caniuse:         []string{"caniuse-id2a", "caniuse-id2b"},
					DescriptionHTML: "<html>2",
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
					Spec:     nil,
					Group:    nil,
					Snapshot: nil,
				},
			},
			expectedUpserts: map[string]gds.FeatureMetadata{
				"id2": {
					WebFeatureID: "id2",
					Description:  "Feature 2 description",
					CanIUseIDs:   []string{"caniuse-id2a", "caniuse-id2b"},
				},
			},
			mockClientError:     nil,
			expectedErrorReturn: nil,
		},
		{
			name:           "Missing feature ID",
			featureKeyToID: map[string]string{},
			inputFeatureData: map[string]webdxfeaturetypes.FeatureValue{
				"feature3": {
					Caniuse:         nil,
					CompatFeatures:  nil,
					Name:            "feature 3",
					Description:     "Feature 3 description",
					DescriptionHTML: "<html>3",
					Discouraged:     nil,
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
					Spec:     nil,
					Group:    nil,
					Snapshot: nil,
				},
			},
			expectedUpserts:     map[string]gds.FeatureMetadata{},
			mockClientError:     nil,
			expectedErrorReturn: nil,
		},
		{
			name:           "Upsert error",
			featureKeyToID: map[string]string{"feature4": "id4"},
			inputFeatureData: map[string]webdxfeaturetypes.FeatureValue{
				"feature4": {
					Caniuse:         nil,
					CompatFeatures:  nil,
					Name:            "feature 4",
					Description:     "Feature 4 description",
					DescriptionHTML: "<html>4",
					Discouraged:     nil,
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
					Spec:     nil,
					Group:    nil,
					Snapshot: nil,
				},
			},
			expectedUpserts:     map[string]gds.FeatureMetadata{},
			mockClientError:     context.DeadlineExceeded,
			expectedErrorReturn: context.DeadlineExceeded,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockWebFeatureDatastoreClient{
				UpsertedData:  make(map[string]gds.FeatureMetadata),
				ErrorToReturn: tc.mockClientError,
			}
			consumer := NewWebFeaturesConsumer(mockClient)

			err := consumer.InsertWebFeaturesMetadata(context.Background(), tc.featureKeyToID, tc.inputFeatureData)
			if !errors.Is(err, tc.expectedErrorReturn) {
				t.Errorf("Unexpected error: got %v, want %v", err, tc.expectedErrorReturn)
			}

			if !reflect.DeepEqual(mockClient.UpsertedData, tc.expectedUpserts) {
				t.Errorf("Upserted data mismatch:\ngot:  %v\nwant: %v", mockClient.UpsertedData, tc.expectedUpserts)
			}
		})
	}
}
