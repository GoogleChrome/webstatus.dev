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
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-github/v79/github"
)

const testArchiveToken = "SECRET-ARCHIVE-TOKEN"

// TestArchiveDownloadTransportUsesFreshHTTP1Connections verifies that archive downloads use
// HTTP/1.1 even against an HTTP/2-capable server, and that no connection is reused across
// downloads, so one slow connection can never be handed to the next download.
func TestArchiveDownloadTransportUsesFreshHTTP1Connections(t *testing.T) {
	t.Parallel()

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("archive-bytes"))
	}))
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()

	serverTransport, ok := srv.Client().Transport.(*http.Transport)
	if !ok {
		t.Fatal("unexpected test server transport type")
	}
	// Only trust the test certificate; keep ALPN under the archive transport's control.
	//nolint:exhaustruct // Test-only TLS config.
	tlsConfig := &tls.Config{
		RootCAs:    serverTransport.TLSClientConfig.RootCAs,
		MinVersion: tls.VersionTLS12,
	}
	client := &http.Client{
		Transport:     newArchiveDownloadTransport(tlsConfig),
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       0,
	}

	for i := range 2 {
		var reused bool
		//nolint:exhaustruct // Only the GotConn hook is needed.
		trace := &httptrace.ClientTrace{
			GotConn: func(info httptrace.GotConnInfo) { reused = info.Reused },
		}
		req, err := http.NewRequestWithContext(
			httptrace.WithClientTrace(context.Background(), trace), http.MethodGet, srv.URL, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("download %d failed: %v", i, err)
		}
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.Proto != "HTTP/1.1" {
			t.Errorf("download %d used %s, want HTTP/1.1", i, resp.Proto)
		}
		if reused {
			t.Errorf("download %d reused a connection, want a fresh connection per download", i)
		}
	}
}

func newArchiveTokenTestClient(t *testing.T, archiveURL string, httpClient *http.Client) *Client {
	t.Helper()

	parsed, err := url.Parse(archiveURL)
	if err != nil {
		t.Fatalf("failed to parse archive url: %v", err)
	}
	repoMock := &mockTarballRepoClient{
		getArchiveLinkFunc: func(
			_ context.Context, _, _ string, _ github.ArchiveFormat, _ *github.RepositoryContentGetOptions, _ int,
		) (*url.URL, *github.Response, error) {
			return parsed, nil, nil
		},
	}

	return &Client{
		repoClient:   repoMock,
		gitClient:    nil,
		issuesClient: nil,
		appsClient:   nil,
		httpClient:   httpClient,
	}
}

// TestDownloadTarballRedactsArchiveTokenFromErrors verifies that the short-lived token in private
// repository archive URLs never appears in returned errors (which are logged and persisted).
func TestDownloadTarballRedactsArchiveTokenFromErrors(t *testing.T) {
	t.Parallel()

	t.Run("non-200 response", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusGatewayTimeout)
		}))
		defer srv.Close()

		c := newArchiveTokenTestClient(t, srv.URL+"/owner/repo/legacy.tar.gz/main?token="+testArchiveToken, srv.Client())
		_, err := c.DownloadTarball(context.Background(), "owner", "repo", "main")
		if err == nil {
			t.Fatal("expected an error for a 504 response")
		}
		if strings.Contains(err.Error(), testArchiveToken) || strings.Contains(err.Error(), "token=") {
			t.Fatalf("error leaks the archive token: %v", err)
		}
		if !IsServerError(err) {
			t.Fatalf("expected the 504 to still classify as a server error, got %v", err)
		}
	})

	t.Run("transport error", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.NotFoundHandler())
		closedURL := srv.URL
		srv.Close()

		// Use our own transport: other tests in this package replace http.DefaultClient's transport.
		c := newArchiveTokenTestClient(t, closedURL+"/owner/repo/legacy.tar.gz/main?token="+testArchiveToken,
			NewArchiveDownloadHTTPClient())
		_, err := c.DownloadTarball(context.Background(), "owner", "repo", "main")
		if err == nil {
			t.Fatal("expected an error when the archive host is unreachable")
		}
		if strings.Contains(err.Error(), testArchiveToken) || strings.Contains(err.Error(), "token=") {
			t.Fatalf("error leaks the archive token: %v", err)
		}
	})
}

func TestWithArchiveHTTPClient(t *testing.T) {
	t.Parallel()

	original := NewClient("token")
	archiveClient := NewArchiveDownloadHTTPClient()
	withArchive := original.WithArchiveHTTPClient(archiveClient)

	if withArchive.httpClient != archiveClient {
		t.Fatal("WithArchiveHTTPClient did not set the archive download client")
	}
	if original.httpClient != nil {
		t.Fatal("WithArchiveHTTPClient must not modify the original client")
	}
}
