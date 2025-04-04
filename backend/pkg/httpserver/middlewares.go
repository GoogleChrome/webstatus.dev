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
	"log/slog"
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/httpmiddlewares"
	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
)

// UserCheckResult represents either a successful user retrieval or an error response.
type UserCheckResult[T any] struct {
	User     *auth.User
	Response T
}

// CheckAuthenticatedUser retrieves the authenticated user from the context.
// If the user is not found, it returns an error response.
func CheckAuthenticatedUser[T any](
	ctx context.Context,
	operation string,
	newErrorResponse func(code int, message string) T) UserCheckResult[T] {
	// At this point, the user should be authenticated and in the context.
	// If for some reason the user is not in the context, it is a library or
	// internal issue and not an user issue. Return 500 error in that case.
	user, found := httpmiddlewares.AuthenticatedUserFromContext(ctx)
	if !found {
		slog.ErrorContext(ctx, "user not found in context. middleware failure", "operation", operation)

		return UserCheckResult[T]{
			User: nil,
			Response: newErrorResponse(
				http.StatusInternalServerError,
				"internal server error",
			),
		}
	}

	var zeroT T

	return UserCheckResult[T]{
		User:     user,
		Response: zeroT,
	}
}

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
