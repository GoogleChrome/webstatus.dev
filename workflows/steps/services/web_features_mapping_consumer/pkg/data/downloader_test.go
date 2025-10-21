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

package data

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPDownloader_Download(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Errorf("Expected to request '/', got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test data"))
		if err != nil {
			t.Fatalf("could not write response: %v", err)
		}
	}))
	defer server.Close()

	downloader := NewHTTPDownloader(server.Client())
	reader, err := downloader.Download(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("could not read data: %v", err)
	}

	if string(data) != "test data" {
		t.Errorf("expected 'test data', got '%s'", string(data))
	}
}
