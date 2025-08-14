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

// ListUserSavedSearches implements backend.StrictServerInterface.
// nolint:ireturn // Expected ireturn for openapi generation.
func (s *Server) ListUserSavedSearches(
	ctx context.Context, request backend.ListUserSavedSearchesRequestObject) (
	backend.ListUserSavedSearchesResponseObject, error) {
	// At this point, the user should be authenticated and in the context.
	// If for some reason the user is not in the context, it is a library or
	// internal issue and not an user issue. Return 500 error in that case.
	user, found := httpmiddlewares.AuthenticatedUserFromContext(ctx)
	if !found {
		slog.ErrorContext(ctx, "user not found in context. middleware malfunction")

		return backend.ListUserSavedSearches500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "internal server error",
		}, nil
	}
	page, err := s.wptMetricsStorer.ListUserSavedSearches(
		ctx,
		user.ID,
		getPageSizeOrDefault(request.Params.PageSize),
		request.Params.PageToken,
	)

	if err != nil {
		if errors.Is(err, backendtypes.ErrInvalidPageToken) {
			slog.WarnContext(ctx, "invalid page token", "token", request.Params.PageToken, "error", err)

			return backend.ListUserSavedSearches400JSONResponse{
				Code:    400,
				Message: "invalid page token",
			}, nil
		}

		slog.ErrorContext(ctx, "unable to list user saved searches", "error", err, "user", user.ID)

		return backend.ListUserSavedSearches500JSONResponse{
			Code:    500,
			Message: "unable to list saved searches for user",
		}, nil
	}

	return backend.ListUserSavedSearches200JSONResponse(*page), nil
}
