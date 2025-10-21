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
	"io"
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters"
	"github.com/GoogleChrome/webstatus.dev/lib/webfeaturesmappingtypes"
)

// Downloader is the interface for downloading the mapping file.
type Downloader interface {
	Download(ctx context.Context, url string) (io.ReadCloser, error)
}

// Parser is the interface for parsing the mapping file.
type Parser interface {
	Parse(in io.ReadCloser) (webfeaturesmappingtypes.WebFeaturesMappings, error)
}

// WebFeaturesMappingJobProcessor is the processor for the web features mapping job.
type WebFeaturesMappingJobProcessor struct {
	adapter    *spanneradapters.WebFeaturesMappingConsumer
	downloader Downloader
	parser     Parser
}

// NewWebFeaturesMappingJobProcessor creates a new WebFeaturesMappingJobProcessor.
func NewWebFeaturesMappingJobProcessor(
	adapter *spanneradapters.WebFeaturesMappingConsumer,
	downloader Downloader,
	parser Parser,
) *WebFeaturesMappingJobProcessor {
	return &WebFeaturesMappingJobProcessor{
		adapter:    adapter,
		downloader: downloader,
		parser:     parser,
	}
}

// Process implements the workerpool.JobProcessor interface.
func (p *WebFeaturesMappingJobProcessor) Process(ctx context.Context, args JobArguments) error {
	// Fetch data
	body, err := p.downloader.Download(ctx, args.URL)
	if err != nil {
		slog.ErrorContext(ctx, "unable to download file", "error", err)

		return err
	}

	// Unmarshal data
	mappings, err := p.parser.Parse(body)
	if err != nil {
		slog.ErrorContext(ctx, "unable to parse file", "error", err)

		return err
	}

	return p.adapter.SyncWebFeaturesMappingData(ctx, mappings)
}
