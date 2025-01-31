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
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/httpmiddlewares"
	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
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
}

type Server struct {
	metadataStorer   WebFeatureMetadataStorer
	wptMetricsStorer WPTMetricsStorer
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

func applyPreRequestMiddlewares(mux *http.ServeMux, middlewares []func(http.Handler) http.Handler) http.Handler {
	var next http.Handler
	next = mux
	// Apply middlewares in reverse order to ensure they execute in the order they are defined.
	// This is because each middleware wraps the next one in the chain.
	for i := len(middlewares) - 1; i >= 0; i-- {
		next = middlewares[i](next)
	}

	return next
}

// contextMiddlewareWrapper is a helper function that wraps a StrictHTTPHandlerFunc
// to ensure that any custom context values set by external middlewares (such as
// authentication middlewares) are properly propagated to the handler function.
// This is necessary because the generated OpenAPI code does not automatically
// pass the modified context to the handler.
//
// By using this wrapper, you can keep your middlewares generic and use the standard
// http.Handler signature without having to know about the OpenAPI-specific
// nethttp.StrictHTTPHandlerFunc. This wrapper handles the adaptation and ensures
// that the context is passed correctly.
func contextMiddlewareWrapper(next nethttp.StrictHTTPHandlerFunc) nethttp.StrictHTTPHandlerFunc {
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

// wrapPostRequestValidationMiddlewares adapts a list of standard HTTP middleware to be compatible
// with the StrictMiddlewareFunc type. The order of middlewares is preserved,
// ensuring that the first middleware in the input slice is the first to be
// applied to an incoming request, and the last middleware in the slice is
// the last to be applied.
func wrapPostRequestValidationMiddlewares(
	middlewares []func(http.Handler) http.Handler) []backend.StrictMiddlewareFunc {
	strictMiddlewares := make([]backend.StrictMiddlewareFunc, len(middlewares))

	for i := range middlewares {
		// Calculate the reversed index
		j := len(middlewares) - 1 - i
		strictMiddlewares[j] = func(f nethttp.StrictHTTPHandlerFunc, _ string) nethttp.StrictHTTPHandlerFunc {

			// This is the adapter function that gets called on each request.
			return func(ctx context.Context, w http.ResponseWriter,
				r *http.Request, req interface{}) (response interface{}, err error) {
				// Create the handler.
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					response, err = contextMiddlewareWrapper(f)(ctx, w, r, req)
				})

				// Wrap the adapted handler with the standard middleware.
				wrappedHandler := middlewares[i](handler)
				wrappedHandler.ServeHTTP(w, r)

				return response, err
			}
		}
	}

	return strictMiddlewares
}

func createHTTPServer(port string, srv backend.StrictServerInterface,
	preRequestMiddlewares, postRequestValidationMiddlewares []func(http.Handler) http.Handler) *http.Server {
	srvStrictHandler := backend.NewStrictHandler(srv,
		wrapPostRequestValidationMiddlewares(postRequestValidationMiddlewares))

	// Use standard library router
	r := http.NewServeMux()

	// We now register our web feature router above as the handler for the interface
	backend.HandlerFromMux(srvStrictHandler, r)

	// Now wrap the middleware
	wrappedHandler := applyPreRequestMiddlewares(r, preRequestMiddlewares)

	// nolint:exhaustruct // No need to populate 3rd party struct
	return &http.Server{
		Handler:           wrappedHandler,
		Addr:              net.JoinHostPort("0.0.0.0", port),
		ReadHeaderTimeout: 30 * time.Second,
	}
}

func NewHTTPServer(
	port string,
	metadataStorer WebFeatureMetadataStorer,
	wptMetricsStorer WPTMetricsStorer,
	preRequestMiddlewares, postRequestValidationMiddlewares []func(http.Handler) http.Handler) *http.Server {
	// Create an instance of our handler which satisfies the generated interface
	srv := &Server{
		metadataStorer:   metadataStorer,
		wptMetricsStorer: wptMetricsStorer,
	}

	return createHTTPServer(port, srv, preRequestMiddlewares, postRequestValidationMiddlewares)
}
