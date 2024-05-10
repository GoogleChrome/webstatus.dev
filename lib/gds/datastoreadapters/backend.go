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

package datastoreadapters

import (
	"context"
	"errors"

	"github.com/GoogleChrome/webstatus.dev/lib/gds"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

type BackendDatastoreClient interface {
	GetWebFeatureMetadata(ctx context.Context, webFeatureID string) (*gds.FeatureMetadata, error)
}

// Backend converts queries to datastore to usable entities for the backend
// service.
type Backend struct {
	client BackendDatastoreClient
}

// NewBackend constructs an adapter for the backend service.
func NewBackend(client BackendDatastoreClient) *Backend {
	return &Backend{client: client}
}

func (d *Backend) GetFeatureMetadata(
	ctx context.Context,
	featureID string,
) (*backend.FeatureMetadata, error) {
	metadata, err := d.client.GetWebFeatureMetadata(ctx, featureID)
	if errors.Is(err, gds.ErrEntityNotFound) {
		// Return an empty metadata for now.
		// The feature exists but datastore doesn't have any metadata for it.
		// This could be because the feature's metadata has not been stored yet.
		return &backend.FeatureMetadata{
			CanIUse:     nil,
			Description: nil,
		}, nil
	} else if err != nil {
		return nil, err
	}

	var canIUse *backend.CanIUseInfo
	if len(metadata.CanIUseIDs) > 0 {
		items := make([]backend.CanIUseItem, 0, len(metadata.CanIUseIDs))
		for idx := range metadata.CanIUseIDs {
			items = append(items, backend.CanIUseItem{
				Id: &metadata.CanIUseIDs[idx],
			})
		}
		canIUse = &backend.CanIUseInfo{
			Items: &items,
		}
	}

	return &backend.FeatureMetadata{
		CanIUse:     canIUse,
		Description: &metadata.Description,
	}, nil
}
