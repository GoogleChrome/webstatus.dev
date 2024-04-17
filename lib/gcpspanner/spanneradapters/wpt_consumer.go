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
// limitations under the License./workspaces/webstatus.dev/lib/gcpspanner/spanneradapters/wptconsumertypes

package spanneradapters

import (
	"context"
	"errors"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/wptconsumertypes"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

// NewWPTWorkflowConsumer constructs an adapter for the wpt consumer service.
func NewWPTWorkflowConsumer(client WPTWorkflowSpannerClient) *WPTConsumer {
	return &WPTConsumer{client: client}
}

// WPTWorkflowSpannerClient expects a subset of the functionality from lib/gcpspanner that
// only apply to inserting WPT data.
type WPTWorkflowSpannerClient interface {
	InsertWPTRun(ctx context.Context, run gcpspanner.WPTRun) error
	UpsertWPTRunFeatureMetrics(ctx context.Context, externalRunID int64, in []gcpspanner.WPTRunFeatureMetric) error
}

// WPTConsumer is the adapter that takes data from the WPT workflow and prepares
// it to be stored in the spanner database.
type WPTConsumer struct {
	client WPTWorkflowSpannerClient
}

func (w *WPTConsumer) InsertWPTRun(ctx context.Context, in shared.TestRun) error {
	// Input validation before trying to insert to make sure it has the appropriate values.
	// Make sure channel == 'stable' or 'experimental'
	if in.Channel() != shared.StableLabel && in.Channel() != shared.ExperimentalLabel {
		return wptconsumertypes.ErrInvalidDataFromWPT
	}
	run := gcpspanner.WPTRun{
		RunID:            in.ID,
		TimeStart:        in.TimeStart,
		TimeEnd:          in.TimeEnd,
		BrowserName:      in.BrowserName,
		BrowserVersion:   in.BrowserVersion,
		Channel:          in.Channel(),
		OSName:           in.OSName,
		OSVersion:        in.OSVersion,
		FullRevisionHash: in.FullRevisionHash,
	}

	err := w.client.InsertWPTRun(ctx, run)
	if err != nil {
		return errors.Join(wptconsumertypes.ErrUnableToStoreWPTRun, err)
	}

	return nil
}

func convertWorkflowMetricsToGCPMetrics(
	metricsPerFeature map[string]wptconsumertypes.WPTFeatureMetric,
) []gcpspanner.WPTRunFeatureMetric {
	ret := make([]gcpspanner.WPTRunFeatureMetric, 0, len(metricsPerFeature))
	for featureID, consumerMetric := range metricsPerFeature {
		ret = append(ret, gcpspanner.WPTRunFeatureMetric{
			FeatureID:     featureID,
			TotalTests:    consumerMetric.TotalTests,
			TestPass:      consumerMetric.TestPass,
			TotalSubtests: consumerMetric.TotalSubtests,
			SubtestPass:   consumerMetric.SubtestPass,
		})
	}

	return ret

}

func (w *WPTConsumer) UpsertWPTRunFeatureMetrics(
	ctx context.Context,
	runID int64,
	metricsPerFeature map[string]wptconsumertypes.WPTFeatureMetric) error {
	metrics := make([]gcpspanner.WPTRunFeatureMetric, 0, len(metricsPerFeature))
	metrics = append(metrics, convertWorkflowMetricsToGCPMetrics(metricsPerFeature)...)

	if len(metrics) > 0 {
		err := w.client.UpsertWPTRunFeatureMetrics(ctx, runID, metrics)
		if err != nil {
			return errors.Join(wptconsumertypes.ErrUnableToStoreWPTRunFeatureMetrics, err)
		}
	}

	return nil
}

// NewWPTRun creates a gcpspanner WPTRun from the incoming TestRun from wpt.fyi.
func NewWPTRun(testRun shared.TestRun) gcpspanner.WPTRun {
	return gcpspanner.WPTRun{
		RunID:            testRun.ID,
		BrowserName:      testRun.BrowserName,
		BrowserVersion:   testRun.BrowserVersion,
		TimeStart:        testRun.TimeStart,
		TimeEnd:          testRun.TimeEnd,
		Channel:          testRun.Channel(),
		OSName:           testRun.OSName,
		OSVersion:        testRun.OSVersion,
		FullRevisionHash: testRun.FullRevisionHash,
	}
}
