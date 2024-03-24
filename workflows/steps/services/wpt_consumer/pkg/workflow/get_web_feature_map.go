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

func NewCacheableWebFeaturesDataGetter(
	client WebFeatureDataGetter,
	cache DataCacher[shared.WebFeaturesData]) *CacheableWebFeaturesDataGetter {
	return &CacheableWebFeaturesDataGetter{
		client: client,
		cache:  cache,
	}
}

type CacheableWebFeaturesDataGetter struct {
	client WebFeatureDataGetter
	cache  DataCacher[shared.WebFeaturesData]
}

const (
	cacheKeyLatest = "latest-web-features"
)

type WebFeatureDataGetter interface {
	Get(context.Context) (shared.WebFeaturesData, error)
}

func (g *CacheableWebFeaturesDataGetter) GetWebFeaturesData(
	ctx context.Context, _ string) (*shared.WebFeaturesData, error) {
	cachedData, err := g.cache.Get(ctx, cacheKeyLatest)
	if err == nil {
		return cachedData, nil
	} else if !errors.Is(err, cachetypes.ErrCachedDataNotFound) {
		slog.Warn("unexpected error when trying to get cache data", "err", err)
	}
	data, err := g.client.Get(ctx)
	if err != nil {
		return nil, err
	}

	if err := g.cache.Cache(ctx, cacheKeyLatest, &data); err != nil {
		slog.Warn("unable to cache web features data", "err", err)
	}

	return &data, nil
}
