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

// PutUserSavedSearchBookmark implements backend.StrictServerInterface.
// nolint: ireturn // Name generated from openapi
func (s *Server) PutUserSavedSearchBookmark(
	ctx context.Context, request backend.PutUserSavedSearchBookmarkRequestObject) (
	backend.PutUserSavedSearchBookmarkResponseObject, error) {
	userCheckResult := CheckAuthenticatedUser(ctx, "PutUserSavedSearchBookmark",
		func(code int, message string) backend.PutUserSavedSearchBookmark500JSONResponse {
			return backend.PutUserSavedSearchBookmark500JSONResponse{
				Code:    code,
				Message: message,
			}
		})
	if userCheckResult.User == nil {
		return userCheckResult.Response, nil
	}

	err := s.wptMetricsStorer.PutUserSavedSearchBookmark(ctx, userCheckResult.User.ID, request.SearchId)
	if err != nil {
		if errors.Is(err, backendtypes.ErrEntityDoesNotExist) {
			return backend.PutUserSavedSearchBookmark404JSONResponse{
				Code:    http.StatusNotFound,
				Message: "saved search to bookmark not found",
			}, nil
		} else if errors.Is(err, backendtypes.ErrUserMaxBookmarks) {
			return backend.PutUserSavedSearchBookmark403JSONResponse{
				Code:    http.StatusForbidden,
				Message: "user has reached the maximum number of allowed bookmarks",
			}, nil
		}
		slog.ErrorContext(ctx, "unable to add bookmark", "error", err)

		return backend.PutUserSavedSearchBookmark500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "unable to add bookmark",
		}, nil

	}

	return backend.PutUserSavedSearchBookmark200Response{}, nil
}
