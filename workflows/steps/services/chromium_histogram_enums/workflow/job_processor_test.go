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

	"github.com/GoogleChrome/webstatus.dev/lib/httputils"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

var (
	errFetchEnums         = errors.New("error fetching enums")
	errParseEnums         = errors.New("error parsing enums")
	errSaveHistogramEnums = errors.New("error saving histogram enums")
	errEnumsResponseNil   = errors.New("enums response is nil")
	errReadEnums          = errors.New("error reading enums")
)

func TestProcess(t *testing.T) {
	sampleHistograms := []metricdatatypes.HistogramName{metricdatatypes.WebDXFeatureEnum, "TestHistogram2"}
	sampleMapping := metricdatatypes.HistogramMapping{
		metricdatatypes.WebDXFeatureEnum: {
			{
				Label: "EnumValue1",
				Value: 1,
			},
			{
				Label: "EnumValue2",
				Value: 2,
			},
		},
		"TestHistogram2": {
			{
				Label: "EnumValue3",
				Value: 3,
			},
			{
				Label: "EnumValue4",
				Value: 4,
			},
		},
	}

	tests := []struct {
		name                  string
		job                   JobArguments
		fetchErr              error
		parseErr              error
		saveHistogramEnumsErr error
		want                  error
	}{
		{
			name:                  "error fetching enums",
			job:                   NewJobArguments(sampleHistograms),
			fetchErr:              errFetchEnums,
			saveHistogramEnumsErr: nil,
			parseErr:              nil,
			want:                  errFetchEnums,
		},
		{
			name:                  "error parsing enums",
			job:                   NewJobArguments(sampleHistograms),
			parseErr:              errParseEnums,
			saveHistogramEnumsErr: nil,
			fetchErr:              nil,
			want:                  errParseEnums,
		},
		{
			name:                  "error saving histogram enums",
			job:                   NewJobArguments(sampleHistograms),
			saveHistogramEnumsErr: errSaveHistogramEnums,
			parseErr:              nil,
			fetchErr:              nil,
			want:                  errSaveHistogramEnums,
		},
		{
			name:                  "success",
			job:                   NewJobArguments(sampleHistograms),
			want:                  nil, // No error on successful processing
			saveHistogramEnumsErr: nil,
			parseErr:              nil,
			fetchErr:              nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock dependencies
			mockFetcher := &mockEnumsFetcher{
				fetchFunc: func(_ context.Context, _ ...httputils.FetchOption) (io.ReadCloser, error) {
					if tt.fetchErr != nil {
						return nil, tt.fetchErr
					}

					return io.NopCloser(strings.NewReader("mock enums data")), nil
				},
			}

			mockParser := &mockEnumsParser{
				parseFunc: func(
					_ context.Context,
					_ io.ReadCloser,
					_ []metricdatatypes.HistogramName) (metricdatatypes.HistogramMapping, error) {
					if tt.parseErr != nil {
						return nil, tt.parseErr
					}

					return sampleMapping, nil
				},
			}

			mockStorer := &mockHistogramStorer{
				saveHistogramEnumsFunc: func(_ context.Context, _ metricdatatypes.HistogramMapping) error {
					return tt.saveHistogramEnumsErr
				},
			}

			// Create the processor
			p := NewChromiumHistogramEnumsJobProcessor(
				mockFetcher,
				mockParser,
				mockStorer,
			)

			// Call Process and check the error
			err := p.Process(context.Background(), tt.job)
			if !errors.Is(err, tt.want) {
				t.Errorf("Process() error = %v, wantErr %v", err, tt.want)
			}
		})
	}
}

// Mock dependencies.
type mockEnumsFetcher struct {
	fetchFunc func(ctx context.Context, opts ...httputils.FetchOption) (io.ReadCloser, error)
}

func (m *mockEnumsFetcher) Fetch(ctx context.Context, opts ...httputils.FetchOption) (io.ReadCloser, error) {
	if m.fetchFunc != nil {
		return m.fetchFunc(ctx, opts...)
	}

	return nil, errEnumsResponseNil
}

type mockEnumsParser struct {
	parseFunc func(
		ctx context.Context,
		rawData io.ReadCloser,
		histograms []metricdatatypes.HistogramName) (metricdatatypes.HistogramMapping, error)
}

func (m *mockEnumsParser) Parse(
	ctx context.Context,
	rawData io.ReadCloser,
	histograms []metricdatatypes.HistogramName) (metricdatatypes.HistogramMapping, error) {
	if m.parseFunc != nil {
		return m.parseFunc(ctx, rawData, histograms)
	}

	return nil, errReadEnums
}

type mockHistogramStorer struct {
	saveHistogramEnumsFunc func(ctx context.Context, mapping metricdatatypes.HistogramMapping) error
}

func (m *mockHistogramStorer) SaveHistogramEnums(ctx context.Context, mapping metricdatatypes.HistogramMapping) error {
	if m.saveHistogramEnumsFunc != nil {
		return m.saveHistogramEnumsFunc(ctx, mapping)
	}

	return nil
}
