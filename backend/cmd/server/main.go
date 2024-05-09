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

package main

import (
	"cmp"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/GoogleChrome/webstatus.dev/backend/pkg/httpserver"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters"
	"github.com/GoogleChrome/webstatus.dev/lib/gds"
	"github.com/GoogleChrome/webstatus.dev/lib/gds/datastoreadapters"
	"github.com/GoogleChrome/webstatus.dev/lib/httpmiddlewares"
	"github.com/GoogleChrome/webstatus.dev/lib/rediscache"
	"github.com/go-chi/cors"
)

func main() {
	var datastoreDB *string
	if value, found := os.LookupEnv("DATASTORE_DATABASE"); found {
		datastoreDB = &value
	}
	projectID := os.Getenv("PROJECT_ID")
	fs, err := gds.NewDatastoreClient(projectID, datastoreDB)
	if err != nil {
		slog.Error("failed to create datastore client", "error", err.Error())
		os.Exit(1)
	}

	spannerDB := os.Getenv("SPANNER_DATABASE")
	spannerInstance := os.Getenv("SPANNER_INSTANCE")
	spannerClient, err := gcpspanner.NewSpannerClient(projectID, spannerInstance, spannerDB)
	if err != nil {
		slog.Error("failed to create spanner client", "error", err.Error())
		os.Exit(1)
	}

	if _, found := os.LookupEnv("SPANNER_EMULATOR_HOST"); found {
		slog.Info("setting spanner to local mode")
		spannerClient.SetFeatureSearchBaseQuery(gcpspanner.LocalFeatureBaseQuery{})
	}

	// Allowed Origin. Can remove after UbP.
	allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")

	redisHost := os.Getenv("REDISHOST")
	redisPort := os.Getenv("REDISPORT")

	cacheDuration := os.Getenv("CACHE_TTL")
	duration, err := time.ParseDuration(cacheDuration)
	if err != nil {
		slog.Error("unable to parse CACHE_TTL duration", "input value", cacheDuration)
		os.Exit(1)
	}

	connectionsStr := os.Getenv("CACHE_CONNECTIONS")
	connections := 10
	if connectionsStr != "" {
		var parseErr error
		connections, parseErr = strconv.Atoi(connectionsStr)
		if parseErr != nil {
			slog.Error("unable to parse cache connections", "input", connectionsStr)
			os.Exit(1)
		}
	}

	cacheKeyPrefix := cmp.Or[string](os.Getenv("K_REVISION"), "test-revision")
	slog.Info("cache settings", "duration", duration, "prefix", cacheKeyPrefix, "connections", connections)

	cache, err := rediscache.NewRedisDataCache[string, []byte](
		cacheKeyPrefix,
		redisHost,
		redisPort,
		duration,
		connections,
	)
	if err != nil {
		slog.Error("unable to create redis cache instance", "error", err)
		os.Exit(1)
	}

	srv, err := httpserver.NewHTTPServer(
		"8080",
		datastoreadapters.NewBackend(fs),
		spanneradapters.NewBackend(spannerClient),
		[]func(http.Handler) http.Handler{
			cors.Handler(
				//nolint: exhaustruct // No need to use every option of 3rd party struct.
				cors.Options{
					AllowedOrigins: []string{allowedOrigin},
					// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
					AllowedMethods: []string{"GET", "OPTIONS"},
					// AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
					// ExposedHeaders:   []string{"Link"},
					AllowCredentials: true, // Remove after UbP
					MaxAge:           300,  // Maximum value not ignored by any of major browsers
				}),
			httpmiddlewares.NewCacheMiddleware(cache),
		},
	)
	if err != nil {
		slog.Error("unable to create server", "error", err.Error())
		os.Exit(1)
	}
	err = srv.ListenAndServe()
	if err != nil {
		slog.Error("unable to start server", "error", err.Error())
		os.Exit(1)
	}
}
