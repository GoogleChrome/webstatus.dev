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
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// operationResponseCache caches operation results using a RawBytesDataCacher.
// It uses the `operationID` as a prefix for all cache keys, enabling
// logical grouping and potential future deletion by prefix.  It also handles
// JSON serialization/deserialization of keys and values.
type operationResponseCache[Key any, Response any] struct {
	cacher               RawBytesDataCacher
	operationID          string
	overrideCacheOptions []cachetypes.CacheOption
}

func (c operationResponseCache[Key, Response]) key(key []byte) string {
	return c.operationID + "-" + string(key)
}

// AttemptCache attempts to cache the given value, associated with the given key,
// within the underlying RawBytesDataCacher. It marshals both the key and value
// to JSON bytes before attempting to cache them. If any error occurs during
// the marshaling or caching process, it logs the error and does nothing else.
//
// Note: This method does not return an error. This is intentional because
// caching failures should not prevent the main operation from completing.
func (c operationResponseCache[Key, Response]) AttemptCache(ctx context.Context, key Key, value *Response) {
	if value == nil {
		// Should never reach here
		slog.ErrorContext(ctx, "unable to cache nil value")

		return
	}

	jsonBytesKey, err := json.Marshal(key)
	if err != nil {
		slog.ErrorContext(ctx, "unable to marshal key for cache store",
			"key", key, "error", err, "operation", c.operationID)

		return
	}
	jsonBytesValue, err := json.Marshal(*value)
	if err != nil {
		slog.ErrorContext(ctx, "unable to marshal value for cache store",
			"value", value, "error", err, "operation", c.operationID)

		return
	}

	err = c.cacher.Cache(ctx, c.key(jsonBytesKey), jsonBytesValue, c.overrideCacheOptions...)
	if err != nil {
		slog.ErrorContext(ctx, "encountered unexpected error when caching",
			"error", err, "key", key, "operation", c.operationID)
	}
}

// Lookup attempts to retrieve a cached value by key. It serializes the key to JSON,
// fetches the corresponding value from the RawBytesDataCacher, and then attempts to
// deserialize the result back to the Response type. It returns true if the value is
// found and successfully deserialized.  Returns false otherwise.  Any errors are logged.
// Errors are logged as this is a cache and shouldn't interrupt the normal flow of the program.
func (c operationResponseCache[Key, Response]) Lookup(ctx context.Context, key Key, value *Response) bool {
	jsonBytesKey, err := json.Marshal(key)
	if err != nil {
		slog.ErrorContext(ctx, "unable to marshal key for cache lookup",
			"error", err, "key", key, "operation", c.operationID)

		return false
	}

	valueBytes, err := c.cacher.Get(ctx, c.key(jsonBytesKey))
	if err != nil {
		if !errors.Is(err, cachetypes.ErrCachedDataNotFound) {
			slog.ErrorContext(ctx, "encountered unexpected error from cache",
				"error", err, "key", key, "operation", c.operationID)
		}

		return false
	}

	err = json.Unmarshal(valueBytes, value)
	if err != nil {
		slog.ErrorContext(ctx, "unable to unmarshal cached data",
			"error", err, "key", key, "operation", c.operationID, "value", string(valueBytes))

		return false
	}

	return true
}

// operationResponseCaches is a struct that holds multiple instances of
// operationResponseCache, each managing caching for a specific API operation.
// Each operationResponseCache instance wraps the underlying RawBytesDataCacher
// to provide type-safe caching and retrieval for its associated operation.
type operationResponseCaches struct {
	getFeatureCache operationResponseCache[
		backend.GetFeatureRequestObject,
		backend.GetFeature200JSONResponse,
	]
	listFeaturesCache operationResponseCache[
		backend.ListFeaturesRequestObject,
		backend.ListFeatures200JSONResponse,
	]
	getFeatureMetadataCache operationResponseCache[
		backend.GetFeatureMetadataRequestObject,
		backend.GetFeatureMetadata200JSONResponse,
	]
	listFeatureWPTMetricsCache operationResponseCache[
		backend.ListFeatureWPTMetricsRequestObject,
		backend.ListFeatureWPTMetrics200JSONResponse,
	]
	listChromiumDailyUsageStatsCache operationResponseCache[
		backend.ListChromiumDailyUsageStatsRequestObject,
		backend.ListChromiumDailyUsageStats200JSONResponse,
	]
	listAggregatedFeatureSupportCache operationResponseCache[
		backend.ListAggregatedFeatureSupportRequestObject,
		backend.ListAggregatedFeatureSupport200JSONResponse,
	]
	ListMissingOneImplementationCountsCache operationResponseCache[
		backend.ListMissingOneImplementationCountsRequestObject,
		backend.ListMissingOneImplementationCounts200JSONResponse,
	]
	listAggregatedWPTMetricsCache operationResponseCache[
		backend.ListAggregatedWPTMetricsRequestObject,
		backend.ListAggregatedWPTMetrics200JSONResponse,
	]
	listAggregatedBaselineStatusCountsCache operationResponseCache[
		backend.ListAggregatedBaselineStatusCountsRequestObject,
		backend.ListAggregatedBaselineStatusCounts200JSONResponse,
	]
}

