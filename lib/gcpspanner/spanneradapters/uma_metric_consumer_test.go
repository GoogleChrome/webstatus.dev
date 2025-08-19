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
	storeDailyChromiumHistogramMetrics func(context.Context,
		metricdatatypes.HistogramName, map[int64]gcpspanner.DailyChromiumHistogramMetric) error
	syncLatestDailyChromiumHistogramMetrics func(context.Context) error
}

func (m *mockUMAMetricsClient) HasDailyChromiumHistogramCapstone(ctx context.Context,
	in gcpspanner.DailyChromiumHistogramEnumCapstone) (*bool, error) {
	return m.hasDailyChromiumHistogramCapstone(ctx, in)
}

func (m *mockUMAMetricsClient) UpsertDailyChromiumHistogramCapstone(ctx context.Context,
	in gcpspanner.DailyChromiumHistogramEnumCapstone) error {
	return m.upsertDailyChromiumHistogramCapstone(ctx, in)
}

func (m *mockUMAMetricsClient) StoreDailyChromiumHistogramMetrics(ctx context.Context,
	histogramName metricdatatypes.HistogramName, in map[int64]gcpspanner.DailyChromiumHistogramMetric) error {
	return m.storeDailyChromiumHistogramMetrics(ctx, histogramName, in)
}

func (m *mockUMAMetricsClient) SyncLatestDailyChromiumHistogramMetrics(ctx context.Context) error {
	return m.syncLatestDailyChromiumHistogramMetrics(ctx)
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
				upsertDailyChromiumHistogramCapstone:    nil,
				storeDailyChromiumHistogramMetrics:      nil,
				syncLatestDailyChromiumHistogramMetrics: nil,
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
				upsertDailyChromiumHistogramCapstone:    nil,
				storeDailyChromiumHistogramMetrics:      nil,
				syncLatestDailyChromiumHistogramMetrics: nil,
			},
			day:         civil.Date{Year: 2024, Month: 1, Day: 1},
			histogram:   metricdatatypes.HistogramName("test"),
			want:        false,
			expectedErr: nil,
		},
		{
			name: "HasDailyChromiumHistogramCapstone returns error",
			client: &mockUMAMetricsClient{
				hasDailyChromiumHistogramCapstone: func(_ context.Context,
					_ gcpspanner.DailyChromiumHistogramEnumCapstone) (*bool, error) {
					return nil, errors.New("test error")
				},
				upsertDailyChromiumHistogramCapstone:    nil,
				storeDailyChromiumHistogramMetrics:      nil,
				syncLatestDailyChromiumHistogramMetrics: nil,
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
				hasDailyChromiumHistogramCapstone: nil,
				upsertDailyChromiumHistogramCapstone: func(_ context.Context,
					_ gcpspanner.DailyChromiumHistogramEnumCapstone) error {
					return nil
				},
				storeDailyChromiumHistogramMetrics:      nil,
				syncLatestDailyChromiumHistogramMetrics: nil,
			},
			day:         civil.Date{Year: 2024, Month: 1, Day: 1},
			histogram:   metricdatatypes.HistogramName("test"),
			expectedErr: nil,
		},
		{
			name: "UpsertDailyChromiumHistogramCapstone returns error",
			client: &mockUMAMetricsClient{
				hasDailyChromiumHistogramCapstone: nil,
				upsertDailyChromiumHistogramCapstone: func(_ context.Context,
					_ gcpspanner.DailyChromiumHistogramEnumCapstone) error {
					return errors.New("test error")
				},
				storeDailyChromiumHistogramMetrics:      nil,
				syncLatestDailyChromiumHistogramMetrics: nil,
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
			name: "success",
			client: &mockUMAMetricsClient{
				hasDailyChromiumHistogramCapstone:    nil,
				upsertDailyChromiumHistogramCapstone: nil,
				storeDailyChromiumHistogramMetrics: func(_ context.Context, _ metricdatatypes.HistogramName,
					_ map[int64]gcpspanner.DailyChromiumHistogramMetric) error {
					return nil
				},
				syncLatestDailyChromiumHistogramMetrics: func(_ context.Context) error {
					return nil
				},
			},
			day: civil.Date{Year: 2024, Month: 1, Day: 1},
			data: metricdatatypes.BucketDataMetrics{
				1: {Rate: 0.5, LowVolume: false, Milestone: ""},
				2: {Rate: 0.75, LowVolume: false, Milestone: ""},
			},
			expectedErr: nil,
		},
		{
			name: "StoreDailyChromiumHistogramMetrics returns error",
			client: &mockUMAMetricsClient{
				hasDailyChromiumHistogramCapstone:    nil,
				upsertDailyChromiumHistogramCapstone: nil,
				storeDailyChromiumHistogramMetrics: func(_ context.Context, _ metricdatatypes.HistogramName,
					_ map[int64]gcpspanner.DailyChromiumHistogramMetric) error {
					return errors.New("test error")
				},
				syncLatestDailyChromiumHistogramMetrics: nil,
			},
			day: civil.Date{Year: 2024, Month: 1, Day: 1},
			data: metricdatatypes.BucketDataMetrics{
				1: {Rate: 0.5, LowVolume: false, Milestone: ""},
			},
			expectedErr: ErrMetricsSaveFailed,
		},
		{
			name: "SyncLatestDailyChromiumHistogramMetrics returns error",
			client: &mockUMAMetricsClient{
				hasDailyChromiumHistogramCapstone:    nil,
				upsertDailyChromiumHistogramCapstone: nil,
				storeDailyChromiumHistogramMetrics: func(_ context.Context, _ metricdatatypes.HistogramName,
					_ map[int64]gcpspanner.DailyChromiumHistogramMetric) error {
					return nil
				},
				syncLatestDailyChromiumHistogramMetrics: func(_ context.Context) error {
					return errors.New("test error")
				},
			},
			day: civil.Date{Year: 2024, Month: 1, Day: 1},
			data: metricdatatypes.BucketDataMetrics{
				1: {Rate: 0.5, LowVolume: false, Milestone: ""},
			},
			expectedErr: ErrMetricsSaveFailed,
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
