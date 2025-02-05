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
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/httpmiddlewares"
)

// TODO: Move recoveryMiddleware into the lib directory to actually be used by the real server.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(req.Context(), "Panic recovered", "error", r)

				// Return an Internal Server Error (500)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, req)
	})
}

func noopMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		next.ServeHTTP(w, req)
	})
}

func TestMiddlewaresOrder(t *testing.T) {
	count := 0
	var preMiddleware1Hit,
		preMiddleware2Hit,
		authMiddlewareHit,
		cacheMiddlewareHit bool

	preRequestMiddlewares := []func(http.Handler) http.Handler{
		recoveryMiddleware,
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				preMiddleware1Hit = true
				if count != 0 {
					t.Errorf("PreRequest Middleware 1: Expected count to be 0, got %d", count)
				}
				count++
				next.ServeHTTP(w, r)
			})
		},
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				preMiddleware2Hit = true
				if count != 1 {
					t.Errorf("PreRequest Middleware 2: Expected count to be 1, got %d", count)
				}
				count++
				next.ServeHTTP(w, r)
			})
		},
	}
	authMiddleware :=
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authMiddlewareHit = true
				if count != 2 {
					t.Errorf("Auth Middleware: Expected count to be 2, got %d", count)
				}
				if cacheMiddlewareHit {
					t.Error("cache middleware hit before auth middleware")
				}
				count++
				next.ServeHTTP(w, r)
			})
		}
	cacheMiddleware :=
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				cacheMiddlewareHit = true
				if count != 3 {
					t.Errorf("Cache Middleware: Expected count to be 3, got %d", count)
				}
				count++
				next.ServeHTTP(w, r)
			})
		}

	mockServer := &mockServerInterface{t: t, expectedUserInCtx: nil, callCount: 0}
	srv := createOpenAPIServerServer("", mockServer, preRequestMiddlewares, cacheMiddleware, authMiddleware)
	s := httptest.NewServer(srv.Handler)
	defer s.Close()

	submitRequest(t, s.URL+"/v1/features", http.MethodGet)

	if !preMiddleware1Hit ||
		!preMiddleware2Hit ||
		!authMiddlewareHit ||
		!cacheMiddlewareHit {
		t.Errorf("expected all middlewares to be hit")
	}

	mockServer.assertCallCount(1)
}

func testAuthScope(t *testing.T, path string, method string, shouldBePresent bool, expectedUserInCtx *auth.User) {
	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			value := r.Context().Value(backend.BearerAuthScopes)
			if shouldBePresent && value == nil {
				t.Error("did not find bearer auth scope, expected it")
			}
			if !shouldBePresent && value != nil {
				t.Error("found bearer auth scope, did not expect it")
			}
			if shouldBePresent && expectedUserInCtx != nil {
				ctx := r.Context()
				ctx = httpmiddlewares.AuthenticatedUserToContext(ctx, expectedUserInCtx)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
	mockServer := &mockServerInterface{t: t, expectedUserInCtx: expectedUserInCtx, callCount: 0}
	srv := createOpenAPIServerServer("", mockServer, []func(http.Handler) http.Handler{
		recoveryMiddleware}, noopMiddleware, authMiddleware)
	s := httptest.NewServer(srv.Handler)
	defer s.Close()

	submitRequest(t, s.URL+path, method)
	mockServer.assertCallCount(1)
}

// This test ensures that the third-party OpenAPI library continues to add
// Bearer authentication scopes to the request context when the route has
// security schemes configured.
func TestAuthScopePresentWhenSecurityConfigured(t *testing.T) {
	testUser := &auth.User{ID: "test"}
	testAuthScope(t, "/v1/users/me/saved-searches", http.MethodGet, true, testUser)
}

// This test ensures that the third-party OpenAPI library continues to omit
// Bearer authentication scopes from the request context when the route does
// not have security schemes configured.
func TestAuthScopeAbsentWhenSecurityNotConfigured(t *testing.T) {
	testAuthScope(t, "/v1/features", http.MethodGet, false, nil)
}
