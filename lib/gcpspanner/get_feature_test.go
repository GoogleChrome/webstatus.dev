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

package gcpspanner

import (
	"context"
	"errors"
	"testing"
)

func TestGetFeature(t *testing.T) {
	client := getTestDatabase(t)
	ctx := context.Background()

	setupRequiredTablesForFeaturesSearch(ctx, client, t)

	// Test for present feature
	result, err := client.GetFeature(ctx, NewFeatureIDFilter("feature2"))
	if err != nil {
		t.Errorf("unexpected error. %s", err.Error())
	}

	expectedResult := valuePtr(getFeatureSearchTestFeature(FeatureSearchTestFeature2, TestViewMetric))

	stabilizeFeatureResult(*result)

	if !AreFeatureResultsEqual(*expectedResult, *result) {
		t.Errorf("unequal results. expected (%+v) received (%+v) ",
			PrettyPrintFeatureResult(*expectedResult), PrettyPrintFeatureResult(*result))
	}

	// Also check the id of the feature.
	id, err := client.GetIDFromFeatureID(ctx, NewFeatureIDFilter("feature2"))
	if err != nil {
		t.Errorf("unexpected error. %s", err.Error())
	}
	if id == nil {
		t.Error("expected an id")
	} else if len(*id) != 36 {
		// TODO. Assert it is indeed a uuid. For now, check the length.
		t.Errorf("expected auto-generated uuid. id is only length %d", len(*id))
	}

	// Test for non existent feature
	result, err = client.GetFeature(ctx, NewFeatureIDFilter("nopefeature2"))
	if !errors.Is(err, ErrQueryReturnedNoResults) {
		t.Errorf("unexpected error. %s", err)
	}
	if result != nil {
		t.Error("expected null result")
	}

	// Also check the id of the feature does not exist
	id, err = client.GetIDFromFeatureID(ctx, NewFeatureIDFilter("nopefeature2"))
	if !errors.Is(err, ErrQueryReturnedNoResults) {
		t.Errorf("unexpected error. %s", err)
	}
	if id != nil {
		t.Error("expected null id")
	}
}
