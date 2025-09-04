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

	"github.com/GoogleChrome/webstatus.dev/lib/developersignaltypes"
)

type JobArguments struct{}

func NewJobArguments() JobArguments {
	return JobArguments{}
}

type SignalsFileParser interface {
	Parse(io.ReadCloser) (*developersignaltypes.FeatureDeveloperSignals, error)
}

type SignalsFileDownloader interface {
	Download(ctx context.Context) (io.ReadCloser, error)
}

type DeveloperSignalsStorer interface {
	SyncLatestFeatureDeveloperSignals(ctx context.Context, data *developersignaltypes.FeatureDeveloperSignals) error
}

type DeveloperSignalsProcessor struct {
	parser     SignalsFileParser
	downloader SignalsFileDownloader
	storer     DeveloperSignalsStorer
}

func NewDeveloperSignalsProcessor(
	parser SignalsFileParser,
	downloader SignalsFileDownloader,
	storer DeveloperSignalsStorer,
) *DeveloperSignalsProcessor {
	return &DeveloperSignalsProcessor{
		parser:     parser,
		downloader: downloader,
		storer:     storer,
	}
}

func (p *DeveloperSignalsProcessor) Process(ctx context.Context, _ JobArguments) error {
	file, err := p.downloader.Download(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "unable to download file", "error", err)

		return err
	}

	data, err := p.parser.Parse(file)
	if err != nil {
		slog.ErrorContext(ctx, "unable to parse file", "error", err)

		return err
	}

	return p.storer.SyncLatestFeatureDeveloperSignals(ctx, data)
}
