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
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

type mockGetFeatureMetadataConfig struct {
	expectedFeatureID string
	result            *gds.FeatureMetadata
	err               error
}

type mockBackendDatastoreClient struct {
	t                         *testing.T
	mockGetFeatureMetadataCfg mockGetFeatureMetadataConfig
}

func (c mockBackendDatastoreClient) GetWebFeatureMetadata(
	_ context.Context, webFeatureID string) (*gds.FeatureMetadata, error) {
	if c.mockGetFeatureMetadataCfg.expectedFeatureID != webFeatureID {
		c.t.Error("unexpected input to mock")
	}

	return c.mockGetFeatureMetadataCfg.result, c.mockGetFeatureMetadataCfg.err
}

func valuePtr[T any](in T) *T { return &in }

var errGetMetadataTestError = errors.New("get feature metadata tests error")

func TestGetFeatureMetadata(t *testing.T) {
	testCases := []struct {
		name                      string
		featureID                 string
		mockGetFeatureMetadataCfg mockGetFeatureMetadataConfig
		expectedMetadata          *backend.FeatureMetadata
		expectedErr               error
	}{
		{
			name:      "success - no can i use ids",
			featureID: "id-1",
			mockGetFeatureMetadataCfg: mockGetFeatureMetadataConfig{
				expectedFeatureID: "id-1",
				result: &gds.FeatureMetadata{
					WebFeatureID: "id-1",
					Description:  "desc",
					CanIUseIDs:   nil,
				},
				err: nil,
			},
			expectedMetadata: &backend.FeatureMetadata{
				Description: valuePtr("desc"),
				CanIUse:     nil,
			},
			expectedErr: nil,
		},
		{
			name:      "success - with can i use ids",
			featureID: "id-1",
			mockGetFeatureMetadataCfg: mockGetFeatureMetadataConfig{
				expectedFeatureID: "id-1",
				result: &gds.FeatureMetadata{
					WebFeatureID: "id-1",
					Description:  "desc",
					CanIUseIDs: []string{
						"caniuse1",
						"caniuse2",
					},
				},
				err: nil,
			},
			expectedMetadata: &backend.FeatureMetadata{
				Description: valuePtr("desc"),
				CanIUse: &backend.CanIUseInfo{
					Items: &[]backend.CanIUseItem{
						{
							Id: valuePtr("caniuse1"),
						},
						{
							Id: valuePtr("caniuse2"),
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name:      "success - default metadata",
			featureID: "id-1",
			mockGetFeatureMetadataCfg: mockGetFeatureMetadataConfig{
				expectedFeatureID: "id-1",
				result:            nil,
				err:               gds.ErrEntityNotFound,
			},
			expectedMetadata: &backend.FeatureMetadata{
				Description: nil,
				CanIUse:     nil,
			},
			expectedErr: nil,
		},

		{
			name:      "error",
			featureID: "id-1",
			mockGetFeatureMetadataCfg: mockGetFeatureMetadataConfig{
				expectedFeatureID: "id-1",
				result:            nil,
				err:               errGetMetadataTestError,
			},
			expectedMetadata: nil,
			expectedErr:      errGetMetadataTestError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := mockBackendDatastoreClient{
				t:                         t,
				mockGetFeatureMetadataCfg: tc.mockGetFeatureMetadataCfg,
			}
			b := NewBackend(mock)
			metadata, err := b.GetFeatureMetadata(context.Background(), tc.featureID)
			if !errors.Is(err, tc.expectedErr) {
				t.Error("unexpected error")
			}
			if !reflect.DeepEqual(metadata, tc.expectedMetadata) {
				t.Error("unexpected metadata")
			}
		})
	}
}
