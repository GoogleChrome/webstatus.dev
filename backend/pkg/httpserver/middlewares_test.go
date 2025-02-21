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
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
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

func TestMiddlewaresOrder(t *testing.T) {
	count := 0
	var preMiddleware1Hit,
		preMiddleware2Hit,
		authMiddlewareHit bool

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
				count++
				next.ServeHTTP(w, r)
			})
		}

	mockServer := &mockServerInterface{t: t, expectedUserInCtx: nil, callCount: 0}
	srv := createOpenAPIServerServer("", mockServer, preRequestMiddlewares, authMiddleware)
	s := httptest.NewServer(srv.Handler)
	defer s.Close()

	submitRequest(t, s.URL+"/v1/features", http.MethodGet)

	if !preMiddleware1Hit ||
		!preMiddleware2Hit ||
		!authMiddlewareHit {
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
		recoveryMiddleware}, authMiddleware)
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

// MockRawBytesDataCacher is a mock implementation of RawBytesDataCacher for testing.
type MockRawBytesDataCacher struct {
	t                   *testing.T
	expectedCacheCalls  []*ExpectedCacheCall
	expectedGetCalls    []*ExpectedGetCall
	actualCacheCalls    []*ActualCacheCall
	actualGetCalls      []*ActualGetCall
	currentCacheCallIdx int
	currentGetCallIdx   int
}

// NewMockRawBytesDataCacher creates a new MockRawBytesDataCacher.
func NewMockRawBytesDataCacher(
	t *testing.T,
	expectedCacheCalls []*ExpectedCacheCall,
	expectedGetCalls []*ExpectedGetCall) *MockRawBytesDataCacher {
	return &MockRawBytesDataCacher{
		t:                   t,
		expectedCacheCalls:  expectedCacheCalls,
		expectedGetCalls:    expectedGetCalls,
		actualCacheCalls:    []*ActualCacheCall{},
		actualGetCalls:      []*ActualGetCall{},
		currentCacheCallIdx: 0,
		currentGetCallIdx:   0,
	}
}

// Cache implements the Cache method of the RawBytesDataCacher interface.
func (m *MockRawBytesDataCacher) Cache(_ context.Context, key string, value []byte) error {
	if m.currentCacheCallIdx >= len(m.expectedCacheCalls) {
		m.t.Errorf("unexpected call to Cache with key: %s", key)

		return nil
	}

	expectedCall := m.expectedCacheCalls[m.currentCacheCallIdx]
	if expectedCall.Key != key {
		m.t.Errorf("expected Cache key: %s, got: %s", expectedCall.Key, key)
	}
	if !reflect.DeepEqual(expectedCall.Value, value) {
		m.t.Errorf("expected Cache value: %v, got: %v", string(expectedCall.Value), string(value))
	}

	m.actualCacheCalls = append(m.actualCacheCalls, &ActualCacheCall{
		Key:   key,
		Value: value,
	})
	m.currentCacheCallIdx++

	return nil
}

// Get implements the Get method of the RawBytesDataCacher interface.
func (m *MockRawBytesDataCacher) Get(_ context.Context, key string) ([]byte, error) {
	if m.currentGetCallIdx >= len(m.expectedGetCalls) {
		m.t.Errorf("unexpected call to Get with key: %s", key)

		return nil, nil
	}

	expectedCall := m.expectedGetCalls[m.currentGetCallIdx]
	if expectedCall.Key != key {
		m.t.Errorf("expected Get key: %s, got: %s", expectedCall.Key, key)
	}

	m.actualGetCalls = append(m.actualGetCalls, &ActualGetCall{
		Key: key,
	})

	m.currentGetCallIdx++

	return expectedCall.Value, expectedCall.Err
}

// AssertExpectations asserts that all expected calls were made.
func (m *MockRawBytesDataCacher) AssertExpectations() {
	if len(m.expectedCacheCalls) != m.currentCacheCallIdx {
		m.t.Errorf("expected %d Cache calls, got %d", len(m.expectedCacheCalls), m.currentCacheCallIdx)
		for i, expectedCall := range m.expectedCacheCalls {
			if i < len(m.actualCacheCalls) {
				actualCall := m.actualCacheCalls[i]
				m.t.Errorf("Expected Cache Call: %+v, got: %+v \n", expectedCall, actualCall)
			} else {
				m.t.Errorf("Expected Cache Call: %+v\n", expectedCall)
			}
		}
	}

	if len(m.expectedGetCalls) != m.currentGetCallIdx {
		m.t.Errorf("expected %d Get calls, got %d", len(m.expectedGetCalls), m.currentGetCallIdx)
		for i, expectedCall := range m.expectedGetCalls {
			if i < len(m.actualGetCalls) {
				actualCall := m.actualGetCalls[i]
				m.t.Errorf("Expected Get Call: %+v, got: %+v \n", expectedCall, actualCall)
			} else {
				m.t.Errorf("Expected Get Call: %+v\n", expectedCall)
			}
		}
	}
}

