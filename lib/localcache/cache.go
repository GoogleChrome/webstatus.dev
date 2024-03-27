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

// LocalDataCache is an in-memory thread safe cache.
// It uses generics so that users of it can uses any type of data they want.
// The key K must be of type comparable. More infomration here: https://go.dev/blog/comparable
// The value V can be any type.
type LocalDataCache[K comparable, V any] struct {
	mu   *sync.RWMutex
	data map[K]V
}

// NewLocalDataCache creates a new LocalDataCache instance.
func NewLocalDataCache[K comparable, V any]() *LocalDataCache[K, V] {
	data := make(map[K]V)

	return &LocalDataCache[K, V]{
		data: data,
		mu:   &sync.RWMutex{},
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
// It returns a copy of the value if it exists in the cache.
//   - It returns a copy instead of a pointer so that users cannot modify it and impact the value stored in the cache.
//
// It returns cachetypes.ErrCachedDataNotFound if it does not exist.
// nolint: ireturn // V is not a interface always. Can ignore this.
func (c *LocalDataCache[K, V]) Get(
	_ context.Context,
	key K,
) (V, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if data, found := c.data[key]; found {
		return data, nil
	}

	// Return a zero valued version of V and the not found error.
	return *new(V), cachetypes.ErrCachedDataNotFound
}
