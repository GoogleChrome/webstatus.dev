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
	// Define a copier function for our test map type. This is what a consumer
	// of the cache would provide for their specific reference type.
	copier := func(in map[string]int) map[string]int {
		if in == nil {
			return nil
		}
		out := make(map[string]int, len(in))
		for k, v := range in {
			out[k] = v
		}

		return out
	}

	// Create a new cache instance with the copier function.
	cache := NewLocalDataCache[string, map[string]int](copier)
	initialMap := map[string]int{"a": 1}
	// Seed the cache with an initial value.
	err := cache.Cache(context.Background(), "test-map", initialMap)
	if err != nil {
		t.Error(err)
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	// Start multiple goroutines to simulate concurrent access.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
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
		}()
	}

	// Wait for all goroutines to complete. If there was a race condition,
	// the test will have already failed when run with the -race flag.
	wg.Wait()
}
