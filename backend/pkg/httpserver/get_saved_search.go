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
	"fmt"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// GetV1SavedSearchesSearchId implements backend.StrictServerInterface.
// nolint:ireturn, revive // Expected ireturn for openapi generation.
func (s *Server) GetV1SavedSearchesSearchId(
	_ context.Context, req backend.GetV1SavedSearchesSearchIdRequestObject) (
	backend.GetV1SavedSearchesSearchIdResponseObject, error) {
	savedSearches := getSavedSearches()
	for _, search := range savedSearches {
		if req.SearchId == search.Id {
			return backend.GetV1SavedSearchesSearchId200JSONResponse(search), nil
		}
	}

	return backend.GetV1SavedSearchesSearchId404JSONResponse{
		Code:    404,
		Message: fmt.Sprintf("unable to find search %s", req.SearchId),
	}, nil
}
