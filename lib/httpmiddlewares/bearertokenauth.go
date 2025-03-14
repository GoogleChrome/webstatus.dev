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

package httpmiddlewares

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
)

var (
	ErrMissingAuthHeader = errors.New("missing authorization header")
	ErrInvalidAuthHeader = errors.New("authorization header is malformed")
)

type authenticatedUserCtxKey struct{}

type BearerTokenAuthenticator interface {
	Authenticate(ctx context.Context, token string) (*auth.User, error)
}

// NewBearerTokenAuthenticationMiddleware returns a middleware that can be used to authenticate requests.
// It detects if a route requires authentication by checking if a field (authCtxKey) is set in the request context.
// If the authCtxKey field is set and the Authorization header is present, the middleware authenticates the user and
// sets the authenticated user in the context. If both authCtxKey and optionalAuthCtxKey fields are set and the
// Authorization header is not present, it allows the request to proceed without authentication.
//
// The errorFn parameter allows the caller to customize the error response returned when authentication fails.
// This makes the middleware more generic and adaptable to different error handling requirements.
//
// It is the responsibility of the caller of this middleware to ensure that the `authCtxKey` is set in the request
// context whenever authentication is needed. This can be done using a wrapper middleware that knows about the OpenAPI
// generator's security semantics.
//
// See https://github.com/oapi-codegen/oapi-codegen/issues/518 for details on the lack of per-endpoint middleware
// support.
func NewBearerTokenAuthenticationMiddleware(authenticator BearerTokenAuthenticator,
	authCtxKey any, optionalAuthCtxKey any,
	errorFn func(context.Context, int, http.ResponseWriter, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			value := r.Context().Value(authCtxKey)
			if value == nil {
				// The route does not have any security requirements set for it.
				next.ServeHTTP(w, r)

				return
			}
			optionalAuthValue := r.Context().Value(optionalAuthCtxKey)
			authHdr := r.Header.Get("Authorization")
			// Check for the Authorization header.
			if authHdr == "" && optionalAuthValue != nil {
				// optionalAuthCtxKey is set and no Authorization header, proceed without authentication.
				next.ServeHTTP(w, r)

				return
			}

			if authHdr == "" {
				errorFn(r.Context(), http.StatusUnauthorized, w, ErrMissingAuthHeader)

				return
			}
			prefix := "Bearer "
			if !strings.HasPrefix(authHdr, prefix) {
				errorFn(r.Context(), http.StatusUnauthorized, w, ErrInvalidAuthHeader)

				return
			}

			u, err := authenticator.Authenticate(r.Context(), strings.TrimPrefix(authHdr, prefix))
			if err != nil {
				errorFn(r.Context(), http.StatusUnauthorized, w, err)

				return
			}

			ctx := r.Context()

			ctx = AuthenticatedUserToContext(ctx, u)

			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// AuthenticatedUserFromContext attempts to get the user from the given context.
func AuthenticatedUserFromContext(ctx context.Context) (u *auth.User, ok bool) {
	u, ok = ctx.Value(authenticatedUserCtxKey{}).(*auth.User)

	return
}

// AuthenticatedUserToContext creates a new context with the user added to it.
func AuthenticatedUserToContext(ctx context.Context, u *auth.User) context.Context {
	return context.WithValue(ctx, authenticatedUserCtxKey{}, u)
}
