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
	"errors"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const featureDiscouragedDetailsTable = "FeatureDiscouragedDetails"

// FeatureDiscouragedDetails contains information about why a feature is discouraged.
type FeatureDiscouragedDetails struct {
	AccordingTo  []string `spanner:"AccordingTo"`
	Alternatives []string `spanner:"Alternatives"`
}

// SpannerFeatureDiscouragedDetails is a wrapper for the FeatureDiscouragedDetails that is actually
// stored in spanner.
type spannerFeatureDiscouragedDetails struct {
	WebFeatureID string `spanner:"WebFeatureID"`
	FeatureDiscouragedDetails
}

// Implements the Mapping interface for FeatureDiscouragedDetails and SpannerFeatureDiscouragedDetails.
type featureDiscouragedDetailsSpannerMapper struct{}

func (m featureDiscouragedDetailsSpannerMapper) GetKeyFromExternal(in spannerFeatureDiscouragedDetails) string {
	return in.WebFeatureID
}

func (m featureDiscouragedDetailsSpannerMapper) Table() string {
	return featureDiscouragedDetailsTable
}

func (m featureDiscouragedDetailsSpannerMapper) Merge(
	incoming spannerFeatureDiscouragedDetails,
	existing spannerFeatureDiscouragedDetails) spannerFeatureDiscouragedDetails {
	existing.AccordingTo = incoming.AccordingTo
	existing.Alternatives = incoming.Alternatives

	return existing
}

func (m featureDiscouragedDetailsSpannerMapper) SelectOne(id string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		WebFeatureID, AccordingTo, Alternatives
	FROM %s
	WHERE WebFeatureID = @webFeatureID
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"webFeatureID": id,
	}
	stmt.Params = parameters

	return stmt
}

func (c *Client) UpsertFeatureDiscouragedDetails(
	ctx context.Context, featureID string, in FeatureDiscouragedDetails) error {
	id, err := c.GetIDFromFeatureKey(ctx, NewFeatureKeyFilter(featureID))
	if err != nil {
		return err
	}
	if id == nil {
		return ErrInternalQueryFailure
	}

	return newEntityWriter[featureDiscouragedDetailsSpannerMapper](c).upsert(ctx, spannerFeatureDiscouragedDetails{
		WebFeatureID:              *id,
		FeatureDiscouragedDetails: in,
	})
}

func (c *Client) getAllDiscouragedFeatureIDs(ctx context.Context, txn *spanner.ReadOnlyTransaction) (
	[]string, error) {
	var featureIDs []string
	stmt := spanner.NewStatement(`SELECT WebFeatureID FROM ` + featureDiscouragedDetailsTable)

	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}

		var featureID string
		if err := row.Columns(&featureID); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}

		featureIDs = append(featureIDs, featureID)
	}

	return featureIDs, nil
}
