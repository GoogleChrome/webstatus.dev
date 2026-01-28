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

// nolint:dupl // WONTFIX
package gcpspanner

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func setupRequiredTablesForWebFeatureChromiumHistogramEnum(
	ctx context.Context,
	t *testing.T,
) map[string]string {
	ret := map[string]string{}
	sampleFeatures := getSampleFeatures()
	for _, feature := range sampleFeatures {
		id, err := spannerClient.upsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert. %s", err.Error())

			continue
		}
		ret[feature.FeatureKey] = *id
	}

	return ret
}

func getSampleWebFeatureChromiumHistogramEnums(
	featureIDMap, enumIDMap map[string]string) []WebFeatureChromiumHistogramEnumValue {
	return []WebFeatureChromiumHistogramEnumValue{
		{WebFeatureID: featureIDMap["feature1"], ChromiumHistogramEnumValueID: enumIDMap["CompressionStreams"]},
		{WebFeatureID: featureIDMap["feature2"], ChromiumHistogramEnumValueID: enumIDMap["ViewTransitions"]},
	}
}

func insertTestWebFeatureChromiumHistogramEnumValues(
	ctx context.Context,
	client *Client,
	t *testing.T,
	values []WebFeatureChromiumHistogramEnumValue,
) {
	err := client.SyncWebFeatureChromiumHistogramEnumValues(ctx, values)
	if err != nil {
		t.Fatalf("failed to sync WebFeatureChromiumHistogramEnumValues. err: %s", err)
	}
}

func (c *Client) readAllWebFeatureChromiumHistogramEnums(
	ctx context.Context, _ *testing.T) ([]WebFeatureChromiumHistogramEnumValue, error) {
	stmt := spanner.NewStatement(
		`SELECT
			WebFeatureID, ChromiumHistogramEnumValueID
		FROM WebFeatureChromiumHistogramEnumValues`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []WebFeatureChromiumHistogramEnumValue
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var chromiumHistogramEnum spannerWebFeatureChromiumHistogramEnum
		if err := row.ToStruct(&chromiumHistogramEnum); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		ret = append(ret, chromiumHistogramEnum.WebFeatureChromiumHistogramEnumValue)
	}

	return ret, nil
}

func sortWebFeatureChromiumHistogramEnums(left, right WebFeatureChromiumHistogramEnumValue) int {
	return cmp.Compare(left.WebFeatureID, right.WebFeatureID)
}

func webFeatureChromiumHistogramEnumEquality(left, right WebFeatureChromiumHistogramEnumValue) bool {
	return left.WebFeatureID == right.WebFeatureID &&
		left.ChromiumHistogramEnumValueID == right.ChromiumHistogramEnumValueID
}

func TestSyncWebFeatureChromiumHistogramEnumValues(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	idMap := setupRequiredTablesForWebFeatureChromiumHistogramEnum(ctx, t)
	sampleEnums := getSampleChromiumHistogramEnums()
	enumIDMap := insertTestChromiumHistogramEnums(ctx, spannerClient, t, sampleEnums)
	sampleEnumValues := getSampleChromiumHistogramEnumValues(enumIDMap)
	insertTestChromiumHistogramEnumValues(ctx, spannerClient, t, sampleEnumValues)
	enumValueLabelToIDMap := make(map[string]string)
	for _, enumValue := range sampleEnumValues {
		id, err := spannerClient.GetIDFromChromiumHistogramEnumValueKey(
			ctx, enumValue.ChromiumHistogramEnumID, enumValue.BucketID)
		if err != nil {
			t.Fatalf("unexpected error getting enum value id. %s", err.Error())
		}
		enumValueLabelToIDMap[enumValue.Label] = *id
	}
	insertTestWebFeatureChromiumHistogramEnumValues(ctx, spannerClient, t,
		getSampleWebFeatureChromiumHistogramEnums(idMap, enumValueLabelToIDMap))

	expected := []WebFeatureChromiumHistogramEnumValue{
		{
			WebFeatureID:                 idMap["feature1"],
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["CompressionStreams"],
		},
		{
			WebFeatureID:                 idMap["feature2"],
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["ViewTransitions"],
		},
	}
	slices.SortFunc(expected, sortWebFeatureChromiumHistogramEnums)

	chromiumHistogramEnums, err := spannerClient.readAllWebFeatureChromiumHistogramEnums(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all ChromiumHistogramEnumValues err: %s", err)
	}
	slices.SortFunc(chromiumHistogramEnums, sortWebFeatureChromiumHistogramEnums)

	if !slices.EqualFunc(expected, chromiumHistogramEnums, webFeatureChromiumHistogramEnumEquality) {
		t.Errorf("unequal ChromiumHistogramEnumValues.\nexpected %+v\nreceived %+v", expected, chromiumHistogramEnums)
	}

	// Test that:
	// 1. Updating an existing enum value does not change anything (e.g. Feature1 still has CompressionStreams)
	// 2. Adding a new enum value works (e.g. Feature3 gets WritingSuggestions)
	// 3. Removing an existing enum value works (e.g. Feature2 loses ViewTransitions)
	err = spannerClient.SyncWebFeatureChromiumHistogramEnumValues(ctx, []WebFeatureChromiumHistogramEnumValue{
		{
			WebFeatureID:                 idMap["feature1"],
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["CompressionStreams"],
		},
		{
			WebFeatureID:                 idMap["feature3"],
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["WritingSuggestions"],
		},
	})
	if err != nil {
		t.Fatalf("failed to sync WebFeatureChromiumHistogramEnumValues. err: %s", err)
	}

	expected = []WebFeatureChromiumHistogramEnumValue{
		{
			WebFeatureID:                 idMap["feature1"],
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["CompressionStreams"],
		},
		{
			WebFeatureID:                 idMap["feature3"],
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["WritingSuggestions"],
		},
	}
	slices.SortFunc(expected, sortWebFeatureChromiumHistogramEnums)

	chromiumHistogramEnums, err = spannerClient.readAllWebFeatureChromiumHistogramEnums(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all ChromiumHistogramEnumValues err: %s", err)
	}
	slices.SortFunc(chromiumHistogramEnums, sortWebFeatureChromiumHistogramEnums)

	if !slices.EqualFunc(expected, chromiumHistogramEnums, webFeatureChromiumHistogramEnumEquality) {
		t.Errorf("unequal ChromiumHistogramEnumValues after update.\nexpected %+v\nreceived %+v",
			expected, chromiumHistogramEnums)
	}
}
