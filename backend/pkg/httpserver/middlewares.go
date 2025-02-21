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
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/httpmiddlewares"
	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
)

// applyPreRequestValidationMiddlewares applies a list of middleware functions to a given http.Handler.
// The middlewares are applied in reverse order to ensure they execute in the order they are defined.
func applyPreRequestValidationMiddlewares(mux *http.ServeMux,
	middlewares []func(http.Handler) http.Handler) http.Handler {
	var next http.Handler
	next = mux
	for i := len(middlewares) - 1; i >= 0; i-- {
		next = middlewares[i](next)
	}

	return next
}

// wrapPostRequestValidationMiddlewaresForOpenAPIHook creates a wrapper function for each middleware that
// requires post-request validation. The wrapper function adapts the middleware to the signature expected by the
// OpenAPI generator.
func wrapPostRequestValidationMiddlewaresForOpenAPIHook(
	authMiddleware func(http.Handler) http.Handler) []backend.StrictMiddlewareFunc {
	openAPIMiddlewares := make([]backend.StrictMiddlewareFunc, 1)
	// OpenAPI middlewares need to inserted in reverse order.
	// This is an implementation detail for the current OpenAPI Generator.
	openAPIMiddlewares[0] = wrapPostRequestValidationMiddlewareForOpenAPIHook(
		authMiddleware, authMiddlewareOpenAPIHook)

	return openAPIMiddlewares
}

// authMiddlewareOpenAPIHook is a wrapper function for the auth middleware that ensures the authenticated user is
// passed to the handler.
func authMiddlewareOpenAPIHook(next nethttp.StrictHTTPHandlerFunc) nethttp.StrictHTTPHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, req interface{}) (interface{}, error) {
		// Get the authenticated user from the request context
		user, ok := httpmiddlewares.AuthenticatedUserFromContext(r.Context())
		if ok {
			// Set the user in the context that will be passed to the handler
			ctx = httpmiddlewares.AuthenticatedUserToContext(ctx, user)
		}

		// Call the next handler with the updated context
		return next(ctx, w, r, req)
	}
}

// wrapPostRequestValidationMiddlewareForOpenAPIHook creates a wrapper function for a given middleware.
// The wrapper function adapts the middleware to the signature expected by the OpenAPI generator.
func wrapPostRequestValidationMiddlewareForOpenAPIHook(middleware func(http.Handler) http.Handler,
	openAPIHook func(nethttp.StrictHTTPHandlerFunc) nethttp.StrictHTTPHandlerFunc) backend.StrictMiddlewareFunc {
	return func(f nethttp.StrictHTTPHandlerFunc, _ string) nethttp.StrictHTTPHandlerFunc {

		// This is the adapter function that gets called on each request.
		return func(ctx context.Context, w http.ResponseWriter,
			r *http.Request, req interface{}) (response interface{}, err error) {
			// Create the handler.
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response, err = openAPIHook(f)(ctx, w, r, req)
			})

			// Wrap the adapted handler with the standard middleware.
			wrappedHandler := middleware(handler)
			wrappedHandler.ServeHTTP(w, r)

			return response, err
		}
	}
}

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
	listMissingOneImplemenationCountsCache operationResponseCache[
		backend.ListMissingOneImplemenationCountsRequestObject,
		backend.ListMissingOneImplemenationCounts200JSONResponse,
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

func initOperationResponseCaches(dataCacher RawBytesDataCacher) *operationResponseCaches {
	return &operationResponseCaches{
		getFeatureCache: operationResponseCache[
			backend.GetFeatureRequestObject,
			backend.GetFeature200JSONResponse,
		]{cacher: dataCacher, operationID: "getFeature"},

		listFeaturesCache: operationResponseCache[
			backend.ListFeaturesRequestObject,
			backend.ListFeatures200JSONResponse,
		]{cacher: dataCacher, operationID: "listFeatures"},

		getFeatureMetadataCache: operationResponseCache[
			backend.GetFeatureMetadataRequestObject,
			backend.GetFeatureMetadata200JSONResponse,
		]{cacher: dataCacher, operationID: "getFeatureMetadata"},

		listFeatureWPTMetricsCache: operationResponseCache[
			backend.ListFeatureWPTMetricsRequestObject,
			backend.ListFeatureWPTMetrics200JSONResponse,
		]{cacher: dataCacher, operationID: "listFeatureWPTMetrics"},

		listChromiumDailyUsageStatsCache: operationResponseCache[
			backend.ListChromiumDailyUsageStatsRequestObject,
			backend.ListChromiumDailyUsageStats200JSONResponse,
		]{cacher: dataCacher, operationID: "listChromiumDailyUsageStats"},

		listAggregatedFeatureSupportCache: operationResponseCache[
			backend.ListAggregatedFeatureSupportRequestObject,
			backend.ListAggregatedFeatureSupport200JSONResponse,
		]{cacher: dataCacher, operationID: "listAggregatedFeatureSupport"},

		listMissingOneImplemenationCountsCache: operationResponseCache[
			backend.ListMissingOneImplemenationCountsRequestObject,
			backend.ListMissingOneImplemenationCounts200JSONResponse,
		]{cacher: dataCacher, operationID: "listMissingOneImplemenationCounts"},

		listAggregatedWPTMetricsCache: operationResponseCache[
			backend.ListAggregatedWPTMetricsRequestObject,
			backend.ListAggregatedWPTMetrics200JSONResponse,
		]{cacher: dataCacher, operationID: "listAggregatedWPTMetrics"},

		listAggregatedBaselineStatusCountsCache: operationResponseCache[
			backend.ListAggregatedBaselineStatusCountsRequestObject,
			backend.ListAggregatedBaselineStatusCounts200JSONResponse,
		]{cacher: dataCacher, operationID: "listAggregatedBaselineStatusCounts"},
	}
}

type operationResponseCache[Key any, Response any] struct {
	cacher      RawBytesDataCacher
	operationID string
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

	err = c.cacher.Cache(ctx, c.key(jsonBytesKey), jsonBytesValue)
	if err != nil {
		slog.ErrorContext(ctx, "encountered unexpected error when caching",
			"error", err, "key", key, "operation", c.operationID)
	}
}

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
