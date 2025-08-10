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

import "context"

type SplitWebFeature struct {
	OriginalFeatureKey string   `spanner:"OriginalFeatureKey"`
	TargetFeatureKeys  []string `spanner:"TargetFeatureKeys"`
}

// SyncSplitWebFeatures reconciles the SplitWebFeatures table with the provided list of features.
// It will insert new details for split web features, update existing ones, and delete any split web features
// that are in the database but not in the provided list.
func (c *Client) SyncSplitWebFeatures(_ context.Context, _ []SplitWebFeature) error {
	// TODO. Will implement once the tables are created.
	// https://github.com/GoogleChrome/webstatus.dev/issues/1669
	return nil
}

// GetSplitWebFeatureByOriginalFeatureKey returns the details about the split feature.
// If details are not found for the feature key, it returns ErrQueryReturnedNoResults.
// Other errors should be investigated and handled appropriately.
func (c *Client) GetSplitWebFeatureByOriginalFeatureKey(
	_ context.Context, _ string) (*SplitWebFeature, error) {
	// TODO. Will implement once the tables are created.
	// https://github.com/GoogleChrome/webstatus.dev/issues/1669
	return nil, ErrQueryReturnedNoResults
}
