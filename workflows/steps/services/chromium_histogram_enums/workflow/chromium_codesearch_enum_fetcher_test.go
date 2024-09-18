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

package workflow

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"testing"
)

// This is more of an integration test to ensure the data is actually base64 encoded.
func TestChromiumCodesearchEnumFetcher_Fetch_Base64Encoded(t *testing.T) {
	ctx := context.Background()
	httpClient := http.DefaultClient
	fetcher, err := NewChromiumCodesearchEnumFetcher(httpClient)
	if err != nil {
		t.Fatalf("Failed to create fetcher: %v", err)
	}

	reader, err := fetcher.Fetch(ctx)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	defer reader.Close()

	// Read the fetched data into a buffer
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, reader)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	fetchedData := buf.String()

	// Attempt to decode the fetched data as base64
	b, err := base64.StdEncoding.DecodeString(fetchedData)
	if err != nil {
		t.Errorf("Fetched data is not valid base64: %v\nData:\n%s", err, string(b))
	}
}
