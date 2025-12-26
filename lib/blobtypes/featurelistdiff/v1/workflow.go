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

package v1

import (
	"context"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
)

// Implements StateCompareWorkflow interface.
type FeatureDiffWorkflow struct {
	diff       *FeatureDiff
	fetcher    FeatureFetcher
	summaryGen SummaryGenerator
}

// FeatureFetcher abstracts the external API.
type FeatureFetcher interface {
	GetFeature(ctx context.Context, featureID string) (*backendtypes.GetFeatureResult, error)
}

type SummaryGenerator interface {
	GenerateJSONSummary(diff FeatureDiff) ([]byte, error)
}

func NewFeatureDiffWorkflow(fetcher FeatureFetcher, summaryGen SummaryGenerator) *FeatureDiffWorkflow {
	return &FeatureDiffWorkflow{
		diff:       new(FeatureDiff),
		fetcher:    fetcher,
		summaryGen: summaryGen,
	}
}

func (w *FeatureDiffWorkflow) HasRemovedFeatures() bool {
	return len(w.diff.Removed) > 0
}

func (w *FeatureDiffWorkflow) HasChanges() bool {
	return w.diff.HasChanges()
}

func (w *FeatureDiffWorkflow) HasDataChanges() bool {
	return w.diff.HasDataChanges()
}

func (w *FeatureDiffWorkflow) SetQueryChanged(changed bool) {
	w.diff.QueryChanged = changed
}

func (w *FeatureDiffWorkflow) GenerateJSONSummary() ([]byte, error) {
	return w.summaryGen.GenerateJSONSummary(*w.diff)
}

func (w *FeatureDiffWorkflow) GetDiff() *FeatureDiff {
	return w.diff
}
