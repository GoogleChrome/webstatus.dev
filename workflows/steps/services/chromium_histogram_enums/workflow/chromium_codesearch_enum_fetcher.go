package workflow

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

func NewChromiumCodesearchEnumFetcher(httpClient *http.Client) (*ChromiumCodesearchEnumFetcher, error) {
	u, err := url.Parse(enumURL)
	if err != nil {
		return nil, err
	}

	return &ChromiumCodesearchEnumFetcher{
		httpClient: httpClient,
		enumURL:    u,
	}, nil
}

// ChromiumCodesearchEnumFetcher fetches the enums from Chromium code search.
// The returned data will be base64 encoded and it is up the consumer to decode
// before reading.
type ChromiumCodesearchEnumFetcher struct {
	httpClient *http.Client
	enumURL    *url.URL
}

const enumURL = "https://chromium.googlesource.com/chromium/src/+/main/tools/metrics/histograms/enums.xml?format=TEXT"

func (f ChromiumCodesearchEnumFetcher) Fetch(ctx context.Context) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.enumURL.String(), nil)
	if err != nil {
		slog.ErrorContext(ctx, "unable to create request", "error", err)

		return nil, err
	}
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		// Clean up by closing since we will not be returning the body
		resp.Body.Close()

		return nil, err
	}

	return resp.Body, nil
}
