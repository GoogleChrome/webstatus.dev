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
	"errors"
	"log/slog"
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/httpmiddlewares"
)

// GetSavedSearch implements backend.StrictServerInterface.
// nolint:ireturn, revive // Expected ireturn for openapi generation.
func (s *Server) GetSavedSearch(
	ctx context.Context, req backend.GetSavedSearchRequestObject) (
	backend.GetSavedSearchResponseObject, error) {
	// At this point, the user should be authenticated and in the context.
	// If for some reason the user is not in the context, treat it as an unauthenticated user
	var userID *string
	user, found := httpmiddlewares.AuthenticatedUserFromContext(ctx)
	if found {
		userID = &user.ID
	}

	search, err := s.wptMetricsStorer.GetSavedSearch(ctx, req.SearchId, userID)
	if err != nil {
		if errors.Is(err, backendtypes.ErrEntityDoesNotExist) {
			return backend.GetSavedSearch404JSONResponse{
				Code:    http.StatusNotFound,
				Message: "saved search not found",
			}, nil
		}

		slog.ErrorContext(ctx, "unable to get saved search", "error", err)

		return backend.GetSavedSearch500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "unable to get saved search",
		}, nil
	}

	return backend.GetSavedSearch200JSONResponse(*search), nil
}
