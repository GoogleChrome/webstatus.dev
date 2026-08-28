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
	"errors"
	"fmt"
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

type mockTokenCacher struct {
	mu    sync.RWMutex
	store map[string][]byte
}

func newMockTokenCacher() *mockTokenCacher {
	return &mockTokenCacher{
		mu:    sync.RWMutex{},
		store: make(map[string][]byte),
	}
}

func (m *mockTokenCacher) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.store[key]
	if !ok {
		return nil, errors.New("not found")
	}

	return val, nil
}

func (m *mockTokenCacher) Cache(_ context.Context, key string, in []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.store[key] = in

	return nil
}

type testTokenResp struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func TestTokenProvider_GetInstallationToken(t *testing.T) {
	t.Parallel()

	_, privPEM := generateTestRSAPEM(t)
	var requestCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)

		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/app/installations/123/access_tokens" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") == "" {
			t.Errorf("missing Authorization header")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(testTokenResp{
			Token:     mockInstToken1,
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		})
	}))
	defer server.Close()

	cacher := newMockTokenCacher()
	tp, err := NewTokenProvider("app-99", privPEM, cacher)
	if err != nil {
		t.Fatalf("NewTokenProvider failed: %v", err)
	}
	tp.baseURL = server.URL
	tp.httpClient = server.Client()

	// 1. Initial token fetch (cache miss, calls HTTP API)
	token, err := tp.GetInstallationToken(context.Background(), "123")
	if err != nil {
		t.Fatalf("GetInstallationToken failed: %v", err)
	}
	if token != mockInstToken1 {
		t.Errorf("unexpected token: %s", token)
	}

	// 2. Cache hit (no additional network call)
	token2, err := tp.GetInstallationToken(context.Background(), "123")
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

type testNoopTokenCacher struct{}

func (testNoopTokenCacher) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, errors.New("not found")
}

func (testNoopTokenCacher) Cache(_ context.Context, _ string, _ []byte) error {
	return nil
}

func TestTokenProvider_ValidationErrors(t *testing.T) {
	t.Parallel()

	_, privPEM := generateTestRSAPEM(t)
	tp, err := NewTokenProvider("app-99", privPEM, testNoopTokenCacher{})
	if err != nil {
		t.Fatalf("NewTokenProvider failed: %v", err)
	}

	// 1. Empty installation ID
	_, err = tp.GetInstallationToken(context.Background(), "")
	if !errors.Is(err, ErrEmptyInstallationID) {
		t.Errorf("expected ErrEmptyInstallationID, got: %v", err)
	}

	// 2. Invalid non-numeric installation ID
	_, err = tp.GetInstallationToken(context.Background(), "non-numeric-id")
	if err == nil {
		t.Errorf("expected error for non-numeric installation ID, got nil")
	}
}

func TestTokenProviderDetachedContext(t *testing.T) {
	t.Parallel()

	_, privPEM := generateTestRSAPEM(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(30 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(testTokenResp{
			Token:     mockInstToken3,
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		})
	}))
	defer server.Close()

	tp, err := NewTokenProvider("app-99", privPEM, testNoopTokenCacher{})
	if err != nil {
		t.Fatalf("NewTokenProvider failed: %v", err)
	}
	tp.baseURL = server.URL
	tp.httpClient = server.Client()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	// Calling with short timeout context that cancels before server responds
	token, err := tp.GetInstallationToken(ctx, "789")
	if err != nil {
		t.Fatalf("expected token acquisition to succeed despite caller cancel, got: %v", err)
	}
	if token != mockInstToken3 {
		t.Errorf("unexpected token: %s", token)
	}
}

func TestNewTokenProvider_Validation(t *testing.T) {
	t.Parallel()

	_, privPEM := generateTestRSAPEM(t)

	// 1. Empty app ID
	_, err := NewTokenProvider("", privPEM, testNoopTokenCacher{})
	if !errors.Is(err, ErrEmptyAppID) {
		t.Errorf("expected ErrEmptyAppID, got: %v", err)
	}

	// 2. Invalid PEM block
	_, err = NewTokenProvider("app-99", []byte("invalid pem data"), testNoopTokenCacher{})
	if !errors.Is(err, ErrInvalidPrivateKey) {
		t.Errorf("expected ErrInvalidPrivateKey, got: %v", err)
	}
}

func TestTokenProvider_CacheExpiration(t *testing.T) {
	t.Parallel()

	_, privPEM := generateTestRSAPEM(t)
	var requestCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count := atomic.AddInt64(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(testTokenResp{
			Token:     fmt.Sprintf("token-version-%d", count),
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		})
	}))
	defer server.Close()

	cacher := newMockTokenCacher()
	tp, err := NewTokenProvider("app-99", privPEM, cacher)
	if err != nil {
		t.Fatalf("NewTokenProvider failed: %v", err)
	}
	tp.baseURL = server.URL
	tp.httpClient = server.Client()

	// 1. Manually insert expired token into cache
	expiredJSON, _ := json.Marshal(cachedInstallationToken{
		Token:     "expired-token",
		ExpiresAt: time.Now().UTC().Add(-10 * time.Minute), // expired in past
	})
	_ = cacher.Cache(context.Background(), "gh:inst_token:123", expiredJSON)

	// 2. Fetch should ignore expired cache entry and fetch fresh token
	token, err := tp.GetInstallationToken(context.Background(), "123")
	if err != nil {
		t.Fatalf("GetInstallationToken failed: %v", err)
	}
	if token != "token-version-1" {
		t.Errorf("expected fresh token-version-1, got: %s", token)
	}
	if count := atomic.LoadInt64(&requestCount); count != 1 {
		t.Errorf("expected 1 HTTP request, got %d", count)
	}

	// 3. Second call with fresh unexpired cache entry should reuse token without HTTP request
	token2, err := tp.GetInstallationToken(context.Background(), "123")
	if err != nil {
		t.Fatalf("second GetInstallationToken failed: %v", err)
	}
	if token2 != "token-version-1" {
		t.Errorf("expected cached token-version-1, got: %s", token2)
	}
	if count := atomic.LoadInt64(&requestCount); count != 1 {
		t.Errorf("expected still 1 HTTP request, got %d", count)
	}
}

