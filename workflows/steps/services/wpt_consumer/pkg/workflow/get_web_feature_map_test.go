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
	"reflect"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

type MockWebFeatureDataGetter struct {
	data shared.WebFeaturesData
	err  error
}

func (g *MockWebFeatureDataGetter) Get(_ context.Context) (shared.WebFeaturesData, error) {
	return g.data, g.err
}

type mockCacheGetConfig struct {
	expectedKey string
	data        shared.WebFeaturesData
	err         error
}

type mockCacheCacheConfig struct {
	expectedKey  string
	expectedData shared.WebFeaturesData
	err          error
}
type MockDataCacher struct {
	mockCacheGetConfig   *mockCacheGetConfig
	mockCacheCacheConfig *mockCacheCacheConfig
	t                    *testing.T
}

func (m *MockDataCacher) Cache(_ context.Context, key string, value shared.WebFeaturesData) error {
	if key != m.mockCacheCacheConfig.expectedKey ||
		!reflect.DeepEqual(value, m.mockCacheCacheConfig.expectedData) {
		m.t.Error("unexpected input to Cache")
	}

	return m.mockCacheCacheConfig.err
}

func (m *MockDataCacher) Get(_ context.Context, key string) (shared.WebFeaturesData, error) {
	if key != m.mockCacheGetConfig.expectedKey {
		m.t.Error("unexpected input to Get")
	}

	return m.mockCacheGetConfig.data, m.mockCacheGetConfig.err
}

type getWebFeaturesDataTest struct {
	name                     string
	mockWebFeatureDataGetter *MockWebFeatureDataGetter
	mockCacheCacheConfig     *mockCacheCacheConfig
	mockCacheGetConfig       *mockCacheGetConfig
	expectedData             shared.WebFeaturesData
	expectedErr              error
}

// nolint: gochecknoglobals
var (
	liveData = shared.WebFeaturesData{
		"test1.html": {
			"feature1": nil,
		},
	}
	cachedData = shared.WebFeaturesData{"from-cache": nil}
)

func TestGetWebFeaturesData(t *testing.T) {
	testCases := []getWebFeaturesDataTest{
		{
			name: "Cache Hit",
			mockWebFeatureDataGetter: &MockWebFeatureDataGetter{
				data: nil,
				err:  nil,
			},
			mockCacheGetConfig: &mockCacheGetConfig{
				expectedKey: cacheKeyLatest,
				data:        cachedData,
				err:         nil,
			},
			mockCacheCacheConfig: nil,
			expectedErr:          nil,
			expectedData:         cachedData,
		},
		{
			name: "Cache Miss",
			mockWebFeatureDataGetter: &MockWebFeatureDataGetter{
				data: liveData,
				err:  nil,
			},
			mockCacheGetConfig: &mockCacheGetConfig{
				expectedKey: cacheKeyLatest,
				data:        nil,
				err:         cachetypes.ErrCachedDataNotFound,
			},
			mockCacheCacheConfig: &mockCacheCacheConfig{
				expectedKey:  cacheKeyLatest,
				expectedData: liveData,
				err:          nil,
			},
			expectedData: liveData,
			expectedErr:  nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			dataCacher := MockDataCacher{
				t:                    t,
				mockCacheGetConfig:   tt.mockCacheGetConfig,
				mockCacheCacheConfig: tt.mockCacheCacheConfig,
			}
			getter := NewCacheableWebFeaturesDataGetter(
				tt.mockWebFeatureDataGetter,
				&dataCacher)

			data, err := getter.GetWebFeaturesData(context.Background(), "test-revision")

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("Expected error: %v, Got: %v", tt.expectedErr, err)
			}

			if !reflect.DeepEqual(data, tt.expectedData) {
				t.Error("unexpected data")
			}
		})
	}
}
