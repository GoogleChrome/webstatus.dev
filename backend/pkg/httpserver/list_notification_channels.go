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
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// ListNotificationChannels handles the GET request to /v1/users/me/notification-channels.
// nolint:ireturn, revive // Expected ireturn for openapi generation.
func (s *Server) ListNotificationChannels(
	ctx context.Context,
	req backend.ListNotificationChannelsRequestObject,
) (backend.ListNotificationChannelsResponseObject, error) {
	userCheckResult := CheckAuthenticatedUser(ctx, "ListNotificationChannels",
		func(code int, message string) backend.ListNotificationChannels500JSONResponse {
			return backend.ListNotificationChannels500JSONResponse{
				Code:    code,
				Message: message,
			}
		})
	if userCheckResult.User == nil {
		return userCheckResult.Response, nil
	}

	pageSize := getPageSizeOrDefault(req.Params.PageSize)

	channels, err := s.wptMetricsStorer.ListNotificationChannels(
		ctx, userCheckResult.User.ID, pageSize, req.Params.PageToken)

	if err != nil {
		if errors.Is(err, backendtypes.ErrInvalidPageToken) {
			return backend.ListNotificationChannels400JSONResponse{
				Code:    http.StatusBadRequest,
				Message: "Invalid page token",
			}, nil
		}

		return backend.ListNotificationChannels500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "Could not list notification channels",
		}, nil
	}

	return backend.ListNotificationChannels200JSONResponse(*channels), nil
}
