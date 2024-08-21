package spanneradapters

import (
	"context"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
)

// ChromiumHistogramEnumConsumer handles the conversion of histogram between the workflow/API input
// format and the format used by the GCP Spanner client.
type ChromiumHistogramEnumConsumer struct {
	client WebFeatureSpannerClient
}

// NewChromiumHistogramEnumConsumer constructs an adapter for the chromium histogram enum consumer service.
func NewChromiumHistogramEnumConsumer(client WebFeatureSpannerClient) *ChromiumHistogramEnumConsumer {
	return &ChromiumHistogramEnumConsumer{client: client}
}

// ChromiumHistogramEnumsClient expects a subset of the functionality from lib/gcpspanner that only apply to
// Chromium Histograms.
type ChromiumHistogramEnumsClient interface {
	UpsertChromiumHistogramEnum(context.Context, gcpspanner.ChromiumHistogramEnum) error
}
