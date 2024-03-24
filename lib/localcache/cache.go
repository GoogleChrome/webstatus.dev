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

type LocalDataCache[T any] struct {
	mu   *sync.RWMutex
	data map[string]*T
}

func NewLocalDataCache[T any]() *LocalDataCache[T] {
	data := make(map[string]*T)

	return &LocalDataCache[T]{
		data: data,
		mu:   &sync.RWMutex{},
	}
}

func (c *LocalDataCache[T]) Cache(
	_ context.Context,
	key string,
	in *T,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = in

	return nil
}

func (c *LocalDataCache[T]) Get(
	_ context.Context,
	key string,
) (*T, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if data, found := c.data[key]; found {
		return data, nil
	}

	return nil, cachetypes.ErrCachedDataNotFound
}
