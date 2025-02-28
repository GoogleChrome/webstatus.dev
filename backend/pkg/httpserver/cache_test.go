// Copyright 2025 Google LLC
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
	"reflect"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
)

// MockRawBytesDataCacher is a mock implementation of RawBytesDataCacher for testing.
type MockRawBytesDataCacher struct {
	t                   *testing.T
	expectedCacheCalls  []*ExpectedCacheCall
	expectedGetCalls    []*ExpectedGetCall
	actualCacheCalls    []*ActualCacheCall
	actualGetCalls      []*ActualGetCall
	currentCacheCallIdx int
	currentGetCallIdx   int
}

// NewMockRawBytesDataCacher creates a new MockRawBytesDataCacher.
func NewMockRawBytesDataCacher(
	t *testing.T,
	expectedCacheCalls []*ExpectedCacheCall,
	expectedGetCalls []*ExpectedGetCall) *MockRawBytesDataCacher {
	return &MockRawBytesDataCacher{
		t:                   t,
		expectedCacheCalls:  expectedCacheCalls,
		expectedGetCalls:    expectedGetCalls,
		actualCacheCalls:    []*ActualCacheCall{},
		actualGetCalls:      []*ActualGetCall{},
		currentCacheCallIdx: 0,
		currentGetCallIdx:   0,
	}
}

func getDefaultCacheConfig() *cachetypes.CacheConfig {
	return cachetypes.NewCacheConfig(0)
}

func getTestAggregatedCacheConfig() *cachetypes.CacheConfig {
	return cachetypes.NewCacheConfig(10)
}

func getTestRouteCacheOptions() RouteCacheOptions {
	return RouteCacheOptions{
		AggregatedFeatureStatsOptions: []cachetypes.CacheOption{
			cachetypes.WithTTL(10),
		},
	}
}

// Cache implements the Cache method of the RawBytesDataCacher interface.
func (m *MockRawBytesDataCacher) Cache(_ context.Context, key string, value []byte,
	options ...cachetypes.CacheOption) error {
	if m.currentCacheCallIdx >= len(m.expectedCacheCalls) {
		m.t.Errorf("unexpected call to Cache with key: %s", key)

		return nil
	}

	expectedCall := m.expectedCacheCalls[m.currentCacheCallIdx]
	if expectedCall.Key != key {
		m.t.Errorf("expected Cache key: %s, got: %s", expectedCall.Key, key)
	}
	if !reflect.DeepEqual(expectedCall.Value, value) {
		m.t.Errorf("expected Cache value: %v, got: %v", string(expectedCall.Value), string(value))
	}

	cacheCfg := getDefaultCacheConfig()

	for _, opt := range options {
		opt(cacheCfg)
	}

	if !reflect.DeepEqual(expectedCall.CacheCfg, cacheCfg) {
		m.t.Errorf("expected Cache config: %v, got: %v", expectedCall.CacheCfg, cacheCfg)
	}

	m.actualCacheCalls = append(m.actualCacheCalls, &ActualCacheCall{
		Key:      key,
		Value:    value,
		CacheCfg: cacheCfg,
	})
	m.currentCacheCallIdx++

	return nil
}

// Get implements the Get method of the RawBytesDataCacher interface.
func (m *MockRawBytesDataCacher) Get(_ context.Context, key string) ([]byte, error) {
	if m.currentGetCallIdx >= len(m.expectedGetCalls) {
		m.t.Errorf("unexpected call to Get with key: %s", key)

		return nil, nil
	}

	expectedCall := m.expectedGetCalls[m.currentGetCallIdx]
	if expectedCall.Key != key {
		m.t.Errorf("expected Get key: %s, got: %s", expectedCall.Key, key)
	}

	m.actualGetCalls = append(m.actualGetCalls, &ActualGetCall{
		Key: key,
	})

	m.currentGetCallIdx++

	return expectedCall.Value, expectedCall.Err
}

// AssertExpectations asserts that all expected calls were made.
func (m *MockRawBytesDataCacher) AssertExpectations() {
	if len(m.expectedCacheCalls) != m.currentCacheCallIdx {
		m.t.Errorf("expected %d Cache calls, got %d", len(m.expectedCacheCalls), m.currentCacheCallIdx)
		for i, expectedCall := range m.expectedCacheCalls {
			if i < len(m.actualCacheCalls) {
				actualCall := m.actualCacheCalls[i]
				m.t.Errorf("Expected Cache Call: %+v, got: %+v \n", expectedCall, actualCall)
			} else {
				m.t.Errorf("Expected Cache Call: %+v\n", expectedCall)
			}
		}
	}

	if len(m.expectedGetCalls) != m.currentGetCallIdx {
		m.t.Errorf("expected %d Get calls, got %d", len(m.expectedGetCalls), m.currentGetCallIdx)
		for i, expectedCall := range m.expectedGetCalls {
			if i < len(m.actualGetCalls) {
				actualCall := m.actualGetCalls[i]
				m.t.Errorf("Expected Get Call: %+v, got: %+v \n", expectedCall, actualCall)
			} else {
				m.t.Errorf("Expected Get Call: %+v\n", expectedCall)
			}
		}
	}
}

