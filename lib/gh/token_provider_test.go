// Copyright 2026 Google LLC
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

package gh

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

//nolint:gosec // Mock test tokens for unit testing
const (
	mockInstToken1 = "mock_val_sample_123"
	mockInstToken2 = "mock_val_coalesce_456"
	mockInstToken3 = "mock_val_detached_789"
	mockInstToken4 = "mock_val_recovered_000"
)

func TestTokenProvider_GetInstallationToken(t *testing.T) {
	t.Parallel()

	_, privPEM := generateTestRSAPEM(t)
	var requestCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)

		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/app/installations/inst-123/access_tokens" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") == "" {
			t.Errorf("missing Authorization header")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(installationTokenResponse{
			Token:     mockInstToken1,
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		})
	}))
	defer server.Close()

	tp := NewTokenProvider("app-99", privPEM,
		WithTokenProviderBaseURL(server.URL),
		WithTokenProviderHTTPClient(server.Client()))

	// 1. Initial token fetch
	token, err := tp.GetInstallationToken(context.Background(), "inst-123")
	if err != nil {
		t.Fatalf("GetInstallationToken failed: %v", err)
	}
	if token != mockInstToken1 {
		t.Errorf("unexpected token: %s", token)
	}

	// 2. Cache hit (no additional network call)
	token2, err := tp.GetInstallationToken(context.Background(), "inst-123")
	if err != nil {
		t.Fatalf("second GetInstallationToken failed: %v", err)
	}
	if token2 != token {
		t.Errorf("cached token mismatch: %s != %s", token2, token)
	}

	if count := atomic.LoadInt64(&requestCount); count != 1 {
		t.Errorf("expected 1 HTTP request, got %d", count)
	}
}

func TestTokenProviderSingleflight(t *testing.T) {
	t.Parallel()

	_, privPEM := generateTestRSAPEM(t)
	var requestCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		time.Sleep(20 * time.Millisecond) // Simulate network delay

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(installationTokenResponse{
			Token:     mockInstToken2,
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		})
	}))
	defer server.Close()

	tp := NewTokenProvider("app-99", privPEM,
		WithTokenProviderBaseURL(server.URL),
		WithTokenProviderHTTPClient(server.Client()))

	const numGoroutines = 20
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			token, err := tp.GetInstallationToken(context.Background(), "inst-concurrent")
			if err != nil {
				t.Errorf("concurrent GetInstallationToken failed: %v", err)
			}
			if token != mockInstToken2 {
				t.Errorf("unexpected token: %s", token)
			}
		}()
	}

	wg.Wait()

	if count := atomic.LoadInt64(&requestCount); count != 1 {
		t.Errorf("singleflight failed: expected 1 request, got %d", count)
	}
}

func TestTokenProviderDetachedContext(t *testing.T) {
	t.Parallel()

	_, privPEM := generateTestRSAPEM(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(30 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(installationTokenResponse{
			Token:     mockInstToken3,
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		})
	}))
	defer server.Close()

	tp := NewTokenProvider("app-99", privPEM,
		WithTokenProviderBaseURL(server.URL),
		WithTokenProviderHTTPClient(server.Client()))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	// Calling with short timeout context that cancels before server responds
	token, err := tp.GetInstallationToken(ctx, "inst-detached")
	if err != nil {
		t.Fatalf("expected token acquisition to succeed despite caller cancel, got: %v", err)
	}
	if token != mockInstToken3 {
		t.Errorf("unexpected token: %s", token)
	}
}

func TestTokenProviderErrorRecovery(t *testing.T) {
	t.Parallel()

	_, privPEM := generateTestRSAPEM(t)
	var callCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count := atomic.AddInt64(&callCount, 1)
		if count == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal github error"))

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(installationTokenResponse{
			Token:     mockInstToken4,
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		})
	}))
	defer server.Close()

	tp := NewTokenProvider("app-99", privPEM,
		WithTokenProviderBaseURL(server.URL),
		WithTokenProviderHTTPClient(server.Client()))

	// First call fails
	_, err := tp.GetInstallationToken(context.Background(), "inst-retry")
	if err == nil {
		t.Fatalf("expected error on first call, got nil")
	}

	// Second call should NOT be blocked by cached failure in singleflight
	token, err := tp.GetInstallationToken(context.Background(), "inst-retry")
	if err != nil {
		t.Fatalf("expected second call to succeed, got: %v", err)
	}
	if token != mockInstToken4 {
		t.Errorf("unexpected token: %s", token)
	}
}
