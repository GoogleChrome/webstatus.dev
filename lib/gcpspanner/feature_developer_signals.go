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

package gcpspanner

import (
	"context"
	"log/slog"

	"cloud.google.com/go/spanner"
)

type spannerFeatureDeveloperSignal struct {
	WebFeatureID string `spanner:"WebFeatureID"`
	Votes        int64  `spanner:"Votes"`
	Link         string `spanner:"Link"`
}

type FeatureDeveloperSignal struct {
	WebFeatureKey string `spanner:"WebFeatureKey"`
	Votes         int64  `spanner:"Votes"`
	Link          string `spanner:"Link"`
}

const latestFeatureDeveloperSignalsTableName = "LatestFeatureDeveloperSignals"

// latestFeatureDeveloperSignalsMapper implements syncableEntityMapper for LatestFeatureDeveloperSignals.
type latestFeatureDeveloperSignalsMapper struct{}

func (m latestFeatureDeveloperSignalsMapper) SelectAll() spanner.Statement {
	return spanner.NewStatement(`
	SELECT
		WebFeatureID,
		Votes
	FROM
		LatestFeatureDeveloperSignals`)
}

func (m latestFeatureDeveloperSignalsMapper) Table() string {
	return latestFeatureDeveloperSignalsTableName
}

func (m latestFeatureDeveloperSignalsMapper) DeleteMutation(in spannerFeatureDeveloperSignal) *spanner.Mutation {
	return spanner.Delete(latestFeatureDeveloperSignalsTableName, spanner.Key{in.WebFeatureID})
}

func (m latestFeatureDeveloperSignalsMapper) GetKeyFromExternal(in spannerFeatureDeveloperSignal) string {
	return in.WebFeatureID
}

func (m latestFeatureDeveloperSignalsMapper) GetKeyFromInternal(in spannerFeatureDeveloperSignal) string {
	return in.WebFeatureID
}

func (m latestFeatureDeveloperSignalsMapper) MergeAndCheckChanged(
	in spannerFeatureDeveloperSignal,
	existing spannerFeatureDeveloperSignal,
) (spannerFeatureDeveloperSignal, bool) {
	merged := spannerFeatureDeveloperSignal{
		WebFeatureID: existing.WebFeatureID,
		Votes:        in.Votes,
		Link:         in.Link,
	}

	hasChanged := merged.Votes != existing.Votes || merged.Link != existing.Link

	return merged, hasChanged
}

func (m latestFeatureDeveloperSignalsMapper) GetChildDeleteKeyMutations(
	_ context.Context, _ *Client, _ []spannerFeatureDeveloperSignal) ([]ExtraMutationsGroup, error) {
	return nil, nil
}

func (m latestFeatureDeveloperSignalsMapper) PreDeleteHook(
	_ context.Context, _ *Client, _ []spannerFeatureDeveloperSignal) ([]ExtraMutationsGroup, error) {
	return nil, nil
}

func (c *Client) SyncLatestFeatureDeveloperSignals(ctx context.Context, input []FeatureDeveloperSignal) error {
	slog.InfoContext(ctx, "Syncing latest feature developer signals", "count", len(input))
	signals := make([]spannerFeatureDeveloperSignal, 0, len(input))
	for _, signal := range input {
		webFeatureID, err := c.GetIDFromFeatureKey(ctx, NewFeatureKeyFilter(signal.WebFeatureKey))
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get web feature ID", "error", err)

			return err
		}
		signals = append(signals, spannerFeatureDeveloperSignal{
			WebFeatureID: *webFeatureID,
			Votes:        signal.Votes,
			Link:         signal.Link,
		})
	}

	return newEntitySynchronizer[latestFeatureDeveloperSignalsMapper](c).Sync(ctx, signals)
}

type latestFeatureDeveloperSignalGetAllMapper struct{}

func (m latestFeatureDeveloperSignalGetAllMapper) SelectAll() spanner.Statement {
	return spanner.NewStatement(`
	SELECT
		wf.FeatureKey AS WebFeatureKey,
		lfd.Votes,
		lfd.Link
	FROM
		LatestFeatureDeveloperSignals AS lfd
	JOIN
		WebFeatures AS wf ON lfd.WebFeatureID = wf.ID`)
}

func (c *Client) GetAllLatestFeatureDeveloperSignals(ctx context.Context) ([]FeatureDeveloperSignal, error) {
	return newAllEntityReader[latestFeatureDeveloperSignalGetAllMapper, FeatureDeveloperSignal](c).readAll(ctx)
}
