// Copyright 2023 Google LLC
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
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/httpmiddlewares"
)

type WebFeatureMetadataStorer interface {
	GetFeatureMetadata(
		ctx context.Context,
		featureID string,
	) (*backend.FeatureMetadata, error)
}

type WPTMetricsStorer interface {
	ListMetricsForFeatureIDBrowserAndChannel(
		ctx context.Context,
		featureID string,
		browser string,
		channel string,
		metricView backend.MetricViewPathParam,
		startAt time.Time,
		endAt time.Time,
		pageSize int,
		pageToken *string) ([]backend.WPTRunMetric, *string, error)
	ListMetricsOverTimeWithAggregatedTotals(
		ctx context.Context,
		featureIDs []string,
		browser string,
		channel string,
		metricView backend.MetricViewPathParam,
		startAt, endAt time.Time,
		pageSize int,
		pageToken *string,
	) ([]backend.WPTRunMetric, *string, error)
	ListChromiumDailyUsageStats(
		ctx context.Context,
		featureID string,
		startAt, endAt time.Time,
		pageSize int,
		pageToken *string,
	) ([]backend.ChromiumUsageStat, *string, error)
	FeaturesSearch(
		ctx context.Context,
		pageToken *string,
		pageSize int,
		searchNode *searchtypes.SearchNode,
		sortOrder *backend.ListFeaturesParamsSort,
		wptMetricType backend.WPTMetricView,
		browsers []backend.BrowserPathParam,
	) (*backend.FeaturePage, error)
	GetFeature(
		ctx context.Context,
		featureID string,
		wptMetricType backend.WPTMetricView,
		browsers []backend.BrowserPathParam,
	) (*backend.Feature, error)
	ListBrowserFeatureCountMetric(
		ctx context.Context,
		browser string,
		startAt time.Time,
		endAt time.Time,
		pageSize int,
		pageToken *string,
	) (*backend.BrowserReleaseFeatureMetricsPage, error)
	GetIDFromFeatureKey(
		ctx context.Context,
		featureID string,
	) (*string, error)
	ListMissingOneImplCounts(
		ctx context.Context,
		targetBrowser string,
		otherBrowsers []string,
		startAt, endAt time.Time,
		pageSize int,
		pageToken *string,
	) (*backend.BrowserReleaseFeatureMetricsPage, error)
	ListBaselineStatusCounts(
		ctx context.Context,
		startAt time.Time,
		endAt time.Time,
		pageSize int,
		pageToken *string,
	) (*backend.BaselineStatusMetricsPage, error)
}

type RawBytesDataCacher interface {
	// Cache stores a value associated with a key in the cache.
	Cache(context.Context, string, []byte) error
	// Get retrieves a value from the cache by its key.
	Get(context.Context, string) ([]byte, error)
}

type ResponseDataCacher[K any, V any] interface {
	// Cache stores a value associated with a key in the cache.
	Cache(context.Context, K, V) error
	// Get retrieves a value from the cache by its key.
	Lookup(context.Context, K, *V) error
}

type Server struct {
	metadataStorer    WebFeatureMetadataStorer
	wptMetricsStorer  WPTMetricsStorer
	bytesDataCacher   RawBytesDataCacher
	operationIDCaches *operationIDCaches
}

func initOperationIDCaches(dataCacher RawBytesDataCacher) *operationIDCaches {
	return &operationIDCaches{
		getFeatureCache: httpmiddlewares.NewOperationCacheMiddleware[
			backend.GetFeatureRequestObject,
			backend.GetFeature200JSONResponse,
		](
			dataCacher, "getFeature"),
		listFeaturesCache: httpmiddlewares.NewOperationCacheMiddleware[
			backend.ListFeaturesRequestObject,
			backend.ListFeatures200JSONResponse,
		](
			dataCacher, "listFeatures"),
		getFeatureMetadataCache: httpmiddlewares.NewOperationCacheMiddleware[
			backend.GetFeatureMetadataRequestObject,
			backend.GetFeatureMetadata200JSONResponse,
		](
			dataCacher, "getFeatureMetadata"),
		listFeatureWPTMetricsCache: httpmiddlewares.NewOperationCacheMiddleware[
			backend.ListFeatureWPTMetricsRequestObject,
			backend.ListFeatureWPTMetrics200JSONResponse,
		](
			dataCacher, "listFeatureWPTMetrics"),
		listChromiumDailyUsageStatsCache: httpmiddlewares.NewOperationCacheMiddleware[
			backend.ListChromiumDailyUsageStatsRequestObject,
			backend.ListChromiumDailyUsageStats200JSONResponse,
		](
			dataCacher, "listChromiumDailyUsageStats"),
		listAggregatedFeatureSupportCache: httpmiddlewares.NewOperationCacheMiddleware[
			backend.ListAggregatedFeatureSupportRequestObject,
			backend.ListAggregatedFeatureSupport200JSONResponse,
		](
			dataCacher, "listAggregatedFeatureSupport"),
		listMissingOneImplemenationCountsCache: httpmiddlewares.NewOperationCacheMiddleware[
			backend.ListMissingOneImplemenationCountsRequestObject,
			backend.ListMissingOneImplemenationCounts200JSONResponse,
		](
			dataCacher, "listMissingOneImplemenationCounts"),
		listAggregatedWPTMetricsCache: httpmiddlewares.NewOperationCacheMiddleware[
			backend.ListAggregatedWPTMetricsRequestObject,
			backend.ListAggregatedWPTMetrics200JSONResponse,
		](
			dataCacher, "listAggregatedWPTMetrics"),
		listAggregatedBaselineStatusCountsCache: httpmiddlewares.NewOperationCacheMiddleware[
			backend.ListAggregatedBaselineStatusCountsRequestObject,
			backend.ListAggregatedBaselineStatusCounts200JSONResponse,
		](
			dataCacher, "listAggregatedBaselineStatusCounts"),
	}
}

