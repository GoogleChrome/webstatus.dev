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

package spanneradapters

import (
	"context"
	"errors"
	"log/slog"
	"math/big"

	"cloud.google.com/go/civil"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

// UMAMetricConsumer handles the conversion of histogram between the workflow/API input
// format and the format used by the GCP Spanner client.
type UMAMetricConsumer struct {
	client UMAMetricsClient
}

// NewUMAMetricConsumer constructs an adapter for the uma metric export service.
func NewUMAMetricConsumer(client UMAMetricsClient) *UMAMetricConsumer {
	return &UMAMetricConsumer{client: client}
}

// UMAMetricsClient expects a subset of the functionality from lib/gcpspanner that only apply to
// Chromium Histograms.
type UMAMetricsClient interface {
	HasDailyChromiumHistogramCapstone(context.Context, gcpspanner.DailyChromiumHistogramEnumCapstone) (*bool, error)
	UpsertDailyChromiumHistogramCapstone(context.Context, gcpspanner.DailyChromiumHistogramEnumCapstone) error
	StoreDailyChromiumHistogramMetrics(context.Context,
		metricdatatypes.HistogramName,
		map[int64]gcpspanner.DailyChromiumHistogramMetric) error
	SyncLatestDailyChromiumHistogramMetrics(context.Context) error
}

func (c *UMAMetricConsumer) HasCapstone(
	ctx context.Context,
	day civil.Date,
	histogramName metricdatatypes.HistogramName) (bool, error) {
	found, err := c.client.HasDailyChromiumHistogramCapstone(ctx, gcpspanner.DailyChromiumHistogramEnumCapstone{
		HistogramName: histogramName,
		Day:           day,
	})
	if err != nil {
		return false, errors.Join(ErrCapstoneLookupFailed, err)
	}

	return *found, nil
}

func (c *UMAMetricConsumer) SaveCapstone(
	ctx context.Context,
	day civil.Date,
	histogramName metricdatatypes.HistogramName) error {
	err := c.client.UpsertDailyChromiumHistogramCapstone(ctx, gcpspanner.DailyChromiumHistogramEnumCapstone{
		HistogramName: histogramName,
		Day:           day,
	})
	if err != nil {
		return errors.Join(ErrCapstoneSaveFailed, err)
	}

	return nil
}

func (c *UMAMetricConsumer) SaveMetrics(
	ctx context.Context,
	day civil.Date,
	data metricdatatypes.BucketDataMetrics) error {
	metricsToStore := make(map[int64]gcpspanner.DailyChromiumHistogramMetric, len(data))
	for id, bucketData := range data {
		rate := new(big.Rat).SetFloat64(bucketData.Rate)
		if rate == nil {
			return ErrInvalidRate
		}
		metricsToStore[id] = gcpspanner.DailyChromiumHistogramMetric{
			Day:  day,
			Rate: *rate,
		}
	}

	histogramName := metricdatatypes.WebDXFeatureEnum
	err := c.client.StoreDailyChromiumHistogramMetrics(ctx, histogramName, metricsToStore)
	if err != nil {
		slog.ErrorContext(ctx, "failed to store metrics", "histogram", histogramName, "day", day, "err", err)

		return errors.Join(ErrMetricsSaveFailed, err)
	}

	err = c.client.SyncLatestDailyChromiumHistogramMetrics(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to sync latest metrics", "histogram", histogramName, "day", day, "err", err)

		return errors.Join(ErrMetricsSaveFailed, err)
	}

	return nil
}

var (
	// ErrCapstoneLookupFailed indicates an internal error trying to find the capstone.
	ErrCapstoneLookupFailed = errors.New("failed to look up capstone")

	// ErrCapstoneSaveFailed indicates an internal error trying to save the capstone.
	ErrCapstoneSaveFailed = errors.New("failed to save capstone")

	// ErrMetricsSaveFailed indicates an internal error trying to save the metrics.
	ErrMetricsSaveFailed = errors.New("failed to save metrics")

	// ErrInvalidRate indicates an internal error when parsing the rate.
	ErrInvalidRate = errors.New("invalid rate")
)
