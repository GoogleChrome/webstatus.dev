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
	UpsertDailyChromiumHistogramMetric(context.Context, metricdatatypes.HistogramName,
		int64, gcpspanner.DailyChromiumHistogramMetric) error
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
	for id, bucketData := range data {
		rate := new(big.Rat).SetFloat64(bucketData.Rate)
		if rate == nil {
			return ErrInvalidRate
		}
		err := c.client.UpsertDailyChromiumHistogramMetric(ctx, metricdatatypes.WebDXFeatureEnum, id,
			gcpspanner.DailyChromiumHistogramMetric{
				Day:  day,
				Rate: *rate,
			})
		if err != nil {
			return errors.Join(ErrMetricsSaveFailed, err)
		}
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
