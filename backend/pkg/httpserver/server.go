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
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
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
	FeaturesSearch(
		ctx context.Context,
		pageToken *string,
		pageSize int,
		searchNode *searchtypes.SearchNode,
		sortOrder *backend.GetV1FeaturesParamsSort,
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
}

type Server struct {
	metadataStorer   WebFeatureMetadataStorer
	wptMetricsStorer WPTMetricsStorer
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

func applyMiddlewares(mux *http.ServeMux, middlewares []func(http.Handler) http.Handler) http.Handler {
	var next http.Handler
	next = mux
	// Apply middlewares in reverse order to ensure they execute in the order they are defined.
	// This is because each middleware wraps the next one in the chain.
	for i := len(middlewares) - 1; i >= 0; i-- {
		next = middlewares[i](next)
	}

	return next
}

func NewHTTPServer(
	port string,
	metadataStorer WebFeatureMetadataStorer,
	wptMetricsStorer WPTMetricsStorer,
	middlewares []func(http.Handler) http.Handler) (*http.Server, error) {
	_, err := backend.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("error loading swagger spec. %w", err)
	}

	// Create an instance of our handler which satisfies the generated interface
	srv := &Server{
		metadataStorer:   metadataStorer,
		wptMetricsStorer: wptMetricsStorer,
	}

	srvStrictHandler := backend.NewStrictHandler(srv, nil)

	// Use standard library router
	r := http.NewServeMux()

	// Use our validation middleware to check all requests against the
	// OpenAPI schema.
	// r.Use(middleware.OapiRequestValidatorWithOptions(swagger, &middleware.Options{
	// 	SilenceServersWarning: true,
	// }))

	// We now register our web feature router above as the handler for the interface
	backend.HandlerFromMux(srvStrictHandler, r)

	// Now wrap the middleware
	wrappedHandler := applyMiddlewares(r, middlewares)

	// nolint:exhaustruct // No need to populate 3rd party struct
	return &http.Server{
		Handler:           wrappedHandler,
		Addr:              net.JoinHostPort("0.0.0.0", port),
		ReadHeaderTimeout: 30 * time.Second,
	}, nil
}
