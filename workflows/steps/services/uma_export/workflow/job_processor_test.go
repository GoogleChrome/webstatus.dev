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
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"cloud.google.com/go/civil"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

var (
	errHasCapstone        = errors.New("error checking for capstone")
	errFetchMetrics       = errors.New("error fetching metrics")
	errParseMetrics       = errors.New("error parsing metrics")
	errSaveMetrics        = errors.New("error saving metrics")
	errSaveCapstone       = errors.New("error saving capstone")
	errReadMetrics        = errors.New("error reading metrics")
	errMetricsResponseNil = errors.New("metrics response is nil")
)

func TestUMAExportJobProcessor_Process(t *testing.T) {
	// Sample test data
	sampleDate := civil.Date{Year: 2024, Month: 9, Day: 18}
	sampleHistogram := metricdatatypes.HistogramName("TestHistogram")
	sampleQuery := metricdatatypes.UMAExportQuery("testquery")
	sampleMetrics := metricdatatypes.BucketDataMetrics{
		1: {Rate: 0.5, Milestone: "M1", LowVolume: false},
		2: {Rate: 0.8, Milestone: "M2", LowVolume: true},
	}

	tests := []struct {
		name            string
		job             JobArguments
		hasCapstone     bool
		hasCapstoneErr  error
		fetchErr        error
		parseRet        metricdatatypes.BucketDataMetrics
		parseErr        error
		saveMetricsErr  error
		saveCapstoneErr error
		want            error
	}{
		{
			name:        "already processed",
			job:         NewJobArguments(sampleQuery, sampleDate, sampleHistogram),
			hasCapstone: true,
			want:        nil,
			// Values that don't matter
			parseRet:        nil,
			parseErr:        nil,
			saveMetricsErr:  nil,
			saveCapstoneErr: nil,
			fetchErr:        nil,
			hasCapstoneErr:  nil,
		},
		{
			name:           "error checking for capstone",
			job:            NewJobArguments(sampleQuery, sampleDate, sampleHistogram),
			hasCapstoneErr: errHasCapstone,
			want:           errHasCapstone,
			// Values that don't matter
			parseRet:        nil,
			parseErr:        nil,
			saveMetricsErr:  nil,
			saveCapstoneErr: nil,
			fetchErr:        nil,
			hasCapstone:     false,
		},
		{
			name:     "error fetching metrics",
			job:      NewJobArguments(sampleQuery, sampleDate, sampleHistogram),
			fetchErr: errFetchMetrics,
			want:     errFetchMetrics,
			// Values that don't matter
			parseRet:        nil,
			parseErr:        nil,
			saveMetricsErr:  nil,
			saveCapstoneErr: nil,
			hasCapstone:     false,
			hasCapstoneErr:  nil,
		},
		{
			name:     "error parsing metrics",
			job:      NewJobArguments(sampleQuery, sampleDate, sampleHistogram),
			parseErr: errParseMetrics,
			want:     errParseMetrics,
			// Values that don't matter
			parseRet:        nil,
			saveMetricsErr:  nil,
			saveCapstoneErr: nil,
			fetchErr:        nil,
			hasCapstone:     false,
			hasCapstoneErr:  nil,
		},
		{
			name:           "error saving metrics",
			job:            NewJobArguments(sampleQuery, sampleDate, sampleHistogram),
			saveMetricsErr: errSaveMetrics,
			want:           errSaveMetrics,
			parseRet:       sampleMetrics,
			// Values that don't matter
			parseErr:        nil,
			saveCapstoneErr: nil,
			fetchErr:        nil,
			hasCapstone:     false,
			hasCapstoneErr:  nil,
		},
		{
			name:            "error saving capstone",
			job:             NewJobArguments(sampleQuery, sampleDate, sampleHistogram),
			saveCapstoneErr: errSaveCapstone,
			want:            errSaveCapstone,
			parseRet:        sampleMetrics,
			// Values that don't matter
			parseErr:       nil,
			saveMetricsErr: nil,
			fetchErr:       nil,
			hasCapstone:    false,
			hasCapstoneErr: nil,
		},
		{
			name:     "success",
			job:      NewJobArguments(sampleQuery, sampleDate, sampleHistogram),
			parseRet: sampleMetrics,
			want:     nil, // No error on successful processing
			// Values that don't matter
			parseErr:        nil,
			saveMetricsErr:  nil,
			saveCapstoneErr: nil,
			fetchErr:        nil,
			hasCapstone:     false,
			hasCapstoneErr:  nil,
		},
		{
			name:     "success no data",
			job:      NewJobArguments(sampleQuery, sampleDate, sampleHistogram),
			parseRet: nil,
			want:     nil, // No error on successful processing
			// Values that don't matter
			parseErr:        nil,
			saveMetricsErr:  nil,
			saveCapstoneErr: nil,
			fetchErr:        nil,
			hasCapstone:     false,
			hasCapstoneErr:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock dependencies
			mockStorer := &mockMetricStorer{
				hasCapstoneFunc: func(_ context.Context, _ civil.Date, _ metricdatatypes.HistogramName) (bool, error) {
					return tt.hasCapstone, tt.hasCapstoneErr
				},
				saveMetricsFunc: func(_ context.Context, _ civil.Date, _ metricdatatypes.BucketDataMetrics) error {
					return tt.saveMetricsErr
				},
				saveCapstoneFunc: func(_ context.Context, _ civil.Date,
					_ metricdatatypes.HistogramName) error {
					return tt.saveCapstoneErr
				},
			}

			mockFetcher := &mockMetricFetcher{
				fetchFunc: func(_ context.Context, _ metricdatatypes.UMAExportQuery) (io.ReadCloser, error) {
					if tt.fetchErr != nil {
						return nil, tt.fetchErr
					}

					return io.NopCloser(strings.NewReader("mock data")), nil
				},
			}

			mockParser := &mockMetricParser{
				parseFunc: func(_ context.Context, _ io.ReadCloser) (metricdatatypes.BucketDataMetrics, error) {
					if tt.parseErr != nil {
						return nil, tt.parseErr
					}

					return tt.parseRet, nil
				},
			}

			// Create the processor
			p := UMAExportJobProcessor{
				metricStorer:  mockStorer,
				metricFetcher: mockFetcher,
				metricsParser: mockParser,
			}

			// Call Process and check the error
			err := p.Process(context.Background(), tt.job)
			if !errors.Is(err, tt.want) {
				t.Errorf("Process() error = %v, wantErr %v", err, tt.want)
			}
		})
	}
}

