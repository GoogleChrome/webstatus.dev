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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
)

// Custom ResponseWriter wrapper.
type responseRecorder struct {
	http.ResponseWriter
	buffer     *bytes.Buffer
	statusCode int
}

func (rw *responseRecorder) Header() http.Header {
	return rw.ResponseWriter.Header()
}

func (rw *responseRecorder) Write(b []byte) (int, error) {
	rw.buffer.Write(b)

	return rw.ResponseWriter.Write(b)
}

func (rw *responseRecorder) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

type DataCacher[K string, V []byte] interface {
	// Cache stores a value associated with a key in the cache.
	Cache(context.Context, K, V) error
	// Get retrieves a value from the cache by its key.
	Get(context.Context, K) (V, error)
}

type OperationIDDataCacher[K any, V any] struct {
	// Currently, all implementations of cacher use a string key and []bytes value
	cacher      DataCacher[string, []byte]
	operationID string
}

func (c OperationIDDataCacher[K, V]) Cache(ctx context.Context, key K, value V) error {
	jsonBytesKey, err := json.Marshal(key)
	if err != nil {
		return err
	}
	jsonBytesValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.cacher.Cache(ctx, c.key(jsonBytesKey), jsonBytesValue)
}

func (c OperationIDDataCacher[K, V]) key(jsonBytesKey []byte) string {
	return c.operationID + "-" + string(jsonBytesKey)
}

func (c OperationIDDataCacher[K, V]) Lookup(ctx context.Context, key K, value *V) error {
	jsonBytesKey, err := json.Marshal(key)
	if err != nil {
		return err
	}

	valueBytes, err := c.cacher.Get(ctx, c.key(jsonBytesKey))
	if err != nil {
		return err
	}

	err = json.Unmarshal(valueBytes, value)

	return err
}

func NewOperationCacheMiddleware[K any, V any](cacher DataCacher[string, []byte], operationID string) *OperationIDDataCacher[K, V] {
	return &OperationIDDataCacher[K, V]{
		cacher:      cacher,
		operationID: operationID,
	}

}

// TODO: Pass in context to be used by slog.ErrorContext.
func NewCacheMiddleware[K string, V []byte](cacher DataCacher[string, []byte]) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)

				return
			}

			cacheKey := r.URL.Path
			if r.URL.RawQuery != "" { // Check if there are query parameters
				cacheKey += "?" + r.URL.Query().Encode()
			}

			// Attempt to get the response from cache
			cachedResponse, err := cacher.Get(r.Context(), cacheKey)
			if err == nil { // Cache hit
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write(cachedResponse)
				if err != nil {
					slog.Error("unable to write cached response", "cacheKey", cacheKey, "error", err)
				}

				return
			} else if !errors.Is(err, cachetypes.ErrCachedDataNotFound) {
				// Unknown internal error. For now log it.
				slog.Error("cache fetched failed for unknown reasons", "error", err)
			}

			recorder := &responseRecorder{
				ResponseWriter: w,
				buffer:         bytes.NewBuffer(nil),
				// Will be changed by the actual server.
				statusCode: 0,
			}

			next.ServeHTTP(recorder, r)
			slog.Info("status", "code", recorder.statusCode, "Headers", recorder.ResponseWriter.Header())

			if recorder.statusCode == http.StatusOK {
				err = cacher.Cache(r.Context(), cacheKey, V(recorder.buffer.Bytes()))
				if err != nil {
					slog.Warn("unable to cache value", "cacheKey", cacheKey, "error", err)
				}
			}
		})
	}
}