// ExpectedCacheCall represents an expected call to Cache.
type ExpectedCacheCall struct {
	Key      string
	Value    []byte
	CacheCfg *cachetypes.CacheConfig
}

// ActualCacheCall represents an actual call made to Cache.
type ActualCacheCall struct {
	Key      string
	Value    []byte
	CacheCfg *cachetypes.CacheConfig
}

// ExpectedGetCall represents an expected call to Get.
type ExpectedGetCall struct {
	Key   string
	Value []byte
	Err   error
}

// ActualGetCall represents an actual call made to Get.
type ActualGetCall struct {
	Key string
}

// Test Types.
type TestKey struct {
	ID    string
	Param string
}

type TestValue struct {
	Name  string
	Value int
}

func TestOperationResponseCache_AttemptCache(t *testing.T) {
	testCases := []struct {
		name               string
		key                TestKey
		value              *TestValue
		expectedCacheCalls []*ExpectedCacheCall
		cacheOptions       []cachetypes.CacheOption
	}{
		{
			name: "Valid Cache Operation",
			key: TestKey{
				ID:    "test-id",
				Param: "test-param",
			},
			value: &TestValue{
				Name:  "Test Item",
				Value: 123,
			},
			expectedCacheCalls: []*ExpectedCacheCall{
				{
					Key:      `customOperation-{"ID":"test-id","Param":"test-param"}`,
					Value:    []byte(`{"Name":"Test Item","Value":123}`),
					CacheCfg: getDefaultCacheConfig(),
				},
			},
			cacheOptions: nil,
		},
		{
			name: "Valid Cache Operation with option",
			key: TestKey{
				ID:    "test-id",
				Param: "test-param",
			},
			value: &TestValue{
				Name:  "Test Item",
				Value: 123,
			},
			expectedCacheCalls: []*ExpectedCacheCall{
				{
					Key:      `customOperation-{"ID":"test-id","Param":"test-param"}`,
					Value:    []byte(`{"Name":"Test Item","Value":123}`),
					CacheCfg: cachetypes.NewCacheConfig(10),
				},
			},
			cacheOptions: []cachetypes.CacheOption{
				cachetypes.WithTTL(10),
			},
		},
		{
			name: "Nil value - should not cache",
			key: TestKey{
				ID:    "test-id",
				Param: "test-param",
			},
			value:              nil,
			expectedCacheCalls: []*ExpectedCacheCall{},
			cacheOptions:       nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCacher := NewMockRawBytesDataCacher(t, tc.expectedCacheCalls, nil)
			cache := operationResponseCache[
				TestKey,
				TestValue,
			]{cacher: mockCacher, operationID: "customOperation", overrideCacheOptions: tc.cacheOptions}

			cache.AttemptCache(context.Background(), tc.key, tc.value)
			mockCacher.AssertExpectations()
		})
	}
}

func TestOperationResponseCache_Lookup(t *testing.T) {
	testCases := []struct {
		name             string
		key              TestKey
		expectedGetCalls []*ExpectedGetCall
		expectedResult   bool
		expectedValue    *TestValue
	}{
		{
			name: "Valid Lookup - Data Found",
			key: TestKey{
				ID:    "test-id",
				Param: "test-param",
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `customOperation-{"ID":"test-id","Param":"test-param"}`,
					Value: []byte(`{"Name":"Test Item","Value":123}`),
					Err:   nil,
				},
			},
			expectedResult: true,
			expectedValue: &TestValue{
				Name:  "Test Item",
				Value: 123,
			},
		},
		{
			name: "Lookup - Data Not Found",
			key: TestKey{
				ID:    "test-id",
				Param: "test-param",
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `customOperation-{"ID":"test-id","Param":"test-param"}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedResult: false,
			expectedValue:  nil,
		},
		{
			name: "Lookup - Cache Error",
			key: TestKey{
				ID:    "test-id",
				Param: "test-param",
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `customOperation-{"ID":"test-id","Param":"test-param"}`,
					Value: nil,
					Err:   errors.New("some error"),
				},
			},
			expectedResult: false,
			expectedValue:  nil,
		},
		{
			name: "Lookup - invalid cached value",
			key: TestKey{
				ID:    "test-id",
				Param: "test-param",
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `customOperation-{"ID":"test-id","Param":"test-param"}`,
					Value: []byte(`invalid json`),
					Err:   nil,
				},
			},
			expectedResult: false,
			expectedValue:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCacher := NewMockRawBytesDataCacher(t, nil, tc.expectedGetCalls)
			cache := operationResponseCache[
				TestKey,
				TestValue,
			]{cacher: mockCacher, operationID: "customOperation", overrideCacheOptions: nil}

			var actualValue TestValue
			result := cache.Lookup(context.Background(), tc.key, &actualValue)

			if result != tc.expectedResult {
				t.Errorf("Expected result %t, got %t", tc.expectedResult, result)
			}
			if tc.expectedResult && !reflect.DeepEqual(&actualValue, tc.expectedValue) {
				t.Errorf("Expected value %v, got %v", tc.expectedValue, &actualValue)
			}
			mockCacher.AssertExpectations()
		})
	}
}