// initOperationResponseCaches initializes and configures each
// operationResponseCache instance within the operationResponseCaches struct.
// While each cache instance uses the same underlying RawBytesDataCacher for storage,
// they operate independently and are specialized for their respective API operations.
func initOperationResponseCaches(dataCacher RawBytesDataCacher,
	routeCacheOptions RouteCacheOptions) *operationResponseCaches {
	return &operationResponseCaches{
		getFeatureCache: operationResponseCache[
			backend.GetFeatureRequestObject,
			backend.GetFeature200JSONResponse,
		]{cacher: dataCacher, operationID: "getFeature", overrideCacheOptions: nil},

		listFeaturesCache: operationResponseCache[
			backend.ListFeaturesRequestObject,
			backend.ListFeatures200JSONResponse,
		]{cacher: dataCacher, operationID: "listFeatures", overrideCacheOptions: nil},

		getFeatureMetadataCache: operationResponseCache[
			backend.GetFeatureMetadataRequestObject,
			backend.GetFeatureMetadata200JSONResponse,
		]{cacher: dataCacher, operationID: "getFeatureMetadata", overrideCacheOptions: nil},

		listFeatureWPTMetricsCache: operationResponseCache[
			backend.ListFeatureWPTMetricsRequestObject,
			backend.ListFeatureWPTMetrics200JSONResponse,
		]{cacher: dataCacher, operationID: "listFeatureWPTMetrics", overrideCacheOptions: nil},

		listChromiumDailyUsageStatsCache: operationResponseCache[
			backend.ListChromiumDailyUsageStatsRequestObject,
			backend.ListChromiumDailyUsageStats200JSONResponse,
		]{cacher: dataCacher, operationID: "listChromiumDailyUsageStats", overrideCacheOptions: nil},

		listAggregatedFeatureSupportCache: operationResponseCache[
			backend.ListAggregatedFeatureSupportRequestObject,
			backend.ListAggregatedFeatureSupport200JSONResponse,
		]{cacher: dataCacher, operationID: "listAggregatedFeatureSupport",
			overrideCacheOptions: routeCacheOptions.AggregatedFeatureStatsOptions},

		ListMissingOneImplementationCountsCache: operationResponseCache[
			backend.ListMissingOneImplementationCountsRequestObject,
			backend.ListMissingOneImplementationCounts200JSONResponse,
		]{cacher: dataCacher, operationID: "ListMissingOneImplementationCounts",
			overrideCacheOptions: routeCacheOptions.AggregatedFeatureStatsOptions},

		listAggregatedWPTMetricsCache: operationResponseCache[
			backend.ListAggregatedWPTMetricsRequestObject,
			backend.ListAggregatedWPTMetrics200JSONResponse,
		]{cacher: dataCacher, operationID: "listAggregatedWPTMetrics", overrideCacheOptions: nil},

		listAggregatedBaselineStatusCountsCache: operationResponseCache[
			backend.ListAggregatedBaselineStatusCountsRequestObject,
			backend.ListAggregatedBaselineStatusCounts200JSONResponse,
		]{cacher: dataCacher, operationID: "listAggregatedBaselineStatusCounts",
			overrideCacheOptions: routeCacheOptions.AggregatedFeatureStatsOptions},
	}
}
