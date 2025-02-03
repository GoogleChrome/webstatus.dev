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

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func (c *Client) getFeatureIDsForEachExcludedFeatureKey(ctx context.Context, txn *spanner.ReadOnlyTransaction) (
	[]string, error) {
	var featureIDs []string
	stmt := spanner.NewStatement(`SELECT wf.ID FROM ExcludedFeatureKeys efk
		INNER JOIN WebFeatures wf ON wf.FeatureKey = efk.FeatureKey`)

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
