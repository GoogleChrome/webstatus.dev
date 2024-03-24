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
	"sync"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

type DataCacher[T any] interface {
	Cache(
		context.Context,
		string,
		*T,
	) error
	Get(
		context.Context,
		string,
	) (*T, error)
}

type LocalDataCache[T any] struct {
	mu   sync.RWMutex
	data map[string]*T
}

func (c *LocalDataCache[T]) Cache(
	ctx context.Context,
	key string,
	in *T,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = in

	return nil
}

var ErrCachedDataNotFound = errors.New("cached data not found for key")

func (c *LocalDataCache[T]) Get(
	ctx context.Context,
	key string,
) (*T, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if data, found := c.data[key]; found {
		return data, nil
	}

	return nil, ErrCachedDataNotFound
}

func NewGitHubWebFeaturesDataGetter(
	client *shared.GitHubWebFeaturesClient) *GitHubWebFeaturesDataGetter {
	return &GitHubWebFeaturesDataGetter{client: client}
}

type GitHubWebFeaturesDataGetter struct {
	client *shared.GitHubWebFeaturesClient
	cache  DataCacher[shared.WebFeaturesData]
}

const (
	cacheKeyLatest = "latest-web-features"
)

func (g GitHubWebFeaturesDataGetter) GetWebFeaturesData(ctx context.Context) (*shared.WebFeaturesData, error) {
	cachedData, err := g.cache.Get(ctx, cacheKeyLatest)
	if !errors.Is(err, ErrCachedDataNotFound) {
		return cachedData, nil
	}
	data, err := g.client.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &data, nil
}
