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

package workflow_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/developersignaltypes"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/developer_signals_consumer/pkg/workflow"
)

// mockSignalsFileParser is a mock implementation of the SignalsFileParser interface.
type mockSignalsFileParser struct {
	data *developersignaltypes.FeatureDeveloperSignals
	err  error
}

func (m *mockSignalsFileParser) Parse(reader io.ReadCloser) (*developersignaltypes.FeatureDeveloperSignals, error) {
	// Ensure the reader is closed, as the real implementation would.
	if reader != nil {
		reader.Close()
	}

	return m.data, m.err
}

// mockSignalsFileDownloader is a mock implementation of the SignalsFileDownloader interface.
type mockSignalsFileDownloader struct {
	reader io.ReadCloser
	err    error
}

func (m *mockSignalsFileDownloader) Download(_ context.Context) (io.ReadCloser, error) {
	return m.reader, m.err
}

// mockDeveloperSignalsStorer is a mock implementation of the DeveloperSignalsStorer interface.
type mockDeveloperSignalsStorer struct {
	err error
}

func (m *mockDeveloperSignalsStorer) SyncLatestFeatureDeveloperSignals(
	_ context.Context, _ *developersignaltypes.FeatureDeveloperSignals) error {
	return m.err
}

func TestDeveloperSignalsProcessor_Process(t *testing.T) {
	ctx := context.Background()
	emptyData := &developersignaltypes.FeatureDeveloperSignals{}
	downloadErr := errors.New("download failed")
	parseErr := errors.New("parsing failed")
	storeErr := errors.New("storage failed")

	testCases := []struct {
		name       string
		downloader workflow.SignalsFileDownloader
		parser     workflow.SignalsFileParser
		storer     workflow.DeveloperSignalsStorer
		wantErr    error
	}{
		{
			name: "Success",
			downloader: &mockSignalsFileDownloader{
				reader: io.NopCloser(strings.NewReader("file content")),
				err:    nil,
			},
			parser:  &mockSignalsFileParser{data: emptyData, err: nil},
			storer:  &mockDeveloperSignalsStorer{err: nil},
			wantErr: nil,
		},
		{
			name:       "Download Error",
			downloader: &mockSignalsFileDownloader{reader: nil, err: downloadErr},
			parser:     &mockSignalsFileParser{data: nil, err: nil},
			storer:     &mockDeveloperSignalsStorer{err: nil},
			wantErr:    downloadErr,
		},
		{
			name: "Parse Error",
			downloader: &mockSignalsFileDownloader{
				reader: io.NopCloser(strings.NewReader("file content")),
				err:    nil,
			},
			parser:  &mockSignalsFileParser{data: nil, err: parseErr},
			storer:  &mockDeveloperSignalsStorer{err: nil},
			wantErr: parseErr,
		},
		{
			name: "Store Error",
			downloader: &mockSignalsFileDownloader{
				reader: io.NopCloser(strings.NewReader("file content")),
				err:    nil,
			},
			parser:  &mockSignalsFileParser{data: emptyData, err: nil},
			storer:  &mockDeveloperSignalsStorer{err: storeErr},
			wantErr: storeErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			processor := workflow.NewDeveloperSignalsProcessor(tc.parser, tc.downloader, tc.storer)
			err := processor.Process(ctx, workflow.NewJobArguments())
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