type operationIDCaches struct {
	getFeatureCache ResponseDataCacher[
		backend.GetFeatureRequestObject,
		backend.GetFeature200JSONResponse,
	]
	listFeaturesCache ResponseDataCacher[
		backend.ListFeaturesRequestObject,
		backend.ListFeatures200JSONResponse,
	]
	getFeatureMetadataCache ResponseDataCacher[
		backend.GetFeatureMetadataRequestObject,
		backend.GetFeatureMetadata200JSONResponse,
	]
	listFeatureWPTMetricsCache ResponseDataCacher[
		backend.ListFeatureWPTMetricsRequestObject,
		backend.ListFeatureWPTMetrics200JSONResponse,
	]
	listChromiumDailyUsageStatsCache ResponseDataCacher[
		backend.ListChromiumDailyUsageStatsRequestObject,
		backend.ListChromiumDailyUsageStats200JSONResponse,
	]
	listAggregatedFeatureSupportCache ResponseDataCacher[
		backend.ListAggregatedFeatureSupportRequestObject,
		backend.ListAggregatedFeatureSupport200JSONResponse,
	]
	listMissingOneImplemenationCountsCache ResponseDataCacher[
		backend.ListMissingOneImplemenationCountsRequestObject,
		backend.ListMissingOneImplemenationCounts200JSONResponse,
	]
	listAggregatedWPTMetricsCache ResponseDataCacher[
		backend.ListAggregatedWPTMetricsRequestObject,
		backend.ListAggregatedWPTMetrics200JSONResponse,
	]
	listAggregatedBaselineStatusCountsCache ResponseDataCacher[
		backend.ListAggregatedBaselineStatusCountsRequestObject,
		backend.ListAggregatedBaselineStatusCounts200JSONResponse,
	]
}

// tryToSearchCacheForResponse will try to search the cache for a response. For any unexpected errors, log them.
func tryToSearchCacheForResponse[K any, V any](ctx context.Context, key K, dataCacher ResponseDataCacher[K, V]) (*V, bool) {
	var resp V
	err := dataCacher.Lookup(ctx, key, &resp)
	if err == nil {
		return &resp, true
	}
	if !errors.Is(err, cachetypes.ErrCachedDataNotFound) {
		slog.ErrorContext(ctx, "cache get failed for an unexpected reason", "error", err)
	}
	return nil, false
}

func cacheAndReturnResponse[K any, V any](ctx context.Context, key K, resp V, dataCacher ResponseDataCacher[K, V]) V {
	err := dataCacher.Cache(ctx, key, resp)
	if err != nil {
		slog.ErrorContext(ctx, "failed to cache response", "key", key)
	}
	return resp
}

// RemoveSavedSearch implements backend.StrictServerInterface.
// nolint: revive, ireturn // Name generated from openapi
func (s *Server) RemoveSavedSearch(
	ctx context.Context, request backend.RemoveSavedSearchRequestObject) (
	backend.RemoveSavedSearchResponseObject, error) {
	return backend.RemoveSavedSearch400JSONResponse{
		Code:    http.StatusBadRequest,
		Message: "TODO",
	}, nil
}

// UpdateSavedSearch implements backend.StrictServerInterface.
// nolint: revive, ireturn // Name generated from openapi
func (s *Server) UpdateSavedSearch(
	ctx context.Context, request backend.UpdateSavedSearchRequestObject) (
	backend.UpdateSavedSearchResponseObject, error) {
	return backend.UpdateSavedSearch400JSONResponse{
		Code:    http.StatusBadRequest,
		Message: "TODO",
	}, nil
}

// CreateSavedSearch implements backend.StrictServerInterface.
// nolint: revive, ireturn // Name generated from openapi
func (s *Server) CreateSavedSearch(ctx context.Context, request backend.CreateSavedSearchRequestObject) (
	backend.CreateSavedSearchResponseObject, error) {
	return backend.CreateSavedSearch400JSONResponse{
		Code:    http.StatusBadRequest,
		Message: "TODO",
	}, nil
}