// Mock dependencies.
type mockMetricStorer struct {
	hasCapstoneFunc  func(ctx context.Context, day civil.Date, histogramName metricdatatypes.HistogramName) (bool, error)
	saveMetricsFunc  func(ctx context.Context, day civil.Date, metrics metricdatatypes.BucketDataMetrics) error
	saveCapstoneFunc func(ctx context.Context, day civil.Date, histogramName metricdatatypes.HistogramName) error
}

func (m *mockMetricStorer) HasCapstone(
	ctx context.Context, day civil.Date, histogramName metricdatatypes.HistogramName) (bool, error) {
	if m.hasCapstoneFunc != nil {
		return m.hasCapstoneFunc(ctx, day, histogramName)
	}

	return false, nil
}

func (m *mockMetricStorer) SaveMetrics(
	ctx context.Context, day civil.Date, metrics metricdatatypes.BucketDataMetrics) error {
	if m.saveMetricsFunc != nil {
		return m.saveMetricsFunc(ctx, day, metrics)
	}

	return nil
}

func (m *mockMetricStorer) SaveCapstone(
	ctx context.Context, day civil.Date, histogramName metricdatatypes.HistogramName) error {
	if m.saveCapstoneFunc != nil {
		return m.saveCapstoneFunc(ctx, day, histogramName)
	}

	return nil
}

type mockMetricFetcher struct {
	fetchFunc func(ctx context.Context, queryName metricdatatypes.UMAExportQuery) (io.ReadCloser, error)
}

func (m *mockMetricFetcher) Fetch(
	ctx context.Context, queryName metricdatatypes.UMAExportQuery) (io.ReadCloser, error) {
	if m.fetchFunc != nil {
		return m.fetchFunc(ctx, queryName)
	}

	return nil, errMetricsResponseNil
}

type mockMetricParser struct {
	parseFunc func(ctx context.Context, rawData io.ReadCloser) (metricdatatypes.BucketDataMetrics, error)
}

func (m *mockMetricParser) Parse(
	ctx context.Context, rawData io.ReadCloser) (metricdatatypes.BucketDataMetrics, error) {
	if m.parseFunc != nil {
		return m.parseFunc(ctx, rawData)
	}

	return nil, errReadMetrics
}
