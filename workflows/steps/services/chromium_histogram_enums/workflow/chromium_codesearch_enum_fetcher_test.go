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
	httpClient := &http.Client{}
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
