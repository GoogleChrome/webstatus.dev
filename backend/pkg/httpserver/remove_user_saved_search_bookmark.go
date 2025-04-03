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

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// RemoveUserSavedSearchBookmark implements backend.StrictServerInterface.
// nolint: ireturn // Name generated from openapi
func (s *Server) RemoveUserSavedSearchBookmark(
	ctx context.Context, request backend.RemoveUserSavedSearchBookmarkRequestObject) (
	backend.RemoveUserSavedSearchBookmarkResponseObject, error) {
	userCheckResult := CheckAuthenticatedUser(ctx, "RemoveUserSavedSearchBookmark",
		func(code int, message string) backend.RemoveUserSavedSearchBookmark500JSONResponse {
			return backend.RemoveUserSavedSearchBookmark500JSONResponse{
				Code:    code,
				Message: message,
			}
		})
	if userCheckResult.User == nil {
		return userCheckResult.Response, nil
	}

	err := s.wptMetricsStorer.RemoveUserSavedSearchBookmark(ctx, userCheckResult.User.ID, request.SearchId)
	if err != nil {
		if errors.Is(err, backendtypes.ErrUserNotAuthorizedForAction) {
			return backend.RemoveUserSavedSearchBookmark403JSONResponse{
				Code:    http.StatusForbidden,
				Message: "saved search owner cannot delete bookmark",
			}, nil
		} else if errors.Is(err, backendtypes.ErrEntityDoesNotExist) {
			return backend.RemoveUserSavedSearchBookmark404JSONResponse{
				Code:    http.StatusNotFound,
				Message: "saved search to bookmark not found",
			}, nil
		}

		slog.ErrorContext(ctx, "unable to remove bookmark", "error", err)

		return backend.RemoveUserSavedSearchBookmark500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "unable to remove bookmark",
		}, nil

	}

	return backend.RemoveUserSavedSearchBookmark204Response{}, nil
}
