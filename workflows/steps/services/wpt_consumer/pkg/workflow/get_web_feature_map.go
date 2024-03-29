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
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

type DataCacher[K comparable, V any] interface {
	// Cache stores a value associated with a key in the cache.
	Cache(context.Context, K, V) error
	// Get retrieves a value from the cache by its key.
	Get(context.Context, K) (V, error)
}

// NewCacheableWebFeaturesDataGetter returns a new CacheableWebFeaturesDataGetter,
// which implements caching behavior on top of an underlying WebFeatureDataGetter.
func NewCacheableWebFeaturesDataGetter(
	client WebFeatureDataGetter,
	cache DataCacher[string, shared.WebFeaturesData]) *CacheableWebFeaturesDataGetter {
	return &CacheableWebFeaturesDataGetter{
		client: client,
		cache:  cache,
	}
}

type CacheableWebFeaturesDataGetter struct {
	client WebFeatureDataGetter
	cache  DataCacher[string, shared.WebFeaturesData]
}

const (
	cacheKeyLatest = "latest-web-features"
)

// WebFeatureDataGetter defines an interface for retrieving web features data
// from some underlying source.
type WebFeatureDataGetter interface {
	Get(context.Context) (shared.WebFeaturesData, error)
}

func (g *CacheableWebFeaturesDataGetter) GetWebFeaturesData(
	ctx context.Context, _ string) (shared.WebFeaturesData, error) {
	// 1. Attempts to retrieve data from the cache.
	// 2. If not found or an unexpected cache error occurs, falls back to fetching data directly.
	// 3. Caches freshly fetched data for future requests.

	// Step 1.
	cachedData, err := g.cache.Get(ctx, cacheKeyLatest)
	if err == nil {
		return cachedData, nil
	} else if !errors.Is(err, cachetypes.ErrCachedDataNotFound) {
		slog.Warn("unexpected error when trying to get cache data", "err", err)
	}

	// Step 2.
	data, err := g.client.Get(ctx)
	if err != nil {
		return nil, err
	}

	// Step 3.
	if err := g.cache.Cache(ctx, cacheKeyLatest, data); err != nil {
		slog.Warn("unable to cache web features data", "err", err)
	}

	return data, nil
}
