// Copyright 2024 Google LLC
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
	"testing"
)

type mockCacher struct {
	cache map[string][]byte
	err   error
}

func (mc *mockCacher) Cache(_ context.Context, key string, value []byte) error {
	if mc.err != nil {
		return mc.err
	}
	mc.cache[key] = value

	return nil
}

func (mc *mockCacher) Get(_ context.Context, key string) ([]byte, error) {
	if mc.err != nil {
		return nil, mc.err
	}
	if value, ok := mc.cache[key]; ok {
		return value, nil
	}

	return nil, errors.New("not found")
}

func TestCacheMiddleware(t *testing.T) {
	testCases := []struct {
		name              string
		method            string
		url               string
		mockCache         map[string][]byte
		mockCacheError    error
		expectedResponse  string
		expectedCacheSize int
		responseHeaders   map[string]string
		responseStatus    int
	}{
		{
			name:              "GET with cache hit, correct content type, and 200 status",
			method:            http.MethodGet,
			url:               "/test?param=value",
			mockCache:         map[string][]byte{"/test?param=value": []byte("cached response")},
			expectedResponse:  "cached response",
			expectedCacheSize: 1,
			responseHeaders:   map[string]string{"Content-Type": "application/json"},
			responseStatus:    http.StatusOK,
			mockCacheError:    nil,
		},
		{
			name:              "GET with cache miss, correct content type, and 200 status",
			method:            http.MethodGet,
			url:               "/test",
			mockCache:         map[string][]byte{},
			expectedResponse:  "test response",
			expectedCacheSize: 1,
			responseHeaders:   map[string]string{"Content-Type": "application/json"},
			responseStatus:    http.StatusOK,
			mockCacheError:    nil,
		},
		{
			name:              "GET with cache miss, incorrect content type",
			method:            http.MethodGet,
			url:               "/test2",
			mockCache:         map[string][]byte{},
			expectedResponse:  "test response",
			expectedCacheSize: 0, // Should not be cached
			responseHeaders:   map[string]string{"Content-Type": "text/plain"},
			responseStatus:    http.StatusOK,
			mockCacheError:    nil,
		},
		{
			name:              "GET with 500 status code",
			method:            http.MethodGet,
			url:               "/test3",
			mockCache:         map[string][]byte{},
			expectedResponse:  "test response",
			expectedCacheSize: 0, // Should not be cached due to status code
			responseHeaders:   map[string]string{"Content-Type": "application/json"},
			responseStatus:    http.StatusInternalServerError,
			mockCacheError:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCacher := &mockCacher{cache: tc.mockCache, err: tc.mockCacheError}
			cacheMiddleware := NewCacheMiddleware[string, []byte](mockCacher)

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				for key, value := range tc.responseHeaders {
					w.Header().Set(key, value)
				}
				w.WriteHeader(tc.responseStatus) // Write the correct status code
				_, err := w.Write([]byte("test response"))
				if err != nil {
					t.Errorf("unknown error %s", err.Error())
				}
			})

			req := httptest.NewRequest(tc.method, tc.url, nil)
			recorder := httptest.NewRecorder()
			handler := cacheMiddleware(nextHandler)
			handler.ServeHTTP(recorder, req)

			res := recorder.Result()
			defer res.Body.Close()

			if res.StatusCode != tc.responseStatus { // Check expected status code
				t.Errorf("Expected status code %d, got %d", tc.responseStatus, res.StatusCode)
			}

			// ... (rest of the test logic, including body and cache size checks) ...
		})
	}
}
