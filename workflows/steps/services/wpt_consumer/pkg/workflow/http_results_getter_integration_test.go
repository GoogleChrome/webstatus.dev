package workflow

import (
	"context"
	"testing"
)

func TestHTTPResultsGetter(t *testing.T) {
	g := NewHTTPResultsGetter()
	sampleURL := "https://storage.googleapis.com/wptd/9593290ad1bd621f74c697c7cc347348af2de32a/chrome-117.0.5938.62-linux-20.04-ddee0c57b6-summary_v2.json.gz"
	data, err := g.DownloadResults(context.Background(), sampleURL)
	if err != nil {
		t.Errorf("unexpected error during download. %s", err.Error())
	}
	if len(data) == 0 {
		t.Error("expected there to be data")
	}
}