func TestTokenProvider_SingleflightCoalescing(t *testing.T) {
	t.Parallel()

	_, privPEM := generateTestRSAPEM(t)
	var requestCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		time.Sleep(30 * time.Millisecond) // slow server to ensure concurrent in-flight requests
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(testTokenResp{
			Token:     mockInstToken2,
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		})
	}))
	defer server.Close()

	cacher := newMockTokenCacher()
	tp, err := NewTokenProvider("app-99", privPEM, cacher)
	if err != nil {
		t.Fatalf("NewTokenProvider failed: %v", err)
	}
	tp.baseURL = server.URL
	tp.httpClient = server.Client()

	const concurrentCallers = 10
	var wg sync.WaitGroup
	tokens := make([]string, concurrentCallers)
	errorsList := make([]error, concurrentCallers)

	for i := range concurrentCallers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tok, callErr := tp.GetInstallationToken(context.Background(), "999")
			tokens[idx] = tok
			errorsList[idx] = callErr
		}(i)
	}
	wg.Wait()

	for i := range concurrentCallers {
		if errorsList[i] != nil {
			t.Errorf("caller %d failed: %v", i, errorsList[i])
		}
		if tokens[i] != mockInstToken2 {
			t.Errorf("caller %d got %s, want %s", i, tokens[i], mockInstToken2)
		}
	}

	// Concurrent requests should coalesce into exactly 1 HTTP request
	if count := atomic.LoadInt64(&requestCount); count != 1 {
		t.Errorf("expected exactly 1 HTTP request due to concurrent coalescing, got %d", count)
	}
}

type errorTokenCacher struct{}

func (e errorTokenCacher) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, errors.New("cache get failed")
}

func (e errorTokenCacher) Cache(_ context.Context, _ string, _ []byte) error {
	return errors.New("cache write failed")
}

func TestTokenProvider_CacheFailureGracefulDegradation(t *testing.T) {
	t.Parallel()

	_, privPEM := generateTestRSAPEM(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(testTokenResp{
			Token:     mockInstToken4,
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		})
	}))
	defer server.Close()

	tp, err := NewTokenProvider("app-99", privPEM, errorTokenCacher{})
	if err != nil {
		t.Fatalf("NewTokenProvider failed: %v", err)
	}
	tp.baseURL = server.URL
	tp.httpClient = server.Client()

	token, err := tp.GetInstallationToken(context.Background(), "444")
	if err != nil {
		t.Fatalf("expected token fetch to succeed despite cache errors, got: %v", err)
	}
	if token != mockInstToken4 {
		t.Errorf("expected token %s, got %s", mockInstToken4, token)
	}
}

func TestTokenProvider_GetInstallationTokenForRepo(t *testing.T) {
	t.Parallel()

	_, privPEM := generateTestRSAPEM(t)

	var installReqCount int64
	var tokenReqCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/repos/valid-owner/valid-repo/installation":
			atomic.AddInt64(&installReqCount, 1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 456}`))
		case "/repos/unknown-owner/unknown-repo/installation":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "Not Found"}`))
		case "/app/installations/456/access_tokens":
			atomic.AddInt64(&tokenReqCount, 1)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(testTokenResp{
				Token:     mockInstToken2,
				ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
			})
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	cacher := newMockTokenCacher()
	tp, err := NewTokenProvider("app-99", privPEM, cacher)
	if err != nil {
		t.Fatalf("NewTokenProvider failed: %v", err)
	}
	tp.baseURL = server.URL
	tp.httpClient = server.Client()

	// 1. Success on valid repo
	tok, err := tp.GetInstallationTokenForRepo(context.Background(), "valid-owner", "valid-repo")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if tok != mockInstToken2 {
		t.Errorf("expected token %s, got %s", mockInstToken2, tok)
	}

	// 2. Cache hit (neither repo installation nor token should re-query)
	tok2, err := tp.GetInstallationTokenForRepo(context.Background(), "valid-owner", "valid-repo")
	if err != nil {
		t.Fatalf("expected cached success, got %v", err)
	}
	if tok2 != mockInstToken2 {
		t.Errorf("expected cached token %s, got %s", mockInstToken2, tok2)
	}
	if got := atomic.LoadInt64(&installReqCount); got != 1 {
		t.Errorf("expected 1 install request, got %d", got)
	}
	if got := atomic.LoadInt64(&tokenReqCount); got != 1 {
		t.Errorf("expected 1 token request, got %d", got)
	}

	// 3. Unknown repo returns ErrRepositoryInstallationNotFound
	_, err = tp.GetInstallationTokenForRepo(context.Background(), "unknown-owner", "unknown-repo")
	if !errors.Is(err, ErrRepositoryInstallationNotFound) {
		t.Errorf("expected ErrRepositoryInstallationNotFound, got %v", err)
	}

	// 4. Empty arguments
	if _, err := tp.GetInstallationTokenForRepo(context.Background(), "", "valid-repo"); err == nil {
		t.Error("expected error for empty owner")
	}
	if _, err := tp.GetInstallationTokenForRepo(context.Background(), "valid-owner", ""); err == nil {
		t.Error("expected error for empty repo")
	}
}
