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

const browserFeatureAvailabilitiesTable = "BrowserFeatureAvailabilities"

// SpannerBrowserFeatureAvailability is a wrapper for the browser availability
// information for a feature stored in spanner.
type SpannerBrowserFeatureAvailability struct {
	WebFeatureID string
	BrowserFeatureAvailability
}

// BrowserFeatureAvailability contains availability information for a particular
// feature in a browser.
type BrowserFeatureAvailability struct {
	BrowserName    string
	BrowserVersion string
}

// InsertBrowserFeatureAvailability will insert the given browser feature availability.
// If the feature availability, does not exist, it will insert a new feature availability.
// If the feature availability exists, it currently does nothing and keeps the existing as-is.
// nolint: dupl // TODO. Will refactor for common patterns.
func (c *Client) InsertBrowserFeatureAvailability(
	ctx context.Context,
	webFeatureID string,
	input BrowserFeatureAvailability) error {
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
			browserFeatureAvailabilitiesTable,
			spanner.Key{*id, input.BrowserName},
			[]string{
				"BrowserVersion",
			})
		if err != nil {
			// Received an error other than not found. Return now.
			if spanner.ErrCode(err) != codes.NotFound {
				return errors.Join(ErrInternalQueryFailure, err)
			}
			featureAvailability := SpannerBrowserFeatureAvailability{
				WebFeatureID:               *id,
				BrowserFeatureAvailability: input,
			}
			m, err := spanner.InsertOrUpdateStruct(browserFeatureAvailabilitiesTable, featureAvailability)
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
			err = txn.BufferWrite([]*spanner.Mutation{m})
			if err != nil {
				return errors.Join(ErrInternalQueryFailure, err)
			}
		}
		// For now, do not overwrite anything for releases.
		return nil

	})
	if err != nil {
		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}
