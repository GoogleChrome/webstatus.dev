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

package httpserver

import (
	"context"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

func getSavedSearches() []backend.SavedSearchResponse {
	ownerStatus1 := backend.SavedSearchResponseOwnerStatusNone
	subscribeStatus1 := backend.SavedSearchResponseSubscriptionStatusActive
	ownerStatus2 := backend.SavedSearchResponseOwnerStatusAdmin
	subscribeStatus2 := backend.SavedSearchResponseSubscriptionStatusNone
	ownerStatus3 := backend.SavedSearchResponseOwnerStatusNone
	subscribeStatus3 := backend.SavedSearchResponseSubscriptionStatusNone

	return []backend.SavedSearchResponse{
		{
			CreatedAt:          time.Date(2024, time.September, 1, 1, 0, 0, 0, time.UTC),
			UpdatedAt:          nil,
			Id:                 "1",
			Name:               "a query I subscribe to",
			Query:              "group:css",
			OwnerStatus:        &ownerStatus1,
			SubscriptionStatus: &subscribeStatus1,
		},
		{
			CreatedAt:          time.Date(2024, time.September, 1, 1, 0, 0, 0, time.UTC),
			UpdatedAt:          nil,
			Id:                 "2",
			Name:               "my personal query",
			Query:              "available_on:chrome AND group:css",
			OwnerStatus:        &ownerStatus2,
			SubscriptionStatus: &subscribeStatus2,
		},
		{
			CreatedAt:          time.Date(2024, time.September, 1, 1, 0, 0, 0, time.UTC),
			UpdatedAt:          nil,
			Id:                 "3",
			Name:               "a new query",
			Query:              "available_on:chrome",
			OwnerStatus:        &ownerStatus3,
			SubscriptionStatus: &subscribeStatus3,
		},
	}
}

// ListSavedSearches implements backend.StrictServerInterface.
// nolint:ireturn // Expected ireturn for openapi generation.
func (s *Server) ListSavedSearches(
	_ context.Context, _ backend.ListSavedSearchesRequestObject) (
	backend.ListSavedSearchesResponseObject, error) {
	searches := getSavedSearches()

	return backend.ListSavedSearches200JSONResponse{
		Metadata: nil,
		Data:     &searches,
	}, nil
}
