package workflow

import (
	"context"
	"encoding/json"
	"net/http"
)

func NewHTTPResultsGetter() *HTTPResultsGetter {
	return &HTTPResultsGetter{
		client: *http.DefaultClient,
	}
}

type HTTPResultsGetter struct {
	client http.Client
}

func (h HTTPResultsGetter) DownloadResults(
	ctx context.Context,
	url string) (ResultsSummaryFile, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// No need to decompress it despite it having the .gz suffix.

	var data ResultsSummaryFile
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}
