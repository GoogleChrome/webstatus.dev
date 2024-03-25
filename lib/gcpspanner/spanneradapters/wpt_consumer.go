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

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	wptWorkflow "github.com/GoogleChrome/webstatus.dev/workflows/steps/services/wpt_consumer/pkg/workflow"
)

// NewWPTWorkflowConsumer constructs an adapter for the wpt consumer service.
func NewWPTWorkflowConsumer(client WPTWorkflowSpannerClient) *WPTConsumer {
	return &WPTConsumer{client: client}
}

// WPTWorkflowSpannerClient expects a subset of the functionality from lib/gcpspanner that
// only apply to inserting WPT data.
type WPTWorkflowSpannerClient interface {
	InsertWPTRun(ctx context.Context, run gcpspanner.WPTRun) error
	UpsertWPTRunFeatureMetric(ctx context.Context, externalRunID int64, in gcpspanner.WPTRunFeatureMetric) error
}

type WPTConsumer struct {
	client WPTWorkflowSpannerClient
}

func (w *WPTConsumer) InsertWPTRun(ctx context.Context, in wptWorkflow.WPTRun) error {
	// TODO. Add input validation before trying to insert to make sure it has the appropriate values.
	// Example: making sure channel == 'stable' or 'experimental'
	run := gcpspanner.WPTRun{
		RunID:            in.ID,
		TimeStart:        in.TimeStart,
		TimeEnd:          in.TimeEnd,
		BrowserName:      in.BrowserName,
		BrowserVersion:   in.BrowserVersion,
		Channel:          in.Channel,
		OSName:           in.OSName,
		OSVersion:        in.OSVersion,
		FullRevisionHash: in.FullRevisionHash,
	}

	return w.client.InsertWPTRun(ctx, run)
}

func (w *WPTConsumer) UpsertWPTRunFeatureMetric(
	ctx context.Context,
	runID int64,
	metricsPerFeature map[string]wptWorkflow.WPTFeatureMetric) error {
	for featureID, consumerMetric := range metricsPerFeature {
		metric := gcpspanner.WPTRunFeatureMetric{
			FeatureID:  featureID,
			TotalTests: consumerMetric.TotalTests,
			TestPass:   consumerMetric.TestPass,
		}
		err := w.client.UpsertWPTRunFeatureMetric(ctx, runID, metric)
		if err != nil {
			return err
		}
	}

	return nil
}
