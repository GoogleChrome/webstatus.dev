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
	"slices"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func getSampleChromiumHistogramEnums() []ChromiumHistogramEnum {
	return []ChromiumHistogramEnum{
		{
			HistogramName: "AnotherHistogram",
		},
		{
			HistogramName: "WebDXFeatureObserver",
		},
	}
}

func insertSampleChromiumHistogramEnums(ctx context.Context, t *testing.T, c *Client) map[string]string {
	enums := getSampleChromiumHistogramEnums()
	m := make(map[string]string, len(enums))
	for _, enum := range enums {
		id, err := c.UpsertChromiumHistogramEnum(ctx, enum)
		if err != nil {
			t.Fatalf("unable to insert sample histogram enums. error %s", err)
		}
		m[enum.HistogramName] = *id
	}

	return m
}

func insertGivenSampleChromiumHistogramEnums(
	ctx context.Context,
	client *Client,
	t *testing.T,
	values []ChromiumHistogramEnum) map[string]string {
	chromiumHistogramEnumIDMap := make(map[string]string, len(values))
	for _, enum := range values {
		id, err := client.UpsertChromiumHistogramEnum(ctx, enum)
		if err != nil {
			t.Fatalf("unable to insert sample histogram enums. error %s", err)
		}
		chromiumHistogramEnumIDMap[enum.HistogramName] = *id
	}

	return chromiumHistogramEnumIDMap
}

// Helper method to get all the enums in a stable order.
func (c *Client) ReadAllChromiumHistogramEnums(ctx context.Context, t *testing.T) ([]ChromiumHistogramEnum, error) {
	stmt := spanner.NewStatement(
		`SELECT
			ID, HistogramName
		FROM ChromiumHistogramEnums
		ORDER BY HistogramName ASC`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []ChromiumHistogramEnum
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var enum spannerChromiumHistogramEnum
		if err := row.ToStruct(&enum); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		if enum.ID == "" {
			t.Error("retrieved enum ID is empty")
		}
		ret = append(ret, enum.ChromiumHistogramEnum)
	}

	return ret, nil
}

func TestUpsertChromiumHistogramEnum(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	insertSampleChromiumHistogramEnums(ctx, t, spannerClient)
	enums, err := spannerClient.ReadAllChromiumHistogramEnums(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	sampleHistogramsEnums := getSampleChromiumHistogramEnums()
	if !slices.Equal[[]ChromiumHistogramEnum](getSampleChromiumHistogramEnums(), enums) {
		t.Errorf("unequal enums. expected %+v actual %+v", sampleHistogramsEnums, enums)
	}

	_, err = spannerClient.UpsertChromiumHistogramEnum(ctx, ChromiumHistogramEnum{
		HistogramName: "WebDXFeatureObserver",
	})
	if err != nil {
		t.Errorf("unexpected error during update. %s", err.Error())
	}
	// TODO: Try to upsert with the generated UUID.
}
