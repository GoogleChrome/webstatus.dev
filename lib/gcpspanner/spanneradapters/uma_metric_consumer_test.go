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
	"testing"

	"cloud.google.com/go/civil"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

type mockUMAMetricsClient struct {
	hasDailyChromiumHistogramCapstone func(context.Context,
		gcpspanner.DailyChromiumHistogramEnumCapstone) (*bool, error)
	upsertDailyChromiumHistogramCapstone func(context.Context,
		gcpspanner.DailyChromiumHistogramEnumCapstone) error
	upsertDailyChromiumHistogramMetric func(context.Context,
		metricdatatypes.HistogramName, int64, gcpspanner.DailyChromiumHistogramMetric) error
}

func (m *mockUMAMetricsClient) HasDailyChromiumHistogramCapstone(ctx context.Context,
	in gcpspanner.DailyChromiumHistogramEnumCapstone) (*bool, error) {
	return m.hasDailyChromiumHistogramCapstone(ctx, in)
}

func (m *mockUMAMetricsClient) UpsertDailyChromiumHistogramCapstone(ctx context.Context,
	in gcpspanner.DailyChromiumHistogramEnumCapstone) error {
	return m.upsertDailyChromiumHistogramCapstone(ctx, in)
}

func (m *mockUMAMetricsClient) UpsertDailyChromiumHistogramMetric(ctx context.Context,
	histogramName metricdatatypes.HistogramName, id int64, in gcpspanner.DailyChromiumHistogramMetric) error {
	return m.upsertDailyChromiumHistogramMetric(ctx, histogramName, id, in)
}

