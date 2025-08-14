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

package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// RemoveSavedSearch implements backend.StrictServerInterface.
// nolint: ireturn // Name generated from openapi
func (s *Server) RemoveSavedSearch(
	ctx context.Context, request backend.RemoveSavedSearchRequestObject) (
	backend.RemoveSavedSearchResponseObject, error) {
	userCheckResult := CheckAuthenticatedUser(ctx, "RemoveSavedSearch",
		func(code int, message string) backend.RemoveSavedSearch500JSONResponse {
			return backend.RemoveSavedSearch500JSONResponse{
				Code:    code,
				Message: message,
			}
		})
	if userCheckResult.User == nil {
		return userCheckResult.Response, nil
	}

	err := s.wptMetricsStorer.DeleteUserSavedSearch(ctx, userCheckResult.User.ID, request.SearchId)
	if err != nil {
		if errors.Is(err, backendtypes.ErrEntityDoesNotExist) {
			return backend.RemoveSavedSearch404JSONResponse{
				Code:    http.StatusNotFound,
				Message: "saved search not found",
			}, nil
		} else if errors.Is(err, backendtypes.ErrUserNotAuthorizedForAction) {
			return backend.RemoveSavedSearch403JSONResponse{
				Code:    http.StatusForbidden,
				Message: "forbidden",
			}, nil
		}

		slog.ErrorContext(ctx, "unknown error deleting saved search", "error", err)

		return backend.RemoveSavedSearch500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "unable to delete saved search",
		}, nil
	}

	return backend.RemoveSavedSearch204Response{}, nil
}
