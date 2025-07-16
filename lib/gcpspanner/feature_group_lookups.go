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

const featureGroupIDsLookupTable = "FeatureGroupIDsLookup"

type FeatureGroupIDsLookup struct {
	ID           string // This is the GroupID. See infra/storage/spanner/migrations/000018.sql
	WebFeatureID string
	Depth        int64
}

func (c *Client) UpsertFeatureGroupLookups(
	ctx context.Context, lookups []FeatureGroupIDsLookup) error {
	// TODO: We should do a diff and delete group lookups no longer needed.
	// This hasn't happened yet.

	return runConcurrentBatch(ctx,
		c, func(entityChan chan<- FeatureGroupIDsLookup) {
			for _, entity := range lookups {
				entityChan <- entity
			}
		}, featureGroupIDsLookupTable)
}
