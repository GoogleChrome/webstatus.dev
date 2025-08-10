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

type MovedWebFeature struct {
	OriginalFeatureKey string `spanner:"OriginalFeatureKey"`
	NewFeatureKey      string `spanner:"NewFeatureKey"`
}

// SyncMovedWebFeatures reconciles the MovedWebFeatures table with the provided list of features.
// It will insert new details for moved web features, update existing ones, and delete any moved web features
// that are no longer present in the provided list.
func (c *Client) SyncMovedWebFeatures(_ context.Context, _ []MovedWebFeature) error {
	// TODO. Will implement once the tables are created.
	// https://github.com/GoogleChrome/webstatus.dev/issues/1669
	return nil
}

// GetMovedWebFeatureDetailsByOriginalFeatureKey returns the details about the moved feature.
// If details are not found for the feature key, it returns ErrQueryReturnedNoResults.
// Other errors should be investigated and handled appropriately.
func (c *Client) GetMovedWebFeatureDetailsByOriginalFeatureKey(
	_ context.Context, _ string) (*MovedWebFeature, error) {
	// TODO. Will implement once the tables are created.
	// https://github.com/GoogleChrome/webstatus.dev/issues/1669
	return nil, ErrQueryReturnedNoResults
}
