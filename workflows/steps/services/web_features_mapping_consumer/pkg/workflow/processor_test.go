// Copyright 2025 Google LLC
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

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters"
	"github.com/GoogleChrome/webstatus.dev/lib/webfeaturesmappingtypes"
)

// mockDownloader is a mock implementation of the Downloader interface.
type mockDownloader struct {
	io.ReadCloser
	err error
}

func (m *mockDownloader) Download(context.Context, string) (io.ReadCloser, error) {
	return m.ReadCloser, m.err
}

// mockParser is a mock implementation of the Parser interface.
type mockParser struct {
	mappings webfeaturesmappingtypes.WebFeaturesMappings
	err      error
}

func (m *mockParser) Parse(io.ReadCloser) (webfeaturesmappingtypes.WebFeaturesMappings, error) {
	return m.mappings, m.err
}

// mockWebFeaturesMappingStorer is a mock implementation of the spanneradapters.WebFeaturesMappingClient interface.
type mockWebFeaturesMappingStorer struct {
	err error
}

func (m *mockWebFeaturesMappingStorer) SyncWebFeaturesMappingData(
	_ context.Context, _ []gcpspanner.WebFeaturesMappingData) error {
	return m.err
}

func TestWebFeaturesMappingJobProcessor_Process(t *testing.T) {
	ctx := context.Background()
	args := NewJobArguments("http://example.com")
	emptyMappings := webfeaturesmappingtypes.WebFeaturesMappings{}
	downloadErr := errors.New("download failed")
	parseErr := errors.New("parsing failed")
	storeErr := errors.New("storage failed")

	testCases := []struct {
		name       string
		downloader Downloader
		parser     Parser
		storer     *mockWebFeaturesMappingStorer
		wantErr    error
	}{
		{
			name: "Success",
			downloader: &mockDownloader{
				ReadCloser: io.NopCloser(strings.NewReader("file content")),
				err:        nil,
			},
			parser:  &mockParser{mappings: emptyMappings, err: nil},
			storer:  &mockWebFeaturesMappingStorer{err: nil},
			wantErr: nil,
		},
		{
			name:       "Download Error",
			downloader: &mockDownloader{ReadCloser: nil, err: downloadErr},
			parser:     &mockParser{mappings: nil, err: nil},
			storer:     &mockWebFeaturesMappingStorer{err: nil},
			wantErr:    downloadErr,
		},
		{
			name: "Parse Error",
			downloader: &mockDownloader{
				ReadCloser: io.NopCloser(strings.NewReader("file content")),
				err:        nil,
			},
			parser:  &mockParser{mappings: nil, err: parseErr},
			storer:  &mockWebFeaturesMappingStorer{err: nil},
			wantErr: parseErr,
		},
		{
			name: "Store Error",
			downloader: &mockDownloader{
				ReadCloser: io.NopCloser(strings.NewReader("file content")),
				err:        nil,
			},
			parser:  &mockParser{mappings: emptyMappings, err: nil},
			storer:  &mockWebFeaturesMappingStorer{err: storeErr},
			wantErr: storeErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := spanneradapters.NewWebFeaturesMappingConsumer(tc.storer)
			processor := NewWebFeaturesMappingJobProcessor(adapter, tc.downloader, tc.parser)
			err := processor.Process(ctx, args)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
