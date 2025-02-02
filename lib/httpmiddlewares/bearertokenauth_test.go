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
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
)

type authCtxKey struct{}

func TestBearerTokenAuthenticationMiddleware(t *testing.T) {
	const testID = "id"
	tests := []struct {
		name               string
		ctxKey             any
		authHeader         string
		mockAuthenticator  func(ctx context.Context, token string) (*auth.User, error)
		mockErrorFn        func(context.Context, int, http.ResponseWriter, error)
		expectedStatusCode int
		expectedBody       string
		expectedUser       *auth.User
	}{
		{
			name:       "No security requirements",
			ctxKey:     nil,
			authHeader: "",
			mockAuthenticator: func(_ context.Context, _ string) (*auth.User, error) {
				t.Fatal("authenticate should not have been called")

				// nolint:nilnil // WONTFIX - should not reach this.
				return nil, nil
			},
			mockErrorFn: func(_ context.Context, _ int, _ http.ResponseWriter, _ error) {
				t.Fatal("errorFn should not have been called")
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       "next handler was called",
			expectedUser:       nil,
		},
		{
			name:       "Missing Authorization header",
			ctxKey:     authCtxKey{},
			authHeader: "",
			mockAuthenticator: func(_ context.Context, _ string) (*auth.User, error) {
				t.Fatal("authenticate should not have been called")

				// nolint:nilnil // WONTFIX - should not reach this.
				return nil, nil
			},
			mockErrorFn: func(_ context.Context, code int, w http.ResponseWriter, err error) {
				if code != http.StatusUnauthorized {
					t.Errorf("expected status code %d, got %d", http.StatusUnauthorized, code)
				}
				if !errors.Is(err, ErrMissingAuthHeader) {
					t.Errorf("expected error %v, got %v", ErrMissingAuthHeader, err)
				}
				w.WriteHeader(code)
			},
			expectedStatusCode: http.StatusUnauthorized,
			expectedUser:       nil,
			expectedBody:       "",
		},
		{
			name:       "Invalid Authorization header",
			ctxKey:     authCtxKey{},
			authHeader: "Invalid Auth",
			mockAuthenticator: func(_ context.Context, _ string) (*auth.User, error) {
				t.Fatal("authenticate should not have been called")

				// nolint:nilnil // WONTFIX - should not reach this.
				return nil, nil
			},
			mockErrorFn: func(_ context.Context, code int, w http.ResponseWriter, err error) {
				if code != http.StatusUnauthorized {
					t.Errorf("expected status code %d, got %d", http.StatusUnauthorized, code)
				}
				if !errors.Is(err, ErrInvalidAuthHeader) {
					t.Errorf("expected error %v, got %v", ErrInvalidAuthHeader, err)
				}
				w.WriteHeader(code)
			},
			expectedStatusCode: http.StatusUnauthorized,
			expectedUser:       nil,
			expectedBody:       "",
		},
		{
			name:       "Authentication failure",
			ctxKey:     authCtxKey{},
			authHeader: "Bearer my-token",
			mockAuthenticator: func(_ context.Context, _ string) (*auth.User, error) {
				return nil, errors.New("authentication failed")
			},
			mockErrorFn: func(_ context.Context, code int, w http.ResponseWriter, err error) {
				if code != http.StatusUnauthorized {
					t.Errorf("expected status code %d, got %d", http.StatusUnauthorized, code)
				}
				if err == nil || err.Error() != "authentication failed" {
					t.Errorf("expected error 'authentication failed', got %v", err)
				}
				w.WriteHeader(code)
			},
			expectedStatusCode: http.StatusUnauthorized,
			expectedUser:       nil,
			expectedBody:       "",
		},
		{
			name:       "Successful authentication",
			ctxKey:     authCtxKey{},
			authHeader: "Bearer my-token",
			mockAuthenticator: func(_ context.Context, token string) (*auth.User, error) {
				if token != "my-token" {
					t.Errorf("expected token 'my-token', got %s", token)
				}

				return &auth.User{
					ID: testID,
				}, nil
			},
			mockErrorFn: func(_ context.Context, _ int, _ http.ResponseWriter, _ error) {
				t.Fatal("errorFn should not have been called")
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       "next handler was called",
			expectedUser: &auth.User{
				ID: testID,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				u, _ := AuthenticatedUserFromContext(r.Context())
				if !reflect.DeepEqual(u, tc.expectedUser) {
					t.Errorf("expected user %+v, received user %+v", tc.expectedUser, u)
				}
				_, err := w.Write([]byte("next handler was called"))
				if err != nil {
					t.Fatal(err)
				}
			})

			middleware := NewBearerTokenAuthenticationMiddleware(
				&mockBearerTokenAuthenticator{tc.mockAuthenticator},
				tc.ctxKey,
				tc.mockErrorFn,
			)

			handler := middleware(nextHandler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			if tc.ctxKey != nil {
				req = req.WithContext(context.WithValue(req.Context(), tc.ctxKey, "authCtxValue"))
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assertStatusCode(t, rr, tc.expectedStatusCode)
			assertResponseBody(t, rr, tc.expectedBody)
		})
	}
}

type mockBearerTokenAuthenticator struct {
	authenticateFn func(ctx context.Context, token string) (*auth.User, error)
}

func (m *mockBearerTokenAuthenticator) Authenticate(ctx context.Context, token string) (*auth.User, error) {
	if m.authenticateFn == nil {
		panic("authenticateFn not set")
	}

	return m.authenticateFn(ctx, token)
}

func assertStatusCode(t *testing.T, rr *httptest.ResponseRecorder, expectedCode int) {
	t.Helper()
	if rr.Code != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, rr.Code)
	}
}

func assertResponseBody(t *testing.T, rr *httptest.ResponseRecorder, expectedBody string) {
	t.Helper()
	if expectedBody != "" && rr.Body.String() != expectedBody {
		t.Errorf("expected body '%s', got '%s'", expectedBody, rr.Body.String())
	}
}
