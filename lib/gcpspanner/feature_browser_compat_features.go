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

	"cloud.google.com/go/spanner"
)

func (c *Client) UpsertBrowserCompatFeatures(ctx context.Context, featureID string, compatFeatures []string) error {
	// Create a delete mutation for the specified KeyRange
	del := spanner.Delete("WebFeatureBrowserCompatFeatures", spanner.Key{featureID}.AsPrefix())

	// Then, insert new ones
	muts := make([]*spanner.Mutation, 0, len(compatFeatures)+1)
	muts = append(muts, del)
	for _, compat := range compatFeatures {
		muts = append(muts, spanner.InsertOrUpdate("WebFeatureBrowserCompatFeatures", []string{
			"ID", "CompatFeature",
		}, []interface{}{featureID, compat}))
	}
	_, err := c.Apply(ctx, muts)

	return err
}
