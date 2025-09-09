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
	"sync"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
)

// Copier is a function that creates a deep copy of a value of type V.
// This is necessary for cache implementations that store reference types (e.g., maps, slices, pointers)
// to prevent race conditions when multiple goroutines access and modify cached data.
type Copier[V any] func(V) V

// LocalDataCache is an in-memory thread safe cache.
// It uses generics so that users of it can uses any type of data they want.
// The key K must be of type comparable. More infomration here: https://go.dev/blog/comparable
// The value V can be any type.
type LocalDataCache[K comparable, V any] struct {
	mu     *sync.RWMutex
	data   map[K]V
	copier Copier[V]
}

// NewLocalDataCache creates a new LocalDataCache instance.
// It accepts an optional copier function. If a copier is provided, the Get
// method will return a deep copy of the cached value. This is critical for
// ensuring thread safety when caching reference types that might be mutated by consumers.
// This design is chosen to provide type safety and performance, avoiding the use of reflection
// for deep copying. If the copier is nil, values will be returned by reference.
func NewLocalDataCache[K comparable, V any](copier Copier[V]) *LocalDataCache[K, V] {
	data := make(map[K]V)

	return &LocalDataCache[K, V]{
		data:   data,
		mu:     &sync.RWMutex{},
		copier: copier,
	}
}

// Cache stores a value in the cache.
func (c *LocalDataCache[K, V]) Cache(
	_ context.Context,
	key K,
	in V,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = in

	return nil
}

// Get retrieves a value from the cache.
// If a copier function was provided to NewLocalDataCache, this method returns a
// deep copy of the value. Otherwise, it returns a direct reference.
// Returning a copy for reference types is crucial for preventing race conditions.
// It returns cachetypes.ErrCachedDataNotFound if the key does not exist.
// nolint: ireturn // V is not a interface always. Can ignore this.
func (c *LocalDataCache[K, V]) Get(
	_ context.Context,
	key K,
) (V, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, found := c.data[key]
	if !found {
		// Return a zero valued version of V and the not found error.
		return *new(V), cachetypes.ErrCachedDataNotFound
	}

	if c.copier != nil {
		return c.copier(data), nil
	}

	return data, nil
}
