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

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
)

const featureSpecsTable = "FeatureSpecs"

// SpannerFeatureSpec is a wrapper for the feature spec
// information for a feature stored in spanner.
type SpannerFeatureSpec struct {
	WebFeatureID string
	FeatureSpec
}

// FeatureSpec contains availability information for a particular
// feature in a browser.
type FeatureSpec struct {
	Links []string
}

// InsertFeatureSpec will insert the given feature spec information.
// If the spec info, does not exist, it will insert a new spec info.
// If the spec info exists, it currently overwrites the data.
func (c *Client) UpsertFeatureSpec(
	ctx context.Context,
	webFeatureID string,
	input FeatureSpec) error {
	id, err := c.GetIDFromFeatureKey(ctx, NewFeatureKeyFilter(webFeatureID))
	if err != nil {
		return err
	}
	if id == nil {
		return ErrInternalQueryFailure
	}
	_, err = c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := txn.ReadRow(
			ctx,
			featureSpecsTable,
			spanner.Key{*id},
			[]string{
				"Links",
			})
		if err != nil {
			// Received an error other than not found. Return now.
			if spanner.ErrCode(err) != codes.NotFound {
				return errors.Join(ErrInternalQueryFailure, err)
			}
		}
		featureSpec := SpannerFeatureSpec{
			WebFeatureID: *id,
			FeatureSpec:  input,
		}
		m, err := spanner.InsertOrUpdateStruct(featureSpecsTable, featureSpec)
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}
		err = txn.BufferWrite([]*spanner.Mutation{m})
		if err != nil {
			return errors.Join(ErrInternalQueryFailure, err)
		}

		return nil

	})
	if err != nil {
		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}
