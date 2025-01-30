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

// NewBearerTokenAuthenticationMiddleware returns a middleware that can be used across all routes.
// Until the openapi generator supports per-endpoint middleware [1], we will need to detect if there are security
// requirements for a particular endpoint. Lucky enough, the openapi generated code sets the BearerAuthScopes
// in the request context. If it is present, we can assume that this route needs authentication.
//
// [1] https://github.com/oapi-codegen/oapi-codegen/issues/518
func NewBearerTokenAuthenticationMiddleware(authenticator BearerTokenAuthenticator, ctxKey any,
	errorFn func(context.Context, int, http.ResponseWriter, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			value := r.Context().Value(ctxKey)
			if value == nil {
				// The route does not have any security set for it.
				next.ServeHTTP(w, r)

				return
			}
			authHdr := r.Header.Get("Authorization")
			// Check for the Authorization header.
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

			ctx = context.WithValue(ctx, authenticatedUserCtxKey{}, u)

			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

func AuthenticatedUserFromContext(ctx context.Context) (u *auth.User, ok bool) {
	u, ok = ctx.Value(authenticatedUserCtxKey{}).(*auth.User)

	return
}
