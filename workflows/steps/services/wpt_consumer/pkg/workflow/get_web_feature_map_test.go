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
	"sync"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
	"github.com/GoogleChrome/webstatus.dev/lib/localcache"
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
	cachedData           shared.WebFeaturesData
	t                    *testing.T
}

func (m *MockDataCacher) Cache(_ context.Context, key string, value shared.WebFeaturesData) error {
	if key != m.mockCacheCacheConfig.expectedKey ||
		!reflect.DeepEqual(value, m.mockCacheCacheConfig.expectedData) {
		m.t.Error("unexpected input to Cache")
	}
	m.cachedData = value

	return m.mockCacheCacheConfig.err
}

func (m *MockDataCacher) Get(_ context.Context, key string) (shared.WebFeaturesData, error) {
	if m.cachedData != nil {
		return m.cachedData, nil
	}
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
				cachedData:           nil,
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

type countingGetter struct {
	mu    sync.Mutex
	calls int
	data  shared.WebFeaturesData
}

func (g *countingGetter) Get(_ context.Context) (shared.WebFeaturesData, error) {
	time.Sleep(10 * time.Millisecond)
	g.mu.Lock()
	defer g.mu.Unlock()
	g.calls++

	return g.data, nil
}

func TestGetWebFeaturesData_Concurrent(t *testing.T) {
	cache := localcache.NewLocalDataCache[string, shared.WebFeaturesData](
		localcache.NestedMapCopier[shared.WebFeaturesData](),
	)
	getterImpl := &countingGetter{
		mu:    sync.Mutex{},
		calls: 0,
		data: shared.WebFeaturesData{
			"test1.html": {"feature1": nil},
			"test2.html": {"feature2": nil},
		},
	}

	getter := NewCacheableWebFeaturesDataGetter(getterImpl, cache)

	numGoroutines := 30
	var wg sync.WaitGroup

	for range numGoroutines {
		wg.Go(func() {
			data, err := getter.GetWebFeaturesData(context.Background(), "test-rev")
			if err != nil {
				t.Error(err)
			}
			// Simulate mutation/read by worker
			if len(data) == 0 {
				t.Error("expected non-empty data")
			}
			data["mutated_test.html"] = map[string]any{"mutated": nil}
		})
	}

	wg.Wait()

	if getterImpl.calls != 1 {
		t.Errorf("expected exactly 1 call to client.Get, got %d", getterImpl.calls)
	}
}
