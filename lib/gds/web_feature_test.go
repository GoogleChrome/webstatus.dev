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

package gds

import (
	"context"
	"slices"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// nolint: exhaustruct // No need to use every option of 3rd party struct.
func TestFeatureDataOperations(t *testing.T) {
	ctx := context.Background()
	client, cleanup := getTestDatabase(ctx, t)
	defer cleanup()

	// Part 1. Try to insert the first version
	err := client.UpsertFeatureData(ctx, "id-1", web_platform_dx__web_features.FeatureData{
		Name: "version-1-name",
	})
	if err != nil {
		t.Errorf("failed to upsert %s", err.Error())
	}
	features, _, err := client.ListWebFeataureData(ctx, nil)
	if err != nil {
		t.Errorf("failed to list %s", err.Error())
	}

	expectedFeatures := []backend.Feature{{FeatureId: "id-1", Spec: nil, Name: "version-1-name"}}
	if !slices.Equal[[]backend.Feature](features, expectedFeatures) {
		t.Errorf("slices not equal actual [%v] expected [%v]", features, expectedFeatures)
	}

	// Part 2. Upsert the second version
	err = client.UpsertFeatureData(ctx, "id-1", web_platform_dx__web_features.FeatureData{
		Name: "version-2-name",
	})
	if err != nil {
		t.Errorf("failed to upsert again %s", err.Error())
	}

	features, _, err = client.ListWebFeataureData(ctx, nil)
	if err != nil {
		t.Errorf("failed to list %s", err.Error())
	}

	expectedFeatures = []backend.Feature{{FeatureId: "id-1", Spec: nil, Name: "version-2-name"}}
	if !slices.Equal[[]backend.Feature](features, expectedFeatures) {
		t.Errorf("slices not equal actual [%v] expected [%v]", features, expectedFeatures)
	}
}
