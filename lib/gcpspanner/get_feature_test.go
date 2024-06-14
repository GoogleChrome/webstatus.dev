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
	restartDatabaseContainer(t)
	ctx := context.Background()

	setupRequiredTablesForFeaturesSearch(ctx, spannerClient, t)

	// Test for present feature
	result, err := spannerClient.GetFeature(ctx, NewFeatureKeyFilter("feature2"), defaultWPTMetricView(),
		getDefaultTestBrowserList())
	if err != nil {
		t.Errorf("unexpected error. %s", err.Error())
	}

	expectedResult := valuePtr(getFeatureSearchTestFeature(FeatureSearchTestFId2))

	stabilizeFeatureResult(*result)

	if !AreFeatureResultsEqual(*expectedResult, *result) {
		t.Errorf("unequal results. expected (%+v) received (%+v) ",
			PrettyPrintFeatureResult(*expectedResult), PrettyPrintFeatureResult(*result))
	}

	// Also check the id of the feature.
	id, err := spannerClient.GetIDFromFeatureKey(ctx, NewFeatureKeyFilter("feature2"))
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
	result, err = spannerClient.GetFeature(ctx, NewFeatureKeyFilter("nopefeature2"), defaultWPTMetricView(),
		getDefaultTestBrowserList())
	if !errors.Is(err, ErrQueryReturnedNoResults) {
		t.Errorf("unexpected error. %s", err)
	}
	if result != nil {
		t.Error("expected null result")
	}

	// Also check the id of the feature does not exist
	id, err = spannerClient.GetIDFromFeatureKey(ctx, NewFeatureKeyFilter("nopefeature2"))
	if !errors.Is(err, ErrQueryReturnedNoResults) {
		t.Errorf("unexpected error. %s", err)
	}
	if id != nil {
		t.Error("expected null id")
	}
}