func TestUMAMetricConsumer_HasCapstone(t *testing.T) {
	tests := []struct {
		name        string
		client      *mockUMAMetricsClient
		day         civil.Date
		histogram   metricdatatypes.HistogramName
		want        bool
		expectedErr error
	}{
		{
			name: "HasDailyChromiumHistogramCapstone returns true",
			client: &mockUMAMetricsClient{
				hasDailyChromiumHistogramCapstone: func(_ context.Context,
					_ gcpspanner.DailyChromiumHistogramEnumCapstone) (*bool, error) {
					result := true

					return &result, nil
				},
				upsertDailyChromiumHistogramCapstone: nil,
				upsertDailyChromiumHistogramMetric:   nil,
			},
			day:         civil.Date{Year: 2024, Month: 1, Day: 1},
			histogram:   metricdatatypes.HistogramName("test"),
			want:        true,
			expectedErr: nil,
		},
		{
			name: "HasDailyChromiumHistogramCapstone returns false",
			client: &mockUMAMetricsClient{
				hasDailyChromiumHistogramCapstone: func(_ context.Context,
					_ gcpspanner.DailyChromiumHistogramEnumCapstone) (*bool, error) {
					result := false

					return &result, nil
				},
				upsertDailyChromiumHistogramCapstone: nil,
				upsertDailyChromiumHistogramMetric:   nil,
			},
			day:         civil.Date{Year: 2024, Month: 1, Day: 1},
			histogram:   metricdatatypes.HistogramName("test"),
			want:        false,
			expectedErr: nil,
		},
		{
			name: "HasDailyChromiumHistogramCapstone returns error",
			client: &mockUMAMetricsClient{
				upsertDailyChromiumHistogramCapstone: nil,
				upsertDailyChromiumHistogramMetric:   nil,
				hasDailyChromiumHistogramCapstone: func(_ context.Context,
					_ gcpspanner.DailyChromiumHistogramEnumCapstone) (*bool, error) {
					return nil, errors.New("test error")
				},
			},
			day:         civil.Date{Year: 2024, Month: 1, Day: 1},
			histogram:   metricdatatypes.HistogramName("test"),
			want:        false,
			expectedErr: ErrCapstoneLookupFailed,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &UMAMetricConsumer{
				client: tc.client,
			}
			got, err := c.HasCapstone(context.Background(), tc.day, tc.histogram)
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("UMAMetricConsumer.HasCapstone() error = %v, expectedErr %v", err, tc.expectedErr)
			}
			if got != tc.want {
				t.Errorf("UMAMetricConsumer.HasCapstone() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUMAMetricConsumer_SaveCapstone(t *testing.T) {
	tests := []struct {
		name        string
		client      *mockUMAMetricsClient
		day         civil.Date
		histogram   metricdatatypes.HistogramName
		expectedErr error
	}{
		{
			name: "UpsertDailyChromiumHistogramCapstone returns nil",
			client: &mockUMAMetricsClient{
				upsertDailyChromiumHistogramCapstone: func(_ context.Context,
					_ gcpspanner.DailyChromiumHistogramEnumCapstone) error {
					return nil
				},
				upsertDailyChromiumHistogramMetric: nil,
				hasDailyChromiumHistogramCapstone:  nil,
			},
			day:         civil.Date{Year: 2024, Month: 1, Day: 1},
			histogram:   metricdatatypes.HistogramName("test"),
			expectedErr: nil,
		},
		{
			name: "UpsertDailyChromiumHistogramCapstone returns error",
			client: &mockUMAMetricsClient{
				upsertDailyChromiumHistogramMetric: nil,
				hasDailyChromiumHistogramCapstone:  nil,
				upsertDailyChromiumHistogramCapstone: func(_ context.Context,
					_ gcpspanner.DailyChromiumHistogramEnumCapstone) error {
					return errors.New("test error")
				},
			},
			day:         civil.Date{Year: 2024, Month: 1, Day: 1},
			histogram:   metricdatatypes.HistogramName("test"),
			expectedErr: ErrCapstoneSaveFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &UMAMetricConsumer{
				client: tt.client,
			}
			err := c.SaveCapstone(context.Background(), tt.day, tt.histogram)
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("UMAMetricConsumer.SaveCapstone() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestUMAMetricConsumer_SaveMetrics(t *testing.T) {
	tests := []struct {
		name        string
		client      *mockUMAMetricsClient
		day         civil.Date
		data        metricdatatypes.BucketDataMetrics
		expectedErr error
	}{
		{
			name: "UpsertDailyChromiumHistogramMetric returns nil",
			client: &mockUMAMetricsClient{
				upsertDailyChromiumHistogramMetric: func(_ context.Context, _ metricdatatypes.HistogramName,
					_ int64, _ gcpspanner.DailyChromiumHistogramMetric) error {
					return nil
				},
				hasDailyChromiumHistogramCapstone:    nil,
				upsertDailyChromiumHistogramCapstone: nil,
			},
			day: civil.Date{Year: 2024, Month: 1, Day: 1},
			data: metricdatatypes.BucketDataMetrics{
				1: {Rate: 0.5, LowVolume: false, Milestone: ""},
				2: {Rate: 0.75, LowVolume: false, Milestone: ""},
			},
			expectedErr: nil,
		},
		{
			name: "UpsertDailyChromiumHistogramMetric returns error",
			client: &mockUMAMetricsClient{
				hasDailyChromiumHistogramCapstone:    nil,
				upsertDailyChromiumHistogramCapstone: nil,
				upsertDailyChromiumHistogramMetric: func(_ context.Context, _ metricdatatypes.HistogramName,
					_ int64, _ gcpspanner.DailyChromiumHistogramMetric) error {
					return errors.New("test error")
				},
			},
			day: civil.Date{Year: 2024, Month: 1, Day: 1},
			data: metricdatatypes.BucketDataMetrics{
				1: {Rate: 0.5, LowVolume: false, Milestone: ""},
			},
			expectedErr: ErrMetricsSaveFailed,
		},
		{
			name: "UpsertDailyChromiumHistogramMetric skips on ErrUsageMetricUpsertNoHistogramEnumFound",
			client: &mockUMAMetricsClient{
				hasDailyChromiumHistogramCapstone:    nil,
				upsertDailyChromiumHistogramCapstone: nil,
				upsertDailyChromiumHistogramMetric: func(_ context.Context, _ metricdatatypes.HistogramName,
					_ int64, _ gcpspanner.DailyChromiumHistogramMetric) error {
					return gcpspanner.ErrUsageMetricUpsertNoHistogramEnumFound
				},
			},
			day: civil.Date{Year: 2024, Month: 1, Day: 1},
			data: metricdatatypes.BucketDataMetrics{
				1: {Rate: 0.5, LowVolume: false, Milestone: ""},
			},
			expectedErr: nil,
		},
		{
			name: "UpsertDailyChromiumHistogramMetric skips on ErrUsageMetricUpsertNoFeatureIDFound",
			client: &mockUMAMetricsClient{
				hasDailyChromiumHistogramCapstone:    nil,
				upsertDailyChromiumHistogramCapstone: nil,
				upsertDailyChromiumHistogramMetric: func(_ context.Context, _ metricdatatypes.HistogramName,
					_ int64, _ gcpspanner.DailyChromiumHistogramMetric) error {
					return gcpspanner.ErrUsageMetricUpsertNoFeatureIDFound
				},
			},
			day: civil.Date{Year: 2024, Month: 1, Day: 1},
			data: metricdatatypes.BucketDataMetrics{
				1: {Rate: 0.5, LowVolume: false, Milestone: ""},
			},
			expectedErr: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &UMAMetricConsumer{
				client: tc.client,
			}
			err := c.SaveMetrics(context.Background(), tc.day, tc.data)
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("UMAMetricConsumer.SaveMetrics() error = %v, expectedErr %v", err, tc.expectedErr)
			}
		})
	}
}
