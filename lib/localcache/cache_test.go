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

package localcache

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
)

type getCacheDataTest struct {
	name          string
	cacheData     map[string]string // Initial state of the cache
	key           string
	expectedValue string
	expectedErr   error
}

type cacheDataTest struct {
	name          string
	cacheData     map[string]string // Initial state of the cache
	key           string
	value         string
	expectedError error
}

func TestLocalDataCache(t *testing.T) {
	// Test for Get Method
	getCacheDataTests := []getCacheDataTest{
		{
			name:          "Cache Hit",
			cacheData:     map[string]string{"hello": "world"},
			key:           "hello",
			expectedValue: "world",
			expectedErr:   nil,
		},
		{
			name:          "Cache Miss",
			cacheData:     map[string]string{}, // Empty cache
			key:           "missing-key",
			expectedValue: "",
			expectedErr:   cachetypes.ErrCachedDataNotFound,
		},
	}
	for _, tt := range getCacheDataTests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewLocalDataCache[string, string](nil)
			cache.data = tt.cacheData
			val, err := cache.Get(context.Background(), tt.key)

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("Expected error: %v, Got: %v", tt.expectedErr, err)
			}
			if !reflect.DeepEqual(val, tt.expectedValue) {
				t.Errorf("Expected value: %v, Got: %v", tt.expectedValue, val)
			}
		})
	}

	// Test for Cache Method
	cacheDataTests := []cacheDataTest{
		{
			name:          "Add New Entry",
			cacheData:     map[string]string{},
			key:           "new-key",
			value:         "new-value",
			expectedError: nil,
		},
		{
			name:          "Overwrite Existing",
			cacheData:     map[string]string{"existing": "old"},
			key:           "existing",
			value:         "updated",
			expectedError: nil,
		},
	}
	for idx, tt := range cacheDataTests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewLocalDataCache[string, string](nil)
			cache.data = tt.cacheData
			err := cache.Cache(context.Background(), tt.key, cacheDataTests[idx].value)

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("Expected error: %v, Got: %v", tt.expectedError, err)
			}
			cachedValue, err := cache.Get(context.Background(), tt.key)
			if err != nil {
				t.Errorf("Error retrieving cached value: %v", err)
			}
			if !reflect.DeepEqual(cachedValue, cacheDataTests[idx].value) {
				t.Errorf("Cached value mismatch. Expected: %v, Got: %v", tt.value, cachedValue)
			}
		})
	}
}

// TestLocalDataCache_ConcurrentMapAccess tests that the cache can safely handle
// concurrent read access to a map value when a copier function is used.
// This test is designed to fail under the Go race detector if the cache's Get
// method does not return a proper copy, leading to a data race.
func TestLocalDataCache_ConcurrentMapAccess(t *testing.T) {
	// Create a new cache instance with the generic MapCopier function.
	cache := NewLocalDataCache[string, map[string]int](MapCopier[map[string]int]())
	initialMap := map[string]int{"a": 1}
	// Seed the cache with an initial value.
	err := cache.Cache(context.Background(), "test-map", initialMap)
	if err != nil {
		t.Error(err)
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	// Start multiple goroutines to simulate concurrent access.
	for range numGoroutines {
		wg.Go(func() {
			// Each goroutine gets the map from the cache.
			m, err := cache.Get(context.Background(), "test-map")
			if err != nil {
				// t.Error is safe for concurrent use by multiple goroutines.
				t.Error(err)
			}
			// Each goroutine modifies its local copy of the map.
			// If the cache's Get method does not use the copier correctly and
			// returns a direct reference to the underlying map, the race
			// detector will report a data race on this line.
			m["a"]++
		})
	}

	// Wait for all goroutines to complete. If there was a race condition,
	// the test will have already failed when run with the -race flag.
	wg.Wait()
}

// TestLocalDataCache_CopyOnWrite tests that mutating the original map passed to Cache
// does not modify the cached data when a copier is provided.
func TestLocalDataCache_CopyOnWrite(t *testing.T) {
	cache := NewLocalDataCache[string, map[string]int](MapCopier[map[string]int]())
	inputMap := map[string]int{"a": 1}

	err := cache.Cache(context.Background(), "test-key", inputMap)
	if err != nil {
		t.Fatalf("unexpected error caching: %v", err)
	}

	// Mutate the original map after caching.
	inputMap["a"] = 999
	inputMap["b"] = 123

	// Retrieve from cache and verify it still has the original value.
	cached, err := cache.Get(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("unexpected error getting: %v", err)
	}

	expected := map[string]int{"a": 1}
	if !reflect.DeepEqual(cached, expected) {
		t.Errorf("expected cached map %v, got %v", expected, cached)
	}
}

// TestLocalDataCache_ConcurrentNestedMapAccess tests that NestedMapCopier provides
// complete thread safety and mutation isolation across 100 concurrent workers modifying
// both inner and outer maps simultaneously.
func TestLocalDataCache_ConcurrentNestedMapAccess(t *testing.T) {
	type NestedMap map[string]map[string]any

	cache := NewLocalDataCache[string, NestedMap](NestedMapCopier[NestedMap]())
	initialData := NestedMap{
		"test1.html": {"feat1": 100, "feat2": 200},
		"test2.html": {"feat3": 300},
	}

	err := cache.Cache(context.Background(), "nested-key", initialData)
	if err != nil {
		t.Fatalf("unexpected error caching: %v", err)
	}

	// Mutate original map to ensure copy-on-store holds
	initialData["test1.html"]["feat1"] = 999
	initialData["test3.html"] = map[string]any{"feat4": 400}

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := range numGoroutines {
		wg.Go(func() {
			m, err := cache.Get(context.Background(), "nested-key")
			if err != nil {
				t.Error(err)
			}
			// Verify original values are untouched
			if m["test1.html"]["feat1"] != 100 {
				t.Errorf("expected feat1=100, got %v", m["test1.html"]["feat1"])
			}
			if _, exists := m["test3.html"]; exists {
				t.Errorf("expected test3.html to not exist in cached copy")
			}

			// Perform concurrent mutations on both outer and inner maps
			m["test1.html"]["feat1"] = i
			delete(m["test1.html"], "feat2")
			m["test1.html"]["new_feat"] = "concurrent_value"
			m["new_test.html"] = map[string]any{"custom": i}
		})
	}

	wg.Wait()

	// Verify the cached state remains completely pristine after all workers finish
	finalCached, err := cache.Get(context.Background(), "nested-key")
	if err != nil {
		t.Fatalf("unexpected error getting: %v", err)
	}

	expectedFinal := NestedMap{
		"test1.html": {"feat1": 100, "feat2": 200},
		"test2.html": {"feat3": 300},
	}
	if !reflect.DeepEqual(finalCached, expectedFinal) {
		t.Errorf("cache corrupted! expected %v, got %v", expectedFinal, finalCached)
	}
}

func TestNestedMapCopier_NilHandling(t *testing.T) {
	type NestedMap map[string]map[string]int
	copier := NestedMapCopier[NestedMap]()

	if copier(nil) != nil {
		t.Errorf("expected nil for nil input")
	}

	inputWithNilInner := NestedMap{
		"test1": nil,
		"test2": {"a": 1},
	}
	copied := copier(inputWithNilInner)
	if copied["test1"] != nil {
		t.Errorf("expected nil inner map for test1, got %v", copied["test1"])
	}
	if copied["test2"]["a"] != 1 {
		t.Errorf("expected 1, got %v", copied["test2"]["a"])
	}
}
