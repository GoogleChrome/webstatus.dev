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
	"errors"
	"reflect"
	"testing"
)

func TestFeatureDataOperations(t *testing.T) {
	ctx := context.Background()
	client, cleanup := getTestDatabase(ctx, t)
	defer cleanup()

	// Part 0. Try to get id that does not exist yet.
	data, err := client.GetWebFeatureMetadata(ctx, "id-1")
	if !errors.Is(err, ErrEntityNotFound) {
		t.Errorf("unexpected error %v", err)
	}
	if data != nil {
		t.Error("expected nil data")
	}

	// Part 1. Try to insert the first version
	version1 := &FeatureMetadata{
		WebFeatureID: "id-1",
		Description:  "initial-description",
		CanIUseIDs:   nil, // not required on first try
	}
	err = client.UpsertFeatureMetadata(ctx, *version1)
	if err != nil {
		t.Errorf("failed to upsert %s", err.Error())
	}
	data, err = client.GetWebFeatureMetadata(ctx, "id-1")
	if err != nil {
		t.Errorf("failed to get feature metadata %s", err.Error())
	}
	if !reflect.DeepEqual(version1, data) {
		t.Errorf("unexpected metadata %v", data)
	}

	// Part 2. Upsert the second version
	partialVersion2 := FeatureMetadata{
		WebFeatureID: "id-1",
		Description:  "", // Do not override description
		CanIUseIDs:   []string{"can-i-use-1", "can-i-use-2"},
	}
	err = client.UpsertFeatureMetadata(ctx, partialVersion2)
	if err != nil {
		t.Errorf("failed to upsert again %s", err.Error())
	}

	expectedVersion2 := &FeatureMetadata{
		WebFeatureID: "id-1",
		Description:  "initial-description",
		CanIUseIDs:   []string{"can-i-use-1", "can-i-use-2"},
	}

	data, err = client.GetWebFeatureMetadata(ctx, "id-1")
	if err != nil {
		t.Errorf("failed to get feature metadata %s", err.Error())
	}
	if !reflect.DeepEqual(expectedVersion2, data) {
		t.Errorf("unexpected metadata %v", data)
	}
}
