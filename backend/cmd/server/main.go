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
	"context"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"time"

	firebase "firebase.google.com/go/v4"
	"github.com/GoogleChrome/webstatus.dev/backend/pkg/httpserver"
	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters"
	"github.com/GoogleChrome/webstatus.dev/lib/gds"
	"github.com/GoogleChrome/webstatus.dev/lib/gds/datastoreadapters"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/httpmiddlewares"
	"github.com/GoogleChrome/webstatus.dev/lib/opentelemetry"
	"github.com/GoogleChrome/webstatus.dev/lib/valkeycache"
	"github.com/go-chi/cors"
)

func parseEnvVarDuration(key string) time.Duration {
	cacheDuration := os.Getenv(key)
	duration, err := time.ParseDuration(cacheDuration)
	if err != nil {
		slog.ErrorContext(context.TODO(), "unable to parse duration", "key", key, "input value", cacheDuration)
		os.Exit(1)
	}

	return duration
}

func main() {
	var datastoreDB *string
	if value, found := os.LookupEnv("DATASTORE_DATABASE"); found {
		datastoreDB = &value
	}
	projectID := os.Getenv("PROJECT_ID")
	fs, err := gds.NewDatastoreClient(projectID, datastoreDB)
	if err != nil {
		slog.ErrorContext(context.TODO(), "failed to create datastore client", "error", err.Error())
		os.Exit(1)
	}

	ctx := context.Background()

	spannerDB := os.Getenv("SPANNER_DATABASE")
	spannerInstance := os.Getenv("SPANNER_INSTANCE")
	spannerClient, err := gcpspanner.NewSpannerClient(projectID, spannerInstance, spannerDB)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create spanner client", "error", err.Error())
		os.Exit(1)
	}

	if _, found := os.LookupEnv("SPANNER_EMULATOR_HOST"); found {
		slog.InfoContext(ctx, "setting spanner to local mode")
		spannerClient.SetFeatureSearchBaseQuery(gcpspanner.LocalFeatureBaseQuery{})
		spannerClient.SetMisingOneImplementationQuery(gcpspanner.LocalMissingOneImplementationQuery{})
	}

	// Allowed Origin. Can remove after UbP.
	allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")

	valkeyHost := os.Getenv("VALKEYHOST")
	valkeyPort := os.Getenv("VALKEYPORT")

	cacheTTL := parseEnvVarDuration("CACHE_TTL")

	cacheKeyPrefix := cmp.Or[string](os.Getenv("K_REVISION"), "test-revision")
	aggregatedFeaturesStatsTTL := parseEnvVarDuration("AGGREGATED_FEATURE_STATS_TTL")
	routeCacheOptions := httpserver.RouteCacheOptions{
		AggregatedFeatureStatsOptions: []cachetypes.CacheOption{
			cachetypes.WithTTL(aggregatedFeaturesStatsTTL),
		},
	}
	slog.InfoContext(ctx, "cache settings", "duration", cacheTTL, "prefix", cacheKeyPrefix,
		"aggregatedFeaturesStatsTTL", aggregatedFeaturesStatsTTL)

	cache, err := valkeycache.NewValkeyDataCache[string, []byte](
		cacheKeyPrefix,
		valkeyHost,
		valkeyPort,
		cacheTTL,
	)
	if err != nil {
		slog.ErrorContext(ctx, "unable to create valkey cache instance", "error", err)
		os.Exit(1)
	}

	// nolint:exhaustruct // WONTFIX - will rely on the defaults on this third party struct.
	firebaseApp, err := firebase.NewApp(context.Background(), &firebase.Config{
		ProjectID: projectID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "error initializing firebase app", "error", err)
		os.Exit(1)
	}

	var firebaseAuthClient auth.UserAuthClient
	// Access Auth service from default app
	firebaseBaseAuthClient, err := firebaseApp.Auth(context.Background())
	if err != nil {
		slog.ErrorContext(ctx, "error getting Auth client", "error", err)
	}

	if firebaseTenantID, found := os.LookupEnv("FIREBASE_AUTH_TENANT_ID"); found {
		tenantClient, err := firebaseBaseAuthClient.TenantManager.AuthForTenant(firebaseTenantID)
		if err != nil {
			slog.ErrorContext(ctx, "error initializing firebase tenant client", "error", err)
			os.Exit(1)
		}
		slog.InfoContext(ctx, "using tenant firebase auth client")
		firebaseAuthClient = tenantClient
	} else {
		slog.InfoContext(ctx, "using non tenant firebase auth client")
		firebaseAuthClient = firebaseBaseAuthClient
	}

	authMiddleware := httpmiddlewares.NewBearerTokenAuthenticationMiddleware(
		auth.NewGCIPAuthenticator(firebaseAuthClient),
		backend.BearerAuthScopes,
		backend.NoAuthScopes,
		httpserver.GenericErrorFn,
	)

	preRequestMiddlewares := []func(http.Handler) http.Handler{
		cors.Handler(
			//nolint: exhaustruct // No need to use every option of 3rd party struct.
			cors.Options{
				AllowedOrigins:   []string{allowedOrigin, "http://*"},
				AllowedMethods:   []string{"GET", "OPTIONS", "PATCH", "DELETE", "PUT", "POST"},
				AllowedHeaders:   []string{"Authorization"},
				AllowCredentials: true, // Remove after UbP
				MaxAge:           300,  // Maximum value not ignored by any of major browsers
			}),
	}

	if os.Getenv("OTEL_SERVICE_NAME") != "" {
		slog.InfoContext(ctx, "opentelemetry settings detected.")
		otelProjectID := os.Getenv("OTEL_GCP_PROJECT_ID")
		if otelProjectID == "" {
			slog.ErrorContext(ctx, "missing project id for opentelemetry")
			os.Exit(1)
		}
		shutdown, err := opentelemetry.SetupOpenTelemetry(ctx, otelProjectID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to setup opentelemetry", "error", err.Error())
			os.Exit(1)
		}
		defer func() {
			err := shutdown(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "unable to shutdown opentelemetry")
			}
		}()
		// Prepend the opentelemtry middleware
		preRequestMiddlewares = slices.Insert(preRequestMiddlewares, 0, opentelemetry.NewOpenTelemetryChiMiddleware())
	}

	srv := httpserver.NewHTTPServer(
		"8080",
		datastoreadapters.NewBackend(fs),
		spanneradapters.NewBackend(spannerClient),
		cache,
		routeCacheOptions,
		preRequestMiddlewares,
		authMiddleware,
	)

	err = srv.ListenAndServe()
	if err != nil {
		slog.ErrorContext(ctx, "unable to start server", "error", err.Error())
		os.Exit(1)
	}
}