// GetUserSavedSearchBookmark implements backend.StrictServerInterface.
// nolint: revive, ireturn // Name generated from openapi
func (s *Server) GetUserSavedSearchBookmark(
	ctx context.Context, request backend.GetUserSavedSearchBookmarkRequestObject) (
	backend.GetUserSavedSearchBookmarkResponseObject, error) {
	return backend.GetUserSavedSearchBookmark400JSONResponse{
		Code:    http.StatusBadRequest,
		Message: "TODO",
	}, nil
}

// PutUserSavedSearchBookmark implements backend.StrictServerInterface.
// nolint: revive, ireturn // Name generated from openapi
func (s *Server) PutUserSavedSearchBookmark(
	ctx context.Context, request backend.PutUserSavedSearchBookmarkRequestObject) (
	backend.PutUserSavedSearchBookmarkResponseObject, error) {
	return backend.PutUserSavedSearchBookmark400JSONResponse{
		Code:    http.StatusBadRequest,
		Message: "TODO",
	}, nil
}

// RemoveUserSavedSearchBookmark implements backend.StrictServerInterface.
// nolint: revive, ireturn // Name generated from openapi
func (s *Server) RemoveUserSavedSearchBookmark(
	ctx context.Context, request backend.RemoveUserSavedSearchBookmarkRequestObject) (
	backend.RemoveUserSavedSearchBookmarkResponseObject, error) {
	return backend.RemoveUserSavedSearchBookmark400JSONResponse{
		Code:    http.StatusBadRequest,
		Message: "TODO",
	}, nil
}

func defaultBrowsers() []backend.BrowserPathParam {
	return []backend.BrowserPathParam{
		backend.Chrome,
		backend.Edge,
		backend.Firefox,
		backend.Safari,
	}
}

func getPageSizeOrDefault(pageSize *int) int {
	// maxPageSize comes from the <repo_root>/openapi/backend/openapi.yaml
	maxPageSize := 100
	if pageSize != nil {
		if *pageSize >= 1 && *pageSize <= maxPageSize {
			return *pageSize
		}
	}

	return maxPageSize
}

func getFeatureIDsOrDefault(featureIDs *[]string) []string {
	var defaultFeatureIDs []string

	return *(cmp.Or[*[]string](featureIDs, &defaultFeatureIDs))
}

type CacheableTypes interface {
	backend.GetFeature200JSONResponse | backend.GetFeatureMetadata200JSONResponse
}

func NewHTTPServer(
	port string,
	metadataStorer WebFeatureMetadataStorer,
	wptMetricsStorer WPTMetricsStorer,
	bytesDataCacher RawBytesDataCacher,
	preRequestValidationMiddlewares []func(http.Handler) http.Handler,
	authMiddleware func(http.Handler) http.Handler) *http.Server {
	// Create an instance of our handler which satisfies the generated interface
	srv := &Server{
		metadataStorer:    metadataStorer,
		wptMetricsStorer:  wptMetricsStorer,
		bytesDataCacher:   bytesDataCacher,
		operationIDCaches: initOperationIDCaches(bytesDataCacher),
	}

	return createOpenAPIServerServer(port, srv, preRequestValidationMiddlewares, authMiddleware)
}

func createOpenAPIServerServer(
	port string,
	srv backend.StrictServerInterface,
	preRequestValidationMiddlewares []func(http.Handler) http.Handler,
	authMiddleware func(http.Handler) http.Handler) *http.Server {

	srvStrictHandler := backend.NewStrictHandler(srv,
		wrapPostRequestValidationMiddlewaresForOpenAPIHook(authMiddleware))

	// Use standard library router
	r := http.NewServeMux()

	// We now register our web feature router above as the handler for the interface
	backend.HandlerFromMux(srvStrictHandler, r)

	// Now wrap the middlewares
	wrappedHandler := applyPreRequestValidationMiddlewares(r, preRequestValidationMiddlewares)

	// nolint:exhaustruct // No need to populate 3rd party struct
	return &http.Server{
		Handler:           wrappedHandler,
		Addr:              net.JoinHostPort("0.0.0.0", port),
		ReadHeaderTimeout: 30 * time.Second,
	}
}

// GenericErrorFn is a reusable method for the middleware layers that they can use to get well structured JSON output
// for BasicErrorModel.
func GenericErrorFn(ctx context.Context, statusCode int, w http.ResponseWriter, err error) {

	var message string
	if err != nil {
		message = err.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	encoderErr := json.NewEncoder(w).Encode(backend.BasicErrorModel{
		Code:    statusCode,
		Message: message,
	})
	if err != nil {
		slog.WarnContext(ctx, "unable to write generic error", "error", encoderErr)
	}
}
