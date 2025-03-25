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
	"fmt"

	"cloud.google.com/go/spanner"
)

const browserFeatureAvailabilitiesTable = "BrowserFeatureAvailabilities"

// Implements the entityMapper interface for BrowserFeatureAvailability and SpannerBrowserFeatureAvailability.
type browserFeatureAvailabilityMapper struct{}

func (m browserFeatureAvailabilityMapper) Table() string {
	return browserFeatureAvailabilitiesTable
}

type browserFeatureAvailabilityKey struct {
	WebFeatureID string
	BrowserName  string
}

func (m browserFeatureAvailabilityMapper) SelectOne(key browserFeatureAvailabilityKey) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		WebFeatureID, BrowserName, BrowserVersion
	FROM %s
	WHERE WebFeatureID = @webFeatureID AND BrowserName = @browserName
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"webFeatureID": key.WebFeatureID,
		"browserName":  key.BrowserName,
	}
	stmt.Params = parameters

	return stmt
}

func (m browserFeatureAvailabilityMapper) GetKey(in spannerBrowserFeatureAvailability) browserFeatureAvailabilityKey {
	return browserFeatureAvailabilityKey{
		WebFeatureID: in.WebFeatureID,
		BrowserName:  in.BrowserName,
	}
}

func (m browserFeatureAvailabilityMapper) DeleteKey(key browserFeatureAvailabilityKey) spanner.Key {
	return spanner.Key{key.WebFeatureID, key.BrowserName}
}

// spannerBrowserFeatureAvailability is a wrapper for the browser availability
// information for a feature stored in spanner.
type spannerBrowserFeatureAvailability struct {
	WebFeatureID string
	BrowserFeatureAvailability
}

// BrowserFeatureAvailability contains availability information for a particular
// feature in a browser.
type BrowserFeatureAvailability struct {
	BrowserName    string
	BrowserVersion string
}

// UpsertBrowserFeatureAvailability will upsert the given browser feature availability.
func (c *Client) UpsertBrowserFeatureAvailability(
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
	featureAvailability := spannerBrowserFeatureAvailability{
		WebFeatureID:               *id,
		BrowserFeatureAvailability: input,
	}

	return newUniqueEntityWriter[
		browserFeatureAvailabilityMapper,
		spannerBrowserFeatureAvailability,
		spannerBrowserFeatureAvailability](c).upsertUniqueKey(ctx, featureAvailability)
}

func (c *Client) fetchAllBrowserAvailabilitiesWithTransaction(
	ctx context.Context, txn *spanner.ReadOnlyTransaction) ([]spannerBrowserFeatureAvailability, error) {
	var availabilities []spannerBrowserFeatureAvailability
	iter := txn.Read(ctx, browserFeatureAvailabilitiesTable, spanner.AllKeys(), []string{
		"BrowserName",
		"BrowserVersion",
		"WebFeatureID",
	})
	defer iter.Stop()
	err := iter.Do(func(row *spanner.Row) error {
		var entry spannerBrowserFeatureAvailability
		if err := row.ToStruct(&entry); err != nil {
			return err
		}
		availabilities = append(availabilities, entry)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return availabilities, nil
}
