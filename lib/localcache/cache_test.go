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
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
)

func valuePtr[T any](in T) *T { return &in }

type getCacheDataTest struct {
	name          string
	cacheData     map[string]*string // Initial state of the cache
	key           string
	expectedValue *string
	expectedErr   error
}

type cacheDataTest struct {
	name          string
	cacheData     map[string]*string // Initial state of the cache
	key           string
	value         string
	expectedError error
}

func TestLocalDataCache(t *testing.T) {
	// Test for Get Method
	getCacheDataTests := []getCacheDataTest{
		{
			name:          "Cache Hit",
			cacheData:     map[string]*string{"hello": valuePtr("world")},
			key:           "hello",
			expectedValue: valuePtr("world"),
			expectedErr:   nil,
		},
		{
			name:          "Cache Miss",
			cacheData:     map[string]*string{}, // Empty cache
			key:           "missing-key",
			expectedValue: nil,
			expectedErr:   cachetypes.ErrCachedDataNotFound,
		},
	}
	for _, tt := range getCacheDataTests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewLocalDataCache[string]()
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
			cacheData:     map[string]*string{},
			key:           "new-key",
			value:         "new-value",
			expectedError: nil,
		},
		{
			name:          "Overwrite Existing",
			cacheData:     map[string]*string{"existing": valuePtr("old")},
			key:           "existing",
			value:         "updated",
			expectedError: nil,
		},
	}
	for idx, tt := range cacheDataTests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewLocalDataCache[string]()
			cache.data = tt.cacheData
			err := cache.Cache(context.Background(), tt.key, &cacheDataTests[idx].value)

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("Expected error: %v, Got: %v", tt.expectedError, err)
			}
			cachedValue, err := cache.Get(context.Background(), tt.key)
			if err != nil {
				t.Errorf("Error retrieving cached value: %v", err)
			}
			if !reflect.DeepEqual(cachedValue, &cacheDataTests[idx].value) {
				t.Errorf("Cached value mismatch. Expected: %v, Got: %v", tt.value, cachedValue)
			}
		})
	}
}
