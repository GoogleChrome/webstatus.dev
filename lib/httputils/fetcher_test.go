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

package httputils

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/fetchtypes"
)

// mockRoundTripper is a mock http.RoundTripper for testing purposes.
type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestNewHTTPFetcher(t *testing.T) {
	// nolint:exhaustruct // WONTFIX - external struct.
	client := &http.Client{}

	t.Run("valid URL", func(t *testing.T) {
		fetcher, err := NewHTTPFetcher("http://example.com", client)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fetcher == nil {
			t.Fatal("fetcher should not be nil")
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		_, err := NewHTTPFetcher(":", client)
		if err == nil {
			t.Fatal("expected an error for invalid URL, but got nil")
		}
	})
}

func TestHTTPFetcher_Fetch(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("test data"))
		}))
		defer server.Close()

		fetcher, err := NewHTTPFetcher(server.URL, server.Client())
		if err != nil {
			t.Fatalf("unexpected error creating fetcher: %v", err)
		}

		body, err := fetcher.Fetch(context.Background())
		if err != nil {
			t.Fatalf("unexpected error on fetch: %v", err)
		}
		defer body.Close()

		data, err := io.ReadAll(body)
		if err != nil {
			t.Fatalf("unexpected error reading body: %v", err)
		}

		if string(data) != "test data" {
			t.Errorf("expected 'test data', got '%s'", string(data))
		}
	})

	t.Run("unexpected status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		fetcher, err := NewHTTPFetcher(server.URL, server.Client())
		if err != nil {
			t.Fatalf("unexpected error creating fetcher: %v", err)
		}

		_, err = fetcher.Fetch(context.Background())
		if !errors.Is(err, fetchtypes.ErrUnexpectedResult) {
			t.Errorf("expected ErrUnexpectedResult, got %v", err)
		}
	})

	t.Run("fetch error", func(t *testing.T) {
		// nolint:exhaustruct // WONTFIX - external struct.
		client := &http.Client{
			Transport: &mockRoundTripper{
				roundTripFunc: func(_ *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				},
			},
		}

		fetcher, err := NewHTTPFetcher("http://example.com", client)
		if err != nil {
			t.Fatalf("unexpected error creating fetcher: %v", err)
		}

		_, err = fetcher.Fetch(context.Background())
		if !errors.Is(err, fetchtypes.ErrFailedToFetch) {
			t.Errorf("expected ErrFailedToFetch, got %v", err)
		}
	})

	t.Run("with headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer token" {
				t.Errorf("expected Authorization header, but it was not set")
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		fetcher, err := NewHTTPFetcher(server.URL, server.Client())
		if err != nil {
			t.Fatalf("unexpected error creating fetcher: %v", err)
		}

		_, err = fetcher.Fetch(context.Background(), WithHeaders(map[string]string{
			"Authorization": "Bearer token",
		}))
		if err != nil {
			t.Fatalf("unexpected error on fetch: %v", err)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer server.Close()

		fetcher, err := NewHTTPFetcher(server.URL, server.Client())
		if err != nil {
			t.Fatalf("unexpected error creating fetcher: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = fetcher.Fetch(ctx)
		if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "context canceled") {
			t.Errorf("expected context canceled error, got %v", err)
		}
	})
}