// ExpectedCacheCall represents an expected call to Cache.
type ExpectedCacheCall struct {
	Key   string
	Value []byte
}

// ActualCacheCall represents an actual call made to Cache.
type ActualCacheCall struct {
	Key   string
	Value []byte
}

// ExpectedGetCall represents an expected call to Get.
type ExpectedGetCall struct {
	Key   string
	Value []byte
	Err   error
}

// ActualGetCall represents an actual call made to Get.
type ActualGetCall struct {
	Key string
}

// Test Types.
type TestKey struct {
	ID    string
	Param string
}

type TestValue struct {
	Name  string
	Value int
}

func TestOperationResponseCache_AttemptCache(t *testing.T) {
	testCases := []struct {
		name               string
		key                TestKey
		value              *TestValue
		expectedCacheCalls []*ExpectedCacheCall
	}{
		{
			name: "Valid Cache Operation",
			key: TestKey{
				ID:    "test-id",
				Param: "test-param",
			},
			value: &TestValue{
				Name:  "Test Item",
				Value: 123,
			},
			expectedCacheCalls: []*ExpectedCacheCall{
				{
					Key:   `customOperation-{"ID":"test-id","Param":"test-param"}`,
					Value: []byte(`{"Name":"Test Item","Value":123}`),
				},
			},
		},
		{
			name: "Nil value - should not cache",
			key: TestKey{
				ID:    "test-id",
				Param: "test-param",
			},
			value:              nil,
			expectedCacheCalls: []*ExpectedCacheCall{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCacher := NewMockRawBytesDataCacher(t, tc.expectedCacheCalls, nil)
			cache := operationResponseCache[
				TestKey,
				TestValue,
			]{cacher: mockCacher, operationID: "customOperation"}

			cache.AttemptCache(context.Background(), tc.key, tc.value)
			mockCacher.AssertExpectations()
		})
	}
}

func TestOperationResponseCache_Lookup(t *testing.T) {
	testCases := []struct {
		name             string
		key              TestKey
		expectedGetCalls []*ExpectedGetCall
		expectedResult   bool
		expectedValue    *TestValue
	}{
		{
			name: "Valid Lookup - Data Found",
			key: TestKey{
				ID:    "test-id",
				Param: "test-param",
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `customOperation-{"ID":"test-id","Param":"test-param"}`,
					Value: []byte(`{"Name":"Test Item","Value":123}`),
					Err:   nil,
				},
			},
			expectedResult: true,
			expectedValue: &TestValue{
				Name:  "Test Item",
				Value: 123,
			},
		},
		{
			name: "Lookup - Data Not Found",
			key: TestKey{
				ID:    "test-id",
				Param: "test-param",
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `customOperation-{"ID":"test-id","Param":"test-param"}`,
					Value: nil,
					Err:   cachetypes.ErrCachedDataNotFound,
				},
			},
			expectedResult: false,
			expectedValue:  nil,
		},
		{
			name: "Lookup - Cache Error",
			key: TestKey{
				ID:    "test-id",
				Param: "test-param",
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `customOperation-{"ID":"test-id","Param":"test-param"}`,
					Value: nil,
					Err:   errors.New("some error"),
				},
			},
			expectedResult: false,
			expectedValue:  nil,
		},
		{
			name: "Lookup - invalid cached value",
			key: TestKey{
				ID:    "test-id",
				Param: "test-param",
			},
			expectedGetCalls: []*ExpectedGetCall{
				{
					Key:   `customOperation-{"ID":"test-id","Param":"test-param"}`,
					Value: []byte(`invalid json`),
					Err:   nil,
				},
			},
			expectedResult: false,
			expectedValue:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCacher := NewMockRawBytesDataCacher(t, nil, tc.expectedGetCalls)
			cache := operationResponseCache[
				TestKey,
				TestValue,
			]{cacher: mockCacher, operationID: "customOperation"}

			var actualValue TestValue
			result := cache.Lookup(context.Background(), tc.key, &actualValue)

			if result != tc.expectedResult {
				t.Errorf("Expected result %t, got %t", tc.expectedResult, result)
			}
			if tc.expectedResult && !reflect.DeepEqual(&actualValue, tc.expectedValue) {
				t.Errorf("Expected value %v, got %v", tc.expectedValue, &actualValue)
			}
			mockCacher.AssertExpectations()
		})
	}
}
